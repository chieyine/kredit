package main

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kredit/internal/db"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

//go:embed runtime_roles.sql
var runtimeRoleBootstrap string

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	databaseURL := os.Getenv("DATABASE_DIRECT_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		return errors.New("DATABASE_DIRECT_URL or DATABASE_URL is required for migrations")
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = database.Close() }()
	// A migration run should fail fast when the database is unreachable
	// rather than hang a deploy step indefinitely.
	pingCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := database.PingContext(pingCtx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	migrationsDir := filepath.Join(root, "db", "migrations")
	if len(args) > 0 {
		if len(args) != 1 || args[0] != "down" {
			return errors.New("usage: migrate [down]")
		}
		if os.Getenv("ALLOW_DB_ROLLBACK") != "true" || (os.Getenv("APP_ENV") != "development" && os.Getenv("APP_ENV") != "test") {
			return errors.New("rollback requires explicit authorization in development or test")
		}
		if err := goose.Down(database, migrationsDir); err != nil {
			return fmt.Errorf("roll back migration: %w", err)
		}
		fmt.Println("one application migration rolled back")
		return nil
	}
	// Migrations 195 onward grant functions to runtime roles. Provision only
	// their non-login identities here; schema/table privileges are installed
	// separately after migrations. Never run the full roles.sql before schema
	// creation, and never grant runtime roles migration privileges.
	if _, err := database.ExecContext(context.Background(), runtimeRoleBootstrap); err != nil {
		return fmt.Errorf("bootstrap runtime roles (ask the database administrator to pre-create kredit_app and kredit_worker when using a restricted migrator): %w", err)
	}
	if err := goose.Up(database, migrationsDir); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	pool, err := dbpool(context.Background(), databaseURL)
	if err != nil {
		return fmt.Errorf("open job queue connection: %w", err)
	}
	defer pool.Close()
	riverMigrator, err := rivermigrate.New(riverpgxv5.New(pool.Raw()), &rivermigrate.Config{Schema: "jobs"})
	if err != nil {
		return err
	}
	if _, err := riverMigrator.Migrate(context.Background(), rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("apply job queue migrations: %w", err)
	}
	fmt.Println("database migrations are up to date")
	return nil
}

func dbpool(ctx context.Context, databaseURL string) (*db.Pool, error) {
	return db.Open(ctx, databaseURL)
}

// extractUpSQL remains a small compatibility helper for migration tests and
// tooling that inspect a Goose migration without executing it.
func extractUpSQL(contents []byte) string {
	const downMarker = "-- +goose Down"
	if index := strings.Index(string(contents), downMarker); index >= 0 {
		return string(contents[:index])
	}
	return string(contents)
}
