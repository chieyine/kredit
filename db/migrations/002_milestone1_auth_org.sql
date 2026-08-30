-- +goose Up
CREATE SCHEMA IF NOT EXISTS app;

-- RLS policies use these helpers so application code can set request context
-- with SET LOCAL while keeping policy expressions stable and testable.
CREATE OR REPLACE FUNCTION app.current_user_id() RETURNS uuid
LANGUAGE sql STABLE
AS $$ SELECT NULLIF(current_setting('app.current_user_id', true), '')::uuid $$;

CREATE OR REPLACE FUNCTION app.current_organization_id() RETURNS uuid
LANGUAGE sql STABLE
AS $$ SELECT NULLIF(current_setting('app.current_organization_id', true), '')::uuid $$;

CREATE TABLE IF NOT EXISTS app.users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    normalized_email TEXT,
    normalized_phone TEXT,
    display_name TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'locked', 'suspended', 'closed')),
    last_authenticated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_identifier_required CHECK (normalized_email IS NOT NULL OR normalized_phone IS NOT NULL),
    CONSTRAINT users_email_or_phone_format CHECK (normalized_email IS NULL OR position('@' IN normalized_email) > 1)
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique ON app.users (normalized_email) WHERE normalized_email IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS users_phone_unique ON app.users (normalized_phone) WHERE normalized_phone IS NOT NULL;

CREATE TABLE IF NOT EXISTS app.sessions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES app.users(id),
    token_hash BYTEA NOT NULL UNIQUE,
    device_label TEXT,
    ip_metadata JSONB,
    user_agent TEXT,
    authentication_level TEXT NOT NULL DEFAULT 'AAL1' CHECK (authentication_level IN ('AAL0', 'AAL1', 'AAL2', 'AAL3')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS sessions_user_active_idx ON app.sessions (user_id, expires_at) WHERE revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS app.otp_challenges (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    target_type TEXT NOT NULL CHECK (target_type IN ('phone', 'email')),
    target_hash BYTEA NOT NULL,
    purpose TEXT NOT NULL,
    code_hmac BYTEA NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0 AND attempt_count <= 5),
    risk_metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS otp_target_active_idx ON app.otp_challenges (target_hash, purpose, expires_at) WHERE consumed_at IS NULL;

CREATE TABLE IF NOT EXISTS app.mfa_methods (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES app.users(id),
    method_type TEXT NOT NULL CHECK (method_type IN ('totp', 'passkey')),
    secret_ciphertext BYTEA,
    credential_reference TEXT,
    verified_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT mfa_secret_or_credential CHECK (secret_ciphertext IS NOT NULL OR credential_reference IS NOT NULL)
);

CREATE UNIQUE INDEX IF NOT EXISTS one_active_mfa_method_per_type ON app.mfa_methods (user_id, method_type) WHERE revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS app.organizations (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    legal_name TEXT NOT NULL,
    trading_name TEXT,
    business_type TEXT NOT NULL,
    registration_info JSONB,
    business_address TEXT NOT NULL,
    industry TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'onboarding' CHECK (status IN ('onboarding', 'pending_review', 'verified', 'suspended', 'closed')),
    default_timezone TEXT NOT NULL DEFAULT 'Africa/Lagos',
    default_currency CHAR(3) NOT NULL DEFAULT 'NGN' CHECK (default_currency = 'NGN'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0)
);

CREATE TABLE IF NOT EXISTS app.memberships (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES app.organizations(id),
    user_id UUID NOT NULL REFERENCES app.users(id),
    role TEXT NOT NULL CHECK (role IN ('owner', 'administrator', 'finance', 'sales', 'collections', 'viewer')),
    status TEXT NOT NULL DEFAULT 'invited' CHECK (status IN ('invited', 'active', 'suspended', 'removed')),
    invited_by UUID REFERENCES app.users(id),
    invited_at TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS active_membership_unique ON app.memberships (organization_id, user_id) WHERE status IN ('invited', 'active', 'suspended');
CREATE INDEX IF NOT EXISTS memberships_user_idx ON app.memberships (user_id, status);
CREATE INDEX IF NOT EXISTS memberships_org_idx ON app.memberships (organization_id, status);

CREATE TABLE IF NOT EXISTS app.organization_invitations (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES app.organizations(id),
    target_type TEXT NOT NULL CHECK (target_type IN ('phone', 'email')),
    target_hash BYTEA NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('administrator', 'finance', 'sales', 'collections', 'viewer')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'expired', 'revoked')),
    invited_by UUID NOT NULL REFERENCES app.users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS organization_invites_target_idx ON app.organization_invitations (target_hash, status, expires_at);

CREATE TABLE IF NOT EXISTS app.audit_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor_user_id UUID REFERENCES app.users(id),
    organization_id UUID REFERENCES app.organizations(id),
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    outcome TEXT NOT NULL CHECK (outcome IN ('success', 'failure', 'denied')),
    severity TEXT NOT NULL DEFAULT 'info' CHECK (severity IN ('info', 'notice', 'warning', 'high', 'critical')),
    request_id TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS audit_org_time_idx ON app.audit_events (organization_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS audit_actor_time_idx ON app.audit_events (actor_user_id, occurred_at DESC);

ALTER TABLE app.organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.organization_invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.mfa_methods ENABLE ROW LEVEL SECURITY;

CREATE POLICY organizations_tenant_isolation ON app.organizations
    USING (id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID)
    WITH CHECK (id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID);

CREATE POLICY memberships_tenant_isolation ON app.memberships
    USING (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID)
    WITH CHECK (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID);

CREATE POLICY invitations_tenant_isolation ON app.organization_invitations
    USING (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID)
    WITH CHECK (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID);

CREATE POLICY audit_tenant_isolation ON app.audit_events
    USING (organization_id IS NULL OR organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID)
    WITH CHECK (organization_id IS NULL OR organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID);

CREATE POLICY users_context_isolation ON app.users
    USING (
        id = NULLIF(current_setting('app.current_user_id', true), '')::UUID
        OR EXISTS (
            SELECT 1
            FROM app.memberships membership
            WHERE membership.user_id = app.users.id
              AND membership.organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID
              AND membership.status IN ('invited', 'active', 'suspended')
        )
    );

CREATE POLICY sessions_user_isolation ON app.sessions
    USING (user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID)
    WITH CHECK (user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID);

CREATE POLICY mfa_user_isolation ON app.mfa_methods
    USING (user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID)
    WITH CHECK (user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID);

-- +goose Down
DROP TABLE IF EXISTS app.audit_events;
DROP TABLE IF EXISTS app.organization_invitations;
DROP TABLE IF EXISTS app.memberships;
DROP TABLE IF EXISTS app.organizations;
DROP TABLE IF EXISTS app.mfa_methods;
DROP TABLE IF EXISTS app.otp_challenges;
DROP TABLE IF EXISTS app.sessions;
DROP TABLE IF EXISTS app.users;
DROP SCHEMA IF EXISTS app;
