-- +goose Up
-- Two authentication controls README section 21 requires but the schema could
-- not express.
--
-- 1. Sessions had only an absolute lifetime. Section 21.2 requires an idle
--    deadline as well, so an abandoned session on a shared device stops being
--    usable long before its thirty days elapse.
-- 2. TOTP verification had no per-account attempt limit, so a six-digit code
--    with a three-step acceptance window was brute-forceable behind a single
--    authenticated session. The lock is time-bounded and clears itself, and
--    account recovery remains the path for a user who cannot wait it out.

ALTER TABLE app.sessions
  ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE app.mfa_methods
  ADD COLUMN IF NOT EXISTS failed_attempts INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ;

ALTER TABLE app.mfa_methods
  DROP CONSTRAINT IF EXISTS mfa_methods_failed_attempts_non_negative;
ALTER TABLE app.mfa_methods
  ADD CONSTRAINT mfa_methods_failed_attempts_non_negative CHECK (failed_attempts >= 0);

-- The lookup returns a fixed column list, so adding last_seen_at means
-- replacing the function rather than redefining it in place.
DROP FUNCTION IF EXISTS app.session_by_token_hash(BYTEA);

-- +goose StatementBegin
CREATE FUNCTION app.session_by_token_hash(p_token_hash BYTEA)
RETURNS TABLE (
    session_id UUID,
    user_id UUID,
    authentication_level TEXT,
    device_label TEXT,
    session_created_at TIMESTAMPTZ,
    session_expires_at TIMESTAMPTZ,
    session_last_seen_at TIMESTAMPTZ,
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
           s.created_at, s.expires_at, s.last_seen_at, s.revoked_at,
           u.normalized_email, u.normalized_phone, u.display_name,
           u.status, u.created_at, u.last_authenticated_at
    FROM app.sessions s
    JOIN app.users u ON u.id = s.user_id
    WHERE s.token_hash = p_token_hash
$$;
-- +goose StatementEnd

-- Refreshing last-seen must not require the caller to hold write access to
-- every session row, and the caller has already proven possession of the token.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.touch_session(p_session_id UUID, p_seen_at TIMESTAMPTZ)
RETURNS VOID
LANGUAGE sql
SECURITY DEFINER
SET search_path = app, pg_catalog
AS $$
    UPDATE app.sessions
    SET last_seen_at = p_seen_at
    WHERE id = p_session_id AND revoked_at IS NULL AND last_seen_at < p_seen_at
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.session_by_token_hash(BYTEA) FROM PUBLIC;
REVOKE ALL ON FUNCTION app.touch_session(UUID, TIMESTAMPTZ) FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        GRANT EXECUTE ON FUNCTION app.session_by_token_hash(BYTEA) TO kredit_app;
        GRANT EXECUTE ON FUNCTION app.touch_session(UUID, TIMESTAMPTZ) TO kredit_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS app.touch_session(UUID, TIMESTAMPTZ);
DROP FUNCTION IF EXISTS app.session_by_token_hash(BYTEA);

-- +goose StatementBegin
CREATE FUNCTION app.session_by_token_hash(p_token_hash BYTEA)
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

ALTER TABLE app.mfa_methods DROP CONSTRAINT IF EXISTS mfa_methods_failed_attempts_non_negative;
ALTER TABLE app.mfa_methods DROP COLUMN IF EXISTS locked_until;
ALTER TABLE app.mfa_methods DROP COLUMN IF EXISTS failed_attempts;
ALTER TABLE app.sessions DROP COLUMN IF EXISTS last_seen_at;
