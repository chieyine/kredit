-- +goose Up
CREATE TABLE app.business_credit_control_history (
 organization_id uuid NOT NULL REFERENCES app.organizations(id),
 version bigint NOT NULL,
 enabled boolean NOT NULL,
 threshold_kobo bigint NOT NULL,
 changed_by uuid NOT NULL REFERENCES app.users(id),
 changed_at timestamptz NOT NULL,
 PRIMARY KEY(organization_id,version)
);
ALTER TABLE app.business_credit_control_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.business_credit_control_history FORCE ROW LEVEL SECURITY;
CREATE POLICY credit_control_history_read ON app.business_credit_control_history FOR SELECT USING (
 organization_id=app.current_organization_id() AND EXISTS(SELECT 1 FROM app.memberships WHERE organization_id=business_credit_control_history.organization_id AND user_id=app.current_user_id() AND status='active' AND role IN ('owner','administrator','finance')));
-- +goose StatementBegin
CREATE FUNCTION app.record_credit_control_version() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
BEGIN
 IF TG_OP='UPDATE' AND NEW.version<>OLD.version+1 THEN RAISE EXCEPTION 'credit control version must advance exactly once'; END IF;
 INSERT INTO app.business_credit_control_history(organization_id,version,enabled,threshold_kobo,changed_by,changed_at)
 VALUES(NEW.organization_id,NEW.version,NEW.enabled,NEW.threshold_kobo,NEW.updated_by,NEW.updated_at);
 RETURN NEW;
END $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.record_credit_control_version() FROM PUBLIC;
INSERT INTO app.business_credit_control_history(organization_id,version,enabled,threshold_kobo,changed_by,changed_at)
 SELECT organization_id,version,enabled,threshold_kobo,updated_by,updated_at FROM app.business_credit_controls;
CREATE TRIGGER credit_control_version AFTER INSERT OR UPDATE ON app.business_credit_controls FOR EACH ROW EXECUTE FUNCTION app.record_credit_control_version();
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
 GRANT SELECT ON app.business_credit_control_history TO kredit_app;
END IF; END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Credit control history requires forward recovery'; END $$;
-- +goose StatementEnd
