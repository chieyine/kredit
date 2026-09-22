package payments

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"kredit/internal/identifier"
	"kredit/internal/ledger"
)

const (
	SourceVoluntary        = "integrated_voluntary"
	SourceSupplierTransfer = "supplier_recorded_transfer"
	SourceBuyerClaim       = "buyer_payment_claim"
	SourceCashRecorded     = "cash_recorded"
	SourceCollected        = "kredit_collection"
	CollectionRecorder     = "collection-worker"
	CollectionKeyPrefix    = "collection-attempt:"
	SourceAdjustment       = "adjustment"
	StateRecognized        = "recognized"
	StateReversed          = "reversed"
)

type ObligationSnapshot struct {
	FeeTerms               *ledger.FeeTerms `json:"fee_terms,omitempty"`
	ID                     string
	BuyerUserID            string
	SupplierOrganizationID string
	PrincipalKobo          ledger.Money
	OutstandingKobo        ledger.Money
	CollectionAt           time.Time
	Currency               string
}

type Payment struct {
	ID                     string       `json:"id"`
	ObligationID           string       `json:"obligation_id"`
	BuyerUserID            string       `json:"buyer_user_id"`
	SupplierOrganizationID string       `json:"supplier_organization_id"`
	SourceType             string       `json:"source_type"`
	AmountKobo             ledger.Money `json:"amount_kobo"`
	Currency               string       `json:"currency"`
	Provider               string       `json:"provider,omitempty"`
	ProviderReference      string       `json:"provider_reference,omitempty"`
	State                  string       `json:"state"`
	PaidAt                 time.Time    `json:"paid_at"`
	RecognizedAt           time.Time    `json:"recognized_at"`
	RecordedBy             string       `json:"recorded_by"`
	ReversalOf             string       `json:"reversal_of,omitempty"`
	CollectionFeeKobo      ledger.Money `json:"collection_fee_kobo"`
}

type Allocation struct {
	PaymentID      string       `json:"payment_id"`
	ObligationID   string       `json:"obligation_id"`
	ScheduleItemID string       `json:"schedule_item_id,omitempty"`
	AmountKobo     ledger.Money `json:"amount_kobo"`
	CreatedAt      time.Time    `json:"created_at"`
}
type RecordInput struct {
	ObligationID      string
	SourceType        string
	AmountKobo        ledger.Money
	Currency          string
	Provider          string
	ProviderReference string
	PaidAt            time.Time
	RecordedBy        string
	IdempotencyKey    string
}
type SnapshotFunc func(string) (ObligationSnapshot, error)
type ApplyFunc func(string, ledger.Money) error
type AllocationFunc func(string, ledger.Money) ([]AllocationTarget, error)
type ReverseAllocationFunc func([]AllocationTarget) error
type AllocationTarget struct {
	ScheduleItemID string
	AmountKobo     ledger.Money
}

// Service is the payment boundary. Durable deployments must use PostgresStore,
// which commits payment, allocation, balance, fee and journal changes together.
// The process-local Store is a development adapter: its arbitrary callbacks
// cannot provide database atomicity. It fails closed after an uncertain write
// and must never be used as a restart-safe or production system of record.
type Service interface {
	RecordContext(context.Context, RecordInput) (Payment, Allocation, error)
	Record(RecordInput) (Payment, Allocation, error)
	Reverse(paymentID, actor, reason string) (Payment, error)
	// List must report failure rather than return an empty history. Showing a
	// supplier "no payments" during a database incident is the class of untrue
	// statement README section 2.3 forbids.
	List(obligationID string) ([]Payment, error)
	Get(paymentID string) (Payment, error)
	Rebuild(obligationID string) (ledger.Money, error)
}

var _ Service = (*Store)(nil)

