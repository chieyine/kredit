package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"
)

type ProviderCapabilities struct {
	PersonVerification    bool
	BusinessVerification  bool
	AuthorityVerification bool
	WebhookVerification   bool
}

type PersonVerificationInput struct {
	SubjectID string
	FullName  string
	Phone     string
	Email     string
}

type BusinessVerificationInput struct {
	SubjectID    string
	LegalName    string
	BusinessType string
	Address      string
	Registration string
}

type AuthorityVerificationInput struct {
	SubjectID  string
	PersonID   string
	BusinessID string
	RoleTitle  string
}

type VerificationSession struct {
	Provider          string
	ProviderID        string
	State             string
	VerificationLevel int
	ExpiresAt         time.Time
	SafeResult        map[string]string
}

type ProviderVerification struct {
	ProviderID        string
	SubjectID         string
	State             string
	VerificationLevel int
	Reasons           []string
	SafeResult        map[string]string
	CompletedAt       time.Time
	ExpiresAt         time.Time
}

type VerifiedIdentityEvent struct {
	ProviderID string
	State      string
	SubjectID  string
	OccurredAt time.Time
	SafeResult map[string]string
}

type IdentityProvider interface {
	Name() string
	Capabilities() ProviderCapabilities
	CreatePersonVerification(ctx context.Context, input PersonVerificationInput) (VerificationSession, error)
	CreateBusinessVerification(ctx context.Context, input BusinessVerificationInput) (VerificationSession, error)
	CreateAuthorityVerification(ctx context.Context, input AuthorityVerificationInput) (VerificationSession, error)
	GetVerification(ctx context.Context, providerID string) (ProviderVerification, error)
	VerifyWebhook(ctx context.Context, headers http.Header, body []byte) (VerifiedIdentityEvent, error)
}

// UnavailableProvider is used by non-development runtimes until a certified
// KYC/KYB adapter is configured. It deliberately exposes no capabilities and
// fails every operation instead of silently treating local data as verified.
type UnavailableProvider struct{ reason string }

func NewUnavailableProvider(reason string) *UnavailableProvider {
	if reason == "" {
		reason = "identity provider is not configured"
	}
	return &UnavailableProvider{reason: reason}
}

func (p *UnavailableProvider) Name() string                       { return "unavailable-identity" }
func (p *UnavailableProvider) Capabilities() ProviderCapabilities { return ProviderCapabilities{} }
func (p *UnavailableProvider) CreatePersonVerification(context.Context, PersonVerificationInput) (VerificationSession, error) {
	return VerificationSession{}, errors.New(p.reason)
}
func (p *UnavailableProvider) CreateBusinessVerification(context.Context, BusinessVerificationInput) (VerificationSession, error) {
	return VerificationSession{}, errors.New(p.reason)
}
func (p *UnavailableProvider) CreateAuthorityVerification(context.Context, AuthorityVerificationInput) (VerificationSession, error) {
	return VerificationSession{}, errors.New(p.reason)
}
func (p *UnavailableProvider) GetVerification(context.Context, string) (ProviderVerification, error) {
	return ProviderVerification{}, errors.New(p.reason)
}
func (p *UnavailableProvider) VerifyWebhook(context.Context, http.Header, []byte) (VerifiedIdentityEvent, error) {
	return VerifiedIdentityEvent{}, errors.New(p.reason)
}

type MockProvider struct {
	mu    sync.RWMutex
	cases map[string]ProviderVerification
	now   func() time.Time
}

func NewMockProvider() *MockProvider {
	return &MockProvider{cases: make(map[string]ProviderVerification), now: func() time.Time { return time.Now().UTC() }}
}

func (p *MockProvider) Name() string { return "mock-identity" }

func (p *MockProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{PersonVerification: true, BusinessVerification: true, AuthorityVerification: true, WebhookVerification: true}
}

func (p *MockProvider) CreatePersonVerification(_ context.Context, input PersonVerificationInput) (VerificationSession, error) {
	if input.SubjectID == "" || input.FullName == "" {
		return VerificationSession{}, errors.New("subject id and full name are required")
	}
	return p.create("person", input.SubjectID, map[string]string{"verified_name": input.FullName})
}

func (p *MockProvider) CreateBusinessVerification(_ context.Context, input BusinessVerificationInput) (VerificationSession, error) {
	if input.SubjectID == "" || input.LegalName == "" {
		return VerificationSession{}, errors.New("subject id and legal name are required")
	}
	return p.create("business", input.SubjectID, map[string]string{"verified_name": input.LegalName})
}

func (p *MockProvider) CreateAuthorityVerification(_ context.Context, input AuthorityVerificationInput) (VerificationSession, error) {
	if input.SubjectID == "" || input.PersonID == "" || input.BusinessID == "" {
		return VerificationSession{}, errors.New("authority subject, person, and business are required")
	}
	return p.create("authority", input.SubjectID, map[string]string{"verified_role": input.RoleTitle})
}

func (p *MockProvider) GetVerification(_ context.Context, providerID string) (ProviderVerification, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	verification, ok := p.cases[providerID]
	if !ok {
		return ProviderVerification{}, errors.New("verification not found")
	}
	verification.Reasons = append([]string(nil), verification.Reasons...)
	verification.SafeResult = cloneMap(verification.SafeResult)
	return verification, nil
}

func (p *MockProvider) VerifyWebhook(_ context.Context, _ http.Header, body []byte) (VerifiedIdentityEvent, error) {
	var event VerifiedIdentityEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return VerifiedIdentityEvent{}, err
	}
	if event.ProviderID == "" || event.SubjectID == "" || event.State == "" {
		return VerifiedIdentityEvent{}, errors.New("provider id, subject id, and state are required")
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = p.now()
	}
	return event, nil
}

func (p *MockProvider) create(subjectType, subjectID string, safeResult map[string]string) (VerificationSession, error) {
	providerID := "mock-" + subjectType + "-" + subjectID
	now := p.now()
	verification := ProviderVerification{ProviderID: providerID, SubjectID: subjectID, State: "verified", VerificationLevel: 2, CompletedAt: now, ExpiresAt: now.Add(365 * 24 * time.Hour), SafeResult: cloneMap(safeResult)}
	p.mu.Lock()
	p.cases[providerID] = verification
	p.mu.Unlock()
	return VerificationSession{Provider: p.Name(), ProviderID: providerID, State: verification.State, VerificationLevel: verification.VerificationLevel, ExpiresAt: verification.ExpiresAt, SafeResult: cloneMap(safeResult)}, nil
}

func cloneMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
