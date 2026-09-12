-- +goose Up
ALTER TABLE app.fees ADD COLUMN waived_kobo bigint NOT NULL DEFAULT 0 CHECK(waived_kobo>=0 AND waived_kobo<=amount_kobo);
-- Old waivers lack line-level attribution. Do not guess it during an upgrade.
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM app.operation_actions WHERE action='fee_waiver') THEN
  RAISE EXCEPTION 'Reconcile historical fee waivers before installing invoice billing';
 END IF;
END $$;
-- +goose StatementEnd
CREATE TABLE app.fee_invoices(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 organization_id uuid NOT NULL REFERENCES app.organizations(id),
 billing_reference text NOT NULL,
 cycle text NOT NULL CHECK(cycle IN ('weekly','monthly')),
 period_end timestamptz NOT NULL,
 issued_at timestamptz NOT NULL DEFAULT now(),
 due_at timestamptz NOT NULL,
 business_name text NOT NULL,
 business_address text NOT NULL,
 payment_instructions text NOT NULL,
 UNIQUE(organization_id,period_end)
);
CREATE TABLE app.fee_invoice_lines(
 invoice_id uuid NOT NULL REFERENCES app.fee_invoices(id),
 fee_id uuid PRIMARY KEY REFERENCES app.fees(id),
 organization_id uuid NOT NULL REFERENCES app.organizations(id),
 amount_kobo bigint NOT NULL CHECK(amount_kobo>0),
 waived_at_issue_kobo bigint NOT NULL CHECK(waived_at_issue_kobo>=0),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.fee_invoice_receipts(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 invoice_id uuid NOT NULL REFERENCES app.fee_invoices(id),
 organization_id uuid NOT NULL REFERENCES app.organizations(id),
 bank_reference text NOT NULL UNIQUE CHECK(length(bank_reference) BETWEEN 3 AND 200),
 amount_kobo bigint NOT NULL CHECK(amount_kobo>0),
 received_at timestamptz NOT NULL,
 recorded_by uuid NOT NULL REFERENCES app.users(id),
 evidence text NOT NULL CHECK(length(evidence) BETWEEN 20 AND 2000),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.invoice_billing_approvals(
 organization_id uuid PRIMARY KEY REFERENCES app.organizations(id),
 billing_reference text NOT NULL,
 payment_instructions text NOT NULL CHECK(length(payment_instructions) BETWEEN 20 AND 2000),
 approved_by uuid NOT NULL REFERENCES app.users(id),
 approved_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE app.fee_invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.fee_invoice_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.fee_invoice_receipts ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.invoice_billing_approvals ENABLE ROW LEVEL SECURITY;
CREATE POLICY invoice_tenant ON app.fee_invoices USING(organization_id=app.current_organization_id()) WITH CHECK(organization_id=app.current_organization_id());
CREATE POLICY invoice_line_tenant ON app.fee_invoice_lines USING(organization_id=app.current_organization_id() AND EXISTS(SELECT 1 FROM app.fee_invoices i WHERE i.id=invoice_id AND i.organization_id=fee_invoice_lines.organization_id) AND EXISTS(SELECT 1 FROM app.fees f WHERE f.id=fee_id AND f.supplier_organization_id=fee_invoice_lines.organization_id));
CREATE POLICY invoice_receipt_tenant ON app.fee_invoice_receipts USING(organization_id=app.current_organization_id() AND EXISTS(SELECT 1 FROM app.fee_invoices i WHERE i.id=invoice_id AND i.organization_id=fee_invoice_receipts.organization_id));
CREATE POLICY invoice_approval_tenant ON app.invoice_billing_approvals USING(organization_id=app.current_organization_id());
INSERT INTO ledger.accounts(code,name,normal_balance) VALUES('PLATFORM_FEE_CASH','Received platform fees','debit') ON CONFLICT(code) DO NOTHING;
-- +goose StatementBegin
CREATE FUNCTION app.invoice_billing_work() RETURNS TABLE(organization_id uuid) LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT p.organization_id FROM app.supplier_onboarding_profiles p JOIN app.invoice_billing_approvals a USING(organization_id)
 WHERE p.billing_state='configured' AND p.billing_method='consolidated_invoice' AND p.billing_provider_reference=a.billing_reference ORDER BY p.organization_id
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION app.invoice_billing_review() RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
BEGIN
 IF NOT app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']) THEN RAISE EXCEPTION 'super-admin authority required'; END IF;
 RETURN jsonb_build_object('businesses',COALESCE((SELECT jsonb_agg(to_jsonb(r)) FROM (SELECT p.organization_id,o.legal_name,p.version,p.billing_method,p.billing_cycle,p.billing_provider_reference FROM app.supplier_onboarding_profiles p JOIN app.organizations o ON o.id=p.organization_id WHERE p.billing_state='pending_verification' ORDER BY p.billing_changed_at LIMIT 100) r),'[]'::jsonb),
 'invoices',COALESCE((SELECT jsonb_agg(to_jsonb(r)) FROM (SELECT id,organization_id,business_name,issued_at,due_at FROM app.fee_invoices ORDER BY issued_at DESC LIMIT 100) r),'[]'::jsonb));
END $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.invoice_billing_work(),app.invoice_billing_review() FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
  GRANT SELECT,INSERT ON app.fee_invoices,app.fee_invoice_lines,app.fee_invoice_receipts TO kredit_app;
  GRANT SELECT,INSERT,UPDATE ON app.invoice_billing_approvals TO kredit_app;
  GRANT EXECUTE ON FUNCTION app.invoice_billing_review() TO kredit_app;
 END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
  GRANT SELECT,INSERT ON app.fee_invoices,app.fee_invoice_lines TO kredit_worker;
  GRANT SELECT ON app.fee_invoice_receipts,app.invoice_billing_approvals TO kredit_worker;
  GRANT EXECUTE ON FUNCTION app.invoice_billing_work() TO kredit_worker;
  REVOKE INSERT,UPDATE,DELETE ON app.settlement_registrations FROM kredit_worker;
 END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Billing evidence requires forward recovery'; END $$;
-- +goose StatementEnd
