package db

import (
	"context"
	"os"
	"testing"
)

func TestSetRuntimeDefaultPreservesExplicitDatabaseSetting(t *testing.T) {
	values := map[string]string{"statement_timeout": "45000"}
	setRuntimeDefault(values, "statement_timeout", "30000")
	setRuntimeDefault(values, "lock_timeout", "5000")
	if values["statement_timeout"] != "45000" {
		t.Fatalf("explicit statement timeout was overwritten: %q", values["statement_timeout"])
	}
	if values["lock_timeout"] != "5000" {
		t.Fatalf("default lock timeout was not set: %q", values["lock_timeout"])
	}
}

func TestOpenAsRoleRequiresRole(t *testing.T) {
	if _, err := OpenAsRole(t.Context(), "postgres://localhost/kredit", ""); err == nil {
		t.Fatal("expected an empty runtime role to be rejected")
	}
}

func TestOpenAsRoleConfinesRuntimeCredential(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" || os.Getenv("APP_DATABASE_URL") == "" {
		t.Skip("KREDIT_INTEGRATION=1, DATABASE_URL, and APP_DATABASE_URL are required")
	}

	if ownerPool, err := OpenAsRole(t.Context(), os.Getenv("DATABASE_URL"), "kredit_app"); err == nil {
		ownerPool.Close()
		t.Fatal("database owner credential was accepted as a runtime login")
	}

	pool, err := OpenAsRole(t.Context(), os.Getenv("APP_DATABASE_URL"), "kredit_app")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	conn, err := pool.Raw().Acquire(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()

	var sessionUser, currentUser string
	if err := conn.QueryRow(t.Context(), `SELECT session_user,current_user`).Scan(&sessionUser, &currentUser); err != nil {
		t.Fatal(err)
	}
	if sessionUser != "kredit_app_login" || currentUser != "kredit_app" {
		t.Fatalf("unexpected runtime identity: session=%q current=%q", sessionUser, currentUser)
	}
	if _, err := conn.Exec(t.Context(), `SET ROLE NONE`); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = conn.Exec(context.Background(), `SET ROLE kredit_app`) }()
	var canUseApp bool
	if err := conn.QueryRow(t.Context(), `SELECT has_schema_privilege(current_user,'app','USAGE')`).Scan(&canUseApp); err != nil {
		t.Fatal(err)
	}
	if canUseApp {
		t.Fatal("runtime login retained application privileges after SET ROLE NONE")
	}
}
