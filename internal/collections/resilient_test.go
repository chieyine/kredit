package collections

import (
	"context"
	"errors"
	"strings"
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
	if status := provider.Health(); status.State != CircuitHalfOpen || status.Healthy {
		t.Fatalf("status=%+v", status)
	}
}

func TestProviderHealthDoesNotExposeCredentialsOrClaimMissingProviderHealthy(t *testing.T) {
	if status := NewResilientProvider(nil, 1, time.Minute).Health(); status.Healthy {
		t.Fatal("missing provider reported healthy")
	}
	provider := NewResilientProvider(&failingProvider{}, 1, time.Minute)
	provider.failure(0, errors.New("request failed token=sensitive-value password=private-value"))
	status := provider.Health()
	if strings.Contains(status.LastError, "sensitive-value") || strings.Contains(status.LastError, "private-value") {
		t.Fatal("provider credentials exposed in health response")
	}
}

func TestUnsupportedCancellationDoesNotConsumeRecoveryProbe(t *testing.T) {
	now := time.Now().UTC()
	base := &failingProvider{}
	provider := NewResilientProvider(base, 1, time.Minute)
	provider.now = func() time.Time { return now }
	_, _ = provider.Submit(context.Background(), Request{})
	now = now.Add(2 * time.Minute)
	if _, err := provider.Cancel(context.Background(), "collection"); err == nil {
		t.Fatal("unsupported cancellation must be rejected")
	}
	_, _ = provider.Submit(context.Background(), Request{})
	if base.failures != 2 {
		t.Fatal("unsupported cancellation stranded the recovery probe")
	}
}

type delayedSuccessProvider struct {
	*failingProvider
	started chan struct{}
	release chan struct{}
}

func (p *delayedSuccessProvider) Submit(ctx context.Context, request Request) (Response, error) {
	if request.ExternalReference != "slow" {
		return Response{}, errors.New("provider timeout")
	}
	close(p.started)
	select {
	case <-p.release:
		return Response{State: ProviderPending}, nil
	case <-ctx.Done():
		return Response{}, ctx.Err()
	}
}

func TestOlderSuccessfulRequestCannotCloseNewlyOpenedCircuit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	base := &delayedSuccessProvider{failingProvider: &failingProvider{}, started: make(chan struct{}), release: make(chan struct{})}
	provider := NewResilientProvider(base, 1, time.Minute)
	finished := make(chan error, 1)
	go func() { _, err := provider.Submit(ctx, Request{ExternalReference: "slow"}); finished <- err }()
	select {
	case <-base.started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if _, err := provider.Submit(ctx, Request{}); err == nil {
		t.Fatal("expected transport failure")
	}
	close(base.release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	if status := provider.Health(); status.State != CircuitOpen || status.Healthy {
		t.Fatalf("older response cleared the new circuit failure: %+v", status)
	}
}
