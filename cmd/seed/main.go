package main

import (
	"context"
	"fmt"
	"kredit/internal/businesspolicy"
	"kredit/internal/config"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

const demoSeedPath = "db/seeds/001_demo.sql"

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	if cfg.Environment != "development" {
		panic("demo seeding requires APP_ENV=development")
	}
	databaseURL := os.Getenv("DATABASE_DIRECT_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		panic("DATABASE_DIRECT_URL or DATABASE_URL is required for demo seeding")
	}
	seedSQL, err := os.ReadFile(demoSeedPath)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	if err = businesspolicy.NewStore(pool, cfg).Ensure(ctx); err != nil {
		panic(err)
	}
	fmt.Printf("loading %s\n", demoSeedPath)
	if _, err := pool.Exec(ctx, string(seedSQL)); err != nil {
		panic(fmt.Errorf("apply demo seed: %w", err))
	}
}
