package db

import (
	"context"
	"errors"
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

// UpdateObligationBalanceTx is the single way an obligation's outstanding
// balance changes. It moves the balance, derives the payment status from the
// same two numbers every other writer uses, bumps the credit request version
// and patches the read projection — and fails if the projection is missing,
// because a balance that moved without its projection is a lie the next reader
// will believe.
//
// payments, operations and disputes each carry a private copy of this. They
// agree; the copy that was written independently (order credit notes) did not,
// and lost the projection, the version and the status vocabulary. New callers
// use this one.
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
	if _, err := tx.Exec(ctx, `UPDATE app.obligations SET outstanding_kobo=$2,payment_status=$3 WHERE id=$1::uuid`, obligationID, int64(outstanding), status); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE app.credit_requests SET version=version+1,updated_at=now() WHERE id=$1::uuid`, requestID); err != nil {
		return err
	}
	command, err := tx.Exec(ctx, `UPDATE app.credit_aggregate_snapshots SET aggregate=jsonb_set(jsonb_set(jsonb_set(aggregate,'{obligation,outstanding_kobo}',to_jsonb($2::bigint),false),'{obligation,payment_status}',to_jsonb($3::text),false),'{request,version}',to_jsonb(version+1),false),version=version+1,updated_at=now() WHERE credit_request_id=$1`, requestID, int64(outstanding), status)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return errors.New("credit aggregate snapshot not found")
	}
	return nil
}
