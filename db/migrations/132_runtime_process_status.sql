-- +goose Up
CREATE TABLE app.runtime_process_status (
 process text PRIMARY KEY CHECK(process IN ('api','worker')),
 versions jsonb NOT NULL CHECK(jsonb_typeof(versions)='object'),
 state text NOT NULL CHECK(state IN ('current','restart_required','applying','saved_configuration_unavailable')),
 updated_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE app.runtime_process_status ENABLE ROW LEVEL SECURITY;
CREATE POLICY runtime_status_read ON app.runtime_process_status FOR SELECT USING (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY runtime_status_insert ON app.runtime_process_status FOR INSERT WITH CHECK ((process='api' AND current_user='kredit_app') OR (process='worker' AND current_user='kredit_worker'));
CREATE POLICY runtime_status_update ON app.runtime_process_status FOR UPDATE USING ((process='api' AND current_user='kredit_app') OR (process='worker' AND current_user='kredit_worker')) WITH CHECK ((process='api' AND current_user='kredit_app') OR (process='worker' AND current_user='kredit_worker'));
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT SELECT,INSERT,UPDATE ON app.runtime_process_status TO kredit_app; REVOKE DELETE ON app.runtime_process_status FROM kredit_app; END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN GRANT SELECT,INSERT,UPDATE ON app.runtime_process_status TO kredit_worker; REVOKE DELETE ON app.runtime_process_status FROM kredit_worker; END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
DROP TABLE app.runtime_process_status;
