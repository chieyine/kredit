package collections

import (
	"context"
	"testing"
	"time"
)

type racingProvider struct {
	*MockProvider
	duringSubmit func(Request)
	duringCancel func()
}

func (p *racingProvider) Submit(_ context.Context, r Request) (Response, error) {
	p.duringSubmit(r)
	return Response{State: ProviderPending, ProviderCollectionID: "provider-id"}, nil
}
func (p *racingProvider) Cancel(_ context.Context, id string) (Response, error) {
	p.duringCancel()
	return Response{State: ProviderReversed, ProviderCollectionID: id}, nil
}

type multiAccountProvider struct{ *MockProvider }

func (p *multiAccountProvider) Capabilities() Capabilities { return Capabilities{MultiAccount: true} }

func TestReservationWrapperPreservesPolicyCapabilities(t *testing.T) {
	e, _, _, s := testEngine(t)
	s.CollectionPolicy = PolicyStandard
	e.provider = &reserveProvider{Provider: &multiAccountProvider{NewMockProvider("secret")}}
	if _, err := e.Start(context.Background(), s.ID, "blocked-policy", time.Now()); err == nil {
		t.Fatal("reservation wrapper bypassed staged-policy restriction")
	}
}
func TestLateSubmissionCannotDowngradeConfirmedPayment(t *testing.T) {
	e, base, _, s := testEngine(t)
	p := &racingProvider{MockProvider: base}
	e.provider = p
	p.duringSubmit = func(r Request) {
		event := Webhook{EventID: "bank-first", ExternalReference: r.ExternalReference, ProviderCollectionID: "provider-id", State: ProviderSucceeded, SucceededAmountKobo: r.AmountKobo}
		event.Signature = p.Sign(event)
		if _, err := e.ProcessWebhook(context.Background(), event); err != nil {
			t.Fatal(err)
		}
	}
	a, err := e.Start(context.Background(), s.ID, "response-race", time.Now())
	if err != nil || a.State != AttemptSucceeded || s.OutstandingKobo != 0 {
		t.Fatalf("confirmed payment overwritten: %+v %v", a, err)
	}
}
func TestCancellationCannotOverwriteConcurrentSuccess(t *testing.T) {
	e, base, _, s := testEngine(t)
	base.SetNextResponse(Response{State: ProviderPending, ProviderCollectionID: "provider-id"})
	a, err := e.Start(context.Background(), s.ID, "cancel-race", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	p := &racingProvider{MockProvider: base}
	e.provider = p
	p.duringCancel = func() {
		event := Webhook{EventID: "success-during-cancel", ExternalReference: a.ExternalReference, ProviderCollectionID: a.ProviderCollectionID, State: ProviderSucceeded, SucceededAmountKobo: a.RequestedAmountKobo}
		event.Signature = p.Sign(event)
		if _, err := e.ProcessWebhook(context.Background(), event); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = e.Cancel(context.Background(), a.ID); err == nil {
		t.Fatal("cancel overwrote concurrent payment")
	}
	a, _ = e.GetAttempt(a.ID)
	if a.State != AttemptSucceeded {
		t.Fatal(a.State)
	}
}
func TestConnectorSignatureBindsAllFinancialFields(t *testing.T) {
	p, err := NewWebhookProvider("connector", "https://provider.example", "token", "secret")
	if err != nil {
		t.Fatal(err)
	}
	original := Webhook{EventID: "event", ExternalReference: "attempt", ProviderCollectionID: "bank-id", State: ProviderFailed, FailureCode: "final", Retryable: false}
	original.Signature = p.Sign(original)
	for _, mutate := range []func(*Webhook){func(e *Webhook) { e.ProviderCollectionID = "other" }, func(e *Webhook) { e.Retryable = true }, func(e *Webhook) { e.FailureCode = "temporary" }} {
		tampered := original
		mutate(&tampered)
		if p.VerifyWebhook(tampered) {
			t.Fatal("unsigned field accepted")
		}
	}
}

func TestReconcilePreservesSettlementProgressAndReversal(t *testing.T) {
	e, p, _, snapshot := testEngine(t)
	ctx := context.Background()
	attempt, err := e.Start(ctx, snapshot.ID, "settlement-test", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	response := Response{State: ProviderSucceeded, ProviderCollectionID: attempt.ProviderCollectionID, SucceededAmountKobo: attempt.SucceededAmountKobo, SettlementState: ProviderSettlementPending}
	p.mu.Lock()
	p.responses[attempt.ProviderCollectionID] = response
	p.mu.Unlock()
	if _, err = e.Reconcile(ctx, attempt.ID); err != nil {
		t.Fatal(err)
	}
	response.SettlementState = ProviderSettled
	response.SettlementReference = "settlement-reference"
	p.mu.Lock()
	p.responses[attempt.ProviderCollectionID] = response
	p.mu.Unlock()
	updated, err := e.Reconcile(ctx, attempt.ID)
	if err != nil || updated.SettlementState != ProviderSettled || updated.SettlementReference != "settlement-reference" {
		t.Fatalf("settlement lookup lost: %+v %v", updated, err)
	}
	response.State = ProviderSettlementPending
	response.SettlementState = ""
	p.mu.Lock()
	p.responses[attempt.ProviderCollectionID] = response
	p.mu.Unlock()
	updated, err = e.Reconcile(ctx, attempt.ID)
	if err != nil || updated.SettlementState != ProviderSettled {
		t.Fatal("late pending downgraded settlement", err)
	}
	response.State = ProviderReversed
	p.mu.Lock()
	p.responses[attempt.ProviderCollectionID] = response
	p.mu.Unlock()
	updated, err = e.Reconcile(ctx, attempt.ID)
	if err != nil || updated.SettlementState != ProviderReversed {
		t.Fatalf("reversal not surfaced: %+v %v", updated, err)
	}
	if snapshot.OutstandingKobo != 0 {
		t.Fatal("settlement metadata fabricated a payment reversal")
	}
}
