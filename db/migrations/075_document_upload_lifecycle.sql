-- +goose Up

ALTER TABLE app.documents
    ADD COLUMN upload_completed_at TIMESTAMPTZ,
    ADD COLUMN upload_expires_at TIMESTAMPTZ,
    ADD COLUMN scan_attempts INTEGER NOT NULL DEFAULT 0 CHECK (scan_attempts >= 0),
    ADD COLUMN scan_lease_until TIMESTAMPTZ;

-- Legacy server-side uploads have a digest and were complete when inserted.
-- Empty-digest rows are direct-upload reservations and get a bounded lifetime.
UPDATE app.documents
SET upload_completed_at = created_at
WHERE sha256 <> '';

UPDATE app.documents
SET upload_expires_at = created_at + interval '10 minutes'
WHERE upload_completed_at IS NULL;

CREATE INDEX documents_scan_claim_idx
ON app.documents(created_at)
WHERE scan_state = 'PENDING' AND upload_completed_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS app.documents_scan_claim_idx;
ALTER TABLE app.documents
    DROP COLUMN IF EXISTS scan_lease_until,
    DROP COLUMN IF EXISTS scan_attempts,
    DROP COLUMN IF EXISTS upload_expires_at,
    DROP COLUMN IF EXISTS upload_completed_at;
