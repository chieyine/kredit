-- +goose Up
CREATE TABLE app.fee_bank_receipts(
 id uuid PRIMARY KEY DEFAULT uuidv7(),organization_id uuid NOT NULL REFERENCES app.organizations(id),provider text NOT NULL,
 bank_reference text NOT NULL UNIQUE,amount_kobo bigint NOT NULL CHECK(amount_kobo>0),direction text NOT NULL CHECK(direction IN ('received','returned')),
 occurred_at timestamptz NOT NULL,recorded_by uuid NOT NULL REFERENCES app.users(id),evidence text NOT NULL CHECK(length(evidence) BETWEEN 20 AND 2000),created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE app.fee_bank_receipts ENABLE ROW LEVEL SECURITY;
CREATE POLICY fee_bank_read ON app.fee_bank_receipts FOR SELECT USING(organization_id=app.current_organization_id());
CREATE POLICY fee_bank_write ON app.fee_bank_receipts FOR INSERT WITH CHECK(organization_id=app.current_organization_id() AND recorded_by=app.current_user_id() AND app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']));
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT SELECT,INSERT ON app.fee_bank_receipts TO kredit_app; END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN GRANT SELECT ON app.fee_bank_receipts TO kredit_worker; END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Bank evidence requires forward recovery'; END $$;
-- +goose StatementEnd
