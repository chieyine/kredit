-- +goose Up
-- Supplier branch boundaries are independent of buying authority and job roles.
CREATE TABLE app.member_branch_scopes (
 organization_id uuid NOT NULL REFERENCES app.organizations(id),
 user_id uuid NOT NULL REFERENCES app.users(id),
 membership_id uuid NOT NULL REFERENCES app.memberships(id),
 membership_version bigint NOT NULL,
 mode text NOT NULL CHECK(mode IN ('all','branches')),
 branch_ids uuid[] NOT NULL DEFAULT '{}',
 version bigint NOT NULL DEFAULT 1 CHECK(version>0),
 updated_by uuid NOT NULL REFERENCES app.users(id),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(organization_id,user_id),
 CHECK(cardinality(branch_ids)<=100 AND (mode='branches' OR cardinality(branch_ids)=0))
);
CREATE TABLE app.member_branch_scope_history (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 organization_id uuid NOT NULL REFERENCES app.organizations(id),
 user_id uuid NOT NULL REFERENCES app.users(id),
 version bigint NOT NULL,
 actor_id uuid NOT NULL REFERENCES app.users(id),
 recorded_at timestamptz NOT NULL DEFAULT now(),
 evidence jsonb NOT NULL,
 UNIQUE(organization_id,user_id,version)
);
-- +goose StatementBegin
CREATE FUNCTION app.branch_scope_all(p_org uuid) RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT CASE WHEN (current_setting('role',true)='kredit_worker' OR (current_setting('role',true)='none' AND pg_has_role(session_user,'kredit_worker','member') AND NOT pg_has_role(session_user,'kredit_app','member'))) THEN true
 WHEN app.current_user_id() IS NULL THEN NOT EXISTS(SELECT 1 FROM app.member_branch_scopes WHERE organization_id=p_org AND mode='branches')
 ELSE EXISTS(SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id LEFT JOIN app.member_branch_scopes s ON s.organization_id=m.organization_id AND s.user_id=m.user_id
 WHERE m.organization_id=p_org AND m.user_id=app.current_user_id() AND m.status='active' AND u.status='active' AND o.status<>'suspended'
 AND (m.role IN ('owner','administrator') OR s.user_id IS NULL OR (s.mode='all' AND s.membership_id=m.id AND s.membership_version=m.purchasing_authority_version))) END;
$$;
CREATE FUNCTION app.branch_customer_access(p_org uuid,p_business uuid) RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT app.branch_scope_all(p_org) OR app.can_purchase(p_business) OR EXISTS(
 SELECT 1 FROM app.member_branch_scopes s JOIN app.memberships m ON m.id=s.membership_id AND m.purchasing_authority_version=s.membership_version JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id
 JOIN app.partner_assignments a ON a.organization_id=s.organization_id AND a.buyer_business_id=p_business JOIN app.business_branches b ON b.id=a.branch_id AND b.organization_id=a.organization_id
 WHERE s.organization_id=p_org AND s.user_id=app.current_user_id() AND m.user_id=s.user_id AND m.organization_id=s.organization_id AND m.status='active' AND u.status='active' AND o.status<>'suspended' AND s.mode='branches' AND a.branch_id=ANY(s.branch_ids) AND b.active);
