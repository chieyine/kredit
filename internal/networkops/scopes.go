package networkops

import (
	"context"
	"errors"
	"sort"

	"github.com/jackc/pgx/v5"
)

type Scope struct {
	UserID    string   `json:"user_id"`
	Name      string   `json:"name"`
	Mode      string   `json:"mode"`
	BranchIDs []string `json:"branch_ids"`
	Version   int64    `json:"version"`
	Active    bool     `json:"active"`
}
type ScopeDirectory struct {
	Scopes    []Scope `json:"scopes"`
	CanManage bool    `json:"can_manage"`
}

func (s Store) ReadScopes(ctx context.Context, actor, org string) (ScopeDirectory, error) {
	out := ScopeDirectory{Scopes: []Scope{}}
	tx, _, err := s.begin(ctx, actor, org, false)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.memberships WHERE organization_id=$1::uuid AND user_id=$2::uuid AND role='owner' AND status='active')`, org, actor).Scan(&out.CanManage); err != nil {
		return out, err
	}
	rows, err := tx.Query(ctx, `SELECT m.user_id::text,COALESCE(NULLIF(u.display_name,''),m.user_id::text),COALESCE(s.mode,'all'),COALESCE(s.branch_ids,'{}'),COALESCE(s.version,0),s.user_id IS NULL OR (s.membership_id=m.id AND s.membership_version=m.purchasing_authority_version) FROM app.memberships m JOIN app.users u ON u.id=m.user_id LEFT JOIN app.member_branch_scopes s ON s.organization_id=m.organization_id AND s.user_id=m.user_id WHERE m.organization_id=$1::uuid AND m.status='active' AND u.status='active' AND m.role NOT IN ('owner','administrator') AND ($2 OR m.user_id=$3::uuid) ORDER BY u.display_name,m.user_id`, org, out.CanManage, actor)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var v Scope
		if err = rows.Scan(&v.UserID, &v.Name, &v.Mode, &v.BranchIDs, &v.Version, &v.Active); err != nil {
			return out, err
		}
		out.Scopes = append(out.Scopes, v)
	}
	return out, rows.Err()
}
func (s Store) SaveScope(ctx context.Context, actor, org string, v Scope) (Scope, error) {
	if v.Version < 0 || (v.Mode != "all" && v.Mode != "branches") || len(v.BranchIDs) > 100 || (v.Mode == "all" && len(v.BranchIDs) > 0) {
		return Scope{}, ErrInvalid
	}
	if v.BranchIDs == nil {
		v.BranchIDs = []string{}
	}
	sort.Strings(v.BranchIDs)
	for i := 1; i < len(v.BranchIDs); i++ {
		if v.BranchIDs[i] == v.BranchIDs[i-1] {
			return Scope{}, ErrInvalid
		}
	}
	tx, _, err := s.begin(ctx, actor, org, true)
	if err != nil {
		return Scope{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var owner bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.memberships WHERE organization_id=$1::uuid AND user_id=$2::uuid AND role='owner' AND status='active')`, org, actor).Scan(&owner); err != nil {
		return Scope{}, err
	}
	if !owner {
		return Scope{}, ErrAuthority
	}
	var version int64
	if v.Version == 0 {
		err = tx.QueryRow(ctx, `INSERT INTO app.member_branch_scopes(organization_id,user_id,mode,branch_ids,updated_by) VALUES($1::uuid,$2::uuid,$3,$4::uuid[],$5::uuid) ON CONFLICT DO NOTHING RETURNING version`, org, v.UserID, v.Mode, v.BranchIDs, actor).Scan(&version)
	} else {
		err = tx.QueryRow(ctx, `UPDATE app.member_branch_scopes SET mode=$3,branch_ids=$4::uuid[],updated_by=$5::uuid,version=version+1,updated_at=now() WHERE organization_id=$1::uuid AND user_id=$2::uuid AND version=$6 RETURNING version`, org, v.UserID, v.Mode, v.BranchIDs, actor, v.Version).Scan(&version)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `SELECT s.version FROM app.member_branch_scopes s JOIN app.memberships m ON m.id=s.membership_id AND m.purchasing_authority_version=s.membership_version WHERE s.organization_id=$1::uuid AND s.user_id=$2::uuid AND s.mode=$3 AND s.branch_ids=$4::uuid[] AND s.updated_by=$5::uuid AND s.version=$6+1 AND m.status='active'`, org, v.UserID, v.Mode, v.BranchIDs, actor, v.Version).Scan(&version)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return Scope{}, ErrConflict
	}
	if err != nil {
		return Scope{}, err
	}
	v.Version = version
	v.Active = true
	return v, tx.Commit(ctx)
}
