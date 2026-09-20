-- +goose Up
-- +goose StatementBegin
DROP POLICY branch_boundary ON app.audit_events;
CREATE POLICY branch_boundary ON app.audit_events AS RESTRICTIVE FOR SELECT USING(app.branch_scope_all(organization_id));
CREATE FUNCTION app.lock_branch_customer(p_org uuid,p_business uuid) RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
BEGIN
 -- The membership row serializes scope creation even when no scope row exists yet.
 PERFORM 1 FROM app.memberships WHERE organization_id=p_org AND user_id=app.current_user_id() FOR SHARE;
 PERFORM 1 FROM app.partner_assignments a JOIN app.business_branches b ON b.id=a.branch_id AND b.organization_id=a.organization_id WHERE a.organization_id=p_org AND a.buyer_business_id=p_business FOR SHARE OF a,b;
 IF NOT app.branch_customer_access(p_org,p_business) THEN RAISE EXCEPTION 'customer is outside current branch authority' USING ERRCODE='42501'; END IF;
END $$;
CREATE FUNCTION app.guard_branch_financial_write() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE value jsonb:=to_jsonb(NEW); org uuid; business uuid; parent_id uuid;
BEGIN
 IF TG_TABLE_NAME IN ('credit_requests','obligations','trade_lines','trade_relationships') THEN
  org:=(value->>'supplier_organization_id')::uuid;business:=(value->>'buyer_business_id')::uuid;
 ELSIF value->>'obligation_id' IS NOT NULL THEN
  SELECT supplier_organization_id,buyer_business_id INTO org,business FROM app.obligations WHERE id=(value->>'obligation_id')::uuid;
 ELSIF value->>'credit_request_id' IS NOT NULL THEN
  SELECT supplier_organization_id,buyer_business_id INTO org,business FROM app.credit_requests WHERE id=(value->>'credit_request_id')::uuid;
 ELSIF value->>'trade_line_id' IS NOT NULL THEN
  SELECT supplier_organization_id,buyer_business_id INTO org,business FROM app.trade_lines WHERE id=(value->>'trade_line_id')::uuid;
 ELSIF value->>'drawdown_id' IS NOT NULL THEN
  SELECT t.supplier_organization_id,t.buyer_business_id INTO org,business FROM app.trade_lines t JOIN app.drawdowns d ON d.trade_line_id=t.id WHERE d.id=(value->>'drawdown_id')::uuid;
 ELSIF value->>'schedule_id' IS NOT NULL THEN
  SELECT o.supplier_organization_id,o.buyer_business_id INTO org,business FROM app.obligations o JOIN app.repayment_schedules r ON r.obligation_id=o.id WHERE r.id=(value->>'schedule_id')::uuid;
 END IF;
 IF org IS NOT NULL THEN PERFORM app.lock_branch_customer(org,business); END IF;
 RETURN NEW;
END $$;
DO $$ DECLARE tab text; BEGIN
 FOREACH tab IN ARRAY ARRAY['credit_requests','obligations','trade_lines','trade_relationships','credit_aggregate_snapshots','agreement_versions','agreement_acceptances','goods_releases','receipt_confirmations','mandates','system_acceptances','credit_offer_approvals','payments','fees','disputes','payment_claims','payment_allocations','repayment_schedules','schedule_items','collection_aggregate_snapshots','collection_attempt_index','collection_attempts','collection_events','collection_reservations','collection_settlement_routes','admin_change_requests','drawdowns','drawdown_reservations','drawdown_receipt_disputes'] LOOP
  EXECUTE format('CREATE TRIGGER branch_financial_write BEFORE INSERT OR UPDATE ON app.%I FOR EACH ROW EXECUTE FUNCTION app.guard_branch_financial_write()',tab);
 END LOOP;
END $$;
REVOKE ALL ON FUNCTION app.lock_branch_customer(uuid,uuid),app.guard_branch_financial_write() FROM PUBLIC;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Branch write authority requires forward recovery'; END $$;
-- +goose StatementEnd
