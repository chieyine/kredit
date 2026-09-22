-- +goose Up
-- Keep the legacy process slot available during rolling upgrades. New binaries
-- report one immutable boot identity each instead of overwriting that slot.
CREATE TABLE app.runtime_process_instances (
  boot_id uuid PRIMARY KEY,
  process text NOT NULL CHECK (process IN ('api','worker')),
  instance_name text NOT NULL CHECK (length(instance_name) BETWEEN 1 AND 128),
  version text NOT NULL CHECK (length(version) BETWEEN 1 AND 128),
  revision text NOT NULL CHECK (length(revision) BETWEEN 1 AND 128),
  versions jsonb NOT NULL CHECK (jsonb_typeof(versions)='object' AND octet_length(versions::text)<=65536),
  state text NOT NULL CHECK (state IN ('current','restart_required','applying','saved_configuration_unavailable','stopped')),
  started_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  updated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX runtime_process_instances_updated ON app.runtime_process_instances(updated_at DESC);
CREATE INDEX runtime_process_instances_prune ON app.runtime_process_instances(process,updated_at);
ALTER TABLE app.runtime_process_instances ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.runtime_process_instances FORCE ROW LEVEL SECURITY;
CREATE POLICY runtime_instances_read ON app.runtime_process_instances FOR SELECT
  USING (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY runtime_instances_insert ON app.runtime_process_instances FOR INSERT
  WITH CHECK ((process='api' AND current_user='kredit_app') OR (process='worker' AND current_user='kredit_worker'));
CREATE POLICY runtime_instances_update ON app.runtime_process_instances FOR UPDATE
  USING ((process='api' AND current_user='kredit_app') OR (process='worker' AND current_user='kredit_worker'))
  WITH CHECK ((process='api' AND current_user='kredit_app') OR (process='worker' AND current_user='kredit_worker'));
CREATE POLICY runtime_instances_prune ON app.runtime_process_instances FOR DELETE
  USING (updated_at < now()-interval '7 days' AND ((process='api' AND current_user='kredit_app') OR (process='worker' AND current_user='kredit_worker')));
-- +goose StatementBegin
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
    GRANT SELECT,INSERT,UPDATE,DELETE ON app.runtime_process_instances TO kredit_app;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
    GRANT SELECT,INSERT,UPDATE,DELETE ON app.runtime_process_instances TO kredit_worker;
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- Stop binaries that require instance heartbeats before rolling this back.
DROP TABLE app.runtime_process_instances;
