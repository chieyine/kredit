package idempotency

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Completion is a one-way transition. A late or duplicate completion must not
// replace the outcome that retries have already observed.
func testCompletionIntegrity(t *testing.T, store Service) {
	t.Helper()
	ctx := t.Context()
	scope, key := "completion-audit:"+uuid.NewString(), uuid.NewString()
	if _, hit, err := store.Reserve(ctx, scope, key, "request-hash"); err != nil || hit {
		t.Fatalf("reserve: hit=%v err=%v", hit, err)
	}
	if err := store.Complete(ctx, scope, key, 201, []byte(`{"reference":"first"}`)); err != nil {
		t.Fatal(err)
	}
	first, hit, err := store.Reserve(ctx, scope, key, "request-hash")
	if err != nil || !hit {
		t.Fatalf("read first result: hit=%v err=%v", hit, err)
	}
	for _, body := range []string{`{"reference":"second"}`, `{"reference":"first"}`} {
		if err := store.Complete(ctx, scope, key, 202, []byte(body)); err == nil {
			t.Error("completed reservation accepted a second completion")
		}
	}
	got, hit, err := store.Reserve(ctx, scope, key, "request-hash")
	if err != nil || !hit || got.Status != first.Status || string(got.ResponseBody) != string(first.ResponseBody) || !got.CompletedAt.Equal(first.CompletedAt) {
		t.Fatalf("completion was overwritten: first=%+v got=%+v hit=%v err=%v", first, got, hit, err)
	}

	// Racing completion attempts may choose either first writer, but exactly
	// one result must win and remain stable after all callers return.
	key = uuid.NewString()
	if _, _, err := store.Reserve(ctx, scope, key, "racing-hash"); err != nil {
		t.Fatal(err)
	}
	const writers = 8
	start := make(chan struct{})
	results := make(chan int, writers)
	var wg sync.WaitGroup
	for i := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if err := store.Complete(ctx, scope, key, 200+i, []byte(`{"ok":true}`)); err != nil {
				results <- 0
				return
			}
			results <- 200 + i
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	var count, status int
	for result := range results {
		if result != 0 {
			count++
			status = result
		}
	}
	if count != 1 {
		t.Errorf("successful completions=%d; want one", count)
	}
	got, _, err = store.Reserve(ctx, scope, key, "racing-hash")
	if err != nil || got.Status != status {
		t.Fatalf("replay does not match the winning completion: status=%d want=%d err=%v", got.Status, status, err)
	}
}

func TestMemoryCompletionIsWriteOnce(t *testing.T) {
	testCompletionIntegrity(t, NewMemoryStore())
}

func TestMemoryCancelledWorkDoesNotReserveOrComplete(t *testing.T) {
	store := NewMemoryStore()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := store.Reserve(ctx, "scope", "cancelled", "hash"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled reserve: %v", err)
	}
	if _, hit, err := store.Reserve(t.Context(), "scope", "cancelled", "hash"); err != nil || hit {
		t.Fatalf("cancelled work created a reservation: hit=%v err=%v", hit, err)
	}
	if err := store.Complete(ctx, "scope", "cancelled", 200, []byte(`{}`)); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled completion: %v", err)
	}
	record, _, err := store.Reserve(t.Context(), "scope", "cancelled", "hash")
	if err != nil || !record.CompletedAt.IsZero() {
		t.Fatalf("cancelled work completed a reservation: %+v %v", record, err)
	}
}

func TestMemoryCompletionRejectsInformationalStatus(t *testing.T) {
	store := NewMemoryStore()
	for _, status := range []int{100, 101, 103, 199} {
		key := uuid.NewString()
		if _, _, err := store.Reserve(t.Context(), "scope", key, "hash"); err != nil {
			t.Fatal(err)
		}
		if err := store.Complete(t.Context(), "scope", key, status, nil); err == nil {
			t.Errorf("informational response %d accepted as final", status)
		}
	}
}

func TestPostgresCompletionIsWriteOnce(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("KREDIT_INTEGRATION=1 and DATABASE_URL are required")
	}
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := NewPostgresStore(pool)
	testCompletionIntegrity(t, store)
	for _, status := range []int{100, 101, 103, 199} {
		key := uuid.NewString()
		if _, _, err := store.Reserve(t.Context(), "status-audit", key, "hash"); err != nil {
			t.Fatal(err)
		}
		if err := store.Complete(t.Context(), "status-audit", key, status, nil); err == nil {
			t.Errorf("informational response %d accepted as final", status)
		}
	}
}
