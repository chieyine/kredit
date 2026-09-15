package referrals

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"kredit/internal/access"
)

func (s *Store) Read(ctx context.Context, actor string, admin bool, agent string, cursors map[string]string) (map[string]any, error) {
	tx, e := s.begin(ctx, actor, "")
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	if admin {
		if e = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionPlatformOwner); e != nil {
			return nil, e
		}
	} else {
		agent = actor
	}
	out := map[string]any{}
	var rules Rules
	if e = tx.QueryRow(ctx, `SELECT to_jsonb(p) FROM app.dsa_program p WHERE id=1`).Scan(&rules); e != nil {
		return nil, e
	}
	out["rules"] = rules
	var self json.RawMessage
	e = tx.QueryRow(ctx, `SELECT to_jsonb(a) FROM app.dsa_agents a WHERE user_id=$1::uuid`, actor).Scan(&self)
	if errors.Is(e, pgx.ErrNoRows) {
		e = nil
	}
	if e != nil {
		return nil, e
	}
	out["self"] = self
	var wallet json.RawMessage
	e = tx.QueryRow(ctx, `SELECT jsonb_build_object('earned_kobo',COALESCE(sum(amount_kobo),0),'matured_kobo',COALESCE(sum(amount_kobo) FILTER(WHERE available_at<=now()),0),'paid_kobo',COALESCE((SELECT sum(amount_kobo) FROM app.dsa_payouts WHERE ($1='' OR agent_id=NULLIF($1,'')::uuid) AND state='paid'),0),'reserved_kobo',COALESCE((SELECT sum(amount_kobo) FROM app.dsa_payouts WHERE ($1='' OR agent_id=NULLIF($1,'')::uuid) AND state='pending'),0),'next_available_at',min(available_at) FILTER(WHERE available_at>now())) FROM app.dsa_earnings WHERE ($1='' OR agent_id=NULLIF($1,'')::uuid)`, agent).Scan(&wallet)
	if e != nil {
		return nil, e
	}
	out["wallet"] = wallet
	next := map[string]string{}
	queries := map[string]string{
		"agents":    `SELECT user_id AS id,user_id,code,name,phone,status,onboarding_limit,version,bank_name,account_name,account_number,bank_updated_at,(SELECT count(*) FROM app.dsa_referrals r WHERE r.agent_id=a.user_id) AS referrals,(SELECT count(*) FROM app.dsa_referrals r WHERE r.agent_id=a.user_id AND r.activated_at IS NOT NULL AND NOT blocked) AS activated FROM app.dsa_agents a WHERE ($1='' OR user_id=NULLIF($1,'')::uuid) AND user_id<$2::uuid ORDER BY user_id DESC LIMIT 101`,
		"referrals": `SELECT organization_id AS id,organization_id,agent_id,business_name,created_at,terms,qualified_at,activated_at,share_ends_at,onboarding_slot,blocked,progress,fees_kobo,checked_at,version FROM app.dsa_referrals WHERE ($1='' OR agent_id=NULLIF($1,'')::uuid) AND organization_id<$2::uuid ORDER BY organization_id DESC LIMIT 101`,
		"earnings":  `SELECT id,organization_id,agent_id,kind,amount_kobo,reason,available_at,created_at FROM app.dsa_earnings WHERE ($1='' OR agent_id=NULLIF($1,'')::uuid) AND id<$2::uuid ORDER BY id DESC LIMIT 101`,
		"payouts":   `SELECT id,agent_id,amount_kobo,bank_name,account_name,account_number,state,bank_reference,evidence,created_at,completed_at,version FROM app.dsa_payouts WHERE ($1='' OR agent_id=NULLIF($1,'')::uuid) AND id<$2::uuid ORDER BY id DESC LIMIT 101`,
	}
	for key, q := range queries {
		before := cursors[key]
		if before == "" {
			before = "ffffffff-ffff-ffff-ffff-ffffffffffff"
		}
		var raw []byte
		if e = tx.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(v)),'[]') FROM (`+q+`) v`, agent, before).Scan(&raw); e != nil {
			return nil, e
		}
		var list []map[string]any
		if e = json.Unmarshal(raw, &list); e != nil {
			return nil, e
		}
		if len(list) > 100 {
			list = list[:100]
			next[key] = list[99]["id"].(string)
		}
		out[key] = list
	}
	out["next"] = next
	return out, tx.Commit(ctx)
}
