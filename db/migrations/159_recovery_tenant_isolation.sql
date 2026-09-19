-- +goose Up
DROP POLICY recovery_code_runtime ON app.account_recovery_codes;
DROP POLICY recovery_request_runtime ON app.account_recovery_requests;
DROP POLICY recovery_evidence_runtime ON app.account_recovery_evidence;
DROP POLICY recovery_event_runtime ON app.account_recovery_events;

CREATE POLICY recovery_request_reviewer ON app.account_recovery_requests
USING (CASE WHEN current_user='kredit_app' THEN app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']) ELSE false END)
WITH CHECK (CASE WHEN current_user='kredit_app' THEN app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']) ELSE false END);
CREATE POLICY recovery_event_reviewer ON app.account_recovery_events
USING (CASE WHEN current_user='kredit_app' THEN app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']) ELSE false END)
WITH CHECK (CASE WHEN current_user='kredit_app' THEN app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']) ELSE false END);

-- A private recovery link must work before authentication. Return only the
-- subject identifier, never code hashes, evidence or completion tokens.
-- +goose StatementBegin
CREATE FUNCTION app.recovery_subject(request uuid) RETURNS uuid
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app
SET row_security=off AS $$
 SELECT target_user_id FROM app.account_recovery_requests WHERE id=request AND expires_at>now();
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.recovery_subject(uuid) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
GRANT EXECUTE ON FUNCTION app.recovery_subject(uuid) TO kredit_app;
END IF; END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Recovery isolation requires forward recovery'; END $$;
-- +goose StatementEnd
