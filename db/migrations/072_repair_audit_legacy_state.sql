-- +goose Up
-- Preserve the accepted debit instant on still-open, unamended schedules that
-- match the old due-time-plus-grace calculation. Paid and explicitly amended
-- items are historical evidence and must not be rewritten.
WITH original AS (
 SELECT i.id,i.state,i.collection_at,i.due_at,i.grace_hours,
        c.collection_at + ((i.due_at AT TIME ZONE s.timezone)::date -
          min((i.due_at AT TIME ZONE s.timezone)::date) OVER (PARTITION BY s.id)) * interval '1 day' AS accepted_at
 FROM app.schedule_items i
 JOIN app.repayment_schedules s ON s.id=i.schedule_id
 JOIN app.obligations o ON o.id=s.obligation_id
 JOIN app.credit_requests c ON c.id=o.credit_request_id
 WHERE NOT EXISTS(SELECT 1 FROM app.admin_change_requests a WHERE a.obligation_id=o.id AND a.kind='schedule_amendment' AND a.state='applied')
)
UPDATE app.schedule_items i SET collection_at=o.accepted_at
FROM original o WHERE i.id=o.id AND o.state NOT IN ('PAID','CANCELLED')
 AND o.collection_at=o.due_at+o.grace_hours*interval '1 hour'
 AND o.collection_at<>o.accepted_at;

-- Persist optional-reminder classification for jobs queued before migration 070.
UPDATE app.notifications n SET priority='routine',
 supplier_organization_id=COALESCE(
  (SELECT c.supplier_organization_id FROM app.credit_requests c WHERE c.id::text=e.aggregate_id),
  (SELECT o.supplier_organization_id FROM app.obligations o WHERE o.id::text=e.aggregate_id))
FROM app.outbox_events e WHERE n.event_reference='outbox:'||e.id::text
 AND e.payload->>'event'='UPCOMING_DUE';

-- +goose Down
-- Corrected dates and consent classifications must not revert to inaccurate values.
SELECT 1;
