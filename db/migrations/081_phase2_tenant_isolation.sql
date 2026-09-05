-- +goose Up

-- Phase 2: the database, not only the HTTP/service layer, is the tenant
-- authorization boundary for supplier financial data. Runtime credentials are
-- shared service credentials, so a policy that checks only current_user is not
-- tenant isolation.

-- Payments / obligations / fees ------------------------------------------------
DROP POLICY IF EXISTS payment_runtime_access ON app.payments;
DROP POLICY IF EXISTS allocation_runtime_access ON app.payment_allocations;
DROP POLICY IF EXISTS fee_runtime_access ON app.fees;
DROP POLICY IF EXISTS obligation_runtime_access ON app.obligations;
DROP POLICY IF EXISTS credit_snapshot_runtime_access ON app.credit_aggregate_snapshots;

DROP POLICY IF EXISTS credit_request_runtime_tenant ON app.credit_requests;
CREATE POLICY credit_request_runtime_tenant ON app.credit_requests
    USING (
        current_user IN ('kredit_app', 'kredit_worker')
        AND supplier_organization_id = app.current_organization_id()
    )
    WITH CHECK (
        current_user IN ('kredit_app', 'kredit_worker')
        AND supplier_organization_id = app.current_organization_id()
    );

DROP POLICY IF EXISTS obligation_runtime_tenant ON app.obligations;
CREATE POLICY obligation_runtime_tenant ON app.obligations
    USING (
        current_user IN ('kredit_app', 'kredit_worker')
        AND supplier_organization_id = app.current_organization_id()
    )
    WITH CHECK (
        current_user IN ('kredit_app', 'kredit_worker')
        AND supplier_organization_id = app.current_organization_id()
    );

DROP POLICY IF EXISTS payment_runtime_tenant ON app.payments;
CREATE POLICY payment_runtime_tenant ON app.payments
    USING (
        current_user IN ('kredit_app', 'kredit_worker')
        AND supplier_organization_id = app.current_organization_id()
    )
    WITH CHECK (
        current_user IN ('kredit_app', 'kredit_worker')
        AND supplier_organization_id = app.current_organization_id()
        AND EXISTS (
            SELECT 1
            FROM app.obligations o
            WHERE o.id = payments.obligation_id
              AND o.supplier_organization_id = payments.supplier_organization_id
              AND o.supplier_organization_id = app.current_organization_id()
        )
    );

DROP POLICY IF EXISTS allocation_runtime_tenant ON app.payment_allocations;
CREATE POLICY allocation_runtime_tenant ON app.payment_allocations
    USING (
        current_user IN ('kredit_app', 'kredit_worker')
        AND EXISTS (
            SELECT 1
            FROM app.obligations o
            WHERE o.id = payment_allocations.obligation_id
              AND o.supplier_organization_id = app.current_organization_id()
        )
    )
    WITH CHECK (
        current_user IN ('kredit_app', 'kredit_worker')
        AND EXISTS (
            SELECT 1
            FROM app.payments p
            JOIN app.obligations o ON o.id = payment_allocations.obligation_id
            WHERE p.id = payment_allocations.payment_id
              AND p.obligation_id = payment_allocations.obligation_id
              AND p.supplier_organization_id = o.supplier_organization_id
              AND o.supplier_organization_id = app.current_organization_id()
        )
    );

DROP POLICY IF EXISTS fee_runtime_tenant ON app.fees;
CREATE POLICY fee_runtime_tenant ON app.fees
    USING (
        current_user IN ('kredit_app', 'kredit_worker')
        AND supplier_organization_id = app.current_organization_id()
    )
    WITH CHECK (
        current_user IN ('kredit_app', 'kredit_worker')
        AND supplier_organization_id = app.current_organization_id()
        AND EXISTS (
            SELECT 1 FROM app.obligations o
            WHERE o.id = fees.obligation_id
              AND o.supplier_organization_id = fees.supplier_organization_id
        )
    );

DROP POLICY IF EXISTS settlement_runtime_tenant ON app.settlement_events;
CREATE POLICY settlement_runtime_tenant ON app.settlement_events
    USING (
        current_user IN ('kredit_app', 'kredit_worker')
        AND supplier_organization_id = app.current_organization_id()
    )
    WITH CHECK (
        current_user IN ('kredit_app', 'kredit_worker')
        AND supplier_organization_id = app.current_organization_id()
        AND (
            payment_id IS NULL OR EXISTS (
                SELECT 1 FROM app.payments p
                WHERE p.id = settlement_events.payment_id
                  AND p.supplier_organization_id = settlement_events.supplier_organization_id
            )
        )
    );

