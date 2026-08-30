-- +goose Up
DROP POLICY IF EXISTS dispute_runtime_access ON app.disputes;
CREATE POLICY dispute_runtime_access ON app.disputes
  USING (current_user IN ('kredit_app','kredit_worker'))
  WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
DROP POLICY IF EXISTS dispute_evidence_runtime_access ON app.dispute_evidence;
CREATE POLICY dispute_evidence_runtime_access ON app.dispute_evidence
  USING (current_user IN ('kredit_app','kredit_worker'))
  WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
DROP POLICY IF EXISTS dispute_decision_runtime_access ON app.dispute_decisions;
CREATE POLICY dispute_decision_runtime_access ON app.dispute_decisions
  USING (current_user IN ('kredit_app','kredit_worker'))
  WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
DROP POLICY IF EXISTS operation_runtime_access ON app.operation_actions;
CREATE POLICY operation_runtime_access ON app.operation_actions
  USING (current_user IN ('kredit_app','kredit_worker'))
  WITH CHECK (current_user IN ('kredit_app','kredit_worker'));

-- +goose Down
DROP POLICY IF EXISTS operation_runtime_access ON app.operation_actions;
DROP POLICY IF EXISTS dispute_decision_runtime_access ON app.dispute_decisions;
DROP POLICY IF EXISTS dispute_evidence_runtime_access ON app.dispute_evidence;
DROP POLICY IF EXISTS dispute_runtime_access ON app.disputes;
