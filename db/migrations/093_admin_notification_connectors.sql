-- +goose Up
-- Only connectors with a runtime consumer may be stored. Endpoint and token
-- share a single encrypted value; no credentials are seeded by a migration.
ALTER TABLE app.platform_settings DROP CONSTRAINT platform_settings_category_known;
ALTER TABLE app.platform_settings ADD CONSTRAINT platform_settings_category_known
CHECK (category IN ('features', 'governance') OR
  (category = 'integrations' AND is_secret AND key IN (
    'integrations.notifications.email',
    'integrations.notifications.sms',
    'integrations.notifications.whatsapp'
  )));

-- +goose Down
-- Preserve encrypted settings and their immutable history on code rollback.
-- Earlier application versions ignore these registered connector keys.
SELECT 1;
