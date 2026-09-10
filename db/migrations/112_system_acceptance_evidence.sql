-- +goose Up
-- System recognition is distinct from buyer-originated receipt evidence.
CREATE TABLE app.system_acceptances (
 id uuid PRIMARY KEY DEFAULT uuidv7(),
 credit_request_id uuid NOT NULL UNIQUE REFERENCES app.credit_requests(id),
 supplier_organization_id uuid NOT NULL REFERENCES app.organizations(id),
 buyer_user_id uuid NOT NULL REFERENCES app.users(id),
 release_id uuid NOT NULL REFERENCES app.goods_releases(id),
 notification_id uuid NOT NULL REFERENCES app.notifications(id),
 receipt_channel text NOT NULL,
 receipt_event_id text NOT NULL,
 minimum_seconds bigint NOT NULL CHECK(minimum_seconds BETWEEN 259200 AND 2592000),
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 FOREIGN KEY(receipt_channel,receipt_event_id) REFERENCES app.notification_delivery_receipts(channel,event_id)
);
ALTER TABLE app.system_acceptances ENABLE ROW LEVEL SECURITY;
CREATE POLICY system_acceptance_tenant ON app.system_acceptances
 USING(buyer_user_id=app.current_user_id() OR supplier_organization_id=app.current_organization_id())
 WITH CHECK(current_user='kredit_worker' AND supplier_organization_id=app.current_organization_id());
CREATE TRIGGER system_acceptance_immutable BEFORE UPDATE OR DELETE ON app.system_acceptances
 FOR EACH ROW EXECUTE FUNCTION app.reject_operations_command_mutation();

-- This narrow read returns only qualifying, already-delivered evidence. It does
-- not manufacture a receipt or accept an agreement on behalf of a customer.
-- +goose StatementBegin
CREATE FUNCTION app.deemed_acceptance_evidence(request_id uuid, wait_seconds bigint)
RETURNS TABLE(organization_id uuid,buyer_id uuid,release_id uuid,notification_id uuid,receipt_channel text,receipt_event_id text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT c.supplier_organization_id,c.buyer_user_id,g.id,n.id,r.channel,r.event_id
 FROM app.credit_requests c
 JOIN app.goods_releases g ON g.id=c.release_id AND g.credit_request_id=c.id
 JOIN app.agreement_acceptances a ON a.id=c.acceptance_id AND a.credit_request_id=c.id
   AND a.accepting_user_id=c.buyer_user_id AND a.business_id=c.buyer_business_id
   AND a.agreement_version_id=c.agreement_version_id
 JOIN app.agreement_versions v ON v.id=a.agreement_version_id AND v.document_hash=a.agreement_hash
 JOIN app.outbox_events e ON e.idempotency_key='goods-release-notification:'||g.id::text
   AND e.aggregate_id=c.id::text AND e.event_type='notification.requested'
 JOIN app.notifications n ON n.event_reference='outbox:'||e.id::text AND n.recipient_id=c.buyer_user_id
 JOIN app.notification_delivery_receipts r ON r.notification_id=n.id
 WHERE c.id=request_id AND c.state='RECEIPT_CONFIRMATION_PENDING'
   AND wait_seconds BETWEEN 259200 AND 2592000
   AND g.released_at<=now()-interval '72 hours'
   AND n.state IN ('delivered','read')
   AND r.received_at<=now()-make_interval(secs=>wait_seconds::double precision)
   AND NOT EXISTS(SELECT 1 FROM app.receipt_confirmations rc WHERE rc.credit_request_id=c.id)
   AND NOT EXISTS(SELECT 1 FROM app.system_acceptances sa WHERE sa.credit_request_id=c.id)
   AND NOT EXISTS(SELECT 1 FROM app.obligations o WHERE o.credit_request_id=c.id)
   AND EXISTS(SELECT 1 FROM app.receipt_confirmations prior
     JOIN app.credit_requests p ON p.id=prior.credit_request_id
     WHERE p.buyer_business_id=c.buyer_business_id AND p.buyer_user_id=c.buyer_user_id
       AND p.id<>c.id AND prior.issue_reason IS DISTINCT FROM 'deemed_acceptance_auto_activated')
 ORDER BY r.received_at,n.id,r.channel,r.event_id LIMIT 1;
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.deemed_acceptance_evidence(uuid,bigint) FROM PUBLIC;

-- +goose StatementBegin
CREATE FUNCTION app.deemed_acceptance_candidates(wait_seconds bigint)
RETURNS TABLE(request_id uuid,organization_id uuid,buyer_id uuid)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT c.id,e.organization_id,e.buyer_id FROM app.credit_requests c
 JOIN LATERAL app.deemed_acceptance_evidence(c.id,wait_seconds) e ON true
 WHERE c.state='RECEIPT_CONFIRMATION_PENDING'
 ORDER BY c.created_at,c.id LIMIT 100;
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.deemed_acceptance_candidates(bigint) FROM PUBLIC;

-- Recheck the evidence inside the activation transaction, even for direct SQL.
-- +goose StatementBegin
CREATE FUNCTION app.guard_system_acceptance() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,app AS $$
DECLARE evidence record;
BEGIN
 IF current_user<>'kredit_worker' THEN RAISE EXCEPTION 'system acceptance requires the system worker' USING ERRCODE='42501'; END IF;
 PERFORM 1 FROM app.credit_requests WHERE id=NEW.credit_request_id FOR UPDATE;
 SELECT * INTO evidence FROM app.deemed_acceptance_evidence(NEW.credit_request_id,NEW.minimum_seconds);
 IF NOT FOUND THEN RAISE EXCEPTION 'system acceptance evidence is not eligible' USING ERRCODE='23514'; END IF;
 NEW.supplier_organization_id:=evidence.organization_id;
 NEW.buyer_user_id:=evidence.buyer_id;
 NEW.release_id:=evidence.release_id;
 NEW.notification_id:=evidence.notification_id;
 NEW.receipt_channel:=evidence.receipt_channel;
 NEW.receipt_event_id:=evidence.receipt_event_id;
 NEW.recorded_at:=clock_timestamp();
 RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER system_acceptance_guard BEFORE INSERT ON app.system_acceptances
 FOR EACH ROW EXECUTE FUNCTION app.guard_system_acceptance();
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
  GRANT SELECT,INSERT ON app.system_acceptances TO kredit_worker;
  GRANT EXECUTE ON FUNCTION app.deemed_acceptance_evidence(uuid,bigint),app.deemed_acceptance_candidates(bigint) TO kredit_worker;
 END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT SELECT ON app.system_acceptances TO kredit_app; END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'System acceptance evidence requires forward recovery'; END $$;
-- +goose StatementEnd
