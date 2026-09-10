-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION app.reject_settings_history_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'platform settings history is append-only';
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER platform_settings_history_immutable
BEFORE UPDATE OR DELETE ON app.platform_settings_history
FOR EACH ROW EXECUTE FUNCTION app.reject_settings_history_mutation();
-- Runtime roles may be provisioned after a fresh schema install.
-- infra/postgres/roles.sql reapplies these restrictions after its baseline grants.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
    REVOKE UPDATE, DELETE, TRUNCATE ON app.platform_settings_history FROM kredit_app;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
    REVOKE INSERT, UPDATE, DELETE, TRUNCATE ON app.platform_settings_history FROM kredit_worker;
  END IF;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- Intentionally preserve the integrity boundary when rolling back application code.
SELECT 1;
