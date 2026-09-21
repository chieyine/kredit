package collections

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type circuitAuditProvider struct{ calls atomic.Int32 }

func (p *circuitAuditProvider) Name() string { return "circuit-audit" }
func (p *circuitAuditProvider) Submit(context.Context, Request) (Response, error) {
	p.calls.Add(1)
	return Response{State: ProviderPending}, nil
}
func (p *circuitAuditProvider) Get(_ context.Context, id string) (Response, error) {
	p.calls.Add(1)
	return Response{State: ProviderPending, ProviderCollectionID: id}, nil
}
func (p *circuitAuditProvider) Cancel(context.Context, string) (Response, error) {
	p.calls.Add(1)
	return Response{State: ProviderFailed}, nil
}
func (p *circuitAuditProvider) VerifyWebhook(Webhook) bool { return false }

func TestCancelledProviderWorkDoesNotReachProviderOrChangeHealth(t *testing.T) {
	for _, operation := range []string{"submit", "get", "cancel", "reference"} {
		t.Run(operation, func(t *testing.T) {
			inner := &circuitAuditProvider{}
			provider := NewResilientProvider(inner, 1, time.Minute)
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			var err error
			switch operation {
			case "submit":
				_, err = provider.Submit(ctx, Request{})
			case "get":
				_, err = provider.Get(ctx, "saved-collection")
			case "cancel":
				_, err = provider.Cancel(ctx, "saved-collection")
			case "reference":
				_, err = provider.GetByReference(ctx, Request{CollectionReference: "saved-collection"})
			}
			if !errors.Is(err, context.Canceled) || inner.calls.Load() != 0 {
				t.Fatalf("cancelled request reached provider: calls=%d err=%v", inner.calls.Load(), err)
			}
			if status := provider.Health(); status.State != CircuitClosed || status.ConsecutiveFailures != 0 {
				t.Fatalf("cancelled request changed health: %+v", status)
			}
		})
	}
}

func TestCancelledRequestCannotConsumeHalfOpenProbe(t *testing.T) {
	inner := &circuitAuditProvider{}
	provider := NewResilientProvider(inner, 1, time.Minute)
	now := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	provider.now = func() time.Time { return now }
	provider.failure(0, errors.New("synthetic transport failure"))
	now = now.Add(2 * time.Minute)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := provider.Get(ctx, "saved-collection"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled probe: %v", err)
	}
	if status := provider.Health(); status.State != CircuitHalfOpen || status.ConsecutiveFailures != 1 || provider.halfOpenProbe {
		t.Fatalf("cancelled call consumed recovery probe: %+v", status)
	}
	if _, err := provider.Get(t.Context(), "saved-collection"); err != nil || inner.calls.Load() != 1 {
		t.Fatalf("real recovery probe failed: calls=%d err=%v", inner.calls.Load(), err)
	}
	if status := provider.Health(); !status.Healthy || status.State != CircuitClosed {
		t.Fatalf("real probe did not restore health: %+v", status)
	}
}

func TestUnsupportedReferenceLookupIsNotAProviderFailure(t *testing.T) {
	inner := &circuitAuditProvider{}
	provider := NewResilientProvider(inner, 1, time.Minute)
	if _, err := provider.GetByReference(t.Context(), Request{ExternalReference: "unconfirmed"}); err == nil {
		t.Fatal("unsupported reference lookup succeeded")
	}
	if status := provider.Health(); status.State != CircuitClosed || status.ConsecutiveFailures != 0 || inner.calls.Load() != 0 {
		t.Fatalf("local capability error changed provider health: %+v", status)
	}
	got, err := provider.GetByReference(t.Context(), Request{CollectionReference: "saved-collection"})
	if err != nil || got.ProviderCollectionID != "saved-collection" || inner.calls.Load() != 1 {
		t.Fatalf("supported ID fallback stopped working: %+v %v", got, err)
	}
}
