package main

import (
	"context"
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
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "production configuration rejected: %v\n", err)
		os.Exit(1)
	}
	if *stored {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		database, err := db.OpenAsRole(ctx, cfg.DatabaseURL, "kredit_app")
		if err != nil {
			fmt.Fprintln(os.Stderr, "runtime database configuration could not be read")
			os.Exit(1)
		}
		defer database.Close()
		settings := platformsettings.NewPostgresStore(database.Raw(), platformsettings.NewEncryptor(cfg.SettingsEncryptionKey), nil)
		cfg, err = config.ApplyStoredConnections(ctx, cfg, settings, "", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "saved runtime configuration rejected: %v\n", err)
			os.Exit(1)
		}
	}
	report := readiness.Evaluate(cfg)
	if !report.Ready {
		fmt.Fprintf(os.Stderr, "production readiness rejected; missing gates: %v\n", report.Missing)
		os.Exit(1)
	}
	fmt.Printf("production configuration accepted for %s (%s); %d readiness gates passed\n", cfg.Environment, cfg.Version, len(report.Gates))
}
