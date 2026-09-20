-- +goose Up
-- Record history after a successful row write, not before an insert conflict.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.record_branch_scope() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE m app.memberships; bid uuid;
BEGIN
 IF NEW.updated_by IS DISTINCT FROM app.current_user_id() THEN RAISE EXCEPTION 'current owner required' USING ERRCODE='42501'; END IF;
 PERFORM 1 FROM app.memberships a JOIN app.users u ON u.id=a.user_id JOIN app.organizations o ON o.id=a.organization_id WHERE a.organization_id=NEW.organization_id AND a.user_id=NEW.updated_by AND a.role='owner' AND a.status='active' AND u.status='active' AND o.status<>'suspended' FOR SHARE OF a,u,o;
 IF NOT FOUND THEN RAISE EXCEPTION 'current owner required' USING ERRCODE='42501'; END IF;
 SELECT * INTO m FROM app.memberships WHERE organization_id=NEW.organization_id AND user_id=NEW.user_id AND status='active' FOR UPDATE;
 IF NOT FOUND OR m.role IN ('owner','administrator') THEN RAISE EXCEPTION 'choose active non-administrative staff' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND (NEW.organization_id<>OLD.organization_id OR NEW.user_id<>OLD.user_id OR NEW.version<>OLD.version+1) THEN RAISE EXCEPTION 'next scope version required' USING ERRCODE='23514'; END IF;
 IF TG_OP='INSERT' AND NEW.version<>1 THEN RAISE EXCEPTION 'initial scope version required' USING ERRCODE='23514'; END IF;
 IF EXISTS(SELECT 1 FROM unnest(NEW.branch_ids) x GROUP BY x HAVING count(*)>1) THEN RAISE EXCEPTION 'duplicate branches' USING ERRCODE='23514'; END IF;
 FOREACH bid IN ARRAY NEW.branch_ids LOOP
  PERFORM 1 FROM app.business_branches WHERE organization_id=NEW.organization_id AND id=bid AND active FOR SHARE;
  IF NOT FOUND THEN RAISE EXCEPTION 'choose active branches in this business' USING ERRCODE='23514'; END IF;
 END LOOP;
 NEW.membership_id:=m.id; NEW.membership_version:=m.purchasing_authority_version;
 RETURN NEW;
END $$;
CREATE OR REPLACE FUNCTION app.append_branch_scope_history() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
BEGIN
 INSERT INTO app.member_branch_scope_history(organization_id,user_id,version,actor_id,evidence) VALUES(NEW.organization_id,NEW.user_id,NEW.version,NEW.updated_by,to_jsonb(NEW));
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS branch_scope_committed_history ON app.member_branch_scopes;
CREATE TRIGGER branch_scope_committed_history AFTER INSERT OR UPDATE ON app.member_branch_scopes FOR EACH ROW EXECUTE FUNCTION app.append_branch_scope_history();
REVOKE ALL ON FUNCTION app.append_branch_scope_history() FROM PUBLIC;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Branch scope replay requires forward recovery'; END $$;
-- +goose StatementEnd
