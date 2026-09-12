-- +goose Up
CREATE TABLE app.message_routes (
 event_key text PRIMARY KEY CHECK(length(event_key)=64),
 channel text NOT NULL CHECK(channel IN ('email','sms','whatsapp')),
 adapter text NOT NULL,
 config_ciphertext text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE app.message_routes IS 'Immutable encrypted provider connection per notification event; preserves original routing across provider changes. No plaintext credentials or message contents.';
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT SELECT,INSERT ON app.message_routes TO kredit_app; REVOKE UPDATE,DELETE ON app.message_routes FROM kredit_app; END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN GRANT SELECT,INSERT ON app.message_routes TO kredit_worker; REVOKE UPDATE,DELETE ON app.message_routes FROM kredit_worker; END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Provider routes require forward recovery'; END $$;
-- +goose StatementEnd
