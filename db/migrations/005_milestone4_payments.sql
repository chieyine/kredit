-- +goose Up
CREATE TABLE app.payments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  obligation_id uuid NOT NULL REFERENCES app.obligations(id),
  buyer_user_id uuid NOT NULL REFERENCES app.users(id),
  supplier_organization_id uuid NOT NULL REFERENCES app.organizations(id),
  source_type text NOT NULL CHECK (source_type IN ('voluntary','collected')),
  amount_kobo bigint NOT NULL CHECK (amount_kobo > 0),
  currency char(3) NOT NULL,
  provider text,
  provider_reference text,
  state text NOT NULL CHECK (state IN ('recognized','reversed')),
  paid_at timestamptz NOT NULL,
  recognized_at timestamptz NOT NULL DEFAULT now(),
  recorded_by uuid NOT NULL REFERENCES app.users(id),
  reversal_of uuid REFERENCES app.payments(id),
  idempotency_key text NOT NULL UNIQUE
);
CREATE UNIQUE INDEX payments_provider_reference_idx ON app.payments(provider, provider_reference) WHERE provider_reference IS NOT NULL;

CREATE TABLE app.payment_allocations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  payment_id uuid NOT NULL REFERENCES app.payments(id),
  obligation_id uuid NOT NULL REFERENCES app.obligations(id),
  amount_kobo bigint NOT NULL CHECK (amount_kobo > 0),
  allocation_order integer NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.fees (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  supplier_organization_id uuid NOT NULL REFERENCES app.organizations(id),
  obligation_id uuid NOT NULL REFERENCES app.obligations(id),
  payment_id uuid REFERENCES app.payments(id),
  fee_type text NOT NULL CHECK (fee_type IN ('base_service','collection')),
  basis_amount_kobo bigint NOT NULL CHECK (basis_amount_kobo >= 0),
  rate_basis_points integer NOT NULL CHECK (rate_basis_points >= 0),
  amount_kobo bigint NOT NULL CHECK (amount_kobo >= 0),
  currency char(3) NOT NULL,
  state text NOT NULL CHECK (state IN ('accrued','paid','waived','refunded')),
  accrued_at timestamptz NOT NULL DEFAULT now(),
  paid_at timestamptz,
  UNIQUE (payment_id, fee_type)
);
CREATE TABLE app.settlement_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  payment_id uuid REFERENCES app.payments(id),
  supplier_organization_id uuid NOT NULL REFERENCES app.organizations(id),
  provider text NOT NULL,
  provider_settlement_reference text NOT NULL,
  gross_amount_kobo bigint NOT NULL CHECK (gross_amount_kobo >= 0),
  fee_amount_kobo bigint NOT NULL DEFAULT 0 CHECK (fee_amount_kobo >= 0),
  net_amount_kobo bigint NOT NULL CHECK (net_amount_kobo >= 0),
  state text NOT NULL,
  expected_at timestamptz,
  actual_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (provider, provider_settlement_reference)
);

ALTER TABLE app.payments ENABLE ROW LEVEL SECURITY;
CREATE POLICY payment_supplier_access ON app.payments USING (supplier_organization_id IN (SELECT organization_id FROM app.memberships WHERE user_id = app.current_user_id() AND status = 'active'));
CREATE POLICY payment_buyer_access ON app.payments USING (buyer_user_id = app.current_user_id());
ALTER TABLE app.payment_allocations ENABLE ROW LEVEL SECURITY;
CREATE POLICY allocation_access ON app.payment_allocations USING (obligation_id IN (SELECT id FROM app.obligations));
ALTER TABLE app.fees ENABLE ROW LEVEL SECURITY;
CREATE POLICY fee_supplier_access ON app.fees USING (supplier_organization_id IN (SELECT organization_id FROM app.memberships WHERE user_id = app.current_user_id() AND status = 'active'));
ALTER TABLE app.settlement_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY settlement_supplier_access ON app.settlement_events USING (supplier_organization_id IN (SELECT organization_id FROM app.memberships WHERE user_id = app.current_user_id() AND status = 'active'));

-- +goose Down
DROP TABLE IF EXISTS app.settlement_events, app.fees, app.payment_allocations, app.payments;
