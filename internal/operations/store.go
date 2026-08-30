package operations

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"kredit/internal/identifier"
	"kredit/internal/ledger"
)

const highValueThreshold ledger.Money = 1000000

type Action struct {
	ID                  string       `json:"id"`
	ActorUserID         string       `json:"actor_user_id"`
	OrganizationID      string       `json:"organization_id"`
	ActionType          string       `json:"action_type"`
	ObligationID        string       `json:"obligation_id"`
	AmountKobo          ledger.Money `json:"amount_kobo"`
	Reason              string       `json:"reason"`
	ApprovedBy          string       `json:"approved_by,omitempty"`
	LedgerTransactionID string       `json:"ledger_transaction_id"`
	CreatedAt           time.Time    `json:"created_at"`
}
type Store struct {
	mu              sync.RWMutex
	ledger          ledger.Service
	applyAdjustment func(string, ledger.Money) error
	actions         map[string]*Action
	now             func() time.Time
	newID           func() string
}

type Service interface {
	WriteOff(string, string, string, ledger.Money, string, string) (Action, error)
	WaiveFee(string, string, string, ledger.Money, string, string) (Action, error)
	ListForOrganization(string) []Action
}

type IdempotentService interface {
	WriteOffWithKey(string, string, string, ledger.Money, string, string, string) (Action, error)
	WaiveFeeWithKey(string, string, string, ledger.Money, string, string, string) (Action, error)
}

var _ Service = (*Store)(nil)

func NewStore(ledgerStore ledger.Service, apply func(string, ledger.Money) error) *Store {
	return &Store{ledger: ledgerStore, applyAdjustment: apply, actions: map[string]*Action{}, now: func() time.Time { return time.Now().UTC() }, newID: newIdentifier}
}
func (s *Store) WriteOff(actor, org, obligation string, amount ledger.Money, reason, approvedBy string) (Action, error) {
	return s.adjust(actor, org, obligation, amount, reason, approvedBy, "write_off")
}
func (s *Store) WaiveFee(actor, org, obligation string, amount ledger.Money, reason, approvedBy string) (Action, error) {
	return s.adjust(actor, org, obligation, amount, reason, approvedBy, "fee_waiver")
}
func (s *Store) adjust(actor, org, obligation string, amount ledger.Money, reason, approvedBy, kind string) (Action, error) {
	if actor == "" || org == "" || obligation == "" || amount <= 0 || strings.TrimSpace(reason) == "" {
		return Action{}, errors.New("actor, organisation, obligation, positive amount, and reason are required")
	}
	if strings.TrimSpace(approvedBy) != "" && strings.TrimSpace(approvedBy) == strings.TrimSpace(actor) {
		return Action{}, errors.New("approval must be performed by a different user")
	}
	if amount >= highValueThreshold && strings.TrimSpace(approvedBy) == "" {
		return Action{}, errors.New("high-value operation requires separate approval")
	}
	if s.ledger == nil {
		return Action{}, errors.New("ledger unavailable")
	}
	var tx ledger.Transaction
	var err error
	if kind == "write_off" {
		tx, err = s.ledger.PostAdjustment(obligation, amount, "write_off", s.now(), "operation:write-off:"+obligation+":"+fmt.Sprint(s.now().UnixNano()))
	} else {
		tx, err = s.ledger.PostFeeWaiver(obligation, amount, s.now(), "operation:fee-waiver:"+obligation+":"+fmt.Sprint(s.now().UnixNano()))
	}
	if err != nil {
		return Action{}, err
	}
	if kind == "write_off" && s.applyAdjustment != nil {
		if err = s.applyAdjustment(obligation, amount); err != nil {
			return Action{}, err
		}
	}
	action := &Action{ID: s.newID(), ActorUserID: actor, OrganizationID: org, ActionType: kind, ObligationID: obligation, AmountKobo: amount, Reason: strings.TrimSpace(reason), ApprovedBy: approvedBy, LedgerTransactionID: tx.ID, CreatedAt: s.now()}
	s.mu.Lock()
	s.actions[action.ID] = action
	s.mu.Unlock()
	return cloneAction(*action), nil
}
func (s *Store) ListForOrganization(org string) []Action {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Action{}
	for _, action := range s.actions {
		if action.OrganizationID == org {
			out = append(out, cloneAction(*action))
		}
	}
	return out
}
func cloneAction(v Action) Action { return v }
func newIdentifier() string       { return identifier.New() }
