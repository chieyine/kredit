-- +goose Up

-- Schedule mutations are executed by the credit lifecycle and River workers
-- after the service layer has already authorized the tenant. Permit those
-- bounded repository operations through the dedicated runtime roles; browser
-- callers never receive database credentials.
ALTER TABLE app.repayment_schedules ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS schedule_runtime_access ON app.repayment_schedules;
CREATE POLICY schedule_runtime_access ON app.repayment_schedules
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));

ALTER TABLE app.schedule_items ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS schedule_item_runtime_access ON app.schedule_items;
CREATE POLICY schedule_item_runtime_access ON app.schedule_items
    USING (current_user IN ('kredit_app', 'kredit_worker'))
    WITH CHECK (current_user IN ('kredit_app', 'kredit_worker'));

-- +goose Down
DROP POLICY IF EXISTS schedule_item_runtime_access ON app.schedule_items;
DROP POLICY IF EXISTS schedule_runtime_access ON app.repayment_schedules;