$$;
CREATE FUNCTION app.branch_credit_access(p_id uuid) RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT COALESCE((SELECT app.branch_customer_access(supplier_organization_id,buyer_business_id) FROM app.credit_requests WHERE id=p_id),false);
$$;
CREATE FUNCTION app.branch_obligation_access(p_id uuid) RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT COALESCE((SELECT app.branch_customer_access(supplier_organization_id,buyer_business_id) FROM app.obligations WHERE id=p_id),false);
$$;
CREATE FUNCTION app.branch_line_access(p_id uuid) RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT COALESCE((SELECT app.branch_customer_access(supplier_organization_id,buyer_business_id) FROM app.trade_lines WHERE id=p_id),false);
$$;
CREATE FUNCTION app.branch_drawdown_access(p_id uuid) RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT COALESCE((SELECT app.branch_line_access(trade_line_id) FROM app.drawdowns WHERE id=p_id),false);
$$;
CREATE FUNCTION app.record_branch_scope() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE m app.memberships; bid uuid;
BEGIN
 IF NEW.updated_by IS DISTINCT FROM app.current_user_id() THEN RAISE EXCEPTION 'current owner required' USING ERRCODE='42501'; END IF;
 PERFORM 1 FROM app.memberships a JOIN app.users u ON u.id=a.user_id JOIN app.organizations o ON o.id=a.organization_id WHERE a.organization_id=NEW.organization_id AND a.user_id=NEW.updated_by AND a.role='owner' AND a.status='active' AND u.status='active' AND o.status<>'suspended' FOR SHARE OF a,u,o;
 IF NOT FOUND THEN RAISE EXCEPTION 'current owner required' USING ERRCODE='42501'; END IF;
 SELECT * INTO m FROM app.memberships WHERE organization_id=NEW.organization_id AND user_id=NEW.user_id AND status='active' FOR UPDATE;
 IF NOT FOUND OR m.role IN ('owner','administrator') THEN RAISE EXCEPTION 'choose active non-administrative staff' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND (NEW.organization_id<>OLD.organization_id OR NEW.user_id<>OLD.user_id OR NEW.version<>OLD.version+1) THEN RAISE EXCEPTION 'next scope version required' USING ERRCODE='23514'; END IF;
 IF TG_OP='INSERT' AND NEW.version<>1 THEN RAISE EXCEPTION 'initial scope version required' USING ERRCODE='23514'; END IF;
 IF EXISTS(SELECT 1 FROM unnest(NEW.branch_ids) x GROUP BY x HAVING count(*)>1) THEN RAISE EXCEPTION 'duplicate branches' USING ERRCODE='23514'; END IF;
 FOREACH bid IN ARRAY NEW.branch_ids LOOP
  PERFORM 1 FROM app.business_branches WHERE organization_id=NEW.organization_id AND id=bid AND active FOR SHARE;
  IF NOT FOUND THEN RAISE EXCEPTION 'choose active branches in this business' USING ERRCODE='23514'; END IF;
 END LOOP;
 NEW.membership_id:=m.id; NEW.membership_version:=m.purchasing_authority_version;
 INSERT INTO app.member_branch_scope_history(organization_id,user_id,version,actor_id,evidence) VALUES(NEW.organization_id,NEW.user_id,NEW.version,NEW.updated_by,to_jsonb(NEW));
 RETURN NEW;
