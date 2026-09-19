-- +goose Up
DROP POLICY dispute_runtime_access ON app.disputes;
DROP POLICY dispute_evidence_runtime_access ON app.dispute_evidence;
DROP POLICY dispute_decision_runtime_access ON app.dispute_decisions;
-- Child policies already inherit visible dispute IDs. Reviewer access is
-- re-evaluated against active platform assignments on every transaction.
CREATE POLICY dispute_reviewer_access ON app.disputes
USING (CASE WHEN current_user='kredit_app' THEN app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','dispute_reviewer']) ELSE false END)
WITH CHECK (CASE WHEN current_user='kredit_app' THEN app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','dispute_reviewer']) ELSE false END);

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Dispute isolation requires forward recovery'; END $$;
-- +goose StatementEnd
