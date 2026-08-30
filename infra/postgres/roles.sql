-- Apply with a migration/admin role after schema migrations. This file is a
-- deployment template; it intentionally does not contain passwords.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        CREATE ROLE kredit_app NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_worker') THEN
        CREATE ROLE kredit_worker NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_migrator') THEN
        CREATE ROLE kredit_migrator NOLOGIN NOSUPERUSER CREATEDB NOCREATEROLE;
    END IF;
END
$$;

REVOKE ALL ON SCHEMA app FROM PUBLIC;
GRANT USAGE ON SCHEMA app TO kredit_app, kredit_worker;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA app TO kredit_app;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA app TO kredit_worker;
REVOKE UPDATE, DELETE ON app.audit_events FROM kredit_app, kredit_worker;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA app TO kredit_app, kredit_worker;

ALTER DEFAULT PRIVILEGES IN SCHEMA app
    GRANT SELECT, INSERT, UPDATE ON TABLES TO kredit_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA app
    GRANT SELECT, INSERT, UPDATE ON TABLES TO kredit_worker;

-- Authentication lookup functions are revoked from PUBLIC by migration 020.
-- Keep the grant here as well so installing roles after migrations produces
-- the same least-privilege runtime contract.
DO $$
BEGIN
    IF to_regprocedure('app.find_or_create_user(text,text,timestamptz)') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.find_or_create_user(TEXT, TEXT, TIMESTAMPTZ) TO kredit_app;
    END IF;
    IF to_regprocedure('app.session_by_token_hash(bytea)') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.session_by_token_hash(BYTEA) TO kredit_app;
    END IF;
    IF to_regprocedure('app.organization_count()') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.organization_count() TO kredit_app;
    END IF;
    IF to_regprocedure('app.buyer_invitation_by_token_hash(bytea)') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.buyer_invitation_by_token_hash(BYTEA) TO kredit_app;
    END IF;
    IF to_regprocedure('app.business_count()') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.business_count() TO kredit_app;
    END IF;
    IF to_regprocedure('app.credit_snapshot_by_id(text)') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.credit_snapshot_by_id(TEXT) TO kredit_app;
    END IF;
    IF to_regprocedure('app.credit_snapshot_by_obligation(text)') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.credit_snapshot_by_obligation(TEXT) TO kredit_app;
    END IF;
    IF to_regprocedure('app.payment_mandate_by_provider(text,text)') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.payment_mandate_by_provider(TEXT, TEXT) TO kredit_app;
    END IF;
END
$$;
