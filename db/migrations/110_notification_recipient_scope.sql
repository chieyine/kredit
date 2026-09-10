-- +goose Up
-- A delivery worker needs minimal identity for a committed notification event
-- before it can establish the tenant context required by financial RLS. This
-- lookup is bound to a persisted outbox identity, not an arbitrary sale ID.
-- +goose StatementBegin
CREATE FUNCTION app.notification_event_identity(event_id uuid,input_type text,input_id text,input_payload jsonb)
RETURNS TABLE(buyer_user_id uuid,supplier_organization_id uuid)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 WITH event AS (
  SELECT aggregate_type,aggregate_id FROM app.outbox_events
  WHERE id=event_id AND event_type='notification.requested' AND aggregate_type=input_type AND aggregate_id=input_id AND payload=input_payload
 )
 SELECT c.buyer_user_id,c.supplier_organization_id FROM event e
 JOIN app.credit_requests c ON c.id::text=e.aggregate_id
 WHERE e.aggregate_type='credit_request'
 UNION ALL
 SELECT p.buyer_user_id,p.supplier_organization_id FROM event e
 JOIN app.payments p ON p.id::text=e.aggregate_id
 WHERE e.aggregate_type='payment'
 UNION ALL
 SELECT COALESCE(person.user_id,business.owner_user_id),m.supplier_organization_id FROM event e
 JOIN app.payment_mandates m ON m.id::text=e.aggregate_id
 LEFT JOIN app.persons person ON m.buyer_subject_type='person' AND person.id=m.buyer_subject_id
 LEFT JOIN app.businesses business ON m.buyer_subject_type='business' AND business.id=m.buyer_subject_id
 WHERE e.aggregate_type='payment_mandate'
 UNION ALL
 SELECT c.buyer_user_id,o.supplier_organization_id FROM event e
 JOIN app.obligations o ON o.id::text=e.aggregate_id
 JOIN app.credit_requests c ON c.id=o.credit_request_id
 WHERE e.aggregate_type NOT IN('credit_request','payment','payment_mandate');
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.notification_event_identity(uuid,text,text,jsonb) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
  GRANT EXECUTE ON FUNCTION app.notification_event_identity(uuid,text,text,jsonb) TO kredit_worker;
 END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Notification tenant discovery requires a forward migration'; END $$;
-- +goose StatementEnd
