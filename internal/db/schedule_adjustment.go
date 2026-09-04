package db

import (
	"context"
	"errors"
	"kredit/internal/ledger"

	"github.com/jackc/pgx/v5"
)

func ReduceSchedulePrincipalTx(ctx context.Context, tx pgx.Tx, obligation string, outstanding, amount ledger.Money, resolvingDispute bool) error {
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
