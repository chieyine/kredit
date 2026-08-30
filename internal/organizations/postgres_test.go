package organizations

import (
	"encoding/hex"
	"testing"

	"kredit/internal/access"

	"github.com/google/uuid"
)

func TestPostgresStoreUsesOpaqueTargetHashesAndUUIDv7IDs(t *testing.T) {
	store := NewPostgresStore(nil, "test-key")
	first := store.hashTarget("email", "Owner@example.test")
	second := store.hashTarget("email", "owner@example.test")
	if hex.EncodeToString(first) != hex.EncodeToString(second) {
		t.Fatal("target hashing must normalize identifiers")
	}
	if len(first) != 32 {
		t.Fatalf("unexpected target hash length: %d", len(first))
	}
	phoneA := store.hashTarget("phone", "+234 801-234-5678")
	phoneB := store.hashTarget("phone", "+2348012345678")
	if hex.EncodeToString(phoneA) != hex.EncodeToString(phoneB) {
		t.Fatal("phone formatting differences must normalize before hashing")
	}
	id, err := uuid.Parse(newUUID())
	if err != nil || id.Version() != 7 {
		t.Fatalf("organization identifiers must be UUIDv7: id=%s err=%v", id, err)
	}
}

func TestPostgresStoreImplementsOrganizationService(t *testing.T) {
	var _ Service = NewPostgresStore(nil, "test-key")
	if !access.RoleOwner.Valid() {
		t.Fatal("access role contract unavailable")
	}
}
