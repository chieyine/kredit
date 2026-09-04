-- +goose Up
-- Recheck eligibility at the reservation commit boundary under the obligation lock.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_collection_reservation() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=app,pg_catalog AS $$
DECLARE due_now NUMERIC; blocked NUMERIC; claimed NUMERIC; lifecycle TEXT; buyer_user UUID; remaining BIGINT; ceiling BIGINT; used BIGINT; held BIGINT; mid UUID; mstate TEXT; mstart TIMESTAMPTZ; mend TIMESTAMPTZ; buyer UUID; supplier UUID; mbuyer UUID; msupplier UUID;
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
   NEW.mandate_id := mid;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_collection_reservation() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=app,pg_catalog AS $$
DECLARE remaining BIGINT; ceiling BIGINT; used BIGINT; held BIGINT; mid UUID; mstate TEXT; mstart TIMESTAMPTZ; mend TIMESTAMPTZ; buyer UUID; supplier UUID; mbuyer UUID; msupplier UUID;
BEGIN
 IF EXISTS(SELECT 1 FROM app.collection_reservations WHERE id=NEW.id) THEN RETURN NEW; END IF;
 SELECT o.outstanding_kobo,c.mandate_id,c.buyer_business_id,c.supplier_organization_id INTO remaining,mid,buyer,supplier FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id WHERE o.id=NEW.obligation_id FOR UPDATE OF o;
 IF NEW.reserved_amount_kobo > remaining THEN RAISE EXCEPTION 'collection exceeds authoritative outstanding balance'; END IF;
 IF EXISTS(SELECT 1 FROM app.collection_reservations WHERE obligation_id=NEW.obligation_id AND state IN ('PROCESSING','COMPLETED')) THEN RAISE EXCEPTION 'collection already reserved'; END IF;
 IF mid IS NOT NULL THEN
   SELECT amount_ceiling_kobo,state,starts_at,ends_at,buyer_subject_id,supplier_organization_id INTO ceiling,mstate,mstart,mend,mbuyer,msupplier FROM app.payment_mandates WHERE id=mid FOR UPDATE;
   IF ceiling IS NULL OR mbuyer<>buyer OR (msupplier IS NOT NULL AND msupplier<>supplier) THEN RAISE EXCEPTION 'mandate ownership mismatch'; END IF;
   IF mstate <> 'active' OR mstart>now() OR mend<=now() THEN RAISE EXCEPTION 'mandate is not active in validity period'; END IF;
   SELECT COALESCE(SUM(a.succeeded_amount_kobo),0) INTO used FROM app.collection_attempts a JOIN app.collection_reservations r ON r.id=a.reservation_id WHERE r.mandate_id=mid;
   SELECT COALESCE(SUM(reserved_amount_kobo),0) INTO held FROM app.collection_reservations WHERE mandate_id=mid AND state IN ('PROCESSING','COMPLETED');
   IF NEW.reserved_amount_kobo > ceiling-used-held THEN RAISE EXCEPTION 'mandate capacity exhausted'; END IF;
   NEW.mandate_id := mid;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
