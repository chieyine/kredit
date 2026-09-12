-- +goose Up
ALTER TABLE app.fee_invoice_lines ADD COLUMN collected_at_issue_kobo bigint NOT NULL DEFAULT 0 CHECK(collected_at_issue_kobo>=0);
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.invoice_billing_work() RETURNS TABLE(organization_id uuid) LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT p.organization_id FROM app.supplier_onboarding_profiles p JOIN app.invoice_billing_approvals a USING(organization_id)
 WHERE p.billing_state='configured' AND p.billing_method IN ('consolidated_invoice','authorized_debit','split_settlement') AND p.billing_provider_reference=a.billing_reference ORDER BY p.organization_id
$$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Fee bill deductions require forward recovery';END $$;
-- +goose StatementEnd
