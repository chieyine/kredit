package operations

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"

	"kredit/internal/businesspolicy"
	"kredit/internal/db"
	"kredit/internal/ledger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DateChange struct {
	ItemID string    `json:"item_id"`
	DueAt  time.Time `json:"due_at"`
}
type ChangeValues struct {
	Amount ledger.Money `json:"amount_kobo"`
	Dates  []DateChange `json:"dates"`
}
type ChangeProposal struct {
	ID           string       `json:"id"`
	ObligationID string       `json:"obligation_id"`
	Kind         string       `json:"kind"`
	Reason       string       `json:"reason"`
	Values       ChangeValues `json:"values"`
	ExpiresAt    time.Time    `json:"expires_at"`
}
type FinancialSnapshot struct {
	Outstanding ledger.Money    `json:"outstanding_kobo"`
	BaseFee     ledger.Money    `json:"base_fee_kobo"`
	Waived      ledger.Money    `json:"waived_kobo"`
	Accrued     ledger.Money    `json:"accrued_kobo"`
	Items       json.RawMessage `json:"items"`
}

func adminRole(ctx context.Context, tx pgx.Tx, actor string, roles ...string) error {
	var ok bool
	if err := tx.QueryRow(ctx, `SELECT app.has_admin_role($1::uuid,$2::text[])`, actor, roles).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return errors.New("required administrator permission is no longer active")
	}
	return nil
}
func financialSnapshot(ctx context.Context, tx pgx.Tx, obligation string, requireActive bool) (FinancialSnapshot, string, string, error) {
	var v FinancialSnapshot
	var org, buyer, status string
	if err := db.SetObligationContext(ctx, tx, obligation); err != nil {
		return v, org, buyer, err
	}
	err := tx.QueryRow(ctx, `SELECT o.outstanding_kobo,o.base_fee_kobo,o.supplier_organization_id::text,c.buyer_user_id::text,o.lifecycle_status FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id WHERE o.id=$1::uuid FOR UPDATE OF o`, obligation).Scan(&v.Outstanding, &v.BaseFee, &org, &buyer, &status)
	if err != nil {
		return v, org, buyer, err
	}
	if requireActive && status != "ACTIVE" {
		return v, org, buyer, errors.New("only an active obligation can be changed")
	}
	// Lock schedule rows before capturing allocations and dates. Every application
	// rechecks this snapshot after acquiring the obligation lock used by payments.
	rows, err := tx.Query(ctx, `SELECT i.id FROM app.schedule_items i JOIN app.repayment_schedules s ON s.id=i.schedule_id WHERE s.obligation_id=$1::uuid ORDER BY i.sequence FOR UPDATE OF i`, obligation)
	if err != nil {
		return v, org, buyer, err
	}
	for rows.Next() {
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return v, org, buyer, err
	}
	err = tx.QueryRow(ctx, `SELECT COALESCE((SELECT sum((metadata->>'amount_kobo')::bigint) FROM app.operation_actions WHERE resource_id=$1::uuid AND action='fee_waiver'),0),COALESCE((SELECT sum(amount_kobo) FROM app.fees WHERE obligation_id=$1::uuid AND state='accrued'),0),COALESCE((SELECT jsonb_agg(jsonb_build_object('id',i.id,'principal_due_kobo',i.principal_due_kobo,'allocated_kobo',i.allocated_kobo,'collected_kobo',i.collected_kobo,'disputed_kobo',i.disputed_kobo,'due_at',i.due_at,'grace_hours',i.grace_hours,'collection_at',i.collection_at,'cancelled',i.state='CANCELLED') ORDER BY i.sequence) FROM app.schedule_items i JOIN app.repayment_schedules s ON s.id=i.schedule_id WHERE s.obligation_id=$1::uuid),'[]'::jsonb)`, obligation).Scan(&v.Waived, &v.Accrued, &v.Items)
	return v, org, buyer, err
}
func validateChange(ctx context.Context, tx pgx.Tx, kind, obligation string, v ChangeValues, before FinancialSnapshot) error {
	switch kind {
	case "write_off":
		if v.Amount <= 0 || v.Amount > before.Outstanding || len(v.Dates) > 0 {
			return errors.New("write-off must be positive and within the outstanding balance")
		}
		return db.GuardUnreservedReduction(ctx, tx, obligation, int64(before.Outstanding-v.Amount))
	case "fee_waiver":
		total, err := ledger.CheckedAdd(before.BaseFee, before.Accrued)
		if err != nil {
			return err
		}
		if v.Amount <= 0 || before.Waived > total || v.Amount > total-before.Waived || len(v.Dates) > 0 {
			return errors.New("waiver must be within accrued, unwaived fees")
		}
	case "schedule_amendment":
		if v.Amount != 0 || len(v.Dates) == 0 || len(v.Dates) > 60 {
			return errors.New("provide one date for each unpaid instalment")
		}
		if err := db.GuardUnreservedReduction(ctx, tx, obligation, 0); err != nil {
			return err
		}
		var items []struct {
			ID        string    `json:"id"`
			Principal int64     `json:"principal_due_kobo"`
			Allocated int64     `json:"allocated_kobo"`
			Disputed  int64     `json:"disputed_kobo"`
			Cancelled bool      `json:"cancelled"`
			Due       time.Time `json:"due_at"`
		}
		if err := json.Unmarshal(before.Items, &items); err != nil {
			return err
		}
		p, err := businesspolicy.ReadTx(ctx, tx)
		if err != nil {
			return err
		}
		var now time.Time
		if err = tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
			return err
		}
		seen := map[string]bool{}
		dates := map[string]time.Time{}
		for _, d := range v.Dates {
			if seen[d.ItemID] || !d.DueAt.After(now.Add(time.Duration(p.Values.NoticeHours)*time.Hour)) || d.DueAt.After(now.AddDate(5, 0, 0)) {
				return errors.New("dates must be unique, allow the notice period, and be within five years")
			}
			seen[d.ItemID] = true
			dates[d.ItemID] = d.DueAt
		}
		var previous time.Time
		count := 0
		changed := false
		for _, i := range items {
			if i.Cancelled || i.Allocated == i.Principal {
				continue
			}
			if i.Disputed > 0 {
				return errors.New("resolve disputed instalments before changing dates")
			}
			d, ok := dates[i.ID]
			if !ok || !d.After(previous) {
				return errors.New("include every unpaid instalment in its existing order")
			}
			changed = changed || !d.Equal(i.Due)
			previous = d
			count++
		}
		if count != len(v.Dates) || !changed {
			return errors.New("provide a changed date for the existing unpaid schedule")
		}
	default:
		return errors.New("unsupported financial change")
	}
	return nil
}
func (s *PostgresStore) ProposeChange(ctx context.Context, actor string, in ChangeProposal) error {
	if _, err := uuid.Parse(in.ID); err != nil {
		return errors.New("a unique proposal ID is required")
	}
	if _, err := uuid.Parse(in.ObligationID); err != nil {
		return errors.New("a valid obligation is required")
	}
	in.Reason = strings.TrimSpace(in.Reason)
	if len(in.Reason) < 8 || len(in.Reason) > 2000 {
		return errors.New("provide a reason between 8 and 2000 characters")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = adminRole(ctx, tx, actor, "platform_admin", "finance_operator"); err != nil {
		return err
	}
	before, org, buyer, err := financialSnapshot(ctx, tx, in.ObligationID, true)
	if err != nil {
		return err
	}
	var same bool
	err = tx.QueryRow(ctx, `SELECT proposed_by=$2::uuid AND obligation_id=$3::uuid AND kind=$4 AND reason=$5 AND proposed_values=$6::jsonb AND expires_at=$7 FROM app.admin_change_requests WHERE id=$1::uuid`, in.ID, actor, in.ObligationID, in.Kind, in.Reason, mustJSON(in.Values), in.ExpiresAt).Scan(&same)
	if err == nil {
		if !same {
			return errors.New("proposal ID was reused with different details")
		}
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	var now time.Time
	if err = tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		return err
	}
	if !in.ExpiresAt.After(now) || in.ExpiresAt.After(now.Add(30*24*time.Hour)) {
		return errors.New("expiry must be within the next 30 days")
	}
	if err = validateChange(ctx, tx, in.Kind, in.ObligationID, in.Values, before); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO app.admin_change_requests(id,kind,obligation_id,organization_id,buyer_id,proposed_by,reason,before_values,proposed_values,expires_at) VALUES($1::uuid,$2,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7,$8::jsonb,$9::jsonb,$10)`, in.ID, in.Kind, in.ObligationID, org, buyer, actor, in.Reason, mustJSON(before), mustJSON(in.Values), in.ExpiresAt)
	if err != nil {
		return err
	}
	if err = changeEvent(ctx, tx, in.ID, actor, "proposed", in.Reason); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
func changeEvent(ctx context.Context, tx pgx.Tx, id, actor, action, reason string) error {
	_, err := tx.Exec(ctx, `INSERT INTO app.admin_change_events(change_id,actor_id,action,reason) VALUES($1::uuid,$2::uuid,$3,$4)`, id, actor, action, reason)
	return err
}

func (s *PostgresStore) DecideChange(ctx context.Context, id, actor, decision, reason string, buyerDecision bool) error {
	if _, err := uuid.Parse(id); err != nil {
		return errors.New("invalid proposal")
	}
	reason = strings.TrimSpace(reason)
	if len(reason) < 8 || len(reason) > 2000 {
		return errors.New("provide decision notes between 8 and 2000 characters")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Determine identity without returning private contents, then use consistent
	// obligation -> schedule -> proposal locking across all financial mutations.
	var obligation, buyer string
	if err = tx.QueryRow(ctx, `SELECT obligation_id::text,buyer_id::text FROM app.admin_change_requests WHERE id=$1::uuid`, id).Scan(&obligation, &buyer); err != nil {
		return err
	}
	if buyerDecision {
		if actor != buyer {
			return errors.New("proposal not found")
		}
	} else {
		roles := []string{"platform_admin", "approver"}
		if decision == "cancel" {
			roles = append(roles, "finance_operator")
		}
		if err = adminRole(ctx, tx, actor, roles...); err != nil {
			return err
		}
	}
	before, org, _, err := financialSnapshot(ctx, tx, obligation, decision == "approve" || decision == "accept")
	if err != nil {
		return err
	}
	var kind, author, state, proposalReason string
	var approver *string
	var old, values []byte
	var expires, now time.Time
	err = tx.QueryRow(ctx, `SELECT kind,proposed_by::text,state,reason,approved_by::text,before_values,proposed_values,expires_at,clock_timestamp() FROM app.admin_change_requests WHERE id=$1::uuid FOR UPDATE`, id).Scan(&kind, &author, &state, &proposalReason, &approver, &old, &values, &expires, &now)
	if err != nil {
		return err
	}
	target := ""
	if buyerDecision {
		if kind != "schedule_amendment" || state != "awaiting_buyer" {
			return errors.New("this change is not awaiting your consent")
		}
		switch decision {
		case "accept":
			target = "applied"
		case "reject":
			target = "rejected"
		default:
			return errors.New("accept or reject the exact proposed dates")
		}
	} else {
		switch decision {
		case "approve":
			if state != "pending" || author == actor {
				return errors.New("another authorized administrator must approve a pending proposal")
			}
			target = "applied"
			if kind == "schedule_amendment" {
				target = "awaiting_buyer"
			}
		case "reject":
			if state != "pending" {
				return errors.New("proposal is no longer pending")
			}
			target = "rejected"
		case "cancel":
			if state != "pending" && state != "awaiting_buyer" {
				return errors.New("proposal is already closed")
			}
			if author != actor {
				if err = adminRole(ctx, tx, actor, "platform_admin", "approver"); err != nil {
					return err
				}
			}
			target = "cancelled"
		default:
			return errors.New("unsupported decision")
		}
	}
	if target == "applied" || target == "awaiting_buyer" {
		if !buyerDecision {
			var conflict bool
			if err = tx.QueryRow(ctx, `SELECT $1::uuid=$2::uuid OR EXISTS(SELECT 1 FROM app.memberships WHERE user_id=$1::uuid AND organization_id=$3::uuid AND status='active')`, actor, buyer, org).Scan(&conflict); err != nil {
				return err
			}
			if conflict {
				return errors.New("the independent reviewer must not be the buyer or a member of the supplier business")
			}
		}
		if !expires.After(now) {
			return errors.New("proposal expired; cancel and submit a new proposal")
		}
		if err = adminRole(ctx, tx, author, "platform_admin", "finance_operator"); err != nil {
			return err
		}
		if buyerDecision {
			if approver == nil {
				return errors.New("independent approval is missing")
			}
			if err = adminRole(ctx, tx, *approver, "platform_admin", "approver"); err != nil {
				return err
			}
		}
		var stored FinancialSnapshot
		var v ChangeValues
		if err = json.Unmarshal(old, &stored); err != nil {
			return err
		}
		if err = json.Unmarshal(values, &v); err != nil {
			return err
		}
		if !reflect.DeepEqual(stored, before) {
			return errors.New("balances or schedule changed; cancel and propose against current details")
		}
		if err = validateChange(ctx, tx, kind, obligation, v, before); err != nil {
			return err
		}
		if target == "applied" {
			if kind == "schedule_amendment" {
				for _, d := range v.Dates {
					_, err = tx.Exec(ctx, `UPDATE app.schedule_items SET due_at=$2::timestamptz,collection_at=$2::timestamptz+make_interval(hours=>grace_hours),state=CASE WHEN allocated_kobo>0 THEN 'PARTIALLY_PAID' ELSE 'OPEN' END WHERE id=$1::uuid`, d.ItemID, d.DueAt)
					if err != nil {
						return err
					}
				}
			} else {
				if _, err = s.adjustTx(ctx, tx, author, org, obligation, v.Amount, proposalReason, actor, kind, id, true); err != nil {
					return err
				}
			}
		}
	}
	if !buyerDecision && decision == "approve" {
		approver = &actor
	}
	_, err = tx.Exec(ctx, `UPDATE app.admin_change_requests SET state=$2,approved_by=$3::uuid,buyer_decided_by=CASE WHEN $4 THEN $5::uuid ELSE buyer_decided_by END,decided_at=clock_timestamp() WHERE id=$1::uuid`, id, target, approver, buyerDecision, actor)
	if err != nil {
		return err
	}
	if err = changeEvent(ctx, tx, id, actor, decision, reason); err != nil {
		return err
	}
	if kind == "schedule_amendment" && (target == "awaiting_buyer" || target == "applied") {
		_, err = tx.Exec(ctx, `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key) VALUES('obligation',$1,'notification.requested',jsonb_build_object('event','SCHEDULE_AMENDMENT','change_id',$2::text),$3)`, obligation, id, "schedule-amendment:"+id+":"+target)
		if err != nil {
			return err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	if target == "applied" && s.invalidate != nil {
		s.invalidate(obligation)
	}
	return nil
}

func (s *PostgresStore) ChangeContext(ctx context.Context, id string) (FinancialSnapshot, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return FinancialSnapshot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	v, _, _, err := financialSnapshot(ctx, tx, id, true)
	return v, err
}
