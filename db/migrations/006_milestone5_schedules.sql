-- +goose Up
CREATE TABLE app.repayment_schedules (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  obligation_id uuid NOT NULL UNIQUE REFERENCES app.obligations(id),
  schedule_type text NOT NULL CHECK (schedule_type IN ('equal','custom')),
  timezone text NOT NULL,
  allocation_policy text NOT NULL,
  cadence text NOT NULL CHECK (cadence IN ('weekly','fortnightly','monthly','custom')),
  grace_hours integer NOT NULL CHECK (grace_hours BETWEEN 0 AND 720),
  status text NOT NULL CHECK (status IN ('ACTIVE','PAUSED','COMPLETED','CANCELLED')),
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.schedule_items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  schedule_id uuid NOT NULL REFERENCES app.repayment_schedules(id),
  sequence integer NOT NULL,
  principal_due_kobo bigint NOT NULL CHECK (principal_due_kobo > 0),
  due_at timestamptz NOT NULL,
  grace_hours integer NOT NULL CHECK (grace_hours BETWEEN 0 AND 720),
  collection_at timestamptz NOT NULL,
  allocated_kobo bigint NOT NULL DEFAULT 0 CHECK (allocated_kobo >= 0 AND allocated_kobo <= principal_due_kobo),
  collected_kobo bigint NOT NULL DEFAULT 0 CHECK (collected_kobo >= 0 AND collected_kobo <= principal_due_kobo),
  state text NOT NULL CHECK (state IN ('OPEN','IN_GRACE','OVERDUE','PARTIALLY_PAID','PAID','CANCELLED')),
  disputed_kobo bigint NOT NULL DEFAULT 0 CHECK (disputed_kobo >= 0),
  collection_block_reason text,
  UNIQUE (schedule_id, sequence)
);
CREATE INDEX schedule_items_due_idx ON app.schedule_items (state, collection_at);

ALTER TABLE app.repayment_schedules ENABLE ROW LEVEL SECURITY;
CREATE POLICY schedule_supplier_access ON app.repayment_schedules USING (obligation_id IN (SELECT id FROM app.obligations WHERE supplier_organization_id IN (SELECT organization_id FROM app.memberships WHERE user_id = app.current_user_id() AND status = 'active')));
CREATE POLICY schedule_buyer_access ON app.repayment_schedules USING (obligation_id IN (SELECT app.obligations.id FROM app.obligations JOIN app.credit_requests ON app.credit_requests.id = app.obligations.credit_request_id WHERE app.credit_requests.buyer_user_id = app.current_user_id()));
ALTER TABLE app.schedule_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY schedule_item_access ON app.schedule_items USING (schedule_id IN (SELECT id FROM app.repayment_schedules));

-- +goose Down
DROP TABLE IF EXISTS app.schedule_items, app.repayment_schedules;
