package disputes

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"kredit/internal/identifier"
	"kredit/internal/ledger"
)

const (
	StateOpen              = "OPEN"
	StateUnderReview       = "UNDER_REVIEW"
	StatePartiallyResolved = "PARTIALLY_RESOLVED"
	StateResolved          = "RESOLVED"
	StateWithdrawn         = "WITHDRAWN"
	EffectFullBlock        = "FULL_BLOCK"
	EffectContestedOnly    = "CONTESTED_ONLY"
	EffectNoAutomaticBlock = "NO_AUTOMATIC_BLOCK"
)

type ObligationSnapshot struct {
	OutstandingKobo        ledger.Money
	SupplierOrganizationID string
	BuyerUserID            string
}
type SnapshotFunc func(string) (ObligationSnapshot, error)
type Dispute struct {
	ID                     string       `json:"id"`
	ObligationID           string       `json:"obligation_id"`
	SupplierOrganizationID string       `json:"supplier_organization_id"`
	BuyerUserID            string       `json:"buyer_user_id"`
	OpenedBy               string       `json:"opened_by"`
	TotalDisputedKobo      ledger.Money `json:"total_disputed_kobo"`
	RemainingDisputedKobo  ledger.Money `json:"remaining_disputed_kobo"`
	Reason                 string       `json:"reason"`
	Explanation            string       `json:"explanation"`
	State                  string       `json:"state"`
	CollectionEffect       string       `json:"collection_effect"`
	AssignedReviewer       string       `json:"assigned_reviewer,omitempty"`
	OpenedAt               time.Time    `json:"opened_at"`
	ResolvedAt             time.Time    `json:"resolved_at,omitempty"`
}
type Evidence struct {
	ID          string    `json:"id"`
	DisputeID   string    `json:"dispute_id"`
	SubmittedBy string    `json:"submitted_by"`
	DocumentID  string    `json:"document_id,omitempty"`
	Statement   string    `json:"statement,omitempty"`
	SubmittedAt time.Time `json:"submitted_at"`
}
type Decision struct {
	ID                    string       `json:"id"`
	DisputeID             string       `json:"dispute_id"`
	ReviewerID            string       `json:"reviewer_id"`
	Outcome               string       `json:"outcome"`
	ValidPrincipalKobo    ledger.Money `json:"valid_principal_kobo"`
	AdjustmentKobo        ledger.Money `json:"adjustment_kobo"`
	RemainingDisputedKobo ledger.Money `json:"remaining_disputed_kobo"`
	Reason                string       `json:"reason"`
	DecidedAt             time.Time    `json:"decided_at"`
}
type OpenInput struct {
	ObligationID       string
	OpenedBy           string
	DisputedAmountKobo ledger.Money
	Reason             string
	Explanation        string
	CollectionEffect   string
}
type DecideInput struct {
	DisputeID             string
	ReviewerID            string
	Outcome               string
	ValidPrincipalKobo    ledger.Money
	AdjustmentKobo        ledger.Money
	RemainingDisputedKobo ledger.Money
	Reason                string
}
type Store struct {
	mu        sync.RWMutex
	snapshot  SnapshotFunc
	ledger    ledger.Service
	apply     func(string, ledger.Money) error
	disputes  map[string]*Dispute
	evidence  map[string][]*Evidence
	decisions map[string][]*Decision
	now       func() time.Time
	newID     func() string
}

type Service interface {
	Open(OpenInput) (Dispute, error)
	AddEvidence(string, string, string, string) (Evidence, error)
	Respond(string, string, string) (Evidence, error)
	Decide(DecideInput) (Dispute, Decision, error)
	BlockedAmount(string) (ledger.Money, error)
	Get(string) (Dispute, []Evidence, []Decision, error)
	ListForOrganization(string) []Dispute
	ListForObligation(string) []Dispute
	ListForBuyer(string) []Dispute
}

var _ Service = (*Store)(nil)

func NewStore(snapshot SnapshotFunc, ledgerStore ledger.Service, apply func(string, ledger.Money) error) *Store {
	return &Store{snapshot: snapshot, ledger: ledgerStore, apply: apply, disputes: map[string]*Dispute{}, evidence: map[string][]*Evidence{}, decisions: map[string][]*Decision{}, now: func() time.Time { return time.Now().UTC() }, newID: newIdentifier}
}

