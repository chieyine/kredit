package payments

import (
	"context"
	"errors"
	"fmt"
	"kredit/internal/ledger"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type journalReplayTestRow struct {
	err     error
	matches bool
}

func (r journalReplayTestRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != 1 {
		return errors.New("unexpected journal validation result shape")
	}
	matched, ok := dest[0].(*bool)
	if !ok {
		return errors.New("journal validation must read an explicit match result")
	}
	*matched = r.matches
	return nil
}

type journalReplayTestTx struct {
	pgx.Tx
	queries   int
	writes    int
	matches   bool
	lookupErr error
}

func (tx *journalReplayTestTx) QueryRow(context.Context, string, ...any) pgx.Row {
	tx.queries++
	if tx.queries == 1 {
		// INSERT ... ON CONFLICT DO NOTHING found an existing journal key.
		return journalReplayTestRow{err: pgx.ErrNoRows}
	}
	return journalReplayTestRow{err: tx.lookupErr, matches: tx.matches}
}

func (tx *journalReplayTestTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	tx.writes++
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func TestAuditJournalReplayRequiresMatchingIntent(t *testing.T) {
	lookupFailure := errors.New("journal lookup unavailable")
	for _, tc := range []struct {
		name      string
		matches   bool
		lookupErr error
		wantError bool
	}{
		{name: "different intent", wantError: true},
		{name: "same intent", matches: true},
		{name: "lookup unavailable", lookupErr: lookupFailure, wantError: true},
		{name: "existing journal not visible", lookupErr: pgx.ErrNoRows, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &journalReplayTestTx{matches: tc.matches, lookupErr: tc.lookupErr}
			err := postLedgerTx(context.Background(), tx, "payment_recognized", "synthetic-reference", "synthetic-key", time.Unix(1800000000, 0), ledger.AccountVoluntarySettlement, ledger.AccountTradeReceivable, 2500)
			if (err != nil) != tc.wantError {
				t.Fatalf("journal replay did not enforce the recorded intent: %v", err)
			}
			if tc.lookupErr != nil && !errors.Is(err, tc.lookupErr) {
				t.Fatalf("lost the journal lookup error: %v", err)
			}
			if tx.queries != 2 || tx.writes != 0 {
				t.Fatalf("replay must validate without writing new postings: queries=%d writes=%d", tx.queries, tx.writes)
			}
		})
	}
}

func TestAuditJournalReplayPostgresMatchesAllFields(t *testing.T) {
	f := newPaymentFixture(t, 10000)
	tx, err := f.pool.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	at := time.Unix(1800000000, 123456000).UTC()
	key := f.paymentKeyPrefix + "-journal-fields"
	if err := postLedgerTx(f.ctx, tx, "payment_recognized", f.obligationID, key, at, ledger.AccountVoluntarySettlement, ledger.AccountTradeReceivable, 2500); err != nil {
		t.Fatal(err)
	}
	for _, sameTime := range []time.Time{at, at.In(time.FixedZone("WAT", 3600)), at.Add(789 * time.Nanosecond)} {
		if err := postLedgerTx(f.ctx, tx, "payment_recognized", f.obligationID, key, sameTime, ledger.AccountVoluntarySettlement, ledger.AccountTradeReceivable, 2500); err != nil {
			t.Fatalf("equivalent journal replay failed: %v", err)
		}
	}
	for _, tc := range []struct {
		name, event, reference, debit, credit string
		at                                    time.Time
		amount                                ledger.Money
	}{
		{"event", "payment_reversed", f.obligationID, ledger.AccountVoluntarySettlement, ledger.AccountTradeReceivable, at, 2500},
		{"reference", "payment_recognized", f.requestID, ledger.AccountVoluntarySettlement, ledger.AccountTradeReceivable, at, 2500},
		{"timestamp", "payment_recognized", f.obligationID, ledger.AccountVoluntarySettlement, ledger.AccountTradeReceivable, at.Add(time.Microsecond), 2500},
		{"amount", "payment_recognized", f.obligationID, ledger.AccountVoluntarySettlement, ledger.AccountTradeReceivable, at, 2501},
		{"debit account", "payment_recognized", f.obligationID, ledger.AccountCollectionSettlement, ledger.AccountTradeReceivable, at, 2500},
		{"direction", "payment_recognized", f.obligationID, ledger.AccountTradeReceivable, ledger.AccountVoluntarySettlement, at, 2500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := postLedgerTx(f.ctx, tx, tc.event, tc.reference, key, tc.at, tc.debit, tc.credit, tc.amount); err == nil {
				t.Fatal("mismatched journal replay was accepted")
			}
		})
	}
	var postings int
	if err := tx.QueryRow(f.ctx, `SELECT count(*) FROM ledger.postings p JOIN ledger.transactions t ON t.id=p.transaction_id WHERE t.idempotency_key=$1`, key).Scan(&postings); err != nil || postings != 2 {
		t.Fatalf("replay altered original postings: count=%d error=%v", postings, err)
	}
}

func TestAuditPaymentRejectsUnrelatedExistingJournalAtomically(t *testing.T) {
	f := newPaymentFixture(t, 10000)
	at := time.Now().UTC().Truncate(time.Microsecond)
	key := f.paymentKeyPrefix + "-foreign-journal"
	journalKey := "payment:" + key
	tx, err := f.pool.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	// Simulate conflicting retained evidence. Do not rewrite or delete it to
	// make a new payment appear reconciled.
	if err := postLedgerTx(f.ctx, tx, "payment_recognized", f.requestID, journalKey, at, ledger.AccountVoluntarySettlement, ledger.AccountTradeReceivable, 2500); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	_, _, err = f.store().RecordContext(f.ctx, RecordInput{ObligationID: f.obligationID, SourceType: SourceVoluntary, AmountKobo: 2500, RecordedBy: f.userID, PaidAt: at, IdempotencyKey: key})
	if err == nil {
		t.Fatal("payment committed against an unrelated existing journal")
	}
	var outstanding, allocated int64
	var payments, outboxEvents int
	if err := f.pool.QueryRow(f.ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, f.obligationID).Scan(&outstanding); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT COALESCE(sum(i.allocated_kobo),0) FROM app.schedule_items i JOIN app.repayment_schedules s ON s.id=i.schedule_id WHERE s.obligation_id=$1::uuid`, f.obligationID).Scan(&allocated); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT count(*) FROM app.payments WHERE idempotency_key=$1`, key).Scan(&payments); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT count(*) FROM app.outbox_events WHERE idempotency_key=$1`, journalKey).Scan(&outboxEvents); err != nil {
		t.Fatal(err)
	}
	if outstanding != 10000 || allocated != 0 || payments != 0 || outboxEvents != 0 {
		t.Fatal(fmt.Sprintf("conflicting journal left partial financial state: outstanding=%d allocated=%d payments=%d events=%d", outstanding, allocated, payments, outboxEvents))
	}
}
