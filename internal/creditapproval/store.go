// Package creditapproval owns business credit-offer review. An approval records
// the exact proposed financial terms; it is neither buyer acceptance nor debt.
package creditapproval

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAuthority = errors.New("current business authority required")
var ErrConflict = errors.New("the offer or approval has changed; refresh before continuing")
var ErrInvalid = errors.New("valid approval details are required")

type Store struct{ Pool *pgxpool.Pool }
type Controls struct {
	Enabled   bool  `json:"enabled"`
	Threshold int64 `json:"threshold_kobo"`
	Version   int64 `json:"version"`
}
type Approval struct {
	CustomerName string          `json:"customer_name"`
	ID           string          `json:"id"`
	RequestID    string          `json:"request_id,omitempty"`
	DrawdownID   string          `json:"drawdown_id,omitempty"`
	Kind         string          `json:"kind"`
	State        string          `json:"state"`
	RequestedBy  string          `json:"requested_by"`
	DecidedBy    string          `json:"decided_by"`
	Reason       string          `json:"reason"`
	CreatedAt    time.Time       `json:"created_at"`
	Proposal     json.RawMessage `json:"proposal"`
	Stale        bool            `json:"stale"`
}
type Inbox struct {
	Controls  Controls        `json:"controls"`
	Approvals []Approval      `json:"approvals"`
	Role      string          `json:"role"`
	UserID    string          `json:"user_id"`
	Reviewers []ReviewerLimit `json:"reviewers"`
}

