package access

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// LockPlatformAuthority checks current user and role authority while holding
// the same lifecycle lock used by role revocation and account suspension.
func LockPlatformAuthority(ctx context.Context, tx pgx.Tx, actor string, permission Permission) error {
	// Match the owner lifecycle triggers' lock order before taking user/role
	// row locks. Otherwise concurrent role revocation and account suspension
	// can wait on each other's locks in opposite order.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(746219830045::bigint)`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, actor); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT a.role FROM app.platform_role_assignments a JOIN app.users u ON u.id=a.user_id WHERE a.user_id=$1::uuid AND a.revoked_at IS NULL AND (a.expires_at IS NULL OR a.expires_at>now()) AND u.status='active' FOR SHARE OF a,u`, actor)
	if err != nil {
		return err
	}
	defer rows.Close()
	allowed := false
	for rows.Next() {
		var role PlatformRole
		if err := rows.Scan(&role); err != nil {
			return err
		}
		allowed = allowed || CanPlatform(role, permission)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !allowed {
		return errors.New("current operator authority is required")
	}
	return nil
}

// LockOrganizationAuthority keeps a supplier command ordered against membership
// revocation and account/business suspension through its financial commit.
func LockOrganizationAuthority(ctx context.Context, tx pgx.Tx, actor, organization string, permission Permission) error {
	var role Role
	err := tx.QueryRow(ctx, `SELECT m.role FROM app.memberships m JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id WHERE m.organization_id=$1::uuid AND m.user_id=$2::uuid AND m.status='active' AND u.status='active' AND o.status<>'suspended' FOR SHARE OF m,u,o`, organization, actor).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("current business authority is required")
	}
	if err != nil {
		return err
	}
	if !Can(role, permission) {
		return errors.New("current business permission is required")
	}
	return nil
}
