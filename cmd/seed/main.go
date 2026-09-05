package main

import (
	"context"
	"fmt"
	"kredit/internal/businesspolicy"
	"kredit/internal/config"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	seed := filepath.Join(root, "db", "seeds", "001_demo.sql")
	seedSQL, err := os.ReadFile(seed)
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
	fmt.Printf("loading %s\n", seed)
	if _, err := pool.Exec(ctx, string(seedSQL)); err != nil {
		panic(fmt.Errorf("apply demo seed: %w", err))
	}
}
