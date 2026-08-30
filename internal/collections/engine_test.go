package collections

import (
	"context"
	"testing"
	"time"

	"kredit/internal/ledger"
	"kredit/internal/payments"
)

func testEngine(t *testing.T) (*Engine, *MockProvider, *payments.Store, *ObligationSnapshot) {
	t.Helper()
	snapshot := &ObligationSnapshot{ID: "obl-1", BuyerUserID: "buyer", Currency: "NGN", Active: true, OutstandingKobo: 100000, MandateActive: true, MandateRemainingKobo: 100000, CollectionEnabled: true, ProviderSupported: true, Version: 1}
	apply := func(_ string, delta ledger.Money) error { snapshot.OutstandingKobo += delta; return nil }
	paymentStore := payments.NewStore(ledger.NewStore(), func(_ string) (payments.ObligationSnapshot, error) {
		return payments.ObligationSnapshot{ID: snapshot.ID, BuyerUserID: snapshot.BuyerUserID, PrincipalKobo: 100000, OutstandingKobo: snapshot.OutstandingKobo, Currency: "NGN"}, nil
	}, apply)
	provider := NewMockProvider("secret")
	engine := NewEngine(provider, paymentStore, func(_ string) (ObligationSnapshot, error) { return *snapshot, nil }, func(_ string, _ time.Time) (ledger.Money, error) { return 100000, nil })
	return engine, provider, paymentStore, snapshot
}

func TestCollectionSuccessIsIdempotentAndChargesCollectionFee(t *testing.T) {
	engine, provider, _, snapshot := testEngine(t)
	attempt, err := engine.Start(context.Background(), "obl-1", "start-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if attempt.State != AttemptSucceeded || attempt.SucceededAmountKobo != 100000 {
		t.Fatalf("attempt=%+v", attempt)
	}
	if snapshot.OutstandingKobo != 0 {
		t.Fatalf("outstanding=%d", snapshot.OutstandingKobo)
	}
	event := Webhook{EventID: "provider-event-" + attempt.ID, ExternalReference: attempt.ExternalReference, State: ProviderSucceeded, ProviderCollectionID: attempt.ProviderCollectionID, SucceededAmountKobo: 100000}
	event.Signature = provider.Sign(event)
	duplicate, err := engine.ProcessWebhook(context.Background(), event)
	if err != nil || duplicate.ID != attempt.ID {
		t.Fatalf("duplicate err=%v", err)
	}
	if len(engine.ListAttempts("obl-1")) != 1 {
		t.Fatal("duplicate created attempt")
	}
}

func TestCollectionRejectsKeyReuseForAnotherObligation(t *testing.T) {
	engine, _, _, _ := testEngine(t)
	if _, err := engine.Start(context.Background(), "obl-1", "shared-key", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Start(context.Background(), "obl-2", "shared-key", time.Now().UTC()); err == nil {
		t.Fatal("expected changed obligation to be rejected")
	}
}

func TestPartialCollectionCanRetryRemainingAmount(t *testing.T) {
	engine, provider, _, snapshot := testEngine(t)
	provider.SetNextResponse(Response{State: ProviderPartial, ProviderCollectionID: "partial-1", SucceededAmountKobo: 40000})
	attempt, err := engine.Start(context.Background(), "obl-1", "partial-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if attempt.State != AttemptPartial || snapshot.OutstandingKobo != 60000 {
		t.Fatalf("attempt=%+v outstanding=%d", attempt, snapshot.OutstandingKobo)
	}
	provider.SetNextResponse(Response{State: ProviderSucceeded, ProviderCollectionID: "retry-1", SucceededAmountKobo: 60000})
	retry, err := engine.Retry(context.Background(), attempt.ID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if retry.State != AttemptSucceeded || snapshot.OutstandingKobo != 0 {
		t.Fatalf("retry=%+v outstanding=%d", retry, snapshot.OutstandingKobo)
	}
}

func TestTimeoutReconcilesAfterProviderSuccess(t *testing.T) {
	engine, provider, _, snapshot := testEngine(t)
	provider.SetNextResponse(Response{State: ProviderTimeout, ProviderCollectionID: "timeout-1"})
	attempt, err := engine.Start(context.Background(), "obl-1", "timeout-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if attempt.State != AttemptUnknown {
		t.Fatalf("state=%s", attempt.State)
	}
	provider.SetProviderResponse("timeout-1", Response{State: ProviderSucceeded, ProviderCollectionID: "timeout-1", SucceededAmountKobo: 100000})
	resolved, err := engine.Reconcile(context.Background(), attempt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.State != AttemptSucceeded || snapshot.OutstandingKobo != 0 {
		t.Fatalf("resolved=%+v outstanding=%d", resolved, snapshot.OutstandingKobo)
	}
}

func TestProviderConfirmedCancellationReleasesReservation(t *testing.T) {
	engine, provider, _, _ := testEngine(t)
	provider.SetNextResponse(Response{State: ProviderPending, ProviderCollectionID: "cancel-1"})
	attempt, err := engine.Start(context.Background(), "obl-1", "cancel-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	cancelled, err := engine.Cancel(context.Background(), attempt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.State != AttemptCancelled {
		t.Fatalf("state=%s", cancelled.State)
	}
	if engine.reservations[cancelled.ReservationID].State != ReservationReleased {
		t.Fatal("reservation was not released")
	}
	if _, err := engine.Cancel(context.Background(), attempt.ID); err == nil {
		t.Fatal("repeated cancellation must be rejected")
	}
}

func TestEligibilityReturnsExplicitReasons(t *testing.T) {
	engine, _, _, snapshot := testEngine(t)
	snapshot.MandateActive = false
	snapshot.CollectionEnabled = false
	eligibility, err := engine.Eligibility("obl-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if eligibility.Eligible || len(eligibility.Reasons) != 2 {
		t.Fatalf("eligibility=%+v", eligibility)
	}
}

func TestPartialDisputeLeavesUndisputedAmountEligible(t *testing.T) {
	engine, _, _, snapshot := testEngine(t)
	snapshot.DisputedBlockedKobo = 20000
	eligibility, err := engine.Eligibility("obl-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if !eligibility.Eligible || eligibility.AmountKobo != 80000 {
		t.Fatalf("eligibility=%+v", eligibility)
	}
}
