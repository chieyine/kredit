package support

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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
	return s.OpenWithNote(context.Background(), subjectType, subjectID, actorID, organizationID, breakGlass, "")
}

// OpenWithNote commits the case and its initial message together.
var ErrInvalidInput = errors.New("invalid support case input")
var ErrClosed = errors.New("closed case cannot transition")

func (s *Store) OpenWithNote(ctx context.Context, subjectType, subjectID, actorID, organizationID string, breakGlass bool, note string) (Case, error) {
	item, _, err := s.OpenWithNoteEvents(ctx, subjectType, subjectID, actorID, organizationID, breakGlass, note)
	return item, err
}

func (s *Store) OpenWithNoteEvents(ctx context.Context, subjectType, subjectID, actorID, organizationID string, breakGlass bool, note string) (Case, []Event, error) {
	if err := ctx.Err(); err != nil {
		return Case{}, nil, err
	}
	note = strings.TrimSpace(note)
	if len(note) > 2000 {
		return Case{}, nil, fmt.Errorf("%w: message must be at most 2000 characters", ErrInvalidInput)
	}
	if strings.TrimSpace(subjectType) == "" || strings.TrimSpace(subjectID) == "" || strings.TrimSpace(actorID) == "" {
		return Case{}, nil, fmt.Errorf("%w: subject and actor are required", ErrInvalidInput)
	}
	if s.pool != nil {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return Case{}, nil, err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		var item Case
		err = tx.QueryRow(ctx, `
			INSERT INTO app.support_cases (organization_id, subject_type, subject_id, opened_by, state, break_glass)
			VALUES (NULLIF($1,'')::uuid, $2, $3, $4::uuid, 'OPEN', $5)
			RETURNING id::text, COALESCE(organization_id::text,''), subject_type, subject_id, opened_by::text, state, break_glass, created_at, updated_at`, organizationID, subjectType, subjectID, actorID, breakGlass).
			Scan(&item.ID, &item.OrganizationID, &item.SubjectType, &item.SubjectID, &item.OpenedBy, &item.State, &item.BreakGlass, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return Case{}, nil, err
		}
		var event Event
		if err := tx.QueryRow(ctx, `
			INSERT INTO app.support_case_events (case_id, actor_id, action, note)
			VALUES ($1::uuid, $2::uuid, 'opened', NULLIF($3,''))
			RETURNING id::text, case_id::text, actor_id::text, action, COALESCE(note,''), created_at`, item.ID, actorID, note).
			Scan(&event.ID, &event.CaseID, &event.ActorID, &event.Action, &event.Note, &event.CreatedAt); err != nil {
			return Case{}, nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Case{}, nil, err
		}
		return item, []Event{event}, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	now := time.Now().UTC()
	item := Case{ID: "case-" + formatID(s.counter), OrganizationID: organizationID, SubjectType: subjectType, SubjectID: subjectID, OpenedBy: actorID, State: Open, BreakGlass: breakGlass, CreatedAt: now, UpdatedAt: now}
	s.cases[item.ID] = item
	s.events[item.ID] = []Event{{ID: "event-" + formatID(s.counter), CaseID: item.ID, ActorID: actorID, Action: "opened", Note: note, CreatedAt: now}}
	return item, append([]Event(nil), s.events[item.ID]...), nil
}

func (s *Store) Transition(caseID, actorID string, state State, note string) (Case, Event, error) {
	return s.TransitionContext(context.Background(), caseID, actorID, state, note)
}

func (s *Store) TransitionContext(ctx context.Context, caseID, actorID string, state State, note string) (Case, Event, error) {
	if err := ctx.Err(); err != nil {
		return Case{}, Event{}, err
	}
	note = strings.TrimSpace(note)
	if len(note) > 2000 {
		return Case{}, Event{}, fmt.Errorf("%w: message must be at most 2000 characters", ErrInvalidInput)
	}
	if !validState(state) || strings.TrimSpace(actorID) == "" {
		return Case{}, Event{}, fmt.Errorf("%w: valid state and actor are required", ErrInvalidInput)
	}
	if s.pool != nil {
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
				return Case{}, Event{}, pgx.ErrNoRows
			}
			return Case{}, Event{}, err
		}
		if item.State == Closed {
			return Case{}, Event{}, ErrClosed
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
		return Case{}, Event{}, pgx.ErrNoRows
	}
	if item.State == Closed {
		return Case{}, Event{}, ErrClosed
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

// Read returns a case and its history from one database snapshot. Read errors
// remain distinguishable from a missing case rather than becoming empty history.
func (s *Store) Read(ctx context.Context, caseID string) (Case, []Event, error) {
	if err := ctx.Err(); err != nil {
		return Case{}, nil, err
	}
	if s.pool != nil {
		var item Case
		var history []byte
		err := s.pool.QueryRow(ctx, `SELECT c.id::text,COALESCE(c.organization_id::text,''),c.subject_type,c.subject_id,c.opened_by::text,c.state,c.break_glass,c.created_at,c.updated_at,
			(SELECT COALESCE(jsonb_agg(to_jsonb(e) ORDER BY e.created_at,e.id),'[]'::jsonb) FROM app.support_case_events e WHERE e.case_id=c.id)
			FROM app.support_cases c WHERE c.id=$1::uuid`, caseID).Scan(&item.ID, &item.OrganizationID, &item.SubjectType, &item.SubjectID, &item.OpenedBy, &item.State, &item.BreakGlass, &item.CreatedAt, &item.UpdatedAt, &history)
		if err != nil {
			return Case{}, nil, err
		}
		var events []Event
		if err := json.Unmarshal(history, &events); err != nil {
			return Case{}, nil, err
		}
		return item, events, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, found := s.cases[caseID]
	if !found {
		return Case{}, nil, pgx.ErrNoRows
	}
	return item, append([]Event{}, s.events[caseID]...), nil
}

func (s *Store) ListForOrganization(organizationID string) []Case {
	items, _ := s.ReadForOrganization(context.Background(), organizationID)
	return items
}

func (s *Store) ReadForOrganization(ctx context.Context, organizationID string) ([]Case, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.pool != nil {
		rows, err := s.pool.Query(ctx, `SELECT id::text, COALESCE(organization_id::text,''), subject_type, subject_id, opened_by::text, state, break_glass, created_at, updated_at FROM app.support_cases WHERE organization_id = $1::uuid ORDER BY created_at DESC, id DESC`, organizationID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		result := make([]Case, 0)
		for rows.Next() {
			var item Case
			if err := rows.Scan(&item.ID, &item.OrganizationID, &item.SubjectType, &item.SubjectID, &item.OpenedBy, &item.State, &item.BreakGlass, &item.CreatedAt, &item.UpdatedAt); err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return result, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Case, 0)
	for _, item := range s.cases {
		if item.OrganizationID == organizationID {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID > result[j].ID
		}
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func validState(state State) bool {
	return state == Open || state == InProgress || state == Resolved || state == Closed
}
func formatID(value int64) string {
	return time.Unix(0, value).UTC().Format("20060102150405.000000000")
}
