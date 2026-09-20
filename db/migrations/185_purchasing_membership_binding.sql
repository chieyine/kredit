-- +goose Up
-- Rejoining or restoring a membership never revives an old purchasing grant.
ALTER TABLE app.memberships ADD COLUMN purchasing_authority_version bigint NOT NULL DEFAULT 1 CHECK(purchasing_authority_version>0);
ALTER TABLE app.purchasing_delegations ADD COLUMN membership_id uuid REFERENCES app.memberships(id), ADD COLUMN membership_authority_version bigint;
ALTER TABLE app.purchasing_delegations DISABLE TRIGGER purchasing_delegation_history;
UPDATE app.purchasing_delegations d SET membership_id=m.id,membership_authority_version=m.purchasing_authority_version FROM app.memberships m WHERE m.organization_id=d.organization_id AND m.user_id=d.user_id AND m.status='active';
ALTER TABLE app.purchasing_delegations ENABLE TRIGGER purchasing_delegation_history;
-- +goose StatementBegin
CREATE FUNCTION app.advance_purchasing_membership_authority() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,app AS $$
BEGIN
 IF (NEW.status,NEW.role,NEW.organization_id,NEW.user_id) IS DISTINCT FROM (OLD.status,OLD.role,OLD.organization_id,OLD.user_id) THEN
  NEW.purchasing_authority_version:=OLD.purchasing_authority_version+1;
 ELSE NEW.purchasing_authority_version:=OLD.purchasing_authority_version;
 END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION app.bind_purchasing_delegation_membership() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
BEGIN
 NEW.membership_id:=NULL; NEW.membership_authority_version:=NULL;
 IF cardinality(NEW.actions)>0 THEN
  SELECT id,purchasing_authority_version INTO NEW.membership_id,NEW.membership_authority_version FROM app.memberships WHERE organization_id=NEW.organization_id AND user_id=NEW.user_id AND status='active' FOR SHARE;
  IF NOT FOUND THEN RAISE EXCEPTION 'current membership required' USING ERRCODE='23514'; END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE OR REPLACE FUNCTION app.can_purchase(profile_id uuid,action_name text DEFAULT 'read',amount_kobo bigint DEFAULT 0) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT EXISTS(SELECT 1 FROM app.businesses b JOIN app.organizations o ON o.id=b.organization_id
 JOIN app.memberships m ON m.organization_id=o.id AND m.user_id=app.current_user_id()
 JOIN app.users u ON u.id=m.user_id
 LEFT JOIN app.purchasing_delegations d ON d.organization_id=o.id AND d.user_id=m.user_id
 WHERE b.id=profile_id AND m.status='active' AND u.status='active' AND o.status<>'suspended'
 AND ((m.role='owner' AND b.owner_user_id=m.user_id) OR
 (d.membership_id=m.id AND d.membership_authority_version=m.purchasing_authority_version
 AND d.expires_at>statement_timestamp() AND 'read'=ANY(d.actions) AND action_name=ANY(d.actions)
 AND (action_name<>'accept' OR amount_kobo BETWEEN 0 AND d.ceiling_kobo))));
$$;
REVOKE ALL ON FUNCTION app.advance_purchasing_membership_authority(),app.bind_purchasing_delegation_membership() FROM PUBLIC;
-- +goose StatementEnd
CREATE TRIGGER purchasing_membership_authority BEFORE UPDATE ON app.memberships FOR EACH ROW EXECUTE FUNCTION app.advance_purchasing_membership_authority();
CREATE TRIGGER purchasing_delegation_binding BEFORE INSERT OR UPDATE ON app.purchasing_delegations FOR EACH ROW EXECUTE FUNCTION app.bind_purchasing_delegation_membership();
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Purchasing membership binding requires forward recovery'; END $$;
-- +goose StatementEnd
