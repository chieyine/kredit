-- +goose Up
CREATE TABLE app.native_identity_sessions (
 id uuid PRIMARY KEY DEFAULT uuidv7(),
 user_id uuid NOT NULL REFERENCES app.users(id),
 provider text NOT NULL,
 subject_id uuid NOT NULL,
 kind text NOT NULL CHECK(kind IN ('person','business','authority')),
 full_name text NOT NULL DEFAULT '',
 requires_review boolean NOT NULL DEFAULT false,
 person_id uuid, business_id uuid,
 state text NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','in_progress','verified','failed','expired','review')),
 safe_result jsonb NOT NULL DEFAULT '{}',
 document_id uuid REFERENCES app.documents(id),
 phone_reference text, phone_expires_at timestamptz,
 operation text NOT NULL DEFAULT '' CHECK(operation IN ('','phone_start','phone_verify','business_lookup')),
 attempts integer NOT NULL DEFAULT 0 CHECK(attempts BETWEEN 0 AND 5),
 consent_version text, consent_at timestamptz,
 reviewed_by uuid REFERENCES app.users(id), review_evidence text,
 version bigint NOT NULL DEFAULT 1,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), expires_at timestamptz,
 UNIQUE(provider,kind,subject_id)
);
ALTER TABLE app.native_identity_sessions ENABLE ROW LEVEL SECURITY;
CREATE POLICY native_identity_access ON app.native_identity_sessions USING(user_id=app.current_user_id() OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer'])) WITH CHECK(user_id=app.current_user_id() OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']));
-- +goose StatementBegin
CREATE FUNCTION app.guard_native_identity() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.user_id<>OLD.user_id OR NEW.provider<>OLD.provider OR NEW.subject_id<>OLD.subject_id OR NEW.kind<>OLD.kind OR NEW.full_name<>OLD.full_name OR NEW.requires_review<>OLD.requires_review OR NEW.person_id IS DISTINCT FROM OLD.person_id OR NEW.business_id IS DISTINCT FROM OLD.business_id THEN RAISE EXCEPTION 'verification subject is immutable'; END IF;
 IF OLD.state IN ('verified','failed') AND NOT (OLD.state='verified' AND OLD.expires_at<=now() AND NEW.state='pending' AND NEW.operation='' AND NEW.reviewed_by IS NULL AND NEW.review_evidence IS NULL AND NEW.safe_result='{}'::jsonb) THEN RAISE EXCEPTION 'verification decision is immutable'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER native_identity_immutable BEFORE UPDATE ON app.native_identity_sessions FOR EACH ROW EXECUTE FUNCTION app.guard_native_identity();
CREATE TRIGGER domain_activity AFTER INSERT OR UPDATE ON app.native_identity_sessions FOR EACH ROW EXECUTE FUNCTION app.record_domain_activity();
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT SELECT,INSERT,UPDATE ON app.native_identity_sessions TO kredit_app; END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Identity evidence requires forward recovery'; END $$;
-- +goose StatementEnd
