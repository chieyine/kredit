package operations

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"kredit/internal/businesspolicy"
	"kredit/internal/db"

	"kredit/internal/identifier"
	"kredit/internal/ledger"
	"kredit/internal/outbox"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool       *pgxpool.Pool
	outbox     *outbox.Store
	invalidate func(string)
}

var _ Service = (*PostgresStore)(nil)
var _ IdempotentService = (*PostgresStore)(nil)

func NewPostgresStore(pool *pgxpool.Pool, events *outbox.Store, invalidate func(string)) *PostgresStore {
	return &PostgresStore{pool: pool, outbox: events, invalidate: invalidate}
}
func (s *PostgresStore) WriteOff(actor, org, obligation string, amount ledger.Money, reason, approvedBy string) (Action, error) {
	return s.WriteOffWithKey(actor, org, obligation, amount, reason, approvedBy, identifier.New())
}
func (s *PostgresStore) WaiveFee(actor, org, obligation string, amount ledger.Money, reason, approvedBy string) (Action, error) {
	return s.WaiveFeeWithKey(actor, org, obligation, amount, reason, approvedBy, identifier.New())
}
func (s *PostgresStore) WriteOffWithKey(actor, org, obligation string, amount ledger.Money, reason, approvedBy, key string) (Action, error) {
	return s.adjust(actor, org, obligation, amount, reason, approvedBy, "write_off", key)
}
func (s *PostgresStore) WaiveFeeWithKey(actor, org, obligation string, amount ledger.Money, reason, approvedBy, key string) (Action, error) {
	return s.adjust(actor, org, obligation, amount, reason, approvedBy, "fee_waiver", key)
}

