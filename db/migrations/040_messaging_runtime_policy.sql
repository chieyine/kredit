-- +goose Up
DROP POLICY IF EXISTS messaging_runtime_access ON app.messaging_events;
CREATE POLICY messaging_runtime_access ON app.messaging_events USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));

-- +goose Down
DROP POLICY IF EXISTS messaging_runtime_access ON app.messaging_events;
