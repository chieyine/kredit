package businesspolicy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"kredit/internal/config"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Pool interface {
	Queryer
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Begin(context.Context) (pgx.Tx, error)
}
type Store struct {
	pool       Pool
	deployment config.Config
}

func NewStore(pool Pool, c config.Config) *Store { return &Store{pool: pool, deployment: c} }

type Snapshot struct {
	Initialized bool   `json:"-"`
	Revision    int64  `json:"revision"`
	Values      Values `json:"values"`
}
type Change struct {
	ID           string     `json:"id"`
	Revision     int64      `json:"revision"`
	BaseRevision int64      `json:"base_revision"`
	Values       Values     `json:"values"`
	Before       Values     `json:"before_values"`
	ProposedBy   string     `json:"proposed_by"`
	Reason       string     `json:"reason"`
	EffectiveAt  time.Time  `json:"effective_at"`
	CreatedAt    time.Time  `json:"created_at"`
	State        string     `json:"state"`
	DecidedBy    *string    `json:"decided_by"`
	DecidedAt    *time.Time `json:"decided_at"`
}
type Event struct {
	ChangeID string    `json:"change_id"`
	ActorID  string    `json:"actor_id"`
	Action   string    `json:"action"`
	Reason   string    `json:"reason"`
	At       time.Time `json:"occurred_at"`
}
type Proposal struct {
	ID           string    `json:"id"`
	BaseRevision int64     `json:"base_revision"`
	Values       Values    `json:"values"`
	Reason       string    `json:"reason"`
	EffectiveAt  time.Time `json:"effective_at"`
}

func (s *Store) Ensure(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return errors.New("persistent policies are unavailable")
	}
	v := Defaults(s.deployment)
	if err := v.Validate(); err != nil {
		return err
	}
	b, _ := json.Marshal(v)
	_, err := s.pool.Exec(ctx, `INSERT INTO app.business_policy_defaults(singleton,values) VALUES(true,$1::jsonb) ON CONFLICT DO NOTHING`, b)
	return err
}

type Queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func ReadTx(ctx context.Context, q Queryer) (Snapshot, error) {
	var out Snapshot
	var b []byte
	err := q.QueryRow(ctx, `SELECT COALESCE((SELECT revision FROM app.business_policy_changes WHERE state='approved' AND effective_at<=now() ORDER BY revision DESC LIMIT 1),0),app.business_policy()`).Scan(&out.Revision, &b)
	if err != nil {
		return out, err
	}
	if string(b) == "{}" {
		return Snapshot{Values: Defaults(config.Config{})}, nil
	}
	out.Initialized = true
	if err = json.Unmarshal(b, &out.Values); err != nil {
		return out, err
	}
	return out, out.Values.Validate()
}
func (s *Store) Read(ctx context.Context) (Snapshot, error) {
	if err := s.Ensure(ctx); err != nil {
		return Snapshot{}, err
	}
	return ReadTx(ctx, s.pool)
}
func (s *Store) History(ctx context.Context) ([]Change, []Event, error) {
	rows, err := s.pool.Query(ctx, `SELECT c.id::text,c.revision,c.base_revision,c.values,c.proposed_by::text,c.reason,c.effective_at,c.created_at,c.state,c.decided_by::text,c.decided_at,COALESCE((SELECT values FROM app.business_policy_changes b WHERE b.revision=c.base_revision),(SELECT values FROM app.business_policy_defaults WHERE singleton)) FROM app.business_policy_changes c ORDER BY revision DESC LIMIT 100`)
	if err != nil {
		return nil, nil, err
	}
	changes := []Change{}
	for rows.Next() {
		var c Change
		var b, before []byte
		if err = rows.Scan(&c.ID, &c.Revision, &c.BaseRevision, &b, &c.ProposedBy, &c.Reason, &c.EffectiveAt, &c.CreatedAt, &c.State, &c.DecidedBy, &c.DecidedAt, &before); err != nil {
			rows.Close()
			return nil, nil, err
		}
		if err = json.Unmarshal(b, &c.Values); err != nil {
			rows.Close()
			return nil, nil, err
		}
		if err = json.Unmarshal(before, &c.Before); err != nil {
			rows.Close()
			return nil, nil, err
		}
		changes = append(changes, c)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, err
	}
	rows, err = s.pool.Query(ctx, `SELECT change_id::text,actor_id::text,action,reason,occurred_at FROM app.business_policy_events WHERE change_id IN(SELECT id FROM app.business_policy_changes ORDER BY revision DESC LIMIT 100) ORDER BY occurred_at DESC`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	events := []Event{}
	for rows.Next() {
		var e Event
		if err = rows.Scan(&e.ChangeID, &e.ActorID, &e.Action, &e.Reason, &e.At); err != nil {
			return nil, nil, err
		}
		events = append(events, e)
	}
	return changes, events, rows.Err()
}
func admin(ctx context.Context, tx pgx.Tx, actor string) error {
	var allowed bool
	err := tx.QueryRow(ctx, `SELECT app.is_active_policy_admin($1::uuid)`, actor).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return errors.New("an active platform administrator is required")
	}
	return nil
}
func reasonOK(s string) bool { return len(strings.TrimSpace(s)) >= 8 && len(s) <= 2000 }
func (s *Store) lock(ctx context.Context) (pgx.Tx, error) {
	if err := s.Ensure(ctx); err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `SELECT singleton FROM app.business_policy_defaults WHERE singleton FOR UPDATE`)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}
