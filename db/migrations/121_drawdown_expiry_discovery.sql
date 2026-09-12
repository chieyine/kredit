-- +goose Up
-- Identifiers only; expiry writes use the tenant-scoped worker transaction.
-- +goose StatementBegin
CREATE FUNCTION app.drawdown_expiry_tenants(p_cursor text,p_limit integer)
RETURNS TABLE(organization_id text) LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT DISTINCT l.supplier_organization_id::text FROM app.trade_lines l
 JOIN app.drawdown_reservations r ON r.trade_line_id=l.id JOIN app.drawdowns d ON d.id=r.drawdown_id
 WHERE r.state IN ('PENDING','CONFIRMED') AND r.expires_at<=now() AND d.state IN ('PENDING_BUYER_CONFIRMATION','BUYER_CONFIRMED')
 AND l.supplier_organization_id::text>COALESCE(p_cursor,'')
 ORDER BY 1 LIMIT greatest(1,least(p_limit,500))
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.drawdown_expiry_tenants(text,integer) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN GRANT EXECUTE ON FUNCTION app.drawdown_expiry_tenants(text,integer) TO kredit_worker; END IF; END $$;
-- +goose StatementEnd
-- +goose Down
DROP FUNCTION app.drawdown_expiry_tenants(text,integer);
