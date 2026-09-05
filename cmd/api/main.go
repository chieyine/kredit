package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kredit/internal/config"
	"kredit/internal/db"
	"kredit/internal/platform/logging"
	"kredit/internal/web"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(runSelfHealthcheck())
	}
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger := logging.New()
	database, err := db.OpenAsRole(context.Background(), cfg.DatabaseURL, "kredit_app")
	if err != nil {
		logger.Error("database startup failed", "error", err)
		os.Exit(1)
	}
	if err := database.CheckSchema(context.Background()); err != nil {
		logger.Error("database schema check failed", "error", err)
		database.Close()
		os.Exit(1)
	}
	if err := database.CheckPersistenceContract(context.Background()); err != nil {
		logger.Error("database persistence contract check failed", "error", err)
		database.Close()
		os.Exit(1)
	}
	defer database.Close()
	runtime := web.NewRuntimeWithDB(cfg, database)
	if (cfg.Environment == "production" || cfg.Environment == "staging") && !runtime.DurableDomainReady() {
		logger.Error("deployment startup blocked: durable domain repositories are not fully wired")
		database.Close()
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = runtime.Tracer.Shutdown(shutdownCtx)
	}()
	server := &http.Server{
		Addr:              cfg.APIListenAddr,
		Handler:           web.NewServerWithRuntime(cfg, logger, runtime).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("api started", "addr", cfg.APIListenAddr, "version", cfg.Version)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("api shutdown failed", "error", err)
			os.Exit(1)
		}
		logger.Info("api stopped")
	}
}

// runSelfHealthcheck lets container healthchecks probe this process without
// requiring curl or a shell inside the distroless runtime image. API_ADDR is
// used only to discover the listen port; the probe host is always loopback so
// deployment input cannot turn the healthcheck into an outbound request.
func runSelfHealthcheck() int {
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		return 1
	}
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get("http://127.0.0.1:" + port + "/api/v1/healthz")
	if err != nil {
		return 1
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
