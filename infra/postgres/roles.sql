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
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_backup') THEN
        CREATE ROLE kredit_backup NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE BYPASSRLS;
    END IF;
END
$$;

REVOKE ALL ON SCHEMA app FROM PUBLIC;
GRANT USAGE ON SCHEMA app TO kredit_app, kredit_worker, kredit_backup;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA app TO kredit_app;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA app TO kredit_worker;
REVOKE UPDATE, DELETE ON app.audit_events FROM kredit_app, kredit_worker;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA app TO kredit_app, kredit_worker;
ALTER DEFAULT PRIVILEGES IN SCHEMA app GRANT USAGE, SELECT ON SEQUENCES TO kredit_app, kredit_worker;

ALTER DEFAULT PRIVILEGES IN SCHEMA app
    GRANT SELECT, INSERT, UPDATE ON TABLES TO kredit_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA app
    GRANT SELECT, INSERT, UPDATE ON TABLES TO kredit_worker;
GRANT SELECT ON ALL TABLES IN SCHEMA app TO kredit_backup;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA app TO kredit_backup;
ALTER DEFAULT PRIVILEGES IN SCHEMA app GRANT SELECT ON TABLES TO kredit_backup;
ALTER DEFAULT PRIVILEGES IN SCHEMA app GRANT USAGE, SELECT ON SEQUENCES TO kredit_backup;

-- Authentication lookup functions are revoked from PUBLIC by migration 020.
-- Keep the grant here as well so installing roles after migrations produces
-- the same least-privilege runtime contract.
DO $$
BEGIN
    IF to_regprocedure('app.has_admin_role(uuid,text[])') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.has_admin_role(uuid,text[]) TO kredit_app;
    END IF;
    IF to_regprocedure('app.admin_actor_name(uuid)') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.admin_actor_name(uuid) TO kredit_app;
    END IF;
    IF to_regprocedure('app.admin_policy_impact(jsonb)') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.admin_policy_impact(jsonb) TO kredit_app;
    END IF;
    IF to_regprocedure('app.admin_attention(uuid,text[])') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.admin_attention(uuid,text[]) TO kredit_app;
    END IF;
    IF to_regprocedure('app.admin_attention_details()') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.admin_attention_details() TO kredit_app;
    END IF;
    IF to_regprocedure('app.is_active_policy_admin(uuid)') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.is_active_policy_admin(uuid) TO kredit_app;
    END IF;
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
    IF to_regprocedure('app.supplier_customers(uuid)') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.supplier_customers(UUID) TO kredit_app;
    END IF;
    IF to_regprocedure('app.credit_snapshot_by_id(text)') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.credit_snapshot_by_id(TEXT) TO kredit_app, kredit_worker;
    END IF;
    IF to_regprocedure('app.credit_snapshot_by_obligation(text)') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.credit_snapshot_by_obligation(TEXT) TO kredit_app, kredit_worker;
    END IF;
    IF to_regprocedure('app.payment_mandate_by_provider(text,text)') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.payment_mandate_by_provider(TEXT, TEXT) TO kredit_app, kredit_worker;
    END IF;
    IF to_regprocedure('app.trade_line_mandate(uuid,uuid,uuid,uuid)') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.trade_line_mandate(UUID, UUID, UUID, UUID) TO kredit_app;
    END IF;
    IF to_regprocedure('app.delete_expired_idempotency_record(text,text)') IS NOT NULL THEN
        GRANT EXECUTE ON FUNCTION app.delete_expired_idempotency_record(TEXT, TEXT) TO kredit_app;
    END IF;
END
$$;

-- Collection/reconciliation posts immutable ledger entries and enqueues River
-- jobs. No runtime role may update or delete financial journal entries.
GRANT USAGE ON SCHEMA ledger, jobs TO kredit_app, kredit_worker, kredit_backup;
GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA ledger TO kredit_app, kredit_worker;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA ledger TO kredit_app, kredit_worker;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA jobs TO kredit_app, kredit_worker;
GRANT DELETE ON ALL TABLES IN SCHEMA jobs TO kredit_worker;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA jobs TO kredit_app, kredit_worker;
GRANT SELECT ON ALL TABLES IN SCHEMA ledger, jobs TO kredit_backup;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA ledger, jobs TO kredit_backup;
ALTER DEFAULT PRIVILEGES IN SCHEMA ledger GRANT SELECT ON TABLES TO kredit_backup;
ALTER DEFAULT PRIVILEGES IN SCHEMA jobs GRANT SELECT ON TABLES TO kredit_backup;
ALTER DEFAULT PRIVILEGES IN SCHEMA ledger GRANT USAGE, SELECT ON SEQUENCES TO kredit_backup;
ALTER DEFAULT PRIVILEGES IN SCHEMA jobs GRANT USAGE, SELECT ON SEQUENCES TO kredit_backup;
REVOKE UPDATE, DELETE ON app.collection_events FROM kredit_app, kredit_worker;
GRANT EXECUTE ON FUNCTION app.record_product_event(TEXT,UUID,UUID,TEXT,TIMESTAMPTZ,TEXT,JSONB) TO kredit_app,kredit_worker;
GRANT EXECUTE ON FUNCTION app.reconcile_supplier_onboarding(TIMESTAMPTZ) TO kredit_worker;

-- Startup verifies the minimum financial migration, including updated trigger bodies.
GRANT SELECT ON public.goose_db_version TO kredit_app,kredit_worker;
GRANT USAGE ON SCHEMA public TO kredit_backup;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO kredit_backup;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO kredit_backup;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO kredit_backup;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO kredit_backup;

-- Authentication hardening installed after the original lookup functions.
GRANT EXECUTE ON FUNCTION app.touch_session(UUID,TIMESTAMPTZ) TO kredit_app;
GRANT EXECUTE ON FUNCTION app.record_rate_limit_attempt(BYTEA,INTERVAL) TO kredit_app;
GRANT EXECUTE ON FUNCTION app.prune_rate_limits(INTERVAL) TO kredit_worker;

-- Keep buyer evidence independent of automated collection workers, including
-- when this provisioning script is rerun after a schema upgrade.
REVOKE INSERT, UPDATE, DELETE ON app.collection_notice_acknowledgements FROM kredit_worker;
REVOKE UPDATE, DELETE ON app.collection_notice_acknowledgements FROM kredit_app;
