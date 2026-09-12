package settlement

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Register persists a send fence before calling the provider. It never stores
// the full bank account number. A lost response is held for operator recovery.
func Register(ctx context.Context, pool *pgxpool.Pool, key string, provider Provider, in Input) (Destination, error) {
	if pool == nil || len(key) < 32 || provider == nil {
		return Destination{}, errors.New("bank registration is unavailable")
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(in.OrganizationID + "\x00" + provider.Name() + "\x00" + provider.ConnectionIdentity() + "\x00" + in.BankCode + "\x00" + in.AccountNumber))
	in.Reference = hex.EncodeToString(mac.Sum(nil))
	if err := ValidateInput(in); err != nil {
		return Destination{}, err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Destination{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_organization_id',$1,true)`, in.OrganizationID); err != nil {
		return Destination{}, err
	}
	tag, err := tx.Exec(ctx, `INSERT INTO app.settlement_registrations(id,organization_id,provider,connection_identity,bank_code,account_last4,state) VALUES($1,$2::uuid,$3,$4,$5,$6,'STARTED') ON CONFLICT(id) DO NOTHING`, in.Reference, in.OrganizationID, provider.Name(), provider.ConnectionIdentity(), in.BankCode, in.AccountNumber[6:])
	if err != nil {
		return Destination{}, err
	}
	if tag.RowsAffected() == 0 {
		var state string
		var raw []byte
		if err = tx.QueryRow(ctx, `SELECT state,result FROM app.settlement_registrations WHERE id=$1 AND organization_id=$2::uuid`, in.Reference, in.OrganizationID).Scan(&state, &raw); err != nil {
			return Destination{}, err
		}

		if state == "REGISTERED" {
			var result Destination
			if json.Unmarshal(raw, &result) != nil || ValidateDestination(in, result) != nil {
				return Destination{}, ErrUnknown
			}
			return result, tx.Commit(ctx)
		}
		if state != "READY" {
			return Destination{}, ErrUnknown
		}
		tag, err = tx.Exec(ctx, `UPDATE app.settlement_registrations SET state='STARTED',created_at=now(),updated_at=now() WHERE id=$1 AND state='READY'`, in.Reference)
		if err != nil || tag.RowsAffected() != 1 {
			return Destination{}, ErrUnknown
		}

	}
	if err = tx.Commit(ctx); err != nil {
		return Destination{}, ErrUnknown
	}
	result, err := provider.CreateDestination(ctx, in)
	if err != nil {
		return Destination{}, ErrUnknown
	}
	if err = ValidateDestination(in, result); err != nil {
		return Destination{}, err
	}
	raw, _ := json.Marshal(result)
	tx, err = pool.Begin(ctx)
	if err != nil {
		return Destination{}, ErrUnknown
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_organization_id',$1,true)`, in.OrganizationID); err != nil {
		return Destination{}, ErrUnknown
	}
	tag, err = tx.Exec(ctx, `UPDATE app.settlement_registrations SET state='REGISTERED',result=$2::jsonb,updated_at=now() WHERE id=$1 AND state='STARTED'`, in.Reference, raw)
	if err != nil || tag.RowsAffected() != 1 {
		return Destination{}, ErrUnknown
	}
	if err = tx.Commit(ctx); err != nil {
		return Destination{}, ErrUnknown
	}
	return result, nil
}
