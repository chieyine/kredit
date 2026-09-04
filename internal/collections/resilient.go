package collections

import (
	"context"
	"errors"
	"sync"
	"time"
)

const (
	CircuitClosed   = "closed"
	CircuitOpen     = "open"
	CircuitHalfOpen = "half_open"
)

type HealthStatus struct {
	State               string    `json:"state"`
	Healthy             bool      `json:"healthy"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastError           string    `json:"last_error,omitempty"`
	LastFailureAt       time.Time `json:"last_failure_at,omitempty"`
	OpenUntil           time.Time `json:"open_until,omitempty"`
}

type HealthProvider interface{ Health() HealthStatus }

// ResilientProvider protects all provider calls with a small circuit breaker.
// Provider timeouts and transport errors open the circuit; successful calls
// close it again. Business responses such as "failed" are not transport
// failures and remain visible to the collection engine.
type ResilientProvider struct {
	mu            sync.Mutex
	inner         Provider
	failureLimit  int
	cooldown      time.Duration
	failures      int
	lastError     string
	lastFailureAt time.Time
	openUntil     time.Time
	halfOpenProbe bool
	now           func() time.Time
}

func NewResilientProvider(inner Provider, failureLimit int, cooldown time.Duration) *ResilientProvider {
	if failureLimit <= 0 {
		failureLimit = 3
	}
	if cooldown <= 0 {
		cooldown = time.Minute
	}
	return &ResilientProvider{inner: inner, failureLimit: failureLimit, cooldown: cooldown, now: func() time.Time { return time.Now().UTC() }}
}

func (p *ResilientProvider) Name() string {
	if p == nil || p.inner == nil {
		return ""
	}
	return p.inner.Name()
}

func (p *ResilientProvider) Capabilities() Capabilities {
	if p == nil || p.inner == nil {
		return Capabilities{}
	}
	if provider, ok := p.inner.(CapabilityProvider); ok {
		return provider.Capabilities()
	}
	return Capabilities{}
}

func (p *ResilientProvider) Submit(ctx context.Context, request Request) (Response, error) {
	if err := p.allow(); err != nil {
		return Response{}, err
	}
	if p.inner == nil {
		return Response{}, errors.New("collection provider is unavailable")
	}
	response, err := p.inner.Submit(ctx, request)
	if err != nil {
		p.failure(err)
		return Response{}, err
	}
	p.success()
	return response, nil
}

func (p *ResilientProvider) Get(ctx context.Context, providerID string) (Response, error) {
	if err := p.allow(); err != nil {
		return Response{}, err
	}
	if p.inner == nil {
		return Response{}, errors.New("collection provider is unavailable")
	}
	response, err := p.inner.Get(ctx, providerID)
	if err != nil {
		p.failure(err)
		return Response{}, err
	}
	p.success()
	return response, nil
}
func (p *ResilientProvider) Cancel(ctx context.Context, providerID string) (Response, error) {
	if err := p.allow(); err != nil {
		return Response{}, err
	}
	provider, ok := p.inner.(CancellationProvider)
	if !ok {
		return Response{}, errors.New("collection provider does not permit cancellation")
	}
	response, err := provider.Cancel(ctx, providerID)
	if err != nil {
		p.failure(err)
		return Response{}, err
	}
	p.success()
	return response, nil
}

func (p *ResilientProvider) VerifyWebhook(event Webhook) bool {
	if p == nil || p.inner == nil {
		return false
	}
	return p.inner.VerifyWebhook(event)
}

func (p *ResilientProvider) Sign(event Webhook) string {
	if p == nil || p.inner == nil {
		return ""
	}
	if signer, ok := p.inner.(WebhookSigner); ok {
		return signer.Sign(event)
	}
	return ""
}

func (p *ResilientProvider) Health() HealthStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.now()
	state := CircuitClosed
	if !p.openUntil.IsZero() && now.Before(p.openUntil) {
		state = CircuitOpen
	} else if p.failures >= p.failureLimit {
		state = CircuitHalfOpen
	}
	return HealthStatus{State: state, Healthy: state != CircuitOpen, ConsecutiveFailures: p.failures, LastError: p.lastError, LastFailureAt: p.lastFailureAt, OpenUntil: p.openUntil}
}

func (p *ResilientProvider) allow() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.inner == nil {
		return errors.New("collection provider is unavailable")
	}
	now := p.now()
	if !p.openUntil.IsZero() && now.Before(p.openUntil) {
		return errors.New("collection provider circuit is open")
	}
	if p.failures >= p.failureLimit {
		if p.halfOpenProbe {
			return errors.New("collection provider circuit probe is in progress")
		}
		p.halfOpenProbe = true
	}
	return nil
}

func (p *ResilientProvider) failure(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failures++
	p.lastError = err.Error()
	p.lastFailureAt = p.now()
	p.halfOpenProbe = false
	if p.failures >= p.failureLimit {
		p.openUntil = p.now().Add(p.cooldown)
	}
}

func (p *ResilientProvider) success() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failures = 0
	p.lastError = ""
	p.openUntil = time.Time{}
	p.halfOpenProbe = false
}

func (p *ResilientProvider) GetByReference(ctx context.Context, request Request) (Response, error) {
	if err := p.allow(); err != nil {
		return Response{}, err
	}
	var response Response
	var err error
	if provider, ok := p.inner.(ReferenceLookupProvider); ok {
		response, err = provider.GetByReference(ctx, request)
	} else if request.CollectionReference != "" {
		response, err = p.inner.Get(ctx, request.CollectionReference)
	} else {
		err = errors.New("provider reference lookup is unavailable")
	}
	if err != nil {
		p.failure(err)
		return Response{}, err
	}
	p.success()
	return response, nil
}
