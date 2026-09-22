package db

import (
	"context"
	"errors"
	"fmt"
	"kredit/internal/ledger"

	"github.com/jackc/pgx/v5"
)

func ReduceSchedulePrincipalTx(ctx context.Context, tx pgx.Tx, obligation string, outstanding, amount ledger.Money, resolvingDispute bool) error {
	if tx == nil || obligation == "" || outstanding < 0 || amount <= 0 || amount > outstanding {
		return errors.New("valid transaction, obligation, and positive reduction within outstanding principal are required")
	}
	rows, err := tx.Query(ctx, `SELECT i.id::text,i.principal_due_kobo,i.allocated_kobo,i.disputed_kobo FROM app.schedule_items i JOIN app.repayment_schedules s ON s.id=i.schedule_id WHERE s.obligation_id=$1::uuid AND i.state<>'CANCELLED' ORDER BY i.sequence DESC FOR UPDATE OF i`, obligation)
	if err != nil {
		return err
	}
	type item struct {
		id                             string
		principal, allocated, disputed ledger.Money
	}
	items := []item{}
	var total ledger.Money
	for rows.Next() {
		var i item
		if err = rows.Scan(&i.id, &i.principal, &i.allocated, &i.disputed); err != nil {
			rows.Close()
			return err
		}
		total, err = ledger.CheckedAdd(total, i.principal-i.allocated)
		if err != nil {
			rows.Close()
			return err
		}
		items = append(items, i)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if total != outstanding {
		return errors.New("schedule and outstanding balance disagree; reconcile before adjusting")
	}
	for _, i := range items {
		if i.disputed > 0 && !resolvingDispute {
			return errors.New("resolve disputed instalments before adjusting principal")
		}
	}
	var pending bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.payment_claims WHERE obligation_id=$1::uuid AND state='pending')`, obligation).Scan(&pending); err != nil {
		return err
	}
	if pending {
		return errors.New("resolve pending payment claims before adjusting principal")
	}
	remaining := amount
	for _, i := range items {
		take := min(remaining, i.principal-i.allocated)
		if take == 0 {
			continue
		}
		next := i.principal - take
		state := ""
		switch next {
		case 0:
			state = "CANCELLED"
			next = i.principal
		case i.allocated:
			state = "PAID"
		}
		if _, err = tx.Exec(ctx, `UPDATE app.schedule_items SET principal_due_kobo=$2,disputed_kobo=CASE WHEN $3='CANCELLED' THEN 0 ELSE LEAST(disputed_kobo,GREATEST(0,$2-allocated_kobo)) END,state=CASE WHEN $3='' THEN state ELSE $3 END WHERE id=$1::uuid`, i.id, int64(next), state); err != nil {
			return err
		}
		remaining -= take
		if remaining == 0 {
			break
		}
	}
	if remaining != 0 {
		return errors.New("adjustment exceeds remaining schedule")
	}
	return nil
}

// UpdateObligationBalanceTx moves an obligation balance, its request version and
// its read projection together. The caller must hold the financial-operation
// locks, install the tenant context, and roll back the entire transaction on
// any error. The authoritative obligation/request relationship and principal
// are checked here as well; independent caller-supplied IDs are not sufficient.
func UpdateObligationBalanceTx(ctx context.Context, tx pgx.Tx, requestID, obligationID string, outstanding, principal ledger.Money) error {
	if tx == nil || requestID == "" || obligationID == "" || outstanding < 0 || principal < 0 || outstanding > principal {
		return errors.New("valid transaction, credit request, obligation and balance within principal are required")
	}
	var status string
	switch outstanding {
	case 0:
		status = "PAID"
	case principal:
		status = "UNPAID"
	default:
		status = "PARTIALLY_PAID"
	}
	// UPDATE retains the existing obligation-then-request lock order. RETURNING
	// requires a matching visible row and supplies canonical stored identifiers.
	var storedRequestID, storedObligationID string
	if err := tx.QueryRow(ctx, `
		UPDATE app.obligations SET outstanding_kobo=$2,payment_status=$3
		WHERE id=$1::uuid AND credit_request_id=$4::uuid AND principal_kobo=$5::bigint
		RETURNING credit_request_id::text,id::text`, obligationID, int64(outstanding), status, requestID, int64(principal)).Scan(&storedRequestID, &storedObligationID); err != nil {
		return fmt.Errorf("update matching obligation and principal: %w", err)
	}
	var version int64
	if err := tx.QueryRow(ctx, `
		UPDATE app.credit_requests SET version=version+1,updated_at=now()
		WHERE id=$1::uuid RETURNING version`, storedRequestID).Scan(&version); err != nil {
		return fmt.Errorf("advance obligation credit request version: %w", err)
	}
	command, err := tx.Exec(ctx, `
		UPDATE app.credit_aggregate_snapshots
		SET aggregate=jsonb_set(jsonb_set(jsonb_set(aggregate,
		    '{obligation,outstanding_kobo}',to_jsonb($2::bigint),false),
		    '{obligation,payment_status}',to_jsonb($3::text),false),
		    '{request,version}',to_jsonb($5::bigint),false),
		    version=$5::bigint,updated_at=now()
		WHERE credit_request_id=$1
		  AND version=$5::bigint-1
		  AND jsonb_typeof(aggregate->'request')='object'
		  AND jsonb_typeof(aggregate->'obligation')='object'
		  AND aggregate#>>'{request,id}'=$1::text
		  AND aggregate#>>'{obligation,id}'=$4
		  AND aggregate#>>'{request,version}'=version::text
		  AND jsonb_typeof(aggregate#>'{obligation,outstanding_kobo}')='number'
		  AND jsonb_typeof(aggregate#>'{obligation,payment_status}')='string'`,
		storedRequestID, int64(outstanding), status, storedObligationID, version)
	if err != nil {
		return fmt.Errorf("update obligation read projection: %w", err)
	}
	if command.RowsAffected() != 1 {
		return errors.New("credit aggregate snapshot is missing, stale or inconsistent; reconcile before adjusting")
	}
	return nil
}
