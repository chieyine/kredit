-- +goose Up

-- Keep support/worker tables inaccessible to arbitrary database users while
-- allowing the dedicated runtime roles to perform their bounded operations.
ALTER TABLE app.provider_webhook_inbox ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS provider_webhook_runtime_access ON app.provider_webhook_inbox;
CREATE POLICY provider_webhook_runtime_access ON app.provider_webhook_inbox
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));

ALTER TABLE app.job_dead_letters ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS dead_letter_runtime_access ON app.job_dead_letters;
CREATE POLICY dead_letter_runtime_access ON app.job_dead_letters
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));

-- Existing support policies were intentionally narrow. Extend them only to
-- the worker role; tenant-facing tables retain their user/organization rules.
CREATE POLICY operation_action_worker_access ON app.operation_actions
    USING (current_user = 'kredit_worker');
CREATE POLICY messaging_worker_access ON app.messaging_events
    USING (current_user = 'kredit_worker');
CREATE POLICY analytics_worker_access ON app.analytics_events
    USING (current_user = 'kredit_worker');
CREATE POLICY collection_event_worker_access ON app.collection_provider_events
    USING (current_user = 'kredit_worker');
CREATE POLICY provider_approval_worker_access ON app.provider_approvals
    USING (current_user = 'kredit_worker');
CREATE POLICY provider_event_worker_access ON app.provider_events
    USING (current_user = 'kredit_worker');
CREATE POLICY provider_reconciliation_worker_access ON app.provider_reconciliation_events
    USING (current_user = 'kredit_worker');
CREATE POLICY release_evidence_worker_access ON app.release_evidence
    USING (current_user = 'kredit_worker');
CREATE POLICY pilot_limits_worker_access ON app.pilot_limit_configs
    USING (current_user = 'kredit_worker');

-- +goose Down
DROP POLICY IF EXISTS dead_letter_runtime_access ON app.job_dead_letters;
ALTER TABLE app.job_dead_letters DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS provider_webhook_runtime_access ON app.provider_webhook_inbox;
ALTER TABLE app.provider_webhook_inbox DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS operation_action_worker_access ON app.operation_actions;
DROP POLICY IF EXISTS messaging_worker_access ON app.messaging_events;
DROP POLICY IF EXISTS analytics_worker_access ON app.analytics_events;
DROP POLICY IF EXISTS collection_event_worker_access ON app.collection_provider_events;
DROP POLICY IF EXISTS provider_approval_worker_access ON app.provider_approvals;
DROP POLICY IF EXISTS provider_event_worker_access ON app.provider_events;
DROP POLICY IF EXISTS provider_reconciliation_worker_access ON app.provider_reconciliation_events;
DROP POLICY IF EXISTS release_evidence_worker_access ON app.release_evidence;
DROP POLICY IF EXISTS pilot_limits_worker_access ON app.pilot_limit_configs;
