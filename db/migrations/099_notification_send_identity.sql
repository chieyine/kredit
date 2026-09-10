-- +goose Up
ALTER TABLE app.notifications ADD COLUMN send_started_at timestamptz;
-- An existing attempted send may have reached the provider even without a
-- confirmed response. Keep that exact message on retry.
UPDATE app.notifications SET send_started_at=COALESCE(sent_at,failed_at,updated_at,now())
  WHERE state IN ('sending','sent','delivered','read','failed');

-- +goose Down
-- Rolling back would regenerate signed message content under an existing
-- provider request identity. Retain the evidence and roll forward.
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'notification send identity requires a forward migration'; END $$;
-- +goose StatementEnd