type Store struct {
	mu              sync.RWMutex
	ledger          ledger.Service
	lookup          SnapshotFunc
	apply           ApplyFunc
	payments        map[string]*Payment
	allocations     map[string][]*Allocation
	byKey           map[string]string
	recording       map[string]*memoryOperation
	blocked         map[string]*memoryOperation
	now             func() time.Time
	newID           func() string
	allocate        AllocationFunc
	reallocate      ReverseAllocationFunc
	markCollected   func(string, ledger.Money) error
	unmarkCollected func(string, ledger.Money) error
}

func NewStore(ledgerStore ledger.Service, lookup SnapshotFunc, apply ApplyFunc) *Store {
	return NewStoreWithAllocator(ledgerStore, lookup, apply, nil, nil)
}

func NewStoreWithAllocator(ledgerStore ledger.Service, lookup SnapshotFunc, apply ApplyFunc, allocate AllocationFunc, reallocate ReverseAllocationFunc) *Store {
	return &Store{ledger: ledgerStore, lookup: lookup, apply: apply, allocate: allocate, reallocate: reallocate, payments: map[string]*Payment{}, allocations: map[string][]*Allocation{}, byKey: map[string]string{}, recording: map[string]*memoryOperation{}, blocked: map[string]*memoryOperation{}, now: func() time.Time { return time.Now().UTC() }, newID: newIdentifier}
}

func (s *Store) SetCollectedMarker(marker func(string, ledger.Money) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.markCollected = marker
}
func (s *Store) SetCollectedReversalMarker(marker func(string, ledger.Money) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unmarkCollected = marker
}

