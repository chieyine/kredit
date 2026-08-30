package identifier

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNew(t *testing.T) {
	id1 := New()
	if id1 == "" {
		t.Fatal("expected non-empty id")
	}
	parsed1, err := uuid.Parse(id1)
	if err != nil {
		t.Fatalf("expected valid uuid, got err: %v", err)
	}
	if parsed1.Version() != 7 {
		t.Fatalf("expected uuid v7, got version %d", parsed1.Version())
	}
	if parsed1.Variant() != uuid.RFC4122 {
		t.Fatalf("expected RFC4122 variant, got %v", parsed1.Variant())
	}

	time.Sleep(2 * time.Millisecond)
	id2 := New()
	parsed2, err := uuid.Parse(id2)
	if err != nil {
		t.Fatalf("expected valid uuid, got err: %v", err)
	}
	if id1 == id2 {
		t.Fatalf("expected distinct ids, got duplicates: %s", id1)
	}
	if strings.Compare(id1, id2) >= 0 {
		t.Fatalf("expected id1 < id2 for monotonic time ordering, got id1=%s, id2=%s", id1, id2)
	}
	_ = parsed2
}

func TestFromKey(t *testing.T) {
	k1 := FromKey("mandate", "ref-123")
	k2 := FromKey("mandate", "ref-123")
	k3 := FromKey("mandate", "ref-456")
	k4 := FromKey("invoice", "ref-123")

	if k1 != k2 {
		t.Fatalf("expected deterministic output for identical input: %s vs %s", k1, k2)
	}
	if k1 == k3 {
		t.Fatalf("expected different outputs for different values: %s vs %s", k1, k3)
	}
	if k1 == k4 {
		t.Fatalf("expected different outputs for different scopes: %s vs %s", k1, k4)
	}
	if _, err := uuid.Parse(k1); err != nil {
		t.Fatalf("expected valid uuid string, got %v", err)
	}
}
