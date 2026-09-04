-- +goose Up
ALTER TABLE app.sessions ADD COLUMN mfa_verified_at TIMESTAMPTZ;

CREATE TABLE app.supplier_onboarding_profiles (
    organization_id UUID PRIMARY KEY REFERENCES app.organizations(id) ON DELETE CASCADE,
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    authorized_representative_name TEXT NOT NULL DEFAULT '',
    authorized_representative_title TEXT NOT NULL DEFAULT '',
    owner_email_verified_at TIMESTAMPTZ,
    owner_phone_verified_at TIMESTAMPTZ,
    kyb_state TEXT NOT NULL DEFAULT 'not_started' CHECK (kyb_state IN ('not_started','draft','submitted','provider_review','approved','rejected','expired')),
    kyb_provider_reference TEXT,
    kyb_reason_code TEXT,
    kyb_submitted_at TIMESTAMPTZ,
    kyb_decided_at TIMESTAMPTZ,
    kyb_expires_at TIMESTAMPTZ,
    settlement_state TEXT NOT NULL DEFAULT 'not_started' CHECK (settlement_state IN ('not_started','pending_verification','provider_review','verified','rejected','expired')),
    settlement_provider TEXT,
    settlement_provider_reference TEXT,
    settlement_bank_name TEXT,
    settlement_account_name TEXT,
    settlement_account_last4 TEXT CHECK (settlement_account_last4 IS NULL OR settlement_account_last4 ~ '^[0-9]{4}$'),
    settlement_reason_code TEXT,
    settlement_changed_at TIMESTAMPTZ,
    billing_state TEXT NOT NULL DEFAULT 'not_started' CHECK (billing_state IN ('not_started','pending_verification','configured','rejected','expired')),
    billing_method TEXT CHECK (billing_method IS NULL OR billing_method IN ('split_settlement','authorized_debit','consolidated_invoice')),
    billing_provider_reference TEXT,
    billing_cycle TEXT,
    billing_changed_at TIMESTAMPTZ,
    default_credit_limit_kobo BIGINT CHECK (default_credit_limit_kobo IS NULL OR default_credit_limit_kobo > 0),
    default_payment_days INTEGER CHECK (default_payment_days IS NULL OR default_payment_days BETWEEN 1 AND 365),
    default_grace_hours INTEGER CHECK (default_grace_hours IS NULL OR default_grace_hours BETWEEN 0 AND 720),
    default_credit_policy_updated_at TIMESTAMPTZ,
    terms_version TEXT,
    terms_accepted_at TIMESTAMPTZ,
    terms_accepted_by UUID REFERENCES app.users(id),
    privacy_version TEXT,
    privacy_accepted_at TIMESTAMPTZ,
    privacy_accepted_by UUID REFERENCES app.users(id),
    owner_mfa_verified_at TIMESTAMPTZ,
    finance_mfa_complete BOOLEAN NOT NULL DEFAULT TRUE,
    readiness_state TEXT NOT NULL DEFAULT 'incomplete' CHECK (readiness_state IN ('incomplete','provider_review','pilot_ready','expired','rejected','suspended')),
    readiness_changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT settlement_reference_only CHECK (
        settlement_state = 'not_started' OR
        (settlement_provider IS NOT NULL AND settlement_provider_reference IS NOT NULL AND settlement_account_last4 IS NOT NULL)
    )
);

CREATE TABLE app.supplier_onboarding_revisions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES app.organizations(id) ON DELETE CASCADE,
    profile_version BIGINT NOT NULL CHECK (profile_version > 0),
    change_type TEXT NOT NULL,
    actor_user_id UUID REFERENCES app.users(id),
    actor_reference TEXT NOT NULL,
    snapshot JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, profile_version)
);

CREATE INDEX supplier_onboarding_state_idx ON app.supplier_onboarding_profiles (readiness_state, updated_at);
CREATE INDEX supplier_onboarding_kyb_review_idx ON app.supplier_onboarding_profiles (kyb_state, kyb_submitted_at);

ALTER TABLE app.supplier_onboarding_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.supplier_onboarding_revisions ENABLE ROW LEVEL SECURITY;

CREATE POLICY supplier_onboarding_tenant_isolation ON app.supplier_onboarding_profiles
    USING (organization_id = app.current_organization_id())
    WITH CHECK (organization_id = app.current_organization_id());