func (s *Store) Record(input RecordInput) (recorded Payment, allocated Allocation, retErr error) {
	if input.ObligationID == "" || input.AmountKobo <= 0 || input.RecordedBy == "" || input.IdempotencyKey == "" {
		return Payment{}, Allocation{}, errors.New("obligation, amount, recorder, and idempotency key are required")
	}
	if !validSource(input.SourceType) {
		return Payment{}, Allocation{}, errors.New("invalid payment source")
	}
	if input.SourceType == SourceCollected && !validCollectedPayment(input) {
		return Payment{}, Allocation{}, errors.New("collected payments require collection-worker provenance, provider identity, and an attempt idempotency key")
	}
	input.PaidAt = input.PaidAt.UTC().Truncate(time.Microsecond)
	s.mu.Lock()
	defer s.mu.Unlock()
	// Reserve the original identity before any collaborator can change state.
	// Failed attempts are not converted into a new payment on a later retry.
	if pending := s.recording[input.IdempotencyKey]; pending != nil {
		if !samePaymentIntent(pending.payment, input) {
			return Payment{}, Allocation{}, errors.New("idempotency key was reused for a different payment")
		}
		return Payment{}, Allocation{}, pending.recoveryError()
	}
	if err := s.memoryBlock(input.ObligationID); err != nil {
		return Payment{}, Allocation{}, err
	}
	if existing := s.byKey[input.IdempotencyKey]; existing != "" {
		stored := s.payments[existing]
		allocations := s.allocations[existing]
		if stored == nil || len(allocations) == 0 {
			return Payment{}, Allocation{}, errors.New("recorded payment could not be replayed; reconcile before retrying")
		}
		if !samePaymentIntent(*stored, input) {
			return Payment{}, Allocation{}, errors.New("idempotency key was reused for a different payment")
		}
		return clonePayment(*stored), allocations[0].clone(), nil
	}
	if s.ledger == nil || s.lookup == nil || s.apply == nil {
		return Payment{}, Allocation{}, errors.New("payment dependencies unavailable")
	}
	snapshot, err := s.lookup(input.ObligationID)
	if err != nil {
		return Payment{}, Allocation{}, err
	}
	if defaultCurrency(input.Currency, snapshot.Currency) != snapshot.Currency {
		return Payment{}, Allocation{}, errors.New("payment currency must match the obligation")
	}
	if input.AmountKobo > snapshot.OutstandingKobo {
		return Payment{}, Allocation{}, errors.New("payment exceeds authoritative outstanding amount")
	}
	now := s.now().UTC().Truncate(time.Microsecond)
	paidAt := input.PaidAt
	if paidAt.IsZero() {
		paidAt = now
	}
	if paidAt.After(now.Add(5 * time.Minute)) {
		return Payment{}, Allocation{}, errors.New("payment date cannot be in the future")
	}
	payment := &Payment{ID: s.newID(), ObligationID: input.ObligationID, BuyerUserID: snapshot.BuyerUserID, SupplierOrganizationID: snapshot.SupplierOrganizationID, SourceType: input.SourceType, AmountKobo: input.AmountKobo, Currency: defaultCurrency(input.Currency, snapshot.Currency), Provider: strings.TrimSpace(input.Provider), ProviderReference: strings.TrimSpace(input.ProviderReference), State: StateRecognized, PaidAt: paidAt, RecognizedAt: now, RecordedBy: input.RecordedBy}
	if payment.SourceType == SourceCollected && !paidAt.Before(snapshot.CollectionAt) {
		fee, err := snapshot.FeeTerms.Collection(payment.AmountKobo)
		if err != nil {
			return Payment{}, Allocation{}, err
		}
		payment.CollectionFeeKobo = fee
	}
	operation := &memoryOperation{payment: *payment, kind: "record", key: input.IdempotencyKey, stage: "payment journal"}
	s.recording[input.IdempotencyKey] = operation
	s.blocked[payment.ObligationID] = operation
	defer s.finishMemoryOperation(operation, &retErr)
	if _, err := s.ledger.PostPayment(payment.ID, payment.AmountKobo, payment.SourceType, paidAt, "payment:"+input.IdempotencyKey); err != nil {
		return Payment{}, Allocation{}, err
	}
	operation.stage = "balance application"
	if err := s.apply(input.ObligationID, -payment.AmountKobo); err != nil {
		return Payment{}, Allocation{}, err
	}
	var targets []AllocationTarget
	if s.allocate != nil {
		operation.stage = "schedule allocation"
		targets, err = s.allocate(input.ObligationID, payment.AmountKobo)
		if err != nil {
			return Payment{}, Allocation{}, err
		}
		if err := validateMemoryAllocations(targets, payment.AmountKobo); err != nil {
			return Payment{}, Allocation{}, err
		}
	}
	if payment.CollectionFeeKobo > 0 {
		operation.stage = "collection fee journal"
		if _, err := s.ledger.PostCollectionFee(payment.ID, payment.CollectionFeeKobo, paidAt, "collection-fee:"+input.IdempotencyKey); err != nil {
			return Payment{}, Allocation{}, err
		}
	}
	if payment.SourceType == SourceCollected && !paidAt.Before(snapshot.CollectionAt) && s.markCollected != nil {
		operation.stage = "collected allocation marker"
		if err := s.markCollected(input.ObligationID, payment.AmountKobo); err != nil {
			return Payment{}, Allocation{}, err
		}
	}
	operation.stage = "payment publication"
	allocations := make([]*Allocation, 0, len(targets))
	if len(targets) == 0 {
		allocations = append(allocations, &Allocation{PaymentID: payment.ID, ObligationID: input.ObligationID, AmountKobo: payment.AmountKobo, CreatedAt: now})
	} else {
		for _, target := range targets {
			allocations = append(allocations, &Allocation{PaymentID: payment.ID, ObligationID: input.ObligationID, ScheduleItemID: target.ScheduleItemID, AmountKobo: target.AmountKobo, CreatedAt: now})
		}
	}
	s.payments[payment.ID] = payment
	s.allocations[payment.ID] = allocations
	s.byKey[input.IdempotencyKey] = payment.ID
	operation.completed = true
	return clonePayment(*payment), allocations[0].clone(), nil
}

func (s *Store) RecordContext(ctx context.Context, input RecordInput) (Payment, Allocation, error) {
	if err := ctx.Err(); err != nil {
		return Payment{}, Allocation{}, err
	}
	return s.Record(input)
}

