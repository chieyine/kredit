-- +goose Up
CREATE TABLE app.mandate_authorization_intents(
 id uuid PRIMARY KEY DEFAULT uuidv7(),user_id uuid NOT NULL REFERENCES app.users(id),business_id uuid NOT NULL REFERENCES app.businesses(id),supplier_organization_id uuid REFERENCES app.organizations(id),
 provider text NOT NULL,purpose text NOT NULL,reference text NOT NULL UNIQUE,input jsonb NOT NULL,
 state text NOT NULL DEFAULT 'STARTED' CHECK(state IN ('STARTED','CONFIRMED','NOT_CREATED')),provider_reference text,
 created_at timestamptz NOT NULL DEFAULT now(),resolved_at timestamptz,resolved_by uuid REFERENCES app.users(id),resolution_note text
);
CREATE UNIQUE INDEX mandate_authorization_live ON app.mandate_authorization_intents(provider,business_id,purpose) WHERE state='STARTED';
ALTER TABLE app.mandate_authorization_intents ENABLE ROW LEVEL SECURITY;
CREATE POLICY mandate_authorization_access ON app.mandate_authorization_intents USING(user_id=app.current_user_id() OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer'])) WITH CHECK(user_id=app.current_user_id() OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']));
-- +goose StatementBegin
CREATE FUNCTION app.guard_mandate_authorization_intent() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'authorization evidence is immutable'; END IF;
 IF OLD.state<>'STARTED' OR (to_jsonb(NEW)-ARRAY['state','provider_reference','resolved_at','resolved_by','resolution_note']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['state','provider_reference','resolved_at','resolved_by','resolution_note']) THEN RAISE EXCEPTION 'authorization intent is immutable'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER mandate_authorization_immutable BEFORE UPDATE OR DELETE ON app.mandate_authorization_intents FOR EACH ROW EXECUTE FUNCTION app.guard_mandate_authorization_intent();
CREATE TRIGGER domain_activity AFTER INSERT OR UPDATE ON app.mandate_authorization_intents FOR EACH ROW EXECUTE FUNCTION app.record_domain_activity();
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT SELECT,INSERT,UPDATE ON app.mandate_authorization_intents TO kredit_app; END IF; END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Authorization evidence requires forward recovery'; END $$;
-- +goose StatementEnd
