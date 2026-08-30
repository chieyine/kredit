package outbox

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"kredit/internal/identifier"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDispatcherPublishesCommittedEventOnce(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" || os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := NewStore(pool)
	key := "dispatcher-test-" + identifier.New()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.AppendTx(ctx, tx, Event{AggregateType: "test", AggregateID: identifier.New(), EventType: "test.committed", Payload: json.RawMessage(`{"safe":true}`), IdempotencyKey: key})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM app.outbox_events WHERE id=$1::uuid`, id) }()
	calls := 0
	dispatcher := NewDispatcher(store, PublishFunc(func(context.Context, Event) error { calls++; return nil }))
	dispatcher.now = func() time.Time { return time.Now().UTC() }
	published, err := dispatcher.DispatchOnce(ctx, 100)
	if err != nil || published < 1 || calls < 1 {
		t.Fatalf("dispatch failed: published=%d calls=%d err=%v", published, calls, err)
	}
	var state string
	if err := pool.QueryRow(ctx, `SELECT state FROM app.outbox_events WHERE id=$1::uuid`, id).Scan(&state); err != nil || state != "published" {
		t.Fatalf("event was not acknowledged: %s %v", state, err)
	}
	before := calls
	if _, err := dispatcher.DispatchOnce(ctx, 100); err != nil {
		t.Fatal(err)
	}
	if calls != before {
		t.Fatal("published event was delivered twice")
	}
}
