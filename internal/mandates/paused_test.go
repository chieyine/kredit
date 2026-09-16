package mandates

import (
	"context"
	"testing"
)

func TestPausedProviderAllowsCancellationButNotNewPermission(t *testing.T) {
	ctx := context.Background()
	original := NewMockProvider()
	m, e := original.CreateAuthorizationSession(ctx, AuthorizationInput{UserID: "buyer", BusinessID: "business", AmountCeiling: 1000})
	if e != nil {
		t.Fatal(e)
	}
	paused := NewPausedProvider(original)
	if _, e = paused.CreateAuthorizationSession(ctx, AuthorizationInput{}); e == nil {
		t.Fatal("new authorization allowed while paused")
	}
	if _, e = paused.GetMandate(ctx, m.ProviderID); e != nil {
		t.Fatal(e)
	}
	if _, e = paused.CancelMandate(ctx, m.ProviderID, "customer cancelled"); e != nil {
		t.Fatal(e)
	}
	if _, e = paused.RestoreAuthorization(ctx, m.ProviderID); e == nil {
		t.Fatal("restore created authority while paused")
	}
}
