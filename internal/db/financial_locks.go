package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// LockObligationRequest locks an existing financial aggregate in two explicit
// statements: obligation first, then its credit request. A multi-table locking
// SELECT does not document or enforce the order independently of its plan.
//
// The caller must install its authorized tenant context first and must not
// already hold the request, payment, or order-note lock ahead of the obligation.
// No identity is derived from a row or elevated here; both queries retain RLS.
// On any error the owning transaction must roll back, without an automatic
// retry of a potentially committed financial operation.
func LockObligationRequest(ctx context.Context, tx pgx.Tx, obligationID string) error {
	identity, ok := TenantFromContext(ctx)
	if tx == nil || obligationID == "" || !ok || identity.OrganizationID == "" {
		return errors.New("transaction, obligation and authorized supplier context are required")
	}
	var requestID string
	if err := tx.QueryRow(ctx, `
        SELECT credit_request_id::text FROM app.obligations
        WHERE id=$1::uuid AND supplier_organization_id=$2::uuid
        FOR UPDATE`, obligationID, identity.OrganizationID).Scan(&requestID); err != nil {
		return fmt.Errorf("lock authorized obligation: %w", err)
	}
	var lockedID string
	if err := tx.QueryRow(ctx, `
        SELECT id::text FROM app.credit_requests
        WHERE id=$1::uuid AND supplier_organization_id=$2::uuid
        FOR UPDATE`, requestID, identity.OrganizationID).Scan(&lockedID); err != nil {
		return fmt.Errorf("lock obligation credit request: %w", err)
	}
	return nil
}
