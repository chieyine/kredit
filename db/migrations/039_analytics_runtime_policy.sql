-- +goose Up
DROP POLICY IF EXISTS analytics_runtime_access ON app.analytics_events;
CREATE POLICY analytics_runtime_access ON app.analytics_events USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));

-- +goose Down
DROP POLICY IF EXISTS analytics_runtime_access ON app.analytics_events;
