-- +goose Up

-- Include the owning buyer user in provider lookups so a mandate recovered
-- after restart can still pass the same buyer-ownership check as a freshly
-- created mandate. The provider reference remains the only lookup key.
DROP FUNCTION IF EXISTS app.payment_mandate_by_provider(TEXT, TEXT);

-- +goose StatementBegin
CREATE FUNCTION app.payment_mandate_by_provider(p_provider TEXT, p_provider_mandate_id TEXT)
RETURNS TABLE (
    id TEXT,
    provider TEXT,
    provider_mandate_id TEXT,
    buyer_subject_id TEXT,
    buyer_user_id TEXT,
    amount_ceiling_kobo BIGINT,
    state TEXT,
    created_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = app, pg_catalog
AS $$
    SELECT m.id::text,
           m.provider,
           m.provider_mandate_id,
           m.buyer_subject_id::text,
           COALESCE(b.owner_user_id, p.user_id)::text,
           m.amount_ceiling_kobo,
           m.state,
           m.created_at
    FROM app.payment_mandates m
    LEFT JOIN app.businesses b
      ON m.buyer_subject_type = 'business' AND b.id = m.buyer_subject_id
    LEFT JOIN app.persons p
      ON m.buyer_subject_type = 'person' AND p.id = m.buyer_subject_id
    WHERE m.provider = p_provider
      AND m.provider_mandate_id = p_provider_mandate_id;
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
-- +goose StatementBegin
CREATE FUNCTION app.payment_mandate_by_provider(p_provider TEXT, p_provider_mandate_id TEXT)
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
    WHERE provider = p_provider AND provider_mandate_id = p_provider_mandate_id;
$$;
-- +goose StatementEnd
