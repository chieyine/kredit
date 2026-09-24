-- Apply with a migration/admin role after schema migrations. This file is a
-- deployment template; it intentionally does not contain passwords.
-- This script owns its transaction; run it standalone after migrations.
-- A later failure must not commit any earlier broad baseline grants.
BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        CREATE ROLE kredit_app NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_worker') THEN
        CREATE ROLE kredit_worker NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_migrator') THEN
        CREATE ROLE kredit_migrator NOLOGIN NOSUPERUSER CREATEDB NOCREATEROLE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_backup') THEN
        CREATE ROLE kredit_backup NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE BYPASSRLS;
    END IF;
    -- Do not silently reuse a privileged role left by an earlier deployment.
    -- Refuse drift rather than changing unrelated owner or backup privileges.
    IF EXISTS (
        SELECT 1 FROM pg_roles
        WHERE rolname IN ('kredit_app','kredit_worker')
          AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls)
    ) THEN
        RAISE EXCEPTION 'kredit_app and kredit_worker must be unprivileged NOLOGIN roles; correct role drift before applying grants';
    END IF;
END
$$;

REVOKE ALL ON SCHEMA app FROM PUBLIC;
GRANT USAGE ON SCHEMA app TO kredit_app, kredit_worker, kredit_backup;

-- Existing application tables keep their repository-level baseline grants;
-- row-level security is the tenant authorization boundary.
--
-- The ALTER DEFAULT PRIVILEGES below withholds grants from tables created
-- after this file runs. That property does not survive a re-run: every deploy
-- path applies this file AFTER migrations, and the blanket GRANT then sweeps
-- up whatever those migrations created. Demonstrated on a live database - a
-- table created after roles.sql has no grants, and holds INSERT/SELECT/UPDATE
-- the moment roles.sql runs again.
--
-- So the review, not this file, is what keeps new data from silently becoming
-- reachable: scripts/db-grant-check.sh diffs the runtime roles' actual reach
-- against docs/compliance/runtime-grant-inventory.txt and fails CI on any
-- change. A table appearing there is a decision someone has to approve.
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
    IF to_regprocedure('app.recovery_queue_metrics()') IS NOT NULL THEN
        REVOKE ALL ON FUNCTION app.recovery_queue_metrics() FROM PUBLIC;
        GRANT EXECUTE ON FUNCTION app.recovery_queue_metrics() TO kredit_app, kredit_worker;
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

GRANT EXECUTE ON FUNCTION app.financial_change_identity(text,boolean),app.lock_transfer_recipient(uuid),app.financial_review_differences(boolean) TO kredit_app;
GRANT EXECUTE ON FUNCTION app.financial_review_differences(boolean) TO kredit_worker;

GRANT EXECUTE ON FUNCTION app.admin_user_directory(text,integer,uuid),app.admin_organization_directory(text,integer,uuid),app.admin_audit_directory(text,integer,uuid),app.admin_team_directory(uuid),app.admin_money_summary(uuid),app.admin_money_activity(integer,uuid) TO kredit_app;

GRANT EXECUTE ON FUNCTION app.drawdown_expiry_tenants(text,integer) TO kredit_worker;

GRANT EXECUTE ON FUNCTION app.pilot_metric(timestamptz,timestamptz,text,text),app.pilot_reconciliation(timestamptz,timestamptz,text) TO kredit_app;

GRANT EXECUTE ON FUNCTION app.collection_mandate_capacity(uuid) TO kredit_app,kredit_worker;

GRANT EXECUTE ON FUNCTION app.recovery_account(text,text) TO kredit_app;
GRANT SELECT,INSERT,UPDATE ON app.buyer_verification_intents,app.mandate_authorization_intents TO kredit_app;
REVOKE INSERT,UPDATE,DELETE ON app.buyer_verification_intents,app.mandate_authorization_intents FROM kredit_worker;
REVOKE DELETE ON app.buyer_verification_intents,app.mandate_authorization_intents FROM kredit_app;

GRANT EXECUTE ON FUNCTION app.provider_work() TO kredit_app;

GRANT SELECT,INSERT,UPDATE ON app.message_submissions TO kredit_app,kredit_worker;
REVOKE DELETE ON app.message_submissions FROM kredit_app,kredit_worker;

GRANT SELECT,INSERT ON app.message_routes TO kredit_app,kredit_worker;
REVOKE UPDATE,DELETE ON app.message_routes FROM kredit_app,kredit_worker;


GRANT SELECT,INSERT,UPDATE ON app.runtime_process_status TO kredit_app,kredit_worker;
REVOKE DELETE ON app.runtime_process_status FROM kredit_app,kredit_worker;