func (s *Store) ReverseContext(ctx context.Context, paymentID, actor, reason string) (Payment, error) {
	if err := ctx.Err(); err != nil {
		return Payment{}, err
	}
	return s.Reverse(paymentID, actor, reason)
}

func (s *Store) GetContext(ctx context.Context, paymentID string) (Payment, error) {
	if err := ctx.Err(); err != nil {
		return Payment{}, err
	}
	return s.Get(paymentID)
}

func (s *Store) RebuildContext(ctx context.Context, obligationID string) (ledger.Money, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return s.Rebuild(obligationID)
}

func (s *Store) ReadContext(ctx context.Context, obligationID string) ([]Payment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.List(obligationID)
}

func validCollectedPayment(input RecordInput) bool {
	return input.RecordedBy == CollectionRecorder &&
		strings.TrimSpace(input.Provider) != "" &&
		strings.TrimSpace(input.ProviderReference) != "" &&
		strings.HasPrefix(input.IdempotencyKey, CollectionKeyPrefix) &&
		strings.TrimSpace(strings.TrimPrefix(input.IdempotencyKey, CollectionKeyPrefix)) != ""
}

func samePaymentIntent(existing Payment, requested RecordInput) bool {
	if existing.ObligationID != requested.ObligationID || existing.SourceType != requested.SourceType || existing.AmountKobo != requested.AmountKobo || existing.RecordedBy != requested.RecordedBy {
		return false
	}
	if defaultCurrency(requested.Currency, existing.Currency) != existing.Currency || strings.TrimSpace(requested.Provider) != existing.Provider || strings.TrimSpace(requested.ProviderReference) != existing.ProviderReference {
		return false
	}
	return requested.PaidAt.IsZero() || requested.PaidAt.Equal(existing.PaidAt)
}

func validSource(source string) bool {
	switch source {
	case SourceVoluntary, SourceSupplierTransfer, SourceBuyerClaim, SourceCashRecorded, SourceCollected, SourceAdjustment:
		return true
	default:
		return false
	}
}

