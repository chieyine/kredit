-- +goose Up
-- Recheck eligibility at the reservation commit boundary under the obligation lock.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_collection_reservation() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE provider_name text; insufficient_count bigint; other_count bigint; due_now NUMERIC; blocked NUMERIC; claimed NUMERIC; lifecycle TEXT; buyer_user UUID; remaining BIGINT; ceiling BIGINT; used BIGINT; held BIGINT; mid UUID; mstate TEXT; mstart TIMESTAMPTZ; mend TIMESTAMPTZ; buyer UUID; supplier UUID; mbuyer UUID; msupplier UUID;
BEGIN
 IF EXISTS(SELECT 1 FROM app.collection_reservations WHERE id=NEW.id) THEN RETURN NEW; END IF;
 SELECT o.outstanding_kobo,c.mandate_id,c.buyer_business_id,c.supplier_organization_id,c.buyer_user_id,o.lifecycle_status INTO remaining,mid,buyer,supplier,buyer_user,lifecycle FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id WHERE o.id=NEW.obligation_id FOR UPDATE OF o;
 IF remaining IS NULL OR lifecycle <> 'ACTIVE' THEN RAISE EXCEPTION 'obligation is not active'; END IF;
 SELECT COALESCE(SUM(GREATEST(i.principal_due_kobo-i.allocated_kobo,0)),0) INTO due_now
 FROM app.schedule_items i JOIN app.repayment_schedules s ON s.id=i.schedule_id
 WHERE s.obligation_id=NEW.obligation_id AND i.state NOT IN ('PAID','CANCELLED') AND i.collection_at<=now();
 SELECT COALESCE(SUM(CASE WHEN collection_effect='FULL_BLOCK' THEN remaining WHEN collection_effect='CONTESTED_ONLY' THEN remaining_disputed_kobo ELSE 0 END),0) INTO blocked
 FROM app.disputes WHERE obligation_id=NEW.obligation_id AND state IN ('OPEN','UNDER_REVIEW','PARTIALLY_RESOLVED');
 SELECT COALESCE(SUM(amount_kobo),0) INTO claimed FROM app.payment_claims WHERE obligation_id=NEW.obligation_id AND state='pending' AND hold_expires_at>now();
 IF NEW.reserved_amount_kobo > GREATEST(0,LEAST(remaining,due_now)-blocked-claimed) THEN RAISE EXCEPTION 'collection exceeds current due and undisputed balance'; END IF;
 IF EXISTS(SELECT 1 FROM app.risk_holds WHERE ((target_type='buyer' AND target_id=buyer_user) OR (target_type='supplier' AND target_id=supplier)) AND scope IN ('collection','all_sensitive') AND lifted_at IS NULL AND expires_at>now()) THEN RAISE EXCEPTION 'collection is on hold'; END IF;
 IF NEW.reserved_amount_kobo > remaining THEN RAISE EXCEPTION 'collection exceeds authoritative outstanding balance'; END IF;
 IF EXISTS(SELECT 1 FROM app.collection_reservations WHERE obligation_id=NEW.obligation_id AND state IN ('PROCESSING','COMPLETED')) THEN RAISE EXCEPTION 'collection already reserved'; END IF;
 IF mid IS NOT NULL THEN
   SELECT amount_ceiling_kobo,state,starts_at,ends_at,buyer_subject_id,supplier_organization_id INTO ceiling,mstate,mstart,mend,mbuyer,msupplier FROM app.payment_mandates WHERE id=mid FOR UPDATE;
   IF ceiling IS NULL OR mbuyer<>buyer OR (msupplier IS NOT NULL AND msupplier<>supplier) THEN RAISE EXCEPTION 'mandate ownership mismatch'; END IF;
   IF mstate <> 'active' OR mstart>now() OR mend<=now() THEN RAISE EXCEPTION 'mandate is not active in validity period'; END IF;
   SELECT COALESCE(SUM(a.succeeded_amount_kobo),0) INTO used FROM app.collection_attempts a JOIN app.collection_reservations r ON r.id=a.reservation_id WHERE r.mandate_id=mid;
   SELECT COALESCE(SUM(reserved_amount_kobo),0) INTO held FROM app.collection_reservations WHERE mandate_id=mid AND state IN ('PROCESSING','COMPLETED');
   IF NEW.reserved_amount_kobo > ceiling-used-held THEN RAISE EXCEPTION 'mandate capacity exhausted'; END IF;
   SELECT provider INTO provider_name FROM app.payment_mandates WHERE id=mid;
   IF provider_name='mono-sweep' THEN
     IF NEW.reserved_amount_kobo NOT BETWEEN 20000 AND 2500000000 THEN RAISE EXCEPTION 'Mono debit amount is outside the supported range'; END IF;
     SELECT count(*) FILTER(WHERE a.failure_code='51'),count(*) FILTER(WHERE COALESCE(a.failure_code,'')<>'51')
       INTO insufficient_count,other_count FROM app.collection_attempts a JOIN app.collection_reservations r ON r.id=a.reservation_id
       WHERE r.mandate_id=mid AND a.state='FAILED' AND (a.requested_at AT TIME ZONE 'Africa/Lagos')::date=(now() AT TIME ZONE 'Africa/Lagos')::date;
     IF insufficient_count>=5 OR other_count>=10 THEN RAISE EXCEPTION 'Mono daily failed-attempt limit reached'; END IF;
     IF EXISTS(SELECT 1 FROM app.collection_reservations r WHERE r.mandate_id=mid AND r.state IN ('PROCESSING','COMPLETED')) THEN RAISE EXCEPTION 'Reconcile the existing mandate debit before collecting again'; END IF;
   END IF;
   NEW.mandate_id := mid;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd


