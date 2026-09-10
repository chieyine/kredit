package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"kredit/internal/platform/logging"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	ID                  string
	AggregateType       string
	AggregateID         string
	EventType           string
	Payload             json.RawMessage
	IdempotencyKey      string
	State               string
	Attempts            int
	AvailableAt         time.Time
	LastError           string
	CreatedAt           time.Time
	PublishedAt         *time.Time
	ProcessingStartedAt *time.Time
}

// Store is deliberately transaction-first: domain code appends an event with
// the same pgx transaction that changes the aggregate, then a worker publishes
// it asynchronously. This removes the dual-write failure mode.
type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) AppendTx(ctx context.Context, tx pgx.Tx, event Event) (string, error) {
	if tx == nil {
		return "", errors.New("outbox transaction is required")
	}
	if event.AggregateType == "" || event.AggregateID == "" || event.EventType == "" || event.IdempotencyKey == "" || !json.Valid(event.Payload) {
		return "", errors.New("outbox aggregate, event, key, and valid payload are required")
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO app.outbox_events (aggregate_type, aggregate_id, event_type, payload, idempotency_key)
		VALUES ($1,$2,$3,$4::jsonb,$5)
		ON CONFLICT (idempotency_key) DO UPDATE SET id = app.outbox_events.id
		WHERE app.outbox_events.aggregate_type=EXCLUDED.aggregate_type
		  AND app.outbox_events.aggregate_id=EXCLUDED.aggregate_id
		  AND app.outbox_events.event_type=EXCLUDED.event_type
		  AND app.outbox_events.payload=EXCLUDED.payload
		RETURNING id::text`, event.AggregateType, event.AggregateID, event.EventType, event.Payload, event.IdempotencyKey).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("outbox idempotency key was reused for a different event")
	}
	if err != nil {
		return "", fmt.Errorf("append outbox event: %w", err)
	}
	return id, nil
}

func (s *Store) Claim(ctx context.Context, limit int) ([]Event, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("outbox database is not configured")
	}
	if limit <= 0 || limit > 500 {
		return nil, errors.New("outbox claim limit must be between 1 and 500")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Recover events whose publisher died after claiming them. The lease is
	// intentionally short relative to normal provider timeouts and is safe to
	// reprocess because consumers must honor IdempotencyKey.
	if _, err := tx.Exec(ctx, `UPDATE app.outbox_events SET state = 'failed', available_at = now(), last_error = COALESCE(last_error, 'processing lease expired'), processing_started_at = NULL WHERE state = 'processing' AND processing_started_at < now() - interval '10 minutes'`); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		SELECT id::text, aggregate_type, aggregate_id, event_type, payload, idempotency_key, state, attempts, available_at, COALESCE(last_error,''), created_at, published_at, processing_started_at
		FROM app.outbox_events
		WHERE state IN ('pending','failed') AND available_at <= now()
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	events := make([]Event, 0, limit)
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.AggregateType, &event.AggregateID, &event.EventType, &event.Payload, &event.IdempotencyKey, &event.State, &event.Attempts, &event.AvailableAt, &event.LastError, &event.CreatedAt, &event.PublishedAt, &event.ProcessingStartedAt); err != nil {
			rows.Close()
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for index := range events {
		event := &events[index]
		event.State = "processing"
		if err := tx.QueryRow(ctx, `UPDATE app.outbox_events SET state = 'processing', attempts = attempts + 1, processing_started_at = now() WHERE id = $1::uuid RETURNING attempts, processing_started_at`, event.ID).Scan(&event.Attempts, &event.ProcessingStartedAt); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return events, nil
}

// Completion is fenced to the exact claim. A slow publisher must not overwrite
// an event that another worker reclaimed after its processing lease expired.
var ErrClaimLost = errors.New("outbox processing claim is no longer current")

func (s *Store) MarkPublished(ctx context.Context, event Event) error {
	if s == nil || s.pool == nil || event.ID == "" || event.ProcessingStartedAt == nil {
		return errors.New("outbox database and processing claim are required")
	}
	command, err := s.pool.Exec(ctx, `UPDATE app.outbox_events SET state='published',published_at=now(),last_error=NULL,processing_started_at=NULL WHERE id=$1::uuid AND state='processing' AND attempts=$2 AND processing_started_at=$3`, event.ID, event.Attempts, event.ProcessingStartedAt)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}

func (s *Store) MarkFailed(ctx context.Context, event Event, reason string, retryAt time.Time) error {
	if s == nil || s.pool == nil || event.ID == "" || event.ProcessingStartedAt == nil {
		return errors.New("outbox database and processing claim are required")
	}
	command, err := s.pool.Exec(ctx, `UPDATE app.outbox_events SET state='failed',available_at=$2,last_error=$3,processing_started_at=NULL WHERE id=$1::uuid AND state='processing' AND attempts=$4 AND processing_started_at=$5`, event.ID, retryAt, logging.SafeError(errors.New(reason)), event.Attempts, event.ProcessingStartedAt)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}
