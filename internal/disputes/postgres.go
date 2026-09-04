package disputes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"kredit/internal/db"

	"kredit/internal/identifier"
	"kredit/internal/ledger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool       *pgxpool.Pool
	invalidate func(string)
}

var _ Service = (*PostgresStore)(nil)

func NewPostgresStore(pool *pgxpool.Pool, invalidate func(string)) *PostgresStore {
	return &PostgresStore{pool: pool, invalidate: invalidate}
}

func (s *PostgresStore) Open(input OpenInput) (Dispute, error) {
	if input.ObligationID == "" || input.OpenedBy == "" || input.DisputedAmountKobo <= 0 || strings.TrimSpace(input.Reason) == "" {
		return Dispute{}, errors.New("obligation, opener, positive disputed amount, and reason are required")
	}
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Dispute{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := db.SetObligationContext(ctx, tx, input.ObligationID); err != nil {
		return Dispute{}, err
	}
	var supplierID, buyerID string
	var outstanding ledger.Money
	if err := tx.QueryRow(ctx, `SELECT o.supplier_organization_id::text,c.buyer_user_id::text,o.outstanding_kobo FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id WHERE o.id=$1::uuid FOR UPDATE OF o`, input.ObligationID).Scan(&supplierID, &buyerID, &outstanding); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Dispute{}, errors.New("obligation not found")
		}
		return Dispute{}, err
	}
	if input.DisputedAmountKobo > outstanding {
		return Dispute{}, errors.New("disputed amount exceeds outstanding")
	}
	var active bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.disputes WHERE obligation_id=$1::uuid AND state IN('OPEN','UNDER_REVIEW','PARTIALLY_RESOLVED'))`, input.ObligationID).Scan(&active); err != nil {
		return Dispute{}, err
	}
	if active {
		return Dispute{}, errors.New("an active dispute already exists for this obligation")
	}
	dispute := Dispute{ID: identifier.New(), ObligationID: input.ObligationID, SupplierOrganizationID: supplierID, BuyerUserID: buyerID, OpenedBy: input.OpenedBy, TotalDisputedKobo: input.DisputedAmountKobo, RemainingDisputedKobo: input.DisputedAmountKobo, Reason: strings.TrimSpace(input.Reason), Explanation: strings.TrimSpace(input.Explanation), State: StateOpen, CollectionEffect: EffectContestedOnly}
	if err := tx.QueryRow(ctx, `INSERT INTO app.disputes(id,obligation_id,supplier_organization_id,buyer_user_id,opened_by,total_disputed_kobo,remaining_disputed_kobo,reason,explanation,state,collection_effect) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$6,$7,NULLIF($8,''),'OPEN',$9) RETURNING opened_at`, dispute.ID, dispute.ObligationID, dispute.SupplierOrganizationID, dispute.BuyerUserID, dispute.OpenedBy, int64(dispute.TotalDisputedKobo), dispute.Reason, dispute.Explanation, dispute.CollectionEffect).Scan(&dispute.OpenedAt); err != nil {
		return Dispute{}, err
	}
	return dispute, tx.Commit(ctx)
}

func (s *PostgresStore) AddEvidence(disputeID, submittedBy, documentID, statement string) (Evidence, error) {
	if submittedBy == "" || (strings.TrimSpace(documentID) == "" && strings.TrimSpace(statement) == "") {
		return Evidence{}, errors.New("evidence submitter and document or statement are required")
	}
	evidence := Evidence{ID: identifier.New(), DisputeID: disputeID, SubmittedBy: submittedBy, DocumentID: strings.TrimSpace(documentID), Statement: strings.TrimSpace(statement)}
	err := s.pool.QueryRow(context.Background(), `INSERT INTO app.dispute_evidence(id,dispute_id,submitted_by,document_id,statement) VALUES($1::uuid,$2::uuid,$3::uuid,NULLIF($4,'')::uuid,NULLIF($5,'')) RETURNING submitted_at`, evidence.ID, evidence.DisputeID, evidence.SubmittedBy, evidence.DocumentID, evidence.Statement).Scan(&evidence.SubmittedAt)
	if err != nil {
		return Evidence{}, err
	}
	return evidence, nil
}

func (s *PostgresStore) Respond(disputeID, actor, response string) (Evidence, error) {
	return s.AddEvidence(disputeID, actor, "", response)
}

func (s *PostgresStore) Decide(input DecideInput) (Dispute, Decision, error) {
	if input.ReviewerID == "" || strings.TrimSpace(input.Outcome) == "" || strings.TrimSpace(input.Reason) == "" {
		return Dispute{}, Decision{}, errors.New("reviewer, outcome, and reason are required")
	}
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Dispute{}, Decision{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	dispute, err := scanDispute(tx.QueryRow(ctx, `SELECT id::text,obligation_id::text,supplier_organization_id::text,buyer_user_id::text,opened_by::text,total_disputed_kobo,remaining_disputed_kobo,reason,COALESCE(explanation,''),state,collection_effect,COALESCE(assigned_reviewer::text,''),opened_at,resolved_at FROM app.disputes WHERE id=$1::uuid FOR UPDATE`, input.DisputeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Dispute{}, Decision{}, errors.New("dispute not found")
	}
	if err != nil {
		return Dispute{}, Decision{}, err
	}
	if dispute.State == StateResolved || dispute.State == StateWithdrawn {
		return Dispute{}, Decision{}, errors.New("dispute is already closed")
	}
	if input.AdjustmentKobo < 0 || input.AdjustmentKobo > dispute.RemainingDisputedKobo {
		return Dispute{}, Decision{}, errors.New("invalid adjustment amount")
	}
	if input.RemainingDisputedKobo < 0 || input.RemainingDisputedKobo > dispute.RemainingDisputedKobo {
		return Dispute{}, Decision{}, errors.New("invalid remaining disputed amount")
	}
	if input.AdjustmentKobo > dispute.RemainingDisputedKobo-input.RemainingDisputedKobo {
		return Dispute{}, Decision{}, errors.New("adjusted principal must be removed from the remaining dispute")
	}
	if input.ValidPrincipalKobo < 0 {
		return Dispute{}, Decision{}, errors.New("valid principal cannot be negative")
	}
	if input.AdjustmentKobo > 0 {
		if err := applyDisputeAdjustmentTx(ctx, tx, dispute, input.AdjustmentKobo); err != nil {
			return Dispute{}, Decision{}, err
		}
	}
	decision := Decision{ID: identifier.New(), DisputeID: dispute.ID, ReviewerID: input.ReviewerID, Outcome: input.Outcome, ValidPrincipalKobo: input.ValidPrincipalKobo, AdjustmentKobo: input.AdjustmentKobo, RemainingDisputedKobo: input.RemainingDisputedKobo, Reason: strings.TrimSpace(input.Reason)}
	if err := tx.QueryRow(ctx, `INSERT INTO app.dispute_decisions(id,dispute_id,reviewer_id,outcome,valid_principal_kobo,adjustment_kobo,remaining_disputed_kobo,reason) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$8) RETURNING decided_at`, decision.ID, decision.DisputeID, decision.ReviewerID, decision.Outcome, int64(decision.ValidPrincipalKobo), int64(decision.AdjustmentKobo), int64(decision.RemainingDisputedKobo), decision.Reason).Scan(&decision.DecidedAt); err != nil {
		return Dispute{}, Decision{}, err
	}
	dispute.RemainingDisputedKobo = input.RemainingDisputedKobo
	dispute.State = StatePartiallyResolved
	var resolved any
	if input.RemainingDisputedKobo == 0 {
		dispute.State = StateResolved
		dispute.ResolvedAt = time.Now().UTC()
		resolved = dispute.ResolvedAt
	} else {
		resolved = nil
	}
	if _, err := tx.Exec(ctx, `UPDATE app.disputes SET remaining_disputed_kobo=$2,state=$3,resolved_at=$4 WHERE id=$1::uuid`, dispute.ID, int64(dispute.RemainingDisputedKobo), dispute.State, resolved); err != nil {
		return Dispute{}, Decision{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Dispute{}, Decision{}, err
	}
	if s.invalidate != nil {
		s.invalidate(dispute.ObligationID)
	}
	return dispute, decision, nil
}

func applyDisputeAdjustmentTx(ctx context.Context, tx pgx.Tx, dispute Dispute, amount ledger.Money) error {
	if err := db.SetObligationContext(ctx, tx, dispute.ObligationID); err != nil {
		return err
	}
	var requestID string
	var outstanding, principal ledger.Money
	if err := tx.QueryRow(ctx, `SELECT credit_request_id::text,outstanding_kobo,principal_kobo FROM app.obligations WHERE id=$1::uuid FOR UPDATE`, dispute.ObligationID).Scan(&requestID, &outstanding, &principal); err != nil {
		return err
	}
	if amount > outstanding {
		return errors.New("adjustment exceeds outstanding")
	}
	if err := db.GuardUnreservedReduction(ctx, tx, dispute.ObligationID, int64(outstanding-amount)); err != nil {
		return err
	}
	key := "dispute-adjustment:" + dispute.ID + ":" + fmt.Sprint(dispute.RemainingDisputedKobo)
	if err := postDisputeLedgerTx(ctx, tx, dispute.ID, key, amount); err != nil {
		return err
	}
	if err := db.ReduceSchedulePrincipalTx(ctx, tx, dispute.ObligationID, outstanding, amount, true); err != nil {
		return err
	}
	newOutstanding := outstanding - amount
	var status string
	switch newOutstanding {
	case 0:
		status = "PAID"
	case principal:
		status = "UNPAID"
	default:
		status = "PARTIALLY_PAID"
	}
	if _, err := tx.Exec(ctx, `UPDATE app.obligations SET outstanding_kobo=$2,payment_status=$3 WHERE id=$1::uuid`, dispute.ObligationID, int64(newOutstanding), status); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE app.credit_requests SET version=version+1,updated_at=now() WHERE id=$1::uuid`, requestID); err != nil {
		return err
	}
	command, err := tx.Exec(ctx, `UPDATE app.credit_aggregate_snapshots SET aggregate=jsonb_set(jsonb_set(jsonb_set(aggregate,'{obligation,outstanding_kobo}',to_jsonb($2::bigint),false),'{obligation,payment_status}',to_jsonb($3::text),false),'{request,version}',to_jsonb(version+1),false),version=version+1,updated_at=now() WHERE credit_request_id=$1`, requestID, int64(newOutstanding), status)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return errors.New("credit aggregate snapshot not found")
	}
	return nil
}

