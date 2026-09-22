package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"kredit/internal/config"
	"kredit/internal/db"
	"kredit/internal/jobs"
	"kredit/internal/notifications"
	"kredit/internal/outbox"
	"kredit/internal/platform/logging"
	"kredit/internal/platformsettings"
	"kredit/internal/web"
)

// healthServer exposes local liveness and readiness. Readiness also requires
// successful progress from the critical scheduling activities.
func startHealthServer(addr string, ready func() error) (*http.Server, <-chan error) {
	server := &http.Server{
		Addr:              addr,
		Handler:           healthHandler(ready),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	failures := make(chan error, 1)
	go func() { failures <- server.ListenAndServe() }()
	return server, failures
}

func healthHandler(ready func() error) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if err := ready(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(runSelfHealthcheck())
	}
	if err := run(); err != nil {
		logging.New().Error("worker stopped", "error", err)
		os.Exit(1)
	}
}

func run() (result error) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg, err := config.LoadBootstrap()
	if err != nil {
		return fmt.Errorf("load worker startup configuration: %w", err)
	}
	logger := logging.New()
	startupCtx, startupCancel := context.WithTimeout(ctx, 45*time.Second)
	defer startupCancel()
	database, err := db.OpenAsRole(startupCtx, cfg.RiverDatabaseURL, "kredit_worker")
	if err != nil {
		return fmt.Errorf("worker database startup: %w", err)
	}
	defer database.Close()
	if err := database.CheckSchema(startupCtx); err != nil {
		return fmt.Errorf("worker database schema: %w", err)
	}
	if err := database.CheckPersistenceContract(startupCtx); err != nil {
		return fmt.Errorf("worker persistence contract: %w", err)
	}
	settings := platformsettings.NewPostgresStore(database.Raw(), platformsettings.NewEncryptor(cfg.SettingsEncryptionKey), nil)
	cfg, err = config.ApplyStoredConnections(startupCtx, cfg, settings, "", nil)
	if err != nil {
		return fmt.Errorf("activate saved worker configuration: %w", err)
	}
	startupCancel()
	runtime := web.NewRuntimeWithDB(cfg, database)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := runtime.Tracer.Shutdown(shutdownCtx); err != nil {
			result = errors.Join(result, fmt.Errorf("flush worker tracing: %w", err))
		}
	}()
	if len(runtime.ProviderFailures) > 0 {
		return errors.New("worker startup blocked: a configured provider could not initialize")
	}
	jobClient, err := jobs.NewClientWithHandlers(database.Raw(), logger, jobs.Handlers{
		CleanupDocuments: runtime.Documents.CleanupOrphans,
		MaturedCredit: func(ctx context.Context) error {
			store, ok := runtime.Credit.(interface {
				AutoActivateMatured(context.Context, time.Time) ([]string, error)
			})
			if !ok {
				return errors.New("durable credit activation unavailable")
			}
			_, err := store.AutoActivateMatured(ctx, time.Now())
			return err
		},
		ProviderWebhook: runtime.HandleProviderNotice,
		Tracer:          runtime.Tracer,
		Metrics:         runtime.Metrics,
		Collection: func(ctx context.Context, args jobs.CollectionArgs) error {
			return runtime.HandleCollectionJob(ctx, cfg, args)
		},
		Notification: func(ctx context.Context, operation, resourceID string) error {
			if operation != jobs.OpDeliver {
				return errors.New("unsupported notification operation")
			}
			return runtime.Notifications.DeliverScheduled(ctx, resourceID)
		},
		Document: func(ctx context.Context, operation, resourceID string) error {
			if operation != jobs.OpScan {
				return errors.New("unsupported document operation")
			}
			if runtime.DocumentScanner == nil {
				return errors.New("document scanner is not configured")
			}
			_, err := runtime.Documents.Scan(ctx, resourceID, runtime.DocumentScanner)
			return err
		},
	})
	if err != nil {
		return fmt.Errorf("initialize worker: %w", err)
	}
	// Scheduling stops before River. Its work context remains alive during the
	// graceful drain; cancellation is the fallback if that drain exceeds 10s.
	jobCtx, cancelJobs := context.WithCancel(context.Background())
	defer cancelJobs()
	if err := jobClient.Start(jobCtx); err != nil {
		return fmt.Errorf("start worker: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := jobClient.Stop(shutdownCtx)
		cancel()
		if err != nil {
			cancelJobs()
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cleanupCancel()
			result = errors.Join(result, fmt.Errorf("drain worker: %w", err), jobClient.Stop(cleanupCtx))
		}
	}()

	dispatcher := outbox.NewDispatcher(runtime.Outbox, outbox.PublishFunc(func(ctx context.Context, event outbox.Event) error {
		if event.EventType == "notification.requested" {
			return runtime.QueueOutboxNotification(ctx, event)
		}
		return jobClient.EnqueueReconciliation(ctx, jobs.ReconciliationArgs{Operation: jobs.OpReconcileLedger, ResourceID: event.AggregateType + ":" + event.AggregateID})
	}))
	schedule := startWorkerSchedule(ctx, logger, []*periodicTask{
		{name: "maintenance", interval: time.Minute, budget: 30 * time.Second, critical: true, work: func(ctx context.Context) error {
			var failures []error
			for _, operation := range []string{jobs.OpExpireReservations, jobs.OpEvaluateSchedules, jobs.OpReconcileSupplierOnboarding} {
				if err := jobClient.EnqueueMaintenance(ctx, jobs.MaintenanceArgs{Operation: operation}); err != nil {
					failures = append(failures, fmt.Errorf("enqueue %s: %w", operation, err))
				}
			}
			return errors.Join(failures...)
		}},
		{name: "collections", interval: time.Minute, budget: 45 * time.Second, critical: true, work: func(ctx context.Context) error {
			return runtime.EnqueueCollectionWork(ctx, cfg)
		}},
		{name: "ledger_reconciliation", interval: 5 * time.Minute, budget: 30 * time.Second, critical: true, work: func(ctx context.Context) error {
			return jobClient.EnqueueReconciliation(ctx, jobs.ReconciliationArgs{Operation: jobs.OpReconcileLedger})
		}},
		{name: "outbox", interval: 2 * time.Second, budget: 30 * time.Second, critical: true, work: func(ctx context.Context) error {
			return dispatchOutbox(ctx, dispatcher, logger)
		}},
		{name: "notifications", interval: 30 * time.Second, budget: 90 * time.Second, work: func(ctx context.Context) error {
			return enqueueDueNotifications(ctx, runtime, jobClient, logger)
		}},
		{name: "documents", interval: 30 * time.Second, budget: 30 * time.Second, work: func(ctx context.Context) error {
			return enqueuePendingDocuments(ctx, runtime, jobClient, logger)
		}},
	})
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		result = errors.Join(result, schedule.Stop(shutdownCtx))
	}()
	watchCtx, cancelWatch := context.WithCancel(ctx)
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		config.WatchConnections(watchCtx, cfg, settings, database.Raw(), "worker", stop)
	}()
	defer func() { cancelWatch(); <-watchDone }()

	healthServer, healthErrors := startHealthServer(envOr("WORKER_HEALTH_ADDR", ":8081"), func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := schedule.Ready(); err != nil {
			return err
		}
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return database.Ping(pingCtx)
	})
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := healthServer.Shutdown(shutdownCtx); err != nil {
			result = errors.Join(result, fmt.Errorf("worker health shutdown: %w", err), healthServer.Close())
		}
	}()
	logger.Info("worker started", "version", cfg.Version)
	select {
	case <-ctx.Done():
		logger.Info("worker stopping")
		return nil
	case err := <-healthErrors:
		stop()
		return fmt.Errorf("worker health server stopped unexpectedly: %w", err)
	}
}

