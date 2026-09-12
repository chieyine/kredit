-- +goose Up
CREATE TABLE app.native_identity_history(
 id uuid PRIMARY KEY DEFAULT uuidv7(),session_id uuid NOT NULL REFERENCES app.native_identity_sessions(id),user_id uuid NOT NULL REFERENCES app.users(id),
 previous_version bigint NOT NULL,previous_decision jsonb NOT NULL,changed_by uuid REFERENCES app.users(id),recorded_at timestamptz NOT NULL DEFAULT now(),UNIQUE(session_id,previous_version)
);
ALTER TABLE app.native_identity_history ENABLE ROW LEVEL SECURITY;
CREATE POLICY native_history_access ON app.native_identity_history FOR SELECT USING(user_id=app.current_user_id() OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']));
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_native_identity() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE renewal boolean; appeal boolean;
BEGIN
 IF NEW.user_id IS DISTINCT FROM OLD.user_id OR NEW.provider IS DISTINCT FROM OLD.provider OR NEW.subject_id IS DISTINCT FROM OLD.subject_id OR NEW.kind IS DISTINCT FROM OLD.kind OR NEW.full_name IS DISTINCT FROM OLD.full_name OR NEW.requires_review IS DISTINCT FROM OLD.requires_review OR NEW.person_id IS DISTINCT FROM OLD.person_id OR NEW.business_id IS DISTINCT FROM OLD.business_id THEN RAISE EXCEPTION 'verification subject is immutable'; END IF;
 IF OLD.state IN ('verified','failed') THEN
  renewal:=OLD.state='verified' AND OLD.expires_at<=now() AND NEW.state='pending' AND NEW.operation='' AND NEW.reviewed_by IS NULL AND NEW.review_evidence IS NULL AND NEW.safe_result='{}'::jsonb;
  appeal:=OLD.state='failed' AND NEW.state='pending' AND NEW.operation='' AND NEW.safe_result='{}'::jsonb AND NEW.reviewed_by=app.current_user_id() AND length(NEW.review_evidence)>=20 AND app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']);
  IF NOT (COALESCE(renewal,false) OR COALESCE(appeal,false)) THEN RAISE EXCEPTION 'verification decision is immutable';END IF;
  INSERT INTO app.native_identity_history(session_id,user_id,previous_version,previous_decision,changed_by) VALUES(OLD.id,OLD.user_id,OLD.version,to_jsonb(OLD),app.current_user_id());
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT SELECT ON app.native_identity_history TO kredit_app;END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Identity review history requires forward recovery';END $$;
-- +goose StatementEnd
