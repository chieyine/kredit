-- +goose Up
-- Buying permissions never widen a supplier-side branch restriction.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.branch_customer_access(p_org uuid,p_business uuid) RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT app.branch_scope_all(p_org) OR EXISTS(
 SELECT 1 FROM app.member_branch_scopes s JOIN app.memberships m ON m.id=s.membership_id AND m.purchasing_authority_version=s.membership_version JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id
 JOIN app.partner_assignments a ON a.organization_id=s.organization_id AND a.buyer_business_id=p_business JOIN app.business_branches b ON b.id=a.branch_id AND b.organization_id=a.organization_id
 WHERE s.organization_id=p_org AND s.user_id=app.current_user_id() AND m.user_id=s.user_id AND m.organization_id=s.organization_id AND m.status='active' AND u.status='active' AND o.status<>'suspended' AND s.mode='branches' AND a.branch_id=ANY(s.branch_ids) AND b.active);
$$;

CREATE POLICY branch_boundary ON app.mandate_authorization_intents AS RESTRICTIVE FOR ALL USING(app.branch_customer_access(supplier_organization_id,business_id));
CREATE POLICY branch_boundary ON app.payment_mandates AS RESTRICTIVE FOR ALL USING(app.branch_scope_all(supplier_organization_id) OR (buyer_subject_type='business' AND app.branch_customer_access(supplier_organization_id,buyer_subject_id)));
CREATE POLICY branch_boundary ON app.relationship_consents AS RESTRICTIVE FOR ALL USING(buyer_user_id=app.current_user_id() OR app.branch_scope_all(supplier_organization_id) OR EXISTS(SELECT 1 FROM app.businesses b WHERE b.owner_user_id=relationship_consents.buyer_user_id AND app.branch_customer_access(relationship_consents.supplier_organization_id,b.id)));
CREATE POLICY branch_boundary ON app.seller_settlement_receipts AS RESTRICTIVE FOR ALL USING(app.branch_scope_all(supplier_organization_id) OR EXISTS(SELECT 1 FROM app.payments p WHERE p.id=seller_settlement_receipts.payment_id AND app.branch_obligation_access(p.obligation_id)));
CREATE POLICY branch_boundary ON app.settlement_events AS RESTRICTIVE FOR ALL USING(app.branch_scope_all(supplier_organization_id) OR EXISTS(SELECT 1 FROM app.payments p WHERE p.id=settlement_events.payment_id AND app.branch_obligation_access(p.obligation_id)));
CREATE POLICY branch_boundary ON app.split_fee_allocations AS RESTRICTIVE FOR ALL USING(app.branch_scope_all(supplier_organization_id) OR EXISTS(SELECT 1 FROM app.payments p WHERE p.id=split_fee_allocations.payment_id AND app.branch_obligation_access(p.obligation_id)));
CREATE OR REPLACE FUNCTION app.trade_line_mandate(p_mandate_id uuid,p_buyer_user_id uuid,p_buyer_business_id uuid,p_supplier_organization_id uuid)
RETURNS TABLE(id text,provider text,provider_mandate_id text,buyer_business_id text,buyer_user_id text,supplier_organization_id text,amount_ceiling_kobo bigint,state text,created_at timestamptz)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT m.id::text,m.provider,m.provider_mandate_id,m.buyer_subject_id::text,b.owner_user_id::text,m.supplier_organization_id::text,m.amount_ceiling_kobo,m.state,m.created_at FROM app.payment_mandates m JOIN app.businesses b ON b.id=m.buyer_subject_id WHERE m.id=p_mandate_id AND m.buyer_subject_type='business' AND m.buyer_subject_id=p_buyer_business_id AND b.owner_user_id=p_buyer_user_id AND m.supplier_organization_id=p_supplier_organization_id AND app.branch_customer_access(p_supplier_organization_id,p_buyer_business_id);
$$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Related branch money boundaries require forward recovery'; END $$;
-- +goose StatementEnd
