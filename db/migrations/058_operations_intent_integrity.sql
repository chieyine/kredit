-- +goose Up
ALTER TABLE app.operations_commands ADD COLUMN request_hash text;
-- Existing commands remain immutable. Their missing request hash cannot be
-- treated as evidence for a new request; callers must inspect the saved result.
CREATE POLICY settlement_runtime_access ON app.settlement_events USING(current_user IN ('kredit_app','kredit_worker')) WITH CHECK(current_user IN ('kredit_app','kredit_worker'));
-- +goose Down
DROP POLICY settlement_runtime_access ON app.settlement_events;
ALTER TABLE app.operations_commands DROP COLUMN request_hash;
