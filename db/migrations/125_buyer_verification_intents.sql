-- +goose Up
CREATE TABLE app.buyer_verification_intents(
 id uuid PRIMARY KEY DEFAULT uuidv7(),user_id uuid NOT NULL REFERENCES app.users(id),
 subject_type text NOT NULL CHECK(subject_type IN ('person','business','authority')),subject_id uuid NOT NULL,
 version bigint NOT NULL DEFAULT 1,provider text NOT NULL,state text NOT NULL DEFAULT 'NEW' CHECK(state IN ('NEW','STARTED','CREATED')),
 resolved_by uuid REFERENCES app.users(id),resolution_note text,resolved_at timestamptz,
 provider_reference text,created_at timestamptz NOT NULL DEFAULT now(),started_at timestamptz,
 UNIQUE(subject_type,subject_id,provider),UNIQUE(provider,provider_reference)
);
ALTER TABLE app.buyer_verification_intents ENABLE ROW LEVEL SECURITY;
CREATE POLICY buyer_verification_self ON app.buyer_verification_intents USING(user_id=app.current_user_id() OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer'])) WITH CHECK(user_id=app.current_user_id() OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']));
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT SELECT,INSERT,UPDATE ON app.buyer_verification_intents TO kredit_app; END IF;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION app.guard_buyer_verification_intent() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'verification history is immutable'; END IF;
 IF (to_jsonb(NEW)-ARRAY['version','state','provider_reference','started_at','resolved_by','resolution_note','resolved_at']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['version','state','provider_reference','started_at','resolved_by','resolution_note','resolved_at']) THEN RAISE EXCEPTION 'verification subject is immutable'; END IF;
 IF OLD.state='CREATED' AND (NEW.state<>'CREATED' OR NEW.provider_reference IS DISTINCT FROM OLD.provider_reference) THEN RAISE EXCEPTION 'verification reference is immutable'; END IF;
 IF OLD.state='STARTED' AND NEW.state='NEW' AND (NEW.resolved_by IS DISTINCT FROM app.current_user_id() OR NOT app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer'])) THEN RAISE EXCEPTION 'provider evidence review is required before retrying'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER buyer_verification_intent_immutable BEFORE UPDATE OR DELETE ON app.buyer_verification_intents FOR EACH ROW EXECUTE FUNCTION app.guard_buyer_verification_intent();
CREATE TRIGGER domain_activity AFTER INSERT OR UPDATE ON app.buyer_verification_intents FOR EACH ROW EXECUTE FUNCTION app.record_domain_activity();
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Verification evidence requires forward recovery'; END $$;
-- +goose StatementEnd