GRANT SELECT, INSERT, UPDATE ON app.settlement_registrations TO kredit_app;
REVOKE DELETE ON app.settlement_registrations FROM kredit_app;
REVOKE INSERT,UPDATE,DELETE ON app.settlement_registrations FROM kredit_worker;
GRANT EXECUTE ON FUNCTION app.settlement_registration_review() TO kredit_app;

GRANT SELECT,INSERT ON app.fee_invoices,app.fee_invoice_lines TO kredit_app,kredit_worker;
REVOKE UPDATE,DELETE ON app.fee_invoices,app.fee_invoice_lines,app.fee_invoice_receipts FROM kredit_app,kredit_worker;
GRANT SELECT,INSERT ON app.fee_invoice_receipts TO kredit_app;
REVOKE INSERT ON app.fee_invoice_receipts FROM kredit_worker;
GRANT SELECT,INSERT,UPDATE ON app.invoice_billing_approvals TO kredit_app;
REVOKE DELETE ON app.invoice_billing_approvals FROM kredit_app;
REVOKE INSERT,UPDATE,DELETE ON app.invoice_billing_approvals FROM kredit_worker;
GRANT EXECUTE ON FUNCTION app.invoice_billing_work() TO kredit_worker;
GRANT EXECUTE ON FUNCTION app.invoice_billing_review() TO kredit_app;

-- Interactive identity evidence is written by authenticated API flows only.
GRANT SELECT, INSERT, UPDATE ON app.native_identity_sessions TO kredit_app;
REVOKE INSERT, UPDATE, DELETE ON app.native_identity_sessions FROM kredit_worker;
-- Frozen payout destinations and recorded bank evidence are append-only.
GRANT SELECT,INSERT ON app.collection_settlement_routes TO kredit_app,kredit_worker;
REVOKE UPDATE,DELETE ON app.collection_settlement_routes FROM kredit_app,kredit_worker;
GRANT SELECT,INSERT ON app.seller_settlement_receipts TO kredit_app;
GRANT SELECT ON app.seller_settlement_receipts TO kredit_worker;
REVOKE UPDATE,DELETE ON app.seller_settlement_receipts FROM kredit_app;
REVOKE INSERT,UPDATE,DELETE ON app.seller_settlement_receipts FROM kredit_worker;

-- Fee allocation and bank evidence retain tenant scopes and append-only facts.
GRANT SELECT,INSERT ON app.split_fee_allocations TO kredit_app,kredit_worker;
REVOKE UPDATE,DELETE ON app.split_fee_allocations FROM kredit_app,kredit_worker;
GRANT SELECT,INSERT,UPDATE ON app.fee_authorizations,app.fee_debits TO kredit_app;
REVOKE DELETE ON app.fee_authorizations,app.fee_debits FROM kredit_app,kredit_worker;
REVOKE INSERT,UPDATE ON app.fee_authorizations FROM kredit_worker;
GRANT SELECT,UPDATE(approved_at) ON app.fee_authorizations TO kredit_worker;
GRANT SELECT,INSERT,UPDATE ON app.fee_debits TO kredit_worker;
GRANT EXECUTE ON FUNCTION app.fee_billing_work() TO kredit_worker;
GRANT SELECT,INSERT ON app.fee_bank_receipts TO kredit_app;
GRANT SELECT ON app.fee_bank_receipts TO kredit_worker;
REVOKE UPDATE,DELETE ON app.fee_bank_receipts FROM kredit_app;
REVOKE INSERT,UPDATE,DELETE ON app.fee_bank_receipts FROM kredit_worker;

GRANT SELECT ON app.native_identity_history TO kredit_app;
REVOKE INSERT,UPDATE,DELETE ON app.native_identity_history FROM kredit_app,kredit_worker;

GRANT EXECUTE ON FUNCTION app.fee_notice_scope(text,text,text) TO kredit_app,kredit_worker;

-- Personal purchases retain their own access policies and immutable money history.
GRANT SELECT,INSERT,UPDATE ON app.consumer_sales,app.consumer_settings TO kredit_app;
GRANT SELECT,INSERT ON app.consumer_events TO kredit_app;
REVOKE UPDATE,DELETE ON app.consumer_events FROM kredit_app,kredit_worker;
GRANT SELECT ON app.consumer_sales,app.consumer_events TO kredit_worker;
GRANT EXECUTE ON FUNCTION app.consumer_contact_matches(text,text),app.consumer_seller_role(uuid,text[]) TO kredit_app,kredit_worker;
GRANT EXECUTE ON FUNCTION app.consumer_reminder_work() TO kredit_worker;

GRANT EXECUTE ON FUNCTION app.has_admin_role(uuid,text[]) TO kredit_worker;

GRANT EXECUTE ON FUNCTION app.consumer_retailer_ready(uuid) TO kredit_app;

GRANT SELECT,INSERT,UPDATE ON app.consumer_restrictions TO kredit_app;

