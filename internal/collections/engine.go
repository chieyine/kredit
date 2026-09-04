package collections

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"kredit/internal/identifier"
	"kredit/internal/ledger"
	"kredit/internal/payments"
)

const (
	AttemptPending        = "PENDING"
	AttemptSubmitted      = "SUBMITTED"
	AttemptUnknown        = "UNKNOWN"
	AttemptSucceeded      = "SUCCEEDED"
	AttemptPartial        = "PARTIAL"
	AttemptFailed         = "FAILED"
	AttemptCancelled      = "CANCELLED"
	ReservationProcessing = "PROCESSING"
	ReservationReleased   = "RELEASED"
	ReservationCompleted  = "COMPLETED"
)

type Eligibility struct {
	Eligible   bool         `json:"eligible"`
	AmountKobo ledger.Money `json:"amount_kobo"`
	Reasons    []string     `json:"reasons"`
}
type ObligationSnapshot struct {
	CollectionPolicy     string
	MandateReference     string
	ID                   string
	BuyerUserID          string
	Currency             string
	Active               bool
	OutstandingKobo      ledger.Money
	DueAmountKobo        ledger.Money
	MandateActive        bool
	MandateRemainingKobo ledger.Money
	CollectionEnabled    bool
	ComplianceHold       bool
	BuyerPaymentHold     bool
	BuyerPaymentHoldKobo ledger.Money
	ProviderSupported    bool
	DisputedBlockedKobo  ledger.Money
	Version              int64
}
type SnapshotFunc func(string) (ObligationSnapshot, error)
type DueFunc func(string, time.Time) (ledger.Money, error)
type CollectionReservation struct {
	ID                 string       `json:"id"`
	ObligationID       string       `json:"obligation_id"`
	ScheduleItemID     string       `json:"schedule_item_id,omitempty"`
	OutstandingVersion int64        `json:"outstanding_version"`
	ReservedAmountKobo ledger.Money `json:"reserved_amount_kobo"`
	State              string       `json:"state"`
	ExpiresAt          time.Time    `json:"expires_at"`
	IdempotencyKey     string       `json:"idempotency_key"`
	CreatedAt          time.Time    `json:"created_at"`
}
type Attempt struct {
	MandateReference     string       `json:"mandate_reference,omitempty"`
	NextRetryAt          time.Time    `json:"next_retry_at,omitempty"`
	ID                   string       `json:"id"`
	ReservationID        string       `json:"reservation_id"`
	ObligationID         string       `json:"obligation_id"`
	Provider             string       `json:"provider"`
	ProviderCollectionID string       `json:"provider_collection_id,omitempty"`
	ExternalReference    string       `json:"external_reference"`
	RequestedAmountKobo  ledger.Money `json:"requested_amount_kobo"`
	SucceededAmountKobo  ledger.Money `json:"succeeded_amount_kobo"`
	State                string       `json:"state"`
	AttemptNumber        int          `json:"attempt_number"`
	RetryClassification  string       `json:"retry_classification,omitempty"`
	FailureCode          string       `json:"failure_code,omitempty"`
	RequestedAt          time.Time    `json:"requested_at"`
	FinalAt              time.Time    `json:"final_at,omitempty"`
	SettlementState      string       `json:"settlement_state,omitempty"`
	SettlementReference  string       `json:"settlement_reference,omitempty"`
}

type Engine struct {
	mu             sync.Mutex
	provider       Provider
	payments       payments.Service
	snapshot       SnapshotFunc
	due            DueFunc
	reservations   map[string]*CollectionReservation
	attempts       map[string]*Attempt
	byKey          map[string]string
	byExternal     map[string]string
	events         map[string]bool
	now            func() time.Time
	featureEnabled bool
	reservationTTL time.Duration
	maxRetries     int
}

type Service interface {
	ProviderStatus() ProviderStatus
	SetFeatureEnabled(bool)
	SetMaxRetries(int)
	Eligibility(string, time.Time) (Eligibility, error)
	Start(context.Context, string, string, time.Time) (Attempt, error)
	SignalWebhook(context.Context, Webhook) (Attempt, error)
	ProcessWebhook(context.Context, Webhook) (Attempt, error)
	Reconcile(context.Context, string) (Attempt, error)
	Retry(context.Context, string, time.Time) (Attempt, error)
	Cancel(context.Context, string) (Attempt, error)
	GetAttempt(string) (Attempt, bool)
	ListAttempts(string) []Attempt
}

