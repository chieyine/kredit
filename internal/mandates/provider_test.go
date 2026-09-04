package mandates

import (
	"context"
	"testing"
)

func TestPostgresProviderFailsClosedWithoutDatabase(t *testing.T) {
	provider := NewPostgresProvider(nil, "test")
	if _, err := provider.CreateAuthorizationSession(context.Background(), AuthorizationInput{UserID: "user-1", BusinessID: "business-1", AmountCeiling: 1}); err == nil {
		t.Fatal("expected missing database error")
	}
	if _, err := provider.GetMandate(context.Background(), "mandate-1"); err == nil {
		t.Fatal("expected missing database error")
	}
}

func TestMockMandateActivatesWithCeiling(t *testing.T) {
	provider := NewMockProvider()
	mandate, err := provider.CreateAuthorizationSession(context.Background(), AuthorizationInput{UserID: "user-1", BusinessID: "business-1", AmountCeiling: 120000000, Purpose: "one-time-credit"})
	if err != nil {
		t.Fatal(err)
	}
	if mandate.Status != Active || mandate.AmountCeiling != 120000000 {
		t.Fatalf("unexpected mandate: %+v", mandate)
	}
	if _, err := provider.GetMandate(context.Background(), mandate.ProviderID); err != nil {
		t.Fatal(err)
	}
}

func TestMockProviderResolvesTradeLineMandateByOwner(t *testing.T) {
	provider := NewMockProvider()
	mandate, err := provider.CreateAuthorizationSession(context.Background(), AuthorizationInput{SupplierOrganizationID: "supplier-1", UserID: "buyer-1", BusinessID: "business-1", AmountCeiling: 100000, Purpose: "recurring"})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := provider.ResolveTradeLineMandate(context.Background(), mandate.ID, "buyer-1", "business-1", "supplier-1")
	if err != nil || resolved.ID != mandate.ID || resolved.Status != Active {
		t.Fatalf("resolved=%#v err=%v", resolved, err)
	}
	if _, err := provider.ResolveTradeLineMandate(context.Background(), mandate.ID, "other-buyer", "business-1", "supplier-1"); err == nil {
		t.Fatal("expected owner mismatch to be rejected")
	}
	if _, err := provider.ResolveTradeLineMandate(context.Background(), mandate.ID, "buyer-1", "business-1", "other-supplier"); err == nil {
		t.Fatal("expected supplier mismatch to be rejected")
	}
}
