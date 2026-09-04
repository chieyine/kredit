-- Development-only login wrappers for the NOLOGIN roles in roles.sql.
-- Production provisions equivalent credentials through its secret manager.
-- Run this as the local database owner after migrations and roles.sql.

-- Repair pre-existing group roles before granting membership. A stale
-- privileged role must not survive a repeated provisioning run.
ALTER ROLE kredit_app NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
ALTER ROLE kredit_worker NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
ALTER ROLE kredit_migrator NOLOGIN NOINHERIT NOSUPERUSER CREATEDB NOCREATEROLE NOBYPASSRLS;
ALTER ROLE kredit_backup NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE BYPASSRLS;

SELECT format(
    'CREATE ROLE kredit_app_login LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD %L',
    :'app_login_password'
)
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app_login')
\gexec

SELECT format(
    'CREATE ROLE kredit_worker_login LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD %L',
    :'worker_login_password'
)
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_worker_login')
\gexec

SELECT format(
    'CREATE ROLE kredit_backup_login LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD %L',
    :'backup_login_password'
)
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_backup_login')
\gexec

ALTER ROLE kredit_app_login LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD :'app_login_password';
ALTER ROLE kredit_worker_login LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD :'worker_login_password';
ALTER ROLE kredit_backup_login LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD :'backup_login_password';

GRANT kredit_app TO kredit_app_login WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
GRANT kredit_worker TO kredit_worker_login WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
GRANT kredit_backup TO kredit_backup_login WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

-- The owner login is retained for migrations, seed fixtures and controlled
-- backups. SET-only membership lets integration tests exercise both runtime
-- roles without inheriting their privileges by default.
GRANT kredit_app TO kredit WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
GRANT kredit_worker TO kredit WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

SELECT 'ALTER ROLE kredit NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS'
WHERE :'demote_owner' = 'true'
  AND EXISTS (
    SELECT 1 FROM pg_roles
    WHERE rolname = 'kredit'
      AND (rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls)
)
\gexec
