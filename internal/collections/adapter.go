package collections

import (
	"context"
	"errors"
	"strings"
	"time"
)

// ApprovalRecord is the minimum evidence required before a provider adapter
// can be enabled. A provider contract or legal/compliance approval reference
// is intentionally explicit rather than inferred from configuration.
type ApprovalRecord struct {
	ProviderName        string       `json:"provider_name"`
	WrittenReference    string       `json:"written_reference"`
	ApprovedBy          string       `json:"approved_by"`
	ApprovedAt          time.Time    `json:"approved_at"`
	AllowedCapabilities []Capability `json:"allowed_capabilities"`
	PilotLimitKobo      int64        `json:"pilot_limit_kobo"`
}

func (a ApprovalRecord) Valid(providerName string, now time.Time) error {
	if strings.TrimSpace(providerName) == "" || a.ProviderName != providerName {
		return errors.New("approval provider does not match adapter")
	}
	if strings.TrimSpace(a.WrittenReference) == "" || strings.TrimSpace(a.ApprovedBy) == "" || a.ApprovedAt.IsZero() {
		return errors.New("written provider approval is required")
	}
	if a.ApprovedAt.After(now) {
		return errors.New("provider approval cannot be in the future")
	}
	if a.PilotLimitKobo <= 0 {
		return errors.New("a positive pilot limit is required")
	}
	if len(a.AllowedCapabilities) == 0 {
		return errors.New("at least one provider capability must be approved")
	}
	return nil
}

func (a ApprovalRecord) Allows(capability Capability) bool {
	for _, allowed := range a.AllowedCapabilities {
		if allowed == capability {
			return true
		}
	}
	return false
}

// ApprovedAdapter is the seam for a real provider SDK. It delegates only
// after the feature flag and written approval have both been satisfied.
type ApprovedAdapter struct {
	inner     Provider
	approval  ApprovalRecord
	featureOn bool
	now       func() time.Time
}

func NewApprovedAdapter(inner Provider, approval ApprovalRecord, featureOn bool) *ApprovedAdapter {
	return &ApprovedAdapter{inner: inner, approval: approval, featureOn: featureOn, now: func() time.Time { return time.Now().UTC() }}
}

func (a *ApprovedAdapter) Name() string {
	if a.inner == nil {
		return ""
	}
	return a.inner.Name()
}
func (a *ApprovedAdapter) Capabilities() Capabilities {
	if cp, ok := a.inner.(CapabilityProvider); ok {
		return cp.Capabilities()
	}
	return Capabilities{}
}
func (a *ApprovedAdapter) Enabled() bool {
	return a.featureOn && a.inner != nil && a.approval.Valid(a.Name(), a.now()) == nil
}
func (a *ApprovedAdapter) Approval() ApprovalRecord { return a.approval }
func (a *ApprovedAdapter) Submit(ctx context.Context, request Request) (Response, error) {
	if err := a.gate(int64(request.AmountKobo)); err != nil {
		return Response{}, err
	}
	return a.inner.Submit(ctx, request)
}
func (a *ApprovedAdapter) Get(ctx context.Context, providerID string) (Response, error) {
	if !a.Enabled() {
		return Response{}, errors.New("real collection provider is disabled")
	}
	return a.inner.Get(ctx, providerID)
}
func (a *ApprovedAdapter) Cancel(ctx context.Context, providerID string) (Response, error) {
	if !a.Enabled() || !a.approval.Allows(CapabilityReversal) {
		return Response{}, errors.New("collection cancellation is not approved")
	}
	provider, ok := a.inner.(CancellationProvider)
	if !ok {
		return Response{}, errors.New("collection provider does not permit cancellation")
	}
	return provider.Cancel(ctx, providerID)
}
func (a *ApprovedAdapter) VerifyWebhook(event Webhook) bool {
	return a.inner != nil && a.inner.VerifyWebhook(event)
}
func (a *ApprovedAdapter) Sign(event Webhook) string {
	if signer, ok := a.inner.(WebhookSigner); ok {
		return signer.Sign(event)
	}
	return ""
}

func (a *ApprovedAdapter) gate(amount int64) error {
	if !a.featureOn {
		return errors.New("real collection feature is disabled")
	}
	if a.inner == nil {
		return errors.New("collection provider is unavailable")
	}
	if err := a.approval.Valid(a.Name(), a.now()); err != nil {
		return err
	}
	if !a.approval.Allows(CapabilityOneTime) {
		return errors.New("one-time collection capability is not approved")
	}
	if a.approval.PilotLimitKobo > 0 && amount > a.approval.PilotLimitKobo {
		return errors.New("collection amount exceeds approved pilot limit")
	}
	return nil
}
