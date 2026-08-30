-- +goose Up

-- Provider callbacks identify a mandate by provider and provider reference,
-- not by the current browser user. Keep the lookup narrowly scoped and
-- SECURITY DEFINER so RLS cannot turn a valid callback into a lost state
-- transition. No account token or unrelated buyer data is returned.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.payment_mandate_by_provider(p_provider TEXT, p_provider_mandate_id TEXT)
RETURNS TABLE (
    id TEXT,
    provider TEXT,
    provider_mandate_id TEXT,
    buyer_subject_id TEXT,
    amount_ceiling_kobo BIGINT,
    state TEXT,
    created_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = app, pg_catalog
AS $$
    SELECT id::text, provider, provider_mandate_id, buyer_subject_id::text,
           amount_ceiling_kobo, state, created_at
    FROM app.payment_mandates
    WHERE provider = p_provider AND provider_mandate_id = p_provider_mandate_id
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.payment_mandate_by_provider(TEXT, TEXT) FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        GRANT EXECUTE ON FUNCTION app.payment_mandate_by_provider(TEXT, TEXT) TO kredit_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS app.payment_mandate_by_provider(TEXT, TEXT);