func (s Store) begin(ctx context.Context, user, org string, roles ...string) (pgx.Tx, string, error) {
	if s.Pool == nil {
		return nil, "", errors.New("business database unavailable")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, "", err
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, user, org); err != nil {
		_ = tx.Rollback(ctx)
		return nil, "", err
	}
	var role string
	err = tx.QueryRow(ctx, `SELECT m.role FROM app.memberships m JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id WHERE m.organization_id=$1::uuid AND m.user_id=$2::uuid AND m.status='active' AND u.status='active' AND o.status<>'suspended' AND m.role=ANY($3::text[]) FOR SHARE OF m,u,o`, org, user, roles).Scan(&role)
	if err != nil {
		_ = tx.Rollback(ctx)
		if errors.Is(err, pgx.ErrNoRows) {
			err = ErrAuthority
		}
		return nil, "", err
	}
	return tx, role, nil
}
func (s Store) Read(ctx context.Context, user, org string) (Inbox, error) {
	out := Inbox{Approvals: []Approval{}, UserID: user}
	tx, role, err := s.begin(ctx, user, org, "owner", "administrator", "finance", "sales")
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	out.Role = role
	err = tx.QueryRow(ctx, `SELECT enabled,threshold_kobo,version FROM app.business_credit_controls WHERE organization_id=$1::uuid`, org).Scan(&out.Controls.Enabled, &out.Controls.Threshold, &out.Controls.Version)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	rows, err := tx.Query(ctx, `SELECT COALESCE(NULLIF(customer.trading_name,''),customer.legal_name,'Business customer'),a.id::text,a.credit_request_id::text,a.state,a.requested_by::text,COALESCE(a.decided_by::text,''),a.reason,a.created_at,a.proposal,(a.request_version<>c.version OR a.fingerprint<>app.credit_offer_fingerprint(c) OR c.state<>'DRAFT') FROM app.credit_offer_approvals a JOIN app.credit_requests c ON c.id=a.credit_request_id LEFT JOIN app.supplier_customers($1::uuid) customer ON customer.buyer_business_id=c.buyer_business_id WHERE a.organization_id=$1::uuid ORDER BY (a.state='pending') DESC,a.created_at DESC,a.id DESC LIMIT 100`, org)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var a Approval
		if err = rows.Scan(&a.CustomerName, &a.ID, &a.RequestID, &a.State, &a.RequestedBy, &a.DecidedBy, &a.Reason, &a.CreatedAt, &a.Proposal, &a.Stale); err != nil {
			return out, err
		}
		a.Kind = "sale"
		out.Approvals = append(out.Approvals, a)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	rows.Close()

	drows, err := tx.Query(ctx, `SELECT COALESCE(NULLIF(customer.trading_name,''),customer.legal_name,'Business customer'),a.id::text,a.drawdown_id::text,a.state,a.requested_by::text,COALESCE(a.decided_by::text,''),a.reason,a.created_at,a.proposal,(a.fingerprint<>app.drawdown_approval_fingerprint(d) OR d.state NOT IN ('PENDING_BUYER_CONFIRMATION','BUYER_CONFIRMED')) FROM app.tradeline_drawdown_approvals a JOIN app.drawdowns d ON d.id=a.drawdown_id JOIN app.trade_lines tl ON tl.id=d.trade_line_id LEFT JOIN app.supplier_customers($1::uuid) customer ON customer.buyer_business_id=tl.buyer_business_id WHERE a.organization_id=$1::uuid ORDER BY (a.state='pending') DESC,a.created_at DESC,a.id DESC LIMIT 100`, org)
	if err == nil {
		defer drows.Close()
		for drows.Next() {
			var a Approval
			if err = drows.Scan(&a.CustomerName, &a.ID, &a.DrawdownID, &a.State, &a.RequestedBy, &a.DecidedBy, &a.Reason, &a.CreatedAt, &a.Proposal, &a.Stale); err == nil {
				a.Kind = "drawdown"
				out.Approvals = append(out.Approvals, a)
			}
		}
	}
	out.Reviewers, err = readReviewerLimits(ctx, tx, org)
	return out, err
}
func (s Store) SetControls(ctx context.Context, user, org string, in Controls) (Controls, error) {
	if in.Threshold < 0 || in.Threshold > 9007199254740991 || in.Version < 0 {
		return Controls{}, ErrInvalid
	}
	tx, _, err := s.begin(ctx, user, org, "owner")
	if err != nil {
		return Controls{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,172))`, org); err != nil {
		return Controls{}, err
	}
	var next Controls
	err = tx.QueryRow(ctx, `INSERT INTO app.business_credit_controls(organization_id,enabled,threshold_kobo,updated_by) SELECT $1::uuid,$2,$3,$4::uuid WHERE $5=0 ON CONFLICT(organization_id) DO NOTHING RETURNING enabled,threshold_kobo,version`, org, in.Enabled, in.Threshold, user, in.Version).Scan(&next.Enabled, &next.Threshold, &next.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `UPDATE app.business_credit_controls SET enabled=$2,threshold_kobo=$3,updated_by=$4::uuid,updated_at=now(),version=version+1 WHERE organization_id=$1::uuid AND version=$5 RETURNING enabled,threshold_kobo,version`, org, in.Enabled, in.Threshold, user, in.Version).Scan(&next.Enabled, &next.Threshold, &next.Version)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return next, ErrConflict
	}
	if err != nil {
		return next, err
	}
	err = tx.Commit(ctx)
	return next, err
}
func (s Store) Request(ctx context.Context, user, org, request string) (string, error) {
	tx, _, err := s.begin(ctx, user, org, "owner", "administrator", "sales")
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var version int64
	var state string
	if err = tx.QueryRow(ctx, `SELECT version,state FROM app.credit_requests WHERE id=$1::uuid AND supplier_organization_id=$2::uuid FOR UPDATE`, request, org).Scan(&version, &state); err != nil {
		return "", err
	}
	if state != "DRAFT" {
		return "", ErrConflict
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO app.credit_offer_approvals(organization_id,credit_request_id,request_version,fingerprint,proposal,requested_by) SELECT supplier_organization_id,id,version,app.credit_offer_fingerprint(c),app.credit_offer_proposal(c),$3::uuid FROM app.credit_requests c WHERE c.id=$1::uuid AND c.supplier_organization_id=$2::uuid ON CONFLICT(organization_id,credit_request_id,request_version) DO NOTHING RETURNING id::text`, request, org, user).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `SELECT id::text FROM app.credit_offer_approvals WHERE organization_id=$1::uuid AND credit_request_id=$2::uuid AND request_version=$3`, org, request, version).Scan(&id)
	}
	if err != nil {
		return "", err
	}
	err = tx.Commit(ctx)
	return id, err
}
func (s Store) Decide(ctx context.Context, user, org, id, decision, reason string) error {
	if (decision != "approved" && decision != "rejected") || len(reason) < 3 || len(reason) > 1000 {
		return ErrInvalid
	}
	tx, _, err := s.begin(ctx, user, org, "owner", "administrator", "finance")
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Match the request-before-approval lock order used when an offer is sent.
	var request string
	if err = tx.QueryRow(ctx, `SELECT credit_request_id::text FROM app.credit_offer_approvals WHERE organization_id=$1::uuid AND id=$2::uuid`, org, id).Scan(&request); err != nil {
		return err
	}
	var creator, state string
	var version int64
	var fingerprint []byte
	if err = tx.QueryRow(ctx, `SELECT created_by::text,state,version,app.credit_offer_fingerprint(c) FROM app.credit_requests c WHERE id=$1::uuid AND supplier_organization_id=$2::uuid FOR UPDATE`, request, org).Scan(&creator, &state, &version, &fingerprint); err != nil {
		return err
	}
	if creator == user {
		return ErrAuthority
	}
	var existingState, reviewer, existingReason string
	if err = tx.QueryRow(ctx, `SELECT state,COALESCE(decided_by::text,''),reason FROM app.credit_offer_approvals WHERE id=$1::uuid`, id).Scan(&existingState, &reviewer, &existingReason); err != nil {
		return err
	}
	if existingState != "pending" {
		if existingState == decision && reviewer == user && existingReason == reason {
			return nil
		}
		return ErrConflict
	}
	if state != "DRAFT" {
		return ErrConflict
	}
	tag, err := tx.Exec(ctx, `UPDATE app.credit_offer_approvals SET state=$3,decided_by=$4::uuid,reason=$5,decided_at=now() WHERE organization_id=$1::uuid AND id=$2::uuid AND state='pending' AND requested_by<>$4::uuid AND fingerprint=$6 AND request_version=$7`, org, id, decision, user, reason, fingerprint, version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return tx.Commit(ctx)
}

func (s Store) RequestDrawdownApproval(ctx context.Context, user, org, drawdownID string) (string, error) {
	tx, _, err := s.begin(ctx, user, org, "owner", "administrator", "sales")
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var state string
	if err = tx.QueryRow(ctx, `SELECT d.state FROM app.drawdowns d JOIN app.trade_lines tl ON tl.id=d.trade_line_id WHERE d.id=$1::uuid AND tl.supplier_organization_id=$2::uuid FOR UPDATE`, drawdownID, org).Scan(&state); err != nil {
		return "", err
	}
	if state != "PENDING_BUYER_CONFIRMATION" && state != "BUYER_CONFIRMED" {
		return "", ErrConflict
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO app.tradeline_drawdown_approvals(organization_id,drawdown_id,fingerprint,proposal,requested_by) SELECT tl.supplier_organization_id,d.id,app.drawdown_approval_fingerprint(d),app.drawdown_approval_proposal(d),$3::uuid FROM app.drawdowns d JOIN app.trade_lines tl ON tl.id=d.trade_line_id WHERE d.id=$1::uuid AND tl.supplier_organization_id=$2::uuid ON CONFLICT(organization_id,drawdown_id,drawdown_version) DO NOTHING RETURNING id::text`, drawdownID, org, user).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `SELECT id::text FROM app.tradeline_drawdown_approvals WHERE organization_id=$1::uuid AND drawdown_id=$2::uuid`, org, drawdownID).Scan(&id)
	}
	if err != nil {
		return "", err
	}
	err = tx.Commit(ctx)
	return id, err
}

func (s Store) DecideDrawdown(ctx context.Context, user, org, id, decision, reason string) error {
	if (decision != "approved" && decision != "rejected") || len(reason) < 3 || len(reason) > 1000 {
		return ErrInvalid
	}
	tx, _, err := s.begin(ctx, user, org, "owner", "administrator", "finance")
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var drawdownID string
	if err = tx.QueryRow(ctx, `SELECT drawdown_id::text FROM app.tradeline_drawdown_approvals WHERE organization_id=$1::uuid AND id=$2::uuid`, org, id).Scan(&drawdownID); err != nil {
		return err
	}
	var state string
	var fingerprint []byte
	if err = tx.QueryRow(ctx, `SELECT d.state,app.drawdown_approval_fingerprint(d) FROM app.drawdowns d JOIN app.trade_lines tl ON tl.id=d.trade_line_id WHERE d.id=$1::uuid AND tl.supplier_organization_id=$2::uuid FOR UPDATE`, drawdownID, org).Scan(&state, &fingerprint); err != nil {
		return err
	}
	var existingState, reviewer, existingReason, requestedBy string
	if err = tx.QueryRow(ctx, `SELECT state,COALESCE(decided_by::text,''),reason,requested_by::text FROM app.tradeline_drawdown_approvals WHERE id=$1::uuid`, id).Scan(&existingState, &reviewer, &existingReason, &requestedBy); err != nil {
		return err
	}
	if requestedBy == user {
		return ErrAuthority
	}
	if existingState != "pending" {
		if existingState == decision && reviewer == user && existingReason == reason {
			return nil
		}
		return ErrConflict
	}
	tag, err := tx.Exec(ctx, `UPDATE app.tradeline_drawdown_approvals SET state=$3,decided_by=$4::uuid,reason=$5,decided_at=now() WHERE organization_id=$1::uuid AND id=$2::uuid AND state='pending' AND requested_by<>$4::uuid AND fingerprint=$6`, org, id, decision, user, reason, fingerprint)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return tx.Commit(ctx)
}
