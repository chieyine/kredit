-- +goose Up
CREATE TABLE app.notification_delivery_receipts (
 channel text NOT NULL, event_id text NOT NULL, payload_hash text NOT NULL,
 notification_id uuid NOT NULL REFERENCES app.notifications(id), received_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(channel,event_id)
);
ALTER TABLE app.notification_delivery_receipts ENABLE ROW LEVEL SECURITY;
CREATE POLICY notification_receipts_runtime ON app.notification_delivery_receipts USING(current_user IN ('kredit_app','kredit_worker')) WITH CHECK(current_user IN ('kredit_app','kredit_worker'));
CREATE TRIGGER notification_receipts_immutable BEFORE UPDATE OR DELETE ON app.notification_delivery_receipts FOR EACH ROW EXECUTE FUNCTION app.reject_operations_command_mutation();
-- +goose StatementBegin
CREATE FUNCTION app.collection_notice_key(item app.schedule_items) RETURNS text LANGUAGE sql IMMUTABLE AS $$
 SELECT 'pre-debit:'||item.id::text||':'||item.principal_due_kobo::text||':'||extract(epoch FROM item.collection_at)::text;
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION app.guard_collection_notice() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE minimum_seconds bigint;
BEGIN
 minimum_seconds:=COALESCE(NULLIF(current_setting('app.collection_notice_min_seconds',true),''),'0')::bigint;
 IF minimum_seconds<=0 OR NEW.state<>'PROCESSING' OR EXISTS(SELECT 1 FROM app.collection_reservations WHERE id=NEW.id) THEN RETURN NEW; END IF;
 -- Match the amount/date-specific intent and a real delivery receipt. Merely
 -- queueing a notice or the connector accepting a send is insufficient.
 PERFORM 1 FROM app.obligations WHERE id=NEW.obligation_id FOR UPDATE;
 IF EXISTS(SELECT 1 FROM app.repayment_schedules s JOIN app.schedule_items i ON i.schedule_id=s.id
 WHERE s.obligation_id=NEW.obligation_id AND i.state NOT IN ('PAID','CANCELLED') AND i.principal_due_kobo>i.allocated_kobo AND i.collection_at<=now()
 AND NOT EXISTS(SELECT 1 FROM app.outbox_events e JOIN app.notifications n ON n.event_reference='outbox:'||e.id::text
 JOIN app.notification_delivery_receipts receipt ON receipt.notification_id=n.id
 WHERE e.idempotency_key=app.collection_notice_key(i) AND n.state IN ('delivered','read') AND receipt.received_at<=now()-make_interval(secs=>minimum_seconds::double precision))) THEN
 RAISE EXCEPTION 'confirmed prior-debit notice and minimum waiting period are required';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER collection_notice_guard BEFORE INSERT ON app.collection_reservations FOR EACH ROW EXECUTE FUNCTION app.guard_collection_notice();
-- +goose Down
DROP TRIGGER collection_notice_guard ON app.collection_reservations;
DROP FUNCTION app.guard_collection_notice();
DROP FUNCTION app.collection_notice_key(app.schedule_items);
DROP TABLE app.notification_delivery_receipts;
