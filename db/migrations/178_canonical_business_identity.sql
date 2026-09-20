-- +goose Up
-- organizations is the canonical trading identity. businesses is its unique
-- purchasing-capability profile, retaining existing contractual references.
-- Identity fields on that profile are projections, never a second editable identity.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.provision_purchasing_workspace(profile_id uuid) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE b app.businesses; inserted_profile app.supplier_onboarding_profiles;
BEGIN
 SELECT * INTO b FROM app.businesses WHERE id=profile_id FOR UPDATE;
 IF NOT FOUND OR b.organization_id IS NOT NULL THEN RETURN; END IF;
 IF b.owner_user_id IS NULL THEN RAISE EXCEPTION 'purchasing profile requires a verified ownership mapping before consolidation' USING ERRCODE='23514'; END IF;
 IF EXISTS(SELECT 1 FROM app.organizations WHERE id=b.id) THEN RAISE EXCEPTION 'business identity collision requires explicit reconciliation' USING ERRCODE='23514'; END IF;
 INSERT INTO app.organizations(id,legal_name,trading_name,business_type,registration_info,business_address,industry,status)
 VALUES(b.id,b.legal_name,b.trading_name,b.business_type,b.registration_info,b.business_address,b.industry,'onboarding');
 INSERT INTO app.memberships(organization_id,user_id,role,status,accepted_at) VALUES(b.id,b.owner_user_id,'owner','active',now());
 UPDATE app.businesses SET organization_id=b.id WHERE id=b.id;
 INSERT INTO app.supplier_onboarding_profiles(organization_id) VALUES(b.id) RETURNING * INTO inserted_profile;
 INSERT INTO app.supplier_onboarding_revisions(id,organization_id,profile_version,change_type,actor_user_id,actor_reference,snapshot)
 VALUES(gen_random_uuid(),b.id,inserted_profile.version,'profile.created',b.owner_user_id,b.owner_user_id::text,to_jsonb(inserted_profile));
END $$;
CREATE OR REPLACE FUNCTION app.finish_purchasing_identity() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
BEGIN PERFORM app.provision_purchasing_workspace(NEW.id); RETURN NULL; END $$;
CREATE OR REPLACE FUNCTION app.project_canonical_business_identity() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE o app.organizations;
BEGIN
 IF TG_OP='UPDATE' AND OLD.organization_id IS NOT NULL AND NEW.organization_id IS DISTINCT FROM OLD.organization_id THEN
  RAISE EXCEPTION 'a purchasing capability cannot move to another trading identity' USING ERRCODE='23514';
 END IF;
 IF NEW.organization_id IS NULL THEN RETURN NEW; END IF;
 SELECT * INTO o FROM app.organizations WHERE id=NEW.organization_id FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'canonical business is unavailable' USING ERRCODE='23503'; END IF;
 NEW.legal_name:=o.legal_name; NEW.trading_name:=o.trading_name;
 NEW.business_type:=o.business_type; NEW.registration_info:=o.registration_info;
 NEW.business_address:=o.business_address; NEW.industry:=o.industry;
 RETURN NEW;
END $$;
CREATE OR REPLACE FUNCTION app.refresh_purchasing_identity() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
BEGIN
 UPDATE app.businesses SET legal_name=NEW.legal_name,trading_name=NEW.trading_name,
 business_type=NEW.business_type,registration_info=NEW.registration_info,business_address=NEW.business_address,industry=NEW.industry
 WHERE organization_id=NEW.id;
 RETURN NEW;
END $$;
-- Do not expose provisioning as a general runtime function. It is reachable
-- only through the constrained profile lifecycle, not arbitrary actor arguments.
REVOKE ALL ON FUNCTION app.provision_purchasing_workspace(uuid),app.finish_purchasing_identity(),app.project_canonical_business_identity(),app.refresh_purchasing_identity() FROM PUBLIC;
DO $$ DECLARE b record; BEGIN
 FOR b IN SELECT id FROM app.businesses WHERE organization_id IS NULL ORDER BY id LOOP
  PERFORM app.provision_purchasing_workspace(b.id);
 END LOOP;
END $$;
-- +goose StatementEnd
DROP TRIGGER IF EXISTS canonical_purchasing_identity ON app.businesses;
CREATE TRIGGER canonical_purchasing_identity BEFORE INSERT OR UPDATE ON app.businesses FOR EACH ROW EXECUTE FUNCTION app.project_canonical_business_identity();
DROP TRIGGER IF EXISTS finish_purchasing_identity ON app.businesses;
CREATE CONSTRAINT TRIGGER finish_purchasing_identity AFTER INSERT OR UPDATE ON app.businesses DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION app.finish_purchasing_identity();
DROP TRIGGER IF EXISTS purchasing_identity_projection ON app.organizations;
CREATE TRIGGER purchasing_identity_projection AFTER UPDATE OF legal_name,trading_name,business_type,registration_info,business_address,industry ON app.organizations FOR EACH ROW EXECUTE FUNCTION app.refresh_purchasing_identity();
-- Existing signed agreements retain their original names and terms. Only the
-- current mutable identity projection is aligned to the canonical business.
UPDATE app.businesses b SET legal_name=o.legal_name,trading_name=o.trading_name,business_type=o.business_type,registration_info=o.registration_info,business_address=o.business_address,industry=o.industry FROM app.organizations o WHERE b.organization_id=o.id;

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Canonical identities require forward recovery'; END $$;
-- +goose StatementEnd
