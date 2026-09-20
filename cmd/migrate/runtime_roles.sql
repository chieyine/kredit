-- Bootstrap role identities only, before migrations refer to them.
-- Object grants remain in infra/postgres/roles.sql, after migrations.
-- Existing roles are never elevated or modified. A restricted migrator works
-- when a database administrator has already provisioned these identities.
DO $$
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended('kredit:runtime-role-bootstrap', 0));
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        CREATE ROLE kredit_app NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_worker') THEN
        CREATE ROLE kredit_worker NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
    END IF;
END
$$;
