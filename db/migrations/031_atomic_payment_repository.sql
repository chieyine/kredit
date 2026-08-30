-- +goose Up

-- System-originated collection jobs have no human user UUID. Preserve the
-- exact actor reference without inventing a user identity.
ALTER TABLE app.payments
    ALTER COLUMN recorded_by DROP NOT NULL,
    ADD COLUMN IF NOT EXISTS recorded_by_reference TEXT,
    ADD COLUMN IF NOT EXISTS collection_fee_kobo BIGINT NOT NULL DEFAULT 0
        CHECK (collection_fee_kobo >= 0);

UPDATE app.payments
SET recorded_by_reference = recorded_by::text
WHERE recorded_by_reference IS NULL;

ALTER TABLE app.payments
    ALTER COLUMN recorded_by_reference SET NOT NULL;

ALTER TABLE app.payment_allocations
    ADD COLUMN IF NOT EXISTS schedule_item_id UUID REFERENCES app.schedule_items(id);

-- API and worker roles are trusted service boundaries. Browser clients never
-- receive these credentials; authorization remains in the HTTP/service layer.
DROP POLICY IF EXISTS payment_runtime_access ON app.payments;
CREATE POLICY payment_runtime_access ON app.payments
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));

DROP POLICY IF EXISTS allocation_runtime_access ON app.payment_allocations;
CREATE POLICY allocation_runtime_access ON app.payment_allocations
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));

DROP POLICY IF EXISTS fee_runtime_access ON app.fees;
CREATE POLICY fee_runtime_access ON app.fees
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));

DROP POLICY IF EXISTS obligation_runtime_access ON app.obligations;
CREATE POLICY obligation_runtime_access ON app.obligations
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));

DROP POLICY IF EXISTS credit_snapshot_runtime_access ON app.credit_aggregate_snapshots;
CREATE POLICY credit_snapshot_runtime_access ON app.credit_aggregate_snapshots
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));

-- +goose Down
DROP POLICY IF EXISTS credit_snapshot_runtime_access ON app.credit_aggregate_snapshots;
DROP POLICY IF EXISTS obligation_runtime_access ON app.obligations;
DROP POLICY IF EXISTS fee_runtime_access ON app.fees;
DROP POLICY IF EXISTS allocation_runtime_access ON app.payment_allocations;
DROP POLICY IF EXISTS payment_runtime_access ON app.payments;
UPDATE app.payments SET recorded_by = buyer_user_id WHERE recorded_by IS NULL;
ALTER TABLE app.payment_allocations DROP COLUMN IF EXISTS schedule_item_id;
ALTER TABLE app.payments
    DROP COLUMN IF EXISTS collection_fee_kobo,
    DROP COLUMN IF EXISTS recorded_by_reference,
    ALTER COLUMN recorded_by SET NOT NULL;
