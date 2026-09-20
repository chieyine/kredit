// Package purchasing manages explicit business buying authority.
package purchasing

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"sort"
	"time"
)

var ErrAuthority = errors.New("current owner authority required")
var ErrConflict = errors.New("purchasing permission changed")
var ErrInvalid = errors.New("invalid purchasing permission")

type Store struct{ Pool *pgxpool.Pool }
type Grant struct {
	Active              bool      `json:"active"`
	UserID              string    `json:"user_id"`
	Name                string    `json:"name"`
	Actions             []string  `json:"actions"`
	CeilingKobo         int64     `json:"ceiling_kobo"`
	DrawdownCeilingKobo int64     `json:"drawdown_ceiling_kobo"`
	ExpiresAt           time.Time `json:"expires_at"`
	Version             int64     `json:"version"`
}
type Directory struct {
	Grants    []Grant `json:"grants"`
	CanManage bool    `json:"can_manage"`
}

func (s Store) begin(ctx context.Context, actor, org string, write bool) (pgx.Tx, bool, error) {
	if s.Pool == nil {
		return nil, false, errors.New("purchasing database unavailable")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	fail := func(e error) (pgx.Tx, bool, error) { tx.Rollback(ctx); return nil, false, e }
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, actor, org); err != nil {
		return fail(err)
	}
	var role string
	err = tx.QueryRow(ctx, `SELECT m.role FROM app.memberships m JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id WHERE m.organization_id=$1::uuid AND m.user_id=$2::uuid AND m.status='active' AND u.status='active' AND o.status<>'suspended' FOR SHARE OF m,u,o`, org, actor).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return fail(ErrAuthority)
	}
	if err != nil {
		return fail(err)
	}
	if write && role != "owner" {
		return fail(ErrAuthority)
	}
	return tx, role == "owner", nil
}
func (s Store) Read(ctx context.Context, actor, org string) (Directory, error) {
	out := Directory{Grants: []Grant{}}
	tx, owner, err := s.begin(ctx, actor, org, false)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	out.CanManage = owner
	rows, err := tx.Query(ctx, `SELECT m.user_id::text,COALESCE(NULLIF(u.display_name,''),m.user_id::text),COALESCE(d.actions,'{}'),COALESCE(d.ceiling_kobo,0),COALESCE(d.drawdown_ceiling_kobo,0),COALESCE(d.expires_at,'epoch'::timestamptz),COALESCE(d.version,0),COALESCE(NOT EXISTS(SELECT 1 FROM app.businesses b WHERE b.organization_id=m.organization_id AND b.owner_user_id=m.user_id AND m.role<>'owner') AND d.membership_id=m.id AND d.membership_authority_version=m.purchasing_authority_version AND d.expires_at>statement_timestamp() AND 'read'=ANY(d.actions),false) FROM app.memberships m JOIN app.users u ON u.id=m.user_id LEFT JOIN app.purchasing_delegations d ON d.organization_id=m.organization_id AND d.user_id=m.user_id WHERE m.organization_id=$1::uuid AND m.status='active' AND u.status='active' AND ($2 OR m.user_id=$3::uuid) ORDER BY u.display_name,m.user_id`, org, owner, actor)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var g Grant
		if err = rows.Scan(&g.UserID, &g.Name, &g.Actions, &g.CeilingKobo, &g.DrawdownCeilingKobo, &g.ExpiresAt, &g.Version, &g.Active); err != nil {
			return out, err
		}
		out.Grants = append(out.Grants, g)
	}
	return out, rows.Err()
}
func (s Store) Save(ctx context.Context, actor, org string, g Grant) (Grant, error) {
	if g.Version < 0 || g.CeilingKobo < 0 || g.CeilingKobo > 9007199254740991 || g.DrawdownCeilingKobo < 0 || g.DrawdownCeilingKobo > 9007199254740991 || g.ExpiresAt.IsZero() {
		return Grant{}, ErrInvalid
	}
	validActions := map[string]bool{
		"read":     true,
		"review":   true,
		"receive":  true,
		"accept":   true,
		"drawdown": true,
		"dispute":  true,
		"claim":    true,
		"amend":    true,
	}
	seen := map[string]bool{}
	for _, a := range g.Actions {
		if !validActions[a] {
			return Grant{}, ErrInvalid
		}
		if seen[a] {
			return Grant{}, ErrInvalid
		}
		seen[a] = true
	}
	if len(g.Actions) > 0 && !seen["read"] {
		return Grant{}, ErrInvalid
	}
	if g.Actions == nil {
		g.Actions = []string{}
	}
	sort.Strings(g.Actions)
	tx, _, err := s.begin(ctx, actor, org, true)
	if err != nil {
		return Grant{}, err
	}
	defer tx.Rollback(ctx)
	var version int64
	if g.Version == 0 {
		err = tx.QueryRow(ctx, `INSERT INTO app.purchasing_delegations(organization_id,user_id,actions,ceiling_kobo,drawdown_ceiling_kobo,expires_at,updated_by) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,$7::uuid) ON CONFLICT DO NOTHING RETURNING version`, org, g.UserID, g.Actions, g.CeilingKobo, g.DrawdownCeilingKobo, g.ExpiresAt, actor).Scan(&version)
	} else {
		err = tx.QueryRow(ctx, `UPDATE app.purchasing_delegations SET actions=$3,ceiling_kobo=$4,drawdown_ceiling_kobo=$5,expires_at=$6,updated_by=$7::uuid,version=version+1,updated_at=now() WHERE organization_id=$1::uuid AND user_id=$2::uuid AND version=$8 RETURNING version`, org, g.UserID, g.Actions, g.CeilingKobo, g.DrawdownCeilingKobo, g.ExpiresAt, actor, g.Version).Scan(&version)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `SELECT version FROM app.purchasing_delegations WHERE organization_id=$1::uuid AND user_id=$2::uuid AND actions=$3 AND ceiling_kobo=$4 AND drawdown_ceiling_kobo=$5 AND expires_at=$6 AND updated_by=$7::uuid AND version=$8+1`, org, g.UserID, g.Actions, g.CeilingKobo, g.DrawdownCeilingKobo, g.ExpiresAt, actor, g.Version).Scan(&version)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return Grant{}, ErrConflict
	}
	if err != nil {
		return Grant{}, err
	}
	g.Version = version
	g.Active = len(g.Actions) > 0 && g.ExpiresAt.After(time.Now())
	return g, tx.Commit(ctx)
}
