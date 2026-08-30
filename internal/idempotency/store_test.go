package idempotency

import (
	"context"
	"testing"
)

func TestMemoryStoreRejectsKeyReuseWithDifferentRequest(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	if _, hit, err := store.Reserve(ctx, "payments", "key-1", "hash-1"); err != nil || hit {
		t.Fatalf("first reserve = hit=%v err=%v", hit, err)
	}
	if _, _, err := store.Reserve(ctx, "payments", "key-1", "hash-2"); err == nil {
		t.Fatal("expected request hash mismatch")
	}
}

func TestMemoryStoreValidatesCompletionAndReplaysExactResponse(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	if err := store.Complete(ctx, "payments", "missing", 200, []byte(`{}`)); err == nil {
		t.Fatal("expected a missing reservation error")
	}
	if _, _, err := store.Reserve(ctx, "payments", "key-1", "hash-1"); err != nil {
		t.Fatal(err)
	}
	if err := store.Complete(ctx, "payments", "key-1", 0, []byte(`{}`)); err == nil {
		t.Fatal("expected an invalid response status error")
	}
	body := []byte(`{"payment_id":"pay-1"}`)
	if err := store.Complete(ctx, "payments", "key-1", 201, body); err != nil {
		t.Fatal(err)
	}
	record, existing, err := store.Reserve(ctx, "payments", "key-1", "hash-1")
	if err != nil || !existing || record.Status != 201 || string(record.ResponseBody) != string(body) || record.CompletedAt.IsZero() {
		t.Fatalf("unexpected replay: existing=%v record=%+v err=%v", existing, record, err)
	}
}
