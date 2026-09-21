package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const rollbackCleanupTimeout = 5 * time.Second

// RollbackFailure releases a failed transaction without discarding its original
// error. It is only for failure paths, not for deciding whether a commit worked.
// A failed commit can have an uncertain outcome; this function never retries it.
//
// Cleanup retains request values but has its own bounded lifetime so a cancelled
// request still gives PostgreSQL an opportunity to release the transaction.
func RollbackFailure(ctx context.Context, tx pgx.Tx, cause error) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackCleanupTimeout)
	defer cancel()

	err := tx.Rollback(cleanup)
	if err == nil || errors.Is(err, pgx.ErrTxClosed) {
		return cause
	}
	return errors.Join(cause, fmt.Errorf("rollback failed: %w", err))
}
