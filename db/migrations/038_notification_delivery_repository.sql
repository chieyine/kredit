-- +goose Up
ALTER TABLE app.notifications DROP CONSTRAINT IF EXISTS notifications_state_check;
ALTER TABLE app.notifications ADD CONSTRAINT notifications_state_check CHECK (state IN ('scheduled','sending','sent','delivered','read','failed','suppressed'));
ALTER TABLE app.notifications
  ADD COLUMN IF NOT EXISTS secure_link text,
  ADD COLUMN IF NOT EXISTS destination_ciphertext bytea,
  ADD COLUMN IF NOT EXISTS lease_expires_at timestamptz,
  ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();
DROP POLICY IF EXISTS notification_runtime_access ON app.notifications;
CREATE POLICY notification_runtime_access ON app.notifications USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
DROP POLICY IF EXISTS notification_preference_runtime_access ON app.notification_preferences;
CREATE POLICY notification_preference_runtime_access ON app.notification_preferences USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));

-- +goose Down
DROP POLICY IF EXISTS notification_preference_runtime_access ON app.notification_preferences;
DROP POLICY IF EXISTS notification_runtime_access ON app.notifications;
ALTER TABLE app.notifications DROP COLUMN IF EXISTS updated_at, DROP COLUMN IF EXISTS lease_expires_at, DROP COLUMN IF EXISTS destination_ciphertext, DROP COLUMN IF EXISTS secure_link;
ALTER TABLE app.notifications DROP CONSTRAINT IF EXISTS notifications_state_check;
ALTER TABLE app.notifications ADD CONSTRAINT notifications_state_check CHECK (state IN ('scheduled','sent','delivered','read','failed','suppressed'));
