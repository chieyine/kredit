package paymentclaims

import (
	"context"
	"errors"
	"time"

	"kredit/internal/identifier"
	"kredit/internal/ledger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool, now: func() time.Time { return time.Now().UTC() }}
}

func (s *PostgresStore) Create(ctx context.Context, input CreateInput) (Claim, error) {
	if s == nil || s.pool == nil {
		return Claim{}, errors.New("payment claim database is unavailable")
	}
	if input.ObligationID == "" || input.BuyerUserID == "" || input.AmountKobo <= 0 || input.TransferReference == "" || input.IdempotencyKey == "" {
		return Claim{}, errors.New("obligation, buyer, positive amount, transfer reference, and idempotency key are required")
	}
	now := s.now()
	if input.PaidAt.IsZero() {
		input.PaidAt = now
	}
	if input.PaidAt.After(now.Add(5 * time.Minute)) {
		return Claim{}, errors.New("payment date cannot be in the future")
	}
	var claim Claim
	err := s.pool.QueryRow(ctx, `
		INSERT INTO app.payment_claims
		(id,obligation_id,buyer_user_id,supplier_organization_id,amount_kobo,currency,paid_at,source_account_masked,transfer_reference,evidence_document_id,hold_expires_at,idempotency_key)
		SELECT $1::uuid,o.id,$2::uuid,o.supplier_organization_id,$3,o.currency,$4,NULLIF($5,''),$6,NULLIF($7,'')::uuid,$8,$9
		FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id
		WHERE o.id=$10::uuid AND c.buyer_user_id=$2::uuid AND $3 <= o.outstanding_kobo
		ON CONFLICT (idempotency_key) DO UPDATE SET idempotency_key=EXCLUDED.idempotency_key
		RETURNING id::text,obligation_id::text,buyer_user_id::text,supplier_organization_id::text,amount_kobo,currency,paid_at,COALESCE(source_account_masked,''),transfer_reference,COALESCE(evidence_document_id::text,''),state,hold_expires_at,COALESCE(reviewed_by::text,''),COALESCE(review_reason,''),COALESCE(payment_id::text,''),created_at,reviewed_at`,
		identifier.New(), input.BuyerUserID, int64(input.AmountKobo), input.PaidAt, input.SourceAccountMasked, input.TransferReference, input.EvidenceDocumentID, now.Add(24*time.Hour), input.IdempotencyKey, input.ObligationID).Scan(claimFields(&claim)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return Claim{}, errors.New("eligible obligation was not found or claim exceeds outstanding amount")
	}
	return claim, err
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Claim, error) {
	var claim Claim
	err := s.pool.QueryRow(ctx, claimSelect+` WHERE id=$1::uuid`, id).Scan(claimFields(&claim)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return Claim{}, errors.New("payment claim not found")
	}
	return claim, err
}

func (s *PostgresStore) ListForObligation(ctx context.Context, id string) []Claim {
	return s.list(ctx, `obligation_id=$1::uuid`, id)
}
func (s *PostgresStore) ListForBuyer(ctx context.Context, id string) []Claim {
	return s.list(ctx, `buyer_user_id=$1::uuid`, id)
}
func (s *PostgresStore) ListForSupplier(ctx context.Context, id string) []Claim {
	return s.list(ctx, `supplier_organization_id=$1::uuid`, id)
}

func (s *PostgresStore) list(ctx context.Context, where, id string) []Claim {
	rows, err := s.pool.Query(ctx, claimSelect+` WHERE `+where+` ORDER BY created_at DESC`, id)
	if err != nil {
		return []Claim{}
	}
	defer rows.Close()
	result := []Claim{}
	for rows.Next() {
		var claim Claim
		if rows.Scan(claimFields(&claim)...) != nil {
			return []Claim{}
		}
		result = append(result, claim)
	}
	if rows.Err() != nil {
		return []Claim{}
	}
	return result
}

func (s *PostgresStore) Decide(ctx context.Context, id, actor, decision, reason, paymentID string) (Claim, error) {
	if decision != Confirmed && decision != Rejected {
		return Claim{}, errors.New("payment claim decision must be confirmed or rejected")
	}
	if actor == "" || reason == "" || (decision == Confirmed && paymentID == "") {
		return Claim{}, errors.New("reviewer, reason, and confirmed payment are required")
	}
	var claim Claim
	err := s.pool.QueryRow(ctx, `UPDATE app.payment_claims SET state=$2,reviewed_by=$3::uuid,review_reason=$4,payment_id=NULLIF($5,'')::uuid,reviewed_at=now() WHERE id=$1::uuid AND state='pending' RETURNING id::text,obligation_id::text,buyer_user_id::text,supplier_organization_id::text,amount_kobo,currency,paid_at,COALESCE(source_account_masked,''),transfer_reference,COALESCE(evidence_document_id::text,''),state,hold_expires_at,COALESCE(reviewed_by::text,''),COALESCE(review_reason,''),COALESCE(payment_id::text,''),created_at,reviewed_at`, id, decision, actor, reason, paymentID).Scan(claimFields(&claim)...)
	if errors.Is(err, pgx.ErrNoRows) {
		existing, getErr := s.Get(ctx, id)
		if getErr == nil && existing.State == decision {
			return existing, nil
		}
		return Claim{}, errors.New("payment claim was not found or already decided")
	}
	return claim, err
}

func (s *PostgresStore) ActiveHold(ctx context.Context, obligationID string, at time.Time) ledger.Money {
	_, _ = s.pool.Exec(ctx, `UPDATE app.payment_claims SET state='expired',reviewed_at=$2 WHERE obligation_id=$1::uuid AND state='pending' AND hold_expires_at <= $2`, obligationID, at)
	var total ledger.Money
	_ = s.pool.QueryRow(ctx, `SELECT COALESCE(SUM(amount_kobo),0) FROM app.payment_claims WHERE obligation_id=$1::uuid AND state='pending' AND hold_expires_at>$2`, obligationID, at).Scan(&total)
	return total
}

const claimSelect = `SELECT id::text,obligation_id::text,buyer_user_id::text,supplier_organization_id::text,amount_kobo,currency,paid_at,COALESCE(source_account_masked,''),transfer_reference,COALESCE(evidence_document_id::text,''),state,hold_expires_at,COALESCE(reviewed_by::text,''),COALESCE(review_reason,''),COALESCE(payment_id::text,''),created_at,reviewed_at FROM app.payment_claims`

func claimFields(claim *Claim) []any {
	return []any{&claim.ID, &claim.ObligationID, &claim.BuyerUserID, &claim.SupplierOrganizationID, &claim.AmountKobo, &claim.Currency, &claim.PaidAt, &claim.SourceAccountMasked, &claim.TransferReference, &claim.EvidenceDocumentID, &claim.State, &claim.HoldExpiresAt, &claim.ReviewedBy, &claim.ReviewReason, &claim.PaymentID, &claim.CreatedAt, &claim.ReviewedAt}
}
