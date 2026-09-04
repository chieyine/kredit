package idempotency

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresStoreRuntimeRoleLifecycle(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("KREDIT_INTEGRATION=1 and DATABASE_URL are required")
	}

	ctx := t.Context()
	owner, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()

	runtimeConfig, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	runtimeConfig.ConnConfig.RuntimeParams["role"] = "kredit_app"
	runtime, err := pgxpool.NewWithConfig(ctx, runtimeConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()

	scope := "integration:" + uuid.NewString()
	key := uuid.NewString()
	t.Cleanup(func() {
		_, _ = owner.Exec(context.Background(), `DELETE FROM app.idempotency_records WHERE scope=$1`, scope)
	})

	store := NewPostgresStore(runtime)
	created, replay, err := store.Reserve(ctx, scope, key, "hash-one")
	if err != nil || replay {
		t.Fatalf("reserve: replay=%t err=%v", replay, err)
	}
	if created.Scope != scope || created.Key != key {
		t.Fatalf("unexpected reservation: %+v", created)
	}
	if err := store.Complete(ctx, scope, key, 201, []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
	completed, replay, err := store.Reserve(ctx, scope, key, "hash-one")
	if err != nil || !replay || completed.Status != 201 {
		t.Fatalf("replay: status=%d replay=%t err=%v", completed.Status, replay, err)
	}
	if _, _, err := store.Reserve(ctx, scope, key, "different-hash"); err == nil {
		t.Fatal("expected reuse with a different request hash to fail")
	}

	if _, err := owner.Exec(ctx, `UPDATE app.idempotency_records SET expires_at=$3, completed_at=$3 WHERE scope=$1 AND idempotency_key=$2`, scope, key, time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	replacement, replay, err := store.Reserve(ctx, scope, key, "hash-two")
	if err != nil || replay || replacement.RequestHash != "hash-two" {
		t.Fatalf("expired replacement: record=%+v replay=%t err=%v", replacement, replay, err)
	}
}
