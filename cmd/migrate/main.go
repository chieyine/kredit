package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kredit/internal/db"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

//go:embed runtime_roles.sql
var runtimeRoleBootstrap string

func main() {
	databaseURL := os.Getenv("DATABASE_DIRECT_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		panic("DATABASE_DIRECT_URL or DATABASE_URL is required for migrations")
	}
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		panic(err)
	}
	defer func() { _ = db.Close() }()
	if err := db.Ping(); err != nil {
		panic(err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}
	migrationsDir := filepath.Join(root, "db", "migrations")
	if len(os.Args) > 1 {
		if len(os.Args) != 2 || os.Args[1] != "down" {
			panic("usage: migrate [down]")
		}
		if os.Getenv("ALLOW_DB_ROLLBACK") != "true" || (os.Getenv("APP_ENV") != "development" && os.Getenv("APP_ENV") != "test") {
			panic("rollback requires explicit authorization in development or test")
		}
		if err := goose.Down(db, migrationsDir); err != nil {
			panic(err)
		}
		fmt.Println("one application migration rolled back")
		return
	}
	// Migrations 195 onward grant functions to runtime roles. Provision only
	// their non-login identities here; schema/table privileges are installed
	// separately after migrations. Never run the full roles.sql before schema
	// creation, and never grant runtime roles migration privileges.
	if _, err := db.ExecContext(context.Background(), runtimeRoleBootstrap); err != nil {
		panic(fmt.Errorf("bootstrap runtime roles (ask the database administrator to pre-create kredit_app and kredit_worker when using a restricted migrator): %w", err))
	}
	if err := goose.Up(db, migrationsDir); err != nil {
		panic(err)
	}
	pool, err := dbpool(context.Background(), databaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	riverMigrator, err := rivermigrate.New(riverpgxv5.New(pool.Raw()), &rivermigrate.Config{Schema: "jobs"})
	if err != nil {
		panic(err)
	}
	if _, err := riverMigrator.Migrate(context.Background(), rivermigrate.DirectionUp, nil); err != nil {
		panic(err)
	}
	fmt.Println("database migrations are up to date")
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
