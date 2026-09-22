package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kredit/internal/config"
	"kredit/internal/db"
	"kredit/internal/platform/logging"
	"kredit/internal/platformsettings"
	"kredit/internal/web"
)

const selfHealthcheckURL = "http://127.0.0.1:8080/api/v1/healthz"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(runSelfHealthcheck())
	}
	if err := run(); err != nil {
		logging.New().Error("api stopped", "error", err)
		os.Exit(1)
	}
}

// run owns resources so every return unwinds cleanup before main can exit.
func run() (result error) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg, err := config.LoadBootstrap()
	if err != nil {
		return fmt.Errorf("load startup configuration: %w", err)
	}
	logger := logging.New()
	startupCtx, startupCancel := context.WithTimeout(ctx, 45*time.Second)
	defer startupCancel()
	database, err := db.OpenAsRole(startupCtx, cfg.DatabaseURL, "kredit_app")
	if err != nil {
		return fmt.Errorf("database startup: %w", err)
	}
	defer database.Close()
	if err := database.CheckSchema(startupCtx); err != nil {
		return fmt.Errorf("database schema: %w", err)
	}
	if err := database.CheckPersistenceContract(startupCtx); err != nil {
		return fmt.Errorf("database persistence contract: %w", err)
	}
	settings := platformsettings.NewPostgresStore(database.Raw(), platformsettings.NewEncryptor(cfg.SettingsEncryptionKey), nil)
	cfg, err = config.ApplyStoredConnections(startupCtx, cfg, settings, "", nil)
	if err != nil {
		return fmt.Errorf("activate saved connection configuration: %w", err)
	}
	startupCancel()
	runtime := web.NewRuntimeWithDB(cfg, database)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := runtime.Tracer.Shutdown(shutdownCtx); err != nil {
			result = errors.Join(result, fmt.Errorf("flush api tracing: %w", err))
		}
	}()
	if (cfg.Environment == "production" || cfg.Environment == "staging") && !runtime.DurableDomainReady() {
		return errors.New("deployment startup blocked: durable domain repositories are not fully wired")
	}
	server := &http.Server{
		Addr:              cfg.APIListenAddr,
		Handler:           web.NewServerWithRuntime(cfg, logger, runtime).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	defer func() {
		if err := server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			result = errors.Join(result, fmt.Errorf("close api listener and connections: %w", err))
		}
	}()
	watchCtx, cancelWatch := context.WithCancel(ctx)
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		config.WatchConnections(watchCtx, cfg, settings, database.Raw(), "api", stop)
	}()
	defer func() {
		cancelWatch()
		<-watchDone
	}()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("api started", "addr", cfg.APIListenAddr, "version", cfg.Version)
		serverErr <- server.ListenAndServe()
	}()
	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve api: %w", err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return errors.Join(fmt.Errorf("api shutdown: %w", err), server.Close())
		}
	}
	logger.Info("api stopped")
	return nil
}

// runSelfHealthcheck probes the container's fixed API contract directly. The
// production container exposes the API on loopback port 8080, so the probe has
// no deployment-controlled host or URL input and cannot be redirected into an
// outbound request.
func runSelfHealthcheck() int {
	return runSelfHealthcheckWithClient(&http.Client{Timeout: 3 * time.Second})
}

// runSelfHealthcheckWithClient keeps the destination fixed while allowing tests
// to replace only the HTTP transport. Tests therefore do not need to make the
// production healthcheck URL configurable.
func runSelfHealthcheckWithClient(client *http.Client) int {
	if client == nil {
		return 1
	}
	probe := *client
	probe.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := probe.Get(selfHealthcheckURL)
	if err != nil {
		return 1
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
