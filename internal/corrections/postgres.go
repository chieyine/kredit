package corrections

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"kredit/internal/identifier"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ pool *pgxpool.Pool }

var _ Service = (*PostgresStore)(nil)

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func (s *PostgresStore) Open(organizationID, subjectType, subjectID, sourceEventID, requestedBy, reason string, evidence []string) (Request, error) {
	if organizationID == "" || subjectType == "" || subjectID == "" || requestedBy == "" || strings.TrimSpace(reason) == "" {
		return Request{}, errors.New("organization, subject, requester, and reason are required")
	}
	encodedEvidence, err := json.Marshal(evidence)
	if err != nil {
		return Request{}, err
	}
	id := identifier.New()
	row := s.pool.QueryRow(context.Background(), `INSERT INTO app.correction_requests(id,organization_id,subject_type,subject_id,source_event_id,requested_by,reason,evidence,state) VALUES($1::uuid,$2::uuid,$3,$4::uuid,NULLIF($5,''),$6::uuid,$7,$8::jsonb,'OPEN') RETURNING id::text,organization_id::text,subject_type,subject_id::text,COALESCE(source_event_id,''),requested_by::text,reason,evidence,state,created_at,updated_at`, id, organizationID, subjectType, subjectID, strings.TrimSpace(sourceEventID), requestedBy, strings.TrimSpace(reason), encodedEvidence)
	return scanRequest(row)
}

func (s *PostgresStore) StartReview(id, reviewerID string) (Request, error) {
	if id == "" || reviewerID == "" {
		return Request{}, errors.New("correction request and reviewer are required")
	}
	row := s.pool.QueryRow(context.Background(), `UPDATE app.correction_requests SET state='UNDER_REVIEW',updated_at=now() WHERE id=$1::uuid AND state='OPEN' RETURNING id::text,organization_id::text,subject_type,subject_id::text,COALESCE(source_event_id,''),requested_by::text,reason,evidence,state,created_at,updated_at`, id)
	request, err := scanRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, errors.New("correction request cannot enter review")
	}
	return request, err
}

func (s *PostgresStore) Decide(id, reviewerID, outcome, reason string) (Request, Decision, error) {
	if id == "" || reviewerID == "" || strings.TrimSpace(reason) == "" {
		return Request{}, Decision{}, errors.New("correction request, reviewer, and reason are required")
	}
	if outcome != StateApproved && outcome != StateRejected {
		return Request{}, Decision{}, errors.New("outcome must be approved or rejected")
	}
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Request{}, Decision{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	request, err := scanRequest(tx.QueryRow(ctx, `SELECT id::text,organization_id::text,subject_type,subject_id::text,COALESCE(source_event_id,''),requested_by::text,reason,evidence,state,created_at,updated_at FROM app.correction_requests WHERE id=$1::uuid FOR UPDATE`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, Decision{}, errors.New("correction request not found")
	}
	if err != nil {
		return Request{}, Decision{}, err
	}
	if request.State != StateOpen && request.State != StateReview {
		return Request{}, Decision{}, errors.New("correction request is already decided")
	}
	if request.RequestedBy == reviewerID {
		return Request{}, Decision{}, errors.New("correction requester cannot approve or reject their own request")
	}
	if _, err := tx.Exec(ctx, `UPDATE app.correction_requests SET state=$2,updated_at=now() WHERE id=$1::uuid`, id, outcome); err != nil {
		return Request{}, Decision{}, err
	}
	decision := Decision{ID: identifier.New(), RequestID: id, ReviewerID: reviewerID, Outcome: outcome, Reason: strings.TrimSpace(reason)}
	if outcome == StateApproved {
		decision.CorrectionID = "correction-event-" + identifier.New()
	}
	if err := tx.QueryRow(ctx, `INSERT INTO app.correction_decisions(id,request_id,reviewer_id,outcome,reason,correction_event_id) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,NULLIF($6,'')) RETURNING decided_at`, decision.ID, id, reviewerID, outcome, decision.Reason, decision.CorrectionID).Scan(&decision.DecidedAt); err != nil {
		return Request{}, Decision{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Request{}, Decision{}, err
	}
	request.State = outcome
	request.UpdatedAt = decision.DecidedAt
	return request, decision, nil
}

func (s *PostgresStore) Get(id string) (Request, []Decision, error) {
	request, err := scanRequest(s.pool.QueryRow(context.Background(), `SELECT id::text,organization_id::text,subject_type,subject_id::text,COALESCE(source_event_id,''),requested_by::text,reason,evidence,state,created_at,updated_at FROM app.correction_requests WHERE id=$1::uuid`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, nil, errors.New("correction request not found")
	}
	if err != nil {
		return Request{}, nil, err
	}
	decisions, err := s.decisions(id)
	return request, decisions, err
}

func (s *PostgresStore) ListForOrganization(organizationID string) []Request {
	rows, err := s.pool.Query(context.Background(), `SELECT id::text,organization_id::text,subject_type,subject_id::text,COALESCE(source_event_id,''),requested_by::text,reason,evidence,state,created_at,updated_at FROM app.correction_requests WHERE organization_id=$1::uuid ORDER BY created_at DESC`, organizationID)
	if err != nil {
		return []Request{}
	}
	defer rows.Close()
	out := []Request{}
	for rows.Next() {
		request, scanErr := scanRequest(rows)
		if scanErr != nil {
			return []Request{}
		}
		out = append(out, request)
	}
	if rows.Err() != nil {
		return []Request{}
	}
	return out
}

type correctionScanner interface{ Scan(...any) error }

func scanRequest(row correctionScanner) (Request, error) {
	var request Request
	var evidence []byte
	err := row.Scan(&request.ID, &request.OrganizationID, &request.SubjectType, &request.SubjectID, &request.SourceEventID, &request.RequestedBy, &request.Reason, &evidence, &request.State, &request.CreatedAt, &request.UpdatedAt)
	if err == nil {
		err = json.Unmarshal(evidence, &request.Evidence)
	}
	return request, err
}

func (s *PostgresStore) decisions(id string) ([]Decision, error) {
	rows, err := s.pool.Query(context.Background(), `SELECT id::text,request_id::text,reviewer_id::text,outcome,reason,COALESCE(correction_event_id,''),decided_at FROM app.correction_decisions WHERE request_id=$1::uuid ORDER BY decided_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Decision{}
	for rows.Next() {
		var decision Decision
		if err := rows.Scan(&decision.ID, &decision.RequestID, &decision.ReviewerID, &decision.Outcome, &decision.Reason, &decision.CorrectionID, &decision.DecidedAt); err != nil {
			return nil, err
		}
		out = append(out, decision)
	}
	return out, rows.Err()
}
