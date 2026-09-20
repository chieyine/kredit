-- +goose Up
-- This is a restrictive revocation floor, not a grant of purchasing authority.
-- Existing tenant, ownership and operator policies still have to authorize a row.
-- Historical owners remain contractual contacts, not permanently authorized actors.
-- +goose StatementBegin
CREATE FUNCTION app.purchasing_authority_current(profile_id uuid,supplier_id uuid DEFAULT NULL) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT NOT EXISTS (
  SELECT 1 FROM app.businesses b
  WHERE b.id=profile_id AND b.organization_id IS NOT NULL AND b.owner_user_id=app.current_user_id()
  AND NOT EXISTS (
   SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id
   WHERE m.organization_id=b.organization_id AND m.user_id=app.current_user_id()
   AND m.role='owner' AND m.status='active' AND u.status='active'
  )
  AND NOT EXISTS (
   SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id
   WHERE m.organization_id=supplier_id AND supplier_id=app.current_organization_id()
   AND m.user_id=app.current_user_id() AND m.status='active' AND u.status='active'
  )
 );
$$;
CREATE FUNCTION app.request_authority_current(request_id uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT COALESCE((SELECT app.purchasing_authority_current(c.buyer_business_id,c.supplier_organization_id)
 FROM app.credit_requests c WHERE c.id=request_id),false);
$$;
CREATE FUNCTION app.obligation_authority_current(obligation_id uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT COALESCE((SELECT app.purchasing_authority_current(o.buyer_business_id,o.supplier_organization_id)
 FROM app.obligations o WHERE o.id=obligation_id),false);
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.purchasing_authority_current(uuid,uuid),app.request_authority_current(uuid),app.obligation_authority_current(uuid) FROM PUBLIC;
-- +goose StatementBegin
DO $$ DECLARE role_name text; BEGIN
 FOREACH role_name IN ARRAY ARRAY['kredit_app','kredit_worker'] LOOP
  IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname=role_name) THEN
   EXECUTE format('GRANT EXECUTE ON FUNCTION app.purchasing_authority_current(uuid,uuid),app.request_authority_current(uuid),app.obligation_authority_current(uuid) TO %I',role_name);
  END IF;
 END LOOP;
END $$;
-- +goose StatementEnd
CREATE POLICY current_business_owner ON app.businesses AS RESTRICTIVE
USING (current_user='kredit_worker' OR app.purchasing_authority_current(id));
CREATE POLICY current_business_representative ON app.business_representatives AS RESTRICTIVE
USING (current_user='kredit_worker' OR app.purchasing_authority_current(business_id));
-- +goose StatementBegin
DO $$ DECLARE table_name text; BEGIN
 FOREACH table_name IN ARRAY ARRAY['credit_requests','obligations','trade_lines','trade_relationships'] LOOP
  EXECUTE format('CREATE POLICY current_purchasing_authority ON app.%I AS RESTRICTIVE USING (current_user=''kredit_worker'' OR app.purchasing_authority_current(buyer_business_id,supplier_organization_id))',table_name);
 END LOOP;
 FOREACH table_name IN ARRAY ARRAY['credit_aggregate_snapshots','agreement_versions','agreement_acceptances','goods_releases','receipt_confirmations','mandates','system_acceptances'] LOOP
  EXECUTE format('CREATE POLICY current_purchasing_authority ON app.%I AS RESTRICTIVE USING (current_user=''kredit_worker'' OR app.request_authority_current(credit_request_id::uuid))',table_name);
 END LOOP;
 FOREACH table_name IN ARRAY ARRAY['payments','payment_claims','payment_allocations','fees','disputes','repayment_schedules','collection_aggregate_snapshots','collection_attempt_index','collection_attempts','collection_events','collection_reservations','collection_settlement_routes'] LOOP
  EXECUTE format('CREATE POLICY current_purchasing_authority ON app.%I AS RESTRICTIVE USING (current_user=''kredit_worker'' OR app.obligation_authority_current(obligation_id::uuid))',table_name);
 END LOOP;
END $$;
-- +goose StatementEnd
CREATE POLICY current_business_mandate_owner ON app.payment_mandates AS RESTRICTIVE
USING (current_user='kredit_worker' OR buyer_subject_type<>'business' OR app.purchasing_authority_current(buyer_subject_id,supplier_organization_id));

-- +goose Down
-- Revocation must not be undone by a binary rollback.
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Business authority requires forward recovery'; END $$;
-- +goose StatementEnd
