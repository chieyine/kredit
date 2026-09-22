package db

import (
	"context"

	"kredit/internal/platform/txcleanup"

	"github.com/jackc/pgx/v5"
)

const rollbackCleanupTimeout = txcleanup.Timeout

// RollbackFailure releases a failed transaction without discarding its original
// error. It is only for failure paths, not for deciding whether a commit worked.
// A failed commit can have an uncertain outcome; this function never retries it.
//
// Cleanup retains request values but has its own bounded lifetime so a cancelled
// request still gives PostgreSQL an opportunity to release the transaction.
func RollbackFailure(ctx context.Context, tx pgx.Tx, cause error) error {
	return txcleanup.Rollback(ctx, tx, cause)
}
