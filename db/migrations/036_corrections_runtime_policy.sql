-- +goose Up
DROP POLICY IF EXISTS correction_runtime_access ON app.correction_requests;
CREATE POLICY correction_runtime_access ON app.correction_requests
  USING (current_user IN ('kredit_app','kredit_worker'))
  WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
DROP POLICY IF EXISTS correction_decision_runtime_access ON app.correction_decisions;
CREATE POLICY correction_decision_runtime_access ON app.correction_decisions
  USING (current_user IN ('kredit_app','kredit_worker'))
  WITH CHECK (current_user IN ('kredit_app','kredit_worker'));

-- +goose Down
DROP POLICY IF EXISTS correction_decision_runtime_access ON app.correction_decisions;
DROP POLICY IF EXISTS correction_runtime_access ON app.correction_requests;