func enqueueDueNotifications(ctx context.Context, runtime *web.Runtime, client *jobs.Client, logger interface{ Error(string, ...any) }) error {
	var failures []error
	for _, channel := range []string{notifications.ChannelEmail, notifications.ChannelSMS, notifications.ChannelWhatsApp} {
		lookupCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		err := runtime.Notifications.ReconcileDelivery(lookupCtx, channel, 20)
		cancel()
		if err != nil {
			logger.Error("notification receipt reconciliation failed", "channel", channel, "error", err)
			failures = append(failures, err)
		}
	}
	ids, err := runtime.Notifications.DueDeliveryIDs(ctx, 100)
	if err != nil {
		return errors.Join(append(failures, err)...)
	}
	for _, id := range ids {
		if err := client.EnqueueNotification(ctx, jobs.NotificationArgs{Operation: jobs.OpDeliver, NotificationID: id}); err != nil {
			logger.Error("notification delivery enqueue failed", "notification_id", id, "error", err)
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func dispatchOutbox(ctx context.Context, dispatcher *outbox.Dispatcher, logger interface{ Error(string, ...any) }) error {
	_, err := dispatcher.DispatchOnce(ctx, 100)
	if err != nil {
		logger.Error("outbox dispatch failed", "error", err)
	}
	return err
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func runSelfHealthcheck() int {
	addr := envOr("WORKER_HEALTH_ADDR", ":8081")
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	client := &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Get("http://" + addr + "/readyz")
	if err != nil {
		return 1
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func enqueuePendingDocuments(ctx context.Context, runtime *web.Runtime, client *jobs.Client, logger interface{ Error(string, ...any) }) error {
	ids, err := runtime.Documents.PendingScanIDs(ctx, 100)
	if err != nil {
		return err
	}
	var failures []error
	for _, id := range ids {
		if err := client.EnqueueDocument(ctx, jobs.DocumentArgs{Operation: jobs.OpScan, DocumentID: id}); err != nil {
			logger.Error("document scan enqueue failed", "document_id", id, "error", err)
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
