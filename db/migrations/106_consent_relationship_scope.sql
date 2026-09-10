-- +goose Up
-- Keep the buyer-only append boundary while supporting limits before activation
-- and withdrawal of earlier consent after a trading relationship ends.
-- A narrowly scoped definer lookup avoids recursive RLS on the evidence table.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.has_own_relationship_consent(p_supplier uuid,p_type text)
RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT EXISTS(SELECT 1 FROM app.relationship_consents c
  WHERE c.buyer_user_id=app.current_user_id()
   AND c.supplier_organization_id=p_supplier AND c.consent_type=p_type)
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.has_own_relationship_consent(uuid,text) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
  GRANT EXECUTE ON FUNCTION app.has_own_relationship_consent(uuid,text) TO kredit_app;
 END IF;
END $$;
-- +goose StatementEnd
DROP POLICY relationship_consent_buyer_record ON app.relationship_consents;
CREATE POLICY relationship_consent_buyer_record ON app.relationship_consents FOR INSERT
WITH CHECK (buyer_user_id=app.current_user_id() AND (
 EXISTS (
  SELECT 1 FROM app.trade_relationships r JOIN app.businesses b ON b.id=r.buyer_business_id
  WHERE r.supplier_organization_id=relationship_consents.supplier_organization_id
   AND b.owner_user_id=app.current_user_id() AND r.status IN ('invited','active')
 ) OR EXISTS (
  SELECT 1 FROM app.trade_lines l WHERE l.buyer_user_id=app.current_user_id()
   AND l.supplier_organization_id=relationship_consents.supplier_organization_id
 ) OR EXISTS (
  SELECT 1 FROM app.credit_requests c WHERE c.buyer_user_id=app.current_user_id()
   AND c.supplier_organization_id=relationship_consents.supplier_organization_id
 ) OR (NOT granted AND app.has_own_relationship_consent(supplier_organization_id,consent_type))
));

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 RAISE EXCEPTION 'Retain buyer withdrawal rights; use a forward migration';
END $$;
-- +goose StatementEnd
