package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"kredit/internal/businesspolicy"
	"kredit/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

const demoSeedPath = "db/seeds/001_demo.sql"

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	if cfg.Environment != "development" {
		return errors.New("demo seeding requires APP_ENV=development")
	}
	databaseURL := os.Getenv("DATABASE_DIRECT_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		return errors.New("DATABASE_DIRECT_URL or DATABASE_URL is required for demo seeding")
	}
	seedSQL, err := os.ReadFile(demoSeedPath)
	if err != nil {
		return fmt.Errorf("read demo seed: %w", err)
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()
	if err = businesspolicy.NewStore(pool, cfg).Ensure(ctx); err != nil {
		return fmt.Errorf("ensure business policy: %w", err)
	}
	fmt.Printf("loading %s\n", demoSeedPath)
	if _, err := pool.Exec(ctx, string(seedSQL)); err != nil {
		return fmt.Errorf("apply demo seed: %w", err)
	}
	return nil
}
