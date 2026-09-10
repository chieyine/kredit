-- +goose Up
ALTER TABLE app.provider_webhook_inbox ADD COLUMN lease_expires_at timestamptz;

-- Provider evidence is immutable; only its processing outcome may change.
-- +goose StatementBegin
CREATE FUNCTION app.guard_provider_webhook_identity() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  IF (NEW.provider,NEW.event_id,NEW.event_type,NEW.payload,NEW.signature_valid,
      NEW.provider_sequence,NEW.received_at)
     IS DISTINCT FROM
     (OLD.provider,OLD.event_id,OLD.event_type,OLD.payload,OLD.signature_valid,
      OLD.provider_sequence,OLD.received_at) THEN
    RAISE EXCEPTION 'provider webhook evidence is immutable';
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER provider_webhook_identity_guard BEFORE UPDATE ON app.provider_webhook_inbox
  FOR EACH ROW EXECUTE FUNCTION app.guard_provider_webhook_identity();

-- +goose Down
-- Preserve callback identity and claim fencing; an old worker must not run
-- against the versioned processing protocol.
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'provider webhook claims require a forward migration'; END $$;
-- +goose StatementEnd
