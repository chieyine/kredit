package referrals

import (
	"context"
	"errors"
	"kredit/internal/db"
	"kredit/internal/ledger"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type facts struct {
	Registration string `json:"registration"`
	Ready        bool   `json:"ready"`
	Accepted     bool   `json:"accepted"`
	Self         bool   `json:"self_referral"`
	Fees         int64  `json:"fees_kobo"`
}

func (s *Store) Refresh(ctx context.Context) error {
	// Cursor pages avoid starvation and release the connection before transactions.
	cursor := "00000000-0000-0000-0000-000000000000"
	for {
		rows, e := s.Pool.Query(ctx, `SELECT organization_id::text FROM app.dsa_referrals WHERE organization_id>$1::uuid ORDER BY organization_id LIMIT 100`, cursor)
		if e != nil {
			return e
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				break
			}
			ids = append(ids, id)
		}
		if e == nil {
			e = rows.Err()
		}
		rows.Close()
		if e != nil {
			return e
		}
		for _, id := range ids {
			tx, e := s.begin(ctx, "", "")
			if e != nil {
				return e
			}
			e = s.refreshOne(ctx, tx, id)
			if e == nil {
				e = tx.Commit(ctx)
			}
			if e != nil {
				return db.RollbackFailure(ctx, tx, e)
			}
			cursor = id
		}
		if len(ids) < 100 {
			return nil
		}
	}
}
func (s *Store) refreshOne(ctx context.Context, tx pgx.Tx, id string) error {
	var agent, status string
	var limit int
	if e := tx.QueryRow(ctx, `SELECT a.user_id::text,a.status,a.onboarding_limit FROM app.dsa_agents a JOIN app.dsa_referrals r ON r.agent_id=a.user_id WHERE r.organization_id=$1::uuid FOR UPDATE OF a`, id).Scan(&agent, &status, &limit); e != nil {
		return e
	}
	var rules Rules
	var start time.Time
	var qualified, activated, ends *time.Time
	var baseline int64
	var slot, blocked bool
	var registration *string
	if e := tx.QueryRow(ctx, `SELECT terms,created_at,qualified_at,activated_at,share_ends_at,activation_baseline_kobo,onboarding_slot,blocked,registration_key FROM app.dsa_referrals WHERE organization_id=$1::uuid FOR UPDATE`, id).Scan(&rules, &start, &qualified, &activated, &ends, &baseline, &slot, &blocked, &registration); e != nil {
		return e
	}
	cutoff := time.Now().UTC()
	if ends != nil && ends.Before(cutoff) {
		cutoff = *ends
	}
	var f facts
	if e := tx.QueryRow(ctx, `SELECT app.dsa_facts($1::uuid,$2,$3)`, id, start, cutoff).Scan(&f); e != nil {
		return e
	}
	progress := "Complete business verification and bank setup"
	if f.Self {
		blocked = true
		progress = "Self-referral detected"
	}
	if !blocked && qualified == nil && f.Ready && len(f.Registration) >= 3 {
		// Serialize the same CAC entity across agents without revealing other businesses.
		if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('dsa-cac:'||$1,0))`, f.Registration); e != nil {
			return e
		}
		var duplicate bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.dsa_referrals WHERE registration_key=$1 AND organization_id<>$2::uuid)`, f.Registration, id).Scan(&duplicate); e != nil {
			return e
		}
		if duplicate {
			blocked = true
			progress = "Duplicate CAC registration needs review"
		} else {
			t := time.Now().UTC()
			qualified = &t
			registration = &f.Registration
			if _, e := tx.Exec(ctx, `UPDATE app.dsa_referrals SET registration_key=$2,qualified_at=$3 WHERE organization_id=$1::uuid`, id, f.Registration, t); e != nil {
				return e
			}
		}
	}
	if registration != nil && *registration != f.Registration {
		blocked = true
		progress = "Business registration changed; review required"
	}
	if !blocked && qualified != nil && !slot && status == "active" {
		var used int
		if e := tx.QueryRow(ctx, `SELECT count(*) FROM app.dsa_referrals WHERE agent_id=$1::uuid AND onboarding_slot`, agent).Scan(&used); e != nil {
			return e
		}
		if used < limit {
			slot = true
		} else {
			progress = "Verified; onboarding reward limit reached"
		}
	}
	if !blocked && qualified != nil && activated == nil && f.Accepted && f.Fees >= rules.Threshold && status == "active" {
		t := time.Now().UTC()
		activated = &t
		month := time.Date(t.Year(), t.Month()+time.Month(rules.ShareMonths), 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
		last := month.AddDate(0, 1, -1).Day()
		end := month.AddDate(0, 0, min(t.Day(), last)-1)
		ends = &end
		baseline = f.Fees
	}
	targets := map[string]int64{"onboarding": 0, "activation": 0, "share": 0}
	if !blocked && qualified != nil {
		if slot {
			targets["onboarding"] = rules.Onboarding
			progress = "Verified onboarding; awaiting paying activation"
		}
		if activated != nil && f.Accepted && f.Fees >= rules.Threshold {
			targets["activation"] = rules.Activation
			targets["share"] = (max(int64(0), f.Fees-baseline)/10000)*rules.ShareBPS + (max(int64(0), f.Fees-baseline)%10000)*rules.ShareBPS/10000
			progress = "Activated; earning fee share"
			if ends != nil && time.Now().After(*ends) {
				progress = "Fee-share earning window completed"
			}
		}
	}
	if blocked && progress == "Complete business verification and bank setup" {
		progress = "Referral restricted; rewards reversed"
	}
	for _, kind := range []string{"onboarding", "activation", "share"} {
		var previous int64
		if e := tx.QueryRow(ctx, `SELECT COALESCE(sum(amount_kobo),0) FROM app.dsa_earnings WHERE organization_id=$1::uuid AND kind=$2`, id, kind).Scan(&previous); e != nil {
			return e
		}
		delta := targets[kind] - previous
		if delta == 0 {
			continue
		}
		var earning string
		var at time.Time
		// Monday payout date after a full seven-day hold, calculated in Lagos time.
		e := tx.QueryRow(ctx, `INSERT INTO app.dsa_earnings(organization_id,agent_id,kind,amount_kobo,reason,available_at) VALUES($1::uuid,$2::uuid,$3,$4,$5,CASE WHEN $4::bigint<0 THEN now() ELSE (date_trunc('week',(now() AT TIME ZONE 'Africa/Lagos')+interval '7 days')+interval '7 days') AT TIME ZONE 'Africa/Lagos' END) RETURNING id::text,created_at`, id, agent, kind, delta, progress).Scan(&earning, &at)
		if e != nil {
			return e
		}
		if e = ledger.NewPostgresStore(nil).PostDSATx(ctx, tx, earning, "earning", delta, at); e != nil {
			return e
		}
	}
	_, e := tx.Exec(ctx, `UPDATE app.dsa_referrals SET activated_at=$2,share_ends_at=$3,activation_baseline_kobo=$4,onboarding_slot=$5,blocked=$6,progress=$7,fees_kobo=$8,checked_at=now(),version=version+CASE WHEN ROW(activated_at,share_ends_at,activation_baseline_kobo,onboarding_slot,blocked,progress,fees_kobo) IS DISTINCT FROM ROW($2::timestamptz,$3::timestamptz,$4::bigint,$5::boolean,$6::boolean,$7::text,$8::bigint) THEN 1 ELSE 0 END WHERE organization_id=$1::uuid`, id, activated, ends, baseline, slot, blocked, progress, f.Fees)
	return e
}
func (s *Store) refreshAgent(ctx context.Context, tx pgx.Tx, agent string) error {
	if _, e := tx.Exec(ctx, `SELECT user_id FROM app.dsa_agents WHERE user_id=$1::uuid FOR UPDATE`, agent); e != nil {
		return e
	}
	rows, e := tx.Query(ctx, `SELECT organization_id::text FROM app.dsa_referrals WHERE agent_id=$1::uuid ORDER BY organization_id`, agent)
	if e != nil {
		return e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			break
		}
		ids = append(ids, id)
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		return e
	}
	for _, id := range ids {
		if e = s.refreshOne(ctx, tx, id); e != nil {
			return e
		}
	}
	return nil
}
func (s *Store) prepare(ctx context.Context, tx pgx.Tx, actor, agent string) (string, error) {
	if e := s.refreshAgent(ctx, tx, agent); e != nil {
		return "", e
	}
	var status string
	var updated time.Time
	if e := tx.QueryRow(ctx, `SELECT status,bank_updated_at FROM app.dsa_agents WHERE user_id=$1::uuid FOR UPDATE`, agent).Scan(&status, &updated); e != nil {
		return "", e
	}
	if status != "active" || time.Since(updated) < 7*24*time.Hour {
		return "", errors.New("payouts require an active agent and bank details unchanged for seven days")
	}
	var active bool
	if e := tx.QueryRow(ctx, `SELECT app.dsa_agent_active($1::uuid)`, agent).Scan(&active); e != nil || !active {
		return "", errors.New("agent account is unavailable")
	}
	var available int64
	if e := tx.QueryRow(ctx, `SELECT LEAST(COALESCE(sum(amount_kobo),0),COALESCE(sum(amount_kobo) FILTER(WHERE available_at<=now()),0))-COALESCE((SELECT sum(amount_kobo) FROM app.dsa_payouts WHERE agent_id=$1::uuid AND state IN ('pending','paid')),0) FROM app.dsa_earnings WHERE agent_id=$1::uuid`, agent).Scan(&available); e != nil {
		return "", e
	}
	if available <= 0 {
		return "", errors.New("no matured, unpaid rewards are available")
	}
	var id string
	e := tx.QueryRow(ctx, `INSERT INTO app.dsa_payouts(agent_id,amount_kobo,bank_name,account_name,account_number,bank_updated_at,created_by) SELECT user_id,$2,bank_name,account_name,account_number,bank_updated_at,$3::uuid FROM app.dsa_agents WHERE user_id=$1::uuid RETURNING id::text`, agent, available, actor).Scan(&id)
	return id, e
}
func (s *Store) finishPayout(ctx context.Context, tx pgx.Tx, actor string, in Input) error {
	var agent string
	if e := tx.QueryRow(ctx, `SELECT agent_id::text FROM app.dsa_payouts WHERE id=$1::uuid`, in.ID).Scan(&agent); e != nil {
		return e
	}
	if e := s.refreshAgent(ctx, tx, agent); e != nil {
		return e
	}
	var state string
	var amount, version int64
	if e := tx.QueryRow(ctx, `SELECT state,amount_kobo,version FROM app.dsa_payouts WHERE id=$1::uuid FOR UPDATE`, in.ID).Scan(&state, &amount, &version); e != nil {
		return e
	}
	if version != in.Version || state != "pending" {
		return errors.New("payout changed; refresh before acting")
	}
	next := "cancelled"
	var ref any
	if in.Action == "paid" {
		next = "paid"
		r := strings.TrimSpace(in.Reference)
		if len(r) < 3 || len(r) > 200 {
			return errors.New("enter the unique bank reference for the completed transfer")
		}
		ref = r
	}
	_, e := tx.Exec(ctx, `UPDATE app.dsa_payouts SET state=$2,bank_reference=$3,evidence=$4,completed_by=$5::uuid,completed_at=now(),version=version+1 WHERE id=$1::uuid`, in.ID, next, ref, in.Reason, actor)
	if e != nil {
		return e
	}
	// This records an already completed transfer. Later reversals remain a visible
	// negative agent balance; never hide actual money sent by refusing reconciliation.
	if next == "paid" {
		return ledger.NewPostgresStore(nil).PostDSATx(ctx, tx, in.ID, "payout", amount, time.Now())
	}
	return nil
}
