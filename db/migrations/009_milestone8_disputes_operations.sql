-- +goose Up
CREATE TABLE app.disputes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  obligation_id uuid NOT NULL REFERENCES app.obligations(id),
  supplier_organization_id uuid NOT NULL REFERENCES app.organizations(id),
  buyer_user_id uuid NOT NULL REFERENCES app.users(id),
  opened_by uuid NOT NULL REFERENCES app.users(id),
  total_disputed_kobo bigint NOT NULL CHECK (total_disputed_kobo > 0),
  remaining_disputed_kobo bigint NOT NULL CHECK (remaining_disputed_kobo >= 0),
  reason text NOT NULL,
  explanation text,
  state text NOT NULL CHECK (state IN ('OPEN','UNDER_REVIEW','PARTIALLY_RESOLVED','RESOLVED','WITHDRAWN')),
  collection_effect text NOT NULL CHECK (collection_effect IN ('FULL_BLOCK','CONTESTED_ONLY','NO_AUTOMATIC_BLOCK')),
  assigned_reviewer uuid REFERENCES app.users(id),
  opened_at timestamptz NOT NULL DEFAULT now(),
  resolved_at timestamptz
);
CREATE TABLE app.dispute_evidence (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  dispute_id uuid NOT NULL REFERENCES app.disputes(id),
  submitted_by uuid NOT NULL REFERENCES app.users(id),
  document_id uuid,
  statement text,
  submitted_at timestamptz NOT NULL DEFAULT now(),
  CHECK (document_id IS NOT NULL OR statement IS NOT NULL)
);
CREATE TABLE app.dispute_decisions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  dispute_id uuid NOT NULL REFERENCES app.disputes(id),
  reviewer_id uuid NOT NULL REFERENCES app.users(id),
  outcome text NOT NULL,
  valid_principal_kobo bigint NOT NULL DEFAULT 0 CHECK (valid_principal_kobo >= 0),
  adjustment_kobo bigint NOT NULL DEFAULT 0 CHECK (adjustment_kobo >= 0),
  remaining_disputed_kobo bigint NOT NULL CHECK (remaining_disputed_kobo >= 0),
  reason text NOT NULL,
  decided_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.operation_actions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_user_id uuid NOT NULL REFERENCES app.users(id),
  organization_id uuid REFERENCES app.organizations(id),
  action text NOT NULL,
  resource_type text NOT NULL,
  resource_id uuid NOT NULL,
  reason text NOT NULL,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE app.disputes ENABLE ROW LEVEL SECURITY;
CREATE POLICY dispute_supplier_access ON app.disputes USING (supplier_organization_id IN (SELECT organization_id FROM app.memberships WHERE user_id = app.current_user_id() AND status = 'active'));
CREATE POLICY dispute_buyer_access ON app.disputes USING (buyer_user_id = app.current_user_id());
ALTER TABLE app.dispute_evidence ENABLE ROW LEVEL SECURITY;
CREATE POLICY dispute_evidence_access ON app.dispute_evidence USING (dispute_id IN (SELECT id FROM app.disputes));
ALTER TABLE app.dispute_decisions ENABLE ROW LEVEL SECURITY;
CREATE POLICY dispute_decision_access ON app.dispute_decisions USING (dispute_id IN (SELECT id FROM app.disputes));
ALTER TABLE app.operation_actions ENABLE ROW LEVEL SECURITY;
CREATE POLICY operation_action_support_access ON app.operation_actions USING (current_user = 'kredit_app');

-- +goose Down
DROP TABLE IF EXISTS app.operation_actions, app.dispute_decisions, app.dispute_evidence, app.disputes;
