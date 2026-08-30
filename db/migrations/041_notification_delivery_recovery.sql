-- +goose Up
ALTER TABLE app.notifications
  ADD COLUMN IF NOT EXISTS delivery_attempts integer NOT NULL DEFAULT 0 CHECK (delivery_attempts >= 0),
  ADD COLUMN IF NOT EXISTS next_attempt_at timestamptz;
CREATE INDEX IF NOT EXISTS notifications_delivery_due_idx
  ON app.notifications (COALESCE(next_attempt_at, scheduled_at, lease_expires_at, updated_at))
  WHERE state IN ('scheduled','failed','sending') AND delivery_attempts < 8;

-- +goose Down
DROP INDEX IF EXISTS app.notifications_delivery_due_idx;
ALTER TABLE app.notifications DROP COLUMN IF EXISTS next_attempt_at, DROP COLUMN IF EXISTS delivery_attempts;
