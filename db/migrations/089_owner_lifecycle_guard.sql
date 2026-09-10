-- +goose Up
-- Once ownership exists, retain an active owner whose authority cannot lapse
-- with the clock. Expiring delegated ownership may coexist with that owner.
CREATE TABLE app.platform_owner_guard (
    singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton),
    initialized boolean NOT NULL DEFAULT false
);
INSERT INTO app.platform_owner_guard(singleton, initialized)
SELECT true, EXISTS(SELECT 1 FROM app.platform_role_assignments WHERE role='platform_owner');
REVOKE ALL ON app.platform_owner_guard FROM PUBLIC;

-- +goose StatementBegin
CREATE FUNCTION app.lock_owner_lifecycle() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
BEGIN
  -- Statement-level locking precedes row locks, and is shared across both tables.
  PERFORM pg_advisory_xact_lock(746219830045::bigint);
  RETURN NULL;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION app.enforce_owner_lifecycle() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE started boolean;
BEGIN
  SELECT initialized INTO STRICT started FROM app.platform_owner_guard WHERE singleton;
  IF NOT started AND EXISTS(SELECT 1 FROM app.platform_role_assignments WHERE role='platform_owner') THEN
    UPDATE app.platform_owner_guard SET initialized=true WHERE singleton;
    started := true;
  END IF;
  IF started AND NOT EXISTS(
    SELECT 1 FROM app.platform_role_assignments r JOIN app.users u ON u.id=r.user_id
    WHERE r.role='platform_owner' AND r.revoked_at IS NULL
      AND r.expires_at IS NULL AND u.status='active'
  ) THEN
    RAISE EXCEPTION 'retain an active owner without an expiry before changing ownership or account status'
      USING ERRCODE='23514';
  END IF;
  RETURN NULL;
END;
$$;
-- +goose StatementEnd

DROP TRIGGER protect_last_platform_owner_trigger ON app.platform_role_assignments;
CREATE TRIGGER owner_roles_lock BEFORE INSERT OR UPDATE OR DELETE ON app.platform_role_assignments
FOR EACH STATEMENT EXECUTE FUNCTION app.lock_owner_lifecycle();
CREATE TRIGGER owner_roles_guard AFTER INSERT OR UPDATE OR DELETE ON app.platform_role_assignments
FOR EACH STATEMENT EXECUTE FUNCTION app.enforce_owner_lifecycle();
CREATE TRIGGER owner_accounts_lock BEFORE UPDATE OF status OR DELETE ON app.users
FOR EACH STATEMENT EXECUTE FUNCTION app.lock_owner_lifecycle();
CREATE TRIGGER owner_accounts_guard AFTER UPDATE OF status OR DELETE ON app.users
FOR EACH STATEMENT EXECUTE FUNCTION app.enforce_owner_lifecycle();
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
    REVOKE ALL ON app.platform_owner_guard FROM kredit_app;
    REVOKE TRUNCATE ON app.platform_role_assignments, app.users FROM kredit_app;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
    REVOKE ALL ON app.platform_owner_guard FROM kredit_worker;
    REVOKE TRUNCATE ON app.platform_role_assignments, app.users FROM kredit_worker;
  END IF;
END;
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.lock_owner_lifecycle(), app.enforce_owner_lifecycle() FROM PUBLIC;

-- Check an existing installation without silently extending or granting ownership.
-- +goose StatementBegin
DO $$
BEGIN
  IF (SELECT initialized FROM app.platform_owner_guard WHERE singleton)
     AND NOT EXISTS(SELECT 1 FROM app.platform_role_assignments r JOIN app.users u ON u.id=r.user_id
       WHERE r.role='platform_owner' AND r.revoked_at IS NULL AND r.expires_at IS NULL AND u.status='active') THEN
    RAISE EXCEPTION 'owner lifecycle migration requires an existing active owner without expiry';
  END IF;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- Code rollback must retain the ownership integrity boundary. Forward repair only.
SELECT 1;