-- DSA programme: immutable earnings, owner-only payouts, scoped worker refresh.
REVOKE ALL ON app.dsa_program,app.dsa_agents,app.dsa_referrals,app.dsa_earnings,app.dsa_payouts FROM kredit_app,kredit_worker;
GRANT SELECT ON app.dsa_program,app.dsa_agents,app.dsa_referrals,app.dsa_earnings,app.dsa_payouts TO kredit_app,kredit_worker;
GRANT INSERT,UPDATE ON app.dsa_agents,app.dsa_referrals,app.dsa_payouts TO kredit_app;
GRANT INSERT ON app.dsa_earnings TO kredit_app,kredit_worker;
GRANT UPDATE ON app.dsa_program TO kredit_app;
GRANT UPDATE ON app.dsa_agents,app.dsa_referrals TO kredit_worker;
GRANT EXECUTE ON FUNCTION app.dsa_code(text),app.dsa_claim(uuid,text) TO kredit_app;
GRANT EXECUTE ON FUNCTION app.dsa_facts(uuid,timestamptz,timestamptz) TO kredit_app,kredit_worker;

GRANT EXECUTE ON FUNCTION app.dsa_agent_active(uuid) TO kredit_app;

-- Tenant-scoped recovery and notification repositories (migrations 159/163).
GRANT EXECUTE ON FUNCTION app.recovery_subject(uuid) TO kredit_app;
GRANT EXECUTE ON FUNCTION app.notification_due_work(integer),app.notification_work_subject(uuid),app.notification_receipt_work(text,integer) TO kredit_worker;
GRANT EXECUTE ON FUNCTION app.notification_receipt_subject(text,text,text),app.notification_meta_candidates(text) TO kredit_app,kredit_worker;

GRANT EXECUTE ON FUNCTION app.isolation_operations_counts(),app.dispute_reference_lookup(text),app.notification_recovery_subject(uuid) TO kredit_app;

GRANT EXECUTE ON FUNCTION app.ensure_business_workspace(uuid) TO kredit_app;

-- Durable network imports and independent credit-offer review.
GRANT SELECT,INSERT,UPDATE ON app.distributor_import_batches TO kredit_app;
GRANT SELECT,INSERT ON app.distributor_import_rows TO kredit_app;
REVOKE UPDATE,DELETE ON app.distributor_import_rows FROM kredit_app,kredit_worker;
REVOKE ALL ON app.distributor_import_batches,app.distributor_import_rows FROM kredit_worker;
GRANT EXECUTE ON FUNCTION app.purchasing_authority_current(uuid,uuid),app.request_authority_current(uuid),app.obligation_authority_current(uuid),app.lock_purchasing_authority(uuid,uuid),app.lock_request_authority(uuid),app.lock_obligation_authority(uuid),app.guard_current_purchasing_authority() TO kredit_app,kredit_worker;
GRANT SELECT,INSERT,UPDATE ON app.business_credit_controls,app.credit_offer_approvals TO kredit_app;
REVOKE ALL ON app.business_credit_controls,app.credit_offer_approvals FROM kredit_worker;
REVOKE ALL ON app.business_credit_control_history FROM kredit_app,kredit_worker;
GRANT SELECT ON app.business_credit_control_history TO kredit_app;

GRANT SELECT,INSERT,UPDATE ON app.credit_reviewer_limits TO kredit_app;
REVOKE DELETE ON app.credit_reviewer_limits FROM kredit_app;
REVOKE ALL ON app.credit_reviewer_limits,app.credit_reviewer_limit_history FROM kredit_worker;
REVOKE ALL ON app.credit_reviewer_limit_history FROM kredit_app;
GRANT SELECT ON app.credit_reviewer_limit_history TO kredit_app;

GRANT SELECT,INSERT,UPDATE ON app.business_branches,app.partner_assignments TO kredit_app;
REVOKE ALL ON app.business_branches,app.partner_assignments,app.network_operation_history FROM kredit_worker;
REVOKE ALL ON app.network_operation_history FROM kredit_app;
GRANT SELECT ON app.network_operation_history TO kredit_app;

-- Explicit business purchasing delegation; immutable history and no worker access.
REVOKE ALL ON app.purchasing_delegations,app.purchasing_delegation_history FROM kredit_app,kredit_worker;
GRANT SELECT,INSERT,UPDATE ON app.purchasing_delegations TO kredit_app;
GRANT SELECT ON app.purchasing_delegation_history TO kredit_app;
GRANT EXECUTE ON FUNCTION app.can_purchase(uuid,text,bigint),app.purchase_request_read(uuid),app.lock_purchase_permission(uuid,text,bigint) TO kredit_app,kredit_worker;

GRANT EXECUTE ON FUNCTION app.purchase_obligation_read(uuid) TO kredit_app,kredit_worker;

