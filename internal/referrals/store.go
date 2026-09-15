package referrals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"kredit/internal/access"
)

type Store struct{ Pool *pgxpool.Pool }
type Rules struct {
	Version      int64 `json:"version"`
	Enabled      bool  `json:"enabled"`
	Onboarding   int64 `json:"onboarding_kobo"`
	Activation   int64 `json:"activation_kobo"`
	Threshold    int64 `json:"threshold_kobo"`
	ShareBPS     int64 `json:"share_bps"`
	ShareMonths  int   `json:"share_months"`
	InitialLimit int   `json:"initial_limit"`
}
type Input struct {
	Action      string `json:"action"`
	ID          string `json:"id"`
	Version     int64  `json:"version"`
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Bank        string `json:"bank_name"`
	AccountName string `json:"account_name"`
	Account     string `json:"account_number"`
	Consent     bool   `json:"consent"`
	Code        string `json:"code"`
	Status      string `json:"status"`
	Limit       int    `json:"onboarding_limit"`
	Blocked     bool   `json:"blocked"`
	Reason      string `json:"reason"`
	Reference   string `json:"bank_reference"`
	Rules       Rules  `json:"rules"`
}

func (s *Store) begin(ctx context.Context, actor, org string) (pgx.Tx, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return nil, e
	}
	_, e = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, actor, org)
	if e != nil {
		tx.Rollback(ctx)
		return nil, e
	}
	return tx, nil
}
func audit(ctx context.Context, tx pgx.Tx, actor, action, id string, in any) error {
	raw, e := json.Marshal(in)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO app.audit_events(actor_user_id,action,resource_type,resource_id,outcome,severity,metadata) VALUES(NULLIF($1,'')::uuid,$2,'dsa',$3,'success','high',$4::jsonb)`, actor, action, id, raw)
	return e
}
func (s *Store) Claim(ctx context.Context, actor, org, code string) error {
	tx, e := s.begin(ctx, actor, org)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	_, e = tx.Exec(ctx, `SELECT app.dsa_claim($1::uuid,$2)`, org, strings.ToUpper(strings.TrimSpace(code)))
	if e != nil {
		return errors.New("Referral could not be confirmed. Use an active code for a new business you own, before its first accepted sale. Each business can have only one referrer.")
	}
	return tx.Commit(ctx)
}
func (s *Store) Lookup(ctx context.Context, code string) (json.RawMessage, error) {
	var v json.RawMessage
	e := s.Pool.QueryRow(ctx, `SELECT app.dsa_code($1)`, strings.ToUpper(strings.TrimSpace(code))).Scan(&v)
	if e == nil && len(v) == 0 {
		e = errors.New("Referral code is unavailable.")
	}
	return v, e
}
func validBank(in Input) bool {
	return len(strings.TrimSpace(in.Name)) >= 3 && len(in.Name) <= 120 && regexp.MustCompile(`^\+234[789][01][0-9]{8}$`).MatchString(in.Phone) && len(strings.TrimSpace(in.Bank)) >= 3 && len(in.Bank) <= 120 && len(strings.TrimSpace(in.AccountName)) >= 3 && len(in.AccountName) <= 120 && regexp.MustCompile(`^[0-9]{10}$`).MatchString(in.Account)
}
func (s *Store) Act(ctx context.Context, actor string, admin bool, in Input) (any, error) {
	tx, e := s.begin(ctx, actor, "")
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	if admin {
		if e = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionPlatformOwner); e != nil {
			return nil, e
		}
	}
	if !admin {
		if in.Action != "enrol" && in.Action != "bank" {
			return nil, errors.New("This action requires Super Admin.")
		}
		if !validBank(in) || !in.Consent {
			return nil, errors.New("Enter your name, WhatsApp number and bank details, and accept the programme terms.")
		}
		if in.Action == "enrol" {
			if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('dsa-program',0))`); e != nil {
				return nil, e
			}
			var rules Rules
			if e = tx.QueryRow(ctx, `SELECT to_jsonb(p) FROM app.dsa_program p WHERE id=1`).Scan(&rules); e != nil {
				return nil, e
			}
			if !rules.Enabled {
				return nil, errors.New("New agent enrolment is paused.")
			}
			if in.Version != rules.Version {
				return nil, errors.New("Programme terms changed. Refresh and review them before joining.")
			}
			_, e = tx.Exec(ctx, `INSERT INTO app.dsa_agents(user_id,code,name,phone,bank_name,account_name,account_number,onboarding_limit,terms_version) VALUES($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(user_id) DO NOTHING`, actor, "DSA-"+strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", "")[:16]), strings.TrimSpace(in.Name), in.Phone, strings.TrimSpace(in.Bank), strings.TrimSpace(in.AccountName), in.Account, rules.InitialLimit, fmt.Sprintf("dsa-v1:rules-%d", rules.Version))
		} else {
			if _, e = tx.Exec(ctx, `SELECT user_id FROM app.dsa_agents WHERE user_id=$1::uuid FOR UPDATE`, actor); e != nil {
				return nil, e
			}
			var tag interface{ RowsAffected() int64 }
			tag, e = tx.Exec(ctx, `UPDATE app.dsa_agents SET name=$2,phone=$3,bank_name=$4,account_name=$5,account_number=$6,bank_updated_at=now(),version=version+1 WHERE user_id=$1::uuid AND version=$7 AND NOT EXISTS(SELECT 1 FROM app.dsa_payouts WHERE agent_id=$1::uuid AND state='pending')`, actor, strings.TrimSpace(in.Name), in.Phone, strings.TrimSpace(in.Bank), strings.TrimSpace(in.AccountName), in.Account, in.Version)
			if e == nil && tag.RowsAffected() != 1 {
				return nil, errors.New("Refresh your details. A pending payout must be resolved before changing its destination.")
			}
		}
		if e != nil {
			return nil, e
		}
		e = audit(ctx, tx, actor, "dsa."+in.Action, actor, map[string]any{"terms_accepted": true, "account_last4": in.Account[6:]})
	} else {
		if len(strings.TrimSpace(in.Reason)) < 20 || len(in.Reason) > 2000 {
			return nil, errors.New("Record a clear reason or completed bank evidence (20–2,000 characters).")
		}
		switch in.Action {
		case "rules":
			if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('dsa-program',0))`); e != nil {
				return nil, e
			}
			r := in.Rules
			if r.Onboarding < 0 || r.Onboarding > 1000000 || r.Activation < 0 || r.Activation > 10000000 || r.Threshold < 10000 || r.Threshold > 100000000 || r.ShareBPS < 0 || r.ShareBPS > 5000 || r.ShareMonths < 1 || r.ShareMonths > 12 || r.InitialLimit < 1 || r.InitialLimit > 1000 {
				return nil, errors.New("Reward settings are outside the allowed limits.")
			}
			tag, err := tx.Exec(ctx, `UPDATE app.dsa_program SET enabled=$1,onboarding_kobo=$2,activation_kobo=$3,threshold_kobo=$4,share_bps=$5,share_months=$6,initial_limit=$7,version=version+1 WHERE id=1 AND version=$8`, r.Enabled, r.Onboarding, r.Activation, r.Threshold, r.ShareBPS, r.ShareMonths, r.InitialLimit, r.Version)
			e = err
			if e == nil && tag.RowsAffected() != 1 {
				return nil, errors.New("Programme settings changed. Refresh before saving.")
			}
		case "agent":
			if in.Status != "active" && in.Status != "suspended" || in.Limit < 0 || in.Limit > 100000 {
				return nil, errors.New("Choose a valid agent status and onboarding reward limit.")
			}
			tag, err := tx.Exec(ctx, `UPDATE app.dsa_agents SET status=$2,onboarding_limit=$3,version=version+1 WHERE user_id=$1::uuid AND version=$4`, in.ID, in.Status, in.Limit, in.Version)
			e = err
			if e == nil && tag.RowsAffected() != 1 {
				return nil, errors.New("Agent details changed. Refresh before saving.")
			}
		case "referral":
			// Lock the agent before the referral, matching the reward and payout engine.
			_, e = tx.Exec(ctx, `SELECT a.user_id FROM app.dsa_agents a JOIN app.dsa_referrals r ON r.agent_id=a.user_id WHERE r.organization_id=$1::uuid FOR UPDATE OF a`, in.ID)
			if e != nil {
				return nil, e
			}
			tag, err := tx.Exec(ctx, `UPDATE app.dsa_referrals SET blocked=$2,reason=$3,version=version+1 WHERE organization_id=$1::uuid AND version=$4`, in.ID, in.Blocked, in.Reason, in.Version)
			e = err
			if e == nil && tag.RowsAffected() != 1 {
				return nil, errors.New("Referral changed. Refresh before saving.")
			}
			if e == nil {
				e = s.refreshOne(ctx, tx, in.ID)
			}
		case "prepare":
			var id string
			id, e = s.prepare(ctx, tx, actor, in.ID)
			if e == nil {
				e = audit(ctx, tx, actor, "dsa.payout.prepared", id, map[string]string{"reason": in.Reason})
			}
			if e == nil {
				e = tx.Commit(ctx)
			}
			return map[string]string{"id": id}, e
		case "paid", "cancel":
			e = s.finishPayout(ctx, tx, actor, in)
		default:
			return nil, errors.New("Choose a supported DSA action.")
		}
		if e == nil {
			e = audit(ctx, tx, actor, "dsa."+in.Action, in.ID, map[string]any{"reason": in.Reason, "status": in.Status, "limit": in.Limit, "blocked": in.Blocked, "rules": in.Rules})
		}
	}
	if e != nil {
		return nil, e
	}
	return map[string]bool{"saved": true}, tx.Commit(ctx)
}
