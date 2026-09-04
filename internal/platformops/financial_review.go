package platformops

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// RefreshFinancialReviews records discrepancies without changing balances or
// inventing provider settlements. Reappearing discrepancies reopen the case.
func (s *Store) RefreshFinancialReviews(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('financial-review-refresh',0))`); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `INSERT INTO app.financial_review_cases(kind,target_id,expected,actual)
 SELECT kind,target_id,expected,actual FROM app.financial_discrepancies
 ON CONFLICT(kind,target_id) DO UPDATE SET expected=EXCLUDED.expected,actual=EXCLUDED.actual,last_seen_at=now(),state='OPEN',resolved_at=NULL
 WHERE app.financial_review_cases.state='RESOLVED' OR app.financial_review_cases.expected<>EXCLUDED.expected OR app.financial_review_cases.actual<>EXCLUDED.actual
 RETURNING id::text`)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err = tx.Exec(ctx, `INSERT INTO app.financial_review_events(case_id,action,reason) VALUES($1::uuid,'DETECTED','Durable financial records disagree')`, id); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE app.financial_review_cases c SET last_seen_at=now() WHERE state='OPEN' AND EXISTS(SELECT 1 FROM app.financial_discrepancies d WHERE d.kind=c.kind AND d.target_id=c.target_id)`)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) FinancialReviews(ctx context.Context) (json.RawMessage, error) {
	var result []byte
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(c)||jsonb_build_object('expected',c.expected::text,'actual',c.actual::text,'history',(SELECT COALESCE(jsonb_agg(to_jsonb(e) ORDER BY e.occurred_at,e.id),'[]'::jsonb) FROM app.financial_review_events e WHERE e.case_id=c.id)) ORDER BY c.first_seen_at),'[]'::jsonb) FROM app.financial_review_cases c WHERE c.state='OPEN'`).Scan(&result)
	return result, err
}

// Resolve requires current evidence to agree, an assigned owner, and a written
// reason. This endpoint never accepts a replacement balance or settlement.
func (s *Store) DecideFinancialReview(ctx context.Context, id, actor, action, reason string) error {
	if action != "claim" && action != "resolve" {
		return errors.New("action must be claim or resolve")
	}
	if len(strings.TrimSpace(reason)) < 8 || len(reason) > 2000 {
		return errors.New("reason must contain 8 to 2000 characters")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('financial-review-refresh',0))`); err != nil {
		return err
	}
	var owner, kind, target, state string
	if err = tx.QueryRow(ctx, `SELECT COALESCE(owner_id::text,''),kind,target_id,state FROM app.financial_review_cases WHERE id=$1::uuid FOR UPDATE`, id).Scan(&owner, &kind, &target, &state); err != nil {
		return err
	}
	if state != "OPEN" {
		return errors.New("case is already resolved")
	}
	if owner != "" && owner != actor {
		return errors.New("case is assigned to another reviewer")
	}
	if action == "resolve" {
		if owner != actor {
			return errors.New("claim this case before resolving it")
		}
		var mismatch bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.financial_discrepancies WHERE kind=$1 AND target_id=$2)`, kind, target).Scan(&mismatch); err != nil {
			return err
		}
		if mismatch {
			return errors.New("financial discrepancy remains unresolved")
		}
		_, err = tx.Exec(ctx, `UPDATE app.financial_review_cases SET state='RESOLVED',resolved_at=now() WHERE id=$1::uuid`, id)
	} else {
		_, err = tx.Exec(ctx, `UPDATE app.financial_review_cases SET owner_id=$2::uuid WHERE id=$1::uuid`, id, actor)
	}
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.financial_review_events(case_id,actor_id,action,reason) VALUES($1::uuid,$2::uuid,$3,$4)`, id, actor, strings.ToUpper(action), strings.TrimSpace(reason)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
