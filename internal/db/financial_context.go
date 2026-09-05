package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// SetTenantContext establishes an already-authorized request or worker identity
// on the transaction-local settings consumed by PostgreSQL RLS policies.
func SetTenantContext(ctx context.Context, tx pgx.Tx) error {
	identity, ok := TenantFromContext(ctx)
	if !ok {
		return errors.New("authorized tenant context is required")
	}
	_, err := tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, identity.UserID, identity.OrganizationID)
	if err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}
	return nil
}

// SetObligationContext establishes authoritative tenant identity before looking
// up an obligation. In particular, it never discovers an organization by first
// reading the target obligation under a service-wide policy.
func SetObligationContext(ctx context.Context, tx pgx.Tx, obligationID string) error {
	identity, ok := TenantFromContext(ctx)
	if !ok || identity.OrganizationID == "" {
		return errors.New("authorized supplier organization context is required")
	}
	if err := SetTenantContext(ctx, tx); err != nil {
		return err
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.obligations WHERE id=$1::uuid AND supplier_organization_id=$2::uuid)`, obligationID, identity.OrganizationID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errors.New("obligation is outside the authorized tenant")
	}
	return nil
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
