package collections

import (
	"context"
	"errors"
	"testing"
	"time"

	"kredit/internal/ledger"
	"kredit/internal/payments"
)

type timeoutOnlyProvider struct{}

func (timeoutOnlyProvider) Name() string { return "timeout-provider" }
func (timeoutOnlyProvider) Submit(context.Context, Request) (Response, error) {
	return Response{}, errors.New("provider timeout")
}
func (timeoutOnlyProvider) Get(context.Context, string) (Response, error) {
	return Response{State: ProviderSucceeded, ProviderCollectionID: "attacker-selected", SucceededAmountKobo: 100000}, nil
}
func (timeoutOnlyProvider) Sign(Webhook) string              { return "valid" }
func (timeoutOnlyProvider) VerifyWebhook(event Webhook) bool { return event.Signature == "valid" }

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
	retry, err := engine.Retry(context.Background(), attempt.ID, attempt.NextRetryAt)
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

func TestPublicWebhookSignalCannotRecognizeMoney(t *testing.T) {
	engine, provider, paymentStore, snapshot := testEngine(t)
	provider.SetNextResponse(Response{State: ProviderPending, ProviderCollectionID: "signal-1"})
	attempt, err := engine.Start(context.Background(), "obl-1", "signal-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	event := Webhook{EventID: "public-success", ExternalReference: attempt.ExternalReference, State: ProviderSucceeded, ProviderCollectionID: "signal-1", SucceededAmountKobo: 100000}
	event.Signature = provider.Sign(event)
	result, err := engine.SignalWebhook(context.Background(), event)
	if err != nil {
		t.Fatal(err)
	}
	if result.State == AttemptSucceeded || snapshot.OutstandingKobo != 100000 {
		t.Fatalf("callback changed financial state: attempt=%+v outstanding=%d", result, snapshot.OutstandingKobo)
	}
	if got, err := paymentStore.List("obl-1"); err != nil || len(got) != 0 {
		t.Fatalf("callback created payment: payments=%+v err=%v", got, err)
	}
	provider.SetProviderResponse("signal-1", Response{State: ProviderSucceeded, ProviderCollectionID: "signal-1", SucceededAmountKobo: 100000})
	result, err = engine.Reconcile(context.Background(), attempt.ID)
	if err != nil || result.State != AttemptSucceeded || snapshot.OutstandingKobo != 0 {
		t.Fatalf("verified reconciliation failed: attempt=%+v outstanding=%d err=%v", result, snapshot.OutstandingKobo, err)
	}
}

func TestWebhookCannotSupplyIdentityForAmbiguousSubmission(t *testing.T) {
	snapshot := &ObligationSnapshot{ID: "obl-1", BuyerUserID: "buyer", Currency: "NGN", Active: true, OutstandingKobo: 100000, MandateActive: true, MandateRemainingKobo: 100000, CollectionEnabled: true, ProviderSupported: true, Version: 1}
	paymentStore := payments.NewStore(ledger.NewStore(), func(string) (payments.ObligationSnapshot, error) {
		return payments.ObligationSnapshot{ID: snapshot.ID, BuyerUserID: snapshot.BuyerUserID, PrincipalKobo: 100000, OutstandingKobo: snapshot.OutstandingKobo, Currency: "NGN"}, nil
	}, func(_ string, delta ledger.Money) error { snapshot.OutstandingKobo += delta; return nil })
	engine := NewEngine(timeoutOnlyProvider{}, paymentStore, func(string) (ObligationSnapshot, error) { return *snapshot, nil }, func(string, time.Time) (ledger.Money, error) { return 100000, nil })
	attempt, err := engine.Start(context.Background(), "obl-1", "ambiguous-submit", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	event := Webhook{EventID: "forged-signal", ExternalReference: attempt.ExternalReference, ProviderCollectionID: "attacker-selected", State: ProviderSucceeded, SucceededAmountKobo: 100000, Signature: "valid"}
	signalled, err := engine.SignalWebhook(context.Background(), event)
	if err != nil {
		t.Fatal(err)
	}
	if signalled.ProviderCollectionID != "" {
		t.Fatalf("callback supplied provider identity %q", signalled.ProviderCollectionID)
	}
	if _, err = engine.Reconcile(context.Background(), attempt.ID); err == nil {
		t.Fatal("ambiguous submission reconciled through callback-supplied identity")
	}
	if snapshot.OutstandingKobo != 100000 {
		t.Fatalf("callback changed outstanding balance to %d", snapshot.OutstandingKobo)
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

func TestProcessingReconciliationKeepsReservation(t *testing.T) {
	e, p, _, snapshot := testEngine(t)
	p.SetNextResponse(Response{State: ProviderPending, ProviderCollectionID: "processing"})
	a, err := e.Start(context.Background(), "obl-1", "processing", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	p.SetProviderResponse("processing", Response{State: ProviderPending, ProviderCollectionID: "processing"})
	a, err = e.Reconcile(context.Background(), a.ID)
	if err != nil || a.State != AttemptSubmitted {
		t.Fatalf("pending became %s: %v", a.State, err)
	}
	if snapshot.OutstandingKobo != 100000 {
		t.Fatal("processing changed balance")
	}
	if _, err = e.Start(context.Background(), "obl-1", "another-key", time.Now()); err == nil {
		t.Fatal("pending reservation allowed duplicate debit")
	}
}
func TestRetryBudgetAndDelayCannotBeBypassedWithNewKeys(t *testing.T) {
	e, p, _, _ := testEngine(t)
	e.SetMaxRetries(2)
	p.SetNextResponse(Response{State: ProviderFailed, Retryable: true})
	a, err := e.Start(context.Background(), "obl-1", "first-failure", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.Retry(context.Background(), a.ID, time.Now()); err == nil {
		t.Fatal("retry delay was bypassed")
	}
	if _, err = e.Start(context.Background(), "obl-1", "new-key", a.NextRetryAt); err == nil {
		t.Fatal("new key bypassed retry workflow")
	}
	p.SetNextResponse(Response{State: ProviderFailed, Retryable: true})
	b, err := e.Retry(context.Background(), a.ID, a.NextRetryAt)
	if err != nil || b.AttemptNumber != 2 {
		t.Fatalf("retry=%+v err=%v", b, err)
	}
	if _, err = e.Retry(context.Background(), b.ID, b.NextRetryAt); err == nil {
		t.Fatal("retry budget was bypassed")
	}
}

func TestReconciliationFlagsLocalSuccessNotConfirmedByProvider(t *testing.T) {
	engine, provider, _, snapshot := testEngine(t)
	attempt, err := engine.Start(context.Background(), "obl-1", "success-mismatch", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	provider.SetProviderResponse(attempt.ProviderCollectionID, Response{State: ProviderPending, ProviderCollectionID: attempt.ProviderCollectionID})
	if _, err = engine.Reconcile(context.Background(), attempt.ID); err == nil {
		t.Fatal("unconfirmed local success was silently accepted")
	}
	if snapshot.OutstandingKobo != 0 {
		t.Fatal("reconciliation changed the immutable payment effect")
	}
}

func TestOptimizeCollectionWindowRollsWeekendToMonday0500WAT(t *testing.T) {
	lagos := time.FixedZone("WAT", 3600)
	// Saturday 2026-09-05 14:00 WAT -> Should roll to Monday 2026-09-07 05:00 WAT
	sat := time.Date(2026, 9, 5, 14, 0, 0, 0, lagos)
	optSat := OptimizeCollectionWindow(sat).In(lagos)
	if optSat.Weekday() != time.Monday || optSat.Hour() != 5 || optSat.Minute() != 0 {
		t.Fatalf("expected Monday 05:00 WAT, got %v", optSat)
	}

	// Sunday 2026-09-06 20:00 WAT -> Should roll to Monday 2026-09-07 05:00 WAT
	sun := time.Date(2026, 9, 6, 20, 0, 0, 0, lagos)
	optSun := OptimizeCollectionWindow(sun).In(lagos)
	if optSun.Weekday() != time.Monday || optSun.Hour() != 5 || optSun.Minute() != 0 {
		t.Fatalf("expected Monday 05:00 WAT, got %v", optSun)
	}

	// Tuesday 2026-09-08 04:00 WAT -> Early morning before 5am -> Tuesday 05:00 WAT
	tue := time.Date(2026, 9, 8, 4, 0, 0, 0, lagos)
	optTue := OptimizeCollectionWindow(tue).In(lagos)
	if optTue.Weekday() != time.Tuesday || optTue.Hour() != 5 {
		t.Fatalf("expected Tuesday 05:00 WAT, got %v", optTue)
	}
}
