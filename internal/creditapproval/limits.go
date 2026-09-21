package creditapproval

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type ReviewerLimit struct {
	UserID  string `json:"user_id"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	Ceiling *int64 `json:"ceiling_kobo"`
	Version int64  `json:"version"`
}

func (s Store) Limits(ctx context.Context, user, org string) ([]ReviewerLimit, error) {
	tx, _, err := s.begin(ctx, user, org, "owner", "administrator", "finance", "sales")
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return readReviewerLimits(ctx, tx, org)
}

func readReviewerLimits(ctx context.Context, tx pgx.Tx, org string) ([]ReviewerLimit, error) {
	rows, err := tx.Query(ctx, `SELECT m.user_id::text,COALESCE(NULLIF(u.display_name,''),m.user_id::text),m.role,l.ceiling_kobo,COALESCE(l.version,0) FROM app.memberships m JOIN app.users u ON u.id=m.user_id LEFT JOIN app.credit_reviewer_limits l ON l.organization_id=m.organization_id AND l.user_id=m.user_id WHERE m.organization_id=$1::uuid AND m.status='active' AND u.status='active' AND m.role IN ('owner','administrator','finance') ORDER BY m.created_at,m.id`, org)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ReviewerLimit{}
	for rows.Next() {
		var item ReviewerLimit
		if err = rows.Scan(&item.UserID, &item.Name, &item.Role, &item.Ceiling, &item.Version); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
func (s Store) SetLimit(ctx context.Context, user, org, target string, ceiling, version int64) (ReviewerLimit, error) {
	if ceiling < 0 || ceiling > 9007199254740991 || version < 0 {
		return ReviewerLimit{}, ErrInvalid
	}
	tx, _, err := s.begin(ctx, user, org, "owner")
	if err != nil {
		return ReviewerLimit{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,175))`, org); err != nil {
		return ReviewerLimit{}, err
	}
	var current string
	err = tx.QueryRow(ctx, `SELECT m.user_id::text FROM app.memberships m JOIN app.users u ON u.id=m.user_id WHERE m.organization_id=$1::uuid AND m.user_id=$2::uuid AND m.status='active' AND u.status='active' AND m.role IN ('owner','administrator','finance') FOR SHARE OF m,u`, org, target).Scan(&current)
	if err != nil {
		return ReviewerLimit{}, err
	}
	var next ReviewerLimit
	err = tx.QueryRow(ctx, `INSERT INTO app.credit_reviewer_limits(organization_id,user_id,ceiling_kobo,updated_by) SELECT $1::uuid,$2::uuid,$3,$4::uuid WHERE $5=0 ON CONFLICT DO NOTHING RETURNING user_id::text,ceiling_kobo,version`, org, target, ceiling, user, version).Scan(&next.UserID, &next.Ceiling, &next.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `UPDATE app.credit_reviewer_limits SET ceiling_kobo=$3,version=version+1,updated_by=$4::uuid,updated_at=now() WHERE organization_id=$1::uuid AND user_id=$2::uuid AND version=$5 RETURNING user_id::text,ceiling_kobo,version`, org, target, ceiling, user, version).Scan(&next.UserID, &next.Ceiling, &next.Version)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ReviewerLimit{}, ErrConflict
	}
	if err != nil {
		return ReviewerLimit{}, err
	}
	return next, tx.Commit(ctx)
}
