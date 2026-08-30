-- +goose Up
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
#variable_conflict use_column
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
REVOKE ALL ON FUNCTION app.find_or_create_user(TEXT, TEXT, TIMESTAMPTZ) FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        GRANT EXECUTE ON FUNCTION app.find_or_create_user(TEXT, TEXT, TIMESTAMPTZ) TO kredit_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- The corrected function is intentionally retained on rollback because the
-- prior definition prevents database-backed authentication.
SELECT 1;
