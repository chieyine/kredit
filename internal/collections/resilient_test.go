package collections

import (
	"context"
	"errors"
	"testing"
	"time"
)

type failingProvider struct{ failures int }

func (p *failingProvider) Name() string { return "failing" }
func (p *failingProvider) Submit(context.Context, Request) (Response, error) {
	p.failures++
	return Response{}, errors.New("provider timeout")
}
func (p *failingProvider) Get(context.Context, string) (Response, error) {
	return Response{}, errors.New("provider timeout")
}
func (p *failingProvider) VerifyWebhook(Webhook) bool { return false }

func TestResilientProviderOpensAndRecoversCircuit(t *testing.T) {
	now := time.Now().UTC()
	base := &failingProvider{}
	provider := NewResilientProvider(base, 2, time.Minute)
	provider.now = func() time.Time { return now }
	if _, err := provider.Submit(context.Background(), Request{AmountKobo: 100}); err == nil {
		t.Fatal("expected provider failure")
	}
	if _, err := provider.Submit(context.Background(), Request{AmountKobo: 100}); err == nil {
		t.Fatal("expected provider failure")
	}
	if status := provider.Health(); status.State != CircuitOpen || status.Healthy {
		t.Fatalf("status=%+v", status)
	}
	if _, err := provider.Submit(context.Background(), Request{AmountKobo: 100}); err == nil {
		t.Fatal("open circuit should reject calls")
	}
	now = now.Add(2 * time.Minute)
	if status := provider.Health(); status.State != CircuitHalfOpen {
		t.Fatalf("status=%+v", status)
	}
}
