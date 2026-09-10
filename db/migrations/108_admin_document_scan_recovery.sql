-- +goose Up
ALTER TABLE app.documents ADD COLUMN scan_review_version bigint NOT NULL DEFAULT 1 CHECK (scan_review_version > 0);
ALTER TABLE app.operations_commands DROP CONSTRAINT operations_commands_command_type_check;
ALTER TABLE app.operations_commands ADD CONSTRAINT operations_commands_command_type_check CHECK (command_type IN (
 'retry_job','retry_webhook','retry_document_scan','suspend_user','restore_user','suspend_organization','restore_organization',
 'place_risk_hold','lift_risk_hold','request_reconciliation','resolve_unknown_submission','retry_collection','cancel_collection'));

CREATE POLICY document_admin_read ON app.documents FOR SELECT
 USING (app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin']));
CREATE POLICY document_admin_recovery ON app.documents FOR UPDATE
 USING (app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin']))
 WITH CHECK (app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin']));

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 RAISE EXCEPTION 'Document recovery commands and their versions must be retained; use a forward migration';
END $$;
-- +goose StatementEnd
