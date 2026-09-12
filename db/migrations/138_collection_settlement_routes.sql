-- +goose Up
CREATE TABLE app.collection_settlement_routes (
 attempt_id uuid PRIMARY KEY REFERENCES app.collection_attempts(id),
 obligation_id uuid NOT NULL REFERENCES app.obligations(id),
 supplier_organization_id uuid NOT NULL REFERENCES app.organizations(id),
 route jsonb NOT NULL CHECK(jsonb_typeof(route)='object'),
 created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE app.collection_settlement_routes ENABLE ROW LEVEL SECURITY;
CREATE POLICY collection_route_access ON app.collection_settlement_routes USING(
 current_user='kredit_worker' OR supplier_organization_id=app.current_organization_id() OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin'])
);
CREATE TABLE app.seller_settlement_receipts (
 id uuid PRIMARY KEY DEFAULT uuidv7(),
 payment_id uuid NOT NULL REFERENCES app.payments(id),
 attempt_id uuid NOT NULL REFERENCES app.collection_settlement_routes(attempt_id),
 supplier_organization_id uuid NOT NULL REFERENCES app.organizations(id),
 bank_reference text NOT NULL UNIQUE,
 amount_kobo bigint NOT NULL CHECK(amount_kobo>0),
 direction text NOT NULL CHECK(direction IN ('paid','returned')),
 occurred_at timestamptz NOT NULL,
 recorded_by uuid NOT NULL REFERENCES app.users(id),
 evidence text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE app.seller_settlement_receipts ENABLE ROW LEVEL SECURITY;
CREATE POLICY seller_receipt_read ON app.seller_settlement_receipts FOR SELECT USING(current_user='kredit_worker' OR supplier_organization_id=app.current_organization_id() OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin']));
CREATE POLICY seller_receipt_record ON app.seller_settlement_receipts FOR INSERT WITH CHECK(recorded_by=app.current_user_id() AND app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']));
INSERT INTO ledger.accounts(code,name,normal_balance) VALUES('SELLER_BANK_SETTLEMENT','Confirmed seller bank receipts','debit') ON CONFLICT(code) DO NOTHING;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
  GRANT SELECT,INSERT ON app.collection_settlement_routes,app.seller_settlement_receipts TO kredit_app;
 END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
  GRANT SELECT,INSERT ON app.collection_settlement_routes TO kredit_worker;
  GRANT SELECT ON app.seller_settlement_receipts TO kredit_worker;
 END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Settlement evidence requires forward recovery'; END $$;
-- +goose StatementEnd
