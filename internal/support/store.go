package support

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type State string

const (
	Open       State = "OPEN"
	InProgress State = "IN_PROGRESS"
	Resolved   State = "RESOLVED"
	Closed     State = "CLOSED"
)

type Case struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id,omitempty"`
	SubjectType    string    `json:"subject_type"`
	SubjectID      string    `json:"subject_id"`
	OpenedBy       string    `json:"opened_by"`
	State          State     `json:"state"`
	BreakGlass     bool      `json:"break_glass"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Event struct {
	ID        string    `json:"id"`
	CaseID    string    `json:"case_id"`
	ActorID   string    `json:"actor_id"`
	Action    string    `json:"action"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct {
	mu      sync.RWMutex
	cases   map[string]Case
	events  map[string][]Event
	counter int64
	pool    *pgxpool.Pool
}

func NewStore() *Store { return &Store{cases: make(map[string]Case), events: make(map[string][]Event)} }

// NewPostgresStore uses the same service contract as the development store,
// but persists cases and their timeline in PostgreSQL. The in-memory maps are
// intentionally not used in this mode, so a second API or worker instance sees
// the same support history.
func NewPostgresStore(pool *pgxpool.Pool) *Store {
	return &Store{cases: make(map[string]Case), events: make(map[string][]Event), pool: pool}
}

