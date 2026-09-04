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

type UserSummary struct {
	ID                  string     `json:"id"`
	DisplayName         string     `json:"display_name"`
	Identifier          string     `json:"identifier"`
	Status              string     `json:"status"`
	OrganizationCount   int64      `json:"organization_count"`
	LastAuthenticatedAt *time.Time `json:"last_authenticated_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

type OrganizationSummary struct {
	ID              string    `json:"id"`
	LegalName       string    `json:"legal_name"`
	TradingName     string    `json:"trading_name,omitempty"`
	BusinessType    string    `json:"business_type"`
	Industry        string    `json:"industry"`
	Status          string    `json:"status"`
	MemberCount     int64     `json:"member_count"`
	OpenSales       int64     `json:"open_sales"`
	OutstandingKobo int64     `json:"outstanding_kobo"`
	Version         int64     `json:"version"`
	CreatedAt       time.Time `json:"created_at"`
}

type MoneySummary struct {
	ReceivedKobo            int64 `json:"received_kobo"`
	ReversedKobo            int64 `json:"reversed_kobo"`
	CollectionRequestedKobo int64 `json:"collection_requested_kobo"`
	CollectionSucceededKobo int64 `json:"collection_succeeded_kobo"`
	OutstandingKobo         int64 `json:"outstanding_kobo"`
	PaymentCount            int64 `json:"payment_count"`
}

type MoneyActivity struct {
	Kind           string    `json:"kind"`
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	AmountKobo     int64     `json:"amount_kobo"`
	State          string    `json:"state"`
	Reference      string    `json:"reference"`
	OccurredAt     time.Time `json:"occurred_at"`
}

type CaseSummary struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id,omitempty"`
	SubjectType    string    `json:"subject_type"`
	SubjectID      string    `json:"subject_id"`
	State          string    `json:"state"`
	BreakGlass     bool      `json:"break_glass"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type DisputeSummary struct {
	ID                    string     `json:"id"`
	ObligationID          string     `json:"obligation_id"`
	OrganizationID        string     `json:"organization_id"`
	TotalDisputedKobo     int64      `json:"total_disputed_kobo"`
	RemainingDisputedKobo int64      `json:"remaining_disputed_kobo"`
	Reason                string     `json:"reason"`
	State                 string     `json:"state"`
	OpenedAt              time.Time  `json:"opened_at"`
	ResolvedAt            *time.Time `json:"resolved_at,omitempty"`
}

type TeamMember struct {
	AssignmentID string     `json:"assignment_id"`
	UserID       string     `json:"user_id"`
	DisplayName  string     `json:"display_name"`
	Identifier   string     `json:"identifier"`
	Role         string     `json:"role"`
	GrantedAt    time.Time  `json:"granted_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
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

func (s *Store) Users(ctx context.Context, query string, limit int) ([]UserSummary, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("operations database is not configured")
	}
	query = strings.TrimSpace(query)
	rows, err := s.pool.Query(ctx, `SELECT u.id::text,COALESCE(NULLIF(u.display_name,''),'Kredit user'),COALESCE(u.normalized_email,u.normalized_phone,''),u.status,count(DISTINCT m.organization_id),u.last_authenticated_at,u.created_at FROM app.users u LEFT JOIN app.memberships m ON m.user_id=u.id AND m.status IN('active','invited','suspended') WHERE ($1='' OR u.id::text=$1 OR lower(COALESCE(u.normalized_email,''))=lower($1) OR COALESCE(u.normalized_phone,'')=$1 OR lower(COALESCE(u.display_name,'')) LIKE '%'||lower($1)||'%') GROUP BY u.id ORDER BY u.created_at DESC LIMIT $2`, query, normalizeLimit(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]UserSummary, 0)
	for rows.Next() {
		var item UserSummary
		if err := rows.Scan(&item.ID, &item.DisplayName, &item.Identifier, &item.Status, &item.OrganizationCount, &item.LastAuthenticatedAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Organizations(ctx context.Context, query string, limit int) ([]OrganizationSummary, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("operations database is not configured")
	}
	query = strings.TrimSpace(query)
	rows, err := s.pool.Query(ctx, `SELECT o.id::text,o.legal_name,COALESCE(o.trading_name,''),o.business_type,o.industry,o.status,(SELECT count(*) FROM app.memberships m WHERE m.organization_id=o.id AND m.status='active'),(SELECT count(*) FROM app.obligations ob WHERE ob.supplier_organization_id=o.id AND ob.lifecycle_status='ACTIVE'),(SELECT COALESCE(sum(ob.outstanding_kobo),0) FROM app.obligations ob WHERE ob.supplier_organization_id=o.id AND ob.lifecycle_status='ACTIVE'),o.version,o.created_at FROM app.organizations o WHERE ($1='' OR o.id::text=$1 OR lower(o.legal_name) LIKE '%'||lower($1)||'%' OR lower(COALESCE(o.trading_name,'')) LIKE '%'||lower($1)||'%') ORDER BY o.created_at DESC LIMIT $2`, query, normalizeLimit(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]OrganizationSummary, 0)
	for rows.Next() {
		var item OrganizationSummary
		if err := rows.Scan(&item.ID, &item.LegalName, &item.TradingName, &item.BusinessType, &item.Industry, &item.Status, &item.MemberCount, &item.OpenSales, &item.OutstandingKobo, &item.Version, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Money(ctx context.Context, limit int) (MoneySummary, []MoneyActivity, error) {
	if s == nil || s.pool == nil {
		return MoneySummary{}, nil, errors.New("operations database is not configured")
	}
	var summary MoneySummary
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(sum(amount_kobo) FILTER(WHERE state='recognized'),0),COALESCE(sum(amount_kobo) FILTER(WHERE state='reversed'),0),(SELECT COALESCE(sum(requested_amount_kobo),0) FROM app.collection_attempts),(SELECT COALESCE(sum(succeeded_amount_kobo),0) FROM app.collection_attempts),COALESCE((SELECT sum(outstanding_kobo) FROM app.obligations WHERE lifecycle_status='ACTIVE'),0),count(*) FROM app.payments`).Scan(&summary.ReceivedKobo, &summary.ReversedKobo, &summary.CollectionRequestedKobo, &summary.CollectionSucceededKobo, &summary.OutstandingKobo, &summary.PaymentCount)
	if err != nil {
		return MoneySummary{}, nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT kind,id,organization_id,amount_kobo,state,reference,occurred_at FROM (SELECT 'payment' kind,p.id::text id,p.supplier_organization_id::text organization_id,p.amount_kobo,p.state,COALESCE(p.provider_reference,p.id::text) reference,p.paid_at occurred_at FROM app.payments p UNION ALL SELECT 'collection',ca.id::text,o.supplier_organization_id::text,ca.requested_amount_kobo,ca.state,ca.external_reference,ca.requested_at FROM app.collection_attempts ca JOIN app.obligations o ON o.id=ca.obligation_id) activity ORDER BY occurred_at DESC LIMIT $1`, normalizeLimit(limit))
	if err != nil {
		return MoneySummary{}, nil, err
	}
	defer rows.Close()
	items := make([]MoneyActivity, 0)
	for rows.Next() {
		var item MoneyActivity
		if err := rows.Scan(&item.Kind, &item.ID, &item.OrganizationID, &item.AmountKobo, &item.State, &item.Reference, &item.OccurredAt); err != nil {
			return MoneySummary{}, nil, err
		}
		items = append(items, item)
	}
	return summary, items, rows.Err()
}

func (s *Store) Cases(ctx context.Context, state string, limit int) ([]CaseSummary, error) {
	return s.cases(ctx, state, limit, "")
}

func (s *Store) CasesForActor(ctx context.Context, state string, limit int, actorID string) ([]CaseSummary, error) {
	if strings.TrimSpace(actorID) == "" {
		return nil, errors.New("case assignment actor is required")
	}
	return s.cases(ctx, state, limit, actorID)
}

func (s *Store) cases(ctx context.Context, state string, limit int, actorID string) ([]CaseSummary, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("operations database is not configured")
	}
	state = strings.TrimSpace(state)
	rows, err := s.pool.Query(ctx, `SELECT c.id::text,COALESCE(c.organization_id::text,''),c.subject_type,c.subject_id,c.state,c.break_glass,c.created_at,c.updated_at FROM app.support_cases c WHERE ($1='' OR c.state=$1) AND ($3='' OR EXISTS(SELECT 1 FROM app.admin_review_assignments a WHERE a.kind='support' AND a.resource_id=c.id AND a.owner_id=$3::uuid)) ORDER BY c.updated_at DESC LIMIT $2`, state, normalizeLimit(limit), actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]CaseSummary, 0)
	for rows.Next() {
		var item CaseSummary
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.SubjectType, &item.SubjectID, &item.State, &item.BreakGlass, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CaseAssignedTo(ctx context.Context, caseID, actorID string) (bool, error) {
	if s == nil || s.pool == nil {
		return false, errors.New("operations database is not configured")
	}
	var allowed bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.admin_review_assignments WHERE kind='support' AND resource_id=$1::uuid AND owner_id=$2::uuid)`, caseID, actorID).Scan(&allowed)
	return allowed, err
}

func (s *Store) Disputes(ctx context.Context, state string, limit int) ([]DisputeSummary, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("operations database is not configured")
	}
	state = strings.TrimSpace(state)
	rows, err := s.pool.Query(ctx, `SELECT id::text,obligation_id::text,supplier_organization_id::text,total_disputed_kobo,remaining_disputed_kobo,reason,state,opened_at,resolved_at FROM app.disputes WHERE ($1='' OR state=$1) ORDER BY opened_at DESC LIMIT $2`, state, normalizeLimit(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]DisputeSummary, 0)
	for rows.Next() {
		var item DisputeSummary
		if err := rows.Scan(&item.ID, &item.ObligationID, &item.OrganizationID, &item.TotalDisputedKobo, &item.RemainingDisputedKobo, &item.Reason, &item.State, &item.OpenedAt, &item.ResolvedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Team(ctx context.Context) ([]TeamMember, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("operations database is not configured")
	}
	rows, err := s.pool.Query(ctx, `SELECT pra.id::text,u.id::text,COALESCE(NULLIF(u.display_name,''),'Kredit administrator'),COALESCE(u.normalized_email,u.normalized_phone,''),pra.role,pra.granted_at,pra.expires_at FROM app.platform_role_assignments pra JOIN app.users u ON u.id=pra.user_id WHERE pra.revoked_at IS NULL AND (pra.expires_at IS NULL OR pra.expires_at>now()) ORDER BY pra.granted_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]TeamMember, 0)
	for rows.Next() {
		var item TeamMember
		if err := rows.Scan(&item.AssignmentID, &item.UserID, &item.DisplayName, &item.Identifier, &item.Role, &item.GrantedAt, &item.ExpiresAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GrantRole(ctx context.Context, actorID, userID, role, reason string, expiresAt *time.Time) (TeamMember, error) {
	if s == nil || s.pool == nil {
		return TeamMember{}, errors.New("operations database is not configured")
	}
	reason = strings.TrimSpace(reason)
	if len(reason) < 8 || len(reason) > 1000 {
		return TeamMember{}, errors.New("reason must be between 8 and 1000 characters")
	}
	var item TeamMember
	err := s.pool.QueryRow(ctx, `WITH assignment AS (
		INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason,expires_at)
		VALUES($1::uuid,$2,$3::uuid,$4,$5)
		ON CONFLICT(user_id,role) WHERE revoked_at IS NULL
		DO UPDATE SET reason=EXCLUDED.reason,expires_at=EXCLUDED.expires_at
		RETURNING id,user_id,role,granted_at,expires_at
	)
	SELECT a.id::text,a.user_id::text,COALESCE(NULLIF(u.display_name,''),'Kredit administrator'),COALESCE(u.normalized_email,u.normalized_phone,''),a.role,a.granted_at,a.expires_at
	FROM assignment a
	JOIN app.users u ON u.id=a.user_id`, userID, role, actorID, reason, expiresAt).Scan(&item.AssignmentID, &item.UserID, &item.DisplayName, &item.Identifier, &item.Role, &item.GrantedAt, &item.ExpiresAt)
	return item, err
}

func (s *Store) RevokeRole(ctx context.Context, actorID, assignmentID string) error {
	if s == nil || s.pool == nil {
		return errors.New("operations database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID, role string
	if err = tx.QueryRow(ctx, `SELECT user_id::text,role FROM app.platform_role_assignments WHERE id=$1::uuid AND revoked_at IS NULL FOR UPDATE`, assignmentID).Scan(&userID, &role); err != nil {
		return errors.New("active role assignment was not found")
	}
	if userID == actorID && role == "platform_admin" {
		var count int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM app.platform_role_assignments WHERE role='platform_admin' AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at>now())`).Scan(&count); err != nil {
			return err
		}
		if count <= 1 {
			return errors.New("the last platform administrator cannot remove their own access")
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE app.platform_role_assignments SET revoked_at=now(),revoked_by=$2::uuid WHERE id=$1::uuid`, assignmentID, actorID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func normalizeLimit(limit int) int {
	if limit < 1 || limit > 200 {
		return 100
	}
	return limit
}