CREATE POLICY supplier_onboarding_revision_tenant_isolation ON app.supplier_onboarding_revisions
    USING (organization_id = app.current_organization_id())
    WITH CHECK (organization_id = app.current_organization_id());

CREATE OR REPLACE FUNCTION app.reconcile_supplier_onboarding(p_now TIMESTAMPTZ)
RETURNS TABLE (organization_id UUID)
LANGUAGE sql
SECURITY DEFINER
SET search_path = app, pg_catalog
AS $$
    WITH updated AS (
        UPDATE app.supplier_onboarding_profiles p
        SET kyb_state = CASE WHEN p.kyb_state = 'approved' AND p.kyb_expires_at <= p_now THEN 'expired' ELSE p.kyb_state END,
            settlement_state = CASE WHEN p.settlement_state IN ('pending_verification','provider_review') AND p.settlement_changed_at <= p_now - INTERVAL '72 hours' THEN 'expired' ELSE p.settlement_state END,
            readiness_state = 'expired', readiness_changed_at = p_now,
            version = p.version + 1, updated_at = p_now
        WHERE (p.kyb_state = 'approved' AND p.kyb_expires_at <= p_now)
           OR (p.settlement_state IN ('pending_verification','provider_review') AND p.settlement_changed_at <= p_now - INTERVAL '72 hours')
        RETURNING p.*
    ), revisions AS (
        INSERT INTO app.supplier_onboarding_revisions
            (id, organization_id, profile_version, change_type, actor_reference, snapshot)
        SELECT pg_catalog.gen_random_uuid(), u.organization_id, u.version, 'requirements.reconciled', 'system:onboarding-reconciliation', to_jsonb(u)
        FROM updated u
        RETURNING supplier_onboarding_revisions.organization_id
    ), outboxed AS (
        INSERT INTO app.outbox_events (aggregate_type, aggregate_id, event_type, payload, idempotency_key)
        SELECT 'supplier_onboarding', u.organization_id::text, 'SupplierOnboardingRequirementExpired',
               jsonb_build_object('organization_id', u.organization_id, 'readiness_state', u.readiness_state, 'kyb_state', u.kyb_state, 'settlement_state', u.settlement_state),
               'supplier-onboarding:' || u.organization_id::text || ':expired:' || u.version::text
        FROM updated u
        ON CONFLICT (idempotency_key) DO NOTHING
        RETURNING aggregate_id
    )
    SELECT revisions.organization_id FROM revisions
$$;

REVOKE ALL ON FUNCTION app.reconcile_supplier_onboarding(TIMESTAMPTZ) FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        GRANT EXECUTE ON FUNCTION app.reconcile_supplier_onboarding(TIMESTAMPTZ) TO kredit_app;
        GRANT SELECT, INSERT, UPDATE ON app.supplier_onboarding_profiles, app.supplier_onboarding_revisions TO kredit_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_worker') THEN
        GRANT EXECUTE ON FUNCTION app.reconcile_supplier_onboarding(TIMESTAMPTZ) TO kredit_worker;
    END IF;
END
$$;
-- +goose StatementEnd

-- Existing suppliers start incomplete. Readiness must be earned from durable
-- evidence; migration code never guesses provider decisions or consent.
INSERT INTO app.supplier_onboarding_profiles (organization_id)
SELECT id FROM app.organizations
ON CONFLICT (organization_id) DO NOTHING;

INSERT INTO app.supplier_onboarding_revisions
    (organization_id, profile_version, change_type, actor_reference, snapshot)
SELECT p.organization_id, p.version, 'profile.migrated', 'migration:047', to_jsonb(p)
FROM app.supplier_onboarding_profiles p
WHERE NOT EXISTS (
    SELECT 1 FROM app.supplier_onboarding_revisions r
    WHERE r.organization_id = p.organization_id AND r.profile_version = p.version
);

-- +goose Down
DROP FUNCTION IF EXISTS app.reconcile_supplier_onboarding(TIMESTAMPTZ);
DROP TABLE IF EXISTS app.supplier_onboarding_revisions;
DROP TABLE IF EXISTS app.supplier_onboarding_profiles;
ALTER TABLE app.sessions DROP COLUMN IF EXISTS mfa_verified_at;
