package organizations

import (
	"testing"

	"kredit/internal/access"

	"github.com/google/uuid"
)

func TestPostgresStoreUsesUUIDv7IDs(t *testing.T) {
	id, err := uuid.Parse(newUUID())
	if err != nil || id.Version() != 7 {
		t.Fatalf("organization identifiers must be UUIDv7: id=%s err=%v", id, err)
	}
}

func TestPostgresStoreImplementsOrganizationService(t *testing.T) {
	var _ Service = NewPostgresStore(nil)
	if !access.RoleOwner.Valid() {
		t.Fatal("access role contract unavailable")
	}
}
