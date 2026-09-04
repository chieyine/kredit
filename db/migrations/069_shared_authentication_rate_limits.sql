-- +goose Up
-- The HTTP rate limiter is a per-process map, so with N API replicas the real
-- budget was N times the configured one, and it reset on every deploy. That is
-- tolerable for ordinary reads; it is not tolerable for the endpoints that
-- issue and verify authentication codes, where the limit is the control that
-- makes a six-digit code safe.
--
-- The counter is keyed by an HMAC of the client address and route group so no
-- raw address is stored, matching the treatment already used for account
-- recovery in migration 048.

CREATE TABLE IF NOT EXISTS app.request_rate_limits (
    bucket_hash BYTEA PRIMARY KEY,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    window_started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT request_rate_limits_count_non_negative CHECK (attempt_count >= 0)
);

CREATE INDEX IF NOT EXISTS request_rate_limits_window_idx
    ON app.request_rate_limits (window_started_at);

-- Returns the attempt count inside the current window after recording this
-- attempt. A caller is over budget when the returned count exceeds its limit.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.record_rate_limit_attempt(
    p_bucket_hash BYTEA,
    p_window INTERVAL
) RETURNS INTEGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = app, pg_catalog
AS $$
DECLARE
  current_count INTEGER;
BEGIN
  INSERT INTO app.request_rate_limits AS existing (bucket_hash, attempt_count, window_started_at)
  VALUES (p_bucket_hash, 1, now())
  ON CONFLICT (bucket_hash) DO UPDATE
    SET attempt_count = CASE
          WHEN existing.window_started_at < now() - p_window THEN 1
          ELSE existing.attempt_count + 1
        END,
        window_started_at = CASE
          WHEN existing.window_started_at < now() - p_window THEN now()
          ELSE existing.window_started_at
        END
  RETURNING attempt_count INTO current_count;
  RETURN current_count;
END;
$$;
-- +goose StatementEnd

-- Expired buckets are removed opportunistically by the worker's maintenance
-- pass; this keeps the table bounded without a scheduled vacuum job.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.prune_rate_limits(p_window INTERVAL)
RETURNS INTEGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = app, pg_catalog
AS $$
DECLARE
  removed INTEGER;
BEGIN
  DELETE FROM app.request_rate_limits WHERE window_started_at < now() - (p_window * 4);
  GET DIAGNOSTICS removed = ROW_COUNT;
  RETURN removed;
END;
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.record_rate_limit_attempt(BYTEA, INTERVAL) FROM PUBLIC;
REVOKE ALL ON FUNCTION app.prune_rate_limits(INTERVAL) FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        GRANT EXECUTE ON FUNCTION app.record_rate_limit_attempt(BYTEA, INTERVAL) TO kredit_app;
        GRANT EXECUTE ON FUNCTION app.prune_rate_limits(INTERVAL) TO kredit_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS app.prune_rate_limits(INTERVAL);
DROP FUNCTION IF EXISTS app.record_rate_limit_attempt(BYTEA, INTERVAL);
DROP TABLE IF EXISTS app.request_rate_limits;
