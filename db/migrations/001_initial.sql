-- +goose Up

-- PostgreSQL 18 exposes uuidv7() natively, while older supported clusters do
-- not. Keep migrations portable by supplying a safe UUID fallback only when
-- the server does not already provide the function. The fallback is v4, but
-- preserves the same uniqueness and opaque-identifier guarantees.
CREATE EXTENSION IF NOT EXISTS pgcrypto;
-- +goose StatementBegin
DO $uuidv7$
BEGIN
    IF to_regprocedure('pg_catalog.uuidv7()') IS NULL
       AND to_regprocedure('public.uuidv7()') IS NULL THEN
        CREATE FUNCTION public.uuidv7() RETURNS uuid
        LANGUAGE sql VOLATILE
        AS $body$ SELECT gen_random_uuid() $body$;
    END IF;
END
$uuidv7$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS app_meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO app_meta (key, value)
VALUES ('schema_baseline', 'milestone-0')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();

INSERT INTO schema_migrations (version)
VALUES ('001_initial')
ON CONFLICT (version) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS app_meta;
DROP TABLE IF EXISTS schema_migrations;
