-- +goose Up
-- Subject policies remain in force. Cross-subject review requires a current,
-- active privacy reviewer; a shared runtime database role is not authority.
DROP POLICY privacy_request_runtime ON app.privacy_requests;
DROP POLICY privacy_event_runtime ON app.privacy_request_events;
DROP POLICY privacy_export_runtime ON app.privacy_exports;
DROP POLICY restriction_runtime ON app.processing_restrictions;

CREATE POLICY privacy_reviewer ON app.privacy_requests
USING (CASE WHEN current_user='kredit_app' THEN app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']) ELSE false END)
WITH CHECK (CASE WHEN current_user='kredit_app' THEN app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']) ELSE false END);
CREATE POLICY privacy_event_reviewer ON app.privacy_request_events
USING (CASE WHEN current_user='kredit_app' THEN app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']) ELSE false END)
WITH CHECK (CASE WHEN current_user='kredit_app' THEN app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']) ELSE false END);
CREATE POLICY privacy_export_reviewer ON app.privacy_exports
USING (CASE WHEN current_user='kredit_app' THEN app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']) ELSE false END)
WITH CHECK (CASE WHEN current_user='kredit_app' THEN app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']) ELSE false END);
CREATE POLICY restriction_reviewer ON app.processing_restrictions
USING (CASE WHEN current_user='kredit_app' THEN app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']) ELSE false END)
WITH CHECK (CASE WHEN current_user='kredit_app' THEN app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']) ELSE false END);

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Privacy isolation requires forward recovery'; END $$;
-- +goose StatementEnd
