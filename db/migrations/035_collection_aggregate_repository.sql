-- +goose Up
ALTER TABLE app.collection_attempts
    ADD COLUMN IF NOT EXISTS settlement_state TEXT,
    ADD COLUMN IF NOT EXISTS settlement_reference TEXT;

CREATE TABLE IF NOT EXISTS app.collection_aggregate_snapshots (
    obligation_id UUID PRIMARY KEY REFERENCES app.obligations(id),
    aggregate JSONB NOT NULL DEFAULT '{}'::jsonb,
    version BIGINT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS app.collection_attempt_index (
    attempt_id UUID PRIMARY KEY,
    external_reference TEXT NOT NULL UNIQUE,
    obligation_id UUID NOT NULL REFERENCES app.obligations(id)
);

ALTER TABLE app.collection_reservations ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS collection_reservation_runtime_access ON app.collection_reservations;
CREATE POLICY collection_reservation_runtime_access ON app.collection_reservations USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
ALTER TABLE app.collection_attempts ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS collection_attempt_runtime_access ON app.collection_attempts;
CREATE POLICY collection_attempt_runtime_access ON app.collection_attempts USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
ALTER TABLE app.collection_aggregate_snapshots ENABLE ROW LEVEL SECURITY;
CREATE POLICY collection_snapshot_runtime_access ON app.collection_aggregate_snapshots USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
ALTER TABLE app.collection_attempt_index ENABLE ROW LEVEL SECURITY;
CREATE POLICY collection_index_runtime_access ON app.collection_attempt_index USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));

-- +goose Down
DROP TABLE IF EXISTS app.collection_attempt_index;
DROP TABLE IF EXISTS app.collection_aggregate_snapshots;
DROP POLICY IF EXISTS collection_attempt_runtime_access ON app.collection_attempts;
DROP POLICY IF EXISTS collection_reservation_runtime_access ON app.collection_reservations;
ALTER TABLE app.collection_attempts DROP COLUMN IF EXISTS settlement_reference, DROP COLUMN IF EXISTS settlement_state;
