package contract

import (
	"context"
	"testing"

	"kredit/internal/collections"
	"kredit/internal/mandates"
	"kredit/internal/providers/mono"
)

// An abstraction that has only ever met one API is a guess. This suite is the
// place that keeps the provider boundary honest: every adapter Kredit can enable
// is registered here, and the shared assertions run against all of them.
//
// Adding a second collection provider should be a one-line addition to
// registeredAdapters plus its own package tests. If it is more than that, the
// boundary leaked and that is the finding.
var (
	_ mandates.Provider    = (*mandates.MockProvider)(nil)
	_ collections.Provider = (*collections.MockProvider)(nil)
	_ mandates.Provider    = (*mono.Client)(nil)
	_ collections.Provider = (*mono.Client)(nil)
)

// capableProvider is the full surface Kredit relies on: the collection contract
// plus the capability declaration the engine reads before routing money.
type capableProvider interface {
	collections.Provider
	Capabilities() collections.Capabilities
}

func registeredAdapters(t *testing.T) map[string]capableProvider {
	t.Helper()
	sweep, err := mono.New("https://api.example.test", "test_sk_contract", "webhook-secret", "https://app.example.test/return", true, func(context.Context, string, string) (string, error) {
		return "customer-1", nil
	})
	if err != nil {
		t.Fatalf("mono adapter could not be constructed: %v", err)
	}
	return map[string]capableProvider{
		"mock-collection": collections.NewMockProvider("contract-secret"),
		"mono-sweep":      sweep,
	}
}

// Kredit's collection model is a variable-amount recurring debit authorised in
// advance. An adapter that cannot declare those capabilities cannot carry the
// product, whatever else it supports.
func TestEveryAdapterDeclaresTheCapabilitiesTheProductRequires(t *testing.T) {
	for name, provider := range registeredAdapters(t) {
		t.Run(name, func(t *testing.T) {
			if provider.Name() != name {
				t.Fatalf("adapter name %q does not match its registration %q", provider.Name(), name)
			}
			capabilities := provider.Capabilities()
			if !capabilities.AuthorizationSession {
				t.Fatal("an adapter must support an authorisation session; a mandate is never assumed")
			}
			if !capabilities.Recurring || !capabilities.Variable {
				t.Fatalf("variable recurring debit is the product's collection model: %+v", capabilities)
			}
		})
	}
}

// A forged webhook must never authenticate, on any adapter. This is the single
// assertion whose failure would let an attacker post money.
func TestNoAdapterAcceptsAForgedWebhook(t *testing.T) {
	for name, provider := range registeredAdapters(t) {
		t.Run(name, func(t *testing.T) {
			forged := collections.Webhook{EventID: "forged-1", ExternalReference: "kredit-attempt-1", State: collections.ProviderSettled, Signature: "forged"}
			if provider.VerifyWebhook(forged) {
				t.Fatal("adapter authenticated a forged webhook signature")
			}
			if provider.VerifyWebhook(collections.Webhook{EventID: "unsigned-1", ExternalReference: "kredit-attempt-1"}) {
				t.Fatal("adapter authenticated an unsigned webhook")
			}
		})
	}
}

// The mandate lifecycle runs against every adapter that can be exercised without
// a live provider account. A remote adapter is proven by its own package tests
// and by sandbox certification; the assertions below are the shared contract.
func TestMandateLifecycleContract(t *testing.T) {
	providers := map[string]mandates.Provider{
		"mock": mandates.NewMockProvider(),
	}
	for name, provider := range providers {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			created, err := provider.CreateAuthorizationSession(ctx, mandates.AuthorizationInput{UserID: "buyer-1", BusinessID: "business-1", AmountCeiling: 2_000_000, Purpose: "contract"})
			if err != nil || created.Status != mandates.Active {
				t.Fatalf("create=%+v err=%v", created, err)
			}
			cancelled, err := provider.CancelMandate(ctx, created.ProviderID, "buyer request")
			if err != nil || cancelled.Status != mandates.Cancelled {
				t.Fatalf("cancel=%+v err=%v", cancelled, err)
			}
			// A restored authorisation is a new authorisation. Reusing the
			// cancelled identifier would let a revoked mandate silently come
			// back to life.
			restored, err := provider.RestoreAuthorization(ctx, created.ProviderID)
			if err != nil || restored.Status != mandates.Active || restored.ProviderID == created.ProviderID {
				t.Fatalf("restore=%+v err=%v", restored, err)
			}
		})
	}
}
