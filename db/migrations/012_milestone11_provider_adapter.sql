-- +goose Up
CREATE TABLE app.provider_approvals (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  provider_name text NOT NULL,
  written_reference text NOT NULL,
  approved_by uuid NOT NULL REFERENCES app.users(id),
  approved_at timestamptz NOT NULL,
  allowed_capabilities jsonb NOT NULL DEFAULT '[]'::jsonb,
  pilot_limit_kobo bigint NOT NULL CHECK (pilot_limit_kobo > 0),
  state text NOT NULL CHECK (state IN ('pending','approved','revoked')),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (provider_name, written_reference)
);
CREATE TABLE app.provider_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  provider_name text NOT NULL,
  provider_event_id text NOT NULL,
  external_reference text NOT NULL,
  event_type text NOT NULL,
  payload_hash text NOT NULL,
  state text NOT NULL,
  settlement_state text,
  received_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (provider_name, provider_event_id)
);
CREATE TABLE app.provider_reconciliation_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  provider_name text NOT NULL,
  provider_collection_id text NOT NULL,
  state text NOT NULL,
  settlement_reference text,
  amount_kobo bigint,
  observed_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE app.provider_approvals ENABLE ROW LEVEL SECURITY;
CREATE POLICY provider_approval_support_access ON app.provider_approvals USING (current_user = 'kredit_app');
ALTER TABLE app.provider_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY provider_event_support_access ON app.provider_events USING (current_user = 'kredit_app');
ALTER TABLE app.provider_reconciliation_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY provider_reconciliation_support_access ON app.provider_reconciliation_events USING (current_user = 'kredit_app');

-- +goose Down
DROP TABLE IF EXISTS app.provider_reconciliation_events, app.provider_events, app.provider_approvals;
