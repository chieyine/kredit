package billing

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"kredit/internal/ledger"
)

type SplitFee struct {
	ID     string       `json:"id"`
	Amount ledger.Money `json:"amount_kobo"`
}

// FreezeSplitTx reserves only fees for this sale that have never been invoiced.
// The accepted fee terms, rather than today's pricing, determine the debit fee.
func FreezeSplitTx(ctx context.Context, tx pgx.Tx, obligation string, amount ledger.Money) ([]SplitFee, ledger.Money, error) {
	var terms *ledger.FeeTerms
	if err := tx.QueryRow(ctx, `SELECT c.fee_terms FROM app.credit_requests c JOIN app.obligations o ON o.credit_request_id=c.id WHERE o.id=$1::uuid`, obligation).Scan(&terms); err != nil {
		return nil, 0, err
	}
	fee, err := terms.Collection(amount)
	if err != nil {
		return nil, 0, err
	}
	rows, err := tx.Query(ctx, `SELECT f.id::text,f.amount_kobo-f.waived_kobo-f.collected_kobo FROM app.fees f WHERE f.obligation_id=$1::uuid AND f.fee_type='base_service' AND f.state='accrued' AND f.amount_kobo>f.waived_kobo+f.collected_kobo AND NOT EXISTS(SELECT 1 FROM app.fee_invoice_lines l WHERE l.fee_id=f.id) ORDER BY f.id FOR SHARE OF f`, obligation)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	parts := []SplitFee{}
	for rows.Next() {
		var p SplitFee
		if err = rows.Scan(&p.ID, &p.Amount); err != nil {
			return nil, 0, err
		}
		p.Amount = min(p.Amount, amount-fee)
		if p.Amount > 0 {
			parts = append(parts, p)
			fee += p.Amount
		}
	}
	return parts, fee, rows.Err()
}

// ApplySplitTx allocates a verified collection. Partial recoveries retain only
// fees supported by the amount actually received; bank payout remains separate.
func ApplySplitTx(ctx context.Context, tx pgx.Tx, payment, key string, amount, collectionFee ledger.Money, at time.Time) error {
	if !strings.HasPrefix(key, "collection-attempt:") {
		return nil
	}
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT route FROM app.collection_settlement_routes WHERE attempt_id=$1::uuid`, strings.TrimPrefix(key, "collection-attempt:")).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var route struct {
		BillingMethod string       `json:"billing_method"`
		BaseFees      []SplitFee   `json:"base_fees"`
		Fee           ledger.Money `json:"fee_amount_kobo"`
	}
	if err = json.Unmarshal(raw, &route); err != nil {
		return err
	}
	if route.BillingMethod != "split_settlement" {
		return nil
	}
	parts := []SplitFee{}
	if collectionFee > 0 {
		var id string
		if err = tx.QueryRow(ctx, `SELECT id::text FROM app.fees WHERE payment_id=$1::uuid AND fee_type='collection'`, payment).Scan(&id); err != nil {
			return err
		}
		parts = append(parts, SplitFee{id, collectionFee})
	}
	total := collectionFee
	for _, p := range route.BaseFees {
		p.Amount = min(p.Amount, amount-total)
		if p.Amount > 0 {
			parts = append(parts, p)
			total += p.Amount
		}
	}
	if total > route.Fee || total > amount {
		return errors.New("verified fee exceeds the frozen settlement allocation")
	}
	for _, p := range parts {
		result, err := tx.Exec(ctx, `UPDATE app.fees SET collected_kobo=collected_kobo+$2 WHERE id=$1::uuid AND state='accrued' AND amount_kobo-waived_kobo-collected_kobo>=$2 AND NOT EXISTS(SELECT 1 FROM app.fee_invoice_lines WHERE fee_id=$1::uuid)`, p.ID, p.Amount)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return errors.New("fee allocation no longer matches the saved collection")
		}
		if _, err = tx.Exec(ctx, `INSERT INTO app.split_fee_allocations(payment_id,fee_id,supplier_organization_id,amount_kobo) SELECT $1::uuid,id,supplier_organization_id,$3 FROM app.fees WHERE id=$2::uuid`, payment, p.ID, p.Amount); err != nil {
			return err
		}
	}
	if total == 0 {
		return nil
	}
	_, err = splitFeeLedger().PostSplitFeeTx(ctx, tx, payment, total, false, at)
	return err
}

// ReverseSplitTx is part of the caller-owned payment reversal transaction.
// Allocation rows are immutable evidence; the reversed payment state excludes
// them from current cash/reward totals. A journal is the durable undo marker.
func ReverseSplitTx(ctx context.Context, tx pgx.Tx, payment string, at time.Time) error {
	var state string
	if err := tx.QueryRow(ctx, `SELECT state FROM app.payments WHERE id=$1::uuid AND reversal_of IS NULL FOR UPDATE`, payment).Scan(&state); err != nil {
		return err
	}
	if state != "reversed" {
		return errors.New("a reversed original payment is required")
	}
	rows, err := tx.Query(ctx, `SELECT fee_id::text,amount_kobo FROM app.split_fee_allocations WHERE payment_id=$1::uuid ORDER BY fee_id`, payment)
	if err != nil {
		return err
	}
	parts := []SplitFee{}
	var total ledger.Money
	for rows.Next() {
		var p SplitFee
		if err = rows.Scan(&p.ID, &p.Amount); err != nil {
			rows.Close()
			return err
		}
		if p.Amount <= 0 {
			rows.Close()
			return errors.New("invalid original fee allocation")
		}
		total, err = ledger.CheckedAdd(total, p.Amount)
		if err != nil {
			rows.Close()
			return err
		}
		parts = append(parts, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil || total == 0 {
		return err
	}
	var reversed bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ledger.transactions WHERE idempotency_key=$1)`, "split_fee_reversed:"+payment).Scan(&reversed); err != nil {
		return err
	}
	if reversed {
		// Validate the prior journal's exact postings without subtracting twice.
		// A zero date reuses the recorded one: a replay arrives at a later clock
		// reading and must not conflict with the original reversal date.
		_, err = splitFeeLedger().PostSplitFeeTx(ctx, tx, payment, total, true, time.Time{})
		return err
	}
	for _, p := range parts {
		result, updateErr := tx.Exec(ctx, `UPDATE app.fees SET collected_kobo=collected_kobo-$2 WHERE id=$1::uuid AND collected_kobo>=$2`, p.ID, p.Amount)
		if updateErr != nil {
			return updateErr
		}
		if result.RowsAffected() != 1 {
			return errors.New("fee allocation no longer matches the reversed collection")
		}
	}
	_, err = splitFeeLedger().PostSplitFeeTx(ctx, tx, payment, total, true, at)
	return err
}

// splitFeeLedger posts split-fee journals through the caller's transaction.
// The store needs no pool because PostSplitFeeTx only writes through the
// transaction it is given; naming it here keeps that fact in one place instead
// of constructing a nil-pool store at each call site.
func splitFeeLedger() *ledger.PostgresStore { return ledger.NewPostgresStore(nil) }
