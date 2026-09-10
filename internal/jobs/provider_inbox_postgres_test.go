package jobs

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

func TestProviderInboxSerializesDeliveryAndRejectsChangedReplay(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("isolated integration database required")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	args := ProviderWebhookArgs{Provider: "synthetic-inbox", EventID: uuid.NewString(), EventType: "debit.completed", Payload: []byte(`{"sequence":"9999999999999999999999999","status":"completed"}`), SignatureValid: true}
	defer func() {
		if _, err := pool.Exec(ctx, `DELETE FROM app.provider_webhook_inbox WHERE provider=$1 AND event_id=$2`, args.Provider, args.EventID); err != nil {
			t.Error(err)
		}
	}()
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	var count atomic.Int32
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	worker := ProviderWebhookWorker{Pool: pool, Handler: func(ctx context.Context, _ ProviderWebhookArgs) error {
		if count.Add(1) == 1 {
			close(started)
		}
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}}
	go func() { done <- worker.Work(ctx, &river.Job[ProviderWebhookArgs]{Args: args}) }()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("first handler did not start")
	}
	if err := worker.Work(ctx, &river.Job[ProviderWebhookArgs]{Args: args}); err == nil {
		t.Error("concurrent delivery was not deferred")
	}
	if count.Load() != 1 {
		t.Error("concurrent delivery invoked handler twice")
	}
	unblock()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("handler did not finish")
	}
	if err := worker.Work(ctx, &river.Job[ProviderWebhookArgs]{Args: args}); err != nil {
		t.Fatal("exact replay failed", err)
	}
	changed := args
	changed.Payload = []byte(`{"status":"failed"}`)
	if err := worker.Work(ctx, &river.Job[ProviderWebhookArgs]{Args: changed}); err == nil {
		t.Error("changed evidence accepted under processed identity")
	}
	if count.Load() != 1 {
		t.Error("processed replay invoked handler again")
	}
	var state string
	var duplicates int
	if err := pool.QueryRow(ctx, `SELECT state,duplicate_count FROM app.provider_webhook_inbox WHERE provider=$1 AND event_id=$2`, args.Provider, args.EventID).Scan(&state, &duplicates); err != nil || state != "processed" || duplicates != 1 {
		t.Fatalf("state=%s duplicates=%d error=%v", state, duplicates, err)
	}
}
