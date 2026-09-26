-- +goose Up
-- A process can stop after committing a domain mutation but before recording
-- its HTTP result. Expiration does not prove that the mutation never happened.
-- Preserve these reservations, and server-error outcomes (including recovered
-- panics), until their effects have been reconciled. Completed responses below
-- 500 retain the existing 24-hour replay contract.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.delete_expired_idempotency_record(
    p_scope TEXT,
    p_idempotency_key TEXT
)
RETURNS BOOLEAN
LANGUAGE sql
SECURITY DEFINER
SET search_path = app, pg_catalog
AS $$
    WITH deleted AS (
        DELETE FROM app.idempotency_records
        WHERE scope = p_scope
          AND idempotency_key = p_idempotency_key
          AND expires_at <= NOW()
          AND completed_at IS NOT NULL
          AND response_status >= 200
          AND response_status < 500
        RETURNING 1
    )
    SELECT EXISTS (SELECT 1 FROM deleted)
$$;
-- +goose StatementEnd

-- CREATE OR REPLACE preserves the existing owner and role-specific grants.
REVOKE ALL ON FUNCTION app.delete_expired_idempotency_record(TEXT, TEXT) FROM PUBLIC;

-- +goose Down
-- Do not restore automatic retries of uncertain financial operations on rollback.
SELECT 1;
