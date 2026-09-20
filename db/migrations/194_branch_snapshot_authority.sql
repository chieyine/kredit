-- +goose Up
-- Privileged branch readers must also retain the independent purchasing
-- revocation floor; tenant matching alone is not current owner authority.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.credit_snapshot_by_id(p_request_id text) RETURNS jsonb LANGUAGE sql SECURITY DEFINER STABLE SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT aggregate FROM app.credit_aggregate_snapshots WHERE credit_request_id=p_request_id AND app.branch_credit_access(credit_request_id::uuid) AND ((current_setting('role',true)='kredit_worker' OR (current_setting('role',true)='none' AND pg_has_role(session_user,'kredit_worker','member') AND NOT pg_has_role(session_user,'kredit_app','member'))) OR app.request_authority_current(credit_request_id::uuid)) AND
 (supplier_organization_id=NULLIF(current_setting('app.current_organization_id',true),'') OR buyer_user_id=NULLIF(current_setting('app.current_user_id',true),'') OR app.purchase_request_read(credit_request_id::uuid));
$$;
CREATE OR REPLACE FUNCTION app.credit_snapshot_by_obligation(p_obligation_id text) RETURNS TABLE(credit_request_id text,aggregate jsonb) LANGUAGE sql SECURITY DEFINER STABLE SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT credit_request_id,aggregate FROM app.credit_aggregate_snapshots WHERE aggregate->'obligation'->>'id'=p_obligation_id AND app.branch_credit_access(credit_request_id::uuid) AND ((current_setting('role',true)='kredit_worker' OR (current_setting('role',true)='none' AND pg_has_role(session_user,'kredit_worker','member') AND NOT pg_has_role(session_user,'kredit_app','member'))) OR app.request_authority_current(credit_request_id::uuid)) AND
 (supplier_organization_id=NULLIF(current_setting('app.current_organization_id',true),'') OR buyer_user_id=NULLIF(current_setting('app.current_user_id',true),'') OR app.purchase_request_read(credit_request_id::uuid)) LIMIT 1;
$$;
CREATE OR REPLACE FUNCTION app.trade_line_mandate(p_mandate_id uuid,p_buyer_user_id uuid,p_buyer_business_id uuid,p_supplier_organization_id uuid)
RETURNS TABLE(id text,provider text,provider_mandate_id text,buyer_business_id text,buyer_user_id text,supplier_organization_id text,amount_ceiling_kobo bigint,state text,created_at timestamptz)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT m.id::text,m.provider,m.provider_mandate_id,m.buyer_subject_id::text,b.owner_user_id::text,m.supplier_organization_id::text,m.amount_ceiling_kobo,m.state,m.created_at FROM app.payment_mandates m JOIN app.businesses b ON b.id=m.buyer_subject_id WHERE m.id=p_mandate_id AND m.buyer_subject_type='business' AND m.buyer_subject_id=p_buyer_business_id AND b.owner_user_id=p_buyer_user_id AND m.supplier_organization_id=p_supplier_organization_id AND app.branch_customer_access(p_supplier_organization_id,p_buyer_business_id) AND ((current_setting('role',true)='kredit_worker' OR (current_setting('role',true)='none' AND pg_has_role(session_user,'kredit_worker','member') AND NOT pg_has_role(session_user,'kredit_app','member'))) OR app.purchasing_authority_current(p_buyer_business_id,p_supplier_organization_id));
$$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Snapshot authority requires forward recovery'; END $$;
-- +goose StatementEnd
