package idempotency

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"kredit/internal/platform/txcleanup"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrKeyReused means an idempotency key arrived again with a different request body.
var ErrKeyReused = errors.New("idempotency key was reused for a different request")

// ErrAmbiguous preserves legacy duplicate reservations for explicit recovery.
var ErrAmbiguous = errors.New("multiple outcomes exist for this idempotency key")

// commandScope retains actor, route and method while separating mutable
// session/authorization metadata used to decide whether a response is replayable.
// Keep this definition aligned with app.idempotency_command_scope.
func commandScope(scope string) string {
	command, _, _ := strings.Cut(scope, " session:")
	return command
}

type Record struct {
	Scope        string
	Key          string
	RequestHash  string
	Status       int
	ResponseBody []byte
	CompletedAt  time.Time
	ExpiresAt    time.Time
}

// HashRequest uses a server-held key so low-entropy private fields, such as
// account numbers, cannot be recovered by guessing against a database digest.
func HashRequest(method, path string, body []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(method + "\n" + path + "\n"))
	_, _ = mac.Write(body)
	_, _ = mac.Write([]byte{'\n'})
	return hex.EncodeToString(mac.Sum(nil))
}

// Service records one result for a reservation. Complete is a one-way
// transition, not an upsert: after it succeeds, callers must use Reserve to
// recover the saved result rather than submitting a replacement completion.
type Service interface {
	Reserve(context.Context, string, string, string) (Record, bool, error)
	Complete(context.Context, string, string, int, []byte) error
}

type MemoryStore struct {
	mu      sync.Mutex
	records map[string]Record
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{records: make(map[string]Record)} }

func (s *MemoryStore) Reserve(ctx context.Context, scope, key, requestHash string) (Record, bool, error) {
	if scope == "" || key == "" || requestHash == "" {
		return Record{}, false, errors.New("idempotency scope, key, and request hash are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Record{}, false, err
	}
	index := commandScope(scope) + "\x00" + key
	if existing, ok := s.records[index]; ok {
		// A missing response or server error can follow a committed mutation.
		// Age alone is never evidence that executing that intent again is safe.
		if !existing.CompletedAt.IsZero() && existing.Status >= 200 && existing.Status < 500 && !existing.ExpiresAt.IsZero() && !time.Now().UTC().Before(existing.ExpiresAt) {
			delete(s.records, index)
		} else {
			if existing.RequestHash != requestHash {
				return Record{}, false, ErrKeyReused
			}
			existing.ResponseBody = append([]byte(nil), existing.ResponseBody...)
			return existing, true, nil
		}
	}
	record := Record{Scope: scope, Key: key, RequestHash: requestHash, ExpiresAt: time.Now().UTC().Add(24 * time.Hour)}
	s.records[index] = record
	return record, false, nil
}

func (s *MemoryStore) Complete(ctx context.Context, scope, key string, status int, body []byte) error {
	if scope == "" || key == "" {
		return errors.New("idempotency scope and key are required")
	}
	if status < 200 || status > 599 {
		return errors.New("idempotency response status is invalid")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	index := commandScope(scope) + "\x00" + key
	record, ok := s.records[index]
	if !ok || record.Scope != scope || !record.CompletedAt.IsZero() {
		return errors.New("idempotency reservation is missing or already completed")
	}
	record.Status, record.ResponseBody, record.CompletedAt = status, append([]byte(nil), body...), time.Now().UTC()
	s.records[index] = record
	return nil
}

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func (s *PostgresStore) Reserve(ctx context.Context, scope, key, requestHash string) (_ Record, _ bool, retErr error) {
	if s == nil || s.pool == nil {
		return Record{}, false, errors.New("idempotency database is not configured")
	}
	if scope == "" || key == "" || requestHash == "" {
		return Record{}, false, errors.New("idempotency scope, key, and request hash are required")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Record{}, false, err
	}
	defer txcleanup.Finish(ctx, tx, &retErr)
	// The database INSERT trigger takes the same lock, including for an older
	// binary. Separate statements obtain a fresh snapshot after a contender exits.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(app.idempotency_command_scope($1)||chr(10)||$2,211))`, scope, key); err != nil {
		return Record{}, false, err
	}
	if _, err = tx.Exec(ctx, `SELECT app.delete_expired_idempotency_record($1,$2)`, scope, key); err != nil {
		return Record{}, false, err
	}
	rows, err := tx.Query(ctx, `
		SELECT scope, idempotency_key, request_hash, COALESCE(response_status, 0), COALESCE(response_body, '{}'::jsonb), completed_at, expires_at
		FROM app.idempotency_records WHERE app.idempotency_command_scope(scope)=app.idempotency_command_scope($1) AND idempotency_key=$2 LIMIT 2`, scope, key)
	if err != nil {
		return Record{}, false, err
	}
	var record Record
	count := 0
	for rows.Next() {
		var completed *time.Time
		if err = rows.Scan(&record.Scope, &record.Key, &record.RequestHash, &record.Status, &record.ResponseBody, &completed, &record.ExpiresAt); err != nil {
			rows.Close()
			return Record{}, false, err
		}
		if completed != nil {
			record.CompletedAt = *completed
		}
		count++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return Record{}, false, err
	}
	if count > 1 {
		return Record{}, false, ErrAmbiguous
	}
	if count == 1 {
		if record.RequestHash != requestHash {
			return Record{}, false, ErrKeyReused
		}
		return record, true, tx.Commit(ctx)
	}
	err = tx.QueryRow(ctx, `INSERT INTO app.idempotency_records(scope,idempotency_key,request_hash)
		VALUES($1,$2,$3) RETURNING scope,idempotency_key,request_hash,expires_at`, scope, key, requestHash).
		Scan(&record.Scope, &record.Key, &record.RequestHash, &record.ExpiresAt)
	if err != nil {
		return Record{}, false, err
	}
	return record, false, tx.Commit(ctx)
}

func (s *PostgresStore) Complete(ctx context.Context, scope, key string, status int, body []byte) error {
	if s == nil || s.pool == nil {
		return errors.New("idempotency database is not configured")
	}
	if scope == "" || key == "" {
		return errors.New("idempotency scope and key are required")
	}
	if status < 200 || status > 599 {
		return errors.New("idempotency response status is invalid")
	}
	if !json.Valid(body) {
		body, _ = json.Marshal(map[string]any{"body": string(body)})
	}
	command, err := s.pool.Exec(ctx, `UPDATE app.idempotency_records SET response_status = $3, response_body = $4::jsonb, completed_at = NOW() WHERE scope = $1 AND idempotency_key = $2 AND completed_at IS NULL`, scope, key, status, body)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return errors.New("idempotency reservation is missing or already completed")
	}
	return nil
}
