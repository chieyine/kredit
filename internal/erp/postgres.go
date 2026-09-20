package erp

import (
	"context"
	"errors"
	"time"

	"kredit/internal/access"
	"kredit/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LoadLedgerMovements returns one signed receivable movement per journal
// transaction. Payment, dispute and credit-note reference IDs are not obligation
// IDs; resolve each through its own domain relation before selecting a tenant.
func LoadLedgerMovements(ctx context.Context, pool *pgxpool.Pool, org string, from, to time.Time) ([]InternalRecord, error) {
	identity, ok := db.TenantFromContext(ctx)
	if !ok || identity.UserID == "" || identity.OrganizationID != org || org == "" {
		return nil, errors.New("authorized reconciliation identity is required")
	}
	if pool == nil {
		return nil, errors.New("authoritative ledger database is required")
	}
	if from.IsZero() || to.IsZero() || to.Before(from) {
		return nil, ErrInvalidRecords
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = db.SetTenantContext(ctx, tx); err != nil {
		return nil, err
	}
	// Explicit authority prevents revoked staff from receiving a misleading empty
	// successful reconciliation when RLS hides every obligation.
	var role string
	var companyWide bool
	if err = tx.QueryRow(ctx, `SELECT m.role,app.branch_scope_all($1::uuid) FROM app.memberships m JOIN app.users u ON u.id=m.user_id WHERE m.organization_id=$1::uuid AND m.user_id=app.current_user_id() AND m.status='active' AND u.status='active'`, org).Scan(&role, &companyWide); err != nil {
		return nil, err
	}
	if !companyWide || !access.Can(access.Role(role), access.PermissionReadFinancial) {
		return nil, errors.New("current company-wide financial authority is required")
	}
	rows, err := tx.Query(ctx, ledgerMovementsSQL, org, from, to, MaxReconciliationRecords+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []InternalRecord{}
	for rows.Next() {
		var record InternalRecord
		if err = rows.Scan(&record.Reference, &record.TransactionID, &record.AmountKobo, &record.Date); err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if len(result) > MaxReconciliationRecords {
		return nil, ErrInvalidRecords
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

const ledgerMovementsSQL = `
 WITH authorized_obligations AS (
  SELECT id FROM app.obligations WHERE supplier_organization_id=$1::uuid
 ), references_in_scope AS (
  SELECT 'obligation'::text AS kind,id::text AS reference FROM authorized_obligations
  UNION ALL
  SELECT 'payment',p.id::text FROM app.payments p JOIN authorized_obligations o ON o.id=p.obligation_id
  UNION ALL
  SELECT 'dispute',d.id::text FROM app.disputes d JOIN authorized_obligations o ON o.id=d.obligation_id
  UNION ALL
  SELECT 'credit_note',n.id::text FROM app.order_credit_notes n JOIN authorized_obligations o ON o.id=n.obligation_id
 )
 SELECT t.id::text,t.id::text,SUM(p.debit_kobo-p.credit_kobo)::bigint,t.effective_at
 FROM references_in_scope r
 JOIN ledger.transactions t ON t.reference_type=r.kind AND t.reference_id=r.reference
 JOIN ledger.postings p ON p.transaction_id=t.id
 JOIN ledger.accounts a ON a.id=p.account_id AND a.code='TRADE_RECEIVABLE_CONTROL'
 WHERE t.effective_at >= $2 AND t.effective_at <= $3
 GROUP BY t.id,t.effective_at
 ORDER BY t.effective_at,t.id
 LIMIT $4`
