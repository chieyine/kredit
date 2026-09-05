-- +goose Up

-- Migration 035 added service-wide policies to the collection repository.
-- PostgreSQL combines permissive policies with OR, so those policies must be
-- removed or they bypass the tenant checks added in 081.
DROP POLICY IF EXISTS collection_reservation_runtime_access ON app.collection_reservations;
DROP POLICY IF EXISTS collection_attempt_runtime_access ON app.collection_attempts;
DROP POLICY IF EXISTS collection_snapshot_runtime_access ON app.collection_aggregate_snapshots;
DROP POLICY IF EXISTS collection_index_runtime_access ON app.collection_attempt_index;

DROP POLICY IF EXISTS collection_snapshot_runtime_tenant ON app.collection_aggregate_snapshots;
CREATE POLICY collection_snapshot_runtime_tenant ON app.collection_aggregate_snapshots
    USING (
        current_user IN ('kredit_app', 'kredit_worker')
        AND EXISTS (
            SELECT 1 FROM app.obligations o
            WHERE o.id = collection_aggregate_snapshots.obligation_id
              AND o.supplier_organization_id = app.current_organization_id()
        )
    )
    WITH CHECK (
        current_user IN ('kredit_app', 'kredit_worker')
        AND EXISTS (
            SELECT 1 FROM app.obligations o
            WHERE o.id = collection_aggregate_snapshots.obligation_id
              AND o.supplier_organization_id = app.current_organization_id()
        )
    );

DROP POLICY IF EXISTS collection_index_runtime_tenant ON app.collection_attempt_index;
CREATE POLICY collection_index_runtime_tenant ON app.collection_attempt_index
    USING (
        current_user IN ('kredit_app', 'kredit_worker')
        AND EXISTS (
            SELECT 1 FROM app.obligations o
            WHERE o.id = collection_attempt_index.obligation_id
              AND o.supplier_organization_id = app.current_organization_id()
        )
    )
    WITH CHECK (
        current_user IN ('kredit_app', 'kredit_worker')
        AND EXISTS (
            SELECT 1 FROM app.obligations o
            WHERE o.id = collection_attempt_index.obligation_id
              AND o.supplier_organization_id = app.current_organization_id()
        )
    );

-- +goose Down
DROP POLICY IF EXISTS collection_index_runtime_tenant ON app.collection_attempt_index;
DROP POLICY IF EXISTS collection_snapshot_runtime_tenant ON app.collection_aggregate_snapshots;

CREATE POLICY collection_reservation_runtime_access ON app.collection_reservations
    USING (current_user IN ('kredit_app','kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY collection_attempt_runtime_access ON app.collection_attempts
    USING (current_user IN ('kredit_app','kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY collection_snapshot_runtime_access ON app.collection_aggregate_snapshots
    USING (current_user IN ('kredit_app','kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY collection_index_runtime_access ON app.collection_attempt_index
    USING (current_user IN ('kredit_app','kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
