-- +goose Up
-- Binding a buying capability requires current ownership of the canonical entity.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.project_canonical_business_identity() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
DECLARE o app.organizations;
BEGIN
 IF TG_OP='UPDATE' AND OLD.organization_id IS NOT NULL AND NEW.organization_id IS DISTINCT FROM OLD.organization_id THEN
  RAISE EXCEPTION 'a purchasing capability cannot move to another trading identity' USING ERRCODE='23514';
 END IF;
 IF NEW.organization_id IS NULL THEN RETURN NEW; END IF;
 IF TG_OP='INSERT' OR OLD.organization_id IS NULL THEN
  PERFORM 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id WHERE m.organization_id=NEW.organization_id AND m.user_id=NEW.owner_user_id AND m.role='owner' AND m.status='active' AND u.status='active' FOR SHARE OF m,u;
  IF NOT FOUND THEN RAISE EXCEPTION 'current ownership required to bind a purchasing profile' USING ERRCODE='42501'; END IF;
 ELSIF (NEW.legal_name,NEW.trading_name,NEW.business_type,NEW.registration_info,NEW.business_address,NEW.industry) IS NOT DISTINCT FROM (OLD.legal_name,OLD.trading_name,OLD.business_type,OLD.registration_info,OLD.business_address,OLD.industry) THEN
  RETURN NEW;
 END IF;
 SELECT * INTO o FROM app.organizations WHERE id=NEW.organization_id FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'canonical business is unavailable' USING ERRCODE='23503'; END IF;
 NEW.legal_name:=o.legal_name; NEW.trading_name:=o.trading_name;
 NEW.business_type:=o.business_type; NEW.registration_info:=o.registration_info;
 NEW.business_address:=o.business_address; NEW.industry:=o.industry;
 RETURN NEW;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Canonical identity binding requires forward recovery'; END $$;
-- +goose StatementEnd
