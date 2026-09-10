-- +goose Up
ALTER TABLE app.notifications ADD COLUMN event_fingerprint text
  CHECK (event_fingerprint IS NULL OR event_fingerprint ~ '^[0-9a-f]{64}$');

-- A delivery can update its lease, refreshed secure link and provider result,
-- but it cannot acquire a different source intent after it was created.
-- +goose StatementBegin
CREATE FUNCTION app.preserve_notification_intent() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.event_fingerprint IS DISTINCT FROM OLD.event_fingerprint THEN
    RAISE EXCEPTION 'notification intent is immutable';
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER notifications_intent_immutable BEFORE UPDATE ON app.notifications
  FOR EACH ROW EXECUTE FUNCTION app.preserve_notification_intent();

-- +goose Down
-- Roll back application code without discarding delivery identity evidence.
-- +goose StatementBegin
DO $$ BEGIN
  RAISE EXCEPTION 'notification intent evidence cannot be discarded; use a forward migration';
END $$;
-- +goose StatementEnd
