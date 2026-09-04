package main

import (
	"context"
	"fmt"
	"kredit/internal/businesspolicy"
	"kredit/internal/config"
	"os"
	"os/exec"
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
		databaseURL = "postgres://kredit:kredit@localhost:5432/kredit?sslmode=disable"
	}
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	seed := filepath.Join(root, "db", "seeds", "001_demo.sql")
	if _, err := os.Stat(seed); err != nil {
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
	command := exec.Command("psql", databaseURL, "-v", "ON_ERROR_STOP=1", "-f", seed)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		panic(err)
	}
}
