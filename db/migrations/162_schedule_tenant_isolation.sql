-- +goose Up
DROP POLICY schedule_runtime_access ON app.repayment_schedules;
DROP POLICY schedule_item_runtime_access ON app.schedule_items;
-- Workers also carry organization context; the obligation policy is the
-- authority for both the parent schedule and its existing child policy.
CREATE POLICY schedule_obligation_access ON app.repayment_schedules
USING (EXISTS (SELECT 1 FROM app.obligations o WHERE o.id=obligation_id));

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Schedule isolation requires forward recovery'; END $$;
-- +goose StatementEnd
