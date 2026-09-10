package audit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"kredit/internal/platform/logging"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	ID             string            `json:"id"`
	At             time.Time         `json:"created_at"`
	ActorUserID    string            `json:"actor_user_id"`
	OrganizationID string            `json:"organization_id"`
	Action         string            `json:"action"`
	ResourceType   string            `json:"resource_type"`
	ResourceID     string            `json:"resource_id"`
	Outcome        string            `json:"outcome"`
	RequestID      string            `json:"request_id"`
	Severity       string            `json:"severity"`
	Metadata       map[string]string `json:"metadata"`
}

type Store struct {
	mu     sync.RWMutex
	events []Event
}

type Service interface {
	Append(Event) Event
	ListForOrganization(string) []Event
	ReadForOrganization(context.Context, string) ([]Event, error)
}

func NewStore() *Store { return &Store{} }

func (s *Store) Append(event Event) Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	if event.ID == "" {
		event.ID = newID()
	}
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	if event.Outcome == "" {
		event.Outcome = "success"
	}
	if event.Severity == "" {
		event.Severity = "info"
	}
	event.Metadata = cloneMetadata(event.Metadata)
	s.events = append(s.events, event)
	return cloneEvent(event)
}

func (s *Store) ListForOrganization(organizationID string) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Event, 0)
	for _, event := range s.events {
		if event.OrganizationID == organizationID {
			result = append(result, cloneEvent(event))
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].At.After(result[j].At) })
	return result
}

func cloneEvent(event Event) Event {
	event.Metadata = cloneMetadata(event.Metadata)
	return event
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return nil
	}
	copyOf := make(map[string]string, len(metadata))
	for key, value := range metadata {
		key = strings.TrimSpace(key)
		if key == "" || len(key) > 64 || sensitiveMetadataKey(key) {
			continue
		}
		value = logging.Redact(value)
		if len(value) > 512 {
			value = value[:512]
		}
		copyOf[key] = value
	}
	return copyOf
}

func sensitiveMetadataKey(key string) bool {
	key = strings.ToLower(key)
	for _, fragment := range []string{"token", "secret", "password", "otp", "pin", "bvn", "nin", "phone", "email", "account", "document", "address", "authorization", "cookie"} {
		if strings.Contains(key, fragment) {
			return true
		}
	}
	return false
}

func newID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(value[:])
}

// PostgresStore keeps audit events append-only in the database. The SQL table
// owns the event timestamp and UUID so multiple API instances share one audit
// timeline.
type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func (s *PostgresStore) Append(event Event) Event {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	saved, err := s.Record(ctx, event)
	if err != nil {
		slog.Error("audit event persistence failed", "action", event.Action, "resource_type", event.ResourceType, "error", logging.Redact(err.Error()))
	}
	return saved
}

// Record returns persistence failures and installs the event's authorized scope.
// Domain transactions still own their atomic decision/event records.
func (s *PostgresStore) Record(ctx context.Context, event Event) (Event, error) {
	event.Metadata = cloneMetadata(event.Metadata)
	event.Outcome = defaultString(event.Outcome, "success")
	event.Severity = defaultString(event.Severity, "info")
	if s == nil || s.pool == nil {
		return event, errors.New("audit database is not configured")
	}
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return event, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return event, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_organization_id',$1,true),set_config('app.current_user_id',$2,true)`, event.OrganizationID, event.ActorUserID); err != nil {
		return event, err
	}
	var id string
	var occurredAt time.Time
	err = tx.QueryRow(ctx, `INSERT INTO app.audit_events (actor_user_id,organization_id,action,resource_type,resource_id,outcome,severity,request_id,metadata) VALUES(NULLIF($1,'')::uuid,NULLIF($2,'')::uuid,$3,$4,NULLIF($5,''),$6,$7,NULLIF($8,''),$9::jsonb) RETURNING id::text,occurred_at`, event.ActorUserID, event.OrganizationID, event.Action, event.ResourceType, event.ResourceID, event.Outcome, event.Severity, event.RequestID, metadata).Scan(&id, &occurredAt)
	if err != nil {
		return event, err
	}
	if err = tx.Commit(ctx); err != nil {
		return event, err
	}
	event.ID, event.At = id, occurredAt
	return event, nil
}

func (s *Store) ReadForOrganization(ctx context.Context, org string) ([]Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.ListForOrganization(org), nil
}
func (s *PostgresStore) ListForOrganization(org string) []Event {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	items, _ := s.ReadForOrganization(ctx, org)
	return items
}
func (s *PostgresStore) ReadForOrganization(ctx context.Context, org string) ([]Event, error) {
	if s == nil || s.pool == nil || org == "" {
		return nil, errors.New("audit scope or database is unavailable")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_organization_id',$1,true)`, org); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text,occurred_at,COALESCE(actor_user_id::text,''),COALESCE(organization_id::text,''),action,COALESCE(resource_type,''),COALESCE(resource_id,''),outcome,COALESCE(request_id,''),severity,metadata FROM app.audit_events WHERE organization_id=$1::uuid ORDER BY occurred_at DESC,id DESC`, org)
	if err != nil {
		return nil, err
	}
	result := []Event{}
	for rows.Next() {
		var event Event
		var metadata []byte
		if err = rows.Scan(&event.ID, &event.At, &event.ActorUserID, &event.OrganizationID, &event.Action, &event.ResourceType, &event.ResourceID, &event.Outcome, &event.RequestID, &event.Severity, &metadata); err != nil {
			rows.Close()
			return nil, err
		}
		if event.Metadata, err = DecodeMetadata(metadata); err != nil {
			rows.Close()
			return nil, err
		}
		result = append(result, event)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// DecodeMetadata tolerates native JSON scalars written by database auditing.
// Historical immutable events need not be rewritten to make their history readable.
func DecodeMetadata(raw []byte) (map[string]string, error) {
	values := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	result := map[string]string{}
	for key, value := range values {
		var text string
		if json.Unmarshal(value, &text) != nil {
			text = string(value)
		}
		result[key] = text
	}
	return cloneMetadata(result), nil
}
