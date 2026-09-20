-- +goose Up
-- Serialize current-owner writes against membership removal and account suspension.
-- +goose StatementBegin
CREATE FUNCTION app.lock_purchasing_authority(profile_id uuid,supplier_id uuid DEFAULT NULL) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
DECLARE workspace uuid; actor uuid;
BEGIN
 actor:=app.current_user_id();
 SELECT organization_id INTO workspace FROM app.businesses WHERE id=profile_id AND owner_user_id=actor;
 IF workspace IS NULL THEN RETURN; END IF;
 PERFORM 1 FROM app.users WHERE id=actor AND status='active' FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'current business authority required' USING ERRCODE='42501'; END IF;
 PERFORM 1 FROM app.memberships WHERE organization_id=workspace AND user_id=actor AND role='owner' AND status='active' FOR SHARE;
 IF FOUND THEN RETURN; END IF;
 IF supplier_id=app.current_organization_id() THEN
  PERFORM 1 FROM app.memberships WHERE organization_id=supplier_id AND user_id=actor AND status='active' FOR SHARE;
  IF FOUND THEN RETURN; END IF;
 END IF;
 RAISE EXCEPTION 'current business authority required' USING ERRCODE='42501';
END $$;
CREATE FUNCTION app.lock_request_authority(request_id uuid) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
DECLARE profile uuid; supplier uuid;
BEGIN
 SELECT buyer_business_id,supplier_organization_id INTO profile,supplier FROM app.credit_requests WHERE id=request_id;
 IF FOUND THEN PERFORM app.lock_purchasing_authority(profile,supplier); END IF;
END $$;
CREATE FUNCTION app.lock_obligation_authority(obligation_id uuid) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
DECLARE profile uuid; supplier uuid;
BEGIN
 SELECT buyer_business_id,supplier_organization_id INTO profile,supplier FROM app.obligations WHERE id=obligation_id;
 IF FOUND THEN PERFORM app.lock_purchasing_authority(profile,supplier); END IF;
END $$;
CREATE FUNCTION app.guard_current_purchasing_authority() RETURNS trigger
LANGUAGE plpgsql SECURITY INVOKER SET search_path=pg_catalog,app AS $$
DECLARE value jsonb;
BEGIN
 -- Existing role-scoped worker policies remain responsible for worker access.
 IF current_user='kredit_worker' THEN RETURN NEW; END IF;
 value:=to_jsonb(NEW);
 IF TG_ARGV[0]='root' THEN
  PERFORM app.lock_purchasing_authority((value->>'buyer_business_id')::uuid,(value->>'supplier_organization_id')::uuid);
 ELSIF TG_ARGV[0]='request' THEN
  PERFORM app.lock_request_authority((value->>'credit_request_id')::uuid);
 ELSIF TG_ARGV[0]='obligation' THEN
  PERFORM app.lock_obligation_authority((value->>'obligation_id')::uuid);
 ELSIF TG_ARGV[0]='mandate' AND value->>'buyer_subject_type'='business' THEN
  PERFORM app.lock_purchasing_authority((value->>'buyer_subject_id')::uuid,(value->>'supplier_organization_id')::uuid);
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.lock_purchasing_authority(uuid,uuid),app.lock_request_authority(uuid),app.lock_obligation_authority(uuid),app.guard_current_purchasing_authority() FROM PUBLIC;
-- +goose StatementBegin
DO $$ DECLARE role_name text; table_name text; BEGIN
 FOREACH role_name IN ARRAY ARRAY['kredit_app','kredit_worker'] LOOP
  IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname=role_name) THEN
   EXECUTE format('GRANT EXECUTE ON FUNCTION app.lock_purchasing_authority(uuid,uuid),app.lock_request_authority(uuid),app.lock_obligation_authority(uuid),app.guard_current_purchasing_authority() TO %I',role_name);
  END IF;
 END LOOP;
 FOREACH table_name IN ARRAY ARRAY['credit_requests','obligations','trade_lines','trade_relationships'] LOOP
  EXECUTE format('CREATE TRIGGER current_purchasing_authority BEFORE INSERT OR UPDATE ON app.%I FOR EACH ROW EXECUTE FUNCTION app.guard_current_purchasing_authority(''root'')',table_name);
 END LOOP;
 FOREACH table_name IN ARRAY ARRAY['credit_aggregate_snapshots','agreement_versions','agreement_acceptances','goods_releases','receipt_confirmations','mandates','system_acceptances'] LOOP
  EXECUTE format('CREATE TRIGGER current_purchasing_authority BEFORE INSERT OR UPDATE ON app.%I FOR EACH ROW EXECUTE FUNCTION app.guard_current_purchasing_authority(''request'')',table_name);
 END LOOP;
 FOREACH table_name IN ARRAY ARRAY['payments','payment_claims','payment_allocations','fees','disputes','repayment_schedules','collection_aggregate_snapshots','collection_attempt_index','collection_attempts','collection_events','collection_reservations','collection_settlement_routes'] LOOP
  EXECUTE format('CREATE TRIGGER current_purchasing_authority BEFORE INSERT OR UPDATE ON app.%I FOR EACH ROW EXECUTE FUNCTION app.guard_current_purchasing_authority(''obligation'')',table_name);
 END LOOP;
END $$;
-- +goose StatementEnd
CREATE TRIGGER current_purchasing_authority BEFORE INSERT OR UPDATE ON app.payment_mandates FOR EACH ROW EXECUTE FUNCTION app.guard_current_purchasing_authority('mandate');

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Business authority locking requires forward recovery'; END $$;
-- +goose StatementEnd
