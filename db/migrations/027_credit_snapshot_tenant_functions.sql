-- +goose Up

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.credit_snapshot_by_id(p_request_id TEXT)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = app, pg_catalog
AS $$
    SELECT aggregate
    FROM app.credit_aggregate_snapshots
    WHERE credit_request_id = p_request_id
      AND (
          supplier_organization_id = NULLIF(current_setting('app.current_organization_id', true), '')
          OR buyer_user_id = NULLIF(current_setting('app.current_user_id', true), '')
      )
$$;
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
      AND (
          supplier_organization_id = NULLIF(current_setting('app.current_organization_id', true), '')
          OR buyer_user_id = NULLIF(current_setting('app.current_user_id', true), '')
      )
    LIMIT 1
$$;
-- +goose StatementEnd

-- +goose Down

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
