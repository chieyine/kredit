-- +goose Up

-- A crashed publisher must not leave an event permanently invisible. The
-- lease is renewed whenever an event is claimed and stale processing rows are
-- returned to the retryable failed state by the next claimant.
ALTER TABLE app.outbox_events
    ADD COLUMN IF NOT EXISTS processing_started_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS outbox_processing_lease_idx
    ON app.outbox_events (processing_started_at)
    WHERE state = 'processing';

-- +goose Down

DROP INDEX IF EXISTS app.outbox_processing_lease_idx;
ALTER TABLE app.outbox_events
    DROP COLUMN IF EXISTS processing_started_at;
