package web

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var errOwnershipChanged = errors.New("ownership or account status changed; reload before transferring")

func transferOwnershipTx(ctx context.Context, tx pgx.Tx, actorID, targetID, reason string) error {
	// Match the database's owner lifecycle lock before taking account or role
	// row locks. A concurrent transfer must recheck the caller after waiting.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(746219830045::bigint)`); err != nil {
		return err
	}
	var eligible bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(
	 SELECT 1 FROM app.platform_role_assignments r JOIN app.users u ON u.id=r.user_id
	 WHERE r.user_id=$1::uuid AND r.role='platform_owner' AND r.revoked_at IS NULL
	 AND (r.expires_at IS NULL OR r.expires_at>clock_timestamp()) AND u.status='active')`, actorID).Scan(&eligible); err != nil {
		return err
	}
	if !eligible || actorID == targetID {
		return errOwnershipChanged
	}
	if err := tx.QueryRow(ctx, `SELECT status='active' FROM app.users WHERE id=$1::uuid FOR SHARE`, targetID).Scan(&eligible); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errOwnershipChanged
		}
		return err
	}
	if !eligible {
		return errOwnershipChanged
	}
	for _, role := range []string{"platform_owner", "platform_admin"} {
		if _, err := tx.Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason)
		 VALUES($1::uuid,$2,$3::uuid,$4)
		 ON CONFLICT(user_id,role) WHERE revoked_at IS NULL
		 DO UPDATE SET reason=EXCLUDED.reason,granted_by=EXCLUDED.granted_by,granted_at=now(),expires_at=NULL`, targetID, role, actorID, reason); err != nil {
			return err
		}
	}
	result, err := tx.Exec(ctx, `UPDATE app.platform_role_assignments SET revoked_at=now(),revoked_by=$1::uuid
	 WHERE user_id=$1::uuid AND role='platform_owner' AND revoked_at IS NULL`, actorID)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return errOwnershipChanged
	}
	return nil
}
