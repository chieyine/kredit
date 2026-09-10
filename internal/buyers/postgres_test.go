package buyers

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"kredit/internal/identity"
)

func TestPortalReadDistinguishesAbsentProfileFromFailure(t *testing.T) {
	memory := NewStore("test-key", identity.NewMockProvider())
	if _, err := memory.ReadPortal(context.Background(), "unknown"); !errors.Is(err, ErrPortalNotFound) {
		t.Fatalf("missing profile: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := memory.ReadPortal(ctx, "unknown"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled read: %v", err)
	}
	store := NewPostgresStore(nil, "test-key", identity.NewMockProvider())
	if _, err := store.ReadPortal(context.Background(), "unknown"); err == nil || errors.Is(err, ErrPortalNotFound) {
		t.Fatalf("database failure must remain distinct from missing profile: %v", err)
	}
}

func TestPostgresBuyerStoreImplementsServiceAndFailsClosedWithoutDatabase(t *testing.T) {
	store := NewPostgresStore(nil, "test-key", identity.NewMockProvider())
	var _ Service = store
	if got := store.CountBusinesses(); got != 0 {
		t.Fatalf("nil database must report no durable businesses, got %d", got)
	}
	if _, err := store.CreateInvitation("user-1", "org-1", CreateInvitationInput{Target: "buyer@example.test", TargetType: "email", LegalName: "Buyer Ltd", BusinessType: "limited_company", BusinessAddress: "Lagos", Industry: "pharmacy"}); err == nil {
		t.Fatal("create invitation must fail closed without a database")
	}
}

func TestPostgresBuyerStoreEncryptionRoundTrip(t *testing.T) {
	store := NewPostgresStore(nil, "test-key", identity.NewMockProvider())
	ciphertext, err := store.encrypt([]byte("buyer@example.test"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(ciphertext, []byte("buyer@example.test")) {
		t.Fatal("invitation target must not be stored as plaintext")
	}
	plaintext, err := store.decrypt(ciphertext)
	if err != nil || string(plaintext) != "buyer@example.test" {
		t.Fatalf("round trip failed: %q %v", plaintext, err)
	}
	if _, err := store.Accept(context.Background(), "", "", AcceptInput{}); err == nil {
		t.Fatal("acceptance must validate identity before touching storage")
	}
}
