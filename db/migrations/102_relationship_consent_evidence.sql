-- +goose Up
-- Suppliers can inspect their customers' choices, but cannot manufacture them.
DROP POLICY relationship_consent_supplier_access ON app.relationship_consents;
CREATE POLICY relationship_consent_supplier_access ON app.relationship_consents FOR SELECT
USING (supplier_organization_id IN (
 SELECT organization_id FROM app.memberships
 WHERE user_id=app.current_user_id() AND status='active'
));
DROP POLICY relationship_consent_buyer_access ON app.relationship_consents;
CREATE POLICY relationship_consent_buyer_read ON app.relationship_consents FOR SELECT
USING (buyer_user_id=app.current_user_id());
CREATE POLICY relationship_consent_buyer_record ON app.relationship_consents FOR INSERT
WITH CHECK (buyer_user_id=app.current_user_id() AND EXISTS (
 SELECT 1 FROM app.trade_relationships r JOIN app.businesses b ON b.id=r.buyer_business_id
 WHERE r.supplier_organization_id=relationship_consents.supplier_organization_id
 AND b.owner_user_id=app.current_user_id() AND r.status IN ('invited','active')
));

-- Changes of choice are new consent records. The old evidence stays intact.
-- +goose StatementBegin
CREATE FUNCTION app.reject_relationship_consent_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$ BEGIN
 RAISE EXCEPTION 'relationship consent evidence is append-only';
END $$;
-- +goose StatementEnd
CREATE TRIGGER relationship_consent_immutable BEFORE UPDATE OR DELETE ON app.relationship_consents
FOR EACH ROW EXECUTE FUNCTION app.reject_relationship_consent_mutation();
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
  REVOKE UPDATE,DELETE,TRUNCATE ON app.relationship_consents FROM kredit_app;
 END IF;
 IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
  REVOKE INSERT,UPDATE,DELETE,TRUNCATE ON app.relationship_consents FROM kredit_worker;
 END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 RAISE EXCEPTION 'Consent evidence must remain append-only; use a forward migration';
END $$;
-- +goose StatementEnd
