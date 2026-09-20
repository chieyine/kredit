-- +goose Up
-- +goose StatementBegin

-- 1. One vocabulary for app.obligations.payment_status.
--
-- Every writer uses PAID / UNPAID / PARTIALLY_PAID except the order credit-note
-- path, which wrote 'settled'. The column had no CHECK, so the value was stored
-- and those obligations then fell out of every report filtering on
-- payment_status='PAID' -- app.pilot_metric (migration 122) and the audit and
-- recovery reporting in migration 155. Normalise the affected rows from the
-- authoritative balance, then make a fourth vocabulary impossible.
UPDATE app.obligations SET payment_status = CASE
    WHEN outstanding_kobo = 0 THEN 'PAID'
    WHEN outstanding_kobo = principal_kobo THEN 'UNPAID'
    ELSE 'PARTIALLY_PAID' END
 WHERE payment_status NOT IN ('PAID', 'UNPAID', 'PARTIALLY_PAID');

ALTER TABLE app.obligations
  ADD CONSTRAINT obligations_payment_status_check
  CHECK (payment_status IN ('PAID', 'UNPAID', 'PARTIALLY_PAID'));

-- 2. Release an idempotency reservation that never produced an outcome.
--
-- The previous definition removed a row only when completed_at IS NOT NULL. The
-- HTTP middleware records an outcome on panic, but a hard kill between Reserve
-- and Complete (OOM, SIGKILL, node eviction) leaves the row reserved with no
-- response, and every retry of that write then receives
-- 409 idempotency_in_progress for good. Once the reservation has expired there
-- is no recorded outcome to replay, so releasing it is safe: the retry simply
-- reserves the key again.
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
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_worker') THEN
        GRANT EXECUTE ON FUNCTION app.delete_expired_idempotency_record(TEXT, TEXT) TO kredit_worker;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app.obligations DROP CONSTRAINT IF EXISTS obligations_payment_status_check;
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
