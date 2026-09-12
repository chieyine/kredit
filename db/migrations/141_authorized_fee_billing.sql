-- +goose Up
CREATE TABLE app.fee_authorizations(
 id uuid PRIMARY KEY DEFAULT uuidv7(),organization_id uuid NOT NULL REFERENCES app.organizations(id),provider text NOT NULL,
 state text NOT NULL CHECK(state IN ('customer_pending','customer_ready','mandate_pending','ready','paused')),
 customer_reference text NOT NULL DEFAULT '',mandate_reference text NOT NULL DEFAULT '',authorization_url text NOT NULL DEFAULT '',
 ceiling_kobo bigint NOT NULL CHECK(ceiling_kobo BETWEEN 20000 AND 2500000000),starts_at timestamptz NOT NULL,ends_at timestamptz NOT NULL,
 identity_fingerprint text NOT NULL,consent_version text NOT NULL,created_by uuid NOT NULL REFERENCES app.users(id),
 approved_by uuid REFERENCES app.users(id),approved_at timestamptz,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX fee_auth_mandate_unique ON app.fee_authorizations(provider,mandate_reference) WHERE mandate_reference!='';
CREATE UNIQUE INDEX fee_auth_one_open ON app.fee_authorizations(organization_id) WHERE state!='paused';
CREATE TABLE app.fee_debits(
 id uuid PRIMARY KEY DEFAULT uuidv7(),organization_id uuid NOT NULL REFERENCES app.organizations(id),authorization_id uuid NOT NULL REFERENCES app.fee_authorizations(id),
 invoice_id uuid NOT NULL REFERENCES app.fee_invoices(id),amount_kobo bigint NOT NULL CHECK(amount_kobo>0),
 state text NOT NULL CHECK(state IN ('pending','succeeded','failed','reversed')),created_at timestamptz NOT NULL DEFAULT now(),checked_at timestamptz,
 UNIQUE(invoice_id)
);
ALTER TABLE app.fee_authorizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.fee_debits ENABLE ROW LEVEL SECURITY;
CREATE POLICY fee_auth_tenant ON app.fee_authorizations USING(organization_id=app.current_organization_id());
CREATE POLICY fee_debit_tenant ON app.fee_debits USING(organization_id=app.current_organization_id());
-- +goose StatementBegin
CREATE FUNCTION app.fee_billing_work() RETURNS TABLE(organization_id uuid) LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$ SELECT DISTINCT a.organization_id FROM app.fee_authorizations a WHERE a.state='ready' OR EXISTS(SELECT 1 FROM app.fee_debits d WHERE d.authorization_id=a.id AND d.state='pending') $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.fee_billing_work() FROM PUBLIC;
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.invoice_billing_work() RETURNS TABLE(organization_id uuid) LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT p.organization_id FROM app.supplier_onboarding_profiles p JOIN app.invoice_billing_approvals a USING(organization_id)
 WHERE p.billing_state='configured' AND p.billing_method IN ('consolidated_invoice','authorized_debit') AND p.billing_provider_reference=a.billing_reference ORDER BY p.organization_id
$$;
-- +goose StatementEnd
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT SELECT,INSERT,UPDATE ON app.fee_authorizations,app.fee_debits TO kredit_app;END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN GRANT SELECT,UPDATE(approved_at) ON app.fee_authorizations TO kredit_worker;GRANT SELECT,INSERT,UPDATE ON app.fee_debits TO kredit_worker;GRANT EXECUTE ON FUNCTION app.fee_billing_work() TO kredit_worker;END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Fee debit evidence requires forward recovery'; END $$;
-- +goose StatementEnd