func (s *Store) Reverse(paymentID, actor, reason string) (reversed Payment, retErr error) {
	if actor == "" || strings.TrimSpace(reason) == "" {
		return Payment{}, errors.New("reversal actor and reason are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.payments[paymentID]
	if p == nil {
		return Payment{}, errors.New("payment not found")
	}
	if err := s.memoryBlock(p.ObligationID); err != nil {
		return Payment{}, err
	}
	if p.State == StateReversed {
		return clonePayment(*p), nil
	}
	if s.ledger == nil || s.apply == nil {
		return Payment{}, errors.New("payment dependencies unavailable")
	}
	targets := make([]AllocationTarget, 0, len(s.allocations[p.ID]))
	for _, allocation := range s.allocations[p.ID] {
		if allocation.ScheduleItemID != "" {
			targets = append(targets, AllocationTarget{ScheduleItemID: allocation.ScheduleItemID, AmountKobo: allocation.AmountKobo})
		}
	}
	if len(targets) > 0 && s.reallocate == nil {
		return Payment{}, errors.New("schedule reversal dependency unavailable")
	}
	if p.SourceType == SourceCollected && s.markCollected != nil && s.unmarkCollected == nil {
		return Payment{}, errors.New("collected allocation reversal dependency unavailable")
	}
	now := s.now().UTC().Truncate(time.Microsecond)
	reversalID := s.newID()
	operation := &memoryOperation{payment: *p, kind: "reverse", stage: "reversal journal"}
	s.blocked[p.ObligationID] = operation
	defer s.finishMemoryOperation(operation, &retErr)
	if _, err := s.ledger.PostPaymentReversal(p.ID, p.AmountKobo, p.SourceType, now, "reversal:"+p.ID); err != nil {
		return Payment{}, err
	}
	if p.CollectionFeeKobo > 0 {
		operation.stage = "collection fee reversal journal"
		if _, err := s.ledger.PostCollectionFeeReversal(p.ID, p.CollectionFeeKobo, now, "collection-fee-reversal:"+p.ID); err != nil {
			return Payment{}, err
		}
	}
	// Preserve collected <= allocated while removing the original allocation.
	if p.SourceType == SourceCollected && s.unmarkCollected != nil {
		operation.stage = "collected allocation reversal"
		if err := s.unmarkCollected(p.ObligationID, p.AmountKobo); err != nil {
			return Payment{}, err
		}
	}
	if s.reallocate != nil {
		operation.stage = "schedule allocation reversal"
		if err := s.reallocate(targets); err != nil {
			return Payment{}, err
		}
	}
	operation.stage = "balance reversal"
	if err := s.apply(p.ObligationID, p.AmountKobo); err != nil {
		return Payment{}, err
	}
	operation.stage = "reversal publication"
	reversal := &Payment{ID: reversalID, ObligationID: p.ObligationID, BuyerUserID: p.BuyerUserID, SupplierOrganizationID: p.SupplierOrganizationID, SourceType: p.SourceType, AmountKobo: p.AmountKobo, Currency: p.Currency, Provider: p.Provider, ProviderReference: p.ProviderReference, State: StateReversed, PaidAt: p.PaidAt, RecognizedAt: now, RecordedBy: actor, ReversalOf: p.ID, CollectionFeeKobo: p.CollectionFeeKobo}
	s.payments[reversal.ID] = reversal
	p.State = StateReversed
	operation.completed = true
	return clonePayment(*p), nil
}

// List returns the obligation's payments oldest first. The PostgreSQL store
// orders by (recognized_at, id); the in-memory store must match so statements
// and receipts read the same in either configuration.
func (s *Store) List(obligationID string) ([]Payment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.memoryBlock(obligationID); err != nil {
		return nil, err
	}
	out := []Payment{}
	for _, p := range s.payments {
		if p.ObligationID == obligationID {
			out = append(out, clonePayment(*p))
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].RecognizedAt.Equal(out[j].RecognizedAt) {
			return out[i].RecognizedAt.Before(out[j].RecognizedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (s *Store) Get(paymentID string) (Payment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, operation := range s.recording {
		if operation.payment.ID == paymentID {
			return Payment{}, operation.recoveryError()
		}
	}
	p := s.payments[paymentID]
	if p == nil {
		return Payment{}, errors.New("payment not found")
	}
	if err := s.memoryBlock(p.ObligationID); err != nil {
		return Payment{}, err
	}
	return clonePayment(*p), nil
}
func (s *Store) Rebuild(obligationID string) (ledger.Money, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.memoryBlock(obligationID); err != nil {
		return 0, err
	}
	if s.lookup == nil || s.apply == nil {
		return 0, errors.New("payment dependencies unavailable")
	}
	snapshot, err := s.lookup(obligationID)
	if err != nil {
		return 0, err
	}
	recognized := ledger.Money(0)
	for _, p := range s.payments {
		if p.ObligationID == obligationID && p.State == StateRecognized {
			recognized, err = ledger.CheckedAdd(recognized, p.AmountKobo)
			if err != nil {
				return 0, errors.New("recognized payment total is too large")
			}
		}
	}
	expected := snapshot.PrincipalKobo - recognized
	if expected < 0 {
		return 0, errors.New("payments exceed principal")
	}
	if expected != snapshot.OutstandingKobo {
		// This demo store has no authoritative dispute/write-off projection.
		// Never resurrect forgiven principal from a payments-only reconstruction.
		return 0, errors.New("balance differs from payment history; complete adjustment history is required for reconciliation")
	}
	return expected, nil
}

func (a Allocation) clone() Allocation { return a }
func clonePayment(p Payment) Payment   { return p }
func defaultCurrency(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.ToUpper(strings.TrimSpace(value))
	}
	return fallback
}
func newIdentifier() string { return identifier.New() }
