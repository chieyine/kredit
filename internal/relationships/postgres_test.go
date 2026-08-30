package relationships

import "testing"

func TestPostgresStoreFailsClosedWithoutDatabase(t *testing.T) {
	store := NewPostgresStore(nil)
	if _, err := store.Record("user-1", "org-1", "history", "v1", "hash", true); err == nil {
		t.Fatal("expected missing database error")
	}
	if got := store.List("user-1"); got == nil {
		t.Fatal("expected empty list, not nil")
	}
}

func TestStoreImplementsService(t *testing.T) {
	var _ Service = NewStore()
}
