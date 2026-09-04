package paymentclaims

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"kredit/internal/identifier"
	"kredit/internal/ledger"
	"kredit/internal/payments"
)

const (
	Pending   = "pending"
	Confirmed = "confirmed"
	Rejected  = "rejected"
	Expired   = "expired"
)

type ObligationSnapshot struct {
	ID                     string
	BuyerUserID            string
	SupplierOrganizationID string
	OutstandingKobo        ledger.Money
	Currency               string
}

type Claim struct {
	ID                     string       `json:"id"`
	ObligationID           string       `json:"obligation_id"`
	BuyerUserID            string       `json:"buyer_user_id"`
	SupplierOrganizationID string       `json:"supplier_organization_id"`
	AmountKobo             ledger.Money `json:"amount_kobo"`
	Currency               string       `json:"currency"`
	PaidAt                 time.Time    `json:"paid_at"`
	SourceAccountMasked    string       `json:"source_account_masked,omitempty"`
	TransferReference      string       `json:"transfer_reference"`
	EvidenceDocumentID     string       `json:"evidence_document_id,omitempty"`
	State                  string       `json:"state"`
	HoldExpiresAt          time.Time    `json:"hold_expires_at"`
	ReviewedBy             string       `json:"reviewed_by,omitempty"`
	ReviewReason           string       `json:"review_reason,omitempty"`
	PaymentID              string       `json:"payment_id,omitempty"`
	CreatedAt              time.Time    `json:"created_at"`
	ReviewedAt             *time.Time   `json:"reviewed_at,omitempty"`
}

type CreateInput struct {
	ObligationID        string
	BuyerUserID         string
	AmountKobo          ledger.Money
	PaidAt              time.Time
	SourceAccountMasked string
	TransferReference   string
	EvidenceDocumentID  string
	IdempotencyKey      string
}

type Service interface {
	Confirm(context.Context, string, string, string, payments.Service) (Claim, error)
	Create(context.Context, CreateInput) (Claim, error)
	Get(context.Context, string) (Claim, error)
	ListForObligation(context.Context, string) []Claim
	ListForBuyer(context.Context, string) []Claim
	ListForSupplier(context.Context, string) []Claim
	Decide(context.Context, string, string, string, string, string) (Claim, error)
	ActiveHold(context.Context, string, time.Time) ledger.Money
}

type SnapshotFunc func(string) (ObligationSnapshot, error)

type Store struct {
	mu      sync.RWMutex
	lookup  SnapshotFunc
	items   map[string]*Claim
	byKey   map[string]string
	now     func() time.Time
	holdFor time.Duration
}

func NewStore(lookup SnapshotFunc) *Store {
	return &Store{lookup: lookup, items: map[string]*Claim{}, byKey: map[string]string{}, now: func() time.Time { return time.Now().UTC() }, holdFor: 24 * time.Hour}
}

