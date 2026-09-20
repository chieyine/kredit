-- +goose Up
-- Reject unusable grants instead of displaying them as active permissions.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.bind_purchasing_delegation_membership() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE member_role text;
BEGIN
 NEW.membership_id:=NULL; NEW.membership_authority_version:=NULL;
 IF cardinality(NEW.actions)>0 THEN
  SELECT id,purchasing_authority_version,role INTO NEW.membership_id,NEW.membership_authority_version,member_role FROM app.memberships WHERE organization_id=NEW.organization_id AND user_id=NEW.user_id AND status='active' FOR SHARE;
  IF NOT FOUND THEN RAISE EXCEPTION 'current membership required' USING ERRCODE='23514'; END IF;
  IF member_role<>'owner' AND EXISTS(SELECT 1 FROM app.businesses WHERE organization_id=NEW.organization_id AND owner_user_id=NEW.user_id) THEN
   RAISE EXCEPTION 'recorded purchasing owner requires the owner role' USING ERRCODE='23514';
  END IF;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Purchasing owner grant validation requires forward recovery'; END $$;
-- +goose StatementEnd
