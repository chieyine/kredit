-- +goose Up
-- FORCE RLS protects audit rows even from their ordinary table owner. The
-- SECURITY DEFINER activity trigger must append an event for any tenant whose
-- authorized write fired it, including multi-tenant development seeding.
-- These policies are usable only while a trigger is running as this exact
-- function owner. Runtime roles neither own the function nor may create app
-- triggers. No UPDATE/DELETE permission or runtime bypass is added.
CREATE POLICY audit_activity_trigger_insert ON app.audit_events FOR INSERT
WITH CHECK (
 pg_trigger_depth()>0 AND current_user=pg_get_userbyid(
   (SELECT proowner FROM pg_proc WHERE oid='app.record_domain_activity()'::regprocedure))
);
-- INSERT ... RETURNING also needs SELECT visibility for the inserted row.
CREATE POLICY audit_activity_trigger_returning ON app.audit_events FOR SELECT
USING (
 pg_trigger_depth()>0 AND current_user=pg_get_userbyid(
   (SELECT proowner FROM pg_proc WHERE oid='app.record_domain_activity()'::regprocedure))
);

-- +goose Down
DROP POLICY IF EXISTS audit_activity_trigger_returning ON app.audit_events;
DROP POLICY IF EXISTS audit_activity_trigger_insert ON app.audit_events;
