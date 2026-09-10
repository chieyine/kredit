package outbox

import (
	"context"
	"encoding/json"
	"errors"
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
	dispatcher := NewDispatcher(store, PublishFunc(func(_ context.Context, event Event) error {
		if event.ID == id {
			calls++
		}
		return nil
	}))
	dispatcher.now = func() time.Time { return time.Now().UTC() }
	for batch := 0; batch < 100 && calls == 0; batch++ {
		if _, err := dispatcher.DispatchOnce(ctx, 100); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("target event delivered %d times", calls)
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

func TestOutboxRejectsConflictingReplay(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("integration database required")
	}
	pool, err := pgxpool.New(t.Context(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	store := NewStore(pool)
	event := Event{AggregateType: "test", AggregateID: identifier.New(), EventType: "test.saved", Payload: json.RawMessage(`{"amount":100,"currency":"NGN"}`), IdempotencyKey: identifier.New()}
	id, err := store.AppendTx(t.Context(), tx, event)
	if err != nil {
		t.Fatal(err)
	}
	event.Payload = json.RawMessage(`{ "currency": "NGN", "amount": 100 }`)
	replay, err := store.AppendTx(t.Context(), tx, event)
	if err != nil || replay != id {
		t.Fatalf("equivalent replay: %v", err)
	}
	event.Payload = json.RawMessage(`{"amount":200,"currency":"NGN"}`)
	if _, err := store.AppendTx(t.Context(), tx, event); err == nil {
		t.Fatal("different event was silently treated as already queued")
	}
}

func TestExpiredPublisherCannotOverwriteNewClaim(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("integration database required")
	}
	pool, err := pgxpool.New(t.Context(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var old Event
	err = pool.QueryRow(t.Context(), `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key,state,attempts,processing_started_at,created_at) VALUES('test',uuidv7(),'test.claim','{}',uuidv7()::text,'processing',1,now()-interval '11 minutes','1970-01-01') RETURNING id::text,attempts,processing_started_at`).Scan(&old.ID, &old.Attempts, &old.ProcessingStartedAt)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM app.outbox_events WHERE id=$1::uuid`, old.ID)
	}()
	store := NewStore(pool)
	claims, err := store.Claim(t.Context(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(claims) != 1 || claims[0].ID != old.ID || claims[0].Attempts != 2 || claims[0].State != "processing" {
		t.Fatalf("unexpected reclaimed event: %+v", claims)
	}
	if err := store.MarkPublished(t.Context(), old); !errors.Is(err, ErrClaimLost) {
		t.Fatalf("stale publisher succeeded: %v", err)
	}
	if err := store.MarkFailed(t.Context(), old, "late failure", time.Now()); !errors.Is(err, ErrClaimLost) {
		t.Fatalf("stale failure succeeded: %v", err)
	}
	if err := store.MarkPublished(t.Context(), claims[0]); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkFailed(t.Context(), claims[0], "late failure", time.Now()); !errors.Is(err, ErrClaimLost) {
		t.Fatalf("published event was reopened: %v", err)
	}
}
