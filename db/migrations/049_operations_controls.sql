-- +goose Up

ALTER TABLE app.users ADD COLUMN IF NOT EXISTS version bigint NOT NULL DEFAULT 1 CHECK (version > 0);
ALTER TABLE app.provider_webhook_inbox ADD COLUMN IF NOT EXISTS duplicate_count integer NOT NULL DEFAULT 0 CHECK (duplicate_count >= 0);
ALTER TABLE app.provider_webhook_inbox ADD COLUMN IF NOT EXISTS provider_sequence bigint;

CREATE TABLE app.operations_commands (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  command_type text NOT NULL CHECK (command_type IN (
    'retry_job','retry_webhook','suspend_user','restore_user','suspend_organization','restore_organization',
    'place_risk_hold','lift_risk_hold','request_reconciliation','resolve_unknown_submission',
    'retry_collection','cancel_collection')),
  target_type text NOT NULL,
  target_id text NOT NULL,
  organization_id uuid REFERENCES app.organizations(id),
  requested_by uuid NOT NULL REFERENCES app.users(id),
  reason text NOT NULL CHECK (length(trim(reason)) >= 8),
  expected_version bigint NOT NULL CHECK (expected_version > 0),
  idempotency_key text NOT NULL,
  impact_preview jsonb NOT NULL,
  correlation_id text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (requested_by,idempotency_key)
);

CREATE TABLE app.operations_command_events (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  command_id uuid NOT NULL REFERENCES app.operations_commands(id),
  state text NOT NULL CHECK (state IN ('PREVIEWED','APPLIED','REJECTED','FAILED')),
  result jsonb NOT NULL DEFAULT '{}'::jsonb,
  occurred_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX operations_command_events_command_idx ON app.operations_command_events(command_id,occurred_at);

CREATE TABLE app.platform_suspensions (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  target_type text NOT NULL CHECK (target_type IN ('user','organization')),
  target_id uuid NOT NULL,
  previous_status text NOT NULL,
  reason text NOT NULL CHECK (length(trim(reason)) >= 8),
  created_by uuid NOT NULL REFERENCES app.users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  lifted_by uuid REFERENCES app.users(id),
  lifted_reason text,
  lifted_at timestamptz,
  version bigint NOT NULL DEFAULT 1 CHECK (version > 0)
);
CREATE UNIQUE INDEX platform_suspension_active_unique ON app.platform_suspensions(target_type,target_id) WHERE lifted_at IS NULL;

CREATE TABLE app.risk_holds (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  target_type text NOT NULL CHECK (target_type IN ('buyer','supplier')),
  target_id uuid NOT NULL,
  scope text NOT NULL CHECK (scope IN ('credit','release','collection','settlement','all_sensitive')),
  reason text NOT NULL CHECK (length(trim(reason)) >= 8),
  expires_at timestamptz NOT NULL,
  created_by uuid NOT NULL REFERENCES app.users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  lifted_by uuid REFERENCES app.users(id),
  lifted_reason text,
  lifted_at timestamptz,
  version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
  CHECK (expires_at > created_at)
);
CREATE INDEX risk_holds_active_idx ON app.risk_holds(target_type,target_id,expires_at) WHERE lifted_at IS NULL;

CREATE TABLE app.reconciliation_cases (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  provider text NOT NULL,
  operation text NOT NULL,
  target_type text NOT NULL,
  target_id text NOT NULL,
  state text NOT NULL DEFAULT 'REQUESTED' CHECK (state IN ('REQUESTED','IN_PROGRESS','RESOLVED','FAILED')),
  reason text NOT NULL CHECK (length(trim(reason)) >= 8),
  resolution text,
  requested_by uuid NOT NULL REFERENCES app.users(id),
  resolved_by uuid REFERENCES app.users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  resolved_at timestamptz,
  version bigint NOT NULL DEFAULT 1 CHECK (version > 0)
);
CREATE INDEX reconciliation_cases_open_idx ON app.reconciliation_cases(created_at) WHERE state IN ('REQUESTED','IN_PROGRESS');

-- Operational tables are deliberately unavailable to tenant RLS context.
-- Only the bounded API/worker database roles may use them.
ALTER TABLE app.operations_commands ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.operations_command_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.platform_suspensions ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.risk_holds ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.reconciliation_cases ENABLE ROW LEVEL SECURITY;
CREATE POLICY operations_commands_runtime ON app.operations_commands USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY operations_command_events_runtime ON app.operations_command_events USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY platform_suspensions_runtime ON app.platform_suspensions USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY risk_holds_runtime ON app.risk_holds USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY reconciliation_cases_runtime ON app.reconciliation_cases USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));

-- Applied operational facts and their event history are append-only. State is
-- represented by new events or explicit lift columns on the governed record.
CREATE OR REPLACE FUNCTION app.reject_operations_command_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'operations command records are immutable'; END $$;
CREATE TRIGGER operations_commands_immutable BEFORE UPDATE OR DELETE ON app.operations_commands FOR EACH ROW EXECUTE FUNCTION app.reject_operations_command_mutation();
CREATE TRIGGER operations_command_events_immutable BEFORE UPDATE OR DELETE ON app.operations_command_events FOR EACH ROW EXECUTE FUNCTION app.reject_operations_command_mutation();

-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
    GRANT SELECT,INSERT,UPDATE ON app.operations_commands,app.operations_command_events,app.platform_suspensions,app.risk_holds,app.reconciliation_cases TO kredit_app;
    GRANT SELECT,UPDATE ON app.users,app.organizations,app.sessions,app.provider_webhook_inbox,app.collection_attempts TO kredit_app;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
    GRANT SELECT,INSERT,UPDATE ON app.operations_commands,app.operations_command_events,app.platform_suspensions,app.risk_holds,app.reconciliation_cases TO kredit_worker;
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS operations_command_events_immutable ON app.operations_command_events;
DROP TRIGGER IF EXISTS operations_commands_immutable ON app.operations_commands;
DROP FUNCTION IF EXISTS app.reject_operations_command_mutation();
DROP TABLE IF EXISTS app.reconciliation_cases,app.risk_holds,app.platform_suspensions,app.operations_command_events,app.operations_commands;
ALTER TABLE app.users DROP COLUMN IF EXISTS version;
ALTER TABLE app.provider_webhook_inbox DROP COLUMN IF EXISTS provider_sequence,DROP COLUMN IF EXISTS duplicate_count;
