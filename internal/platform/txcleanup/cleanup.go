// Package txcleanup gives transaction cleanup a lifetime independent of the
// request that initiated it. It never retries a transaction or a commit.
package txcleanup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

const Timeout = 5 * time.Second

// Rollback preserves the initiating error. ErrTxClosed is expected after either
// a completed commit or a commit failure; neither establishes a retry decision.
func Rollback(ctx context.Context, tx pgx.Tx, cause error) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), Timeout)
	defer cancel()
	if err := tx.Rollback(cleanup); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return errors.Join(cause, fmt.Errorf("rollback failed: %w", err))
	}
	return cause
}

// Finish must be deferred directly with a pointer to the named return error.
// During a panic it records cleanup failure and re-panics with the original
// value, rather than replacing the panic or losing the cleanup opportunity.
func Finish(ctx context.Context, tx pgx.Tx, result *error) {
	if value := recover(); value != nil {
		if err := Rollback(ctx, tx, nil); err != nil {
			slog.Error("transaction cleanup during panic failed", "error", err)
		}
		panic(value)
	}
	*result = Rollback(ctx, tx, *result)
}
