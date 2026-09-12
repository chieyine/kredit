-- +goose Up
ALTER TABLE app.fees ADD COLUMN collected_kobo bigint NOT NULL DEFAULT 0 CHECK(collected_kobo>=0 AND collected_kobo+waived_kobo<=amount_kobo);
CREATE TABLE app.split_fee_allocations (
 payment_id uuid NOT NULL REFERENCES app.payments(id),
 fee_id uuid NOT NULL REFERENCES app.fees(id),
 supplier_organization_id uuid NOT NULL REFERENCES app.organizations(id),
 amount_kobo bigint NOT NULL CHECK(amount_kobo>0),
 PRIMARY KEY(payment_id,fee_id)
);
ALTER TABLE app.split_fee_allocations ENABLE ROW LEVEL SECURITY;
CREATE POLICY split_fee_tenant ON app.split_fee_allocations USING(supplier_organization_id=app.current_organization_id() OR current_user='kredit_worker');
INSERT INTO ledger.accounts(code,name,normal_balance) VALUES('PLATFORM_FEE_PROVIDER_CLEARING','Platform fees held for bank settlement','debit') ON CONFLICT(code) DO NOTHING;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT SELECT,INSERT ON app.split_fee_allocations TO kredit_app; END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN GRANT SELECT,INSERT ON app.split_fee_allocations TO kredit_worker; END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Fee allocations require forward recovery'; END $$;
-- +goose StatementEnd
