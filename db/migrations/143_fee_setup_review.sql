-- +goose Up
-- A setup interrupted before the seller chooses billing must still be visible.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.invoice_billing_review() RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
BEGIN
 IF NOT app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']) THEN RAISE EXCEPTION 'super-admin authority required'; END IF;
 RETURN jsonb_build_object('businesses',COALESCE((SELECT jsonb_agg(to_jsonb(r)) FROM (SELECT p.organization_id,o.legal_name,p.version,p.billing_method,p.billing_cycle,p.billing_provider_reference FROM app.supplier_onboarding_profiles p JOIN app.organizations o ON o.id=p.organization_id WHERE p.billing_state='pending_verification' OR EXISTS(SELECT 1 FROM app.fee_authorizations a WHERE a.organization_id=p.organization_id AND a.state!='paused') ORDER BY p.billing_changed_at,p.organization_id) r),'[]'::jsonb),
 'invoices',COALESCE((SELECT jsonb_agg(to_jsonb(r)) FROM (SELECT id,organization_id,business_name,issued_at,due_at FROM app.fee_invoices ORDER BY issued_at DESC) r),'[]'::jsonb));
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Fee review requires forward recovery'; END $$;
-- +goose StatementEnd
