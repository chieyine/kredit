-- +goose Up
-- Return only the display names of suppliers already connected to this buyer.
-- Historical connections remain visible so optional consent can be withdrawn.
-- +goose StatementBegin
CREATE FUNCTION app.buyer_suppliers()
RETURNS TABLE(supplier_organization_id uuid,legal_name text,trading_name text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, app AS $$
 WITH related AS (
  SELECT r.supplier_organization_id FROM app.trade_relationships r
  JOIN app.businesses b ON b.id=r.buyer_business_id
  WHERE b.owner_user_id=app.current_user_id()
  UNION SELECT c.supplier_organization_id FROM app.credit_requests c WHERE c.buyer_user_id=app.current_user_id()
  UNION SELECT l.supplier_organization_id FROM app.trade_lines l WHERE l.buyer_user_id=app.current_user_id()
  UNION SELECT c.supplier_organization_id FROM app.relationship_consents c WHERE c.buyer_user_id=app.current_user_id()
 )
 SELECT o.id,o.legal_name,COALESCE(o.trading_name,'') FROM related r JOIN app.organizations o ON o.id=r.supplier_organization_id
 ORDER BY o.legal_name,o.id
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.buyer_suppliers() FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
  GRANT EXECUTE ON FUNCTION app.buyer_suppliers() TO kredit_app;
 END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION app.buyer_suppliers();