func (s *Store) Open(subjectType, subjectID, actorID, organizationID string, breakGlass bool) (Case, error) {
	if strings.TrimSpace(subjectType) == "" || strings.TrimSpace(subjectID) == "" || strings.TrimSpace(actorID) == "" {
		return Case{}, errors.New("subject and actor are required")
	}
	if s.pool != nil {
		ctx := context.Background()
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return Case{}, err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		var item Case
		err = tx.QueryRow(ctx, `
			INSERT INTO app.support_cases (organization_id, subject_type, subject_id, opened_by, state, break_glass)
			VALUES (NULLIF($1,'')::uuid, $2, $3, $4::uuid, 'OPEN', $5)
			RETURNING id::text, COALESCE(organization_id::text,''), subject_type, subject_id, opened_by::text, state, break_glass, created_at, updated_at`, organizationID, subjectType, subjectID, actorID, breakGlass).
			Scan(&item.ID, &item.OrganizationID, &item.SubjectType, &item.SubjectID, &item.OpenedBy, &item.State, &item.BreakGlass, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return Case{}, err
		}
		var event Event
		if err := tx.QueryRow(ctx, `
			INSERT INTO app.support_case_events (case_id, actor_id, action)
			VALUES ($1::uuid, $2::uuid, 'opened')
			RETURNING id::text, case_id::text, actor_id::text, action, COALESCE(note,''), created_at`, item.ID, actorID).
			Scan(&event.ID, &event.CaseID, &event.ActorID, &event.Action, &event.Note, &event.CreatedAt); err != nil {
			return Case{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Case{}, err
		}
		return item, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	now := time.Now().UTC()
	item := Case{ID: "case-" + formatID(s.counter), OrganizationID: organizationID, SubjectType: subjectType, SubjectID: subjectID, OpenedBy: actorID, State: Open, BreakGlass: breakGlass, CreatedAt: now, UpdatedAt: now}
	s.cases[item.ID] = item
	s.events[item.ID] = []Event{{ID: "event-" + formatID(s.counter), CaseID: item.ID, ActorID: actorID, Action: "opened", CreatedAt: now}}
	return item, nil
}

func (s *Store) Transition(caseID, actorID string, state State, note string) (Case, Event, error) {
	if !validState(state) || strings.TrimSpace(actorID) == "" {
		return Case{}, Event{}, errors.New("valid state and actor are required")
	}
	if s.pool != nil {
		ctx := context.Background()
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return Case{}, Event{}, err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		var item Case
		err = tx.QueryRow(ctx, `
			SELECT id::text, COALESCE(organization_id::text,''), subject_type, subject_id, opened_by::text, state, break_glass, created_at, updated_at
			FROM app.support_cases WHERE id = $1::uuid FOR UPDATE`, caseID).
			Scan(&item.ID, &item.OrganizationID, &item.SubjectType, &item.SubjectID, &item.OpenedBy, &item.State, &item.BreakGlass, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Case{}, Event{}, errors.New("case not found")
			}
			return Case{}, Event{}, err
		}
		if item.State == Closed {
			return Case{}, Event{}, errors.New("closed case cannot transition")
		}
		if err := tx.QueryRow(ctx, `UPDATE app.support_cases SET state = $2, updated_at = now() WHERE id = $1::uuid RETURNING updated_at`, caseID, string(state)).Scan(&item.UpdatedAt); err != nil {
			return Case{}, Event{}, err
		}
		item.State = state
		var event Event
		if err := tx.QueryRow(ctx, `
			INSERT INTO app.support_case_events (case_id, actor_id, action, note)
			VALUES ($1::uuid, $2::uuid, 'state_changed', NULLIF($3,''))
			RETURNING id::text, case_id::text, actor_id::text, action, COALESCE(note,''), created_at`, caseID, actorID, note).
			Scan(&event.ID, &event.CaseID, &event.ActorID, &event.Action, &event.Note, &event.CreatedAt); err != nil {
			return Case{}, Event{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Case{}, Event{}, err
		}
		return item, event, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.cases[caseID]
	if !ok {
		return Case{}, Event{}, errors.New("case not found")
	}
	if item.State == Closed {
		return Case{}, Event{}, errors.New("closed case cannot transition")
	}
	s.counter++
	now := time.Now().UTC()
	item.State, item.UpdatedAt = state, now
	s.cases[caseID] = item
	event := Event{ID: "event-" + formatID(s.counter), CaseID: caseID, ActorID: actorID, Action: "state_changed", Note: note, CreatedAt: now}
	s.events[caseID] = append(s.events[caseID], event)
	return item, event, nil
}

func (s *Store) Timeline(caseID string) []Event {
	if s.pool != nil {
		rows, err := s.pool.Query(context.Background(), `SELECT id::text, case_id::text, actor_id::text, action, COALESCE(note,''), created_at FROM app.support_case_events WHERE case_id = $1::uuid ORDER BY created_at`, caseID)
		if err != nil {
			return nil
		}
		defer rows.Close()
		result := make([]Event, 0)
		for rows.Next() {
			var event Event
			if err := rows.Scan(&event.ID, &event.CaseID, &event.ActorID, &event.Action, &event.Note, &event.CreatedAt); err != nil {
				return nil
			}
			result = append(result, event)
		}
		if rows.Err() != nil {
			return nil
		}
		return result
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Event(nil), s.events[caseID]...)
}

func (s *Store) Get(caseID string) (Case, bool) {
	if s.pool != nil {
		var item Case
		err := s.pool.QueryRow(context.Background(), `SELECT id::text, COALESCE(organization_id::text,''), subject_type, subject_id, opened_by::text, state, break_glass, created_at, updated_at FROM app.support_cases WHERE id = $1::uuid`, caseID).
			Scan(&item.ID, &item.OrganizationID, &item.SubjectType, &item.SubjectID, &item.OpenedBy, &item.State, &item.BreakGlass, &item.CreatedAt, &item.UpdatedAt)
		return item, err == nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.cases[caseID]
	return item, ok
}

func (s *Store) ListForOrganization(organizationID string) []Case {
	if s.pool != nil {
		rows, err := s.pool.Query(context.Background(), `SELECT id::text, COALESCE(organization_id::text,''), subject_type, subject_id, opened_by::text, state, break_glass, created_at, updated_at FROM app.support_cases WHERE organization_id = $1::uuid ORDER BY created_at DESC`, organizationID)
		if err != nil {
			return nil
		}
		defer rows.Close()
		result := make([]Case, 0)
		for rows.Next() {
			var item Case
			if err := rows.Scan(&item.ID, &item.OrganizationID, &item.SubjectType, &item.SubjectID, &item.OpenedBy, &item.State, &item.BreakGlass, &item.CreatedAt, &item.UpdatedAt); err != nil {
				return nil
			}
			result = append(result, item)
		}
		if rows.Err() != nil {
			return nil
		}
		return result
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Case, 0)
	for _, item := range s.cases {
		if item.OrganizationID == organizationID {
			result = append(result, item)
		}
	}
	return result
}

func validState(state State) bool {
	return state == Open || state == InProgress || state == Resolved || state == Closed
}
func formatID(value int64) string {
	return time.Unix(0, value).UTC().Format("20060102150405.000000000")
}
