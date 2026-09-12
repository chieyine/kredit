package mandates

import (
	"context"
	"encoding/json"
	"errors"
	"kredit/internal/access"
	"strings"
	"time"
)

type AuthorizationAttempt struct {
	ID         string    `json:"id"`
	BusinessID string    `json:"business_id"`
	Reference  string    `json:"reference"`
	CreatedAt  time.Time `json:"created_at"`
}

func (p *PostgresProvider) ListAuthorizationAttempts(ctx context.Context, actor string) ([]AuthorizationAttempt, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionProviderOperations); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text,business_id::text,reference,created_at FROM app.mandate_authorization_intents WHERE state='STARTED' ORDER BY created_at,id LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []AuthorizationAttempt{}
	for rows.Next() {
		var item AuthorizationAttempt
		if err = rows.Scan(&item.ID, &item.BusinessID, &item.Reference, &item.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (p *PostgresProvider) ResolveAuthorizationAttempt(ctx context.Context, actor, id, action, providerID, reason string) error {
	providerID = strings.TrimSpace(providerID)
	if action == "link" && (providerID == "" || len(providerID) > 128) {
		return errors.New("a valid provider reference is required")
	}
	if len(strings.TrimSpace(reason)) < 20 || len(reason) > 2000 || (action != "link" && action != "not_created") {
		return errors.New("provider evidence and a valid action are required")
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionProviderOperations); err != nil {
		return err
	}
	var payload []byte
	var created time.Time
	if err = tx.QueryRow(ctx, `SELECT input,created_at FROM app.mandate_authorization_intents WHERE id=$1::uuid AND state='STARTED' FOR UPDATE`, id).Scan(&payload, &created); err != nil {
		return err
	}
	if time.Since(created) < 2*time.Minute {
		return errors.New("wait two minutes for the original request to finish")
	}
	var input AuthorizationInput
	if err = json.Unmarshal(payload, &input); err != nil {
		return err
	}
	if action == "not_created" {
		_, err = tx.Exec(ctx, `UPDATE app.mandate_authorization_intents SET state='NOT_CREATED',resolved_at=now(),resolved_by=$2::uuid,resolution_note=$3 WHERE id=$1::uuid`, id, actor, reason)
		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	// Release the database transaction before requesting authoritative evidence.
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	if p.remote == nil {
		return errors.New("connect the mandate provider before recovery")
	}
	mandate, err := p.remote.GetMandate(ctx, strings.TrimSpace(providerID))
	if err != nil {
		return err
	}
	if mandate.Reference == "" || mandate.Reference != input.Reference || mandate.ProviderID != strings.TrimSpace(providerID) || mandate.AmountCeiling != input.AmountCeiling || !mandate.Variable || mandate.EndsAt.Before(input.RequiredUntil) {
		return errors.New("provider mandate does not match the original reference, amount or validity")
	}
	if mandate.Status != Active {
		return errors.New("this mandate is not active; recover its original authorization link in the Mono dashboard before attaching it")
	}
	_, err = p.saveAuthorization(ctx, input, mandate, id, actor, reason)
	return err
}
