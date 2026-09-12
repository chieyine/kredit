package buyers

import (
	"context"
	"errors"
	"kredit/internal/access"
	"kredit/internal/identity"
	"strings"
	"time"
)

type VerificationAttempt struct {
	ID          string    `json:"id"`
	SubjectID   string    `json:"subject_id"`
	SubjectType string    `json:"subject_type"`
	Provider    string    `json:"provider"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *PostgresStore) ListVerificationAttempts(ctx context.Context, actor string) ([]VerificationAttempt, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionProviderOperations); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text,subject_id::text,subject_type,provider,created_at FROM app.buyer_verification_intents WHERE state='STARTED' ORDER BY created_at,id LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []VerificationAttempt{}
	for rows.Next() {
		var item VerificationAttempt
		if err = rows.Scan(&item.ID, &item.SubjectID, &item.SubjectType, &item.Provider, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (s *PostgresStore) ResolveVerificationAttempt(ctx context.Context, actor, id, action, reference, reason string) error {
	reference = strings.TrimSpace(reference)
	if action == "link" && (reference == "" || len(reference) > 128) {
		return errors.New("a valid provider reference is required")
	}
	if len(strings.TrimSpace(reason)) < 20 || len(reason) > 2000 || (action != "link" && action != "not_created") {
		return errors.New("record the provider evidence and choose a valid action")
	}
	var subject, kind, provider, user string
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionProviderOperations); err != nil {
		return err
	}
	var started time.Time
	var version int64
	err = tx.QueryRow(ctx, `SELECT subject_id::text,subject_type,provider,user_id::text,started_at,version FROM app.buyer_verification_intents WHERE id=$1::uuid AND state='STARTED' FOR UPDATE`, id).Scan(&subject, &kind, &provider, &user, &started, &version)
	if err != nil {
		return err
	}
	if time.Since(started) < 2*time.Minute {
		return errors.New("wait two minutes for the original request to finish")
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	var result identity.ProviderVerification
	if action == "link" {
		result, err = identity.GetFrom(identity.WithActor(ctx, actor), s.identity, provider, strings.TrimSpace(reference))
		if err != nil {
			return err
		}
		if result.ProviderID != strings.TrimSpace(reference) || result.SubjectID != subject {
			return errors.New("the provider result does not belong to the original subject")
		}
	}
	tx, err = s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionProviderOperations); err != nil {
		return err
	}
	var state string
	var currentVersion int64
	if err = tx.QueryRow(ctx, `SELECT state,version FROM app.buyer_verification_intents WHERE id=$1::uuid FOR UPDATE`, id).Scan(&state, &currentVersion); err != nil {
		return err
	}
	if state != "STARTED" || currentVersion != version {
		return errors.New("this request has already been resolved")
	}
	if action == "link" {
		if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, user); err != nil {
			return err
		}
		session := identity.VerificationSession{Provider: provider, ProviderID: result.ProviderID, State: strings.ToLower(result.State), VerificationLevel: result.VerificationLevel, SafeResult: result.SafeResult, ExpiresAt: result.ExpiresAt}
		if err = insertVerification(ctx, tx, subject, kind, session, time.Now().UTC()); err != nil {
			return err
		}
		state = "CREATED"
	} else {
		state = "NEW"
		reference = ""
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, actor); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE app.buyer_verification_intents SET state=$2,version=version+1,provider_reference=NULLIF($3,''),started_at=CASE WHEN $2='NEW' THEN NULL ELSE started_at END,resolved_by=$4::uuid,resolution_note=$5,resolved_at=now() WHERE id=$1::uuid`, id, state, strings.TrimSpace(reference), actor, reason)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
