-- +goose Up
CREATE TABLE app.correction_requests (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES app.organizations(id),
  subject_type text NOT NULL,
  subject_id uuid NOT NULL,
  source_event_id text,
  requested_by uuid NOT NULL REFERENCES app.users(id),
  reason text NOT NULL,
  evidence jsonb NOT NULL DEFAULT '[]'::jsonb,
  state text NOT NULL CHECK (state IN ('OPEN','UNDER_REVIEW','APPROVED','REJECTED')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.correction_decisions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  request_id uuid NOT NULL REFERENCES app.correction_requests(id),
  reviewer_id uuid NOT NULL REFERENCES app.users(id),
  outcome text NOT NULL CHECK (outcome IN ('APPROVED','REJECTED')),
  reason text NOT NULL,
  correction_event_id text,
  decided_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.analytics_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  subject_id_hash text NOT NULL,
  purpose text NOT NULL,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  occurred_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE app.correction_requests ENABLE ROW LEVEL SECURITY;
CREATE POLICY correction_org_access ON app.correction_requests USING (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID);
ALTER TABLE app.correction_decisions ENABLE ROW LEVEL SECURITY;
CREATE POLICY correction_decision_org_access ON app.correction_decisions USING (EXISTS (SELECT 1 FROM app.correction_requests r WHERE r.id = request_id AND r.organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID));
ALTER TABLE app.analytics_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY analytics_support_access ON app.analytics_events USING (current_user = 'kredit_app');

-- +goose Down
DROP TABLE IF EXISTS app.analytics_events, app.correction_decisions, app.correction_requests;
