-- +goose Up
-- Keep the acknowledgement of every delivered version. A new amount/date
-- creates a new notice and must not overwrite the buyer's earlier evidence.
ALTER TABLE app.collection_notice_acknowledgements
  DROP CONSTRAINT collection_notice_acknowledgements_pkey,
  ADD PRIMARY KEY (schedule_item_id, notification_id);

-- Buyer schedule reads traverse obligations. Supplier-only visibility hid
-- the buyer's own schedule and made acknowledgement impossible after 081.
CREATE POLICY obligation_buyer_read ON app.obligations FOR SELECT
  USING (EXISTS (SELECT 1 FROM app.credit_requests c
    WHERE c.id = credit_request_id AND c.buyer_user_id = app.current_user_id()));

DROP POLICY collection_notice_ack_buyer_insert ON app.collection_notice_acknowledgements;
CREATE POLICY collection_notice_ack_buyer_insert ON app.collection_notice_acknowledgements FOR INSERT
  WITH CHECK (current_user = 'kredit_app' AND buyer_user_id = app.current_user_id()
    AND EXISTS (
      SELECT 1 FROM app.schedule_items i
      JOIN app.repayment_schedules s ON s.id = i.schedule_id
      JOIN app.obligations o ON o.id = s.obligation_id
      JOIN app.credit_requests c ON c.id = o.credit_request_id
      JOIN app.outbox_events e ON e.idempotency_key = app.collection_notice_key(i)
      JOIN app.notifications n ON n.event_reference = 'outbox:' || e.id::text
      JOIN app.notification_delivery_receipts receipt ON receipt.notification_id = n.id
      WHERE i.id = collection_notice_acknowledgements.schedule_item_id
        AND c.buyer_user_id = app.current_user_id()
        AND n.id = collection_notice_acknowledgements.notification_id
        AND n.recipient_id = app.current_user_id() AND n.state IN ('delivered','read')
        AND receipt.channel = collection_notice_acknowledgements.receipt_channel
        AND receipt.event_id = collection_notice_acknowledgements.receipt_event_id
        AND s.status = 'ACTIVE' AND i.state NOT IN ('PAID','CANCELLED')
        AND i.principal_due_kobo > i.allocated_kobo
    ));

-- +goose Down
-- Removing versioned evidence is not a safe rollback. Older binaries must
-- not be restored over this schema; roll forward or restore a full backup.
-- +goose StatementBegin
DO $$ BEGIN
  RAISE EXCEPTION 'collection notice evidence requires a forward migration';
END $$;
-- +goose StatementEnd
