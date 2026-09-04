package audit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

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
	return event
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
		value = strings.Map(func(r rune) rune {
			if r == '\n' || r == '\r' || r == '\t' {
				return ' '
			}
			return r
		}, value)
		if len(value) > 512 {
			value = value[:512]
		}
		copyOf[key] = value
	}
	return copyOf
}

func sensitiveMetadataKey(key string) bool {
	key = strings.ToLower(key)
	for _, fragment := range []string{"token", "secret", "password", "otp", "pin", "bvn", "nin", "phone", "email", "account", "document", "address"} {
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
	if s == nil || s.pool == nil {
		return event
	}
	metadata, err := json.Marshal(cloneMetadata(event.Metadata))
	if err != nil {
		return event
	}
	var id string
	var occurredAt time.Time
	err = s.pool.QueryRow(context.Background(), `
		INSERT INTO app.audit_events (actor_user_id, organization_id, action, resource_type, resource_id, outcome, severity, request_id, metadata)
		VALUES (NULLIF($1,'')::uuid, NULLIF($2,'')::uuid, $3, $4, NULLIF($5,''), $6, $7, NULLIF($8,''), $9::jsonb)
		RETURNING id::text, occurred_at`, event.ActorUserID, event.OrganizationID, event.Action, event.ResourceType, event.ResourceID, defaultString(event.Outcome, "success"), defaultString(event.Severity, "info"), event.RequestID, metadata).Scan(&id, &occurredAt)
	if err != nil {
		return event
	}
	event.ID, event.At = id, occurredAt
	return event
}

func (s *PostgresStore) ListForOrganization(organizationID string) []Event {
	if s == nil || s.pool == nil || organizationID == "" {
		return nil
	}
	rows, err := s.pool.Query(context.Background(), `
		SELECT id::text, occurred_at, COALESCE(actor_user_id::text,''), COALESCE(organization_id::text,''), action, COALESCE(resource_type,''), COALESCE(resource_id,''), outcome, COALESCE(request_id,''), severity, metadata
		FROM app.audit_events WHERE organization_id = $1::uuid ORDER BY occurred_at DESC`, organizationID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := make([]Event, 0)
	for rows.Next() {
		var event Event
		var metadata []byte
		if err := rows.Scan(&event.ID, &event.At, &event.ActorUserID, &event.OrganizationID, &event.Action, &event.ResourceType, &event.ResourceID, &event.Outcome, &event.RequestID, &event.Severity, &metadata); err != nil {
			return nil
		}
		if err := json.Unmarshal(metadata, &event.Metadata); err != nil {
			return nil
		}
		result = append(result, event)
	}
	if rows.Err() != nil {
		return nil
	}
	return result
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
