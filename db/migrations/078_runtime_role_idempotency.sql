-- +goose Up

-- Idempotency is an API concern. The worker has no reason to read or mutate
-- request keys, while the API needs reserve, replay and completion access.
DROP POLICY IF EXISTS idempotency_runtime_access ON app.idempotency_records;
CREATE POLICY idempotency_runtime_access ON app.idempotency_records
  USING (current_user = 'kredit_app')
  WITH CHECK (current_user = 'kredit_app');

-- Expired completed keys may be reused, but the API must not receive general
-- DELETE permission over the idempotency table.
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
        RETURNING 1
    )
    SELECT EXISTS (SELECT 1 FROM deleted)
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.delete_expired_idempotency_record(TEXT, TEXT) FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        GRANT EXECUTE ON FUNCTION app.delete_expired_idempotency_record(TEXT, TEXT) TO kredit_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        REVOKE EXECUTE ON FUNCTION app.delete_expired_idempotency_record(TEXT, TEXT) FROM kredit_app;
    END IF;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.delete_expired_idempotency_record(TEXT, TEXT) FROM PUBLIC;
DROP FUNCTION IF EXISTS app.delete_expired_idempotency_record(TEXT, TEXT);
DROP POLICY IF EXISTS idempotency_runtime_access ON app.idempotency_records;