var _ Service = (*Engine)(nil)

func NewEngine(provider Provider, paymentStore payments.Service, snapshot SnapshotFunc, due DueFunc) *Engine {
	return &Engine{provider: provider, payments: paymentStore, snapshot: snapshot, due: due, reservations: map[string]*CollectionReservation{}, attempts: map[string]*Attempt{}, byKey: map[string]string{}, byExternal: map[string]string{}, events: map[string]bool{}, now: func() time.Time { return time.Now().UTC() }, featureEnabled: true, reservationTTL: 30 * time.Minute, maxRetries: 3}
}

type ProviderStatus struct {
	Name           string       `json:"name"`
	FeatureEnabled bool         `json:"feature_enabled"`
	Capabilities   Capabilities `json:"capabilities"`
	Health         HealthStatus `json:"health"`
}

func (e *Engine) ProviderStatus() ProviderStatus {
	e.mu.Lock()
	defer e.mu.Unlock()
	status := ProviderStatus{FeatureEnabled: e.featureEnabled}
	if e.provider != nil {
		status.Name = e.provider.Name()
		if cp, ok := e.provider.(CapabilityProvider); ok {
			status.Capabilities = cp.Capabilities()
		}
		if hp, ok := e.provider.(HealthProvider); ok {
			status.Health = hp.Health()
		} else {
			status.Health = HealthStatus{State: CircuitClosed, Healthy: true}
		}
	}
	return status
}
func (e *Engine) SetFeatureEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.featureEnabled = enabled
}
func (e *Engine) SetMaxRetries(max int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if max > 0 {
		e.maxRetries = max
	}
}
func (e *Engine) Eligibility(obligationID string, now time.Time) (Eligibility, error) {
	if e.snapshot == nil {
		return Eligibility{}, errors.New("collection dependencies unavailable")
	}
	snapshot, err := e.snapshot(obligationID)
	if err != nil {
		return Eligibility{}, err
	}
	return e.eligibilityForSnapshot(obligationID, now, snapshot)
}

func (e *Engine) eligibilityForSnapshot(obligationID string, now time.Time, snapshot ObligationSnapshot) (Eligibility, error) {
	if e.provider == nil || e.due == nil {
		return Eligibility{}, errors.New("collection dependencies unavailable")
	}
	amount, err := e.due(obligationID, now)
	if err != nil {
		return Eligibility{}, err
	}
	reasons := []string{}
	capabilities := Capabilities{}
	if provider, ok := e.provider.(CapabilityProvider); ok {
		capabilities = provider.Capabilities()
	}
	if err := ValidatePolicy(snapshot.CollectionPolicy, capabilities); err != nil {
		reasons = append(reasons, "collection_policy_not_supported")
	}

	if !snapshot.Active {
		reasons = append(reasons, "obligation_inactive")
	}
	if snapshot.OutstandingKobo <= 0 {
		reasons = append(reasons, "no_outstanding_balance")
	}
	if amount <= 0 {
		reasons = append(reasons, "nothing_due")
	}
	if !snapshot.MandateActive {
		reasons = append(reasons, "mandate_inactive")
	}
	if snapshot.MandateRemainingKobo <= 0 {
		reasons = append(reasons, "mandate_ceiling_exhausted")
	}
	if !snapshot.CollectionEnabled {
		reasons = append(reasons, "supplier_collection_disabled")
	}
	if snapshot.ComplianceHold {
		reasons = append(reasons, "compliance_hold")
	}
	if snapshot.BuyerPaymentHold {
		reasons = append(reasons, "buyer_payment_hold")
	}
	if !snapshot.ProviderSupported {
		reasons = append(reasons, "provider_unsupported")
	}
	e.mu.Lock()
	for _, reservation := range e.reservations {
		if reservation.ObligationID == obligationID && (reservation.State == ReservationProcessing || reservation.State == ReservationCompleted) {
			reasons = append(reasons, "active_collection_reservation")
			break
		}
	}
	enabled := e.featureEnabled
	e.mu.Unlock()
	if !enabled {
		reasons = append(reasons, "feature_disabled")
	}
	target := amount
	if target > snapshot.OutstandingKobo {
		target = snapshot.OutstandingKobo
	}
	if target > snapshot.MandateRemainingKobo {
		target = snapshot.MandateRemainingKobo
	}
	if snapshot.DisputedBlockedKobo > 0 {
		target -= snapshot.DisputedBlockedKobo
		if target < 0 {
			target = 0
		}
		if target == 0 {
			reasons = append(reasons, "dispute_blocks_due_amount")
		}
	}
	if snapshot.BuyerPaymentHoldKobo > 0 {
		target -= snapshot.BuyerPaymentHoldKobo
		if target < 0 {
			target = 0
		}
		if target == 0 {
			reasons = append(reasons, "buyer_payment_claim_blocks_due_amount")
		}
	}
	return Eligibility{Eligible: len(reasons) == 0, AmountKobo: target, Reasons: reasons}, nil
}

