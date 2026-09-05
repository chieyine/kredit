-- +goose Up

-- Provider callbacks arrive with a provider reference rather than a tenant.
-- Resolve only the attempt/obligation/organization identifiers under the
-- function owner, then return to normal worker RLS for every financial read or
-- mutation.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.collection_attempt_identity_by_external(p_external_reference text)
RETURNS TABLE(attempt_id uuid, obligation_id uuid, organization_id uuid)
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = pg_catalog, app
AS $$
    SELECT idx.attempt_id, idx.obligation_id, o.supplier_organization_id
    FROM app.collection_attempt_index idx
    JOIN app.obligations o ON o.id = idx.obligation_id
    WHERE idx.external_reference = p_external_reference;
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.collection_attempt_identity_by_external(text) FROM PUBLIC;

-- +goose Down
DROP FUNCTION IF EXISTS app.collection_attempt_identity_by_external(text);
