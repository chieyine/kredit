-- +goose Up

ALTER TABLE app.trade_lines
    ADD CONSTRAINT trade_lines_mandate_fk
        FOREIGN KEY (mandate_id) REFERENCES app.payment_mandates(id),
    ADD CONSTRAINT trade_lines_active_mandate_required
        CHECK (state <> 'ACTIVE' OR (mandate_id IS NOT NULL AND mandate_active));

-- Resolve only the safe mandate facts required to authorize a trade line.
-- +goose StatementBegin
CREATE FUNCTION app.trade_line_mandate(p_mandate_id UUID, p_buyer_user_id UUID, p_buyer_business_id UUID)
RETURNS TABLE (
    id TEXT,
    provider TEXT,
    provider_mandate_id TEXT,
    buyer_business_id TEXT,
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
           b.owner_user_id::text,
           m.amount_ceiling_kobo,
           m.state,
           m.created_at
    FROM app.payment_mandates m
    JOIN app.businesses b ON b.id = m.buyer_subject_id
    WHERE m.id = p_mandate_id
      AND m.buyer_subject_type = 'business'
      AND m.buyer_subject_id = p_buyer_business_id
      AND b.owner_user_id = p_buyer_user_id
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.trade_line_mandate(UUID, UUID, UUID) FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        GRANT EXECUTE ON FUNCTION app.trade_line_mandate(UUID, UUID, UUID) TO kredit_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS app.trade_line_mandate(UUID, UUID, UUID);
ALTER TABLE app.trade_lines
    DROP CONSTRAINT IF EXISTS trade_lines_active_mandate_required,
    DROP CONSTRAINT IF EXISTS trade_lines_mandate_fk;
