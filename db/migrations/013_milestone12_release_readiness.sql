-- +goose Up
CREATE TABLE app.release_evidence (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  gate_name text NOT NULL,
  reference text NOT NULL,
  reviewed_by uuid NOT NULL REFERENCES app.users(id),
  reviewed_at timestamptz NOT NULL,
  state text NOT NULL CHECK (state IN ('pending','accepted','rejected')),
  notes text,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (gate_name, reference)
);
CREATE TABLE app.pilot_limit_configs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  version text NOT NULL UNIQUE,
  max_supplier_organizations bigint NOT NULL CHECK (max_supplier_organizations > 0),
  max_buyer_businesses bigint NOT NULL CHECK (max_buyer_businesses > 0),
  max_principal_kobo bigint NOT NULL CHECK (max_principal_kobo > 0),
  max_active_exposure_kobo bigint NOT NULL CHECK (max_active_exposure_kobo > 0),
  max_drawdowns_per_line_day bigint NOT NULL CHECK (max_drawdowns_per_line_day > 0),
  max_collection_retries bigint NOT NULL CHECK (max_collection_retries > 0),
  enhanced_review_kobo bigint NOT NULL CHECK (enhanced_review_kobo > 0),
  allowed_provider_accounts text NOT NULL,
  allowed_industries text NOT NULL,
  enabled boolean NOT NULL DEFAULT false,
  approved_by uuid REFERENCES app.users(id),
  approved_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE app.release_evidence ENABLE ROW LEVEL SECURITY;
CREATE POLICY release_evidence_support_access ON app.release_evidence USING (current_user = 'kredit_app');
ALTER TABLE app.pilot_limit_configs ENABLE ROW LEVEL SECURITY;
CREATE POLICY pilot_limits_support_access ON app.pilot_limit_configs USING (current_user = 'kredit_app');

-- +goose Down
DROP TABLE IF EXISTS app.pilot_limit_configs, app.release_evidence;
