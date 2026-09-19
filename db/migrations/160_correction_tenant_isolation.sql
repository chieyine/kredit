-- +goose Up
DROP POLICY correction_runtime_access ON app.correction_requests;
DROP POLICY correction_decision_runtime_access ON app.correction_decisions;

-- Customers can submit and read their own history annotations. Organization
-- reviewers retain the existing organization policy and application checks.
CREATE POLICY correction_requester_access ON app.correction_requests
USING (requested_by=app.current_user_id())
WITH CHECK (requested_by=app.current_user_id());
CREATE POLICY correction_decision_requester_read ON app.correction_decisions FOR SELECT
USING (EXISTS (SELECT 1 FROM app.correction_requests r
 WHERE r.id=request_id AND r.requested_by=app.current_user_id()));

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Correction isolation requires forward recovery'; END $$;
-- +goose StatementEnd
