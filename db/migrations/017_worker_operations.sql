-- +goose Up

-- Provider events are accepted exactly once before asynchronous processing.
-- The unique provider/event key is the durable duplicate-webhook boundary.
CREATE TABLE IF NOT EXISTS app.provider_webhook_inbox (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    provider text NOT NULL,
    event_id text NOT NULL,
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    signature_valid boolean NOT NULL,
    state text NOT NULL DEFAULT 'received' CHECK (state IN ('received', 'processing', 'processed', 'failed')),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    last_error text,
    received_at timestamptz NOT NULL DEFAULT now(),
    processed_at timestamptz,
    UNIQUE (provider, event_id)
);
CREATE INDEX IF NOT EXISTS provider_webhook_inbox_pending_idx ON app.provider_webhook_inbox (received_at) WHERE state IN ('received', 'failed');

-- River keeps retryable jobs in its own schema. This table preserves the
-- terminal failure payload for operational replay and audit workflows.
CREATE TABLE IF NOT EXISTS app.job_dead_letters (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    river_job_id bigint NOT NULL,
    job_kind text NOT NULL,
    queue text NOT NULL,
    encoded_args jsonb NOT NULL,
    error text NOT NULL,
    attempts integer NOT NULL CHECK (attempts > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (river_job_id)
);

-- +goose Down
DROP TABLE IF EXISTS app.job_dead_letters;
DROP TABLE IF EXISTS app.provider_webhook_inbox;
