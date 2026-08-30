-- +goose Up

-- OTP delivery requires recovering the normalized target after a successful
-- code check. Keep the target encrypted; the HMAC remains the lookup key.
ALTER TABLE app.otp_challenges
    ADD COLUMN IF NOT EXISTS target_ciphertext BYTEA;

-- Authentication lookups cross the unauthenticated boundary. These narrowly
-- scoped SECURITY DEFINER functions avoid weakening tenant RLS policies while
-- keeping the application role from receiving unrestricted table access.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.find_or_create_user(
    p_target_type TEXT,
    p_target_value TEXT,
    p_now TIMESTAMPTZ
) RETURNS TABLE (
    id UUID,
    normalized_email TEXT,
    normalized_phone TEXT,
    display_name TEXT,
    status TEXT,
    created_at TIMESTAMPTZ,
    last_authenticated_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = app, pg_catalog
AS $$
BEGIN
    IF p_target_type = 'email' THEN
        RETURN QUERY
        INSERT INTO app.users AS u (normalized_email, status, last_authenticated_at)
        VALUES (p_target_value, 'active', NULL)
        ON CONFLICT (normalized_email) WHERE normalized_email IS NOT NULL
        DO UPDATE SET last_authenticated_at = u.last_authenticated_at
        RETURNING u.id, u.normalized_email, u.normalized_phone,
                  u.display_name, u.status, u.created_at,
                  u.last_authenticated_at;
    ELSIF p_target_type = 'phone' THEN
        RETURN QUERY
        INSERT INTO app.users AS u (normalized_phone, status, last_authenticated_at)
        VALUES (p_target_value, 'active', NULL)
        ON CONFLICT (normalized_phone) WHERE normalized_phone IS NOT NULL
        DO UPDATE SET last_authenticated_at = u.last_authenticated_at
        RETURNING u.id, u.normalized_email, u.normalized_phone,
                  u.display_name, u.status, u.created_at,
                  u.last_authenticated_at;
    ELSE
        RAISE EXCEPTION 'unsupported authentication target type';
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.session_by_token_hash(p_token_hash BYTEA)
RETURNS TABLE (
    session_id UUID,
    user_id UUID,
    authentication_level TEXT,
    device_label TEXT,
    session_created_at TIMESTAMPTZ,
    session_expires_at TIMESTAMPTZ,
    session_revoked_at TIMESTAMPTZ,
    email TEXT,
    phone TEXT,
    display_name TEXT,
    user_status TEXT,
    user_created_at TIMESTAMPTZ,
    last_authenticated_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = app, pg_catalog
AS $$
    SELECT s.id, u.id, s.authentication_level, s.device_label,
           s.created_at, s.expires_at, s.revoked_at,
           u.normalized_email, u.normalized_phone, u.display_name,
           u.status, u.created_at, u.last_authenticated_at
    FROM app.sessions s
    JOIN app.users u ON u.id = s.user_id
    WHERE s.token_hash = p_token_hash
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.find_or_create_user(TEXT, TEXT, TIMESTAMPTZ) FROM PUBLIC;
REVOKE ALL ON FUNCTION app.session_by_token_hash(BYTEA) FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        GRANT EXECUTE ON FUNCTION app.find_or_create_user(TEXT, TEXT, TIMESTAMPTZ) TO kredit_app;
        GRANT EXECUTE ON FUNCTION app.session_by_token_hash(BYTEA) TO kredit_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- A member may inspect their own memberships across organisations. Writes
-- still require the organisation transaction context, preserving tenant
-- isolation for invitations and role changes.
DROP POLICY IF EXISTS memberships_tenant_isolation ON app.memberships;
CREATE POLICY memberships_tenant_isolation ON app.memberships
    USING (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID
        OR user_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID
    )
    WITH CHECK (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID
    );

-- +goose Down
DROP POLICY IF EXISTS memberships_tenant_isolation ON app.memberships;
CREATE POLICY memberships_tenant_isolation ON app.memberships
    USING (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID)
    WITH CHECK (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID);
DROP FUNCTION IF EXISTS app.session_by_token_hash(BYTEA);
DROP FUNCTION IF EXISTS app.find_or_create_user(TEXT, TEXT, TIMESTAMPTZ);
ALTER TABLE app.otp_challenges DROP COLUMN IF EXISTS target_ciphertext;
