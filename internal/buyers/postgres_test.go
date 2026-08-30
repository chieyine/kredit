package buyers

import (
	"bytes"
	"context"
	"testing"

	"kredit/internal/identity"
)

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
