-- +goose Up

CREATE TABLE IF NOT EXISTS app.trade_relationships (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    supplier_organization_id UUID NOT NULL REFERENCES app.organizations(id),
    buyer_business_id UUID NOT NULL REFERENCES app.businesses(id),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('invited', 'active', 'paused', 'closed')),
    first_transaction_at TIMESTAMPTZ,
    last_transaction_at TIMESTAMPTZ,
    supplier_customer_code TEXT,
    supplier_private_notes TEXT,
    default_policy_reference TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (supplier_organization_id, buyer_business_id)
);
CREATE INDEX IF NOT EXISTS trade_relationships_supplier_idx ON app.trade_relationships (supplier_organization_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS trade_relationships_buyer_idx ON app.trade_relationships (buyer_business_id, status, updated_at DESC);

CREATE TABLE IF NOT EXISTS app.payment_mandates (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    buyer_subject_type TEXT NOT NULL CHECK (buyer_subject_type IN ('person', 'business')),
    buyer_subject_id UUID NOT NULL,
    provider TEXT NOT NULL,
    provider_mandate_id TEXT NOT NULL,
    mandate_type TEXT NOT NULL,
    amount_ceiling_kobo BIGINT NOT NULL CHECK (amount_ceiling_kobo > 0),
    frequency_ceiling INTEGER,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    state TEXT NOT NULL CHECK (state IN ('pending', 'active', 'cancelled', 'expired', 'failed', 'suspended')),
    -- Provider account references are recoverable restricted data.  Keep only
    -- encrypted ciphertext in the database; plaintext tokens must never be
    -- persisted or returned from a query.
    primary_account_token_ciphertext BYTEA,
    capability_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    accepted_disclosure_version TEXT NOT NULL,
    provider_updated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_mandate_id)
);
CREATE INDEX IF NOT EXISTS payment_mandates_buyer_idx ON app.payment_mandates (buyer_subject_type, buyer_subject_id, state);

CREATE TABLE IF NOT EXISTS app.mandate_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    mandate_id UUID NOT NULL REFERENCES app.payment_mandates(id),
    provider_event_id TEXT NOT NULL,
    old_state TEXT,
    new_state TEXT NOT NULL,
    reason_code TEXT,
    event_at TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (mandate_id, provider_event_id)
);
CREATE INDEX IF NOT EXISTS mandate_events_mandate_idx ON app.mandate_events (mandate_id, event_at DESC);

ALTER TABLE app.trade_relationships ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS trade_relationships_tenant_isolation ON app.trade_relationships;
CREATE POLICY trade_relationships_tenant_isolation ON app.trade_relationships
    USING (supplier_organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID)
    WITH CHECK (supplier_organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID);

ALTER TABLE app.payment_mandates ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS payment_mandates_buyer_isolation ON app.payment_mandates;
CREATE POLICY payment_mandates_buyer_isolation ON app.payment_mandates
    USING (
        (buyer_subject_type = 'person' AND buyer_subject_id IN (SELECT id FROM app.persons WHERE user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID))
        OR (buyer_subject_type = 'business' AND buyer_subject_id IN (SELECT id FROM app.businesses WHERE owner_user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID))
    )
    WITH CHECK (
        (buyer_subject_type = 'person' AND buyer_subject_id IN (SELECT id FROM app.persons WHERE user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID))
        OR (buyer_subject_type = 'business' AND buyer_subject_id IN (SELECT id FROM app.businesses WHERE owner_user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID))
    );

ALTER TABLE app.mandate_events ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS mandate_events_buyer_isolation ON app.mandate_events;
CREATE POLICY mandate_events_buyer_isolation ON app.mandate_events
    USING (mandate_id IN (SELECT id FROM app.payment_mandates));

-- +goose Down
DROP TABLE IF EXISTS app.mandate_events;
DROP TABLE IF EXISTS app.payment_mandates;
DROP TABLE IF EXISTS app.trade_relationships;
