-- +goose Up

-- Document metadata and support timelines are now written by the durable
-- repository adapters. The application and worker roles are trusted service
-- boundaries; tenant authorization is still enforced by the HTTP/service
-- layer before these repositories are called.
ALTER TABLE app.documents ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS document_runtime_access ON app.documents;
CREATE POLICY document_runtime_access ON app.documents
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));

ALTER TABLE app.support_cases ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS support_case_runtime_access ON app.support_cases;
CREATE POLICY support_case_runtime_access ON app.support_cases
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));

ALTER TABLE app.support_case_events ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS support_case_event_runtime_access ON app.support_case_events;
CREATE POLICY support_case_event_runtime_access ON app.support_case_events
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));

-- +goose Down
DROP POLICY IF EXISTS support_case_event_runtime_access ON app.support_case_events;
DROP POLICY IF EXISTS support_case_runtime_access ON app.support_cases;
DROP POLICY IF EXISTS document_runtime_access ON app.documents;
