package operations

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type postingResultTx struct {
	pgx.Tx
	results []string
	calls   int
}

func (t *postingResultTx) QueryRow(context.Context, string, ...any) pgx.Row {
	return postingTransactionRow{}
}
func (t *postingResultTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	result := t.results[t.calls]
	t.calls++
	return pgconn.NewCommandTag(result), nil
}

type postingTransactionRow struct{}

func (postingTransactionRow) Scan(dest ...any) error { *dest[0].(*string) = "transaction"; return nil }

func TestAdjustmentRequiresBothLedgerPostings(t *testing.T) {
	for _, tc := range []struct {
		name      string
		results   []string
		wantError bool
	}{
		{"missing debit account", []string{"INSERT 0 0"}, true},
		{"missing credit account", []string{"INSERT 0 1", "INSERT 0 0"}, true},
		{"both accounts", []string{"INSERT 0 1", "INSERT 0 1"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &postingResultTx{results: tc.results}
			_, err := postOperationLedgerTx(context.Background(), tx, "write_off", "obligation", "key", 100)
			if (err != nil) != tc.wantError {
				t.Fatalf("posting error=%v; want failure=%v", err, tc.wantError)
			}
			if tx.calls != len(tc.results) {
				t.Fatalf("attempted %d postings, want %d", tx.calls, len(tc.results))
			}
		})
	}
}
