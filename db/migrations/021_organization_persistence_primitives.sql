-- +goose Up

-- Supplier pilot limits need a global count, but tenant RLS must not be
-- bypassed by ordinary application queries. This function exposes only the
-- aggregate needed by the guarded organization-creation path.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.organization_count()
RETURNS BIGINT
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = app, pg_catalog
AS $$
    SELECT count(*)::BIGINT FROM app.organizations
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.organization_count() FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        GRANT EXECUTE ON FUNCTION app.organization_count() TO kredit_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS app.organization_count();
