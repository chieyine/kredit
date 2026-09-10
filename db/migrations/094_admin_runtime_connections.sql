-- +goose Up
ALTER TABLE app.platform_settings DROP CONSTRAINT platform_settings_category_known;
ALTER TABLE app.platform_settings ADD CONSTRAINT platform_settings_category_known
CHECK (category IN ('features', 'governance') OR
 (category='integrations' AND is_secret AND key IN (
  'integrations.notifications.email','integrations.notifications.sms','integrations.notifications.whatsapp',
  'integrations.runtime.identity','integrations.runtime.mono','integrations.runtime.scanner',
  'integrations.runtime.collections','integrations.runtime.launch'
 )));
-- +goose Down
-- Retain encrypted configuration and its immutable history on code rollback.
SELECT 1;
