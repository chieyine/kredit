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

-- Existing application tables keep their repository-level baseline grants;
-- row-level security is the tenant authorization boundary. New tables are
-- deliberately NOT auto-granted to either runtime role: a migration must make
-- an explicit privilege decision before new data becomes reachable.
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA app TO kredit_app;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA app TO kredit_worker;
REVOKE UPDATE, DELETE ON app.audit_events FROM kredit_app, kredit_worker;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA app TO kredit_app, kredit_worker;
ALTER DEFAULT PRIVILEGES IN SCHEMA app GRANT USAGE, SELECT ON SEQUENCES TO kredit_app, kredit_worker;

ALTER DEFAULT PRIVILEGES IN SCHEMA app
    REVOKE SELECT, INSERT, UPDATE ON TABLES FROM kredit_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA app
    REVOKE SELECT, INSERT, UPDATE ON TABLES FROM kredit_worker;
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
    IF to_regprocedure('app.phase5_financial_metrics()') IS NOT NULL THEN
        REVOKE ALL ON FUNCTION app.phase5_financial_metrics() FROM PUBLIC;
        GRANT EXECUTE ON FUNCTION app.phase5_financial_metrics() TO kredit_app, kredit_worker;
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

-- Buyer-originated evidence is independent of autonomous collection workers.
-- The worker may read evidence needed to decide whether a debit is eligible, but
-- it cannot create, rewrite, or delete the buyer's decision/evidence records.
REVOKE INSERT, UPDATE, DELETE ON app.agreement_acceptances FROM kredit_worker;
REVOKE INSERT, UPDATE, DELETE ON app.receipt_confirmations FROM kredit_worker;
REVOKE INSERT, UPDATE, DELETE ON app.payment_claims FROM kredit_worker;
REVOKE INSERT, UPDATE, DELETE ON app.collection_notice_acknowledgements FROM kredit_worker;
REVOKE UPDATE, DELETE ON app.collection_notice_acknowledgements FROM kredit_app;

-- Phase 2 narrow global collection discovery. These functions expose only
-- resource and tenant identifiers; financial reads/writes still use RLS.
GRANT EXECUTE ON FUNCTION app.collection_due_work_page(TEXT,INTEGER) TO kredit_worker;
GRANT EXECUTE ON FUNCTION app.collection_attempt_work_page(TEXT,INTEGER) TO kredit_worker;
GRANT EXECUTE ON FUNCTION app.collection_identity_by_attempt(UUID) TO kredit_worker;
GRANT EXECUTE ON FUNCTION app.collection_identity_by_external(TEXT) TO kredit_worker;
GRANT EXECUTE ON FUNCTION app.collection_attempt_identity_by_external(TEXT) TO kredit_worker;
GRANT EXECUTE ON FUNCTION app.enqueue_pre_debit_notices() TO kredit_worker;
GRANT EXECUTE ON FUNCTION app.enqueue_due_payment_notices(INTEGER) TO kredit_worker;

-- Preserve the owner and settings boundaries after the broad baseline grants.
-- This also supports installing roles after migrations 087-091 on a fresh DB.
REVOKE ALL ON app.platform_owner_guard FROM kredit_app, kredit_worker;
REVOKE UPDATE, DELETE, TRUNCATE ON app.platform_settings_history FROM kredit_app;
REVOKE INSERT, UPDATE, DELETE, TRUNCATE ON app.platform_settings_history FROM kredit_worker;
REVOKE INSERT, UPDATE, DELETE, TRUNCATE ON app.platform_governance, app.platform_settings FROM kredit_worker;
REVOKE TRUNCATE ON app.platform_role_assignments, app.users FROM kredit_app, kredit_worker;
GRANT EXECUTE ON FUNCTION app.is_platform_owner(uuid), app.current_governance_mode() TO kredit_app;
GRANT EXECUTE ON FUNCTION app.current_governance_mode() TO kredit_worker;
GRANT EXECUTE ON FUNCTION app.public_payment_receipt(uuid) TO kredit_app;

-- Supplier messaging consent is buyer-originated, append-only evidence.
REVOKE UPDATE, DELETE, TRUNCATE ON app.relationship_consents FROM kredit_app;
REVOKE INSERT, UPDATE, DELETE, TRUNCATE ON app.relationship_consents FROM kredit_worker;

-- Buyer-owned supplier names for reminder and consent controls.
GRANT EXECUTE ON FUNCTION app.buyer_suppliers() TO kredit_app;
GRANT EXECUTE ON FUNCTION app.has_own_relationship_consent(uuid,text) TO kredit_app;

-- Committed notification discovery returns identifiers only; subsequent reads use RLS.
GRANT EXECUTE ON FUNCTION app.notification_event_identity(UUID,TEXT,TEXT,JSONB) TO kredit_worker;

-- System recognition is separate from buyer-originated evidence.
GRANT SELECT ON app.system_acceptances TO kredit_app;
GRANT SELECT,INSERT ON app.system_acceptances TO kredit_worker;
REVOKE INSERT,UPDATE,DELETE ON app.system_acceptances FROM kredit_app;
REVOKE UPDATE,DELETE ON app.system_acceptances FROM kredit_worker;
GRANT EXECUTE ON FUNCTION app.deemed_acceptance_evidence(uuid,bigint),app.deemed_acceptance_candidates(bigint) TO kredit_worker;

GRANT EXECUTE ON FUNCTION app.document_object_is_orphan(text) TO kredit_worker;

GRANT EXECUTE ON FUNCTION app.system_acceptance_settings() TO kredit_worker;

GRANT SELECT,INSERT,UPDATE ON app.customer_registration_attempts TO kredit_app;
REVOKE INSERT,UPDATE,DELETE ON app.customer_registration_attempts FROM kredit_worker;
