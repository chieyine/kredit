-- +goose Up
-- Organizations is the canonical trading identity.
-- buyer_organization_id links active credit requests, trade lines, and obligations
-- directly to the canonical organization while preserving buyer_business_id for contractual history.

ALTER TABLE app.credit_requests ADD COLUMN IF NOT EXISTS buyer_organization_id uuid REFERENCES app.organizations(id);
ALTER TABLE app.trade_lines ADD COLUMN IF NOT EXISTS buyer_organization_id uuid REFERENCES app.organizations(id);
ALTER TABLE app.obligations ADD COLUMN IF NOT EXISTS buyer_organization_id uuid REFERENCES app.organizations(id);

UPDATE app.credit_requests cr SET buyer_organization_id = b.organization_id FROM app.businesses b WHERE cr.buyer_business_id = b.id AND cr.buyer_organization_id IS NULL;
UPDATE app.trade_lines tl SET buyer_organization_id = b.organization_id FROM app.businesses b WHERE tl.buyer_business_id = b.id AND tl.buyer_organization_id IS NULL;
UPDATE app.obligations ob SET buyer_organization_id = b.organization_id FROM app.businesses b WHERE ob.buyer_business_id = b.id AND ob.buyer_organization_id IS NULL;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.sync_canonical_buyer_organization() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE 
  org_id uuid;
  biz_org_id uuid;
BEGIN
  IF NEW.buyer_organization_id IS NULL AND NEW.buyer_business_id IS NOT NULL THEN
    SELECT organization_id INTO org_id FROM app.businesses WHERE id = NEW.buyer_business_id;
    NEW.buyer_organization_id := org_id;
  ELSIF NEW.buyer_organization_id IS NOT NULL AND NEW.buyer_business_id IS NULL THEN
    SELECT id INTO NEW.buyer_business_id FROM app.businesses WHERE organization_id = NEW.buyer_organization_id ORDER BY created_at LIMIT 1;
  ELSIF NEW.buyer_organization_id IS NOT NULL AND NEW.buyer_business_id IS NOT NULL THEN
    SELECT organization_id INTO biz_org_id FROM app.businesses WHERE id = NEW.buyer_business_id;
    IF biz_org_id IS DISTINCT FROM NEW.buyer_organization_id THEN
      RAISE EXCEPTION 'Conflicting organization identity: buyer_business_id % belongs to organization %, not %', NEW.buyer_business_id, biz_org_id, NEW.buyer_organization_id USING ERRCODE = '23503';
    END IF;
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS sync_credit_requests_canonical_org ON app.credit_requests;
CREATE TRIGGER sync_credit_requests_canonical_org BEFORE INSERT OR UPDATE ON app.credit_requests FOR EACH ROW EXECUTE FUNCTION app.sync_canonical_buyer_organization();

DROP TRIGGER IF EXISTS sync_trade_lines_canonical_org ON app.trade_lines;
CREATE TRIGGER sync_trade_lines_canonical_org BEFORE INSERT OR UPDATE ON app.trade_lines FOR EACH ROW EXECUTE FUNCTION app.sync_canonical_buyer_organization();

DROP TRIGGER IF EXISTS sync_obligations_canonical_org ON app.obligations;
CREATE TRIGGER sync_obligations_canonical_org BEFORE INSERT OR UPDATE ON app.obligations FOR EACH ROW EXECUTE FUNCTION app.sync_canonical_buyer_organization();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.organization_purchasing_profile(p_org uuid) RETURNS uuid
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
  SELECT id FROM app.businesses WHERE organization_id = p_org ORDER BY created_at LIMIT 1;
$$;

CREATE OR REPLACE FUNCTION app.purchasing_profile_organization(p_profile uuid) RETURNS uuid
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
  SELECT organization_id FROM app.businesses WHERE id = p_profile;
$$;

CREATE OR REPLACE FUNCTION app.can_purchase_organization(p_org uuid, action_name text DEFAULT 'read', amount_kobo bigint DEFAULT 0) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
  SELECT EXISTS(SELECT 1 FROM app.businesses b WHERE b.organization_id = p_org AND app.can_purchase(b.id, action_name, amount_kobo));
$$;
-- +goose StatementEnd

GRANT EXECUTE ON FUNCTION app.organization_purchasing_profile(uuid), app.purchasing_profile_organization(uuid), app.can_purchase_organization(uuid, text, bigint) TO kredit_app, kredit_worker;

CREATE INDEX IF NOT EXISTS credit_requests_buyer_org_idx ON app.credit_requests(buyer_organization_id, updated_at DESC, id);
CREATE INDEX IF NOT EXISTS trade_lines_buyer_org_idx ON app.trade_lines(buyer_organization_id, updated_at DESC, id);
CREATE INDEX IF NOT EXISTS obligations_buyer_org_idx ON app.obligations(buyer_organization_id, activated_at, id);

-- +goose Down
DROP TRIGGER IF EXISTS sync_obligations_canonical_org ON app.obligations;
DROP TRIGGER IF EXISTS sync_trade_lines_canonical_org ON app.trade_lines;
DROP TRIGGER IF EXISTS sync_credit_requests_canonical_org ON app.credit_requests;
DROP FUNCTION IF EXISTS app.sync_canonical_buyer_organization();
DROP FUNCTION IF EXISTS app.can_purchase_organization(uuid, text, bigint);
DROP FUNCTION IF EXISTS app.purchasing_profile_organization(uuid);
DROP FUNCTION IF EXISTS app.organization_purchasing_profile(uuid);
DROP INDEX IF EXISTS app.obligations_buyer_org_idx;
DROP INDEX IF EXISTS app.trade_lines_buyer_org_idx;
DROP INDEX IF EXISTS app.credit_requests_buyer_org_idx;
ALTER TABLE app.obligations DROP COLUMN IF EXISTS buyer_organization_id;
ALTER TABLE app.trade_lines DROP COLUMN IF EXISTS buyer_organization_id;
ALTER TABLE app.credit_requests DROP COLUMN IF EXISTS buyer_organization_id;
