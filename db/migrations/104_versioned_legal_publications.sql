-- +goose Up
-- Extend the reviewed publication workflow to versioned legal documents.
ALTER TABLE app.platform_settings DROP CONSTRAINT platform_settings_category_known;
ALTER TABLE app.platform_settings ADD CONSTRAINT platform_settings_category_known
CHECK (category IN ('features', 'governance') OR
 (category='integrations' AND is_secret AND key IN (
  'integrations.notifications.email','integrations.notifications.sms','integrations.notifications.whatsapp',
  'integrations.runtime.identity','integrations.runtime.mono','integrations.runtime.scanner',
  'integrations.runtime.collections','integrations.runtime.launch'
 )) OR
 (category='website' AND NOT is_secret AND key IN ('website.home','website.faq','website.pricing','website.terms','website.privacy','website.complaints')));

ALTER TABLE app.drawdowns ADD COLUMN legal_versions JSONB
 CHECK (legal_versions IS NULL OR COALESCE(
 jsonb_typeof(legal_versions) = 'object'
 AND jsonb_typeof(legal_versions->'terms_version') = 'string'
 AND length(btrim(legal_versions->>'terms_version')) > 0
 AND jsonb_typeof(legal_versions->'privacy_version') = 'string'
 AND length(btrim(legal_versions->>'privacy_version')) > 0, false));
-- Preserve legacy hashes with NULL; new offers record these version references.
-- +goose StatementBegin
CREATE FUNCTION app.guard_drawdown_legal_versions() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.legal_versions IS DISTINCT FROM OLD.legal_versions THEN
  RAISE EXCEPTION 'Accepted drawdown legal versions are immutable';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER drawdown_legal_versions_immutable BEFORE UPDATE OF legal_versions ON app.drawdowns
FOR EACH ROW EXECUTE FUNCTION app.guard_drawdown_legal_versions();

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 RAISE EXCEPTION 'Website publications and their history must be preserved; use a forward migration';
END $$;
-- +goose StatementEnd
