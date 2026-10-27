package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"kredit/internal/config"
	"kredit/internal/db"
	"kredit/internal/platformsettings"
	"kredit/internal/readiness"
)

func main() {
	stored := flag.Bool("stored", false, "include encrypted admin configuration from the database")
	flag.Parse()
	// run owns the database connection, so it is closed before the process exits.
	if err := run(*stored); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(stored bool) error {
	load := config.Load
	if stored {
		load = config.LoadBootstrap
	}
	cfg, err := load()
	if err != nil {
		return fmt.Errorf("production configuration rejected: %w", err)
	}
	if stored {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		database, err := db.OpenAsRole(ctx, cfg.DatabaseURL, "kredit_app")
		if err != nil {
			return errors.New("runtime database configuration could not be read")
		}
		defer database.Close()
		settings := platformsettings.NewPostgresStore(database.Raw(), platformsettings.NewEncryptor(cfg.SettingsEncryptionKey), nil)
		cfg, err = config.ApplyStoredConnections(ctx, cfg, settings, "", nil)
		if err != nil {
			return fmt.Errorf("saved runtime configuration rejected: %w", err)
		}
	}
	report := readiness.Evaluate(cfg)
	if !report.Ready {
		return fmt.Errorf("production readiness rejected; missing gates: %v", report.Missing)
	}
	fmt.Printf("production configuration accepted for %s (%s); %d readiness gates passed\n", cfg.Environment, cfg.Version, len(report.Gates))
	return nil
}
