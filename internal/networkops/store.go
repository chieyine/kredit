// Package networkops owns supplier-private branch and partner assignments.
// Assignments organize work; they never grant purchasing or downstream access.
package networkops

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"kredit/internal/access"
	"strings"
)

var ErrConflict = errors.New("network record changed")
var ErrInvalid = errors.New("invalid network record")
var ErrAuthority = errors.New("current business authority required")

type Store struct{ Pool *pgxpool.Pool }
type Branch struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Territory string `json:"territory"`
	Active    bool   `json:"active"`
	Version   int64  `json:"version"`
}
type Partner struct {
	BusinessID    string `json:"business_id"`
	Name          string `json:"name"`
	BranchID      string `json:"branch_id"`
	ManagerID     string `json:"manager_id"`
	ManagerActive bool   `json:"manager_active"`
	Version       int64  `json:"version"`
}
type Manager struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Workspace struct {
	Branches  []Branch  `json:"branches"`
	Partners  []Partner `json:"partners"`
	Managers  []Manager `json:"managers"`
	CanManage bool      `json:"can_manage"`
}

func (s Store) begin(ctx context.Context, user, org string, write bool) (pgx.Tx, bool, error) {
	if s.Pool == nil {
		return nil, false, errors.New("business database unavailable")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, user, org); err != nil {
		tx.Rollback(ctx)
		return nil, false, err
	}
	var role access.Role
	err = tx.QueryRow(ctx, `SELECT m.role FROM app.memberships m JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id WHERE m.organization_id=$1::uuid AND m.user_id=$2::uuid AND m.status='active' AND u.status='active' AND o.status<>'suspended' FOR SHARE OF m,u,o`, org, user).Scan(&role)
	manage := role == access.RoleOwner || role == access.RoleAdministrator
	if err != nil {
		tx.Rollback(ctx)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, ErrAuthority
		}
		return nil, false, err
	}
	if write && !manage {
		tx.Rollback(ctx)
		return nil, false, ErrAuthority
	}
	return tx, manage, nil
}
func (s Store) Read(ctx context.Context, user, org string) (Workspace, error) {
	out := Workspace{Branches: []Branch{}, Partners: []Partner{}, Managers: []Manager{}}
	tx, manage, err := s.begin(ctx, user, org, false)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	out.CanManage = manage
	rows, err := tx.Query(ctx, `SELECT id::text,name,territory,active,version FROM app.business_branches WHERE organization_id=$1::uuid ORDER BY active DESC,name,id`, org)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var b Branch
		if err = rows.Scan(&b.ID, &b.Name, &b.Territory, &b.Active, &b.Version); err != nil {
			rows.Close()
			return out, err
		}
		out.Branches = append(out.Branches, b)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = tx.Query(ctx, `SELECT c.buyer_business_id::text,COALESCE(NULLIF(c.trading_name,''),c.legal_name),COALESCE(a.branch_id::text,''),COALESCE(a.manager_user_id::text,''),EXISTS(SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id WHERE m.organization_id=$1::uuid AND m.user_id=a.manager_user_id AND m.status='active' AND u.status='active' AND m.role IN ('owner','administrator','sales')),COALESCE(a.version,0) FROM app.supplier_customers($1::uuid) c LEFT JOIN app.partner_assignments a ON a.organization_id=$1::uuid AND a.buyer_business_id=c.buyer_business_id ORDER BY c.legal_name,c.buyer_business_id`, org)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var p Partner
		if err = rows.Scan(&p.BusinessID, &p.Name, &p.BranchID, &p.ManagerID, &p.ManagerActive, &p.Version); err != nil {
			rows.Close()
			return out, err
		}
		out.Partners = append(out.Partners, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = tx.Query(ctx, `SELECT m.user_id::text,COALESCE(NULLIF(u.display_name,''),m.user_id::text) FROM app.memberships m JOIN app.users u ON u.id=m.user_id WHERE m.organization_id=$1::uuid AND m.status='active' AND u.status='active' AND m.role IN ('owner','administrator','sales') ORDER BY m.created_at,m.id`, org)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var m Manager
		if err = rows.Scan(&m.ID, &m.Name); err != nil {
			return out, err
		}
		out.Managers = append(out.Managers, m)
	}
	return out, rows.Err()
}
func (s Store) SaveBranch(ctx context.Context, user, org string, b Branch) (Branch, error) {
	b.Name = strings.TrimSpace(b.Name)
	b.Territory = strings.TrimSpace(b.Territory)
	if b.Name == "" || len(b.Name) > 100 || len(b.Territory) > 160 || b.Version < 0 {
		return Branch{}, ErrInvalid
	}
	tx, _, err := s.begin(ctx, user, org, true)
	if err != nil {
		return Branch{}, err
	}
	defer tx.Rollback(ctx)
	var version int64
	if b.Version == 0 {
		err = tx.QueryRow(ctx, `INSERT INTO app.business_branches(id,organization_id,name,territory,active,updated_by) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6::uuid) ON CONFLICT DO NOTHING RETURNING version`, b.ID, org, b.Name, b.Territory, b.Active, user).Scan(&version)
	} else {
		err = tx.QueryRow(ctx, `UPDATE app.business_branches SET name=$3,territory=$4,active=$5,updated_by=$6::uuid,version=version+1,updated_at=now() WHERE id=$1::uuid AND organization_id=$2::uuid AND version=$7 RETURNING version`, b.ID, org, b.Name, b.Territory, b.Active, user, b.Version).Scan(&version)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `SELECT version FROM app.business_branches WHERE id=$1::uuid AND organization_id=$2::uuid AND name=$3 AND territory=$4 AND active=$5 AND updated_by=$6::uuid AND version=$7+1`, b.ID, org, b.Name, b.Territory, b.Active, user, b.Version).Scan(&version)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return Branch{}, ErrConflict
	}
	if err != nil {
		return Branch{}, err
	}
	b.Version = version
	return b, tx.Commit(ctx)
}
func (s Store) Assign(ctx context.Context, user, org string, p Partner) (Partner, error) {
	if p.Version < 0 {
		return Partner{}, ErrInvalid
	}
	tx, _, err := s.begin(ctx, user, org, true)
	if err != nil {
		return Partner{}, err
	}
	defer tx.Rollback(ctx)
	var version int64
	if p.Version == 0 {
		err = tx.QueryRow(ctx, `INSERT INTO app.partner_assignments(organization_id,buyer_business_id,branch_id,manager_user_id,updated_by) VALUES($1::uuid,$2::uuid,NULLIF($3,'')::uuid,NULLIF($4,'')::uuid,$5::uuid) ON CONFLICT DO NOTHING RETURNING version`, org, p.BusinessID, p.BranchID, p.ManagerID, user).Scan(&version)
	} else {
		err = tx.QueryRow(ctx, `UPDATE app.partner_assignments SET branch_id=NULLIF($3,'')::uuid,manager_user_id=NULLIF($4,'')::uuid,updated_by=$5::uuid,version=version+1,updated_at=now() WHERE organization_id=$1::uuid AND buyer_business_id=$2::uuid AND version=$6 RETURNING version`, org, p.BusinessID, p.BranchID, p.ManagerID, user, p.Version).Scan(&version)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `SELECT version FROM app.partner_assignments WHERE organization_id=$1::uuid AND buyer_business_id=$2::uuid AND COALESCE(branch_id::text,'')=$3 AND COALESCE(manager_user_id::text,'')=$4 AND updated_by=$5::uuid AND version=$6+1`, org, p.BusinessID, p.BranchID, p.ManagerID, user, p.Version).Scan(&version)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return Partner{}, ErrConflict
	}
	if err != nil {
		return Partner{}, err
	}
	p.Version = version
	return p, tx.Commit(ctx)
}