-- Snapshot rows already have a tenant-aware policy from migration 024. Removing
-- the service-wide policy above makes that policy authoritative for runtime
-- connections.

-- Buyer payment claims ----------------------------------------------------------
DROP POLICY IF EXISTS payment_claims_runtime_access ON app.payment_claims;
DROP POLICY IF EXISTS payment_claims_runtime_tenant ON app.payment_claims;
CREATE POLICY payment_claims_runtime_tenant ON app.payment_claims
    USING (
        current_user IN ('kredit_app', 'kredit_worker')
        AND supplier_organization_id = app.current_organization_id()
    )
    WITH CHECK (
        current_user IN ('kredit_app', 'kredit_worker')
        AND supplier_organization_id = app.current_organization_id()
        AND EXISTS (
            SELECT 1 FROM app.obligations o
            WHERE o.id = payment_claims.obligation_id
              AND o.supplier_organization_id = payment_claims.supplier_organization_id
        )
    );

-- Trade-line financial exposure ------------------------------------------------
DROP POLICY IF EXISTS trade_line_runtime_access ON app.trade_lines;
DROP POLICY IF EXISTS drawdown_runtime_access ON app.drawdowns;
DROP POLICY IF EXISTS drawdown_reservation_runtime_access ON app.drawdown_reservations;

DROP POLICY IF EXISTS trade_line_runtime_tenant ON app.trade_lines;
CREATE POLICY trade_line_runtime_tenant ON app.trade_lines
    USING (
        current_user IN ('kredit_app', 'kredit_worker')
        AND supplier_organization_id = app.current_organization_id()
    )
    WITH CHECK (
        current_user IN ('kredit_app', 'kredit_worker')
        AND supplier_organization_id = app.current_organization_id()
    );

DROP POLICY IF EXISTS drawdown_runtime_tenant ON app.drawdowns;
CREATE POLICY drawdown_runtime_tenant ON app.drawdowns
    USING (
        current_user IN ('kredit_app', 'kredit_worker')
        AND EXISTS (
            SELECT 1 FROM app.trade_lines tl
            WHERE tl.id = drawdowns.trade_line_id
              AND tl.supplier_organization_id = app.current_organization_id()
        )
    )
    WITH CHECK (
        current_user IN ('kredit_app', 'kredit_worker')
        AND EXISTS (
            SELECT 1 FROM app.trade_lines tl
            WHERE tl.id = drawdowns.trade_line_id
              AND tl.supplier_organization_id = app.current_organization_id()
        )
    );

DROP POLICY IF EXISTS drawdown_reservation_runtime_tenant ON app.drawdown_reservations;
CREATE POLICY drawdown_reservation_runtime_tenant ON app.drawdown_reservations
    USING (
        current_user IN ('kredit_app', 'kredit_worker')
        AND EXISTS (
            SELECT 1 FROM app.trade_lines tl
            WHERE tl.id = drawdown_reservations.trade_line_id
              AND tl.supplier_organization_id = app.current_organization_id()
        )
    )
    WITH CHECK (
        current_user IN ('kredit_app', 'kredit_worker')
        AND EXISTS (
            SELECT 1 FROM app.trade_lines tl
            JOIN app.drawdowns d ON d.id = drawdown_reservations.drawdown_id
            WHERE tl.id = drawdown_reservations.trade_line_id
              AND d.trade_line_id = tl.id
              AND tl.supplier_organization_id = app.current_organization_id()
        )
    );

-- Collection execution ----------------------------------------------------------
DROP POLICY IF EXISTS collection_reservation_runtime_tenant ON app.collection_reservations;
CREATE POLICY collection_reservation_runtime_tenant ON app.collection_reservations
    USING (
        current_user IN ('kredit_app', 'kredit_worker')
        AND EXISTS (
            SELECT 1 FROM app.obligations o
            WHERE o.id = collection_reservations.obligation_id
              AND o.supplier_organization_id = app.current_organization_id()
        )
    )
    WITH CHECK (
        current_user IN ('kredit_app', 'kredit_worker')
        AND EXISTS (
            SELECT 1 FROM app.obligations o
            WHERE o.id = collection_reservations.obligation_id
              AND o.supplier_organization_id = app.current_organization_id()
        )
    );

