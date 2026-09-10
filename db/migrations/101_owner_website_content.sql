-- +goose Up
-- Public copy is non-secret, but only these consumed editor slots are allowed.
ALTER TABLE app.platform_settings DROP CONSTRAINT platform_settings_category_known;
ALTER TABLE app.platform_settings ADD CONSTRAINT platform_settings_category_known
CHECK (category IN ('features', 'governance') OR
 (category='integrations' AND is_secret AND key IN (
  'integrations.notifications.email','integrations.notifications.sms','integrations.notifications.whatsapp',
  'integrations.runtime.identity','integrations.runtime.mono','integrations.runtime.scanner',
  'integrations.runtime.collections','integrations.runtime.launch'
 )) OR
 (category='website' AND NOT is_secret AND key IN ('website.home','website.faq','website.pricing')));

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 RAISE EXCEPTION 'Website publications and their history must be preserved; use a forward migration';
END $$;
-- +goose StatementEnd
