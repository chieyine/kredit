-- +goose Up

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.supplier_customers(p_organization_id UUID)
RETURNS TABLE (
    buyer_user_id UUID,
    buyer_business_id UUID,
    legal_name TEXT,
    trading_name TEXT,
    industry TEXT,
    status TEXT
)
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = app, pg_catalog
AS $$
    SELECT b.owner_user_id, b.id, b.legal_name, b.trading_name, b.industry, b.status
    FROM app.trade_relationships r
    JOIN app.businesses b ON b.id = r.buyer_business_id
    WHERE r.supplier_organization_id = p_organization_id
      AND r.status IN ('active', 'paused')
      AND p_organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID
    ORDER BY b.legal_name
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.supplier_customers(UUID) FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        GRANT EXECUTE ON FUNCTION app.supplier_customers(UUID) TO kredit_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS app.supplier_customers(UUID);