DROP POLICY IF EXISTS collection_attempt_runtime_tenant ON app.collection_attempts;
CREATE POLICY collection_attempt_runtime_tenant ON app.collection_attempts
    USING (
        current_user IN ('kredit_app', 'kredit_worker')
        AND EXISTS (
            SELECT 1 FROM app.obligations o
            WHERE o.id = collection_attempts.obligation_id
              AND o.supplier_organization_id = app.current_organization_id()
        )
    )
    WITH CHECK (
        current_user IN ('kredit_app', 'kredit_worker')
        AND EXISTS (
            SELECT 1 FROM app.collection_reservations r
            JOIN app.obligations o ON o.id = r.obligation_id
            WHERE r.id = collection_attempts.reservation_id
              AND r.obligation_id = collection_attempts.obligation_id
              AND o.supplier_organization_id = app.current_organization_id()
        )
    );

DROP POLICY IF EXISTS collection_event_runtime ON app.collection_events;
DROP POLICY IF EXISTS collection_event_runtime_tenant ON app.collection_events;
CREATE POLICY collection_event_runtime_tenant ON app.collection_events
    USING (
        current_user IN ('kredit_app', 'kredit_worker')
        AND EXISTS (
            SELECT 1 FROM app.obligations o
            WHERE o.id = collection_events.obligation_id
              AND o.supplier_organization_id = app.current_organization_id()
        )
    )
    WITH CHECK (
        current_user IN ('kredit_app', 'kredit_worker')
        AND EXISTS (
            SELECT 1 FROM app.collection_attempts a
            JOIN app.obligations o ON o.id = a.obligation_id
            WHERE a.id = collection_events.attempt_id
              AND a.obligation_id = collection_events.obligation_id
              AND o.supplier_organization_id = app.current_organization_id()
        )
    );

-- Buyer-originated evidence is never writable by the autonomous worker. This
-- is enforced both by RLS and table privileges in infra/postgres/roles.sql.
DROP POLICY IF EXISTS agreement_acceptance_worker_write ON app.agreement_acceptances;
DROP POLICY IF EXISTS receipt_confirmation_worker_write ON app.receipt_confirmations;

-- +goose Down
DROP POLICY IF EXISTS collection_event_runtime_tenant ON app.collection_events;
DROP POLICY IF EXISTS collection_attempt_runtime_tenant ON app.collection_attempts;
DROP POLICY IF EXISTS collection_reservation_runtime_tenant ON app.collection_reservations;
DROP POLICY IF EXISTS drawdown_reservation_runtime_tenant ON app.drawdown_reservations;
DROP POLICY IF EXISTS drawdown_runtime_tenant ON app.drawdowns;
DROP POLICY IF EXISTS trade_line_runtime_tenant ON app.trade_lines;
DROP POLICY IF EXISTS payment_claims_runtime_tenant ON app.payment_claims;
DROP POLICY IF EXISTS settlement_runtime_tenant ON app.settlement_events;
DROP POLICY IF EXISTS fee_runtime_tenant ON app.fees;
DROP POLICY IF EXISTS allocation_runtime_tenant ON app.payment_allocations;
DROP POLICY IF EXISTS payment_runtime_tenant ON app.payments;
DROP POLICY IF EXISTS obligation_runtime_tenant ON app.obligations;
DROP POLICY IF EXISTS credit_request_runtime_tenant ON app.credit_requests;

-- Rollback restores the prior service-boundary policies.
CREATE POLICY payment_runtime_access ON app.payments
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));
CREATE POLICY allocation_runtime_access ON app.payment_allocations
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));
CREATE POLICY fee_runtime_access ON app.fees
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));
CREATE POLICY obligation_runtime_access ON app.obligations
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));
CREATE POLICY credit_snapshot_runtime_access ON app.credit_aggregate_snapshots
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));
CREATE POLICY payment_claims_runtime_access ON app.payment_claims
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));
CREATE POLICY trade_line_runtime_access ON app.trade_lines
    USING (current_user IN ('kredit_app','kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY drawdown_runtime_access ON app.drawdowns
    USING (current_user IN ('kredit_app','kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY drawdown_reservation_runtime_access ON app.drawdown_reservations
    USING (current_user IN ('kredit_app','kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY collection_event_runtime ON app.collection_events
    USING (current_user IN ('kredit_app','kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
