-- +goose Up
-- Return only the identity needed to send private recovery instructions.
-- HTTP responses never contain this identity or indicate account existence.
CREATE FUNCTION app.recovery_account(p_identifier text,p_channel text) RETURNS TABLE(user_id uuid)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT id FROM app.users WHERE status IN ('active','locked') AND
 ((p_channel='email' AND normalized_email=p_identifier) OR (p_channel='phone' AND normalized_phone=p_identifier))
$$;
REVOKE ALL ON FUNCTION app.recovery_account(text,text) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT EXECUTE ON FUNCTION app.recovery_account(text,text) TO kredit_app; END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
DROP FUNCTION app.recovery_account(text,text);