END $$;
CREATE TRIGGER member_branch_scope_history BEFORE INSERT OR UPDATE ON app.member_branch_scopes FOR EACH ROW EXECUTE FUNCTION app.record_branch_scope();
ALTER TABLE app.member_branch_scopes ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.member_branch_scopes FORCE ROW LEVEL SECURITY;
ALTER TABLE app.member_branch_scope_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.member_branch_scope_history FORCE ROW LEVEL SECURITY;
CREATE POLICY branch_scope_read ON app.member_branch_scopes FOR SELECT USING(organization_id=app.current_organization_id() AND (user_id=app.current_user_id() OR app.branch_scope_all(organization_id)));
CREATE POLICY branch_scope_current ON app.member_branch_scopes AS RESTRICTIVE FOR ALL USING(EXISTS(SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id WHERE m.organization_id=member_branch_scopes.organization_id AND m.user_id=app.current_user_id() AND m.status='active' AND u.status='active' AND o.status<>'suspended'));
CREATE POLICY branch_scope_write ON app.member_branch_scopes FOR ALL USING(organization_id=app.current_organization_id() AND EXISTS(SELECT 1 FROM app.memberships WHERE organization_id=member_branch_scopes.organization_id AND user_id=app.current_user_id() AND role='owner' AND status='active')) WITH CHECK(organization_id=app.current_organization_id() AND updated_by=app.current_user_id());
CREATE POLICY branch_scope_history_read ON app.member_branch_scope_history FOR SELECT USING(organization_id=app.current_organization_id() AND app.branch_scope_all(organization_id));
CREATE POLICY branch_boundary ON app.credit_requests AS RESTRICTIVE FOR ALL USING(app.branch_customer_access(supplier_organization_id,buyer_business_id));
CREATE POLICY branch_boundary ON app.obligations AS RESTRICTIVE FOR ALL USING(app.branch_customer_access(supplier_organization_id,buyer_business_id));
CREATE POLICY branch_boundary ON app.trade_lines AS RESTRICTIVE FOR ALL USING(app.branch_customer_access(supplier_organization_id,buyer_business_id));
CREATE POLICY branch_boundary ON app.trade_relationships AS RESTRICTIVE FOR ALL USING(app.branch_customer_access(supplier_organization_id,buyer_business_id));
CREATE POLICY branch_boundary ON app.partner_assignments AS RESTRICTIVE FOR ALL USING(app.branch_customer_access(organization_id,buyer_business_id));
CREATE POLICY branch_boundary ON app.credit_aggregate_snapshots AS RESTRICTIVE FOR ALL USING(app.branch_credit_access(credit_request_id::uuid));
CREATE POLICY branch_boundary ON app.agreement_versions AS RESTRICTIVE FOR ALL USING(app.branch_credit_access(credit_request_id::uuid));
CREATE POLICY branch_boundary ON app.agreement_acceptances AS RESTRICTIVE FOR ALL USING(app.branch_credit_access(credit_request_id::uuid));
CREATE POLICY branch_boundary ON app.goods_releases AS RESTRICTIVE FOR ALL USING(app.branch_credit_access(credit_request_id::uuid));
CREATE POLICY branch_boundary ON app.receipt_confirmations AS RESTRICTIVE FOR ALL USING(app.branch_credit_access(credit_request_id::uuid));
CREATE POLICY branch_boundary ON app.mandates AS RESTRICTIVE FOR ALL USING(app.branch_credit_access(credit_request_id::uuid));
CREATE POLICY branch_boundary ON app.system_acceptances AS RESTRICTIVE FOR ALL USING(app.branch_credit_access(credit_request_id::uuid));
CREATE POLICY branch_boundary ON app.credit_offer_approvals AS RESTRICTIVE FOR ALL USING(app.branch_credit_access(credit_request_id::uuid));
CREATE POLICY branch_boundary ON app.payments AS RESTRICTIVE FOR ALL USING(app.branch_obligation_access(obligation_id::uuid));
CREATE POLICY branch_boundary ON app.fees AS RESTRICTIVE FOR ALL USING(app.branch_obligation_access(obligation_id::uuid));
CREATE POLICY branch_boundary ON app.disputes AS RESTRICTIVE FOR ALL USING(app.branch_obligation_access(obligation_id::uuid));
CREATE POLICY branch_boundary ON app.payment_claims AS RESTRICTIVE FOR ALL USING(app.branch_obligation_access(obligation_id::uuid));
CREATE POLICY branch_boundary ON app.payment_allocations AS RESTRICTIVE FOR ALL USING(app.branch_obligation_access(obligation_id::uuid));
CREATE POLICY branch_boundary ON app.repayment_schedules AS RESTRICTIVE FOR ALL USING(app.branch_obligation_access(obligation_id::uuid));
CREATE POLICY branch_boundary ON app.collection_aggregate_snapshots AS RESTRICTIVE FOR ALL USING(app.branch_obligation_access(obligation_id::uuid));
CREATE POLICY branch_boundary ON app.collection_attempt_index AS RESTRICTIVE FOR ALL USING(app.branch_obligation_access(obligation_id::uuid));
CREATE POLICY branch_boundary ON app.collection_attempts AS RESTRICTIVE FOR ALL USING(app.branch_obligation_access(obligation_id::uuid));
CREATE POLICY branch_boundary ON app.collection_events AS RESTRICTIVE FOR ALL USING(app.branch_obligation_access(obligation_id::uuid));
CREATE POLICY branch_boundary ON app.collection_reservations AS RESTRICTIVE FOR ALL USING(app.branch_obligation_access(obligation_id::uuid));
CREATE POLICY branch_boundary ON app.collection_settlement_routes AS RESTRICTIVE FOR ALL USING(app.branch_obligation_access(obligation_id::uuid));
CREATE POLICY branch_boundary ON app.admin_change_requests AS RESTRICTIVE FOR ALL USING(app.branch_obligation_access(obligation_id::uuid));
CREATE POLICY branch_boundary ON app.schedule_items AS RESTRICTIVE FOR ALL USING(EXISTS(SELECT 1 FROM app.repayment_schedules r WHERE r.id=schedule_items.schedule_id AND app.branch_obligation_access(r.obligation_id)));
CREATE POLICY branch_boundary ON app.drawdowns AS RESTRICTIVE FOR ALL USING(app.branch_line_access(trade_line_id));
CREATE POLICY branch_boundary ON app.drawdown_reservations AS RESTRICTIVE FOR ALL USING(app.branch_line_access(trade_line_id));
CREATE POLICY branch_boundary ON app.drawdown_receipt_disputes AS RESTRICTIVE FOR ALL USING(app.branch_drawdown_access(drawdown_id));
CREATE POLICY branch_boundary ON app.business_branches AS RESTRICTIVE FOR SELECT USING(app.branch_scope_all(organization_id) OR EXISTS(SELECT 1 FROM app.member_branch_scopes s JOIN app.memberships m ON m.id=s.membership_id AND m.purchasing_authority_version=s.membership_version WHERE s.organization_id=business_branches.organization_id AND s.user_id=app.current_user_id() AND m.status='active' AND business_branches.id=ANY(s.branch_ids)));
CREATE POLICY branch_boundary ON app.network_operation_history AS RESTRICTIVE FOR SELECT USING(app.branch_scope_all(organization_id));
CREATE POLICY branch_boundary ON app.businesses AS RESTRICTIVE FOR ALL USING(app.current_organization_id() IS NULL OR organization_id=app.current_organization_id() OR app.branch_customer_access(app.current_organization_id(),id));
CREATE POLICY branch_boundary ON app.buyer_invitations AS RESTRICTIVE FOR ALL USING(app.branch_scope_all(organization_id));
CREATE POLICY branch_boundary ON app.distributor_import_batches AS RESTRICTIVE FOR ALL USING(app.branch_scope_all(organization_id));
CREATE POLICY branch_boundary ON app.audit_events AS RESTRICTIVE FOR ALL USING(app.branch_scope_all(organization_id));
CREATE POLICY branch_boundary ON app.documents AS RESTRICTIVE FOR ALL USING(app.branch_scope_all(organization_id));
CREATE POLICY branch_boundary ON app.fee_authorizations AS RESTRICTIVE FOR ALL USING(app.branch_scope_all(organization_id));
CREATE POLICY branch_boundary ON app.fee_bank_receipts AS RESTRICTIVE FOR ALL USING(app.branch_scope_all(organization_id));
CREATE POLICY branch_boundary ON app.fee_debits AS RESTRICTIVE FOR ALL USING(app.branch_scope_all(organization_id));
CREATE POLICY branch_boundary ON app.fee_invoice_lines AS RESTRICTIVE FOR ALL USING(app.branch_scope_all(organization_id));
CREATE POLICY branch_boundary ON app.fee_invoice_receipts AS RESTRICTIVE FOR ALL USING(app.branch_scope_all(organization_id));
CREATE POLICY branch_boundary ON app.fee_invoices AS RESTRICTIVE FOR ALL USING(app.branch_scope_all(organization_id));
CREATE POLICY branch_boundary ON app.invoice_billing_approvals AS RESTRICTIVE FOR ALL USING(app.branch_scope_all(organization_id));
CREATE OR REPLACE FUNCTION app.credit_snapshot_by_id(p_request_id text) RETURNS jsonb LANGUAGE sql SECURITY DEFINER STABLE SET search_path=pg_catalog,app AS $$
 SELECT aggregate FROM app.credit_aggregate_snapshots WHERE credit_request_id=p_request_id AND app.branch_credit_access(credit_request_id::uuid) AND
 (supplier_organization_id=NULLIF(current_setting('app.current_organization_id',true),'') OR buyer_user_id=NULLIF(current_setting('app.current_user_id',true),'') OR app.purchase_request_read(credit_request_id::uuid));