func (s *Store) Propose(ctx context.Context, actor string, in Proposal) (string, error) {
	if _, err := uuid.Parse(in.ID); err != nil {
		return "", errors.New("a unique change identifier is required")
	}
	if !reasonOK(in.Reason) {
		return "", errors.New("provide a reason between 8 and 2000 characters")
	}
	if err := in.Values.ValidateDeployment(s.deployment); err != nil {
		return "", err
	}
	tx, err := s.lock(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = admin(ctx, tx, actor); err != nil {
		return "", err
	}
	// Explicit proposal IDs retain retry safety even beyond the HTTP cache lifetime.
	var existing []byte
	var who, why string
	var base int64
	var at time.Time
	err = tx.QueryRow(ctx, `SELECT values,proposed_by::text,reason,base_revision,effective_at FROM app.business_policy_changes WHERE id=$1::uuid`, in.ID).Scan(&existing, &who, &why, &base, &at)
	if err == nil {
		var v Values
		_ = json.Unmarshal(existing, &v)
		if who != actor || why != strings.TrimSpace(in.Reason) || base != in.BaseRevision || !at.Equal(in.EffectiveAt) || Changed(v, in.Values) {
			return "", errors.New("change identifier was already used for different settings")
		}
		return in.ID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	current, err := ReadTx(ctx, tx)
	if err != nil {
		return "", err
	}
	if current.Revision != in.BaseRevision {
		return "", errors.New("settings changed; refresh before proposing")
	}
	if !Changed(current.Values, in.Values) {
		return "", errors.New("the proposal does not change any settings")
	}
	var busy bool
	var now time.Time
	err = tx.QueryRow(ctx, `SELECT clock_timestamp(),EXISTS(SELECT 1 FROM app.business_policy_changes WHERE state='pending' OR (state='approved' AND effective_at>now()))`).Scan(&now, &busy)
	if err != nil {
		return "", err
	}
	if busy {
		return "", errors.New("resolve or cancel the pending or scheduled change first")
	}
	if !in.EffectiveAt.After(now) || in.EffectiveAt.After(now.AddDate(1, 0, 0)) {
		return "", errors.New("choose a future effective date within one year")
	}
	b, _ := json.Marshal(in.Values)
	_, err = tx.Exec(ctx, `INSERT INTO app.business_policy_changes(id,base_revision,values,proposed_by,reason,effective_at) VALUES($1::uuid,$2,$3::jsonb,$4::uuid,$5,$6)`, in.ID, in.BaseRevision, b, actor, strings.TrimSpace(in.Reason), in.EffectiveAt)
	if err != nil {
		return "", err
	}
	if err = record(ctx, tx, in.ID, actor, "proposed", in.Reason); err != nil {
		return "", err
	}
	return in.ID, tx.Commit(ctx)
}
func (s *Store) Decide(ctx context.Context, id, actor, action, reason string) error {
	if !reasonOK(reason) {
		return errors.New("provide a reason between 8 and 2000 characters")
	}
	if action != "approve" && action != "reject" && action != "cancel" {
		return errors.New("unsupported decision")
	}
	tx, err := s.lock(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var allowed bool
	roles := []string{"platform_admin", "approver"}
	if action == "cancel" {
		roles = append(roles, "policy_manager")
	}
	if err = tx.QueryRow(ctx, `SELECT app.has_admin_role($1::uuid,$2::text[])`, actor, roles).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return errors.New("this role cannot approve policy changes")
	}

	var author, state string
	var base int64
	var at, now time.Time
	var b []byte
	err = tx.QueryRow(ctx, `SELECT proposed_by::text,state,base_revision,effective_at,values,clock_timestamp() FROM app.business_policy_changes WHERE id=$1::uuid FOR UPDATE`, id).Scan(&author, &state, &base, &at, &b, &now)
	if err != nil {
		return err
	}
	if state != "pending" && (action != "cancel" || state != "approved" || !at.After(now)) {
		return errors.New("this change can no longer receive that decision")
	}
	if action == "approve" {
		if err = admin(ctx, tx, author); err != nil {
			return errors.New("the proposing administrator is no longer authorized; cancel and submit a new proposal")
		}
		if actor == author {
			return errors.New("another platform administrator must approve this change")
		}
		if !at.After(now) {
			return errors.New("effective date has passed; cancel and submit a new proposal")
		}
		current, err := ReadTx(ctx, tx)
		if err != nil {
			return err
		}
		if current.Revision != base {
			return errors.New("the proposal is based on outdated settings")
		}
		var v Values
		if err = json.Unmarshal(b, &v); err != nil {
			return err
		}
		if err = v.ValidateDeployment(s.deployment); err != nil {
			return err
		}
	}
	next := map[string]string{"approve": "approved", "reject": "rejected", "cancel": "cancelled"}[action]
	_, err = tx.Exec(ctx, `UPDATE app.business_policy_changes SET state=$2,decided_by=$3::uuid,decided_at=clock_timestamp() WHERE id=$1::uuid`, id, next, actor)
	if err != nil {
		return err
	}
	if err = record(ctx, tx, id, actor, action, reason); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func record(ctx context.Context, tx pgx.Tx, id, actor, action, reason string) error {
	_, err := tx.Exec(ctx, `INSERT INTO app.business_policy_events(change_id,actor_id,action,reason) VALUES($1::uuid,$2::uuid,$3,$4)`, id, actor, action, strings.TrimSpace(reason))
	if err != nil {
		return fmt.Errorf("record policy decision: %w", err)
	}
	return nil
}

// ValidateStartup prevents a tighter deployment approval from being silently
// exceeded by an existing active or scheduled admin policy.
func (s *Store) ValidateStartup(ctx context.Context) error {
	snapshot, err := s.Read(ctx)
	if err != nil {
		return err
	}
	if err = snapshot.Values.ValidateDeployment(s.deployment); err != nil {
		return err
	}
	rows, err := s.pool.Query(ctx, `SELECT values FROM app.business_policy_changes WHERE state='approved' AND effective_at>now()`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var b []byte
		if err = rows.Scan(&b); err != nil {
			return err
		}
		var v Values
		if err = json.Unmarshal(b, &v); err != nil {
			return err
		}
		if err = v.ValidateDeployment(s.deployment); err != nil {
			return err
		}
	}
	return rows.Err()
}
