package bankdebit

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"kredit/internal/access"
	"kredit/internal/mandates"
)

func (s *Store) Pending(ctx context.Context, actor string) ([]mandates.AuthorizationAttempt, error) {
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	if e = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionProviderOperations); e != nil {
		return nil, e
	}
	if _, e = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, actor); e != nil {
		return nil, e
	}
	rows, e := tx.Query(ctx, `SELECT provider,reference,input->>'BusinessID',created_at FROM app.bank_debit_enrollments WHERE state='STARTED' ORDER BY created_at LIMIT 100`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	result := []mandates.AuthorizationAttempt{}
	for rows.Next() {
		var p string
		var v mandates.AuthorizationAttempt
		if e = rows.Scan(&p, &v.Reference, &v.BusinessID, &v.CreatedAt); e != nil {
			return nil, e
		}
		v.ID = "native:" + p + ":" + v.Reference
		result = append(result, v)
	}
	return result, rows.Err()
}
func (s *Store) Review(ctx context.Context, actor, provider, ref string) (Enrollment, error) {
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return Enrollment{}, e
	}
	defer tx.Rollback(ctx)
	if e = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionProviderOperations); e != nil {
		return Enrollment{}, e
	}
	var owner string
	var started time.Time
	if e = tx.QueryRow(ctx, `SELECT buyer_user_id FROM app.payment_mandate_by_provider($1,$2)`, provider, ref).Scan(&owner); e != nil {
		return Enrollment{}, e
	}
	if _, e = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, owner); e != nil {
		return Enrollment{}, e
	}
	v, e := s.read(ctx, tx, provider, ref, true)
	if e != nil {
		return v, e
	}
	if e = tx.QueryRow(ctx, `SELECT updated_at FROM app.bank_debit_enrollments WHERE provider=$1 AND reference=$2`, provider, ref).Scan(&started); e != nil {
		return v, e
	}
	if v.State != "STARTED" || time.Since(started) < 2*time.Minute {
		return v, errors.New("wait for the original bank request to finish before reviewing it")
	}
	return v, tx.Commit(ctx)
}
func (s *Store) Resolve(ctx context.Context, actor string, v Enrollment, action string, result Result, reason string) error {
	if len(strings.TrimSpace(reason)) < 20 || len(reason) > 2000 || (action != "link" && action != "not_created") {
		return errors.New("provider evidence and a valid recovery action are required")
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if e = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionProviderOperations); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, v.Input.UserID); e != nil {
		return e
	}
	locked, e := s.read(ctx, tx, v.Provider, v.Reference, true)
	if e != nil {
		return e
	}
	if locked.State != "STARTED" || locked.Version != v.Version {
		return errors.New("bank authorization has already changed")
	}
	state := "CONFIRMED"
	if action == "not_created" {
		state = "DRAFT"
		result = Result{}
	} else if result.Reference == "" {
		return errors.New("verified provider reference required")
	}
	b, _ := json.Marshal(result)
	_, e = tx.Exec(ctx, `UPDATE app.bank_debit_enrollments SET state=$3,result=$4::jsonb,reviewed_by=$5::uuid,review_note=$6,version=version+1,updated_at=now() WHERE provider=$1 AND reference=$2`, v.Provider, v.Reference, state, b, actor, reason)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}

type Recoverer interface {
	Recover(context.Context, string, string, string, string, string) error
}
