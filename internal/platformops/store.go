package platformops

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

type Overview struct {
	QueuedJobs       int64 `json:"queued_jobs"`
	FailedJobs       int64 `json:"failed_jobs"`
	DeadLetterJobs   int64 `json:"dead_letter_jobs"`
	PendingOutbox    int64 `json:"pending_outbox"`
	FailedOutbox     int64 `json:"failed_outbox"`
	ProviderFailures int64 `json:"provider_failures"`
	OpenCases        int64 `json:"open_cases"`
	OpenDisputes     int64 `json:"open_disputes"`
}

type Job struct {
	ID          int64      `json:"id"`
	Kind        string     `json:"kind"`
	Queue       string     `json:"queue"`
	State       string     `json:"state"`
	Attempt     int        `json:"attempt"`
	MaxAttempts int        `json:"max_attempts"`
	ScheduledAt time.Time  `json:"scheduled_at"`
	AttemptedAt *time.Time `json:"attempted_at,omitempty"`
	FinalizedAt *time.Time `json:"finalized_at,omitempty"`
}

type ProviderEvent struct {
	Provider    string     `json:"provider"`
	EventID     string     `json:"event_id"`
	EventType   string     `json:"event_type"`
	State       string     `json:"state"`
	Attempts    int        `json:"attempts"`
	LastError   string     `json:"last_error,omitempty"`
	ReceivedAt  time.Time  `json:"received_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

type SearchResult struct {
	Type           string `json:"type"`
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id,omitempty"`
	State          string `json:"state"`
	Reference      string `json:"reference"`
}

type AuditEvent struct {
	ID             string            `json:"id"`
	OccurredAt     time.Time         `json:"occurred_at"`
	ActorUserID    string            `json:"actor_user_id,omitempty"`
	OrganizationID string            `json:"organization_id,omitempty"`
	Action         string            `json:"action"`
	ResourceType   string            `json:"resource_type"`
	ResourceID     string            `json:"resource_id,omitempty"`
	Outcome        string            `json:"outcome"`
	Severity       string            `json:"severity"`
	RequestID      string            `json:"request_id,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

func (s *Store) Overview(ctx context.Context) (Overview, error) {
	if s == nil || s.pool == nil {
		return Overview{}, errors.New("operations database is not configured")
	}
	var v Overview
	err := s.pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM jobs.river_job WHERE state IN ('available','pending','retryable','running','scheduled')),
		(SELECT count(*) FROM jobs.river_job WHERE state='retryable'),
		(SELECT count(*) FROM app.job_dead_letters),
		(SELECT count(*) FROM app.outbox_events WHERE state IN ('pending','processing')),
		(SELECT count(*) FROM app.outbox_events WHERE state='failed'),
		(SELECT count(*) FROM app.provider_webhook_inbox WHERE state='failed'),
		(SELECT count(*) FROM app.support_cases WHERE state IN ('OPEN','IN_PROGRESS')),
		(SELECT count(*) FROM app.disputes WHERE state IN ('OPEN','UNDER_REVIEW','PARTIALLY_RESOLVED'))`).
		Scan(&v.QueuedJobs, &v.FailedJobs, &v.DeadLetterJobs, &v.PendingOutbox, &v.FailedOutbox, &v.ProviderFailures, &v.OpenCases, &v.OpenDisputes)
	return v, err
}

func (s *Store) Jobs(ctx context.Context, limit int) ([]Job, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("operations database is not configured")
	}
	limit = normalizeLimit(limit)
	rows, err := s.pool.Query(ctx, `SELECT id,kind,queue,state,attempt,max_attempts,scheduled_at,attempted_at,finalized_at FROM jobs.river_job ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Job, 0)
	for rows.Next() {
		var item Job
		if err := rows.Scan(&item.ID, &item.Kind, &item.Queue, &item.State, &item.Attempt, &item.MaxAttempts, &item.ScheduledAt, &item.AttemptedAt, &item.FinalizedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ProviderEvents(ctx context.Context, limit int) ([]ProviderEvent, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("operations database is not configured")
	}
	rows, err := s.pool.Query(ctx, `SELECT provider,event_id,event_type,state,attempts,COALESCE(last_error,''),received_at,processed_at FROM app.provider_webhook_inbox ORDER BY received_at DESC LIMIT $1`, normalizeLimit(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ProviderEvent, 0)
	for rows.Next() {
		var item ProviderEvent
		if err := rows.Scan(&item.Provider, &item.EventID, &item.EventType, &item.State, &item.Attempts, &item.LastError, &item.ReceivedAt, &item.ProcessedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Search(ctx context.Context, query string) ([]SearchResult, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("operations database is not configured")
	}
	query = strings.TrimSpace(query)
	if len(query) < 4 || len(query) > 128 {
		return nil, errors.New("search reference must be between 4 and 128 characters")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT kind,id,organization_id,state,reference FROM (
			SELECT 'credit_request' kind,cr.id::text id,cr.supplier_organization_id::text organization_id,cr.state,cr.id::text reference FROM app.credit_requests cr
			UNION ALL SELECT 'payment',p.id::text,o.supplier_organization_id::text,p.state,COALESCE(p.provider_reference,p.id::text) FROM app.payments p JOIN app.obligations o ON o.id=p.obligation_id
			UNION ALL SELECT 'collection',ca.id::text,o.supplier_organization_id::text,ca.state,ca.external_reference FROM app.collection_attempts ca JOIN app.obligations o ON o.id=ca.obligation_id
			UNION ALL SELECT 'support_case',sc.id::text,COALESCE(sc.organization_id::text,''),sc.state,sc.id::text FROM app.support_cases sc
			UNION ALL SELECT 'dispute',d.id::text,d.supplier_organization_id::text,d.state,d.id::text FROM app.disputes d
		) records WHERE lower(reference)=lower($1) OR lower(id::text)=lower($1) LIMIT 25`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]SearchResult, 0)
	for rows.Next() {
		var item SearchResult
		if err := rows.Scan(&item.Type, &item.ID, &item.OrganizationID, &item.State, &item.Reference); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Audit(ctx context.Context, organizationID string, limit int) ([]AuditEvent, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("operations database is not configured")
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text,occurred_at,COALESCE(actor_user_id::text,''),COALESCE(organization_id::text,''),action,resource_type,COALESCE(resource_id,''),outcome,severity,COALESCE(request_id,''),metadata FROM app.audit_events WHERE ($1='' OR organization_id=$1::uuid) ORDER BY occurred_at DESC LIMIT $2`, organizationID, normalizeLimit(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AuditEvent, 0)
	for rows.Next() {
		var item AuditEvent
		if err := rows.Scan(&item.ID, &item.OccurredAt, &item.ActorUserID, &item.OrganizationID, &item.Action, &item.ResourceType, &item.ResourceID, &item.Outcome, &item.Severity, &item.RequestID, &item.Metadata); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func normalizeLimit(limit int) int {
	if limit < 1 || limit > 200 {
		return 100
	}
	return limit
}
