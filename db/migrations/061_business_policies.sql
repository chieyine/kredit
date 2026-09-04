-- +goose Up
CREATE TABLE app.business_policy_defaults (singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton), values jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE app.business_policy_changes (
 id uuid PRIMARY KEY, revision bigint GENERATED ALWAYS AS IDENTITY UNIQUE,
 base_revision bigint NOT NULL CHECK(base_revision>=0), values jsonb NOT NULL,
 proposed_by uuid NOT NULL REFERENCES app.users(id), reason text NOT NULL CHECK(length(trim(reason)) BETWEEN 8 AND 2000),
 effective_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 state text NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','approved','rejected','cancelled')),
 decided_by uuid REFERENCES app.users(id), decided_at timestamptz,
 CHECK(state<>'approved' OR (decided_by IS NOT NULL AND decided_by<>proposed_by AND decided_at IS NOT NULL AND effective_at>=decided_at))
);
CREATE UNIQUE INDEX business_policy_one_pending ON app.business_policy_changes((true)) WHERE state='pending';
CREATE TABLE app.business_policy_events(id uuid PRIMARY KEY DEFAULT uuidv7(),change_id uuid NOT NULL REFERENCES app.business_policy_changes(id),actor_id uuid NOT NULL REFERENCES app.users(id),action text NOT NULL,reason text NOT NULL CHECK(length(trim(reason)) BETWEEN 8 AND 2000),occurred_at timestamptz NOT NULL DEFAULT now());
ALTER TABLE app.business_policy_defaults ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.business_policy_changes ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.business_policy_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY business_policy_defaults_runtime ON app.business_policy_defaults USING(current_user IN ('kredit_app','kredit_worker')) WITH CHECK(current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY business_policy_changes_runtime ON app.business_policy_changes USING(current_user IN ('kredit_app','kredit_worker')) WITH CHECK(current_user='kredit_app');
CREATE POLICY business_policy_events_runtime ON app.business_policy_events USING(current_user IN ('kredit_app','kredit_worker')) WITH CHECK(current_user='kredit_app');
CREATE TRIGGER business_policy_defaults_immutable BEFORE UPDATE OR DELETE ON app.business_policy_defaults FOR EACH ROW EXECUTE FUNCTION app.reject_operations_command_mutation();
CREATE TRIGGER business_policy_events_immutable BEFORE UPDATE OR DELETE ON app.business_policy_events FOR EACH ROW EXECUTE FUNCTION app.reject_operations_command_mutation();
-- +goose StatementBegin
CREATE FUNCTION app.guard_business_policy_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'policy history is immutable'; END IF;
 IF TG_OP='UPDATE' THEN
 IF (to_jsonb(NEW)-ARRAY['state','decided_by','decided_at']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['state','decided_by','decided_at']) THEN RAISE EXCEPTION 'proposed policy is immutable'; END IF;
 IF NOT ((OLD.state='pending' AND NEW.state IN ('approved','rejected','cancelled')) OR (OLD.state='approved' AND OLD.effective_at>clock_timestamp() AND NEW.state='cancelled')) THEN RAISE EXCEPTION 'invalid policy transition'; END IF;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER business_policy_change_guard BEFORE UPDATE OR DELETE ON app.business_policy_changes FOR EACH ROW EXECUTE FUNCTION app.guard_business_policy_change();
-- +goose StatementBegin
CREATE FUNCTION app.business_policy() RETURNS jsonb LANGUAGE sql STABLE AS $$
 SELECT COALESCE((SELECT values FROM app.business_policy_changes WHERE state='approved' AND effective_at<=now() ORDER BY revision DESC LIMIT 1),(SELECT values FROM app.business_policy_defaults WHERE singleton),'{}'::jsonb);
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION app.is_active_policy_admin(actor uuid) RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT EXISTS(SELECT 1 FROM app.platform_role_assignments r JOIN app.users u ON u.id=r.user_id WHERE r.user_id=actor AND r.role='platform_admin' AND r.revoked_at IS NULL AND (r.expires_at IS NULL OR r.expires_at>now()) AND u.status='active');
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.is_active_policy_admin(uuid) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
  GRANT EXECUTE ON FUNCTION app.is_active_policy_admin(uuid) TO kredit_app;
  GRANT SELECT,INSERT,UPDATE ON app.business_policy_defaults,app.business_policy_changes,app.business_policy_events TO kredit_app;
  GRANT USAGE,SELECT ON SEQUENCE app.business_policy_changes_revision_seq TO kredit_app;
 END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
  GRANT SELECT,INSERT ON app.business_policy_defaults TO kredit_worker;
  GRANT SELECT ON app.business_policy_changes,app.business_policy_events TO kredit_worker;
 END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM app.business_policy_changes) THEN RAISE EXCEPTION 'policy decision history must be retained; use a forward migration'; END IF;
END $$;
-- +goose StatementEnd
DROP FUNCTION IF EXISTS app.is_active_policy_admin(uuid);
DROP FUNCTION app.business_policy();
DROP TABLE app.business_policy_events,app.business_policy_changes,app.business_policy_defaults;
DROP FUNCTION app.guard_business_policy_change();
