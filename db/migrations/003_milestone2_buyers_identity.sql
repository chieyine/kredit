-- +goose Up
CREATE TABLE IF NOT EXISTS app.persons (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID UNIQUE REFERENCES app.users(id),
    full_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'invited' CHECK (status IN ('invited', 'pending_verification', 'verified', 'suspended', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS app.businesses (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID REFERENCES app.organizations(id),
    owner_user_id UUID REFERENCES app.users(id),
    legal_name TEXT NOT NULL,
    trading_name TEXT,
    business_type TEXT NOT NULL,
    registration_info JSONB,
    business_address TEXT NOT NULL,
    industry TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending_verification' CHECK (status IN ('pending_verification', 'verified', 'suspended', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS businesses_owner_user_idx ON app.businesses (owner_user_id);
CREATE INDEX IF NOT EXISTS businesses_organization_idx ON app.businesses (organization_id);

CREATE TABLE IF NOT EXISTS app.business_representatives (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    business_id UUID NOT NULL REFERENCES app.businesses(id),
    person_id UUID NOT NULL REFERENCES app.persons(id),
    role_title TEXT NOT NULL,
    authority_type TEXT NOT NULL,
    authority_verification_status TEXT NOT NULL DEFAULT 'pending' CHECK (authority_verification_status IN ('pending', 'verified', 'rejected', 'expired')),
    starts_at DATE,
    ends_at DATE,
    evidence_reference TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS active_business_representative_unique ON app.business_representatives (business_id, person_id) WHERE authority_verification_status IN ('pending', 'verified');

CREATE TABLE IF NOT EXISTS app.verification_cases (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    subject_type TEXT NOT NULL CHECK (subject_type IN ('person', 'business', 'authority')),
    subject_id UUID NOT NULL,
    provider TEXT NOT NULL,
    provider_reference TEXT NOT NULL,
    verification_level INTEGER NOT NULL CHECK (verification_level >= 0),
    state TEXT NOT NULL CHECK (state IN ('pending', 'in_progress', 'verified', 'failed', 'expired', 'review')),
    reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
    safe_result JSONB NOT NULL DEFAULT '{}'::jsonb,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS verification_provider_reference_unique ON app.verification_cases (provider, provider_reference);
CREATE INDEX IF NOT EXISTS verification_subject_idx ON app.verification_cases (subject_type, subject_id, state);

CREATE TABLE IF NOT EXISTS app.identity_consents (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES app.users(id),
    consent_type TEXT NOT NULL,
    version TEXT NOT NULL,
    accepted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    evidence_hash TEXT
);

CREATE INDEX IF NOT EXISTS identity_consents_user_idx ON app.identity_consents (user_id, consent_type, accepted_at DESC);

CREATE TABLE IF NOT EXISTS app.bank_account_references (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    owner_type TEXT NOT NULL CHECK (owner_type IN ('person', 'business')),
    owner_id UUID NOT NULL,
    provider TEXT NOT NULL,
    provider_reference TEXT NOT NULL,
    bank_code TEXT,
    masked_account TEXT,
    account_name_result TEXT,
    account_type TEXT,
    ownership_state TEXT NOT NULL DEFAULT 'pending' CHECK (ownership_state IN ('pending', 'verified', 'failed', 'expired')),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS bank_provider_reference_unique ON app.bank_account_references (provider, provider_reference);
CREATE INDEX IF NOT EXISTS bank_owner_idx ON app.bank_account_references (owner_type, owner_id, active);

CREATE TABLE IF NOT EXISTS app.buyer_invitations (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES app.organizations(id),
    target_type TEXT NOT NULL CHECK (target_type IN ('phone', 'email')),
    target_hash BYTEA NOT NULL,
    token_hash BYTEA NOT NULL UNIQUE,
    proposed_legal_name TEXT NOT NULL,
    proposed_trading_name TEXT,
    proposed_business_type TEXT NOT NULL,
    proposed_address TEXT NOT NULL,
    proposed_industry TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'expired', 'revoked')),
    lookup_attempts INTEGER NOT NULL DEFAULT 0 CHECK (lookup_attempts >= 0),
    created_by UUID NOT NULL REFERENCES app.users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    accepted_by_user_id UUID REFERENCES app.users(id)
);

CREATE INDEX IF NOT EXISTS buyer_invitation_org_idx ON app.buyer_invitations (organization_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS buyer_invitation_target_idx ON app.buyer_invitations (target_hash, status, expires_at);

ALTER TABLE app.persons ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.businesses ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.business_representatives ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.verification_cases ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.identity_consents ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.bank_account_references ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.buyer_invitations ENABLE ROW LEVEL SECURITY;

CREATE POLICY persons_user_isolation ON app.persons
    USING (user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID)
    WITH CHECK (user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID);

CREATE POLICY businesses_user_isolation ON app.businesses
    USING (
        owner_user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID
        OR EXISTS (
            SELECT 1 FROM app.business_representatives representative
            JOIN app.persons person ON person.id = representative.person_id
            WHERE representative.business_id = app.businesses.id
              AND person.user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID
        )
    );

CREATE POLICY representatives_user_isolation ON app.business_representatives
    USING (person_id IN (
        SELECT id FROM app.persons
        WHERE user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID
    ));

CREATE POLICY verification_user_isolation ON app.verification_cases
    USING (
        (subject_type = 'person' AND subject_id IN (SELECT id FROM app.persons WHERE user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID))
        OR (subject_type = 'business' AND subject_id IN (SELECT id FROM app.businesses WHERE owner_user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID))
        OR (subject_type = 'authority' AND subject_id IN (SELECT representative.id FROM app.business_representatives representative JOIN app.persons person ON person.id = representative.person_id WHERE person.user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID))
    );

CREATE POLICY consent_user_isolation ON app.identity_consents
    USING (user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID)
    WITH CHECK (user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID);

CREATE POLICY bank_reference_user_isolation ON app.bank_account_references
    USING (
        (owner_type = 'person' AND owner_id IN (SELECT id FROM app.persons WHERE user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID))
        OR (owner_type = 'business' AND owner_id IN (SELECT id FROM app.businesses WHERE owner_user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID))
    );

CREATE POLICY buyer_invitation_tenant_isolation ON app.buyer_invitations
    USING (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID)
    WITH CHECK (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID);

-- +goose Down
DROP TABLE IF EXISTS app.buyer_invitations;
DROP TABLE IF EXISTS app.bank_account_references;
DROP TABLE IF EXISTS app.identity_consents;
DROP TABLE IF EXISTS app.verification_cases;
DROP TABLE IF EXISTS app.business_representatives;
DROP TABLE IF EXISTS app.businesses;
DROP TABLE IF EXISTS app.persons;

