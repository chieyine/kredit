package contract

import (
	"context"
	"testing"

	"kredit/internal/mandates"
)

// This suite is intentionally provider-neutral. Every approved adapter is
// expected to run these lifecycle assertions in addition to its signed-webhook
// collection contract in internal/collections/provider_contract_test.go.
func TestMandateLifecycleContract(t *testing.T) {
	ctx := context.Background()
	provider := mandates.NewMockProvider()
	created, err := provider.CreateAuthorizationSession(ctx, mandates.AuthorizationInput{UserID: "buyer-1", BusinessID: "business-1", AmountCeiling: 2_000_000, Purpose: "contract"})
	if err != nil || created.Status != mandates.Active {
		t.Fatalf("create=%+v err=%v", created, err)
	}
	cancelled, err := provider.CancelMandate(ctx, created.ProviderID, "buyer request")
	if err != nil || cancelled.Status != mandates.Cancelled {
		t.Fatalf("cancel=%+v err=%v", cancelled, err)
	}
	restored, err := provider.RestoreAuthorization(ctx, created.ProviderID)
	if err != nil || restored.Status != mandates.Active || restored.ProviderID == created.ProviderID {
		t.Fatalf("restore=%+v err=%v", restored, err)
	}
}
