-- +goose Up
-- Local authorization sessions fence bank-token creation before external I/O.
CREATE TABLE app.bank_debit_enrollments (
 provider text NOT NULL, reference text NOT NULL,
 user_id uuid NOT NULL REFERENCES app.users(id),
 version bigint NOT NULL DEFAULT 1, input jsonb NOT NULL, state text NOT NULL DEFAULT 'DRAFT' CHECK(state IN ('DRAFT','STARTED','CONFIRMED','CANCELLED')),
 reviewed_by uuid REFERENCES app.users(id),review_note text NOT NULL DEFAULT '',
 details_ciphertext text NOT NULL DEFAULT '', result jsonb NOT NULL DEFAULT '{}',
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(provider,reference)
);
ALTER TABLE app.bank_debit_enrollments ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.bank_debit_enrollments FORCE ROW LEVEL SECURITY;
CREATE POLICY bank_debit_owner ON app.bank_debit_enrollments FOR ALL
 USING(user_id=app.current_user_id()) WITH CHECK(user_id=app.current_user_id());
CREATE POLICY bank_debit_operator_read ON app.bank_debit_enrollments FOR SELECT
 USING(app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']));
-- Existing payment_mandate_by_provider supplies routing identity. No new
-- security-definer access bypass is needed for this protected table.
CREATE UNIQUE INDEX bank_debit_vendor_reference ON app.bank_debit_enrollments(provider,(result->>'reference')) WHERE state='CONFIRMED';
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
 GRANT SELECT,INSERT,UPDATE ON app.bank_debit_enrollments TO kredit_app;
 END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
 GRANT SELECT,INSERT,UPDATE ON app.bank_debit_enrollments TO kredit_worker;
 END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Bank authorization evidence requires forward recovery'; END $$;
-- +goose StatementEnd