-- Branch boundaries narrow existing tenant rights; scope history is immutable.
REVOKE ALL ON app.member_branch_scopes,app.member_branch_scope_history FROM kredit_app,kredit_worker;
GRANT SELECT,INSERT,UPDATE ON app.member_branch_scopes TO kredit_app;
GRANT SELECT ON app.member_branch_scope_history TO kredit_app;
GRANT EXECUTE ON FUNCTION app.branch_scope_all(uuid),app.branch_customer_access(uuid,uuid),app.branch_credit_access(uuid),app.branch_obligation_access(uuid),app.branch_line_access(uuid),app.branch_drawdown_access(uuid) TO kredit_app,kredit_worker;

-- Blueprint redesign extensions (migrations 195 - 199).
-- 195: Canonical organization consolidation functions
GRANT EXECUTE ON FUNCTION app.organization_purchasing_profile(uuid), app.purchasing_profile_organization(uuid), app.can_purchase_organization(uuid, text, bigint) TO kredit_app, kredit_worker;

-- 197: Independent drawdown approvals
REVOKE ALL ON app.tradeline_drawdown_approvals FROM kredit_app, kredit_worker;
GRANT SELECT, INSERT, UPDATE ON app.tradeline_drawdown_approvals TO kredit_app, kredit_worker;
GRANT EXECUTE ON FUNCTION app.drawdown_approval_proposal(app.drawdowns), app.drawdown_approval_fingerprint(app.drawdowns) TO kredit_app, kredit_worker;

-- 198: Item-level order lifecycles, shipments, delivery receipts, and credit notes
REVOKE ALL ON app.order_line_items, app.order_shipments, app.order_shipment_items, app.order_delivery_receipts, app.order_credit_notes FROM kredit_app, kredit_worker;
GRANT SELECT, INSERT, UPDATE, DELETE ON app.order_line_items, app.order_shipments, app.order_shipment_items, app.order_delivery_receipts, app.order_credit_notes TO kredit_app, kredit_worker;

-- 199: Partner terms and opening balance import batches
REVOKE ALL ON app.partner_terms_import_batches, app.partner_terms_import_rows FROM kredit_app, kredit_worker;
GRANT SELECT, INSERT, UPDATE ON app.partner_terms_import_batches, app.partner_terms_import_rows TO kredit_app, kredit_worker;
REVOKE ALL ON FUNCTION app.order_supplier_authorized(uuid,text[]),app.order_evidence_visible(uuid),app.apply_order_shipment_item(),app.complete_order_receipt(),app.guard_order_credit_note(),app.guard_terms_import_review(),app.guard_terms_import_row() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.order_supplier_authorized(uuid,text[]),app.order_evidence_visible(uuid) TO kredit_app;
REVOKE ALL ON app.order_line_items,app.order_shipments,app.order_shipment_items,app.order_delivery_receipts,app.order_credit_notes FROM kredit_app,kredit_worker;
GRANT SELECT,INSERT ON app.order_line_items,app.order_shipments,app.order_shipment_items,app.order_delivery_receipts,app.order_credit_notes TO kredit_app;
GRANT UPDATE(status,approved_by,approved_at) ON app.order_credit_notes TO kredit_app;
REVOKE ALL ON app.partner_terms_import_batches,app.partner_terms_import_rows FROM kredit_app,kredit_worker;
GRANT SELECT,INSERT ON app.partner_terms_import_batches,app.partner_terms_import_rows TO kredit_app;
GRANT UPDATE(state,approved_by,approved_at,cancelled_by,cancelled_at) ON app.partner_terms_import_batches TO kredit_app;

-- Read-only credit-note provenance for tenant-scoped balance reconciliation.
GRANT SELECT ON app.order_credit_notes TO kredit_worker;
-- The reconciliation read path requires both nested invoker predicates.
-- This read-only predicate does not confer evidence mutation authority.
GRANT EXECUTE ON FUNCTION app.order_evidence_visible(uuid),app.order_supplier_authorized(uuid,text[]) TO kredit_worker;

-- Per-boot heartbeats; RLS limits deletion to this process kind's stale rows.
GRANT SELECT,INSERT,UPDATE,DELETE ON app.runtime_process_instances TO kredit_app,kredit_worker;

-- Most SECURITY DEFINER functions pin search_path without naming pg_temp, and
-- PostgreSQL then searches pg_temp first for relations. A session that can
-- create temporary tables could shadow an unqualified table inside such a
-- function. No runtime code, job queue or backup uses temporary tables, so
-- the privilege is withdrawn from every non-owner role instead.
DO $$
BEGIN
    EXECUTE format('REVOKE TEMPORARY ON DATABASE %I FROM PUBLIC', current_database());
END
$$;

COMMIT;
