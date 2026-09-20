-- +goose Up
ALTER POLICY distributor_import_row_authority ON app.distributor_import_rows
WITH CHECK (
 created_by=app.current_user_id() AND EXISTS(
 SELECT 1 FROM app.distributor_import_batches b JOIN app.buyer_invitations i ON i.organization_id=b.organization_id
 WHERE b.id=distributor_import_rows.batch_id AND b.state IN ('approved','completed')
 AND distributor_import_rows.row_number<=b.row_count AND i.id=distributor_import_rows.invitation_id
 ));
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Import row integrity requires forward recovery'; END $$;
-- +goose StatementEnd
