package reports

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"kredit/internal/access"
	"kredit/internal/db"
)

func (s *Store) enterpriseSnapshot(ctx context.Context, orgID string) (*Store, []BranchExposure, map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, err
	}
	if orgID == "" {
		return nil, nil, nil, errors.New("business identity is required")
	}
	if s.pool == nil {
		return s, nil, map[string]string{}, nil
	}
	identity, ok := db.TenantFromContext(ctx)
	if !ok || identity.UserID == "" || identity.OrganizationID != orgID {
		return nil, nil, nil, errors.New("authorized portfolio identity is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = db.SetTenantContext(ctx, tx); err != nil {
		return nil, nil, nil, err
	}
	var role string
	if err = tx.QueryRow(ctx, `SELECT m.role FROM app.memberships m JOIN app.users u ON u.id=m.user_id WHERE m.organization_id=$1::uuid AND m.user_id=app.current_user_id() AND m.status='active' AND u.status='active'`, orgID).Scan(&role); err != nil {
		return nil, nil, nil, err
	}
	if !access.Can(access.Role(role), access.PermissionReadFinancial) {
		return nil, nil, nil, errors.New("current financial read authority is required")
	}
	snapshot, err := s.financialSnapshotTx(ctx, tx, orgID, "")
	if err != nil {
		return nil, nil, nil, err
	}
	// Include inactive branches that still own historical customer exposure.
	// Their names and assignments are subject to the existing branch RLS.
	rows, err := tx.Query(ctx, `SELECT id::text,name,territory FROM app.business_branches WHERE organization_id=$1::uuid ORDER BY id`, orgID)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	branches := []BranchExposure{}
	for rows.Next() {
		var b BranchExposure
		if err = rows.Scan(&b.BranchID, &b.BranchName, &b.Territory); err != nil {
			return nil, nil, nil, err
		}
		branches = append(branches, b)
	}
	if err = rows.Err(); err != nil {
		return nil, nil, nil, err
	}
	rows.Close()
	rows, err = tx.Query(ctx, `SELECT buyer_business_id::text,COALESCE(branch_id::text,'') FROM app.partner_assignments WHERE organization_id=$1::uuid ORDER BY buyer_business_id`, orgID)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	assignments := map[string]string{}
	for rows.Next() {
		var business, branch string
		if err = rows.Scan(&business, &branch); err != nil {
			return nil, nil, nil, err
		}
		assignments[business] = branch
	}
	if err = rows.Err(); err != nil {
		return nil, nil, nil, err
	}
	rows.Close()
	if err = tx.Commit(ctx); err != nil {
		return nil, nil, nil, err
	}
	return snapshot, branches, assignments, nil
}
