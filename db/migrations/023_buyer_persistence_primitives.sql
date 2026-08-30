-- +goose Up

-- Buyer invitation delivery needs a recoverable target after a token is
-- presented, but the plaintext target must never be stored. The HMAC remains
-- the lookup key and this ciphertext is only decrypted inside the adapter.
ALTER TABLE app.buyer_invitations
    ADD COLUMN IF NOT EXISTS target_ciphertext BYTEA;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.buyer_invitation_by_token_hash(p_token_hash BYTEA)
RETURNS TABLE (
    id UUID,
    organization_id UUID,
    target_type TEXT,
    target_hash BYTEA,
    target_ciphertext BYTEA,
    proposed_legal_name TEXT,
    proposed_trading_name TEXT,
    proposed_business_type TEXT,
    proposed_address TEXT,
    proposed_industry TEXT,
    status TEXT,
    lookup_attempts INTEGER,
    created_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ,
    accepted_by_user_id UUID
)
LANGUAGE sql
SECURITY DEFINER
SET search_path = app, pg_catalog
AS $$
    SELECT i.id, i.organization_id, i.target_type, i.target_hash,
           i.target_ciphertext, i.proposed_legal_name,
           i.proposed_trading_name, i.proposed_business_type,
           i.proposed_address, i.proposed_industry, i.status,
           i.lookup_attempts, i.created_at, i.expires_at, i.accepted_at,
           i.accepted_by_user_id
    FROM app.buyer_invitations i
    WHERE i.token_hash = p_token_hash
      AND i.expires_at > NOW()
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.buyer_invitation_by_token_hash(BYTEA) FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        GRANT EXECUTE ON FUNCTION app.buyer_invitation_by_token_hash(BYTEA) TO kredit_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.business_count()
RETURNS BIGINT
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = app, pg_catalog
AS $$ SELECT count(*) FROM app.businesses $$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.business_count() FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        GRANT EXECUTE ON FUNCTION app.business_count() TO kredit_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS app.buyer_invitation_by_token_hash(BYTEA);
DROP FUNCTION IF EXISTS app.business_count();
ALTER TABLE app.buyer_invitations DROP COLUMN IF EXISTS target_ciphertext;
