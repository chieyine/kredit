package collections

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"kredit/internal/ledger"
)

type Capability string

const (
	CapabilityAuthorization Capability = "authorization_session"
	CapabilityOneTime       Capability = "one_time_collection"
	CapabilityRecurring     Capability = "recurring_collection"
	CapabilityVariable      Capability = "variable_amount_collection"
	CapabilitySettlement    Capability = "settlement_reconciliation"
	CapabilityReversal      Capability = "reversal"
	CapabilityMultiAccount  Capability = "multi_account_collection"
)

type Capabilities struct {
	AuthorizationSession bool `json:"authorization_session"`
	OneTime              bool `json:"one_time_collection"`
	Recurring            bool `json:"recurring_collection"`
	Variable             bool `json:"variable_amount_collection"`
	Settlement           bool `json:"settlement_reconciliation"`
	Reversal             bool `json:"reversal"`
	MultiAccount         bool `json:"multi_account_collection"`
	PartialRecovery      bool `json:"partial_recovery"`
	AutomaticRetries     bool `json:"automatic_retries"`
}

// ReferenceLookupProvider reconciles ambiguous submissions without requiring a
// response ID. The original persisted request is the authoritative identity.
type ReferenceLookupProvider interface {
	GetByReference(context.Context, Request) (Response, error)
}

type CapabilityProvider interface{ Capabilities() Capabilities }
type WebhookSigner interface{ Sign(Webhook) string }
type CancellationProvider interface {
	Cancel(context.Context, string) (Response, error)
}

const (
	ProviderSucceeded         = "succeeded"
	ProviderPartial           = "partial"
	ProviderFailed            = "failed"
	ProviderPending           = "pending"
	ProviderTimeout           = "timeout"
	ProviderSettled           = "settled"
	ProviderSettlementPending = "settlement_pending"
	ProviderReversed          = "reversed"
)

type Request struct {
	CollectionReference string
	MandateReference    string
	ExternalReference   string
	ObligationID        string
	BuyerUserID         string
	AmountKobo          ledger.Money
	Currency            string
}
type Response struct {
	State                string
	ProviderCollectionID string
	SucceededAmountKobo  ledger.Money
	FailureCode          string
	Retryable            bool
	SettlementState      string
	SettlementReference  string
}
type Webhook struct {
	EventID              string
	ExternalReference    string
	State                string
	ProviderCollectionID string
	SucceededAmountKobo  ledger.Money
	FailureCode          string
	Retryable            bool
	SettlementState      string
	SettlementReference  string
	Signature            string
}
type Provider interface {
	Name() string
	Submit(context.Context, Request) (Response, error)
	Get(context.Context, string) (Response, error)
	VerifyWebhook(Webhook) bool
}

type MockProvider struct {
	mu        sync.Mutex
	secret    []byte
	next      *Response
	responses map[string]Response
	now       func() time.Time
}

func NewMockProvider(secret string) *MockProvider {
	return &MockProvider{secret: []byte(secret), responses: map[string]Response{}, now: func() time.Time { return time.Now().UTC() }}
}
func (p *MockProvider) Name() string { return "mock-collection" }
func (p *MockProvider) Capabilities() Capabilities {
	return Capabilities{AuthorizationSession: true, OneTime: true, Recurring: true, Variable: true, Settlement: true, Reversal: true, MultiAccount: false}
}
func (p *MockProvider) SetNextResponse(response Response) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.next = &response
}
func (p *MockProvider) Submit(_ context.Context, request Request) (Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.next != nil {
		response := *p.next
		p.next = nil
		if response.ProviderCollectionID == "" {
			response.ProviderCollectionID = "mock-collection-" + fmt.Sprint(p.now().UnixNano())
		}
		return response, nil
	}
	return Response{State: ProviderSucceeded, ProviderCollectionID: "mock-collection-" + fmt.Sprint(p.now().UnixNano()), SucceededAmountKobo: request.AmountKobo}, nil
}
func (p *MockProvider) Get(_ context.Context, providerID string) (Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	response, ok := p.responses[providerID]
	if !ok {
		return Response{State: ProviderPending, ProviderCollectionID: providerID}, nil
	}
	return response, nil
}
func (p *MockProvider) Cancel(_ context.Context, providerID string) (Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	response := Response{State: ProviderReversed, ProviderCollectionID: providerID}
	p.responses[providerID] = response
	return response, nil
}
func (p *MockProvider) SetProviderResponse(providerID string, response Response) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.responses[providerID] = response
}
func (p *MockProvider) Sign(event Webhook) string {
	event.Signature = ""
	payload, _ := json.Marshal(event)
	mac := hmac.New(sha256.New, p.secret)
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
func (p *MockProvider) VerifyWebhook(event Webhook) bool {
	if len(p.secret) == 0 {
		return event.Signature == ""
	}
	expected := p.Sign(event)
	return hmac.Equal([]byte(expected), []byte(event.Signature))
}
