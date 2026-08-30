package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"kredit/internal/config"
	"kredit/internal/db"
	"kredit/internal/jobs"
	"kredit/internal/outbox"
	"kredit/internal/platform/logging"
	"kredit/internal/web"
)

// healthServer exposes a minimal liveness/readiness endpoint so Kubernetes can
// detect a wedged worker. It binds only inside the pod network.
func startHealthServer(addr string, ready func() error) *http.Server {
	mux := healthHandler(ready)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	go func() {
		_ = server.ListenAndServe()
	}()
	return server
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
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger := logging.New()
	database, err := db.Open(context.Background(), cfg.RiverDatabaseURL)
	if err != nil {
		logger.Error("worker database startup failed", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	if err := database.CheckSchema(context.Background()); err != nil {
		logger.Error("worker database schema check failed", "error", err)
		os.Exit(1)
	}
	if err := database.CheckPersistenceContract(context.Background()); err != nil {
		logger.Error("worker persistence contract check failed", "error", err)
		os.Exit(1)
	}
	runtime := web.NewRuntimeWithDB(cfg, database)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = runtime.Tracer.Shutdown(shutdownCtx)
	}()
	jobClient, err := jobs.NewClientWithHandlers(database.Raw(), logger, jobs.Handlers{
		Tracer:  runtime.Tracer,
		Metrics: runtime.Metrics,
		Collection: func(ctx context.Context, operation, resourceID string) error {
			switch operation {
			case jobs.OpReconcileProvider:
				_, err := runtime.Collections.Reconcile(ctx, resourceID)
				return err
			default:
				return errors.New("unsupported collection operation")
			}
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
		logger.Error("worker initialization failed", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	healthServer := startHealthServer(envOr("WORKER_HEALTH_ADDR", ":8081"), func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return database.Ping(pingCtx)
	})
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = healthServer.Shutdown(shutdownCtx)
	}()

	logger.Info("worker started", "version", cfg.Version)
	if err := jobClient.Start(ctx); err != nil {
		logger.Error("worker failed to start", "error", err)
		os.Exit(1)
	}
	maintenanceTicker := time.NewTicker(time.Minute)
	defer maintenanceTicker.Stop()
	reconciliationTicker := time.NewTicker(5 * time.Minute)
	defer reconciliationTicker.Stop()
	deliveryTicker := time.NewTicker(30 * time.Second)
	defer deliveryTicker.Stop()
	outboxTicker := time.NewTicker(2 * time.Second)
	defer outboxTicker.Stop()
	dispatcher := outbox.NewDispatcher(runtime.Outbox, outbox.PublishFunc(func(ctx context.Context, event outbox.Event) error {
		// Every committed financial event is handed to River before its outbox
		// row is acknowledged. The reconciliation job is intentionally keyed by
		// aggregate so bursts collapse without losing the safety check.
		return jobClient.EnqueueReconciliation(ctx, jobs.ReconciliationArgs{Operation: jobs.OpReconcileLedger, ResourceID: event.AggregateType + ":" + event.AggregateID})
	}))
	if err := jobClient.EnqueueMaintenance(ctx, jobs.MaintenanceArgs{Operation: jobs.OpExpireReservations}); err != nil {
		logger.Error("initial maintenance job enqueue failed", "error", err)
	}
	if err := jobClient.EnqueueMaintenance(ctx, jobs.MaintenanceArgs{Operation: jobs.OpEvaluateSchedules}); err != nil {
		logger.Error("initial schedule evaluation enqueue failed", "error", err)
	}
	if err := jobClient.EnqueueMaintenance(ctx, jobs.MaintenanceArgs{Operation: jobs.OpReconcileSupplierOnboarding}); err != nil {
		logger.Error("initial supplier onboarding reconciliation enqueue failed", "error", err)
	}
	if err := jobClient.EnqueueReconciliation(ctx, jobs.ReconciliationArgs{Operation: jobs.OpReconcileLedger}); err != nil {
		logger.Error("initial financial reconciliation enqueue failed", "error", err)
	}
	enqueueDueNotifications(ctx, runtime, jobClient, logger)
	enqueuePendingDocuments(ctx, runtime, jobClient, logger)
	dispatchOutbox(ctx, dispatcher, logger)
	go func() {
		for {
			select {
			case <-maintenanceTicker.C:
				if err := jobClient.EnqueueMaintenance(ctx, jobs.MaintenanceArgs{Operation: jobs.OpExpireReservations}); err != nil {
					logger.Error("maintenance job enqueue failed", "error", err)
				}
				if err := jobClient.EnqueueMaintenance(ctx, jobs.MaintenanceArgs{Operation: jobs.OpEvaluateSchedules}); err != nil {
					logger.Error("schedule evaluation enqueue failed", "error", err)
				}
				if err := jobClient.EnqueueMaintenance(ctx, jobs.MaintenanceArgs{Operation: jobs.OpReconcileSupplierOnboarding}); err != nil {
					logger.Error("supplier onboarding reconciliation enqueue failed", "error", err)
				}
			case <-reconciliationTicker.C:
				if err := jobClient.EnqueueReconciliation(ctx, jobs.ReconciliationArgs{Operation: jobs.OpReconcileLedger}); err != nil {
					logger.Error("financial reconciliation enqueue failed", "error", err)
				}
			case <-deliveryTicker.C:
				enqueueDueNotifications(ctx, runtime, jobClient, logger)
				enqueuePendingDocuments(ctx, runtime, jobClient, logger)
			case <-outboxTicker.C:
				dispatchOutbox(ctx, dispatcher, logger)
			case <-ctx.Done():
				return
			}
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := jobClient.Stop(shutdownCtx); err != nil {
		logger.Error("worker shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("worker stopped")
}

func enqueueDueNotifications(ctx context.Context, runtime *web.Runtime, client *jobs.Client, logger interface{ Error(string, ...any) }) {
	ids, err := runtime.Notifications.DueDeliveryIDs(ctx, 100)
	if err != nil {
		logger.Error("notification delivery discovery failed", "error", err)
		return
	}
	for _, id := range ids {
		if err := client.EnqueueNotification(ctx, jobs.NotificationArgs{Operation: jobs.OpDeliver, NotificationID: id}); err != nil {
			logger.Error("notification delivery enqueue failed", "notification_id", id, "error", err)
		}
	}
}

func dispatchOutbox(ctx context.Context, dispatcher *outbox.Dispatcher, logger interface{ Error(string, ...any) }) {
	if _, err := dispatcher.DispatchOnce(ctx, 100); err != nil {
		logger.Error("outbox dispatch failed", "error", err)
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

// runSelfHealthcheck lets container healthchecks probe the worker's local
// health server without requiring curl or a shell inside the runtime image.
func runSelfHealthcheck() int {
	addr := envOr("WORKER_HEALTH_ADDR", ":8081")
	host := addr
	if strings.HasPrefix(host, ":") {
		host = "127.0.0.1" + host
	}
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get("http://" + host + "/readyz")
	if err != nil {
		return 1
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func enqueuePendingDocuments(ctx context.Context, runtime *web.Runtime, client *jobs.Client, logger interface{ Error(string, ...any) }) {
	ids, err := runtime.Documents.PendingScanIDs(ctx, 100)
	if err != nil {
		logger.Error("document scan discovery failed", "error", err)
		return
	}
	for _, id := range ids {
		if err := client.EnqueueDocument(ctx, jobs.DocumentArgs{Operation: jobs.OpScan, DocumentID: id}); err != nil {
			logger.Error("document scan enqueue failed", "document_id", id, "error", err)
		}
	}
}