func (s *Store) Open(input OpenInput) (Dispute, error) {
	if input.ObligationID == "" || input.OpenedBy == "" || input.DisputedAmountKobo <= 0 || strings.TrimSpace(input.Reason) == "" {
		return Dispute{}, errors.New("obligation, opener, positive disputed amount, and reason are required")
	}
	snapshot, err := s.snapshot(input.ObligationID)
	if err != nil {
		return Dispute{}, err
	}
	if input.DisputedAmountKobo > snapshot.OutstandingKobo {
		return Dispute{}, errors.New("disputed amount exceeds outstanding")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.disputes {
		if existing.ObligationID == input.ObligationID && (existing.State == StateOpen || existing.State == StateUnderReview || existing.State == StatePartiallyResolved) {
			return Dispute{}, errors.New("an active dispute already exists for this obligation")
		}
	}
	// A party may identify the principal it contests, but it may not choose the
	// collection policy. Until independent review, only that exact principal is
	// blocked; a nominal dispute can never suspend the whole obligation.
	dispute := &Dispute{ID: s.newID(), ObligationID: input.ObligationID, SupplierOrganizationID: snapshot.SupplierOrganizationID, BuyerUserID: snapshot.BuyerUserID, OpenedBy: input.OpenedBy, TotalDisputedKobo: input.DisputedAmountKobo, RemainingDisputedKobo: input.DisputedAmountKobo, Reason: strings.TrimSpace(input.Reason), Explanation: strings.TrimSpace(input.Explanation), State: StateOpen, CollectionEffect: EffectContestedOnly, OpenedAt: s.now()}
	s.disputes[dispute.ID] = dispute
	return cloneDispute(*dispute), nil
}
func (s *Store) AddEvidence(disputeID, submittedBy, documentID, statement string) (Evidence, error) {
	if submittedBy == "" || strings.TrimSpace(documentID) == "" && strings.TrimSpace(statement) == "" {
		return Evidence{}, errors.New("evidence submitter and document or statement are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.disputes[disputeID] == nil {
		return Evidence{}, errors.New("dispute not found")
	}
	evidence := &Evidence{ID: s.newID(), DisputeID: disputeID, SubmittedBy: submittedBy, DocumentID: strings.TrimSpace(documentID), Statement: strings.TrimSpace(statement), SubmittedAt: s.now()}
	s.evidence[disputeID] = append(s.evidence[disputeID], evidence)
	return cloneEvidence(*evidence), nil
}
func (s *Store) Respond(disputeID, actor, response string) (Evidence, error) {
	return s.AddEvidence(disputeID, actor, "", response)
}
func (s *Store) Decide(input DecideInput) (Dispute, Decision, error) {
	if input.ReviewerID == "" || strings.TrimSpace(input.Outcome) == "" || strings.TrimSpace(input.Reason) == "" {
		return Dispute{}, Decision{}, errors.New("reviewer, outcome, and reason are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	dispute := s.disputes[input.DisputeID]
	if dispute == nil {
		return Dispute{}, Decision{}, errors.New("dispute not found")
	}
	if dispute.State == StateResolved || dispute.State == StateWithdrawn {
		return Dispute{}, Decision{}, errors.New("dispute is already closed")
	}
	if input.AdjustmentKobo < 0 || input.AdjustmentKobo > dispute.RemainingDisputedKobo {
		return Dispute{}, Decision{}, errors.New("invalid adjustment amount")
	}
	if input.RemainingDisputedKobo < 0 || input.RemainingDisputedKobo > dispute.RemainingDisputedKobo {
		return Dispute{}, Decision{}, errors.New("invalid remaining disputed amount")
	}
	if input.AdjustmentKobo > dispute.RemainingDisputedKobo-input.RemainingDisputedKobo {
		return Dispute{}, Decision{}, errors.New("adjusted principal must be removed from the remaining dispute")
	}
	if input.ValidPrincipalKobo < 0 {
		return Dispute{}, Decision{}, errors.New("valid principal cannot be negative")
	}
	if input.AdjustmentKobo > 0 {
		// This development store posts the journal entry and then applies the
		// balance in two steps with no compensation between them. PostgresStore
		// does both inside one transaction and is the implementation that runs
		// anywhere money is real; the divergence is confined to local
		// development and is called out here so it is not mistaken for the
		// intended atomicity.
		if s.ledger == nil || s.apply == nil {
			return Dispute{}, Decision{}, errors.New("adjustment dependencies unavailable")
		}
		if _, err := s.ledger.PostAdjustment(dispute.ID, input.AdjustmentKobo, "dispute_adjustment", s.now(), "dispute-adjustment:"+dispute.ID+":"+fmt.Sprint(len(s.decisions[dispute.ID])+1)); err != nil {
			return Dispute{}, Decision{}, err
		}
		if err := s.apply(dispute.ObligationID, input.AdjustmentKobo); err != nil {
			return Dispute{}, Decision{}, err
		}
	}
	decision := &Decision{ID: s.newID(), DisputeID: dispute.ID, ReviewerID: input.ReviewerID, Outcome: input.Outcome, ValidPrincipalKobo: input.ValidPrincipalKobo, AdjustmentKobo: input.AdjustmentKobo, RemainingDisputedKobo: input.RemainingDisputedKobo, Reason: strings.TrimSpace(input.Reason), DecidedAt: s.now()}
	s.decisions[dispute.ID] = append(s.decisions[dispute.ID], decision)
	dispute.RemainingDisputedKobo = input.RemainingDisputedKobo
	if input.RemainingDisputedKobo == 0 {
		dispute.State = StateResolved
		dispute.ResolvedAt = s.now()
	} else {
		dispute.State = StatePartiallyResolved
	}
	return cloneDispute(*dispute), cloneDecision(*decision), nil
}
func (s *Store) BlockedAmount(obligationID string) (ledger.Money, error) {
	snapshot, err := s.snapshot(obligationID)
	if err != nil {
		return 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := ledger.Money(0)
	for _, dispute := range s.disputes {
		if dispute.ObligationID != obligationID || (dispute.State != StateOpen && dispute.State != StateUnderReview && dispute.State != StatePartiallyResolved) {
			continue
		}
		if dispute.CollectionEffect == EffectFullBlock {
			return snapshot.OutstandingKobo, nil
		}
		if dispute.CollectionEffect == EffectContestedOnly {
			total += dispute.RemainingDisputedKobo
		}
	}
	if total > snapshot.OutstandingKobo {
		total = snapshot.OutstandingKobo
	}
	return total, nil
}
func (s *Store) Get(id string) (Dispute, []Evidence, []Decision, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d := s.disputes[id]
	if d == nil {
		return Dispute{}, nil, nil, errors.New("dispute not found")
	}
	e := []Evidence{}
	for _, item := range s.evidence[id] {
		e = append(e, cloneEvidence(*item))
	}
	decisions := []Decision{}
	for _, item := range s.decisions[id] {
		decisions = append(decisions, cloneDecision(*item))
	}
	return cloneDispute(*d), e, decisions, nil
}
func (s *Store) ListForOrganization(orgID string) []Dispute {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Dispute{}
	for _, d := range s.disputes {
		if d.SupplierOrganizationID == orgID {
			out = append(out, cloneDispute(*d))
		}
	}
	return out
}

func (s *Store) ListForObligation(obligationID string) []Dispute {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Dispute{}
	for _, d := range s.disputes {
		if d.ObligationID == obligationID {
			out = append(out, cloneDispute(*d))
		}
	}
	return out
}

func (s *Store) ListForBuyer(buyerUserID string) []Dispute {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Dispute{}
	for _, d := range s.disputes {
		if d.BuyerUserID == buyerUserID {
			out = append(out, cloneDispute(*d))
		}
	}
	return out
}
func cloneDispute(v Dispute) Dispute    { return v }
func cloneEvidence(v Evidence) Evidence { return v }
func cloneDecision(v Decision) Decision { return v }
func newIdentifier() string             { return identifier.New() }
