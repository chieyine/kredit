package db

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type rollbackFailureTx struct {
	pgx.Tx
	rollback func(context.Context) error
}

func (tx rollbackFailureTx) Rollback(ctx context.Context) error {
	return tx.rollback(ctx)
}

func TestRollbackFailurePreservesOriginalCause(t *testing.T) {
	primary := errors.New("primary operation failed")
	transport := errors.New("rollback transport failed")
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "successful cleanup"},
		{name: "already closed", err: pgx.ErrTxClosed},
		{name: "wrapped already closed", err: fmt.Errorf("transaction: %w", pgx.ErrTxClosed)},
		{name: "transport failure", err: transport},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			tx := rollbackFailureTx{rollback: func(context.Context) error {
				calls++
				return tc.err
			}}
			got := RollbackFailure(context.Background(), tx, primary)
			if !errors.Is(got, primary) || calls != 1 {
				t.Fatalf("lost primary cause or repeated rollback: error=%v calls=%d", got, calls)
			}
			if errors.Is(tc.err, transport) {
				if !errors.Is(got, transport) {
					t.Fatalf("lost cleanup error: %v", got)
				}
			} else if got != primary {
				t.Fatalf("normal cleanup changed the original error: %v", got)
			}
		})
	}
}

func TestRollbackFailureBoundsCleanupAndPreservesContextValues(t *testing.T) {
	type key struct{}
	parent, cancel := context.WithCancel(context.WithValue(context.Background(), key{}, "tenant-test"))
	cancel()
	var cleanup context.Context
	tx := rollbackFailureTx{rollback: func(ctx context.Context) error {
		cleanup = ctx
		if ctx.Err() != nil || ctx.Value(key{}) != "tenant-test" {
			t.Fatalf("cleanup lost request values or inherited cancellation: %v", ctx.Err())
		}
		deadline, ok := ctx.Deadline()
		remaining := time.Until(deadline)
		if !ok || remaining <= 0 || remaining > rollbackCleanupTimeout {
			t.Fatalf("cleanup must have a bounded deadline: ok=%v remaining=%v", ok, remaining)
		}
		return nil
	}}
	if got := RollbackFailure(parent, tx, context.Canceled); !errors.Is(got, context.Canceled) {
		t.Fatalf("lost request cancellation: %v", got)
	}
	if cleanup == nil || cleanup.Err() != context.Canceled {
		t.Fatal("cleanup context was not cancelled after rollback returned")
	}
}
