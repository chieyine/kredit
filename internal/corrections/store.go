package corrections

import (
	"errors"
	"strings"
	"sync"
	"time"
)

const (
	StateOpen     = "OPEN"
	StateReview   = "UNDER_REVIEW"
	StateApproved = "APPROVED"
	StateRejected = "REJECTED"
)

type Request struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	SubjectType    string    `json:"subject_type"`
	SubjectID      string    `json:"subject_id"`
	SourceEventID  string    `json:"source_event_id"`
	RequestedBy    string    `json:"requested_by"`
	Reason         string    `json:"reason"`
	Evidence       []string  `json:"evidence,omitempty"`
	State          string    `json:"state"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Decision struct {
	ID           string    `json:"id"`
	RequestID    string    `json:"request_id"`
	ReviewerID   string    `json:"reviewer_id"`
	Outcome      string    `json:"outcome"`
	Reason       string    `json:"reason"`
	CorrectionID string    `json:"correction_event_id"`
	DecidedAt    time.Time `json:"decided_at"`
}

type Store struct {
	mu        sync.RWMutex
	requests  map[string]*Request
	decisions map[string][]Decision
	now       func() time.Time
	next      uint64
}

type Service interface {
	Open(string, string, string, string, string, string, []string) (Request, error)
	StartReview(string, string) (Request, error)
	Decide(string, string, string, string) (Request, Decision, error)
	Get(string) (Request, []Decision, error)
	ListForOrganization(string) []Request
}

var _ Service = (*Store)(nil)

func NewStore() *Store {
	return &Store{requests: map[string]*Request{}, decisions: map[string][]Decision{}, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Store) Open(organizationID, subjectType, subjectID, sourceEventID, requestedBy, reason string, evidence []string) (Request, error) {
	if organizationID == "" || subjectType == "" || subjectID == "" || requestedBy == "" || strings.TrimSpace(reason) == "" {
		return Request{}, errors.New("organization, subject, requester, and reason are required")
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	id := "correction-" + now.Format("20060102150405") + "-" + stringID(s.next)
	r := &Request{ID: id, OrganizationID: organizationID, SubjectType: subjectType, SubjectID: subjectID, SourceEventID: strings.TrimSpace(sourceEventID), RequestedBy: requestedBy, Reason: strings.TrimSpace(reason), Evidence: append([]string(nil), evidence...), State: StateOpen, CreatedAt: now, UpdatedAt: now}
	s.requests[id] = r
	return clone(*r), nil
}

func (s *Store) StartReview(id, reviewerID string) (Request, error) {
	return s.transition(id, reviewerID, StateReview)
}

func (s *Store) Decide(id, reviewerID, outcome, reason string) (Request, Decision, error) {
	if reviewerID == "" || strings.TrimSpace(reason) == "" {
		return Request{}, Decision{}, errors.New("reviewer and reason are required")
	}
	if outcome != StateApproved && outcome != StateRejected {
		return Request{}, Decision{}, errors.New("outcome must be approved or rejected")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.requests[id]
	if r == nil {
		return Request{}, Decision{}, errors.New("correction request not found")
	}
	if r.State != StateOpen && r.State != StateReview {
		return Request{}, Decision{}, errors.New("correction request is already decided")
	}
	if reviewerID == r.RequestedBy {
		return Request{}, Decision{}, errors.New("correction requester cannot approve or reject their own request")
	}
	now := s.now()
	r.State = outcome
	r.UpdatedAt = now
	s.next++
	d := Decision{ID: "correction-decision-" + stringID(s.next), RequestID: id, ReviewerID: reviewerID, Outcome: outcome, Reason: strings.TrimSpace(reason), DecidedAt: now}
	if outcome == StateApproved {
		s.next++
		d.CorrectionID = "correction-event-" + stringID(s.next)
	}
	s.decisions[id] = append(s.decisions[id], d)
	return clone(*r), d, nil
}

func (s *Store) Get(id string) (Request, []Decision, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r := s.requests[id]
	if r == nil {
		return Request{}, nil, errors.New("correction request not found")
	}
	return clone(*r), append([]Decision(nil), s.decisions[id]...), nil
}
func (s *Store) ListForOrganization(orgID string) []Request {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Request{}
	for _, r := range s.requests {
		if r.OrganizationID == orgID {
			out = append(out, clone(*r))
		}
	}
	return out
}
func (s *Store) transition(id, actor, state string) (Request, error) {
	if actor == "" {
		return Request{}, errors.New("actor is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.requests[id]
	if r == nil {
		return Request{}, errors.New("correction request not found")
	}
	if r.State != StateOpen {
		return Request{}, errors.New("correction request cannot enter review")
	}
	r.State = state
	r.UpdatedAt = s.now()
	return clone(*r), nil
}
func clone(r Request) Request { r.Evidence = append([]string(nil), r.Evidence...); return r }
func stringID(n uint64) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	if n == 0 {
		return "0"
	}
	out := ""
	for n > 0 {
		out = string(alphabet[n%uint64(len(alphabet))]) + out
		n /= uint64(len(alphabet))
	}
	return out
}
