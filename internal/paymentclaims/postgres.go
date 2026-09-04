package paymentclaims

import (
	"context"
	"errors"
	"time"

	"kredit/internal/db"
	"kredit/internal/identifier"
	"kredit/internal/ledger"
	"kredit/internal/payments"
	"strings"

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
	input.SourceAccountMasked = strings.TrimSpace(input.SourceAccountMasked)
	input.TransferReference = strings.TrimSpace(input.TransferReference)
	input.EvidenceDocumentID = strings.TrimSpace(input.EvidenceDocumentID)
	if input.TransferReference == "" {
		return Claim{}, errors.New("transfer reference is required")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Claim{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Serialize retries even if a reused key names another obligation. Never
	// disclose the existing claim until its owner and full intent match.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0)),set_config('app.current_user_id',$2,true)`, "payment-claim:"+input.IdempotencyKey, input.BuyerUserID); err != nil {
		return Claim{}, err
	}
	var existing Claim
	err = tx.QueryRow(ctx, claimSelect+` WHERE idempotency_key=$1`, input.IdempotencyKey).Scan(claimFields(&existing)...)
	if err == nil {
		if !sameClaimIntent(existing, input) {
			return Claim{}, errors.New("idempotency key belongs to a different payment claim")
		}
		return existing, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Claim{}, err
	}
	var remaining ledger.Money
	var supplier, currency string
	err = tx.QueryRow(ctx, `SELECT o.outstanding_kobo,o.supplier_organization_id::text,o.currency FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id WHERE o.id=$1::uuid AND c.buyer_user_id=$2::uuid FOR UPDATE OF o`, input.ObligationID, input.BuyerUserID).Scan(&remaining, &supplier, &currency)
	if err != nil {
		return Claim{}, err
	}
	if input.AmountKobo > remaining {
		return Claim{}, errors.New("claim exceeds outstanding amount")
	}
	now := s.now()
	if input.PaidAt.IsZero() {
		input.PaidAt = now
	}
	if input.PaidAt.After(now.Add(5 * time.Minute)) {
		return Claim{}, errors.New("payment date cannot be in the future")
	}
	var claim Claim
	err = tx.QueryRow(ctx, `INSERT INTO app.payment_claims
  (id,obligation_id,buyer_user_id,supplier_organization_id,amount_kobo,currency,paid_at,source_account_masked,transfer_reference,evidence_document_id,hold_expires_at,idempotency_key)
  VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,NULLIF($8,''),$9,NULLIF($10,'')::uuid,$11,$12)
  RETURNING id::text,obligation_id::text,buyer_user_id::text,supplier_organization_id::text,amount_kobo,currency,paid_at,COALESCE(source_account_masked,''),transfer_reference,COALESCE(evidence_document_id::text,''),state,hold_expires_at,COALESCE(reviewed_by::text,''),COALESCE(review_reason,''),COALESCE(payment_id::text,''),created_at,reviewed_at`,
		identifier.New(), input.ObligationID, input.BuyerUserID, supplier, int64(input.AmountKobo), currency, input.PaidAt, input.SourceAccountMasked, input.TransferReference, input.EvidenceDocumentID, now.Add(24*time.Hour), input.IdempotencyKey).Scan(claimFields(&claim)...)
	if err != nil {
		return Claim{}, err
	}
	return claim, tx.Commit(ctx)
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
	var total ledger.Money
	if err := s.pool.QueryRow(ctx, `SELECT COALESCE(SUM(amount_kobo),0) FROM app.payment_claims WHERE obligation_id=$1::uuid AND state='pending' AND hold_expires_at>$2`, obligationID, at).Scan(&total); err != nil {
		// This interface returns a hold amount, not an error. An unavailable
		// hold check must block the entire collectible amount, never allow it.
		return ledger.Money(1<<63 - 1)
	}
	return total
}

const claimSelect = `SELECT id::text,obligation_id::text,buyer_user_id::text,supplier_organization_id::text,amount_kobo,currency,paid_at,COALESCE(source_account_masked,''),transfer_reference,COALESCE(evidence_document_id::text,''),state,hold_expires_at,COALESCE(reviewed_by::text,''),COALESCE(review_reason,''),COALESCE(payment_id::text,''),created_at,reviewed_at FROM app.payment_claims`

func claimFields(claim *Claim) []any {
	return []any{&claim.ID, &claim.ObligationID, &claim.BuyerUserID, &claim.SupplierOrganizationID, &claim.AmountKobo, &claim.Currency, &claim.PaidAt, &claim.SourceAccountMasked, &claim.TransferReference, &claim.EvidenceDocumentID, &claim.State, &claim.HoldExpiresAt, &claim.ReviewedBy, &claim.ReviewReason, &claim.PaymentID, &claim.CreatedAt, &claim.ReviewedAt}
}

func (s *PostgresStore) Confirm(ctx context.Context, id, actor, reason string, recorder payments.Service) (Claim, error) {
	pg, ok := recorder.(*payments.PostgresStore)
	if !ok || strings.TrimSpace(actor) == "" || strings.TrimSpace(reason) == "" {
		return Claim{}, errors.New("reviewer, reason, and transactional payment service are required")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Claim{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var claim Claim
	if err = tx.QueryRow(ctx, claimSelect+` WHERE id=$1::uuid`, id).Scan(claimFields(&claim)...); err != nil {
		return Claim{}, err
	}
	if err = db.SetObligationContext(ctx, tx, claim.ObligationID); err != nil {
		return Claim{}, err
	}
	// Match the common obligation-before-claim lock order used by collection.
	if _, err = tx.Exec(ctx, `SELECT id FROM app.obligations WHERE id=$1::uuid FOR UPDATE`, claim.ObligationID); err != nil {
		return Claim{}, err
	}
	if err = tx.QueryRow(ctx, claimSelect+` WHERE id=$1::uuid FOR UPDATE`, id).Scan(claimFields(&claim)...); err != nil {
		return Claim{}, err
	}
	if claim.State == Confirmed {
		return claim, tx.Commit(ctx)
	}
	if claim.State != Pending {
		return Claim{}, errors.New("payment claim is not pending")
	}
	payment, _, err := pg.RecordTx(ctx, tx, claimPaymentInput(claim, actor))
	if err != nil {
		return Claim{}, err
	}
	var reviewed time.Time
	if err = tx.QueryRow(ctx, `UPDATE app.payment_claims SET state='confirmed',reviewed_by=$2::uuid,review_reason=$3,payment_id=$4::uuid,reviewed_at=now() WHERE id=$1::uuid RETURNING reviewed_at`, id, actor, strings.TrimSpace(reason), payment.ID).Scan(&reviewed); err != nil {
		return Claim{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Claim{}, err
	}
	pg.AfterCommit(claim.ObligationID)
	claim.State = Confirmed
	claim.ReviewedBy = actor
	claim.ReviewReason = strings.TrimSpace(reason)
	claim.PaymentID = payment.ID
	claim.ReviewedAt = &reviewed
	return claim, nil
}

func (s *PostgresStore) readList(ctx context.Context, where, id string) ([]Claim, error) {
	rows, err := s.pool.Query(ctx, claimSelect+` WHERE `+where+` ORDER BY created_at DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Claim{}
	for rows.Next() {
		var claim Claim
		if err = rows.Scan(claimFields(&claim)...); err != nil {
			return nil, err
		}
		result = append(result, claim)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PostgresStore) ReadForBuyer(ctx context.Context, id string) ([]Claim, error) {
	return s.readList(ctx, `buyer_user_id=$1::uuid`, id)
}

func (s *PostgresStore) ReadForSupplier(ctx context.Context, id string) ([]Claim, error) {
	return s.readList(ctx, `supplier_organization_id=$1::uuid`, id)
}

func (s *PostgresStore) ReadForObligation(ctx context.Context, id string) ([]Claim, error) {
	return s.readList(ctx, `obligation_id=$1::uuid`, id)
}