-- Narrow identifier discovery after current authority has been installed.
-- +goose StatementBegin
CREATE FUNCTION app.financial_change_identity(p_reference text,p_change boolean DEFAULT false)
RETURNS TABLE(obligation_id text,organization_id text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT o.id::text,o.supplier_organization_id::text FROM app.obligations o
 JOIN app.credit_requests c ON c.id=o.credit_request_id
 WHERE ((NOT p_change AND (o.id::text=p_reference OR c.id::text=p_reference)) OR
  (p_change AND EXISTS(SELECT 1 FROM app.admin_change_requests a WHERE a.id::text=p_reference AND a.obligation_id=o.id)))
 AND (app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','finance_operator','approver']) OR
  (p_change AND c.buyer_user_id=app.current_user_id())) LIMIT 1
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.financial_change_identity(text,boolean) FROM PUBLIC;

-- Only the locked owner lifecycle may inspect another user's transfer eligibility.
-- +goose StatementBegin
CREATE FUNCTION app.lock_transfer_recipient(p_target uuid) RETURNS boolean
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE eligible boolean;
BEGIN
 PERFORM pg_advisory_xact_lock(746219830045::bigint);
 IF NOT app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']) OR p_target=app.current_user_id() THEN RAISE EXCEPTION 'current owner authority required'; END IF;
 SELECT status='active' INTO eligible FROM app.users WHERE id=p_target FOR SHARE;
 RETURN COALESCE(eligible,false);
END $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.lock_transfer_recipient(uuid) FROM PUBLIC;

-- A controlled global projection for reconciliation. It exposes differences,
-- not unrestricted access to source financial tables.
-- +goose StatementBegin
CREATE FUNCTION app.financial_review_differences(p_lock boolean DEFAULT false)
RETURNS TABLE(kind text,target_id text,expected numeric,actual numeric)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
BEGIN
 IF NOT (current_setting('role',true)='kredit_worker' OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer'])) THEN RAISE EXCEPTION 'financial review authority required'; END IF;
 IF p_lock THEN
  LOCK TABLE app.obligations,app.credit_requests,app.payments,app.collection_attempts,app.settlement_events,app.repayment_schedules,app.schedule_items,app.disputes,ledger.transactions,ledger.postings,ledger.accounts IN SHARE MODE;
 END IF;
 RETURN QUERY SELECT d.kind,d.target_id,d.expected::numeric,d.actual::numeric FROM app.financial_discrepancies d;
END $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.financial_review_differences(boolean) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
  GRANT EXECUTE ON FUNCTION app.financial_change_identity(text,boolean),app.lock_transfer_recipient(uuid),app.financial_review_differences(boolean) TO kredit_app;
 END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN GRANT EXECUTE ON FUNCTION app.financial_review_differences(boolean) TO kredit_worker; END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Financial boundary repairs require forward recovery'; END $$;
-- +goose StatementEnd
