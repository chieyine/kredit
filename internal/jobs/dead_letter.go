package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type DeadLetterHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewDeadLetterHandler(pool *pgxpool.Pool, logger *slog.Logger) *DeadLetterHandler {
	return &DeadLetterHandler{pool: pool, logger: logger}
}

func (h *DeadLetterHandler) HandleError(ctx context.Context, job *rivertype.JobRow, err error) *river.ErrorHandlerResult {
	if h != nil && h.pool != nil && job != nil && job.Attempt >= job.MaxAttempts {
		_, insertErr := h.pool.Exec(ctx, `INSERT INTO app.job_dead_letters (river_job_id, job_kind, queue, encoded_args, error, attempts) VALUES ($1,$2,$3,$4::jsonb,$5,$6) ON CONFLICT (river_job_id) DO UPDATE SET error = EXCLUDED.error, attempts = EXCLUDED.attempts`, job.ID, job.Kind, job.Queue, string(job.EncodedArgs), fmt.Sprint(err), job.Attempt)
		if insertErr != nil && h.logger != nil {
			h.logger.Error("failed to persist job dead letter", "job_id", job.ID, "error", insertErr)
		}
	}
	if h != nil && h.logger != nil && job != nil {
		h.logger.Error("job failed", "job_id", job.ID, "kind", job.Kind, "queue", job.Queue, "attempt", job.Attempt, "max_attempts", job.MaxAttempts, "error", err)
	}
	return &river.ErrorHandlerResult{}
}

func (h *DeadLetterHandler) HandlePanic(ctx context.Context, job *rivertype.JobRow, panicVal any, trace string) *river.ErrorHandlerResult {
	return h.HandleError(ctx, job, fmt.Errorf("panic: %v\n%s", panicVal, trace))
}
