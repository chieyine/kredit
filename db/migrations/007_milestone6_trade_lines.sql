-- +goose Up
CREATE TABLE app.trade_lines (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  supplier_organization_id uuid NOT NULL REFERENCES app.organizations(id),
  buyer_user_id uuid NOT NULL REFERENCES app.users(id),
  buyer_business_id uuid NOT NULL,
  approved_limit_kobo bigint NOT NULL CHECK (approved_limit_kobo > 0),
  current_exposure_kobo bigint NOT NULL DEFAULT 0 CHECK (current_exposure_kobo >= 0),
  reserved_pending_kobo bigint NOT NULL DEFAULT 0 CHECK (reserved_pending_kobo >= 0),
  available_limit_kobo bigint NOT NULL CHECK (available_limit_kobo >= 0),
  cadence text NOT NULL,
  default_grace_hours integer NOT NULL CHECK (default_grace_hours BETWEEN 0 AND 720),
  start_at timestamptz NOT NULL,
  end_at timestamptz NOT NULL,
  state text NOT NULL CHECK (state IN ('PROPOSED','PENDING_BUYER_ACCEPTANCE','PENDING_MANDATE','ACTIVE','SUSPENDED','EXPIRED','CLOSED')),
  mandate_id uuid,
  mandate_active boolean NOT NULL DEFAULT false,
  suspension_reason text,
  terms_version text NOT NULL,
  version bigint NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (end_at > start_at),
  CHECK (available_limit_kobo = approved_limit_kobo - current_exposure_kobo - reserved_pending_kobo)
);
CREATE TABLE app.drawdowns (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  trade_line_id uuid NOT NULL REFERENCES app.trade_lines(id),
  principal_kobo bigint NOT NULL CHECK (principal_kobo > 0),
  goods_description text NOT NULL,
  invoice_reference text,
  state text NOT NULL CHECK (state IN ('PENDING_BUYER_CONFIRMATION','BUYER_CONFIRMED','ACTIVATED','CANCELLED')),
  reservation_id uuid,
  obligation_id uuid REFERENCES app.obligations(id),
  buyer_confirmed_at timestamptz,
  activated_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.drawdown_reservations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  trade_line_id uuid NOT NULL REFERENCES app.trade_lines(id),
  drawdown_id uuid NOT NULL REFERENCES app.drawdowns(id),
  amount_kobo bigint NOT NULL CHECK (amount_kobo > 0),
  state text NOT NULL CHECK (state IN ('PENDING','CONFIRMED','CONVERTED','EXPIRED','RELEASED')),
  expires_at timestamptz NOT NULL,
  idempotency_key text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE app.trade_lines ENABLE ROW LEVEL SECURITY;
CREATE POLICY trade_line_supplier_access ON app.trade_lines USING (supplier_organization_id IN (SELECT organization_id FROM app.memberships WHERE user_id = app.current_user_id() AND status = 'active'));
CREATE POLICY trade_line_buyer_access ON app.trade_lines USING (buyer_user_id = app.current_user_id());
ALTER TABLE app.drawdowns ENABLE ROW LEVEL SECURITY;
CREATE POLICY drawdown_line_access ON app.drawdowns USING (trade_line_id IN (SELECT id FROM app.trade_lines));
ALTER TABLE app.drawdown_reservations ENABLE ROW LEVEL SECURITY;
CREATE POLICY reservation_line_access ON app.drawdown_reservations USING (trade_line_id IN (SELECT id FROM app.trade_lines));

-- +goose Down
DROP TABLE IF EXISTS app.drawdown_reservations, app.drawdowns, app.trade_lines;
