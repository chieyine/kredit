-- +goose Up
-- A missing governance row is an infrastructure error, never solo-owner consent.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.current_governance_mode() RETURNS text
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE configured text;
BEGIN
  SELECT mode INTO STRICT configured FROM app.platform_governance WHERE id='singleton';
  RETURN configured;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- Keep failure closed during code rollback.
SELECT 1;