$$;
CREATE OR REPLACE FUNCTION app.credit_snapshot_by_obligation(p_obligation_id text) RETURNS TABLE(credit_request_id text,aggregate jsonb) LANGUAGE sql SECURITY DEFINER STABLE SET search_path=pg_catalog,app AS $$
 SELECT credit_request_id,aggregate FROM app.credit_aggregate_snapshots WHERE aggregate->'obligation'->>'id'=p_obligation_id AND app.branch_credit_access(credit_request_id::uuid) AND
 (supplier_organization_id=NULLIF(current_setting('app.current_organization_id',true),'') OR buyer_user_id=NULLIF(current_setting('app.current_user_id',true),'') OR app.purchase_request_read(credit_request_id::uuid)) LIMIT 1;
$$;
CREATE OR REPLACE FUNCTION app.supplier_customers(p_organization_id UUID)
RETURNS TABLE (
    buyer_user_id UUID,
    buyer_business_id UUID,
    legal_name TEXT,
    trading_name TEXT,
    industry TEXT,
    status TEXT
)
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = app, pg_catalog
AS $$
    SELECT b.owner_user_id, b.id, b.legal_name, b.trading_name, b.industry, b.status
    FROM app.trade_relationships r
    JOIN app.businesses b ON b.id = r.buyer_business_id
    WHERE r.supplier_organization_id = p_organization_id
      AND r.status IN ('active', 'paused') AND app.branch_customer_access(p_organization_id,b.id)
      AND p_organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID
    ORDER BY b.legal_name
$$;


REVOKE ALL ON FUNCTION app.branch_scope_all(uuid),app.branch_customer_access(uuid,uuid),app.branch_credit_access(uuid),app.branch_obligation_access(uuid),app.branch_line_access(uuid),app.branch_drawdown_access(uuid),app.record_branch_scope() FROM PUBLIC;
DO $$ DECLARE r text; BEGIN
 FOREACH r IN ARRAY ARRAY['kredit_app','kredit_worker'] LOOP
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname=r) THEN EXECUTE format('GRANT EXECUTE ON FUNCTION app.branch_scope_all(uuid),app.branch_customer_access(uuid,uuid),app.branch_credit_access(uuid),app.branch_obligation_access(uuid),app.branch_line_access(uuid),app.branch_drawdown_access(uuid) TO %I',r); END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
 GRANT SELECT,INSERT,UPDATE ON app.member_branch_scopes TO kredit_app;
 GRANT SELECT ON app.member_branch_scope_history TO kredit_app;
 END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Branch boundaries require forward recovery'; END $$;
-- +goose StatementEnd
