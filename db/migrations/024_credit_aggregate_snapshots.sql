-- +goose Up

-- Transitional durable aggregate boundary for the credit lifecycle. The
-- normalized lifecycle tables remain the source schema; this immutable-shaped
-- snapshot lets the repository hydrate the complete aggregate after a process
-- restart while the remaining read models are migrated.
CREATE TABLE IF NOT EXISTS app.credit_aggregate_snapshots (
    credit_request_id TEXT PRIMARY KEY,
    supplier_organization_id TEXT NOT NULL,
    buyer_user_id TEXT NOT NULL,
    aggregate JSONB NOT NULL,
    version BIGINT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS credit_snapshot_supplier_idx ON app.credit_aggregate_snapshots (supplier_organization_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS credit_snapshot_buyer_idx ON app.credit_aggregate_snapshots (buyer_user_id, updated_at DESC);

ALTER TABLE app.credit_aggregate_snapshots ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS credit_snapshot_tenant_access ON app.credit_aggregate_snapshots;
CREATE POLICY credit_snapshot_tenant_access ON app.credit_aggregate_snapshots
    USING (
        supplier_organization_id = NULLIF(current_setting('app.current_organization_id', true), '')
        OR buyer_user_id = NULLIF(current_setting('app.current_user_id', true), '')
    )
    WITH CHECK (supplier_organization_id = NULLIF(current_setting('app.current_organization_id', true), ''));

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.credit_snapshot_by_id(p_request_id TEXT)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = app, pg_catalog
AS $$ SELECT aggregate FROM app.credit_aggregate_snapshots WHERE credit_request_id = p_request_id $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.credit_snapshot_by_obligation(p_obligation_id TEXT)
RETURNS TABLE (credit_request_id TEXT, aggregate JSONB)
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = app, pg_catalog
AS $$
    SELECT credit_request_id, aggregate
    FROM app.credit_aggregate_snapshots
    WHERE aggregate->'obligation'->>'id' = p_obligation_id
    LIMIT 1
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.credit_snapshot_by_id(TEXT) FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        GRANT EXECUTE ON FUNCTION app.credit_snapshot_by_id(TEXT) TO kredit_app;
        GRANT EXECUTE ON FUNCTION app.credit_snapshot_by_obligation(TEXT) TO kredit_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS app.credit_snapshot_by_id(TEXT);
DROP FUNCTION IF EXISTS app.credit_snapshot_by_obligation(TEXT);
DROP TABLE IF EXISTS app.credit_aggregate_snapshots;
