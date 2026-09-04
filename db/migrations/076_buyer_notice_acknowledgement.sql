-- +goose Up

CREATE TABLE app.collection_notice_acknowledgements (
    schedule_item_id UUID PRIMARY KEY REFERENCES app.schedule_items(id),
    buyer_user_id UUID NOT NULL REFERENCES app.users(id),
    notification_id UUID NOT NULL REFERENCES app.notifications(id),
    receipt_channel TEXT NOT NULL,
    receipt_event_id TEXT NOT NULL,
    FOREIGN KEY (receipt_channel, receipt_event_id)
      REFERENCES app.notification_delivery_receipts(channel, event_id),
    acknowledged_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE app.collection_notice_acknowledgements ENABLE ROW LEVEL SECURITY;
CREATE POLICY collection_notice_ack_buyer
ON app.collection_notice_acknowledgements
USING (buyer_user_id = app.current_user_id())
WITH CHECK (buyer_user_id = app.current_user_id());
CREATE POLICY collection_notice_ack_worker_read
ON app.collection_notice_acknowledgements FOR SELECT
USING (current_user = 'kredit_worker');

-- A connector receipt is transport evidence, not buyer consent. Collection
-- requires both the receipt and an acknowledgement made by the authenticated
-- buyer, each old enough to satisfy the notice period.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_collection_notice() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE minimum_seconds bigint;
BEGIN
 minimum_seconds:=COALESCE(NULLIF(current_setting('app.collection_notice_min_seconds',true),''),'0')::bigint;
 IF minimum_seconds<=0 OR NEW.state<>'PROCESSING' OR EXISTS(SELECT 1 FROM app.collection_reservations WHERE id=NEW.id) THEN RETURN NEW; END IF;
 PERFORM 1 FROM app.obligations WHERE id=NEW.obligation_id FOR UPDATE;
 IF EXISTS(SELECT 1 FROM app.repayment_schedules s JOIN app.schedule_items i ON i.schedule_id=s.id
 WHERE s.obligation_id=NEW.obligation_id AND i.state NOT IN ('PAID','CANCELLED') AND i.principal_due_kobo>i.allocated_kobo AND i.collection_at<=now()
 AND NOT EXISTS(SELECT 1 FROM app.outbox_events e JOIN app.notifications n ON n.event_reference='outbox:'||e.id::text
 JOIN app.notification_delivery_receipts receipt ON receipt.notification_id=n.id
 JOIN app.collection_notice_acknowledgements ack
   ON ack.schedule_item_id=i.id
  AND ack.notification_id=n.id
  AND ack.receipt_channel=receipt.channel
  AND ack.receipt_event_id=receipt.event_id
 WHERE e.idempotency_key=app.collection_notice_key(i) AND n.state IN ('delivered','read')
 AND receipt.received_at<=now()-make_interval(secs=>minimum_seconds::double precision)
 AND ack.acknowledged_at<=now()-make_interval(secs=>minimum_seconds::double precision))) THEN
 RAISE EXCEPTION 'confirmed delivery, buyer acknowledgement, and minimum waiting period are required';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd

-- Pilot policy forbids silence-based activation. Keep the database boundary
-- closed even if a stale worker or future caller invokes the old code path.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_deemed_acceptance() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF NEW.issue_reason = 'deemed_acceptance_auto_activated' THEN
    RAISE EXCEPTION 'deemed acceptance is disabled by pilot policy';
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_collection_notice() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE minimum_seconds bigint;
BEGIN
 minimum_seconds:=COALESCE(NULLIF(current_setting('app.collection_notice_min_seconds',true),''),'0')::bigint;
 IF minimum_seconds<=0 OR NEW.state<>'PROCESSING' OR EXISTS(SELECT 1 FROM app.collection_reservations WHERE id=NEW.id) THEN RETURN NEW; END IF;
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

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_deemed_acceptance() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.issue_reason = 'deemed_acceptance_auto_activated' THEN
    RAISE EXCEPTION 'deemed acceptance remains disabled during rollback';
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

DROP TABLE IF EXISTS app.collection_notice_acknowledgements;