func postDisputeLedgerTx(ctx context.Context, tx pgx.Tx, referenceID, key string, amount ledger.Money) error {
	var id string
	err := tx.QueryRow(ctx, `INSERT INTO ledger.transactions(event_type,reference_type,reference_id,idempotency_key,effective_at) VALUES('dispute_adjustment','dispute',$1,$2,now()) ON CONFLICT(idempotency_key) DO NOTHING RETURNING id::text`, referenceID, key).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, p := range []struct {
		account       string
		debit, credit int64
	}{{ledger.AccountReturnsAdjustment, int64(amount), 0}, {ledger.AccountTradeReceivable, 0, int64(amount)}} {
		if _, err := tx.Exec(ctx, `INSERT INTO ledger.postings(transaction_id,account_id,debit_kobo,credit_kobo) SELECT $1::uuid,id,$3,$4 FROM ledger.accounts WHERE code=$2`, id, p.account, p.debit, p.credit); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresStore) BlockedAmount(obligationID string) (ledger.Money, error) {
	var outstanding, total ledger.Money
	err := s.pool.QueryRow(context.Background(), `SELECT o.outstanding_kobo,COALESCE(SUM(CASE WHEN d.collection_effect='FULL_BLOCK' THEN o.outstanding_kobo WHEN d.collection_effect='CONTESTED_ONLY' THEN d.remaining_disputed_kobo ELSE 0 END),0) FROM app.obligations o LEFT JOIN app.disputes d ON d.obligation_id=o.id AND d.state IN('OPEN','UNDER_REVIEW','PARTIALLY_RESOLVED') WHERE o.id=$1::uuid GROUP BY o.outstanding_kobo`, obligationID).Scan(&outstanding, &total)
	if total > outstanding {
		total = outstanding
	}
	return total, err
}

func (s *PostgresStore) Get(id string) (Dispute, []Evidence, []Decision, error) {
	d, err := scanDispute(s.pool.QueryRow(context.Background(), `SELECT id::text,obligation_id::text,supplier_organization_id::text,buyer_user_id::text,opened_by::text,total_disputed_kobo,remaining_disputed_kobo,reason,COALESCE(explanation,''),state,collection_effect,COALESCE(assigned_reviewer::text,''),opened_at,resolved_at FROM app.disputes WHERE id=$1::uuid`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Dispute{}, nil, nil, errors.New("dispute not found")
	}
	if err != nil {
		return Dispute{}, nil, nil, err
	}
	e, err := s.evidence(id)
	if err != nil {
		return Dispute{}, nil, nil, err
	}
	ds, err := s.decisions(id)
	if err != nil {
		return Dispute{}, nil, nil, err
	}
	return d, e, ds, nil
}
func (s *PostgresStore) ListForOrganization(id string) []Dispute {
	return s.list(`supplier_organization_id=$1::uuid`, id)
}
func (s *PostgresStore) ListForObligation(id string) []Dispute {
	return s.list(`obligation_id=$1::uuid`, id)
}
func (s *PostgresStore) ListForBuyer(id string) []Dispute {
	return s.list(`buyer_user_id=$1::uuid`, id)
}

type disputeScanner interface{ Scan(...any) error }

func scanDispute(r disputeScanner) (Dispute, error) {
	var d Dispute
	var resolved *time.Time
	err := r.Scan(&d.ID, &d.ObligationID, &d.SupplierOrganizationID, &d.BuyerUserID, &d.OpenedBy, &d.TotalDisputedKobo, &d.RemainingDisputedKobo, &d.Reason, &d.Explanation, &d.State, &d.CollectionEffect, &d.AssignedReviewer, &d.OpenedAt, &resolved)
	if resolved != nil {
		d.ResolvedAt = *resolved
	}
	return d, err
}
func (s *PostgresStore) list(where, id string) []Dispute {
	rows, err := s.pool.Query(context.Background(), `SELECT id::text,obligation_id::text,supplier_organization_id::text,buyer_user_id::text,opened_by::text,total_disputed_kobo,remaining_disputed_kobo,reason,COALESCE(explanation,''),state,collection_effect,COALESCE(assigned_reviewer::text,''),opened_at,resolved_at FROM app.disputes WHERE `+where+` ORDER BY opened_at DESC`, id)
	if err != nil {
		return []Dispute{}
	}
	defer rows.Close()
	out := []Dispute{}
	for rows.Next() {
		d, e := scanDispute(rows)
		if e != nil {
			return []Dispute{}
		}
		out = append(out, d)
	}
	if rows.Err() != nil {
		return []Dispute{}
	}
	return out
}
func (s *PostgresStore) evidence(id string) ([]Evidence, error) {
	rows, err := s.pool.Query(context.Background(), `SELECT id::text,dispute_id::text,submitted_by::text,COALESCE(document_id::text,''),COALESCE(statement,''),submitted_at FROM app.dispute_evidence WHERE dispute_id=$1::uuid ORDER BY submitted_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Evidence{}
	for rows.Next() {
		var e Evidence
		if err := rows.Scan(&e.ID, &e.DisputeID, &e.SubmittedBy, &e.DocumentID, &e.Statement, &e.SubmittedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
func (s *PostgresStore) decisions(id string) ([]Decision, error) {
	rows, err := s.pool.Query(context.Background(), `SELECT id::text,dispute_id::text,reviewer_id::text,outcome,valid_principal_kobo,adjustment_kobo,remaining_disputed_kobo,reason,decided_at FROM app.dispute_decisions WHERE dispute_id=$1::uuid ORDER BY decided_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Decision{}
	for rows.Next() {
		var d Decision
		if err := rows.Scan(&d.ID, &d.DisputeID, &d.ReviewerID, &d.Outcome, &d.ValidPrincipalKobo, &d.AdjustmentKobo, &d.RemainingDisputedKobo, &d.Reason, &d.DecidedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *PostgresStore) readList(where, id string) ([]Dispute, error) {
	rows, err := s.pool.Query(context.Background(), `SELECT id::text,obligation_id::text,supplier_organization_id::text,buyer_user_id::text,opened_by::text,total_disputed_kobo,remaining_disputed_kobo,reason,COALESCE(explanation,''),state,collection_effect,COALESCE(assigned_reviewer::text,''),opened_at,resolved_at FROM app.disputes WHERE `+where+` ORDER BY opened_at DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Dispute{}
	for rows.Next() {
		d, e := scanDispute(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
func (s *PostgresStore) ReadForOrganization(id string) ([]Dispute, error) {
	return s.readList(`supplier_organization_id=$1::uuid`, id)
}

func (s *PostgresStore) ReadForObligation(id string) ([]Dispute, error) {
	return s.readList(`obligation_id=$1::uuid`, id)
}

func (s *PostgresStore) ReadForBuyer(id string) ([]Dispute, error) {
	return s.readList(`buyer_user_id=$1::uuid`, id)
}
