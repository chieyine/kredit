-- +goose Up
ALTER TABLE app.notifications ADD COLUMN created_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE app.notifications ADD COLUMN provider_checked_at timestamptz;
CREATE INDEX notifications_provider_lookup_due ON app.notifications(provider_checked_at NULLS FIRST,created_at) WHERE channel='email' AND state='sent';
-- +goose Down
DROP INDEX app.notifications_provider_lookup_due;
ALTER TABLE app.notifications DROP COLUMN provider_checked_at, DROP COLUMN created_at;
