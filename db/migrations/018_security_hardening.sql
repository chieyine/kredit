-- +goose Up

-- Audit history is an append-only security boundary. Application code never
-- receives an UPDATE or DELETE path for these rows, and the database enforces
-- that invariant even if a privileged query is accidentally introduced.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.prevent_audit_event_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'audit events are append-only';
END;
$$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS audit_events_append_only ON app.audit_events;
CREATE TRIGGER audit_events_append_only
    BEFORE UPDATE OR DELETE ON app.audit_events
    FOR EACH ROW EXECUTE FUNCTION app.prevent_audit_event_mutation();

-- Application and worker roles must not bypass tenant policies merely because
-- they own a table. The migration owner remains able to repair schema during
-- controlled migrations; runtime roles are granted only the required paths.
ALTER TABLE app.audit_events FORCE ROW LEVEL SECURITY;

-- +goose Down
ALTER TABLE app.audit_events NO FORCE ROW LEVEL SECURITY;
DROP TRIGGER IF EXISTS audit_events_append_only ON app.audit_events;
DROP FUNCTION IF EXISTS app.prevent_audit_event_mutation();
