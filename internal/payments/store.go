package payments

import (
	"errors"
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
	SourceAdjustment       = "adjustment"
	StateRecognized        = "recognized"
	StateReversed          = "reversed"
)

type ObligationSnapshot struct {
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

// Service is the payment system of record. Implementations must make payment,
// allocation, balance, fee, and journal changes atomically and preserve
// idempotency across process restarts.
type Service interface {
	Record(RecordInput) (Payment, Allocation, error)
	Reverse(paymentID, actor, reason string) (Payment, error)
	List(obligationID string) []Payment
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
	return &Store{ledger: ledgerStore, lookup: lookup, apply: apply, allocate: allocate, reallocate: reallocate, payments: map[string]*Payment{}, allocations: map[string][]*Allocation{}, byKey: map[string]string{}, now: func() time.Time { return time.Now().UTC() }, newID: newIdentifier}
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

func (s *Store) Record(input RecordInput) (Payment, Allocation, error) {
	if input.ObligationID == "" || input.AmountKobo <= 0 || input.RecordedBy == "" || input.IdempotencyKey == "" {
		return Payment{}, Allocation{}, errors.New("obligation, amount, recorder, and idempotency key are required")
	}
	if !validSource(input.SourceType) {
		return Payment{}, Allocation{}, errors.New("invalid payment source")
	}
	if input.SourceType == SourceCollected && strings.TrimSpace(input.Provider) == "" {
		return Payment{}, Allocation{}, errors.New("collected payments require a provider")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing := s.byKey[input.IdempotencyKey]; existing != "" {
		payment := *s.payments[existing]
		if !samePaymentIntent(payment, input) {
			return Payment{}, Allocation{}, errors.New("idempotency key was reused for a different payment")
		}
		return clonePayment(payment), s.allocations[existing][0].clone(), nil
	}
	if s.ledger == nil || s.lookup == nil || s.apply == nil {
		return Payment{}, Allocation{}, errors.New("payment dependencies unavailable")
	}
	snapshot, err := s.lookup(input.ObligationID)
	if err != nil {
		return Payment{}, Allocation{}, err
	}
	if input.AmountKobo > snapshot.OutstandingKobo {
		return Payment{}, Allocation{}, errors.New("payment exceeds authoritative outstanding amount")
	}
	now := s.now()
	paidAt := input.PaidAt
	if paidAt.IsZero() {
		paidAt = now
	}
	payment := &Payment{ID: s.newID(), ObligationID: input.ObligationID, BuyerUserID: snapshot.BuyerUserID, SupplierOrganizationID: snapshot.SupplierOrganizationID, SourceType: input.SourceType, AmountKobo: input.AmountKobo, Currency: defaultCurrency(input.Currency, snapshot.Currency), Provider: strings.TrimSpace(input.Provider), ProviderReference: strings.TrimSpace(input.ProviderReference), State: StateRecognized, PaidAt: paidAt, RecognizedAt: now, RecordedBy: input.RecordedBy}
	tx, err := s.ledger.PostPayment(payment.ID, payment.AmountKobo, payment.SourceType, paidAt, "payment:"+input.IdempotencyKey)
	if err != nil {
		return Payment{}, Allocation{}, err
	}
	_ = tx
	if err = s.apply(input.ObligationID, -payment.AmountKobo); err != nil {
		_, _ = s.ledger.PostPaymentReversal(payment.ID, payment.AmountKobo, payment.SourceType, now, "rollback:"+input.IdempotencyKey)
		return Payment{}, Allocation{}, err
	}
	var targets []AllocationTarget
	if s.allocate != nil {
		targets, err = s.allocate(input.ObligationID, payment.AmountKobo)
		if err != nil {
			_ = s.apply(input.ObligationID, payment.AmountKobo)
			_, _ = s.ledger.PostPaymentReversal(payment.ID, payment.AmountKobo, payment.SourceType, now, "rollback-allocation:"+input.IdempotencyKey)
			return Payment{}, Allocation{}, err
		}
	}
	if payment.SourceType == SourceCollected && !paidAt.Before(snapshot.CollectionAt) {
		fee, _ := ledger.BaseFee(payment.AmountKobo)
		payment.CollectionFeeKobo = fee
		if fee > 0 {
			if _, err = s.ledger.PostCollectionFee(payment.ID, fee, paidAt, "collection-fee:"+input.IdempotencyKey); err != nil {
				if s.reallocate != nil {
					_ = s.reallocate(targets)
				}
				_ = s.apply(input.ObligationID, payment.AmountKobo)
				_, _ = s.ledger.PostPaymentReversal(payment.ID, payment.AmountKobo, payment.SourceType, now, "rollback-fee:"+input.IdempotencyKey)
				return Payment{}, Allocation{}, err
			}
		}
		if s.markCollected != nil {
			if err = s.markCollected(input.ObligationID, payment.AmountKobo); err != nil {
				if s.reallocate != nil {
					_ = s.reallocate(targets)
				}
				_ = s.apply(input.ObligationID, payment.AmountKobo)
				_, _ = s.ledger.PostPaymentReversal(payment.ID, payment.AmountKobo, payment.SourceType, now, "rollback-collected:"+input.IdempotencyKey)
				if fee > 0 {
					_, _ = s.ledger.PostCollectionFeeReversal(payment.ID, fee, paidAt, "rollback-collected-fee:"+input.IdempotencyKey)
				}
				return Payment{}, Allocation{}, err
			}
		}
	}
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
	return clonePayment(*payment), allocations[0].clone(), nil
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

func (s *Store) Reverse(paymentID, actor, reason string) (Payment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.payments[paymentID]
	if p == nil {
		return Payment{}, errors.New("payment not found")
	}
	if p.State == StateReversed {
		return clonePayment(*p), nil
	}
	if actor == "" || strings.TrimSpace(reason) == "" {
		return Payment{}, errors.New("reversal actor and reason are required")
	}
	if s.ledger == nil || s.apply == nil {
		return Payment{}, errors.New("payment dependencies unavailable")
	}
	if _, err := s.ledger.PostPaymentReversal(p.ID, p.AmountKobo, p.SourceType, s.now(), "reversal:"+p.ID); err != nil {
		return Payment{}, err
	}
	if p.CollectionFeeKobo > 0 {
		if _, err := s.ledger.PostCollectionFeeReversal(p.ID, p.CollectionFeeKobo, s.now(), "collection-fee-reversal:"+p.ID); err != nil {
			s.compensateReversalLedger(p, false)
			return Payment{}, err
		}
	}
	// Release collected markers before reversing schedule allocations. The
	// PostgreSQL schedule repository enforces collected_kobo <= allocated_kobo.
	if p.SourceType == SourceCollected && s.unmarkCollected != nil {
		if err := s.unmarkCollected(p.ObligationID, p.AmountKobo); err != nil {
			s.compensateReversalLedger(p, p.CollectionFeeKobo > 0)
			return Payment{}, err
		}
	}
	if s.reallocate != nil {
		targets := make([]AllocationTarget, 0, len(s.allocations[p.ID]))
		for _, allocation := range s.allocations[p.ID] {
			if allocation.ScheduleItemID != "" {
				targets = append(targets, AllocationTarget{ScheduleItemID: allocation.ScheduleItemID, AmountKobo: allocation.AmountKobo})
			}
		}
		if err := s.reallocate(targets); err != nil {
			if p.SourceType == SourceCollected && s.markCollected != nil {
				_ = s.markCollected(p.ObligationID, p.AmountKobo)
			}
			s.compensateReversalLedger(p, p.CollectionFeeKobo > 0)
			return Payment{}, err
		}
	}
	if err := s.apply(p.ObligationID, p.AmountKobo); err != nil {
		if p.SourceType == SourceCollected && s.markCollected != nil {
			_ = s.markCollected(p.ObligationID, p.AmountKobo)
		}
		if s.allocate != nil {
			_, _ = s.allocate(p.ObligationID, p.AmountKobo)
		}
		s.compensateReversalLedger(p, p.CollectionFeeKobo > 0)
		return Payment{}, err
	}
	// Preserve the original payment as an immutable historical record and
	// append a separate reversal event that points back to it. A self-referential
	// reversal marker makes rebuilds and audit history ambiguous.
	p.State = StateReversed
	reversal := &Payment{ID: s.newID(), ObligationID: p.ObligationID, BuyerUserID: p.BuyerUserID, SupplierOrganizationID: p.SupplierOrganizationID, SourceType: p.SourceType, AmountKobo: p.AmountKobo, Currency: p.Currency, Provider: p.Provider, ProviderReference: p.ProviderReference, State: StateReversed, PaidAt: p.PaidAt, RecognizedAt: s.now(), RecordedBy: actor, ReversalOf: p.ID, CollectionFeeKobo: p.CollectionFeeKobo}
	s.payments[reversal.ID] = reversal
	return clonePayment(*p), nil
}

func (s *Store) compensateReversalLedger(payment *Payment, fee bool) {
	if payment == nil || s.ledger == nil {
		return
	}
	now := s.now()
	_, _ = s.ledger.PostPayment(payment.ID, payment.AmountKobo, payment.SourceType, now, "rollback-reversal:"+payment.ID)
	if fee && payment.CollectionFeeKobo > 0 {
		_, _ = s.ledger.PostCollectionFee(payment.ID, payment.CollectionFeeKobo, now, "rollback-collection-fee-reversal:"+payment.ID)
	}
}

func (s *Store) List(obligationID string) []Payment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Payment{}
	for _, p := range s.payments {
		if p.ObligationID == obligationID {
			out = append(out, clonePayment(*p))
		}
	}
	return out
}

func (s *Store) Get(paymentID string) (Payment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p := s.payments[paymentID]
	if p == nil {
		return Payment{}, errors.New("payment not found")
	}
	return clonePayment(*p), nil
}
func (s *Store) Rebuild(obligationID string) (ledger.Money, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
		if err := s.apply(obligationID, expected-snapshot.OutstandingKobo); err != nil {
			return 0, err
		}
	}
	return expected, nil
}

func (a Allocation) clone() Allocation { return a }
func clonePayment(p Payment) Payment   { return p }
func defaultCurrency(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
func newIdentifier() string { return identifier.New() }
