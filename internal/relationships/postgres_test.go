package relationships

import (
	"context"
	"testing"
)

func TestPostgresStoreFailsClosedWithoutDatabase(t *testing.T) {
	store := NewPostgresStore(nil)
	if _, err := store.Record(context.Background(), "user-1", "org-1", "history", "v1", "hash", true); err == nil {
		t.Fatal("expected missing database error")
	}
	if got, err := store.List(context.Background(), "user-1"); err == nil || got != nil {
		t.Fatal("missing database must return an error, not a successful empty permission history")
	}
}

func TestStoreImplementsService(t *testing.T) {
	var _ Service = NewStore()
}