func (s *PostgresStore) adjust(actor, org, obligation string, amount ledger.Money, reason, approvedBy, kind, key string) (Action, error) {
	if actor == "" || org == "" || obligation == "" || amount <= 0 || strings.TrimSpace(reason) == "" || strings.TrimSpace(key) == "" {
		return Action{}, errors.New("actor, organisation, obligation, positive amount, reason, and idempotency key are required")
	}
	if approvedBy == actor && approvedBy != "" {
		return Action{}, errors.New("approval must be performed by a different user")
	}

	if strings.TrimSpace(approvedBy) != "" {
		return Action{}, errors.New("a caller-supplied approver is not evidence of approval")
	}
	actionID := identifier.FromKey("operation:"+kind+":"+obligation, key)
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Action{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	action, err := s.adjustTx(ctx, tx, actor, org, obligation, amount, reason, approvedBy, kind, actionID, false)
	if err != nil {
		return Action{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Action{}, err
	}
	if kind == "write_off" && s.invalidate != nil {
		s.invalidate(obligation)
	}
	return action, nil
}

// verified is reachable only through the stored, independent approval transaction.
func (s *PostgresStore) adjustTx(ctx context.Context, tx pgx.Tx, actor, org, obligation string, amount ledger.Money, reason, approvedBy, kind, actionID string, verified bool) (Action, error) {
	if err := db.SetObligationContext(ctx, tx, obligation); err != nil {
		return Action{}, err
	}
	var requestID, supplierID string
	var outstanding, principal, baseFee ledger.Money
	if err := tx.QueryRow(ctx, `SELECT o.credit_request_id::text,o.supplier_organization_id::text,o.outstanding_kobo,o.principal_kobo,o.base_fee_kobo FROM app.obligations o WHERE o.id=$1::uuid FOR UPDATE`, obligation).Scan(&requestID, &supplierID, &outstanding, &principal, &baseFee); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Action{}, errors.New("obligation not found")
		}
		return Action{}, err
	}
	if supplierID != org {
		return Action{}, errors.New("obligation does not belong to organisation")
	}
	if existing, found, err := loadOperationTx(ctx, tx, actionID); err != nil {
		return Action{}, err
	} else if found {
		if existing.ActorUserID != actor || existing.OrganizationID != org || existing.AmountKobo != amount || existing.Reason != strings.TrimSpace(reason) || existing.ApprovedBy != approvedBy {
			return Action{}, errors.New("idempotency key was reused for a different operation")
		}
		return existing, nil
	}
	policy, err := businesspolicy.ReadTx(ctx, tx)
	if err != nil {
		return Action{}, err
	}
	threshold := min(highValueThreshold, ledger.Money(policy.Values.CorrectionThreshold))
	if !verified && amount >= threshold {
		return Action{}, errors.New("operation reaches the configured approval threshold")
	}
	var adjusted int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM((metadata->>'amount_kobo')::bigint),0) FROM app.operation_actions WHERE resource_id=$1::uuid AND action IN ('write_off','fee_waiver')`, obligation).Scan(&adjusted); err != nil {
		return Action{}, err
	}
	if !verified && adjusted >= int64(threshold-amount) {
		return Action{}, errors.New("cumulative corrections require a proposal in Admin Financial changes")
	}

	if kind == "write_off" && amount > outstanding {
		return Action{}, errors.New("write-off exceeds outstanding balance")
	}
	if kind == "write_off" {
		if err := db.GuardUnreservedReduction(ctx, tx, obligation, int64(outstanding-amount)); err != nil {
			return Action{}, err
		}
	}
	if kind == "fee_waiver" {
		var prior ledger.Money
		if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM((metadata->>'amount_kobo')::bigint),0) FROM app.operation_actions WHERE resource_id=$1::uuid AND action='fee_waiver'`, obligation).Scan(&prior); err != nil {
			return Action{}, err
		}
		var collectionFees ledger.Money
		if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(amount_kobo),0) FROM app.fees WHERE obligation_id=$1::uuid AND state='accrued'`, obligation).Scan(&collectionFees); err != nil {
			return Action{}, err
		}
		totalFees, err := ledger.CheckedAdd(baseFee, collectionFees)
		if err != nil {
			return Action{}, errors.New("accrued fees overflow")
		}
		if prior > totalFees || amount > totalFees-prior {
			return Action{}, errors.New("fee waiver exceeds accrued fees")
		}
	}
	ledgerID, err := postOperationLedgerTx(ctx, tx, kind, obligation, "operation:"+actionID, amount)
	if err != nil {
		return Action{}, err
	}
	if kind == "write_off" {
		if err := db.ReduceSchedulePrincipalTx(ctx, tx, obligation, outstanding, amount, false); err != nil {
			return Action{}, err
		}
		if err := updateOperationBalanceTx(ctx, tx, requestID, obligation, outstanding-amount, principal); err != nil {
			return Action{}, err
		}
	}
	metadata, _ := json.Marshal(map[string]any{"amount_kobo": amount, "approved_by": approvedBy, "ledger_transaction_id": ledgerID, "kind": kind})
	action := Action{ID: actionID, ActorUserID: actor, OrganizationID: org, ActionType: kind, ObligationID: obligation, AmountKobo: amount, Reason: strings.TrimSpace(reason), ApprovedBy: approvedBy, LedgerTransactionID: ledgerID}
	if err := tx.QueryRow(ctx, `INSERT INTO app.operation_actions(id,actor_user_id,organization_id,action,resource_type,resource_id,reason,metadata) VALUES($1::uuid,$2::uuid,$3::uuid,$4,'obligation',$5::uuid,$6,$7::jsonb) RETURNING created_at`, action.ID, action.ActorUserID, action.OrganizationID, action.ActionType, action.ObligationID, action.Reason, metadata).Scan(&action.CreatedAt); err != nil {
		return Action{}, err
	}
	if s.outbox != nil {
		payload, _ := json.Marshal(action)
		if _, err := s.outbox.AppendTx(ctx, tx, outbox.Event{AggregateType: "obligation", AggregateID: obligation, EventType: "operation." + kind, Payload: payload, IdempotencyKey: "operation:" + actionID}); err != nil {
			return Action{}, err
		}
	}
	return action, nil
}

func postOperationLedgerTx(ctx context.Context, tx pgx.Tx, kind, referenceID, key string, amount ledger.Money) (string, error) {
	eventType := "write_off"
	referenceType := "obligation"
	debit := ledger.AccountWriteOff
	credit := ledger.AccountTradeReceivable
	if kind == "fee_waiver" {
		eventType = "fee_waived"
		referenceType = "obligation"
		debit = ledger.AccountPlatformServiceRevenue
		credit = ledger.AccountSupplierFeeReceivable
	}
	var id string
	err := tx.QueryRow(ctx, `INSERT INTO ledger.transactions(event_type,reference_type,reference_id,idempotency_key,effective_at) VALUES($1,$2,$3,$4,now()) ON CONFLICT(idempotency_key) DO NOTHING RETURNING id::text`, eventType, referenceType, referenceID, key).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `SELECT id::text FROM ledger.transactions WHERE idempotency_key=$1`, key).Scan(&id)
		return id, err
	}
	if err != nil {
		return "", err
	}
	for _, p := range []struct {
		account       string
		debit, credit int64
	}{{debit, int64(amount), 0}, {credit, 0, int64(amount)}} {
		if _, err := tx.Exec(ctx, `INSERT INTO ledger.postings(transaction_id,account_id,debit_kobo,credit_kobo) SELECT $1::uuid,id,$3,$4 FROM ledger.accounts WHERE code=$2`, id, p.account, p.debit, p.credit); err != nil {
			return "", err
		}
	}
	return id, nil
}
func updateOperationBalanceTx(ctx context.Context, tx pgx.Tx, requestID, obligationID string, outstanding, principal ledger.Money) error {
	var status string
	switch outstanding {
	case 0:
		status = "PAID"
	case principal:
		status = "UNPAID"
	default:
		status = "PARTIALLY_PAID"
	}
	if _, err := tx.Exec(ctx, `UPDATE app.obligations SET outstanding_kobo=$2,payment_status=$3 WHERE id=$1::uuid`, obligationID, int64(outstanding), status); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE app.credit_requests SET version=version+1,updated_at=now() WHERE id=$1::uuid`, requestID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE app.credit_aggregate_snapshots SET aggregate=jsonb_set(jsonb_set(jsonb_set(aggregate,'{obligation,outstanding_kobo}',to_jsonb($2::bigint),false),'{obligation,payment_status}',to_jsonb($3::text),false),'{request,version}',to_jsonb(version+1),false),version=version+1,updated_at=now() WHERE credit_request_id=$1`, requestID, int64(outstanding), status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("credit aggregate snapshot not found")
	}
	return nil
}
func loadOperationTx(ctx context.Context, tx pgx.Tx, id string) (Action, bool, error) {
	var a Action
	var metadata []byte
	err := tx.QueryRow(ctx, `SELECT id::text,actor_user_id::text,COALESCE(organization_id::text,''),action,resource_id::text,reason,metadata,created_at FROM app.operation_actions WHERE id=$1::uuid`, id).Scan(&a.ID, &a.ActorUserID, &a.OrganizationID, &a.ActionType, &a.ObligationID, &a.Reason, &metadata, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Action{}, false, nil
	}
	if err != nil {
		return Action{}, false, err
	}
	var m struct {
		Amount   ledger.Money `json:"amount_kobo"`
		Approved string       `json:"approved_by"`
		Ledger   string       `json:"ledger_transaction_id"`
	}
	if err := json.Unmarshal(metadata, &m); err != nil {
		return Action{}, false, err
	}
	a.AmountKobo = m.Amount
	a.ApprovedBy = m.Approved
	a.LedgerTransactionID = m.Ledger
	return a, true, nil
}
func (s *PostgresStore) ListForOrganization(org string) []Action {
	rows, err := s.pool.Query(context.Background(), `SELECT id::text FROM app.operation_actions WHERE organization_id=$1::uuid ORDER BY created_at DESC`, org)
	if err != nil {
		return []Action{}
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil {
			return []Action{}
		}
		ids = append(ids, id)
	}
	if rows.Err() != nil {
		return []Action{}
	}
	rows.Close()
	out := []Action{}
	for _, id := range ids {
		tx, e := s.pool.Begin(context.Background())
		if e != nil {
			return []Action{}
		}
		a, ok, e := loadOperationTx(context.Background(), tx, id)
		_ = tx.Rollback(context.Background())
		if e != nil || !ok {
			return []Action{}
		}
		out = append(out, a)
	}
	if rows.Err() != nil {
		return []Action{}
	}
	return out
}

// Reduce the latest unpaid instalments first while retaining payment allocations
// and paid-item references needed for reversals. The accepted agreement and the
// operation audit retain the original commitment.