func (e *Engine) Start(ctx context.Context, obligationID, idempotencyKey string, now time.Time) (Attempt, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return Attempt{}, errors.New("idempotency key is required")
	}
	e.mu.Lock()
	if existing := e.byKey[obligationID+"\x00"+idempotencyKey]; existing != "" {
		out := cloneAttempt(*e.attempts[existing])
		e.mu.Unlock()
		if out.ObligationID != obligationID {
			return Attempt{}, errors.New("idempotency key was reused for a different collection")
		}
		return out, nil
	}
	e.mu.Unlock()
	if e.snapshot == nil {
		return Attempt{}, errors.New("collection dependencies unavailable")
	}
	snapshot, err := e.snapshot(obligationID)
	if err != nil {
		return Attempt{}, err
	}
	eligibility, err := e.eligibilityForSnapshot(obligationID, now, snapshot)
	if err != nil {
		return Attempt{}, err
	}
	if !eligibility.Eligible {
		return Attempt{}, fmt.Errorf("collection ineligible: %s", strings.Join(eligibility.Reasons, ","))
	}
	e.mu.Lock()
	if existing := e.byKey[obligationID+"\x00"+idempotencyKey]; existing != "" {
		out := cloneAttempt(*e.attempts[existing])
		e.mu.Unlock()
		if out.ObligationID != obligationID {
			return Attempt{}, errors.New("idempotency key was reused for a different collection")
		}
		return out, nil
	}
	if !e.featureEnabled {
		e.mu.Unlock()
		return Attempt{}, errors.New("collection feature is disabled")
	}
	for _, reservation := range e.reservations {
		if reservation.ObligationID == obligationID && (reservation.State == ReservationProcessing || reservation.State == ReservationCompleted) {
			e.mu.Unlock()
			return Attempt{}, errors.New("collection reservation already active")
		}
	}
	var latest *Attempt
	for _, previous := range e.attempts {
		if previous.ObligationID == obligationID && (latest == nil || previous.RequestedAt.After(latest.RequestedAt)) {
			latest = previous
		}
	}
	if latest != nil && (latest.State == AttemptFailed || latest.State == AttemptPartial) {
		if idempotencyKey != "retry:"+latest.ID || latest.RetryClassification == "final" || latest.AttemptNumber >= e.maxRetries || (!latest.NextRetryAt.IsZero() && now.Before(latest.NextRetryAt)) {
			e.mu.Unlock()
			return Attempt{}, errors.New("collection requires an eligible controlled retry")
		}
	}
	attemptID := identifier.FromKey("collection-attempt:"+obligationID, idempotencyKey)
	reservation := &CollectionReservation{ID: identifier.FromKey("collection-reservation:"+obligationID, idempotencyKey), ObligationID: obligationID, OutstandingVersion: snapshot.Version, ReservedAmountKobo: eligibility.AmountKobo, State: ReservationProcessing, ExpiresAt: e.now().Add(e.reservationTTL), IdempotencyKey: idempotencyKey, CreatedAt: e.now()}
	number := 1
	if latest != nil && idempotencyKey == "retry:"+latest.ID {
		number = latest.AttemptNumber + 1
	}
	attempt := &Attempt{MandateReference: snapshot.MandateReference, ID: attemptID, ReservationID: reservation.ID, ObligationID: obligationID, Provider: e.provider.Name(), ExternalReference: "kredit-" + attemptID, RequestedAmountKobo: eligibility.AmountKobo, State: AttemptPending, AttemptNumber: number, RequestedAt: e.now()}
	e.reservations[reservation.ID] = reservation
	e.attempts[attempt.ID] = attempt
	e.byKey[obligationID+"\x00"+idempotencyKey] = attempt.ID
	e.byExternal[attempt.ExternalReference] = attempt.ID
	e.mu.Unlock()
	response, submitErr := e.provider.Submit(ctx, Request{MandateReference: attempt.MandateReference, ExternalReference: attempt.ExternalReference, ObligationID: obligationID, BuyerUserID: snapshot.BuyerUserID, AmountKobo: attempt.RequestedAmountKobo, Currency: snapshot.Currency})
	if submitErr != nil {
		response = Response{State: ProviderTimeout}
	}
	event := Webhook{EventID: "provider-event-" + attempt.ID, ExternalReference: attempt.ExternalReference, State: response.State, ProviderCollectionID: response.ProviderCollectionID, SucceededAmountKobo: response.SucceededAmountKobo, FailureCode: response.FailureCode, Retryable: response.Retryable, SettlementState: response.SettlementState, SettlementReference: response.SettlementReference}
	if signer, ok := e.provider.(WebhookSigner); ok {
		event.Signature = signer.Sign(event)
	}
	if _, err = e.ProcessWebhook(ctx, event); err != nil {
		return Attempt{}, err
	}
	e.mu.Lock()
	out := cloneAttempt(*attempt)
	e.mu.Unlock()
	return out, nil
}

