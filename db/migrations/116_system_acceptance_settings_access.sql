-- +goose Up
-- Workers receive only these two non-secret controls, not settings-table access.
-- +goose StatementBegin
CREATE FUNCTION app.system_acceptance_settings() RETURNS TABLE(enabled boolean,hours bigint)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT COALESCE((SELECT value='true'::jsonb FROM app.platform_settings WHERE key='features.system_acceptance'),false),
 COALESCE((SELECT (value::text)::bigint FROM app.platform_settings WHERE key='automation.system_acceptance_hours'),72);
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.system_acceptance_settings() FROM PUBLIC;
ALTER FUNCTION app.guard_system_acceptance_policy() SECURITY DEFINER;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN GRANT EXECUTE ON FUNCTION app.system_acceptance_settings() TO kredit_worker; END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Automatic recognition controls require forward recovery'; END $$;
-- +goose StatementEnd