func (s *Store) Create(_ context.Context, input CreateInput) (Claim, error) {
	if s == nil || s.lookup == nil {
		return Claim{}, errors.New("payment claim service is unavailable")
	}
	if strings.TrimSpace(input.ObligationID) == "" || strings.TrimSpace(input.BuyerUserID) == "" || input.AmountKobo <= 0 || strings.TrimSpace(input.TransferReference) == "" || strings.TrimSpace(input.IdempotencyKey) == "" {
		return Claim{}, errors.New("obligation, buyer, positive amount, transfer reference, and idempotency key are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if id := s.byKey[input.IdempotencyKey]; id != "" {
		existing := *s.items[id]
		if !sameClaimIntent(existing, input) {
			return Claim{}, errors.New("idempotency key belongs to a different payment claim")
		}
		return existing, nil
	}
	snapshot, err := s.lookup(input.ObligationID)
	if err != nil {
		return Claim{}, err
	}
	if snapshot.BuyerUserID != input.BuyerUserID {
		return Claim{}, errors.New("payment claim does not belong to buyer")
	}
	if input.AmountKobo > snapshot.OutstandingKobo {
		return Claim{}, errors.New("payment claim exceeds authoritative outstanding amount")
	}
	now := s.now()
	paidAt := input.PaidAt
	if paidAt.IsZero() {
		paidAt = now
	}
	if paidAt.After(now.Add(5 * time.Minute)) {
		return Claim{}, errors.New("payment date cannot be in the future")
	}
	claim := &Claim{ID: identifier.New(), ObligationID: snapshot.ID, BuyerUserID: input.BuyerUserID, SupplierOrganizationID: snapshot.SupplierOrganizationID, AmountKobo: input.AmountKobo, Currency: snapshot.Currency, PaidAt: paidAt, SourceAccountMasked: strings.TrimSpace(input.SourceAccountMasked), TransferReference: strings.TrimSpace(input.TransferReference), EvidenceDocumentID: strings.TrimSpace(input.EvidenceDocumentID), State: Pending, HoldExpiresAt: now.Add(s.holdFor), CreatedAt: now}
	s.items[claim.ID] = claim
	s.byKey[input.IdempotencyKey] = claim.ID
	return *claim, nil
}

func (s *Store) Get(_ context.Context, id string) (Claim, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	claim := s.items[id]
	if claim == nil {
		return Claim{}, errors.New("payment claim not found")
	}
	return *claim, nil
}

func (s *Store) ListForObligation(_ context.Context, obligationID string) []Claim {
	return s.list(func(c *Claim) bool { return c.ObligationID == obligationID })
}

func (s *Store) ListForBuyer(_ context.Context, buyerUserID string) []Claim {
	return s.list(func(c *Claim) bool { return c.BuyerUserID == buyerUserID })
}
func (s *Store) ListForSupplier(_ context.Context, organizationID string) []Claim {
	return s.list(func(c *Claim) bool { return c.SupplierOrganizationID == organizationID })
}

func (s *Store) list(include func(*Claim) bool) []Claim {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []Claim{}
	for _, claim := range s.items {
		if include(claim) {
			result = append(result, *claim)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result
}

func (s *Store) Decide(_ context.Context, id, actor, decision, reason, paymentID string) (Claim, error) {
	if decision != Confirmed && decision != Rejected {
		return Claim{}, errors.New("payment claim decision must be confirmed or rejected")
	}
	if strings.TrimSpace(actor) == "" || strings.TrimSpace(reason) == "" {
		return Claim{}, errors.New("reviewer and reason are required")
	}
	if decision == Confirmed && strings.TrimSpace(paymentID) == "" {
		return Claim{}, errors.New("confirmed claim requires a recognized payment")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	claim := s.items[id]
	if claim == nil {
		return Claim{}, errors.New("payment claim not found")
	}
	if claim.State != Pending {
		if claim.State == decision {
			return *claim, nil
		}
		return Claim{}, errors.New("payment claim has already been decided")
	}
	reviewedAt := s.now()
	claim.State, claim.ReviewedBy, claim.ReviewReason, claim.PaymentID, claim.ReviewedAt = decision, actor, strings.TrimSpace(reason), strings.TrimSpace(paymentID), &reviewedAt
	return *claim, nil
}

func (s *Store) ActiveHold(_ context.Context, obligationID string, at time.Time) ledger.Money {
	s.mu.Lock()
	defer s.mu.Unlock()
	var total ledger.Money
	for _, claim := range s.items {
		if claim.ObligationID != obligationID || claim.State != Pending {
			continue
		}
		if !at.Before(claim.HoldExpiresAt) {
			claim.State = Expired
			continue
		}
		total += claim.AmountKobo
	}
	return total
}

// PostgreSQL persists timestamps to microsecond precision.
func sameClaimIntent(claim Claim, input CreateInput) bool {
	return claim.ObligationID == input.ObligationID && claim.BuyerUserID == input.BuyerUserID && claim.AmountKobo == input.AmountKobo && claim.TransferReference == strings.TrimSpace(input.TransferReference) && claim.SourceAccountMasked == strings.TrimSpace(input.SourceAccountMasked) && claim.EvidenceDocumentID == strings.TrimSpace(input.EvidenceDocumentID) && (input.PaidAt.IsZero() || claim.PaidAt.Truncate(time.Microsecond).Equal(input.PaidAt.Truncate(time.Microsecond)))
}

func claimPaymentInput(claim Claim, actor string) payments.RecordInput {
	return payments.RecordInput{ObligationID: claim.ObligationID, SourceType: payments.SourceBuyerClaim, AmountKobo: claim.AmountKobo, Currency: claim.Currency, ProviderReference: claim.TransferReference, PaidAt: claim.PaidAt, RecordedBy: actor, IdempotencyKey: "payment-claim:" + claim.ID}
}

func (s *Store) Confirm(ctx context.Context, id, actor, reason string, recorder payments.Service) (Claim, error) {
	if strings.TrimSpace(actor) == "" || strings.TrimSpace(reason) == "" || recorder == nil {
		return Claim{}, errors.New("reviewer, reason, and payment service are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	claim := s.items[id]
	if claim == nil {
		return Claim{}, errors.New("payment claim not found")
	}
	if claim.State == Confirmed {
		return *claim, nil
	}
	if claim.State != Pending {
		return Claim{}, errors.New("payment claim is not pending")
	}
	payment, _, err := recorder.Record(claimPaymentInput(*claim, actor))
	if err != nil {
		return Claim{}, err
	}
	now := s.now()
	claim.State = Confirmed
	claim.ReviewedBy = actor
	claim.ReviewReason = strings.TrimSpace(reason)
	claim.PaymentID = payment.ID
	claim.ReviewedAt = &now
	return *claim, nil
}
