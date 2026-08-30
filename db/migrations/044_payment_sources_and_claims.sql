-- +goose Up

ALTER TABLE app.payments DROP CONSTRAINT IF EXISTS payments_source_type_check;
UPDATE app.payments SET source_type = CASE source_type
    WHEN 'voluntary' THEN 'supplier_recorded_transfer'
    WHEN 'collected' THEN 'kredit_collection'
    ELSE source_type
END;
ALTER TABLE app.payments ADD CONSTRAINT payments_source_type_check CHECK (
    source_type IN (
        'integrated_voluntary',
        'supplier_recorded_transfer',
        'buyer_payment_claim',
        'cash_recorded',
        'kredit_collection',
        'adjustment'
    )
);

CREATE TABLE app.payment_claims (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    obligation_id UUID NOT NULL REFERENCES app.obligations(id),
    buyer_user_id UUID NOT NULL REFERENCES app.users(id),
    supplier_organization_id UUID NOT NULL REFERENCES app.organizations(id),
    amount_kobo BIGINT NOT NULL CHECK (amount_kobo > 0),
    currency CHAR(3) NOT NULL DEFAULT 'NGN' CHECK (currency = 'NGN'),
    paid_at TIMESTAMPTZ NOT NULL,
    source_account_masked TEXT,
    transfer_reference TEXT NOT NULL,
    evidence_document_id UUID REFERENCES app.documents(id),
    state TEXT NOT NULL DEFAULT 'pending' CHECK (state IN ('pending','confirmed','rejected','expired')),
    hold_expires_at TIMESTAMPTZ NOT NULL,
    reviewed_by UUID REFERENCES app.users(id),
    review_reason TEXT,
    payment_id UUID REFERENCES app.payments(id),
    idempotency_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    CHECK (hold_expires_at > created_at),
    CHECK ((state = 'pending' AND reviewed_at IS NULL) OR (state <> 'pending' AND reviewed_at IS NOT NULL))
);
CREATE INDEX payment_claims_obligation_idx ON app.payment_claims (obligation_id, state, hold_expires_at);
CREATE INDEX payment_claims_supplier_idx ON app.payment_claims (supplier_organization_id, created_at DESC);
CREATE INDEX payment_claims_buyer_idx ON app.payment_claims (buyer_user_id, created_at DESC);

ALTER TABLE app.payment_claims ENABLE ROW LEVEL SECURITY;
CREATE POLICY payment_claims_supplier_access ON app.payment_claims
    USING (supplier_organization_id IN (
        SELECT organization_id FROM app.memberships
        WHERE user_id = app.current_user_id() AND status = 'active'
    ));
CREATE POLICY payment_claims_buyer_access ON app.payment_claims
    USING (buyer_user_id = app.current_user_id());
CREATE POLICY payment_claims_runtime_access ON app.payment_claims
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));

-- +goose Down
DROP TABLE IF EXISTS app.payment_claims;
ALTER TABLE app.payments DROP CONSTRAINT IF EXISTS payments_source_type_check;
UPDATE app.payments SET source_type = CASE
    WHEN source_type = 'kredit_collection' THEN 'collected'
    ELSE 'voluntary'
END;
ALTER TABLE app.payments ADD CONSTRAINT payments_source_type_check
    CHECK (source_type IN ('voluntary','collected'));