// SignalWebhook authenticates and records a public callback without trusting
// callback-supplied money or terminal state. Reconcile performs the independent
// provider lookup that is allowed to change financial state.
func (e *Engine) SignalWebhook(_ context.Context, event Webhook) (Attempt, error) {
	if e.provider == nil || event.EventID == "" || event.ExternalReference == "" || !e.provider.VerifyWebhook(event) {
		return Attempt{}, errors.New("invalid collection webhook signature")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	attempt := e.attempts[e.byExternal[event.ExternalReference]]
	if attempt == nil {
		return Attempt{}, errors.New("collection attempt not found")
	}
	if e.events[event.EventID] {
		return cloneAttempt(*attempt), nil
	}
	if attempt.ProviderCollectionID != "" && event.ProviderCollectionID != "" && attempt.ProviderCollectionID != event.ProviderCollectionID {
		return Attempt{}, errors.New("provider collection identity changed; reconciliation required")
	}
	if attempt.State == AttemptPending {
		attempt.State = AttemptSubmitted
	}
	e.events[event.EventID] = true
	return cloneAttempt(*attempt), nil
}

func (e *Engine) ProcessWebhook(_ context.Context, event Webhook) (Attempt, error) {
	if e.provider == nil || event.EventID == "" || event.ExternalReference == "" || !e.provider.VerifyWebhook(event) {
		return Attempt{}, errors.New("invalid collection webhook signature")
	}
	e.mu.Lock()
	if e.events[event.EventID] {
		id := e.byExternal[event.ExternalReference]
		attempt := e.attempts[id]
		if attempt == nil {
			e.mu.Unlock()
			return Attempt{}, errors.New("collection attempt not found")
		}
		out := cloneAttempt(*attempt)
		e.mu.Unlock()
		return out, nil
	}
	attemptID := e.byExternal[event.ExternalReference]
	attempt := e.attempts[attemptID]
	if attempt == nil {
		e.mu.Unlock()
		return Attempt{}, errors.New("collection attempt not found")
	}
	if event.State == ProviderFailed && event.SucceededAmountKobo != 0 {
		e.mu.Unlock()
		return Attempt{}, errors.New("failed outcome contains recovered money; reconciliation required")
	}
	if attempt.ProviderCollectionID != "" && event.ProviderCollectionID != "" && attempt.ProviderCollectionID != event.ProviderCollectionID {
		e.mu.Unlock()
		return Attempt{}, errors.New("provider collection identity changed; reconciliation required")
	}
	if attempt.SettlementState == ProviderReversed && event.State != ProviderReversed {
		event.SettlementState = ""
		event.SettlementReference = ""
		if event.State == ProviderSettled || event.State == ProviderSettlementPending {
			out := cloneAttempt(*attempt)
			e.mu.Unlock()
			return out, nil
		}
	}
	if attempt.SettlementState == ProviderSettled && (event.State == ProviderSettlementPending || event.SettlementState == ProviderSettlementPending) {
		event.SettlementState = ""
		event.SettlementReference = ""
		if event.State == ProviderSettlementPending {
			out := cloneAttempt(*attempt)
			e.mu.Unlock()
			return out, nil
		}
	}
	if event.State == ProviderSettled || event.State == ProviderSettlementPending || event.State == ProviderReversed {
		attempt.SettlementState = event.State
		attempt.SettlementReference = event.SettlementReference
		e.events[event.EventID] = true
		out := cloneAttempt(*attempt)
		e.mu.Unlock()
		return out, nil
	}
	if attempt.State == AttemptSucceeded || attempt.State == AttemptPartial || attempt.State == AttemptFailed {
		if (event.State == ProviderSucceeded || event.State == ProviderPartial) && event.SucceededAmountKobo != attempt.SucceededAmountKobo || event.State == ProviderFailed && attempt.SucceededAmountKobo > 0 {
			e.mu.Unlock()
			return Attempt{}, errors.New("provider outcome conflicts with recorded financial result; controlled reconciliation required")
		}
		if event.SettlementState != "" {
			attempt.SettlementState = event.SettlementState
			attempt.SettlementReference = event.SettlementReference
		}
		e.events[event.EventID] = true
		out := cloneAttempt(*attempt)
		e.mu.Unlock()
		return out, nil
	}
	if event.State == ProviderPending || event.State == ProviderTimeout {
		if event.ProviderCollectionID != "" {
			attempt.ProviderCollectionID = event.ProviderCollectionID
		}
		if event.State == ProviderTimeout {
			attempt.State = AttemptUnknown
		} else {
			attempt.State = AttemptSubmitted
		}
		e.events[event.EventID] = true
		out := cloneAttempt(*attempt)
		e.mu.Unlock()
		return out, nil
	}
	if event.State != ProviderSucceeded && event.State != ProviderPartial && event.State != ProviderFailed {
		e.mu.Unlock()
		return Attempt{}, errors.New("unrecognized provider outcome; reconciliation required")
	}
	if event.SucceededAmountKobo < 0 || event.SucceededAmountKobo > attempt.RequestedAmountKobo {
		e.mu.Unlock()
		return Attempt{}, errors.New("provider amount outside reserved amount")
	}
	if event.State == ProviderSucceeded || event.State == ProviderPartial {
		if event.SucceededAmountKobo == 0 {
			e.mu.Unlock()
			return Attempt{}, errors.New("successful collection amount must be positive")
		}
		if e.payments == nil {
			e.mu.Unlock()
			return Attempt{}, errors.New("payment store unavailable")
		}
		// Recording the payment opens its own database transaction while this
		// engine-wide mutex is held, so collections in this process serialise
		// behind one round trip. That is deliberate for now: the lock is what
		// keeps the reservation, the attempt state and the posted payment
		// consistent, and releasing it here would let a second webhook observe
		// a half-applied attempt. Narrowing it means moving the reservation
		// invariants into the database and is scheduled as its own change
		// rather than bolted on here.
		_, _, err := e.payments.Record(payments.RecordInput{ObligationID: attempt.ObligationID, SourceType: payments.SourceCollected, AmountKobo: event.SucceededAmountKobo, Provider: e.provider.Name(), ProviderReference: event.ProviderCollectionID, PaidAt: time.Time{}, RecordedBy: payments.CollectionRecorder, IdempotencyKey: payments.CollectionKeyPrefix + attempt.ID})
		if err != nil {
			e.mu.Unlock()
			return Attempt{}, err
		}
		attempt.SucceededAmountKobo = event.SucceededAmountKobo
		attempt.ProviderCollectionID = event.ProviderCollectionID
		attempt.SettlementState = event.SettlementState
		attempt.SettlementReference = event.SettlementReference
		attempt.FinalAt = e.now()
		if event.SucceededAmountKobo < attempt.RequestedAmountKobo {
			attempt.State = AttemptPartial
			attempt.RetryClassification = "partial"
			attempt.NextRetryAt = e.now().Add(24 * time.Hour)
		} else {
			attempt.State = AttemptSucceeded
		}
		if reservation := e.reservations[attempt.ReservationID]; reservation != nil {
			reservation.State = ReservationReleased
		}
		e.events[event.EventID] = true
		out := cloneAttempt(*attempt)
		e.mu.Unlock()
		return out, nil
	}
	attempt.State = AttemptFailed
	attempt.FailureCode = event.FailureCode
	attempt.RetryClassification = retryClass(event.Retryable, event.FailureCode)
	attempt.FinalAt = e.now()
	if event.Retryable {
		attempt.NextRetryAt = e.now().Add(24 * time.Hour)
	}
	if reservation := e.reservations[attempt.ReservationID]; reservation != nil {
		reservation.State = ReservationReleased
	}
	e.events[event.EventID] = true
	out := cloneAttempt(*attempt)
	e.mu.Unlock()
	return out, nil
}

func (e *Engine) Reconcile(ctx context.Context, attemptID string) (Attempt, error) {
	e.mu.Lock()
	attempt := e.attempts[attemptID]
	if attempt == nil {
		e.mu.Unlock()
		return Attempt{}, errors.New("collection attempt not found")
	}
	saved := cloneAttempt(*attempt)
	e.mu.Unlock()
	var response Response
	var err error
	if lookup, ok := e.provider.(ReferenceLookupProvider); ok && saved.MandateReference != "" {
		response, err = lookup.GetByReference(ctx, Request{CollectionReference: saved.ProviderCollectionID, MandateReference: saved.MandateReference, ExternalReference: saved.ExternalReference, ObligationID: saved.ObligationID, AmountKobo: saved.RequestedAmountKobo, Currency: "NGN"})
	} else if saved.ProviderCollectionID != "" {
		response, err = e.provider.Get(ctx, saved.ProviderCollectionID)
	} else {
		return Attempt{}, errors.New("provider collection identity not available; reconciliation required")
	}
	if err != nil {
		return Attempt{}, err
	}
	if saved.ProviderCollectionID != "" && response.ProviderCollectionID != saved.ProviderCollectionID {
		return Attempt{}, errors.New("provider lookup returned a different collection identity")
	}
	if response.ProviderCollectionID == "" {
		return Attempt{}, errors.New("provider lookup did not return a collection identity")
	}
	if saved.SucceededAmountKobo > 0 && response.State != ProviderSettled && response.State != ProviderSettlementPending && response.State != ProviderReversed && ((response.State != ProviderSucceeded && response.State != ProviderPartial) || response.SucceededAmountKobo != saved.SucceededAmountKobo) {
		return Attempt{}, errors.New("provider lookup does not confirm recorded payment; controlled reconciliation required")
	}
	encoded, _ := json.Marshal(response)
	digest := sha256.Sum256(encoded)
	event := Webhook{EventID: fmt.Sprintf("reconcile-%s-%x", attemptID, digest), ExternalReference: saved.ExternalReference, State: response.State, ProviderCollectionID: response.ProviderCollectionID, SucceededAmountKobo: response.SucceededAmountKobo, FailureCode: response.FailureCode, Retryable: response.Retryable, SettlementState: response.SettlementState, SettlementReference: response.SettlementReference}
	if signer, ok := e.provider.(WebhookSigner); ok {
		event.Signature = signer.Sign(event)
	}
	return e.ProcessWebhook(ctx, event)
}
func (e *Engine) Retry(ctx context.Context, attemptID string, now time.Time) (Attempt, error) {
	e.mu.Lock()
	attempt := e.attempts[attemptID]
	if attempt == nil {
		e.mu.Unlock()
		return Attempt{}, errors.New("collection attempt not found")
	}
	if attempt.State != AttemptFailed && attempt.State != AttemptPartial {
		e.mu.Unlock()
		return Attempt{}, errors.New("attempt is not retryable")
	}
	if e.maxRetries > 0 && attempt.AttemptNumber >= e.maxRetries {
		e.mu.Unlock()
		return Attempt{}, errors.New("collection retry limit reached")
	}
	if attempt.RetryClassification == "final" {
		e.mu.Unlock()
		return Attempt{}, errors.New("attempt has final failure")
	}
	if !attempt.NextRetryAt.IsZero() && now.Before(attempt.NextRetryAt) {
		e.mu.Unlock()
		return Attempt{}, errors.New("retry is not yet eligible")
	}
	obligationID := attempt.ObligationID
	e.mu.Unlock()
	return e.Start(ctx, obligationID, "retry:"+attemptID, now)
}
func (e *Engine) Cancel(ctx context.Context, attemptID string) (Attempt, error) {
	e.mu.Lock()
	attempt := e.attempts[attemptID]
	if attempt == nil {
		e.mu.Unlock()
		return Attempt{}, errors.New("collection attempt not found")
	}
	if attempt.State != AttemptPending && attempt.State != AttemptSubmitted && attempt.State != AttemptUnknown {
		e.mu.Unlock()
		return Attempt{}, errors.New("attempt cannot be cancelled in its current state")
	}
	providerID := attempt.ProviderCollectionID
	provider, ok := e.provider.(CancellationProvider)
	capabilities := Capabilities{}
	if cp, supported := e.provider.(CapabilityProvider); supported {
		capabilities = cp.Capabilities()
	}
	if !ok || !capabilities.Reversal {
		e.mu.Unlock()
		return Attempt{}, errors.New("collection provider does not permit cancellation")
	}
	e.mu.Unlock()
	if providerID == "" {
		return Attempt{}, errors.New("provider collection id not available")
	}
	response, err := provider.Cancel(ctx, providerID)
	if err != nil {
		return Attempt{}, err
	}
	if response.State != ProviderReversed {
		return Attempt{}, errors.New("provider did not confirm cancellation")
	}
	e.mu.Lock()
	if attempt.State != AttemptPending && attempt.State != AttemptSubmitted && attempt.State != AttemptUnknown {
		e.mu.Unlock()
		return Attempt{}, errors.New("collection result changed during cancellation; reconciliation required")
	}
	attempt.State, attempt.FinalAt = AttemptCancelled, e.now()
	if reservation := e.reservations[attempt.ReservationID]; reservation != nil {
		reservation.State = ReservationReleased
	}
	out := cloneAttempt(*attempt)
	e.mu.Unlock()
	return out, nil
}
func (e *Engine) GetAttempt(id string) (Attempt, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	attempt, ok := e.attempts[id]
	if !ok {
		return Attempt{}, false
	}
	return cloneAttempt(*attempt), true
}
func (e *Engine) ListAttempts(obligationID string) []Attempt {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := []Attempt{}
	for _, attempt := range e.attempts {
		if attempt.ObligationID == obligationID {
			out = append(out, cloneAttempt(*attempt))
		}
	}
	return out
}
func retryClass(retryable bool, _ string) string {
	if retryable {
		return "retryable"
	}
	return "final"
}
func cloneAttempt(v Attempt) Attempt { return v }

// OptimizeCollectionWindow returns the earliest safe banking window for an automated
// debit in Nigeria (Africa/Lagos timezone, UTC+1). It avoids weekend interbank
// downtime by rolling Saturday and Sunday dates to Monday morning at 05:00 WAT,
// which triggers collections before merchants transfer daily POS/store float.
func OptimizeCollectionWindow(t time.Time) time.Time {
	const targetHour = 5
	lagos := time.FixedZone("WAT", 3600)

	date := rollToBankingDay(t.In(lagos))
	result := time.Date(date.Year(), date.Month(), date.Day(), targetHour, 0, 0, 0, lagos)
	for result.Before(t) {
		result = rollToBankingDay(result.AddDate(0, 0, 1))
		result = time.Date(result.Year(), result.Month(), result.Day(), targetHour, 0, 0, 0, lagos)
	}
	return result.UTC()
}

// rollToBankingDay moves a weekend date forward to the following Monday.
// Nigerian interbank direct debit does not settle at the weekend.
func rollToBankingDay(value time.Time) time.Time {
	switch value.Weekday() {
	case time.Saturday:
		return value.AddDate(0, 0, 2)
	case time.Sunday:
		return value.AddDate(0, 0, 1)
	default:
		return value
	}
}
