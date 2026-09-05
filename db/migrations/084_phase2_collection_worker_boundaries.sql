-- +goose Up

-- Global collection discovery is a worker concern, but granting the worker
-- unrestricted SELECT on every tenant's obligations defeats tenant RLS. These
-- SECURITY DEFINER functions expose only the identifiers required to enqueue
-- tenant-scoped work. Actual collection reads/writes still run as kredit_worker
-- under normal row-level security after the organization context is installed.

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.collection_due_work_page(p_cursor text, p_limit integer)
RETURNS TABLE(resource_id text, organization_id uuid)
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = pg_catalog, app
AS $$
    SELECT o.id::text, o.supplier_organization_id
    FROM app.obligations o
    WHERE o.lifecycle_status = 'ACTIVE'
      AND o.outstanding_kobo > 0
      AND o.id::text > COALESCE(p_cursor, '')
      AND EXISTS (
          SELECT 1
          FROM app.repayment_schedules s
          JOIN app.schedule_items i ON i.schedule_id = s.id
          WHERE s.obligation_id = o.id
            AND i.state NOT IN ('PAID','CANCELLED')
            AND i.collection_at <= now()
            AND i.principal_due_kobo > i.allocated_kobo
      )
      AND NOT EXISTS (
          SELECT 1
          FROM app.collection_reservations r
          WHERE r.obligation_id = o.id
            AND r.state IN ('PROCESSING','COMPLETED')
      )
    ORDER BY o.id::text
    LIMIT greatest(1, least(COALESCE(p_limit, 100), 500));
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.collection_attempt_work_page(p_cursor text, p_limit integer)
RETURNS TABLE(resource_id text, organization_id uuid)
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = pg_catalog, app
AS $$
    SELECT a.id::text, o.supplier_organization_id
    FROM app.collection_attempts a
    JOIN app.obligations o ON o.id = a.obligation_id
    WHERE a.state IN ('PENDING','SUBMITTED','UNKNOWN')
      AND a.id::text > COALESCE(p_cursor, '')
    ORDER BY a.id::text
    LIMIT greatest(1, least(COALESCE(p_limit, 100), 500));
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.collection_identity_by_attempt(p_attempt_id uuid)
RETURNS TABLE(obligation_id uuid, organization_id uuid)
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = pg_catalog, app
AS $$
    SELECT idx.obligation_id, o.supplier_organization_id
    FROM app.collection_attempt_index idx
    JOIN app.obligations o ON o.id = idx.obligation_id
    WHERE idx.attempt_id = p_attempt_id;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.collection_identity_by_external(p_external_reference text)
RETURNS TABLE(obligation_id uuid, organization_id uuid)
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = pg_catalog, app
AS $$
    SELECT idx.obligation_id, o.supplier_organization_id
    FROM app.collection_attempt_index idx
    JOIN app.obligations o ON o.id = idx.obligation_id
    WHERE idx.external_reference = p_external_reference;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.enqueue_pre_debit_notices()
RETURNS bigint
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, app
AS $$
DECLARE
    affected bigint;
BEGIN
    INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key)
    SELECT 'obligation',s.obligation_id::text,'notification.requested',
           jsonb_build_object('event','PRE_DEBIT_NOTICE','schedule_item_id',i.id,'amount_kobo',i.principal_due_kobo),
           app.collection_notice_key(i)
    FROM app.schedule_items i
    JOIN app.repayment_schedules s ON s.id=i.schedule_id
    JOIN app.obligations o ON o.id=s.obligation_id
    WHERE o.lifecycle_status='ACTIVE'
      AND o.outstanding_kobo>0
      AND i.state NOT IN ('PAID','CANCELLED')
      AND i.principal_due_kobo>i.allocated_kobo
      AND i.collection_at<=now()+interval '31 days'
    ON CONFLICT(idempotency_key) DO NOTHING;
    GET DIAGNOSTICS affected = ROW_COUNT;
    RETURN affected;
END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.enqueue_due_payment_notices(p_upcoming_days integer)
RETURNS bigint
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, app
AS $$
DECLARE
    affected bigint;
    horizon integer := greatest(0, least(COALESCE(p_upcoming_days, 0), 90));
BEGIN
    INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key)
    SELECT 'obligation',s.obligation_id::text,'notification.requested',
           jsonb_build_object('event',CASE WHEN i.due_at<=now() THEN 'PAYMENT_DUE' ELSE 'UPCOMING_DUE' END,'schedule_item_id',i.id,'amount_kobo',i.principal_due_kobo-i.allocated_kobo),
           CASE WHEN i.due_at<=now() THEN 'due:' ELSE 'upcoming:' END||i.id::text||':'||i.due_at::text
    FROM app.schedule_items i
    JOIN app.repayment_schedules s ON s.id=i.schedule_id
    JOIN app.obligations o ON o.id=s.obligation_id
    WHERE o.lifecycle_status='ACTIVE'
      AND o.outstanding_kobo>0
      AND i.state NOT IN ('PAID','CANCELLED')
      AND i.principal_due_kobo>i.allocated_kobo
      AND i.due_at<=now()+make_interval(days=>horizon)
    ON CONFLICT(idempotency_key) DO NOTHING;
    GET DIAGNOSTICS affected = ROW_COUNT;
    RETURN affected;
END
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.collection_due_work_page(text,integer) FROM PUBLIC;
REVOKE ALL ON FUNCTION app.collection_attempt_work_page(text,integer) FROM PUBLIC;
REVOKE ALL ON FUNCTION app.collection_identity_by_attempt(uuid) FROM PUBLIC;
REVOKE ALL ON FUNCTION app.collection_identity_by_external(text) FROM PUBLIC;
REVOKE ALL ON FUNCTION app.enqueue_pre_debit_notices() FROM PUBLIC;
REVOKE ALL ON FUNCTION app.enqueue_due_payment_notices(integer) FROM PUBLIC;

-- +goose Down
DROP FUNCTION IF EXISTS app.enqueue_due_payment_notices(integer);
DROP FUNCTION IF EXISTS app.enqueue_pre_debit_notices();
DROP FUNCTION IF EXISTS app.collection_identity_by_external(text);
DROP FUNCTION IF EXISTS app.collection_identity_by_attempt(uuid);
DROP FUNCTION IF EXISTS app.collection_attempt_work_page(text,integer);
DROP FUNCTION IF EXISTS app.collection_due_work_page(text,integer);
