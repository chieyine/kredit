package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Record struct {
	Scope        string
	Key          string
	RequestHash  string
	Status       int
	ResponseBody []byte
	CompletedAt  time.Time
	ExpiresAt    time.Time
}

func HashRequest(method, path string, body []byte) string {
	digest := sha256.Sum256(append(append([]byte(method+"\n"+path+"\n"), body...), '\n'))
	return hex.EncodeToString(digest[:])
}

type Service interface {
	Reserve(context.Context, string, string, string) (Record, bool, error)
	Complete(context.Context, string, string, int, []byte) error
}

type MemoryStore struct {
	mu      sync.Mutex
	records map[string]Record
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{records: make(map[string]Record)} }

func (s *MemoryStore) Reserve(_ context.Context, scope, key, requestHash string) (Record, bool, error) {
	if scope == "" || key == "" || requestHash == "" {
		return Record{}, false, errors.New("idempotency scope, key, and request hash are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index := scope + "\x00" + key
	if existing, ok := s.records[index]; ok {
		if !existing.CompletedAt.IsZero() && !existing.ExpiresAt.IsZero() && !time.Now().UTC().Before(existing.ExpiresAt) {
			delete(s.records, index)
		} else {
			if existing.RequestHash != requestHash {
				return Record{}, false, errors.New("idempotency key was reused for a different request")
			}
			return existing, true, nil
		}
	}
	record := Record{Scope: scope, Key: key, RequestHash: requestHash, ExpiresAt: time.Now().UTC().Add(24 * time.Hour)}
	s.records[index] = record
	return record, false, nil
}

func (s *MemoryStore) Complete(_ context.Context, scope, key string, status int, body []byte) error {
	if scope == "" || key == "" {
		return errors.New("idempotency scope and key are required")
	}
	if status < 100 || status > 599 {
		return errors.New("idempotency response status is invalid")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index := scope + "\x00" + key
	record, ok := s.records[index]
	if !ok {
		return errors.New("idempotency reservation not found")
	}
	record.Status, record.ResponseBody, record.CompletedAt = status, append([]byte(nil), body...), time.Now().UTC()
	s.records[index] = record
	return nil
}

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func (s *PostgresStore) Reserve(ctx context.Context, scope, key, requestHash string) (Record, bool, error) {
	if s == nil || s.pool == nil {
		return Record{}, false, errors.New("idempotency database is not configured")
	}
	if scope == "" || key == "" || requestHash == "" {
		return Record{}, false, errors.New("idempotency scope, key, and request hash are required")
	}
	var record Record
	var response []byte
	var completed *time.Time
	if _, err := s.pool.Exec(ctx, `SELECT app.delete_expired_idempotency_record($1, $2)`, scope, key); err != nil {
		return Record{}, false, err
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO app.idempotency_records (scope, idempotency_key, request_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (scope, idempotency_key) DO NOTHING
		RETURNING scope, idempotency_key, request_hash, COALESCE(response_status, 0), COALESCE(response_body, '{}'::jsonb), completed_at, expires_at`, scope, key, requestHash).
		Scan(&record.Scope, &record.Key, &record.RequestHash, &record.Status, &response, &completed, &record.ExpiresAt)
	if err == nil {
		record.ResponseBody = response
		if completed != nil {
			record.CompletedAt = *completed
		}
		return record, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Record{}, false, err
	}
	// A conflict means another request already reserved this key. Read its
	// immutable request hash and snapshot so callers can replay or reject it.
	err = s.pool.QueryRow(ctx, `
		SELECT scope, idempotency_key, request_hash, COALESCE(response_status, 0), COALESCE(response_body, '{}'::jsonb), completed_at, expires_at
		FROM app.idempotency_records WHERE scope = $1 AND idempotency_key = $2`, scope, key).
		Scan(&record.Scope, &record.Key, &record.RequestHash, &record.Status, &response, &completed, &record.ExpiresAt)
	if err != nil {
		return Record{}, false, err
	}
	if record.RequestHash != requestHash {
		return Record{}, false, errors.New("idempotency key was reused for a different request")
	}
	record.ResponseBody = response
	if completed != nil {
		record.CompletedAt = *completed
	}
	return record, true, nil
}

func (s *PostgresStore) Complete(ctx context.Context, scope, key string, status int, body []byte) error {
	if s == nil || s.pool == nil {
		return errors.New("idempotency database is not configured")
	}
	if scope == "" || key == "" {
		return errors.New("idempotency scope and key are required")
	}
	if status < 100 || status > 599 {
		return errors.New("idempotency response status is invalid")
	}
	if !json.Valid(body) {
		body, _ = json.Marshal(map[string]any{"body": string(body)})
	}
	command, err := s.pool.Exec(ctx, `UPDATE app.idempotency_records SET response_status = $3, response_body = $4::jsonb, completed_at = NOW() WHERE scope = $1 AND idempotency_key = $2`, scope, key, status, body)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return errors.New("idempotency reservation not found")
	}
	return nil
}
