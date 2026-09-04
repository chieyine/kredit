package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// SetObligationContext establishes persisted tenant identity for a financial
// transaction. Callers must authorize the action before entering the repository.
func SetObligationContext(ctx context.Context, tx pgx.Tx, obligationID string) error {
	var buyer, supplier string
	if err := tx.QueryRow(ctx, `SELECT s.buyer_user_id,s.supplier_organization_id FROM app.credit_aggregate_snapshots s JOIN app.obligations o ON o.credit_request_id::text=s.credit_request_id WHERE o.id=$1::uuid`, obligationID).Scan(&buyer, &supplier); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, buyer, supplier)
	return err
}

// GuardUnreservedReduction requires the caller to hold the obligation row lock.
// Debt forgiveness cannot consume principal covered by an unresolved bank debit.
func GuardUnreservedReduction(ctx context.Context, tx pgx.Tx, obligationID string, nextOutstanding int64) error {
	var held int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(reserved_amount_kobo),0) FROM app.collection_reservations WHERE obligation_id=$1::uuid AND state IN ('PROCESSING','COMPLETED')`, obligationID).Scan(&held); err != nil {
		return err
	}
	if nextOutstanding < held {
		return errors.New("balance reduction conflicts with an unresolved debit; reconcile first")
	}
	return nil
}
