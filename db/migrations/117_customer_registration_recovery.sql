-- +goose Up
CREATE TABLE app.customer_registration_attempts(
 id uuid PRIMARY KEY DEFAULT uuidv7(),business_id uuid NOT NULL REFERENCES app.businesses(id),
 user_id uuid NOT NULL REFERENCES app.users(id),provider text NOT NULL DEFAULT 'mono-sweep',
 identity_fingerprint text NOT NULL,consent_version text NOT NULL,
 state text NOT NULL DEFAULT 'PENDING' CHECK(state IN('PENDING','CONFIRMED','NOT_CREATED')),
 provider_reference text,resolved_by uuid REFERENCES app.users(id),resolution_note text,
 created_at timestamptz NOT NULL DEFAULT now(),resolved_at timestamptz
);
CREATE UNIQUE INDEX customer_registration_live ON app.customer_registration_attempts(provider,business_id) WHERE state<>'NOT_CREATED';
ALTER TABLE app.customer_registration_attempts ENABLE ROW LEVEL SECURITY;
CREATE POLICY customer_registration_access ON app.customer_registration_attempts
 USING(user_id=app.current_user_id() OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']))
 WITH CHECK(user_id=app.current_user_id() OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']));
-- +goose StatementBegin
CREATE FUNCTION app.guard_customer_registration() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'registration history is immutable'; END IF;
 IF OLD.state<>'PENDING' OR (NEW.id,NEW.business_id,NEW.user_id,NEW.provider,NEW.identity_fingerprint,NEW.consent_version,NEW.created_at)
 IS DISTINCT FROM (OLD.id,OLD.business_id,OLD.user_id,OLD.provider,OLD.identity_fingerprint,OLD.consent_version,OLD.created_at)
 THEN RAISE EXCEPTION 'registration intent is immutable'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER customer_registration_guard BEFORE UPDATE OR DELETE ON app.customer_registration_attempts FOR EACH ROW EXECUTE FUNCTION app.guard_customer_registration();
CREATE TRIGGER domain_activity AFTER INSERT OR UPDATE ON app.customer_registration_attempts FOR EACH ROW EXECUTE FUNCTION app.record_domain_activity();
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT SELECT,INSERT,UPDATE ON app.customer_registration_attempts TO kredit_app; END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Registration evidence requires forward recovery'; END $$;
-- +goose StatementEnd
