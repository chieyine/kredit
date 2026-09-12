# File-by-file audit register — 12 September 2026

**940 files inspected. No agents or tests used. Findings remain open.**

This register covers every tracked file in the supplied working tree, including source, tests, migrations, infrastructure, documentation and assets. Every application source file was read directly. Historical structured files were parsed completely and their distinct semantic records/diagnostics reviewed; archived patch provenance was inspected against its current source files. Images were opened individually. Generated metadata inspection is not a manual review of third-party dependency source.

A finding link identifies a participating file; it does not mean every linked file is defective or should change. “No separate finding” means no additional confirmed defect was recorded in this static pass, not that the file passed a test. Symbol names identify the reviewed file content; the JSON register retains the complete extracted declaration list and earlier review notes.

Secrets, installed dependencies, build caches, Git internals and local service data are excluded from source-body inspection. Local SSH files were examined by filename/permissions only (F039); their contents were not opened. The .gitignore changed externally during this audit and was reread; its old and current hashes are retained. No other inventoried source changed.

See [findings](FINDINGS.md), [provider research](PROVIDERS.md), [machine-readable register](file-review-index.json), and [CSV](file-review.csv).


## .github/workflows

| File | Reviewed content | Result |
| --- | --- | --- |
| [.github/workflows/ci.yml](</Users/macbookpro/Documents/Kredit.com/.github/workflows/ci.yml>) | ci.yml: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [.github/workflows/phase2-tenant-isolation.yml](</Users/macbookpro/Documents/Kredit.com/.github/workflows/phase2-tenant-isolation.yml>) | phase2-tenant-isolation.yml: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [.github/workflows/phase3-financial-proof.yml](</Users/macbookpro/Documents/Kredit.com/.github/workflows/phase3-financial-proof.yml>) | phase3-financial-proof.yml: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [.github/workflows/phase4-provider-verification.yml](</Users/macbookpro/Documents/Kredit.com/.github/workflows/phase4-provider-verification.yml>) | phase4-provider-verification.yml: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [.github/workflows/phase5-production-assurance.yml](</Users/macbookpro/Documents/Kredit.com/.github/workflows/phase5-production-assurance.yml>) | phase5-production-assurance.yml: deployment, boundaries, failure handling and referenced paths | [F036](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:153) |
| [.github/workflows/phase6-context-audit.yml](</Users/macbookpro/Documents/Kredit.com/.github/workflows/phase6-context-audit.yml>) | phase6-context-audit.yml: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [.github/workflows/product-audit.yml](</Users/macbookpro/Documents/Kredit.com/.github/workflows/product-audit.yml>) | product-audit.yml: deployment, boundaries, failure handling and referenced paths | No separate finding |

## Root files

| File | Reviewed content | Result |
| --- | --- | --- |
| [.dockerignore](</Users/macbookpro/Documents/Kredit.com/.dockerignore>) | .dockerignore: deployment, boundaries, failure handling and referenced paths | [F039](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:165) |
| [.env.example](</Users/macbookpro/Documents/Kredit.com/.env.example>) | .env.example: configuration/data and current consumers | No separate finding |
| [.gitignore](</Users/macbookpro/Documents/Kredit.com/.gitignore>) | .gitignore: deployment, boundaries, failure handling and referenced paths | [F039](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:165) |
| [.go-version](</Users/macbookpro/Documents/Kredit.com/.go-version>) | .go-version: configuration/data and current consumers | No separate finding |
| [.golangci.yml](</Users/macbookpro/Documents/Kredit.com/.golangci.yml>) | .golangci.yml: configuration/data and current consumers | No separate finding |
| [.node-version](</Users/macbookpro/Documents/Kredit.com/.node-version>) | .node-version: configuration/data and current consumers | No separate finding |
| [.pnpm-version](</Users/macbookpro/Documents/Kredit.com/.pnpm-version>) | .pnpm-version: configuration/data and current consumers | No separate finding |
| [CHANGELOG.md](</Users/macbookpro/Documents/Kredit.com/CHANGELOG.md>) | Changelog: source consistency and evidence limits | No separate finding |
| [IMPLEMENTATION_PLAN.md](</Users/macbookpro/Documents/Kredit.com/IMPLEMENTATION_PLAN.md>) | Kredit Production V1 Implementation Plan: source consistency and evidence limits | No separate finding |
| [IMPLEMENTATION_STATUS.md](</Users/macbookpro/Documents/Kredit.com/IMPLEMENTATION_STATUS.md>) | Implementation Status: source consistency and evidence limits | No separate finding |
| [KREDIT-CODE-AUDIT.md](</Users/macbookpro/Documents/Kredit.com/KREDIT-CODE-AUDIT.md>) | Kredit — full platform code audit: source consistency and evidence limits | No separate finding |
| [KREDIT-LAUNCH-HANDOVER.md](</Users/macbookpro/Documents/Kredit.com/KREDIT-LAUNCH-HANDOVER.md>) | Kredit — what changed, and what you must run before you deploy: source consistency and evidence limits | No separate finding |
| [KREDIT-TEST-FIXES-ROUND-2.md](</Users/macbookpro/Documents/Kredit.com/KREDIT-TEST-FIXES-ROUND-2.md>) | Round two: the ten remaining browser failures: source consistency and evidence limits | No separate finding |
| [KREDIT-TEST-FIXES.md](</Users/macbookpro/Documents/Kredit.com/KREDIT-TEST-FIXES.md>) | What the first full test run found, and what I changed: source consistency and evidence limits | No separate finding |
| [Kredit_Local_Launch_Master_Prompt_v2_2026-09-07.md](</Users/macbookpro/Documents/Kredit.com/Kredit_Local_Launch_Master_Prompt_v2_2026-09-07.md>) | Kredit — Local launch implementation and full design/content overhaul: source consistency and evidence limits | No separate finding |
| [README.local.md](</Users/macbookpro/Documents/Kredit.com/README.local.md>) | Local development quick start: source consistency and evidence limits | No separate finding |
| [README.md](</Users/macbookpro/Documents/Kredit.com/README.md>) | Kredit: source consistency and evidence limits | [F002](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:13) |
| [SECURITY.md](</Users/macbookpro/Documents/Kredit.com/SECURITY.md>) | Security policy: source consistency and evidence limits | No separate finding |
| [Taskfile.yml](</Users/macbookpro/Documents/Kredit.com/Taskfile.yml>) | Taskfile.yml: deployment, boundaries, failure handling and referenced paths | [F037](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:157) |
| [docker-compose.yml](</Users/macbookpro/Documents/Kredit.com/docker-compose.yml>) | docker-compose.yml: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [go.mod](</Users/macbookpro/Documents/Kredit.com/go.mod>) | Dependency/toolchain pins, integrity, imports and available advisory evidence | No separate finding |
| [go.sum](</Users/macbookpro/Documents/Kredit.com/go.sum>) | Dependency/toolchain pins, integrity, imports and available advisory evidence | No separate finding |
| [package.json](</Users/macbookpro/Documents/Kredit.com/package.json>) | Dependency/toolchain pins, integrity, imports and available advisory evidence | No separate finding |
| [pnpm-lock.yaml](</Users/macbookpro/Documents/Kredit.com/pnpm-lock.yaml>) | Dependency/toolchain pins, integrity, imports and available advisory evidence | No separate finding |
| [pnpm-workspace.yaml](</Users/macbookpro/Documents/Kredit.com/pnpm-workspace.yaml>) | pnpm-workspace.yaml: configuration/data and current consumers | No separate finding |
| [redocly.yaml](</Users/macbookpro/Documents/Kredit.com/redocly.yaml>) | redocly.yaml: configuration/data and current consumers | No separate finding |

## api

| File | Reviewed content | Result |
| --- | --- | --- |
| [api/openapi.yaml](</Users/macbookpro/Documents/Kredit.com/api/openapi.yaml>) | openapi.yaml: configuration/data and current consumers | No separate finding |

## cmd/api

| File | Reviewed content | Result |
| --- | --- | --- |
| [cmd/api/main.go](</Users/macbookpro/Documents/Kredit.com/cmd/api/main.go>) | main.go: configuration/data and current consumers; main, runSelfHealthcheck, runSelfHealthcheckWithClient | No separate finding |
| [cmd/api/main_test.go](</Users/macbookpro/Documents/Kredit.com/cmd/api/main_test.go>) | main_test: fixtures, assertions and production equivalence; RoundTrip, TestRunSelfHealthcheck, TestHealthcheckRejectsRedirectAndUnhealthyStatus | No separate finding |

## cmd/bootstrap-owner

| File | Reviewed content | Result |
| --- | --- | --- |
| [cmd/bootstrap-owner/main.go](</Users/macbookpro/Documents/Kredit.com/cmd/bootstrap-owner/main.go>) | main.go: configuration/data and current consumers; main, eligibleOwner | No separate finding |
| [cmd/bootstrap-owner/main_test.go](</Users/macbookpro/Documents/Kredit.com/cmd/bootstrap-owner/main_test.go>) | main_test: fixtures, assertions and production equivalence; TestEligibleOwnerRequiresActiveAccountAndFreshSession | No separate finding |

## cmd/configcheck

| File | Reviewed content | Result |
| --- | --- | --- |
| [cmd/configcheck/main.go](</Users/macbookpro/Documents/Kredit.com/cmd/configcheck/main.go>) | main.go: configuration/data and current consumers; main | No separate finding |
| [cmd/configcheck/main_test.go](</Users/macbookpro/Documents/Kredit.com/cmd/configcheck/main_test.go>) | main_test: fixtures, assertions and production equivalence; TestConfigCheckFailsClosedWithoutLaunchEvidence | No separate finding |

## cmd/migrate

| File | Reviewed content | Result |
| --- | --- | --- |
| [cmd/migrate/main.go](</Users/macbookpro/Documents/Kredit.com/cmd/migrate/main.go>) | main.go: configuration/data and current consumers; main, dbpool, extractUpSQL | No separate finding |
| [cmd/migrate/main_test.go](</Users/macbookpro/Documents/Kredit.com/cmd/migrate/main_test.go>) | main_test: fixtures, assertions and production equivalence; TestExtractUpSQLStopsBeforeDownSection | No separate finding |

## cmd/provider-simulator

| File | Reviewed content | Result |
| --- | --- | --- |
| [cmd/provider-simulator/main.go](</Users/macbookpro/Documents/Kredit.com/cmd/provider-simulator/main.go>) | main.go: configuration/data and current consumers; main, newSimulator, handler, createVerification and 20 other declarations | No separate finding |
| [cmd/provider-simulator/main_test.go](</Users/macbookpro/Documents/Kredit.com/cmd/provider-simulator/main_test.go>) | main_test: fixtures, assertions and production equivalence; TestSimulatorCoversProviderContractsAndIdempotency, TestCancelMandateMalformedJSONWritesOneError, TestSimulatorRejectsOversizedAndTrailingJSON, TestSimulatorHealthcheckUsesConfiguredAddress and 3 other declarations | No separate finding |

## cmd/reconcile

| File | Reviewed content | Result |
| --- | --- | --- |
| [cmd/reconcile/main.go](</Users/macbookpro/Documents/Kredit.com/cmd/reconcile/main.go>) | main.go: configuration/data and current consumers; main, databaseURLFromEnv, fail | No separate finding |
| [cmd/reconcile/main_test.go](</Users/macbookpro/Documents/Kredit.com/cmd/reconcile/main_test.go>) | main_test: fixtures, assertions and production equivalence; TestDatabaseURLFromEnv | No separate finding |

## cmd/seed

| File | Reviewed content | Result |
| --- | --- | --- |
| [cmd/seed/main.go](</Users/macbookpro/Documents/Kredit.com/cmd/seed/main.go>) | main.go: configuration/data and current consumers; main | No separate finding |
| [cmd/seed/main_test.go](</Users/macbookpro/Documents/Kredit.com/cmd/seed/main_test.go>) | main_test: fixtures, assertions and production equivalence; TestSeedFileExists | No separate finding |

## cmd/worker

| File | Reviewed content | Result |
| --- | --- | --- |
| [cmd/worker/main.go](</Users/macbookpro/Documents/Kredit.com/cmd/worker/main.go>) | main.go: configuration/data and current consumers; startHealthServer, healthHandler, main, enqueueDueNotifications and 4 other declarations | No separate finding |
| [cmd/worker/main_test.go](</Users/macbookpro/Documents/Kredit.com/cmd/worker/main_test.go>) | main_test: fixtures, assertions and production equivalence; TestHealthHandlerExposesLivenessAndReadiness, TestHealthHandlerRejectsMutationMethods, TestHealthServerReportsListenFailure | No separate finding |

## db/migrations

| File | Reviewed content | Result |
| --- | --- | --- |
| [db/migrations/001_initial.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/001_initial.sql>) | Migration 001: initial; Up/Down, policies and later definitions; public.uuidv7, schema_migrations, app_meta | No separate finding |
| [db/migrations/002_milestone1_auth_org.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/002_milestone1_auth_org.sql>) | Migration 002: milestone1 auth org; Up/Down, policies and later definitions; app.current_user_id, app.current_organization_id, app.users, app.sessions and 13 other declarations | No separate finding |
| [db/migrations/003_milestone2_buyers_identity.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/003_milestone2_buyers_identity.sql>) | Migration 003: milestone2 buyers identity; Up/Down, policies and later definitions; app.persons, app.businesses, app.business_representatives, app.verification_cases and 10 other declarations | No separate finding |
| [db/migrations/004_milestone3_credit_ledger.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/004_milestone3_credit_ledger.sql>) | Migration 004: milestone3 credit ledger; Up/Down, policies and later definitions; app.credit_requests, app.agreement_versions, app.agreement_acceptances, app.mandates and 10 other declarations | No separate finding |
| [db/migrations/005_milestone4_payments.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/005_milestone4_payments.sql>) | Migration 005: milestone4 payments; Up/Down, policies and later definitions; app.payments, app.payment_allocations, app.fees, app.settlement_events and 5 other declarations | No separate finding |
| [db/migrations/006_milestone5_schedules.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/006_milestone5_schedules.sql>) | Migration 006: milestone5 schedules; Up/Down, policies and later definitions; app.repayment_schedules, app.schedule_items, schedule_supplier_access, schedule_buyer_access and 1 other declarations | No separate finding |
| [db/migrations/007_milestone6_trade_lines.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/007_milestone6_trade_lines.sql>) | Migration 007: milestone6 trade lines; Up/Down, policies and later definitions; app.trade_lines, app.drawdowns, app.drawdown_reservations, trade_line_supplier_access and 3 other declarations | No separate finding |
| [db/migrations/008_milestone7_collections.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/008_milestone7_collections.sql>) | Migration 008: milestone7 collections; Up/Down, policies and later definitions; app.collection_reservations, app.collection_attempts, app.collection_provider_events, collection_reservation_supplier_access and 2 other declarations | No separate finding |
| [db/migrations/009_milestone8_disputes_operations.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/009_milestone8_disputes_operations.sql>) | Migration 009: milestone8 disputes operations; Up/Down, policies and later definitions; app.disputes, app.dispute_evidence, app.dispute_decisions, app.operation_actions and 5 other declarations | No separate finding |
| [db/migrations/010_milestone9_notifications_whatsapp.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/010_milestone9_notifications_whatsapp.sql>) | Migration 010: milestone9 notifications whatsapp; Up/Down, policies and later definitions; app.notification_templates, app.notification_preferences, app.notifications, app.messaging_events and 3 other declarations | No separate finding |
| [db/migrations/011_milestone10_reporting_history.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/011_milestone10_reporting_history.sql>) | Migration 011: milestone10 reporting history; Up/Down, policies and later definitions; app.correction_requests, app.correction_decisions, app.analytics_events, correction_org_access and 2 other declarations | No separate finding |
| [db/migrations/012_milestone11_provider_adapter.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/012_milestone11_provider_adapter.sql>) | Migration 012: milestone11 provider adapter; Up/Down, policies and later definitions; app.provider_approvals, app.provider_events, app.provider_reconciliation_events, provider_approval_support_access and 2 other declarations | No separate finding |
| [db/migrations/013_milestone12_release_readiness.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/013_milestone12_release_readiness.sql>) | Migration 013: milestone12 release readiness; Up/Down, policies and later definitions; app.release_evidence, app.pilot_limit_configs, release_evidence_support_access, pilot_limits_support_access | No separate finding |
| [db/migrations/014_river_jobs_schema.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/014_river_jobs_schema.sql>) | Migration 014: river jobs schema; Up/Down, policies and later definitions | No separate finding |
| [db/migrations/015_missing_domain_tables.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/015_missing_domain_tables.sql>) | Migration 015: missing domain tables; Up/Down, policies and later definitions; app.relationship_consents, app.documents, app.support_cases, app.support_case_events and 3 other declarations | No separate finding |
| [db/migrations/016_financial_core_hardening.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/016_financial_core_hardening.sql>) | Migration 016: financial core hardening; Up/Down, policies and later definitions; app.current_user_id, app.current_organization_id, ledger.transactions, ledger.assert_balanced_transaction and 1 other declarations | No separate finding |
| [db/migrations/017_worker_operations.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/017_worker_operations.sql>) | Migration 017: worker operations; Up/Down, policies and later definitions; app.provider_webhook_inbox, app.job_dead_letters | No separate finding |
| [db/migrations/018_security_hardening.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/018_security_hardening.sql>) | Migration 018: security hardening; Up/Down, policies and later definitions; app.prevent_audit_event_mutation, audit_events_append_only, app.audit_events | No separate finding |
| [db/migrations/019_runtime_role_policies.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/019_runtime_role_policies.sql>) | Migration 019: runtime role policies; Up/Down, policies and later definitions; app.provider_webhook_inbox, provider_webhook_runtime_access, app.job_dead_letters, dead_letter_runtime_access and 9 other declarations | No separate finding |
| [db/migrations/020_auth_persistence_primitives.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/020_auth_persistence_primitives.sql>) | Migration 020: auth persistence primitives; Up/Down, policies and later definitions; app.otp_challenges, app.find_or_create_user, app.session_by_token_hash, memberships_tenant_isolation | No separate finding |
| [db/migrations/021_organization_persistence_primitives.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/021_organization_persistence_primitives.sql>) | Migration 021: organization persistence primitives; Up/Down, policies and later definitions; app.organization_count | No separate finding |
| [db/migrations/022_network_and_mandates.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/022_network_and_mandates.sql>) | Migration 022: network and mandates; Up/Down, policies and later definitions; app.trade_relationships, app.payment_mandates, app.mandate_events, trade_relationships_tenant_isolation and 2 other declarations | No separate finding |
| [db/migrations/023_buyer_persistence_primitives.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/023_buyer_persistence_primitives.sql>) | Migration 023: buyer persistence primitives; Up/Down, policies and later definitions; app.buyer_invitations, app.buyer_invitation_by_token_hash, app.business_count | No separate finding |
| [db/migrations/024_credit_aggregate_snapshots.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/024_credit_aggregate_snapshots.sql>) | Migration 024: credit aggregate snapshots; Up/Down, policies and later definitions; app.credit_aggregate_snapshots, credit_snapshot_tenant_access, app.credit_snapshot_by_id, app.credit_snapshot_by_obligation | No separate finding |
| [db/migrations/025_mandate_lookup_function.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/025_mandate_lookup_function.sql>) | Migration 025: mandate lookup function; Up/Down, policies and later definitions; app.payment_mandate_by_provider | No separate finding |
| [db/migrations/026_mandate_lookup_owner.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/026_mandate_lookup_owner.sql>) | Migration 026: mandate lookup owner; Up/Down, policies and later definitions; app.payment_mandate_by_provider | No separate finding |
| [db/migrations/027_credit_snapshot_tenant_functions.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/027_credit_snapshot_tenant_functions.sql>) | Migration 027: credit snapshot tenant functions; Up/Down, policies and later definitions; app.credit_snapshot_by_id, app.credit_snapshot_by_obligation | No separate finding |
| [db/migrations/028_outbox_processing_lease.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/028_outbox_processing_lease.sql>) | Migration 028: outbox processing lease; Up/Down, policies and later definitions; app.outbox_events | No separate finding |
| [db/migrations/029_runtime_domain_repository_policies.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/029_runtime_domain_repository_policies.sql>) | Migration 029: runtime domain repository policies; Up/Down, policies and later definitions; app.documents, document_runtime_access, app.support_cases, support_case_runtime_access and 2 other declarations | No separate finding |
| [db/migrations/030_schedule_repository_runtime_policy.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/030_schedule_repository_runtime_policy.sql>) | Migration 030: schedule repository runtime policy; Up/Down, policies and later definitions; app.repayment_schedules, schedule_runtime_access, app.schedule_items, schedule_item_runtime_access | No separate finding |
| [db/migrations/031_atomic_payment_repository.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/031_atomic_payment_repository.sql>) | Migration 031: atomic payment repository; Up/Down, policies and later definitions; app.payments, app.payment_allocations, payment_runtime_access, allocation_runtime_access and 3 other declarations | No separate finding |
| [db/migrations/032_supplier_customer_directory.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/032_supplier_customer_directory.sql>) | Migration 032: supplier customer directory; Up/Down, policies and later definitions; app.supplier_customers | No separate finding |
| [db/migrations/033_fix_ledger_balance_trigger.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/033_fix_ledger_balance_trigger.sql>) | Migration 033: fix ledger balance trigger; Up/Down, policies and later definitions; ledger.assert_balanced_transaction | No separate finding |
| [db/migrations/034_trade_line_runtime_policy.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/034_trade_line_runtime_policy.sql>) | Migration 034: trade line runtime policy; Up/Down, policies and later definitions; app.trade_lines, trade_line_runtime_access, app.drawdowns, drawdown_runtime_access and 2 other declarations | No separate finding |
| [db/migrations/035_collection_aggregate_repository.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/035_collection_aggregate_repository.sql>) | Migration 035: collection aggregate repository; Up/Down, policies and later definitions; app.collection_attempts, app.collection_aggregate_snapshots, app.collection_attempt_index, app.collection_reservations and 4 other declarations | No separate finding |
| [db/migrations/036_corrections_runtime_policy.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/036_corrections_runtime_policy.sql>) | Migration 036: corrections runtime policy; Up/Down, policies and later definitions; correction_runtime_access, correction_decision_runtime_access | No separate finding |
| [db/migrations/037_dispute_operation_runtime_policy.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/037_dispute_operation_runtime_policy.sql>) | Migration 037: dispute operation runtime policy; Up/Down, policies and later definitions; dispute_runtime_access, dispute_evidence_runtime_access, dispute_decision_runtime_access, operation_runtime_access | No separate finding |
| [db/migrations/038_notification_delivery_repository.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/038_notification_delivery_repository.sql>) | Migration 038: notification delivery repository; Up/Down, policies and later definitions; app.notifications, notification_runtime_access, notification_preference_runtime_access | No separate finding |
| [db/migrations/039_analytics_runtime_policy.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/039_analytics_runtime_policy.sql>) | Migration 039: analytics runtime policy; Up/Down, policies and later definitions; analytics_runtime_access | No separate finding |
| [db/migrations/040_messaging_runtime_policy.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/040_messaging_runtime_policy.sql>) | Migration 040: messaging runtime policy; Up/Down, policies and later definitions; messaging_runtime_access | No separate finding |
| [db/migrations/041_notification_delivery_recovery.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/041_notification_delivery_recovery.sql>) | Migration 041: notification delivery recovery; Up/Down, policies and later definitions; app.notifications | No separate finding |
| [db/migrations/042_platform_operations_roles.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/042_platform_operations_roles.sql>) | Migration 042: platform operations roles; Up/Down, policies and later definitions; app.platform_role_assignments, platform_role_runtime_access | No separate finding |
| [db/migrations/043_fix_auth_user_upsert_ambiguity.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/043_fix_auth_user_upsert_ambiguity.sql>) | Migration 043: fix auth user upsert ambiguity; Up/Down, policies and later definitions; app.find_or_create_user | No separate finding |
| [db/migrations/044_payment_sources_and_claims.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/044_payment_sources_and_claims.sql>) | Migration 044: payment sources and claims; Up/Down, policies and later definitions; app.payments, app.payment_claims, payment_claims_supplier_access, payment_claims_buyer_access and 1 other declarations | No separate finding |
| [db/migrations/045_trade_line_mandate_integrity.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/045_trade_line_mandate_integrity.sql>) | Migration 045: trade line mandate integrity; Up/Down, policies and later definitions; app.trade_lines, app.trade_line_mandate | No separate finding |
| [db/migrations/046_trade_line_drawdown_lifecycle.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/046_trade_line_drawdown_lifecycle.sql>) | Migration 046: trade line drawdown lifecycle; Up/Down, policies and later definitions; app.drawdowns, app.drawdown_reservations, app.drawdown_receipt_disputes | No separate finding |
| [db/migrations/047_supplier_onboarding_readiness.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/047_supplier_onboarding_readiness.sql>) | Migration 047: supplier onboarding readiness; Up/Down, policies and later definitions; app.sessions, app.supplier_onboarding_profiles, app.supplier_onboarding_revisions, supplier_onboarding_tenant_isolation and 2 other declarations | No separate finding |
| [db/migrations/048_user_control_and_privacy.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/048_user_control_and_privacy.sql>) | Migration 048: user control and privacy; Up/Down, policies and later definitions; app.notification_preferences, app.account_recovery_codes, app.account_recovery_rate_limits, app.account_recovery_requests and 25 other declarations | No separate finding |
| [db/migrations/049_operations_controls.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/049_operations_controls.sql>) | Migration 049: operations controls; Up/Down, policies and later definitions; app.users, app.provider_webhook_inbox, app.operations_commands, app.operations_command_events and 11 other declarations | No separate finding |
| [db/migrations/050_product_analytics.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/050_product_analytics.sql>) | Migration 050: product analytics; Up/Down, policies and later definitions; app.analytics_events, app.record_product_event, app.analytics_onboarding_event, app.analytics_invitation_event and 31 other declarations | No separate finding |
| [db/migrations/051_analytics_contract_completion.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/051_analytics_contract_completion.sql>) | Migration 051: analytics contract completion; Up/Down, policies and later definitions; app.analytics_credit_child_event, app.analytics_mandate_event, app.analytics_payment_mandate_event, app.analytics_obligation_event and 4 other declarations | No separate finding |
| [db/migrations/052_sweep_collection_safety.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/052_sweep_collection_safety.sql>) | Migration 052: sweep collection safety; Up/Down, policies and later definitions; app.payment_mandates, payment_mandate_worker_read, app.provider_customer_bindings, provider_customer_runtime and 10 other declarations | No separate finding |
| [db/migrations/053_payment_reversal_integrity.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/053_payment_reversal_integrity.sql>) | Migration 053: payment reversal integrity; Up/Down, policies and later definitions; app.guard_manual_payment_reservations | No separate finding |
| [db/migrations/054_collection_eligibility_lock.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/054_collection_eligibility_lock.sql>) | Migration 054: collection eligibility lock; Up/Down, policies and later definitions; app.guard_collection_reservation | No separate finding |
| [db/migrations/055_durable_financial_notices.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/055_durable_financial_notices.sql>) | Migration 055: durable financial notices; Up/Down, policies and later definitions; app.enqueue_financial_notice, app.capture_financial_notice, payment_notice, claim_notice and 2 other declarations | No separate finding |
| [db/migrations/056_financial_reconciliation.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/056_financial_reconciliation.sql>) | Migration 056: financial reconciliation; Up/Down, policies and later definitions; app.financial_discrepancies, app.financial_review_cases, app.financial_review_events, financial_review_cases_runtime and 2 other declarations | No separate finding |
| [db/migrations/057_collection_notice_delivery.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/057_collection_notice_delivery.sql>) | Migration 057: collection notice delivery; Up/Down, policies and later definitions; app.notification_delivery_receipts, notification_receipts_runtime, notification_receipts_immutable, app.collection_notice_key and 2 other declarations | No separate finding |
| [db/migrations/058_operations_intent_integrity.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/058_operations_intent_integrity.sql>) | Migration 058: operations intent integrity; Up/Down, policies and later definitions; app.operations_commands, settlement_runtime_access | No separate finding |
| [db/migrations/059_adjustment_notification_evidence.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/059_adjustment_notification_evidence.sql>) | Migration 059: adjustment notification evidence; Up/Down, policies and later definitions; app.capture_financial_notice, app.capture_adjustment_notice, adjustment_notice, financial_adjustment_actions_immutable | No separate finding |
| [db/migrations/060_provider_reversal_review.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/060_provider_reversal_review.sql>) | Migration 060: provider reversal review; Up/Down, policies and later definitions; app.financial_discrepancies | No separate finding |
| [db/migrations/061_business_policies.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/061_business_policies.sql>) | Migration 061: business policies; Up/Down, policies and later definitions; app.business_policy_defaults, app.business_policy_changes, app.business_policy_events, business_policy_defaults_runtime and 8 other declarations | No separate finding |
| [db/migrations/062_policy_enforcement_and_fee_terms.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/062_policy_enforcement_and_fee_terms.sql>) | Migration 062: policy enforcement and fee terms; Up/Down, policies and later definitions; app.credit_requests, app.drawdowns, app.guard_offer_policy, credit_offer_policy and 11 other declarations | No separate finding |
| [db/migrations/063_preserve_existing_commitments.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/063_preserve_existing_commitments.sql>) | Migration 063: preserve existing commitments; Up/Down, policies and later definitions; app.guard_offer_policy, app.guard_exposure_policy, goods_release_exposure_policy | No separate finding |
| [db/migrations/064_admin_workflows.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/064_admin_workflows.sql>) | Migration 064: admin workflows; Up/Down, policies and later definitions; app.platform_role_assignments, app.has_admin_role, app.is_active_policy_admin, app.admin_change_requests and 14 other declarations | No separate finding |
| [db/migrations/065_admin_attention_details.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/065_admin_attention_details.sql>) | Migration 065: admin attention details; Up/Down, policies and later definitions; app.admin_attention_details | No separate finding |
| [db/migrations/066_admin_support_review_queue.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/066_admin_support_review_queue.sql>) | Migration 066: admin support review queue; Up/Down, policies and later definitions; app.admin_review_queue | No separate finding |
| [db/migrations/067_canonical_phone_identifiers.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/067_canonical_phone_identifiers.sql>) | Migration 067: canonical phone identifiers; Up/Down, policies and later definitions; app.normalize_phone, app.users | No separate finding |
| [db/migrations/068_session_idle_and_mfa_throttle.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/068_session_idle_and_mfa_throttle.sql>) | Migration 068: session idle and mfa throttle; Up/Down, policies and later definitions; app.sessions, app.mfa_methods, app.session_by_token_hash, app.touch_session | No separate finding |
| [db/migrations/069_shared_authentication_rate_limits.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/069_shared_authentication_rate_limits.sql>) | Migration 069: shared authentication rate limits; Up/Down, policies and later definitions; app.request_rate_limits, app.record_rate_limit_attempt, app.prune_rate_limits | No separate finding |
| [db/migrations/070_deemed_acceptance_evidence.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/070_deemed_acceptance_evidence.sql>) | Migration 070: deemed acceptance evidence; Up/Down, policies and later definitions; app.guard_deemed_acceptance, deemed_acceptance_guard | No separate finding |
| [db/migrations/071_audit_financial_integrity.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/071_audit_financial_integrity.sql>) | Migration 071: audit financial integrity; Up/Down, policies and later definitions; app.trade_lines, app.sync_drawdown_exposure, obligation_drawdown_exposure, app.notifications | No separate finding |
| [db/migrations/072_repair_audit_legacy_state.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/072_repair_audit_legacy_state.sql>) | Migration 072: repair audit legacy state; Up/Down, policies and later definitions | No separate finding |
| [db/migrations/073_audit_security_invariants.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/073_audit_security_invariants.sql>) | Migration 073: audit security invariants; Up/Down, policies and later definitions; app.trade_line_mandate, app.guard_collected_payment_provenance, collected_payment_provenance, app.force_dispute_collection_effect and 1 other declarations | No separate finding |
| [db/migrations/074_mfa_replay_and_rotation.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/074_mfa_replay_and_rotation.sql>) | Migration 074: mfa replay and rotation; Up/Down, policies and later definitions; app.mfa_methods | No separate finding |
| [db/migrations/075_document_upload_lifecycle.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/075_document_upload_lifecycle.sql>) | Migration 075: document upload lifecycle; Up/Down, policies and later definitions; app.documents | No separate finding |
| [db/migrations/076_buyer_notice_acknowledgement.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/076_buyer_notice_acknowledgement.sql>) | Migration 076: buyer notice acknowledgement; Up/Down, policies and later definitions; app.collection_notice_acknowledgements, collection_notice_ack_buyer, collection_notice_ack_worker_read, app.guard_collection_notice and 1 other declarations | No separate finding |
| [db/migrations/077_document_tenant_rls.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/077_document_tenant_rls.sql>) | Migration 077: document tenant rls; Up/Down, policies and later definitions; document_tenant_or_worker, document_runtime_access | No separate finding |
| [db/migrations/078_runtime_role_idempotency.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/078_runtime_role_idempotency.sql>) | Migration 078: runtime role idempotency; Up/Down, policies and later definitions; idempotency_runtime_access, app.delete_expired_idempotency_record | No separate finding |
| [db/migrations/079_notice_acknowledgement_permissions.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/079_notice_acknowledgement_permissions.sql>) | Migration 079: notice acknowledgement permissions; Up/Down, policies and later definitions; collection_notice_ack_buyer_read, collection_notice_ack_buyer_insert | No separate finding |
| [db/migrations/080_collected_payment_reversal_provenance.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/080_collected_payment_reversal_provenance.sql>) | Migration 080: collected payment reversal provenance; Up/Down, policies and later definitions; app.guard_collected_payment_provenance | No separate finding |
| [db/migrations/081_phase2_tenant_isolation.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/081_phase2_tenant_isolation.sql>) | Migration 081: phase2 tenant isolation; Up/Down, policies and later definitions; credit_request_runtime_tenant, obligation_runtime_tenant, payment_runtime_tenant, allocation_runtime_tenant and 19 other declarations | No separate finding |
| [db/migrations/082_phase2_collection_tenant_isolation.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/082_phase2_collection_tenant_isolation.sql>) | Migration 082: phase2 collection tenant isolation; Up/Down, policies and later definitions; collection_snapshot_runtime_tenant, collection_index_runtime_tenant, collection_reservation_runtime_access, collection_attempt_runtime_access and 2 other declarations | No separate finding |
| [db/migrations/083_phase2_buyer_evidence_guard.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/083_phase2_buyer_evidence_guard.sql>) | Migration 083: phase2 buyer evidence guard; Up/Down, policies and later definitions; app.guard_buyer_originated_evidence, agreement_acceptances_worker_guard, receipt_confirmations_worker_guard, payment_claims_worker_guard and 1 other declarations | No separate finding |
| [db/migrations/084_phase2_collection_worker_boundaries.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/084_phase2_collection_worker_boundaries.sql>) | Migration 084: phase2 collection worker boundaries; Up/Down, policies and later definitions; app.collection_due_work_page, app.collection_attempt_work_page, app.collection_identity_by_attempt, app.collection_identity_by_external and 2 other declarations | No separate finding |
| [db/migrations/085_phase2_provider_collection_identity.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/085_phase2_provider_collection_identity.sql>) | Migration 085: phase2 provider collection identity; Up/Down, policies and later definitions; app.collection_attempt_identity_by_external | No separate finding |
| [db/migrations/086_phase5_financial_monitoring.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/086_phase5_financial_monitoring.sql>) | Migration 086: phase5 financial monitoring; Up/Down, policies and later definitions; app.phase5_financial_metrics | No separate finding |
| [db/migrations/087_owner_super_admin_controls.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/087_owner_super_admin_controls.sql>) | Migration 087: owner super admin controls; Up/Down, policies and later definitions; app.platform_role_assignments, app.is_platform_owner, app.is_active_policy_admin, app.protect_last_platform_owner and 12 other declarations | No separate finding |
| [db/migrations/088_settings_history_immutable.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/088_settings_history_immutable.sql>) | Migration 088: settings history immutable; Up/Down, policies and later definitions; app.reject_settings_history_mutation, platform_settings_history_immutable | No separate finding |
| [db/migrations/089_owner_lifecycle_guard.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/089_owner_lifecycle_guard.sql>) | Migration 089: owner lifecycle guard; Up/Down, policies and later definitions; app.platform_owner_guard, app.lock_owner_lifecycle, app.enforce_owner_lifecycle, owner_roles_lock and 3 other declarations | No separate finding |
| [db/migrations/090_governance_fail_closed.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/090_governance_fail_closed.sql>) | Migration 090: governance fail closed; Up/Down, policies and later definitions; app.current_governance_mode | No separate finding |
| [db/migrations/091_retire_unread_platform_settings.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/091_retire_unread_platform_settings.sql>) | Migration 091: retire unread platform settings; Up/Down, policies and later definitions; app.platform_settings | No separate finding |
| [db/migrations/092_public_payment_receipt.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/092_public_payment_receipt.sql>) | Migration 092: public payment receipt; Up/Down, policies and later definitions; app.public_payment_receipt | No separate finding |
| [db/migrations/093_admin_notification_connectors.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/093_admin_notification_connectors.sql>) | Migration 093: admin notification connectors; Up/Down, policies and later definitions; app.platform_settings | No separate finding |
| [db/migrations/094_admin_runtime_connections.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/094_admin_runtime_connections.sql>) | Migration 094: admin runtime connections; Up/Down, policies and later definitions; app.platform_settings | No separate finding |
| [db/migrations/095_delivery_issue_blocks_deemed_acceptance.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/095_delivery_issue_blocks_deemed_acceptance.sql>) | Migration 095: delivery issue blocks deemed acceptance; Up/Down, policies and later definitions; app.guard_deemed_acceptance_delivery_issue, deemed_acceptance_delivery_issue_guard | No separate finding |
| [db/migrations/096_close_legacy_tenant_bypasses.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/096_close_legacy_tenant_bypasses.sql>) | Migration 096: close legacy tenant bypasses; Up/Down, policies and later definitions | No separate finding |
| [db/migrations/097_collection_notice_versions.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/097_collection_notice_versions.sql>) | Migration 097: collection notice versions; Up/Down, policies and later definitions; app.collection_notice_acknowledgements, obligation_buyer_read, collection_notice_ack_buyer_insert | No separate finding |
| [db/migrations/098_provider_webhook_claims.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/098_provider_webhook_claims.sql>) | Migration 098: provider webhook claims; Up/Down, policies and later definitions; app.provider_webhook_inbox, app.guard_provider_webhook_identity, provider_webhook_identity_guard | No separate finding |
| [db/migrations/099_notification_send_identity.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/099_notification_send_identity.sql>) | Migration 099: notification send identity; Up/Down, policies and later definitions; app.notifications | No separate finding |
| [db/migrations/100_preserve_agreement_bytes.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/100_preserve_agreement_bytes.sql>) | Migration 100: preserve agreement bytes; Up/Down, policies and later definitions; app.agreement_versions | No separate finding |
| [db/migrations/101_owner_website_content.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/101_owner_website_content.sql>) | Migration 101: owner website content; Up/Down, policies and later definitions; app.platform_settings | No separate finding |
| [db/migrations/102_relationship_consent_evidence.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/102_relationship_consent_evidence.sql>) | Migration 102: relationship consent evidence; Up/Down, policies and later definitions; relationship_consent_supplier_access, relationship_consent_buyer_read, relationship_consent_buyer_record, app.reject_relationship_consent_mutation and 1 other declarations | No separate finding |
| [db/migrations/103_solo_owner_privacy_completion.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/103_solo_owner_privacy_completion.sql>) | Migration 103: solo owner privacy completion; Up/Down, policies and later definitions; app.privacy_requests, app.guard_solo_owner_privacy_completion, privacy_solo_owner_guard | No separate finding |
| [db/migrations/104_versioned_legal_publications.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/104_versioned_legal_publications.sql>) | Migration 104: versioned legal publications; Up/Down, policies and later definitions; app.platform_settings, app.drawdowns, app.guard_drawdown_legal_versions, drawdown_legal_versions_immutable | No separate finding |
| [db/migrations/105_buyer_supplier_directory.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/105_buyer_supplier_directory.sql>) | Migration 105: buyer supplier directory; Up/Down, policies and later definitions; app.buyer_suppliers | No separate finding |
| [db/migrations/106_consent_relationship_scope.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/106_consent_relationship_scope.sql>) | Migration 106: consent relationship scope; Up/Down, policies and later definitions; app.has_own_relationship_consent, relationship_consent_buyer_record | No separate finding |
| [db/migrations/107_notification_intent_fingerprint.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/107_notification_intent_fingerprint.sql>) | Migration 107: notification intent fingerprint; Up/Down, policies and later definitions; app.notifications, app.preserve_notification_intent, notifications_intent_immutable | No separate finding |
| [db/migrations/108_admin_document_scan_recovery.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/108_admin_document_scan_recovery.sql>) | Migration 108: admin document scan recovery; Up/Down, policies and later definitions; app.documents, app.operations_commands, document_admin_read, document_admin_recovery | No separate finding |
| [db/migrations/109_guide_and_contact_publication.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/109_guide_and_contact_publication.sql>) | Migration 109: guide and contact publication; Up/Down, policies and later definitions; app.platform_settings | No separate finding |
| [db/migrations/110_notification_recipient_scope.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/110_notification_recipient_scope.sql>) | Migration 110: notification recipient scope; Up/Down, policies and later definitions; app.notification_event_identity | No separate finding |
| [db/migrations/111_team_invitation_lifecycle.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/111_team_invitation_lifecycle.sql>) | Migration 111: team invitation lifecycle; Up/Down, policies and later definitions; app.organization_invitations, app.close_membership_invitation, membership_invitation_lifecycle | No separate finding |
| [db/migrations/112_system_acceptance_evidence.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/112_system_acceptance_evidence.sql>) | Migration 112: system acceptance evidence; Up/Down, policies and later definitions; app.system_acceptances, system_acceptance_tenant, system_acceptance_immutable, app.deemed_acceptance_evidence and 3 other declarations | No separate finding |
| [db/migrations/113_document_orphan_recovery.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/113_document_orphan_recovery.sql>) | Migration 113: document orphan recovery; Up/Down, policies and later definitions; app.document_object_is_orphan | No separate finding |
| [db/migrations/114_system_acceptance_admin_controls.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/114_system_acceptance_admin_controls.sql>) | Migration 114: system acceptance admin controls; Up/Down, policies and later definitions; app.guard_system_acceptance_policy, system_acceptance_policy | No separate finding |
| [db/migrations/115_atomic_domain_activity.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/115_atomic_domain_activity.sql>) | Migration 115: atomic domain activity; Up/Down, policies and later definitions; app.record_domain_activity, domain_activity | No separate finding |
| [db/migrations/116_system_acceptance_settings_access.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/116_system_acceptance_settings_access.sql>) | Migration 116: system acceptance settings access; Up/Down, policies and later definitions; app.system_acceptance_settings | No separate finding |
| [db/migrations/117_customer_registration_recovery.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/117_customer_registration_recovery.sql>) | Migration 117: customer registration recovery; Up/Down, policies and later definitions; app.customer_registration_attempts, customer_registration_access, app.guard_customer_registration, customer_registration_guard and 1 other declarations | No separate finding |

## db/seeds

| File | Reviewed content | Result |
| --- | --- | --- |
| [db/seeds/001_demo.sql](</Users/macbookpro/Documents/Kredit.com/db/seeds/001_demo.sql>) | Synthetic seed identities, readiness and financial consistency | No separate finding |

## docs

| File | Reviewed content | Result |
| --- | --- | --- |
| [docs/data-map.md](</Users/macbookpro/Documents/Kredit.com/docs/data-map.md>) | Kredit data map: source consistency and evidence limits | No separate finding |
| [docs/threat-model.md](</Users/macbookpro/Documents/Kredit.com/docs/threat-model.md>) | Kredit threat model: source consistency and evidence limits | No separate finding |

## docs/adr

| File | Reviewed content | Result |
| --- | --- | --- |
| [docs/adr/0001-modular-monolith.md](</Users/macbookpro/Documents/Kredit.com/docs/adr/0001-modular-monolith.md>) | ADR 0001: Modular monolith for production v1: source consistency and evidence limits | No separate finding |
| [docs/adr/0002-financial-source-of-truth.md](</Users/macbookpro/Documents/Kredit.com/docs/adr/0002-financial-source-of-truth.md>) | ADR 0002: PostgreSQL and the ledger are authoritative: source consistency and evidence limits | No separate finding |
| [docs/adr/0003-provider-neutral-identity.md](</Users/macbookpro/Documents/Kredit.com/docs/adr/0003-provider-neutral-identity.md>) | ADR 0003: Provider-neutral identity verification: source consistency and evidence limits | No separate finding |
| [docs/adr/0004-mono-sweep-collection-boundary.md](</Users/macbookpro/Documents/Kredit.com/docs/adr/0004-mono-sweep-collection-boundary.md>) | ADR 0004: Mono Sweep behind the collection boundary: source consistency and evidence limits | No separate finding |
| [docs/adr/0005-hand-written-http-and-sql.md](</Users/macbookpro/Documents/Kredit.com/docs/adr/0005-hand-written-http-and-sql.md>) | ADR 0005: Hand-written HTTP handlers and SQL, with the OpenAPI document as the enforced contract: source consistency and evidence limits | No separate finding |
| [docs/adr/README.md](</Users/macbookpro/Documents/Kredit.com/docs/adr/README.md>) | Architecture decision records: source consistency and evidence limits | No separate finding |

## docs/api

| File | Reviewed content | Result |
| --- | --- | --- |
| [docs/api/README.md](</Users/macbookpro/Documents/Kredit.com/docs/api/README.md>) | API documentation: source consistency and evidence limits | No separate finding |
| [docs/api/openapi.yaml](</Users/macbookpro/Documents/Kredit.com/docs/api/openapi.yaml>) | openapi.yaml: configuration/data and current consumers | No separate finding |

## docs/architecture

| File | Reviewed content | Result |
| --- | --- | --- |
| [docs/architecture/product-engineering-principles.md](</Users/macbookpro/Documents/Kredit.com/docs/architecture/product-engineering-principles.md>) | Kredit product and engineering principles: source consistency and evidence limits | No separate finding |

## docs/compliance

| File | Reviewed content | Result |
| --- | --- | --- |
| [docs/compliance/README.md](</Users/macbookpro/Documents/Kredit.com/docs/compliance/README.md>) | Compliance documentation: source consistency and evidence limits | [F041](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:173) |
| [docs/compliance/data-inventory.md](</Users/macbookpro/Documents/Kredit.com/docs/compliance/data-inventory.md>) | Restricted-data inventory: source consistency and evidence limits | [F041](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:173) |
| [docs/compliance/data-inventory.tsv](</Users/macbookpro/Documents/Kredit.com/docs/compliance/data-inventory.tsv>) | data-inventory.tsv: provenance, semantic records and current applicability | [F041](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:173) |
| [docs/compliance/dpia-trade-history-sharing.md](</Users/macbookpro/Documents/Kredit.com/docs/compliance/dpia-trade-history-sharing.md>) | DPIA — cross-supplier trade history sharing: source consistency and evidence limits | [F041](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:173) |

## docs/content

| File | Reviewed content | Result |
| --- | --- | --- |
| [docs/content/article-research-2026-09-04.csv](</Users/macbookpro/Documents/Kredit.com/docs/content/article-research-2026-09-04.csv>) | article-research-2026-09-04.csv: provenance, semantic records and current applicability | No separate finding |
| [docs/content/editorial-standard.md](</Users/macbookpro/Documents/Kredit.com/docs/content/editorial-standard.md>) | Kredit editorial standard: source consistency and evidence limits | No separate finding |
| [docs/content/search-research-2026-09-04.md](</Users/macbookpro/Documents/Kredit.com/docs/content/search-research-2026-09-04.md>) | Kredit article research — 4 September 2026: source consistency and evidence limits | No separate finding |
| [docs/content/topic-backlog.md](</Users/macbookpro/Documents/Kredit.com/docs/content/topic-backlog.md>) | Unpublished guide ideas: source consistency and evidence limits | No separate finding |

## docs/launch-audit-2026-09-08

| File | Reviewed content | Result |
| --- | --- | --- |
| [docs/launch-audit-2026-09-08/ADMIN-CONNECTIONS.md](</Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-08/ADMIN-CONNECTIONS.md>) | Admin connection controls: source consistency and evidence limits | No separate finding |
| [docs/launch-audit-2026-09-08/FILE-BY-FILE.md](</Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-08/FILE-BY-FILE.md>) | Final code review checkpoint — 10 September 2026: source consistency and evidence limits | No separate finding |
| [docs/launch-audit-2026-09-08/PROGRESS.md](</Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-08/PROGRESS.md>) | Current checkpoint: source consistency and evidence limits | No separate finding |
| [docs/launch-audit-2026-09-08/admin-connections-mobile-actions.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-08/admin-connections-mobile-actions.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-audit-2026-09-08/admin-connections-mobile.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-08/admin-connections-mobile.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-audit-2026-09-08/file-by-file-audit.json](</Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-08/file-by-file-audit.json>) | file-by-file-audit.json: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-audit-2026-09-08/file-review-index.json](</Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-08/file-review-index.json>) | file-review-index.json: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-audit-2026-09-08/home-desktop.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-08/home-desktop.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-audit-2026-09-08/home-mobile.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-08/home-mobile.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-audit-2026-09-08/initial-browser.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-08/initial-browser.log>) | initial-browser.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-audit-2026-09-08/signin-desktop.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-08/signin-desktop.png>) | Historical screenshot; layout and provenance | No separate finding |

## docs/launch-readiness

| File | Reviewed content | Result |
| --- | --- | --- |
| [docs/launch-readiness/OPERATING-NOTES.md](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/OPERATING-NOTES.md>) | Local implementation checkpoint — 7 September 2026: source consistency and evidence limits | No separate finding |
| [docs/launch-readiness/RESUME.md](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/RESUME.md>) | Resume from this local checkpoint: source consistency and evidence limits | No separate finding |
| [docs/launch-readiness/TEST-RESULTS.md](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/TEST-RESULTS.md>) | Executed local verification: source consistency and evidence limits | No separate finding |
| [docs/launch-readiness/baseline.txt](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/baseline.txt>) | baseline.txt: configuration/data and current consumers | No separate finding |
| [docs/launch-readiness/changed-files.txt](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/changed-files.txt>) | changed-files.txt: configuration/data and current consumers | No separate finding |
| [docs/launch-readiness/changes.patch](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/changes.patch>) | changes.patch: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence-manifest.json](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence-manifest.json>) | evidence-manifest.json: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/after-complaints.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/after-complaints.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-readiness/evidence/after-home.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/after-home.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-readiness/evidence/after-pricing.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/after-pricing.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-readiness/evidence/after-privacy.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/after-privacy.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-readiness/evidence/after-signin.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/after-signin.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-readiness/evidence/after-terms-mobile-closed.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/after-terms-mobile-closed.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-readiness/evidence/after-terms-mobile-open.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/after-terms-mobile-open.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-readiness/evidence/after-terms.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/after-terms.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-readiness/evidence/all-go-final.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/all-go-final.log>) | all-go-final.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/all-go-unsandboxed.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/all-go-unsandboxed.log>) | all-go-unsandboxed.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/all-go.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/all-go.log>) | all-go.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/api.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/api.log>) | api.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/before-home.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/before-home.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-readiness/evidence/before-pricing.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/before-pricing.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-readiness/evidence/before-privacy.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/before-privacy.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-readiness/evidence/before-signin.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/before-signin.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-readiness/evidence/before-terms.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/before-terms.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-readiness/evidence/browser-check.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/browser-check.log>) | browser-check.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/crypto-test.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/crypto-test.log>) | crypto-test.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/db-roles.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/db-roles.log>) | db-roles.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/focused-go.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/focused-go.log>) | focused-go.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/migrations-final.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/migrations-final.log>) | migrations-final.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/migrations.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/migrations.log>) | migrations.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/node-preview.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/node-preview.log>) | node-preview.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/outage-check.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/outage-check.log>) | outage-check.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/outage-preview.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/outage-preview.log>) | outage-preview.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/owner-api-integration.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/owner-api-integration.log>) | owner-api-integration.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/owner-baseline-migrations.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/owner-baseline-migrations.log>) | owner-baseline-migrations.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/owner-fixed-migrations.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/owner-fixed-migrations.log>) | owner-fixed-migrations.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/owner-lifecycle-after.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/owner-lifecycle-after.log>) | owner-lifecycle-after.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/owner-lifecycle-before.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/owner-lifecycle-before.log>) | owner-lifecycle-before.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/postgres-init.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/postgres-init.log>) | postgres-init.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/pricing-source-outage.png](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/pricing-source-outage.png>) | Historical screenshot; layout and provenance | No separate finding |
| [docs/launch-readiness/evidence/public-playwright.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/public-playwright.log>) | public-playwright.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/race.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/race.log>) | race.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/release-api.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/release-api.log>) | release-api.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/release-browser-check.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/release-browser-check.log>) | release-browser-check.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/release-capabilities.json](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/release-capabilities.json>) | release-capabilities.json: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/release-pricing.json](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/release-pricing.json>) | release-pricing.json: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/settings-integration-final.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/settings-integration-final.log>) | settings-integration-final.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/settings-integration.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/settings-integration.log>) | settings-integration.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/web-build-final.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/web-build-final.log>) | web-build-final.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/web-build.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/web-build.log>) | web-build.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/web-check-final.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/web-check-final.log>) | web-check-final.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/evidence/web-check.log](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/evidence/web-check.log>) | web-check.log: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/route-state-inventory.csv](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/route-state-inventory.csv>) | route-state-inventory.csv: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/settings-inventory.csv](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/settings-inventory.csv>) | settings-inventory.csv: provenance, semantic records and current applicability | No separate finding |
| [docs/launch-readiness/source-manifest.json](</Users/macbookpro/Documents/Kredit.com/docs/launch-readiness/source-manifest.json>) | source-manifest.json: provenance, semantic records and current applicability | No separate finding |

## docs/operations

| File | Reviewed content | Result |
| --- | --- | --- |
| [docs/operations/PRODUCTION-DEPLOYMENT-GUIDE.md](</Users/macbookpro/Documents/Kredit.com/docs/operations/PRODUCTION-DEPLOYMENT-GUIDE.md>) | Production Deployment Guide: kredit.ng: source consistency and evidence limits | [F006](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:33), [F038](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:161), [F042](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:177) |
| [docs/operations/admin-surface-enablement.md](</Users/macbookpro/Documents/Kredit.com/docs/operations/admin-surface-enablement.md>) | Admin surface enablement: source consistency and evidence limits | No separate finding |
| [docs/operations/observability.md](</Users/macbookpro/Documents/Kredit.com/docs/operations/observability.md>) | Observability contract: source consistency and evidence limits | No separate finding |
| [docs/operations/persistence-migration.md](</Users/macbookpro/Documents/Kredit.com/docs/operations/persistence-migration.md>) | Persistence migration contract: source consistency and evidence limits | No separate finding |
| [docs/operations/provider-adapter.md](</Users/macbookpro/Documents/Kredit.com/docs/operations/provider-adapter.md>) | Provider adapter release gate: source consistency and evidence limits | No separate finding |
| [docs/operations/provider-certification-plan.md](</Users/macbookpro/Documents/Kredit.com/docs/operations/provider-certification-plan.md>) | Collection provider certification and contingency: source consistency and evidence limits | No separate finding |
| [docs/operations/readme-conformance.md](</Users/macbookpro/Documents/Kredit.com/docs/operations/readme-conformance.md>) | README conformance audit: source consistency and evidence limits | No separate finding |
| [docs/operations/readme-gap-audit.md](</Users/macbookpro/Documents/Kredit.com/docs/operations/readme-gap-audit.md>) | README implementation gap audit: source consistency and evidence limits | No separate finding |

## docs/product

| File | Reviewed content | Result |
| --- | --- | --- |
| [docs/product/analytics-event-catalog.md](</Users/macbookpro/Documents/Kredit.com/docs/product/analytics-event-catalog.md>) | Product analytics event catalog: source consistency and evidence limits | No separate finding |
| [docs/product/interface-copy.md](</Users/macbookpro/Documents/Kredit.com/docs/product/interface-copy.md>) | Financial and trust interface copy: source consistency and evidence limits | No separate finding |
| [docs/product/kredit-interface-standard.md](</Users/macbookpro/Documents/Kredit.com/docs/product/kredit-interface-standard.md>) | Kredit interface standard: source consistency and evidence limits | No separate finding |
| [docs/product/open-questions.md](</Users/macbookpro/Documents/Kredit.com/docs/product/open-questions.md>) | Product and external-dependency decision register: source consistency and evidence limits | No separate finding |
| [docs/product/pilot-kill-thresholds.md](</Users/macbookpro/Documents/Kredit.com/docs/product/pilot-kill-thresholds.md>) | Pilot kill thresholds: source consistency and evidence limits | No separate finding |
| [docs/product/pilot-scorecard.md](</Users/macbookpro/Documents/Kredit.com/docs/product/pilot-scorecard.md>) | Pilot KPI scorecard: source consistency and evidence limits | No separate finding |
| [docs/product/public-launch-and-admin-controls.md](</Users/macbookpro/Documents/Kredit.com/docs/product/public-launch-and-admin-controls.md>) | Owner Super Admin and private service activation: source consistency and evidence limits | No separate finding |
| [docs/product/readme-completion-plan.md](</Users/macbookpro/Documents/Kredit.com/docs/product/readme-completion-plan.md>) | README completion implementation plan: source consistency and evidence limits | No separate finding |
| [docs/product/readme-completion-traceability.md](</Users/macbookpro/Documents/Kredit.com/docs/product/readme-completion-traceability.md>) | README completion-plan traceability: source consistency and evidence limits | No separate finding |
| [docs/product/strategic-recommendations.md](</Users/macbookpro/Documents/Kredit.com/docs/product/strategic-recommendations.md>) | Kredit — Strategic Recommendations: source consistency and evidence limits | No separate finding |
| [docs/product/wave0-contracts.md](</Users/macbookpro/Documents/Kredit.com/docs/product/wave0-contracts.md>) | Wave 0 shared contract registry: source consistency and evidence limits | No separate finding |
| [docs/product/workstream-evidence.tsv](</Users/macbookpro/Documents/Kredit.com/docs/product/workstream-evidence.tsv>) | workstream-evidence.tsv: provenance, semantic records and current applicability | No separate finding |
| [docs/product/world-class-product-standard.md](</Users/macbookpro/Documents/Kredit.com/docs/product/world-class-product-standard.md>) | World-class product standard: source consistency and evidence limits | No separate finding |

## docs/release

| File | Reviewed content | Result |
| --- | --- | --- |
| [docs/release/certification-report.md](</Users/macbookpro/Documents/Kredit.com/docs/release/certification-report.md>) | Production-v1 certification report: source consistency and evidence limits | No separate finding |
| [docs/release/go-live-runbook.md](</Users/macbookpro/Documents/Kredit.com/docs/release/go-live-runbook.md>) | Kredit go-live runbook: source consistency and evidence limits | [F042](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:177) |
| [docs/release/readiness-checklist.md](</Users/macbookpro/Documents/Kredit.com/docs/release/readiness-checklist.md>) | Production readiness checklist: source consistency and evidence limits | No separate finding |
| [docs/release/wave5-accessibility-evidence.md](</Users/macbookpro/Documents/Kredit.com/docs/release/wave5-accessibility-evidence.md>) | Wave 5 accessibility evidence: source consistency and evidence limits | No separate finding |
| [docs/release/wave6-analytics-evidence.md](</Users/macbookpro/Documents/Kredit.com/docs/release/wave6-analytics-evidence.md>) | Wave 6 product analytics evidence: source consistency and evidence limits | No separate finding |

## docs/runbooks

| File | Reviewed content | Result |
| --- | --- | --- |
| [docs/runbooks/README.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/README.md>) | Operational runbooks: source consistency and evidence limits | No separate finding |
| [docs/runbooks/admin-workflows.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/admin-workflows.md>) | Admin workflows: source consistency and evidence limits | [F042](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:177) |
| [docs/runbooks/backup-restore.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/backup-restore.md>) | Backup restore: source consistency and evidence limits | No separate finding |
| [docs/runbooks/break-glass-access.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/break-glass-access.md>) | Break-glass access: source consistency and evidence limits | No separate finding |
| [docs/runbooks/business-settings.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/business-settings.md>) | Business settings: source consistency and evidence limits | [F042](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:177) |
| [docs/runbooks/dispute-adjustment.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/dispute-adjustment.md>) | Dispute adjustment: source consistency and evidence limits | No separate finding |
| [docs/runbooks/failed-jobs.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/failed-jobs.md>) | Failed jobs: source consistency and evidence limits | No separate finding |
| [docs/runbooks/failed-webhooks.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/failed-webhooks.md>) | Failed webhooks: source consistency and evidence limits | [F026](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:113), [F042](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:177) |
| [docs/runbooks/financial-operations.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/financial-operations.md>) | Financial and support operations runbook: source consistency and evidence limits | No separate finding |
| [docs/runbooks/mandate-cancellation.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/mandate-cancellation.md>) | Mandate cancellation: source consistency and evidence limits | No separate finding |
| [docs/runbooks/mono-sweep.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/mono-sweep.md>) | Mono Sweep sandbox integration: source consistency and evidence limits | [F042](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:177) |
| [docs/runbooks/operations-controls.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/operations-controls.md>) | Protected operations controls: source consistency and evidence limits | No separate finding |
| [docs/runbooks/payment-reversal.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/payment-reversal.md>) | Payment reversal: source consistency and evidence limits | No separate finding |
| [docs/runbooks/phase5-production-assurance.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/phase5-production-assurance.md>) | Phase 5 — Production assurance: source consistency and evidence limits | [F042](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:177) |
| [docs/runbooks/pilot-scorecard.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/pilot-scorecard.md>) | Weekly pilot scorecard review: source consistency and evidence limits | No separate finding |
| [docs/runbooks/production-pilot.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/production-pilot.md>) | Production pilot runbook: source consistency and evidence limits | No separate finding |
| [docs/runbooks/provider-reconciliation.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/provider-reconciliation.md>) | Provider reconciliation: source consistency and evidence limits | No separate finding |
| [docs/runbooks/provider-simulator.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/provider-simulator.md>) | Provider simulator runbook: source consistency and evidence limits | No separate finding |
| [docs/runbooks/worker-operations.md](</Users/macbookpro/Documents/Kredit.com/docs/runbooks/worker-operations.md>) | Worker operations: source consistency and evidence limits | No separate finding |

## docs/security

| File | Reviewed content | Result |
| --- | --- | --- |
| [docs/security/production-security-checklist.md](</Users/macbookpro/Documents/Kredit.com/docs/security/production-security-checklist.md>) | Production security checklist: source consistency and evidence limits | No separate finding |

## docs/testing

| File | Reviewed content | Result |
| --- | --- | --- |
| [docs/testing/admin-settings-2026-09-03.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/admin-settings-2026-09-03.md>) | Admin business settings implementation: source consistency and evidence limits | No separate finding |
| [docs/testing/admin-workflows-2026-09-03.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/admin-workflows-2026-09-03.md>) | Admin workflow implementation and verification — 3 September 2026: source consistency and evidence limits | No separate finding |
| [docs/testing/audit-2026-09-06-implementation.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/audit-2026-09-06-implementation.md>) | 6 September product and reliability audit implementation: source consistency and evidence limits | No separate finding |
| [docs/testing/audit-2026-09-06-refinement.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/audit-2026-09-06-refinement.md>) | Audit refinement: source consistency and evidence limits | No separate finding |
| [docs/testing/audit-2026-09-06-verification.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/audit-2026-09-06-verification.md>) | Audit implementation verification: source consistency and evidence limits | No separate finding |
| [docs/testing/audit-2026-09-07-completion.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/audit-2026-09-07-completion.md>) | Audit completion — 7 September 2026: source consistency and evidence limits | No separate finding |
| [docs/testing/code-audit-2026-09-03-files.csv](</Users/macbookpro/Documents/Kredit.com/docs/testing/code-audit-2026-09-03-files.csv>) | code-audit-2026-09-03-files.csv: provenance, semantic records and current applicability | No separate finding |
| [docs/testing/code-audit-2026-09-03.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/code-audit-2026-09-03.md>) | Kredit code audit — 3 September 2026: source consistency and evidence limits | No separate finding |
| [docs/testing/code-audit-2026-09-04-files.csv](</Users/macbookpro/Documents/Kredit.com/docs/testing/code-audit-2026-09-04-files.csv>) | code-audit-2026-09-04-files.csv: provenance, semantic records and current applicability | No separate finding |
| [docs/testing/code-audit-2026-09-04.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/code-audit-2026-09-04.md>) | Kredit file-by-file code audit — 4 September 2026: source consistency and evidence limits | No separate finding |
| [docs/testing/code-audit-fixes-2026-09-04.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/code-audit-fixes-2026-09-04.md>) | Kredit code-audit fixes — 4 September 2026: source consistency and evidence limits | No separate finding |
| [docs/testing/code-completion-2026-09-02.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/code-completion-2026-09-02.md>) | Code completion — 2 September 2026: source consistency and evidence limits | No separate finding |
| [docs/testing/code-review-2026-09-02-files.tsv](</Users/macbookpro/Documents/Kredit.com/docs/testing/code-review-2026-09-02-files.tsv>) | code-review-2026-09-02-files.tsv: provenance, semantic records and current applicability | No separate finding |
| [docs/testing/code-review-2026-09-02.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/code-review-2026-09-02.md>) | Direct code review — 2 September 2026: source consistency and evidence limits | No separate finding |
| [docs/testing/mono-sweep-evidence.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/mono-sweep-evidence.md>) | Mono Sweep acceptance evidence — 2 September 2026: source consistency and evidence limits | No separate finding |
| [docs/testing/phase4-provider-evidence.template.json](</Users/macbookpro/Documents/Kredit.com/docs/testing/phase4-provider-evidence.template.json>) | phase4-provider-evidence.template.json: provenance, semantic records and current applicability | No separate finding |
| [docs/testing/phase4-provider-verification.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/phase4-provider-verification.md>) | Phase 4 — External-provider verification: source consistency and evidence limits | No separate finding |
| [docs/testing/phase6-hardening.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/phase6-hardening.md>) | Phase 6 — hardening and product polish: source consistency and evidence limits | No separate finding |
| [docs/testing/prelaunch-followup-2026-09-04.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/prelaunch-followup-2026-09-04.md>) | Prelaunch editorial and engineering follow-up — 4 September 2026: source consistency and evidence limits | No separate finding |
| [docs/testing/solo-audit-2026-09-04-files.csv](</Users/macbookpro/Documents/Kredit.com/docs/testing/solo-audit-2026-09-04-files.csv>) | solo-audit-2026-09-04-files.csv: provenance, semantic records and current applicability | No separate finding |
| [docs/testing/solo-audit-2026-09-04.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/solo-audit-2026-09-04.md>) | Kredit solo audit — 4 September 2026: source consistency and evidence limits | No separate finding |
| [docs/testing/test-matrix.md](</Users/macbookpro/Documents/Kredit.com/docs/testing/test-matrix.md>) | Required test matrix: source consistency and evidence limits | No separate finding |

## infra/containers

| File | Reviewed content | Result |
| --- | --- | --- |
| [infra/containers/.gitkeep](</Users/macbookpro/Documents/Kredit.com/infra/containers/.gitkeep>) | .gitkeep: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [infra/containers/Dockerfile.api](</Users/macbookpro/Documents/Kredit.com/infra/containers/Dockerfile.api>) | Dockerfile.api: deployment, boundaries, failure handling and referenced paths | [F039](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:165) |
| [infra/containers/Dockerfile.web](</Users/macbookpro/Documents/Kredit.com/infra/containers/Dockerfile.web>) | Dockerfile.web: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [infra/containers/README.md](</Users/macbookpro/Documents/Kredit.com/infra/containers/README.md>) | README.md: deployment, boundaries, failure handling and referenced paths | No separate finding |

## infra/environments

| File | Reviewed content | Result |
| --- | --- | --- |
| [infra/environments/.gitkeep](</Users/macbookpro/Documents/Kredit.com/infra/environments/.gitkeep>) | .gitkeep: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [infra/environments/Caddyfile.prod](</Users/macbookpro/Documents/Kredit.com/infra/environments/Caddyfile.prod>) | Caddyfile.prod: deployment, boundaries, failure handling and referenced paths | [F003](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:17), [F011](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:53) |
| [infra/environments/README.md](</Users/macbookpro/Documents/Kredit.com/infra/environments/README.md>) | README.md: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [infra/environments/docker-compose.prod.yml](</Users/macbookpro/Documents/Kredit.com/infra/environments/docker-compose.prod.yml>) | docker-compose.prod.yml: deployment, boundaries, failure handling and referenced paths | [F002](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:13), [F004](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:23), [F006](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:33) |
| [infra/environments/env.production.example](</Users/macbookpro/Documents/Kredit.com/infra/environments/env.production.example>) | env.production.example: deployment, boundaries, failure handling and referenced paths | [F002](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:13), [F005](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:29), [F041](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:173), [F042](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:177) |
| [infra/environments/main.tf](</Users/macbookpro/Documents/Kredit.com/infra/environments/main.tf>) | main.tf: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [infra/environments/outputs.tf](</Users/macbookpro/Documents/Kredit.com/infra/environments/outputs.tf>) | outputs.tf: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [infra/environments/variables.tf](</Users/macbookpro/Documents/Kredit.com/infra/environments/variables.tf>) | variables.tf: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [infra/environments/versions.tf](</Users/macbookpro/Documents/Kredit.com/infra/environments/versions.tf>) | versions.tf: deployment, boundaries, failure handling and referenced paths | No separate finding |

## infra/monitoring

| File | Reviewed content | Result |
| --- | --- | --- |
| [infra/monitoring/.gitkeep](</Users/macbookpro/Documents/Kredit.com/infra/monitoring/.gitkeep>) | .gitkeep: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [infra/monitoring/README.md](</Users/macbookpro/Documents/Kredit.com/infra/monitoring/README.md>) | README.md: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [infra/monitoring/otel-collector.yaml](</Users/macbookpro/Documents/Kredit.com/infra/monitoring/otel-collector.yaml>) | otel-collector.yaml: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [infra/monitoring/phase5-rules.test.yaml](</Users/macbookpro/Documents/Kredit.com/infra/monitoring/phase5-rules.test.yaml>) | phase5-rules.test.yaml: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [infra/monitoring/prometheus-rules.yaml](</Users/macbookpro/Documents/Kredit.com/infra/monitoring/prometheus-rules.yaml>) | prometheus-rules.yaml: deployment, boundaries, failure handling and referenced paths | No separate finding |

## infra/postgres

| File | Reviewed content | Result |
| --- | --- | --- |
| [infra/postgres/development-logins.sql](</Users/macbookpro/Documents/Kredit.com/infra/postgres/development-logins.sql>) | development-logins.sql: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [infra/postgres/roles.sql](</Users/macbookpro/Documents/Kredit.com/infra/postgres/roles.sql>) | roles.sql: deployment, boundaries, failure handling and referenced paths | No separate finding |

## internal/access

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/access/admin_roles_test.go](</Users/macbookpro/Documents/Kredit.com/internal/access/admin_roles_test.go>) | Role permissions and fresh identity requirements; TestAdminDutiesRemainSeparate, TestComplianceCannotExecuteOperationalCommands, TestViewerCannotMutateDisputes | No separate finding |
| [internal/access/authority.go](</Users/macbookpro/Documents/Kredit.com/internal/access/authority.go>) | Role permissions and fresh identity requirements; LockPlatformAuthority | No separate finding |
| [internal/access/roles.go](</Users/macbookpro/Documents/Kredit.com/internal/access/roles.go>) | Role permissions and fresh identity requirements; Valid, ParseRole, CanPlatform, Can and 1 other declarations | No separate finding |
| [internal/access/roles_test.go](</Users/macbookpro/Documents/Kredit.com/internal/access/roles_test.go>) | Role permissions and fresh identity requirements; TestRolePermissions, TestPlatformOwnerPermissionsAndExclusivity | No separate finding |

## internal/agreementdocs

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/agreementdocs/document.go](</Users/macbookpro/Documents/Kredit.com/internal/agreementdocs/document.go>) | Accepted agreement presentation and canonical evidence; RenderDrawdownHTML, RenderHTML, lagosLocation | No separate finding |
| [internal/agreementdocs/document_test.go](</Users/macbookpro/Documents/Kredit.com/internal/agreementdocs/document_test.go>) | Accepted agreement presentation and canonical evidence; TestRenderHTMLIncludesRequiredEvidence, TestRenderHTMLRejectsTamperedCanonicalAgreement, TestRenderDrawdownHTMLVerifiesExactTerms | No separate finding |

## internal/audit

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/audit/audit_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/audit/audit_postgres_test.go>) | Audit persistence, scope and redaction; TestPostgresActivityAllowsMissingRequestID | No separate finding |
| [internal/audit/domain_activity_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/audit/domain_activity_postgres_test.go>) | Audit persistence, scope and redaction; TestDomainActivityCommitsAndRollsBackWithPrivateRequest | No separate finding |
| [internal/audit/store.go](</Users/macbookpro/Documents/Kredit.com/internal/audit/store.go>) | Audit persistence, scope and redaction; NewStore, Append, ListForOrganization, cloneEvent and 8 other declarations | No separate finding |
| [internal/audit/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/audit/store_test.go>) | Audit persistence, scope and redaction; TestAppendDropsSensitiveMetadata, TestEventJSONUsesThePublicActivitySchema | No separate finding |

## internal/auth

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/auth/audit_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/auth/audit_postgres_test.go>) | OTP, sessions, MFA, account status and replay limits; TestPostgresAbandonedTOTPEnrollmentCanRestart, TestPostgresStepUpRejectsInactiveAccount | No separate finding |
| [internal/auth/fuzz_test.go](</Users/macbookpro/Documents/Kredit.com/internal/auth/fuzz_test.go>) | OTP, sessions, MFA, account status and replay limits; FuzzPhoneNormalisation | No separate finding |
| [internal/auth/postgres.go](</Users/macbookpro/Documents/Kredit.com/internal/auth/postgres.go>) | OTP, sessions, MFA, account status and replay limits; VerifyAndAttachIdentifier, UserByID, NewPostgresStore, NewPostgresStoreWithKeys and 23 other declarations | No separate finding |
| [internal/auth/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/auth/postgres_test.go>) | OTP, sessions, MFA, account status and replay limits; TestPostgresStoreEncryptsRecoverableTargets, TestPostgresStoreImplementsService | No separate finding |
| [internal/auth/store.go](</Users/macbookpro/Documents/Kredit.com/internal/auth/store.go>) | OTP, sessions, MFA, account status and replay limits; NewStore, NewStoreWithKeys, RequestOTP, VerifyOTP and 29 other declarations | No separate finding |
| [internal/auth/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/auth/store_test.go>) | OTP, sessions, MFA, account status and replay limits; TestOTPLoginStoresOnlyOpaqueSessionLookup, TestTOTPEnrollmentElevatesSession, TestOTPCooldownCannotBeBypassedByChangingPurpose, TestEnrollmentCannotReplaceExistingMFA and 7 other declarations | No separate finding |
| [internal/auth/target_test.go](</Users/macbookpro/Documents/Kredit.com/internal/auth/target_test.go>) | OTP, sessions, MFA, account status and replay limits; TestOTPVerificationCanBeBoundToInvitationTarget | No separate finding |

## internal/businesspolicy

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/businesspolicy/policy.go](</Users/macbookpro/Documents/Kredit.com/internal/businesspolicy/policy.go>) | Versioned policy changes, approval authority and financial terms; Catalog, Defaults, Validate, ValidateDeployment and 3 other declarations | No separate finding |
| [internal/businesspolicy/policy_test.go](</Users/macbookpro/Documents/Kredit.com/internal/businesspolicy/policy_test.go>) | Versioned policy changes, approval authority and financial terms; TestPolicyRequiresCompleteValidValuesAndDeploymentCeilings | No separate finding |
| [internal/businesspolicy/store.go](</Users/macbookpro/Documents/Kredit.com/internal/businesspolicy/store.go>) | Versioned policy changes, approval authority and financial terms; NewStore, Ensure, ReadTx, Read and 8 other declarations | No separate finding |
| [internal/businesspolicy/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/businesspolicy/store_test.go>) | Versioned policy changes, approval authority and financial terms; TestPostgresIndependentApprovalHistoryAndEffectivePolicy, TestPolicyLockSerializesWriters, TestPolicyCapsApplyAtDatabaseWriteBoundary | No separate finding |

## internal/buyers

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/buyers/audit_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/buyers/audit_postgres_test.go>) | Invitation ownership, business identity, consent and onboarding recovery; TestPostgresInvitationHashAndSecondSupplierReuse | No separate finding |
| [internal/buyers/postgres.go](</Users/macbookpro/Documents/Kredit.com/internal/buyers/postgres.go>) | Invitation ownership, business identity, consent and onboarding recovery; NewPostgresStore, SetInvitationGuard, SetAcceptanceGuard, SetBusinessLimit and 19 other declarations | [F020](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:89), [F028](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:121) |
| [internal/buyers/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/buyers/postgres_test.go>) | Invitation ownership, business identity, consent and onboarding recovery; TestPortalReadDistinguishesAbsentProfileFromFailure, TestPostgresBuyerStoreImplementsServiceAndFailsClosedWithoutDatabase, TestPostgresBuyerStoreEncryptionRoundTrip | No separate finding |
| [internal/buyers/store.go](</Users/macbookpro/Documents/Kredit.com/internal/buyers/store.go>) | Invitation ownership, business identity, consent and onboarding recovery; ListCustomers, SetInvitationGuard, SetAcceptanceGuard, CountBusinesses and 25 other declarations | [F028](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:121) |
| [internal/buyers/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/buyers/store_test.go>) | Invitation ownership, business identity, consent and onboarding recovery; TestBuyerInvitationIsSingleUseAndCreatesVerifiedPortal, TestSecondSupplierInvitationReusesVerifiedIdentity | No separate finding |

## internal/collections

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/collections/adapter.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/adapter.go>) | Debt eligibility, reservations, submission, retries and payment recognition; Valid, Allows, NewApprovedAdapter, Name and 10 other declarations | No separate finding |
| [internal/collections/adapter_test.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/adapter_test.go>) | Debt eligibility, reservations, submission, retries and payment recognition; TestApprovedAdapterRequiresWrittenApprovalAndPilotLimit, TestProviderStatusExposesSandboxCapabilities, TestApprovalCapabilitiesCannotBeChangedThroughSharedSlices | No separate finding |
| [internal/collections/admin_workflows_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/admin_workflows_postgres_test.go>) | Debt eligibility, reservations, submission, retries and payment recognition; workflowActors, proposal, TestAdminWorkflowIndependentCorrectionAndConcurrentApproval, TestAdminWorkflowStaleAndReservedCorrections and 3 other declarations | No separate finding |
| [internal/collections/audit_regression_test.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/audit_regression_test.go>) | Debt eligibility, reservations, submission, retries and payment recognition; Submit, Cancel, Capabilities, TestReservationWrapperPreservesPolicyCapabilities and 4 other declarations | No separate finding |
| [internal/collections/completion_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/completion_postgres_test.go>) | Debt eligibility, reservations, submission, retries and payment recognition; TestCompletionReportsReadDurableBalancesAndFailOnReadErrors, TestCompletionNoticeRequiresDeliveryAndWaitingPeriod, TestCompletionReconciliationCannotCloseUnresolvedDifference, TestCompletionReadsUnderRuntimeRole and 3 other declarations | No separate finding |
| [internal/collections/engine.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/engine.go>) | Debt eligibility, reservations, submission, retries and payment recognition; NewEngine, ProviderStatus, SetFeatureEnabled, SetMaxRetries and 14 other declarations | [F007](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:37), [F008](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:41), [F009](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:45) |
| [internal/collections/engine_test.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/engine_test.go>) | Debt eligibility, reservations, submission, retries and payment recognition; Name, Submit, Get, Sign and 18 other declarations | No separate finding |
| [internal/collections/fee_policy_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/fee_policy_postgres_test.go>) | Debt eligibility, reservations, submission, retries and payment recognition; TestCollectionUsesAcceptedFeeTermsRatherThanCurrentDefaults | No separate finding |
| [internal/collections/financial_audit_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/financial_audit_postgres_test.go>) | Debt eligibility, reservations, submission, retries and payment recognition; TestAuditFullPaymentReplayAndReversal, TestAuditForgivenessPreservedAndOperationKeysScoped, TestAuditWriteOffCannotConsumePendingDebit, TestPostgresReservationRejectsNewHoldAfterStaleSnapshot and 1 other declarations | No separate finding |
| [internal/collections/policy.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/policy.go>) | Debt eligibility, reservations, submission, retries and payment recognition; ValidatePolicy | No separate finding |
| [internal/collections/policy_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/policy_postgres_test.go>) | Debt eligibility, reservations, submission, retries and payment recognition; TestEligibilityReflectsAdminPauseBeforeFirstAttempt | No separate finding |
| [internal/collections/postgres.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/postgres.go>) | Debt eligibility, reservations, submission, retries and payment recognition; NewPostgresEngine, RequirePriorNotice, ProviderStatus, SetFeatureEnabled and 28 other declarations | [F007](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:37), [F021](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:93) |
| [internal/collections/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/postgres_test.go>) | Debt eligibility, reservations, submission, retries and payment recognition; TestPostgresCollectionIsRestartSafeAndFinanciallyIdempotent | No separate finding |
| [internal/collections/provider.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/provider.go>) | Debt eligibility, reservations, submission, retries and payment recognition; NewMockProvider, Name, Capabilities, SetNextResponse and 6 other declarations | No separate finding |
| [internal/collections/provider_contract_test.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/provider_contract_test.go>) | Debt eligibility, reservations, submission, retries and payment recognition; TestMockProviderContractScenarios | No separate finding |
| [internal/collections/resilient.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/resilient.go>) | Debt eligibility, reservations, submission, retries and payment recognition; NewResilientProvider, Name, Capabilities, Submit and 9 other declarations | No separate finding |
| [internal/collections/resilient_test.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/resilient_test.go>) | Debt eligibility, reservations, submission, retries and payment recognition; Name, Submit, Get, VerifyWebhook and 4 other declarations | No separate finding |
| [internal/collections/sweep_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/sweep_postgres_test.go>) | Debt eligibility, reservations, submission, retries and payment recognition; Record, Reverse, List, Get and 17 other declarations | [F040](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:169) |
| [internal/collections/webhook_provider.go](</Users/macbookpro/Documents/Kredit.com/internal/collections/webhook_provider.go>) | Debt eligibility, reservations, submission, retries and payment recognition; NewWebhookProvider, Name, Capabilities, Submit and 5 other declarations | [F005](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:29) |

## internal/config

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/config/admin_connections.go](</Users/macbookpro/Documents/Kredit.com/internal/config/admin_connections.go>) | Production startup requirements and provider configuration; ApplyStoredConnections, PublicConnectionValues, PrepareConnectionUpdate | No separate finding |
| [internal/config/admin_connections_test.go](</Users/macbookpro/Documents/Kredit.com/internal/config/admin_connections_test.go>) | Production startup requirements and provider configuration; Get, encodedConnection, TestAdminIdentityIsConsumedAtStartupWithoutChangingInfrastructure, TestAdminMonoPauseRetainsReconciliationAndSecret and 4 other declarations | No separate finding |
| [internal/config/config.go](</Users/macbookpro/Documents/Kredit.com/internal/config/config.go>) | Production startup requirements and provider configuration; Load, Validate, validateSecret, validateIdentifier and 6 other declarations | [F002](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:13) |
| [internal/config/config_test.go](</Users/macbookpro/Documents/Kredit.com/internal/config/config_test.go>) | Production startup requirements and provider configuration; TestLoadUsesNigerianMoneyDefaults, TestLoadParsesExternalDecisionFeatureGates, TestExternalDecisionFeatureGatesFailClosedWithoutEvidence, TestProductionRejectsDevelopmentSecrets and 19 other declarations | No separate finding |
| [internal/config/mono_production_test.go](</Users/macbookpro/Documents/Kredit.com/internal/config/mono_production_test.go>) | Production startup requirements and provider configuration; TestMonoProductionRequiresCertificationAndLiveCredentials, TestMonoStagingRefusesLiveCredential | No separate finding |

## internal/corrections

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/corrections/history.go](</Users/macbookpro/Documents/Kredit.com/internal/corrections/history.go>) | Correction requests, decisions and preserved history; ReadForBuyer | No separate finding |
| [internal/corrections/postgres.go](</Users/macbookpro/Documents/Kredit.com/internal/corrections/postgres.go>) | Correction requests, decisions and preserved history; NewPostgresStore, Open, StartReview, Decide and 4 other declarations | No separate finding |
| [internal/corrections/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/corrections/postgres_test.go>) | Correction requests, decisions and preserved history; TestPostgresCorrectionIsRestartSafeAndEnforcesSeparateReviewer | No separate finding |
| [internal/corrections/store.go](</Users/macbookpro/Documents/Kredit.com/internal/corrections/store.go>) | Correction requests, decisions and preserved history; NewStore, Open, StartReview, Decide and 5 other declarations | No separate finding |
| [internal/corrections/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/corrections/store_test.go>) | Correction requests, decisions and preserved history; TestCorrectionIsAppendOnlyAndAuditable | No separate finding |

## internal/credit

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/credit/agreement_bytes.go](</Users/macbookpro/Documents/Kredit.com/internal/credit/agreement_bytes.go>) | Sale state transitions, immutable acceptance, delivery and activation; UnmarshalJSON | No separate finding |
| [internal/credit/agreement_bytes_test.go](</Users/macbookpro/Documents/Kredit.com/internal/credit/agreement_bytes_test.go>) | Sale state transitions, immutable acceptance, delivery and activation; TestAgreementPreservesAcceptedHashAcrossJSONNormalization | No separate finding |
| [internal/credit/calendar.go](</Users/macbookpro/Documents/Kredit.com/internal/credit/calendar.go>) | Sale state transitions, immutable acceptance, delivery and activation; CollectionInstant, ExplicitCollectionInstant | No separate finding |
| [internal/credit/calendar_test.go](</Users/macbookpro/Documents/Kredit.com/internal/credit/calendar_test.go>) | Sale state transitions, immutable acceptance, delivery and activation; TestCollectionInstantIsLagosAndIndependentOfProcessTimezone, TestCollectionInstantRejectsInvalidTerms, TestCollectionInstantMonthAndLeapBoundaries, TestExplicitCollectionInstantRejectsNormalizedDates | No separate finding |
| [internal/credit/postgres.go](</Users/macbookpro/Documents/Kredit.com/internal/credit/postgres.go>) | Sale state transitions, immutable acceptance, delivery and activation; NewPostgresStore, SetDeemedAcceptanceNotice, deemedAcceptanceNoticeWindow, assertDeemedAcceptanceEvidence and 57 other declarations | [F021](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:93), [F022](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:97), [F031](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:133) |
| [internal/credit/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/credit/postgres_test.go>) | Sale state transitions, immutable acceptance, delivery and activation; TestPostgresCreditStoreImplementsServiceAndFailsClosedWithoutDatabase, nowForTest | No separate finding |
| [internal/credit/reads.go](</Users/macbookpro/Documents/Kredit.com/internal/credit/reads.go>) | Sale state transitions, immutable acceptance, delivery and activation; ReadForSupplier, ReadForBuyer, readViews | No separate finding |
| [internal/credit/store.go](</Users/macbookpro/Documents/Kredit.com/internal/credit/store.go>) | Sale state transitions, immutable acceptance, delivery and activation; SetCreationGuard, SetDeemedAcceptanceGate, deemedAcceptancePermitted, hasRespondedToAReleaseLocked and 41 other declarations | [F022](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:97), [F023](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:101), [F031](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:133) |
| [internal/credit/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/credit/store_test.go>) | Sale state transitions, immutable acceptance, delivery and activation; TestTradeLineDrawdownActivationCreatesOneObligationAndLedgerEntry, TestInstalmentTermsArePartOfImmutableAgreement, TestCustomInstalmentTermsMustEqualPrincipal, TestCreditLifecycleActivatesObligationAndPostsBalancedLedger and 12 other declarations | No separate finding |
| [internal/credit/system_acceptance_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/credit/system_acceptance_postgres_test.go>) | Sale state transitions, immutable acceptance, delivery and activation; TestSystemAcceptanceUsesSeparateEvidenceOnColdWorkers | No separate finding |

## internal/db

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/db/financial_context.go](</Users/macbookpro/Documents/Kredit.com/internal/db/financial_context.go>) | Runtime roles, transaction context and persistence requirements; SetTenantContext, SetObligationContext, GuardUnreservedReduction | No separate finding |
| [internal/db/notice_permissions_test.go](</Users/macbookpro/Documents/Kredit.com/internal/db/notice_permissions_test.go>) | Runtime roles, transaction context and persistence requirements; TestNoticeAcknowledgementRuntimePermissions | No separate finding |
| [internal/db/owner_permissions_test.go](</Users/macbookpro/Documents/Kredit.com/internal/db/owner_permissions_test.go>) | Runtime roles, transaction context and persistence requirements; TestOwnerProtectionsSurviveRoleProvisioning | No separate finding |
| [internal/db/persistence_contract.go](</Users/macbookpro/Documents/Kredit.com/internal/db/persistence_contract.go>) | Runtime roles, transaction context and persistence requirements; MissingPersistenceObjects, CheckPersistenceContract, missingPersistenceFunctions, missingPersistenceColumns and 1 other declarations | [F042](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:177) |
| [internal/db/persistence_contract_test.go](</Users/macbookpro/Documents/Kredit.com/internal/db/persistence_contract_test.go>) | Runtime roles, transaction context and persistence requirements; TestRequiredPersistenceObjectsAreUniqueAndQualified, TestPersistenceContractFeaturesAreDeclared, TestMissingPersistenceObjectsRequiresPool | No separate finding |
| [internal/db/postgres.go](</Users/macbookpro/Documents/Kredit.com/internal/db/postgres.go>) | Runtime roles, transaction context and persistence requirements; Open, OpenAsRole, open, setRuntimeDefault and 8 other declarations | No separate finding |
| [internal/db/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/db/postgres_test.go>) | Runtime roles, transaction context and persistence requirements; TestSetRuntimeDefaultPreservesExplicitDatabaseSetting, TestOpenAsRoleRequiresRole, TestOpenAsRoleConfinesRuntimeCredential, TestDatabasePoolRejectsInconsistentLimitsBeforeConnecting | No separate finding |
| [internal/db/relationship_permissions_test.go](</Users/macbookpro/Documents/Kredit.com/internal/db/relationship_permissions_test.go>) | Runtime roles, transaction context and persistence requirements; TestRelationshipChoicesBelongToBuyerAndRemainAppendOnly | No separate finding |
| [internal/db/schedule_adjustment.go](</Users/macbookpro/Documents/Kredit.com/internal/db/schedule_adjustment.go>) | Runtime roles, transaction context and persistence requirements; ReduceSchedulePrincipalTx | No separate finding |
| [internal/db/tenant_context.go](</Users/macbookpro/Documents/Kredit.com/internal/db/tenant_context.go>) | Runtime roles, transaction context and persistence requirements; WithTenantContext, TenantFromContext | No separate finding |

## internal/disputes

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/disputes/ledger_posting_test.go](</Users/macbookpro/Documents/Kredit.com/internal/disputes/ledger_posting_test.go>) | Contested balances, evidence, decisions and ledger adjustment; QueryRow, Exec, Scan, TestAdjustmentRequiresBothLedgerPostings | No separate finding |
| [internal/disputes/postgres.go](</Users/macbookpro/Documents/Kredit.com/internal/disputes/postgres.go>) | Contested balances, evidence, decisions and ledger adjustment; NewPostgresStore, Open, AddEvidence, Respond and 16 other declarations | [F030](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:129) |
| [internal/disputes/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/disputes/postgres_test.go>) | Contested balances, evidence, decisions and ledger adjustment; TestPostgresDisputeDecisionIsAtomicAndRestartSafe | No separate finding |
| [internal/disputes/store.go](</Users/macbookpro/Documents/Kredit.com/internal/disputes/store.go>) | Contested balances, evidence, decisions and ledger adjustment; NewStore, Open, AddEvidence, Respond and 10 other declarations | [F030](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:129) |
| [internal/disputes/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/disputes/store_test.go>) | Contested balances, evidence, decisions and ledger adjustment; TestPartialDisputeBlocksContestedAmountAndRecordsAdjustment, TestCallerCannotEscalateDisputeToFullBlock | No separate finding |

## internal/documents

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/documents/cleanup.go](</Users/macbookpro/Documents/Kredit.com/internal/documents/cleanup.go>) | Upload ownership, quarantine, scan state and authorized download; CleanupOrphans | No separate finding |
| [internal/documents/cleanup_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/documents/cleanup_postgres_test.go>) | Upload ownership, quarantine, scan state and authorized download; ListObjects, DeleteObject, TestCleanupPreservesCompletedAndRecentObjects | No separate finding |
| [internal/documents/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/documents/postgres_test.go>) | Upload ownership, quarantine, scan state and authorized download; TestPostgresStoreRoundTrip | No separate finding |
| [internal/documents/s3.go](</Users/macbookpro/Documents/Kredit.com/internal/documents/s3.go>) | Upload ownership, quarantine, scan state and authorized download; NewS3ObjectStore, Put, SignedURL, SignedUploadURL and 4 other declarations | [F038](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:161) |
| [internal/documents/s3_test.go](</Users/macbookpro/Documents/Kredit.com/internal/documents/s3_test.go>) | Upload ownership, quarantine, scan state and authorized download; TestSignedUploadBindsWriteOnceAndEncryptionHeaders | No separate finding |
| [internal/documents/scanner.go](</Users/macbookpro/Documents/Kredit.com/internal/documents/scanner.go>) | Upload ownership, quarantine, scan state and authorized download; Scan, NewWebhookScanner, PendingScanIDs | No separate finding |
| [internal/documents/scanner_test.go](</Users/macbookpro/Documents/Kredit.com/internal/documents/scanner_test.go>) | Upload ownership, quarantine, scan state and authorized download; TestDevelopmentScannerReleasesQuarantinedDocument, TestWebhookScannerUsesStableIdempotencyKey, RoundTrip, Scan and 2 other declarations | No separate finding |
| [internal/documents/store.go](</Users/macbookpro/Documents/Kredit.com/internal/documents/store.go>) | Upload ownership, quarantine, scan state and authorized download; NewUnavailableObjectStore, Put, SignedURL, NewStore and 26 other declarations | No separate finding |
| [internal/documents/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/documents/store_test.go>) | Upload ownership, quarantine, scan state and authorized download; TestDocumentRequiresCleanScanBeforeDownload, TestConcurrentUploadSlotsRespectPerUserQuota, TestDirectUploadSlotCreatesQuarantinedMetadata, TestDocumentReadsKeepTenantAndOutageOutcomesDistinct and 2 other declarations | No separate finding |

## internal/feedback

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/feedback/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/feedback/postgres_test.go>) | Feedback identity, bounded input and replay; TestPostgresFeedbackReturnsStoredMonthlyAnswer | No separate finding |
| [internal/feedback/store.go](</Users/macbookpro/Documents/Kredit.com/internal/feedback/store.go>) | Feedback identity, bounded input and replay; NewStore, NewPostgresStore, Submit | No separate finding |
| [internal/feedback/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/feedback/store_test.go>) | Feedback identity, bounded input and replay; TestSubmitValidatesAndStoresPrivacySafeFeedback, TestSubmitRejectsUnscopedOrFreeFormValues, TestSubmitCountsOneAnswerPerPersonAndPageEachMonth | No separate finding |

## internal/idempotency

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/idempotency/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/idempotency/postgres_test.go>) | Request identity, saved outcomes and uncertain retries; TestPostgresStoreRuntimeRoleLifecycle | No separate finding |
| [internal/idempotency/store.go](</Users/macbookpro/Documents/Kredit.com/internal/idempotency/store.go>) | Request identity, saved outcomes and uncertain retries; HashRequest, NewMemoryStore, Reserve, Complete and 1 other declarations | No separate finding |
| [internal/idempotency/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/idempotency/store_test.go>) | Request identity, saved outcomes and uncertain retries; TestMemoryStoreRejectsKeyReuseWithDifferentRequest, TestMemoryStoreValidatesCompletionAndReplaysExactResponse, TestUnfinishedReservationDoesNotExpireIntoAnotherMutation, TestReplayCannotMutateSavedResponse | No separate finding |

## internal/identifier

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/identifier/id.go](</Users/macbookpro/Documents/Kredit.com/internal/identifier/id.go>) | Identifier normalization and collision boundaries; New, FromKey | No separate finding |
| [internal/identifier/id_test.go](</Users/macbookpro/Documents/Kredit.com/internal/identifier/id_test.go>) | Identifier normalization and collision boundaries; TestNew, TestFromKey | No separate finding |

## internal/identity

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/identity/provider.go](</Users/macbookpro/Documents/Kredit.com/internal/identity/provider.go>) | Provider verification results and sensitive data handling; NewUnavailableProvider, Name, Capabilities, CreatePersonVerification and 7 other declarations | No separate finding |
| [internal/identity/provider_test.go](</Users/macbookpro/Documents/Kredit.com/internal/identity/provider_test.go>) | Provider verification results and sensitive data handling; TestMockProviderReturnsSafeVerifiedResults, TestUnavailableProviderFailsClosed | No separate finding |
| [internal/identity/safe_result.go](</Users/macbookpro/Documents/Kredit.com/internal/identity/safe_result.go>) | Provider verification results and sensitive data handling; SafeVerificationResult, hasLongDigitRun | No separate finding |
| [internal/identity/safe_result_test.go](</Users/macbookpro/Documents/Kredit.com/internal/identity/safe_result_test.go>) | Provider verification results and sensitive data handling; TestVerificationFactsDiscardRawIdentityAndInvalidStatus | No separate finding |
| [internal/identity/webhook.go](</Users/macbookpro/Documents/Kredit.com/internal/identity/webhook.go>) | Provider verification results and sensitive data handling; NewWebhookProvider, Name, Capabilities, CreatePersonVerification and 6 other declarations | [F005](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:29) |

## internal/jobs

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/jobs/client.go](</Users/macbookpro/Documents/Kredit.com/internal/jobs/client.go>) | Durable work, discovery, retries, leases and financial maintenance; Kind, InsertOpts, Work, IsMiddleware and 17 other declarations | [F024](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:105) |
| [internal/jobs/client_test.go](</Users/macbookpro/Documents/Kredit.com/internal/jobs/client_test.go>) | Durable work, discovery, retries, leases and financial maintenance; TestMaintenanceArgsKind, TestMaintenanceOperationsIncludeScheduleEvaluation, TestJobClassesUseDedicatedQueuesAndRetryBudgets | No separate finding |
| [internal/jobs/collection_requeue_test.go](</Users/macbookpro/Documents/Kredit.com/internal/jobs/collection_requeue_test.go>) | Durable work, discovery, retries, leases and financial maintenance; TestPeriodicCollectionReconciliationCanRunAgainAfterCompletion, TestAllPeriodicJobsCanRepeatAfterCompletion | No separate finding |
| [internal/jobs/dead_letter.go](</Users/macbookpro/Documents/Kredit.com/internal/jobs/dead_letter.go>) | Durable work, discovery, retries, leases and financial maintenance; NewDeadLetterHandler, HandleError, HandlePanic | No separate finding |
| [internal/jobs/provider_inbox_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/jobs/provider_inbox_postgres_test.go>) | Durable work, discovery, retries, leases and financial maintenance; TestProviderInboxSerializesDeliveryAndRejectsChangedReplay | No separate finding |
| [internal/jobs/reconciliation_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/jobs/reconciliation_postgres_test.go>) | Durable work, discovery, retries, leases and financial maintenance; TestReconciliationDetectsEmptyAndUnbalancedJournals | No separate finding |
| [internal/jobs/telemetry_test.go](</Users/macbookpro/Documents/Kredit.com/internal/jobs/telemetry_test.go>) | Durable work, discovery, retries, leases and financial maintenance; TestTelemetryMiddlewareRecordsJobOutcome | No separate finding |

## internal/ledger

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/ledger/fee_terms.go](</Users/macbookpro/Documents/Kredit.com/internal/ledger/fee_terms.go>) | Exact kobo amounts, balanced postings, reversals and immutable journals; Rates, Clone, Validate, Base and 6 other declarations | No separate finding |
| [internal/ledger/fee_terms_test.go](</Users/macbookpro/Documents/Kredit.com/internal/ledger/fee_terms_test.go>) | Exact kobo amounts, balanced postings, reversals and immutable journals; TestRecordedFeesAndZeroFeeJournal, TestMinFeeFloorEnforcement, TestFeeRoundingAlwaysFavoursTheSupplier | No separate finding |
| [internal/ledger/postgres.go](</Users/macbookpro/Documents/Kredit.com/internal/ledger/postgres.go>) | Exact kobo amounts, balanced postings, reversals and immutable journals; NewPostgresStore, NewPostgresStoreWithOutbox, PostPayment, PostPaymentReversal and 13 other declarations | No separate finding |
| [internal/ledger/reconcile.go](</Users/macbookpro/Documents/Kredit.com/internal/ledger/reconcile.go>) | Exact kobo amounts, balanced postings, reversals and immutable journals; Reconcile, reconcileQuery | No separate finding |
| [internal/ledger/reconcile_test.go](</Users/macbookpro/Documents/Kredit.com/internal/ledger/reconcile_test.go>) | Exact kobo amounts, balanced postings, reversals and immutable journals; TestReconcileRequiresDatabase, TestReconcileFindsJournalWithoutPostings | No separate finding |
| [internal/ledger/store.go](</Users/macbookpro/Documents/Kredit.com/internal/ledger/store.go>) | Exact kobo amounts, balanced postings, reversals and immutable journals; NewStore, PostPayment, PostPaymentReversal, isRecognizedPaymentSource and 14 other declarations | No separate finding |
| [internal/ledger/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/ledger/store_test.go>) | Exact kobo amounts, balanced postings, reversals and immutable journals; TestActivationPostsBalancedPrincipalAndBaseFee, TestLedgerRejectsChangedIntentForIdempotencyKey, TestMoneyArithmeticRejectsOverflow, TestVerifyChain and 3 other declarations | No separate finding |

## internal/legalpublication

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/legalpublication/current.go](</Users/macbookpro/Documents/Kredit.com/internal/legalpublication/current.go>) | Stable legal versions and accepted-record references; Initial, SettingsReader, Resolve | No separate finding |
| [internal/legalpublication/current_test.go](</Users/macbookpro/Documents/Kredit.com/internal/legalpublication/current_test.go>) | Stable legal versions and accepted-record references; Get, TestCurrentDocumentsUseVerifiedPublicationOnly | No separate finding |
| [internal/legalpublication/versions.go](</Users/macbookpro/Documents/Kredit.com/internal/legalpublication/versions.go>) | Stable legal versions and accepted-record references | No separate finding |

## internal/mandates

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/mandates/connector_contract_test.go](</Users/macbookpro/Documents/Kredit.com/internal/mandates/connector_contract_test.go>) | Authorization scope, persistence, provider state and cancellation; TestConnectorsRejectRedirects, TestMandateConnectorValidatesIdentityAndCancellation, TestCollectionConnectorRejectsMismatchedLookup | No separate finding |
| [internal/mandates/provider.go](</Users/macbookpro/Documents/Kredit.com/internal/mandates/provider.go>) | Authorization scope, persistence, provider state and cancellation; NewPostgresProviderWithRemote, NewPostgresProvider, Name, CreateAuthorizationSession and 8 other declarations | [F010](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:49) |
| [internal/mandates/provider_test.go](</Users/macbookpro/Documents/Kredit.com/internal/mandates/provider_test.go>) | Authorization scope, persistence, provider state and cancellation; TestPostgresProviderFailsClosedWithoutDatabase, TestMockMandateActivatesWithCeiling, TestMockProviderResolvesTradeLineMandateByOwner | No separate finding |
| [internal/mandates/reads.go](</Users/macbookpro/Documents/Kredit.com/internal/mandates/reads.go>) | Authorization scope, persistence, provider state and cancellation; ReadForBuyer | No separate finding |
| [internal/mandates/sweep_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/mandates/sweep_postgres_test.go>) | Authorization scope, persistence, provider state and cancellation; Name, CreateAuthorizationSession, GetMandate, RestoreAuthorization and 1 other declarations | No separate finding |
| [internal/mandates/unavailable.go](</Users/macbookpro/Documents/Kredit.com/internal/mandates/unavailable.go>) | Authorization scope, persistence, provider state and cancellation; NewUnavailableProvider, Name, CreateAuthorizationSession, GetMandate and 2 other declarations | No separate finding |
| [internal/mandates/webhook.go](</Users/macbookpro/Documents/Kredit.com/internal/mandates/webhook.go>) | Authorization scope, persistence, provider state and cancellation; NewWebhookProvider, Name, CreateAuthorizationSession, GetMandate and 4 other declarations | [F005](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:29) |

## internal/notifications

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/notifications/configured.go](</Users/macbookpro/Documents/Kredit.com/internal/notifications/configured.go>) | Recipient identity, delivery retries, links and authenticated receipts; NewConfiguredProvider, Channel, ResolveConnector, Send | No separate finding |
| [internal/notifications/configured_test.go](</Users/macbookpro/Documents/Kredit.com/internal/notifications/configured_test.go>) | Recipient identity, delivery retries, links and authenticated receipts; Get, TestConfiguredDeliveryRotationDisableAndReadFailure | No separate finding |
| [internal/notifications/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/notifications/postgres_test.go>) | Recipient identity, delivery retries, links and authenticated receipts; TestPostgresNotificationIsDurableAndDeduplicatedBeforeProviderSend, TestScheduledNotificationIsRecoveredAndDeliveredOnce, TestScheduledSupplierReminderRechecksWithdrawnConsent, TestPostgresPreferenceVersionAndFutureSuppression | No separate finding |
| [internal/notifications/receipts.go](</Users/macbookpro/Documents/Kredit.com/internal/notifications/receipts.go>) | Recipient identity, delivery retries, links and authenticated receipts; RecordDeliveryReceipt | [F001](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:7) |
| [internal/notifications/store.go](</Users/macbookpro/Documents/Kredit.com/internal/notifications/store.go>) | Recipient identity, delivery retries, links and authenticated receipts; SendRecoveryInstructions, SendOTP, SendInvitation, NewMockProvider and 43 other declarations | [F012](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:57) |
| [internal/notifications/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/notifications/store_test.go>) | Recipient identity, delivery retries, links and authenticated receipts; TestCriticalNotificationFallsBackAndDeduplicates, TestNotificationReplayCannotChangeRecipientOrTerms, TestRoutineNotificationRespectsQuietHoursAndSecureLink, TestQuietHoursBoundariesAndCalendarRollover and 5 other declarations | No separate finding |
| [internal/notifications/webhook.go](</Users/macbookpro/Documents/Kredit.com/internal/notifications/webhook.go>) | Recipient identity, delivery retries, links and authenticated receipts; NewWebhookProvider, Channel, Send | [F001](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:7), [F005](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:29) |
| [internal/notifications/webhook_test.go](</Users/macbookpro/Documents/Kredit.com/internal/notifications/webhook_test.go>) | Recipient identity, delivery retries, links and authenticated receipts; RoundTrip, TestWebhookProviderDeliversOTPWithoutLoggingCredential | No separate finding |

## internal/observability

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/observability/financial.go](</Users/macbookpro/Documents/Kredit.com/internal/observability/financial.go>) | Redacted telemetry, financial metrics and alert semantics; DurableFinancialMetrics | No separate finding |
| [internal/observability/phase5_financial_integration_test.go](</Users/macbookpro/Documents/Kredit.com/internal/observability/phase5_financial_integration_test.go>) | Redacted telemetry, financial metrics and alert semantics; TestPhase5GlobalMetricsPreserveTenantIsolation | No separate finding |
| [internal/observability/phase5_financial_test.go](</Users/macbookpro/Documents/Kredit.com/internal/observability/phase5_financial_test.go>) | Redacted telemetry, financial metrics and alert semantics; TestPhase5FinancialMetricsRejectMissingDatabase | No separate finding |
| [internal/observability/store.go](</Users/macbookpro/Documents/Kredit.com/internal/observability/store.go>) | Redacted telemetry, financial metrics and alert semantics; NewStore, Inc, ObserveDuration, metricName and 2 other declarations | No separate finding |
| [internal/observability/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/observability/store_test.go>) | Redacted telemetry, financial metrics and alert semantics; TestStoreRecordsBoundedDurationSummaries, TestStoreRejectsUnboundedMetricNames, TestDurationCountersRemainCumulativeWhenSamplesRotate | No separate finding |
| [internal/observability/tracer.go](</Users/macbookpro/Documents/Kredit.com/internal/observability/tracer.go>) | Redacted telemetry, financial metrics and alert semantics; NewNoopTracer, NewTracer, Start, Shutdown and 1 other declarations | No separate finding |
| [internal/observability/tracer_test.go](</Users/macbookpro/Documents/Kredit.com/internal/observability/tracer_test.go>) | Redacted telemetry, financial metrics and alert semantics; TestNoopTracerStartsSafeSpan, TestExtractTraceContextOnlyUsesTraceparent | No separate finding |

## internal/onboarding

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/onboarding/postgres.go](</Users/macbookpro/Documents/Kredit.com/internal/onboarding/postgres.go>) | Supplier readiness, verified contacts, settlement and billing; NewPostgresStore, scanProfile, begin, Ensure and 18 other declarations | No separate finding |
| [internal/onboarding/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/onboarding/postgres_test.go>) | Supplier readiness, verified contacts, settlement and billing; TestPostgresOnboardingPersistsVersionsMasksSettlementAndReconcilesExpiry, containsText | No separate finding |
| [internal/onboarding/store.go](</Users/macbookpro/Documents/Kredit.com/internal/onboarding/store.go>) | Supplier readiness, verified contacts, settlement and billing; NewStore, Ensure, Get, mutate and 15 other declarations | [F018](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:81), [F019](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:85) |
| [internal/onboarding/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/onboarding/store_test.go>) | Supplier readiness, verified contacts, settlement and billing; TestReadinessIsDerivedAndProviderExpiryRevokesIt, TestProviderStatesVersionConflictsAndRestrictedSettlementInput, TestKYBDecisionIsBoundToProviderReferenceAndVersion, TestFinanceMFAChangesPreserveOwnerVerification and 1 other declarations | No separate finding |

## internal/operations

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/operations/approvals.go](</Users/macbookpro/Documents/Kredit.com/internal/operations/approvals.go>) | Protected financial corrections, approval and ledger effects; adminRole, financialSnapshot, validateChange, ProposeChange and 4 other declarations | No separate finding |
| [internal/operations/ledger_posting_test.go](</Users/macbookpro/Documents/Kredit.com/internal/operations/ledger_posting_test.go>) | Protected financial corrections, approval and ledger effects; QueryRow, Exec, Scan, TestAdjustmentRequiresBothLedgerPostings | No separate finding |
| [internal/operations/postgres.go](</Users/macbookpro/Documents/Kredit.com/internal/operations/postgres.go>) | Protected financial corrections, approval and ledger effects; NewPostgresStore, WriteOff, WaiveFee, WriteOffWithKey and 7 other declarations | No separate finding |
| [internal/operations/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/operations/postgres_test.go>) | Protected financial corrections, approval and ledger effects; TestPostgresWriteOffIsAtomicIdempotentAndRestartSafe | No separate finding |
| [internal/operations/store.go](</Users/macbookpro/Documents/Kredit.com/internal/operations/store.go>) | Protected financial corrections, approval and ledger effects; NewStore, WriteOff, WaiveFee, adjust and 3 other declarations | No separate finding |
| [internal/operations/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/operations/store_test.go>) | Protected financial corrections, approval and ledger effects; TestWriteOffRequiresApprovalAtHighValueAndReducesBalance, TestFeeWaiverIsAudited, TestClaimedApprovalCannotAuthorizeHighValueOperation, TestUnavailableOperationHistoryIsAnError | No separate finding |

## internal/organizations

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/organizations/audit_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/organizations/audit_postgres_test.go>) | Membership authority, invitations, account limits and lifecycle; TestPostgresInactiveMembershipAndOwnerRoleAreProtected | No separate finding |
| [internal/organizations/invitation_lifecycle_test.go](</Users/macbookpro/Documents/Kredit.com/internal/organizations/invitation_lifecycle_test.go>) | Membership authority, invitations, account limits and lifecycle; TestTeamInvitationLifecycle, TestRestrictedTeamInvitationLifecycle | No separate finding |
| [internal/organizations/limit_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/organizations/limit_postgres_test.go>) | Membership authority, invitations, account limits and lifecycle; TestConcurrentOrganizationCreationHonorsLimit | No separate finding |
| [internal/organizations/postgres.go](</Users/macbookpro/Documents/Kredit.com/internal/organizations/postgres.go>) | Membership authority, invitations, account limits and lifecycle; NewPostgresStore, SetCreateGuard, SetOrganizationLimit, Count and 19 other declarations | No separate finding |
| [internal/organizations/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/organizations/postgres_test.go>) | Membership authority, invitations, account limits and lifecycle; TestPostgresStoreUsesOpaqueTargetHashesAndUUIDv7IDs, TestPostgresStoreImplementsOrganizationService | No separate finding |
| [internal/organizations/store.go](</Users/macbookpro/Documents/Kredit.com/internal/organizations/store.go>) | Membership authority, invitations, account limits and lifecycle; SetCreateGuard, Count, NewStore, Create and 18 other declarations | No separate finding |
| [internal/organizations/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/organizations/store_test.go>) | Membership authority, invitations, account limits and lifecycle; TestOrganizationMembershipsAreTenantScoped, TestUnregisteredBusinessCanCreateOrganization, TestMembershipStatusChangesProtectOwnerAndActor, TestInvitationActivatesAfterUserAuthentication and 2 other declarations | No separate finding |

## internal/outbox

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/outbox/dispatcher.go](</Users/macbookpro/Documents/Kredit.com/internal/outbox/dispatcher.go>) | Committed event publication, leases and duplicate handling; Publish, NewDispatcher, DispatchOnce, retryDelay | No separate finding |
| [internal/outbox/dispatcher_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/outbox/dispatcher_postgres_test.go>) | Committed event publication, leases and duplicate handling; TestDispatcherPublishesCommittedEventOnce, TestOutboxRejectsConflictingReplay, TestExpiredPublisherCannotOverwriteNewClaim | No separate finding |
| [internal/outbox/dispatcher_test.go](</Users/macbookpro/Documents/Kredit.com/internal/outbox/dispatcher_test.go>) | Committed event publication, leases and duplicate handling; TestRetryDelayIsBounded | No separate finding |
| [internal/outbox/store.go](</Users/macbookpro/Documents/Kredit.com/internal/outbox/store.go>) | Committed event publication, leases and duplicate handling; NewStore, AppendTx, Claim, MarkPublished and 1 other declarations | No separate finding |

## internal/paymentclaims

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/paymentclaims/postgres.go](</Users/macbookpro/Documents/Kredit.com/internal/paymentclaims/postgres.go>) | Reported transfers, temporary holds and supplier confirmation; NewPostgresStore, Create, beginScoped, Get and 11 other declarations | No separate finding |
| [internal/paymentclaims/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/paymentclaims/postgres_test.go>) | Reported transfers, temporary holds and supplier confirmation; TestPostgresClaimIsRestartSafeAndExpiresHold, TestUnavailablePaymentClaimCheckBlocksCollection | No separate finding |
| [internal/paymentclaims/store.go](</Users/macbookpro/Documents/Kredit.com/internal/paymentclaims/store.go>) | Reported transfers, temporary holds and supplier confirmation; NewStore, Create, Get, ListForObligation and 9 other declarations | No separate finding |
| [internal/paymentclaims/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/paymentclaims/store_test.go>) | Reported transfers, temporary holds and supplier confirmation; TestClaimHoldExpiresAndDecisionRequiresPayment, TestClaimHoldsCannotOverflowAndReturnedReviewsCannotChangeHistory, TestClaimRetryRejectsChangedOwnerAndIntent | No separate finding |

## internal/payments

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/payments/ledger_posting_test.go](</Users/macbookpro/Documents/Kredit.com/internal/payments/ledger_posting_test.go>) | Payment recognition, allocation, reversal and tenant scope; QueryRow, Exec, Scan, TestPaymentRequiresBothLedgerPostings | No separate finding |
| [internal/payments/postgres.go](</Users/macbookpro/Documents/Kredit.com/internal/payments/postgres.go>) | Payment recognition, allocation, reversal and tenant scope; NewPostgresStore, Record, RecordContext, AfterCommit and 18 other declarations | [F007](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:37), [F015](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:69) |
| [internal/payments/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/payments/postgres_test.go>) | Payment recognition, allocation, reversal and tenant scope; newPaymentFixture, store, TestPostgresPaymentIsAtomicIdempotentAndRestartSafe, TestPostgresPaymentReplayPreservesDatabaseTimestampPrecision and 3 other declarations | No separate finding |
| [internal/payments/receipt.go](</Users/macbookpro/Documents/Kredit.com/internal/payments/receipt.go>) | Payment recognition, allocation, reversal and tenant scope; PublicReceiptContext | No separate finding |
| [internal/payments/receipt_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/payments/receipt_postgres_test.go>) | Payment recognition, allocation, reversal and tenant scope; TestPublicReceiptWithRuntimeRole | No separate finding |
| [internal/payments/store.go](</Users/macbookpro/Documents/Kredit.com/internal/payments/store.go>) | Payment recognition, allocation, reversal and tenant scope; NewStore, NewStoreWithAllocator, SetCollectedMarker, SetCollectedReversalMarker and 18 other declarations | No separate finding |
| [internal/payments/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/payments/store_test.go>) | Payment recognition, allocation, reversal and tenant scope; TestRecordAllocationReversalAndCollectionFee, TestCollectedPaymentRequiresWorkerAttemptProvenance, TestScheduleAwarePaymentAllocatesAndReversesItem, TestPaymentCannotExceedOutstanding and 2 other declarations | No separate finding |

## internal/platform

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/platform/logging/logging.go](</Users/macbookpro/Documents/Kredit.com/internal/platform/logging/logging.go>) | Shared platform definitions and contracts; New, NewSanitizingHandler, Enabled, Handle and 5 other declarations | No separate finding |
| [internal/platform/logging/sanitize.go](</Users/macbookpro/Documents/Kredit.com/internal/platform/logging/sanitize.go>) | Shared platform definitions and contracts; Redact, SafePath, SafeAttributes | No separate finding |
| [internal/platform/logging/sanitize_test.go](</Users/macbookpro/Documents/Kredit.com/internal/platform/logging/sanitize_test.go>) | Shared platform definitions and contracts; TestSafePathRemovesQueriesAndOpaqueIdentifiers, TestSafeAttributesRedactsRestrictedKeys, TestSanitizingHandlerRedactsErrorAndSensitiveAttributes, TestSafePathRedactsPublicFinancialLinks and 2 other declarations | No separate finding |

## internal/platformops

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/platformops/controls.go](</Users/macbookpro/Documents/Kredit.com/internal/platformops/controls.go>) | Administrator reads, controls, authority and reconciliation; ReplayCommand, PreflightCommand, PreviewCommand, previewCommand and 7 other declarations | No separate finding |
| [internal/platformops/controls_test.go](</Users/macbookpro/Documents/Kredit.com/internal/platformops/controls_test.go>) | Administrator reads, controls, authority and reconciliation; TestEveryCommandRequiresSafetyEnvelope, TestControlledSuspendRestoreHoldAndIdempotency, TestCommandTargetCannotMisdirectAuditOrNotification | No separate finding |
| [internal/platformops/directory_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/platformops/directory_postgres_test.go>) | Administrator reads, controls, authority and reconciliation; TestUserDirectoryIncludesCurrentControlVersion | [F040](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:169) |
| [internal/platformops/external_commands.go](</Users/macbookpro/Documents/Kredit.com/internal/platformops/external_commands.go>) | Administrator reads, controls, authority and reconciliation; ExternalCommand, BeginExternalCommand, FinishExternalCommand, CommandPermission | No separate finding |
| [internal/platformops/external_commands_test.go](</Users/macbookpro/Documents/Kredit.com/internal/platformops/external_commands_test.go>) | Administrator reads, controls, authority and reconciliation; TestExternalIntentIsDurableAndClaimedOnce | No separate finding |
| [internal/platformops/financial_review.go](</Users/macbookpro/Documents/Kredit.com/internal/platformops/financial_review.go>) | Administrator reads, controls, authority and reconciliation; RefreshFinancialReviews, FinancialReviews, DecideFinancialReview | [F017](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:77) |
| [internal/platformops/store.go](</Users/macbookpro/Documents/Kredit.com/internal/platformops/store.go>) | Administrator reads, controls, authority and reconciliation; NewStore, Overview, Jobs, ProviderEvents and 14 other declarations | [F015](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:69), [F016](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:73) |

## internal/platformsettings

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/platformsettings/connector_test.go](</Users/macbookpro/Documents/Kredit.com/internal/platformsettings/connector_test.go>) | Encrypted settings, current authority and reviewed publication; TestConnectorValidation | No separate finding |
| [internal/platformsettings/crypto.go](</Users/macbookpro/Documents/Kredit.com/internal/platformsettings/crypto.go>) | Encrypted settings, current authority and reviewed publication; NewEncryptor, Ready, KeyID, gcm and 5 other declarations | No separate finding |
| [internal/platformsettings/crypto_root_test.go](</Users/macbookpro/Documents/Kredit.com/internal/platformsettings/crypto_root_test.go>) | Encrypted settings, current authority and reviewed publication; TestMissingEncryptionRootFailsClosed, TestShortEncryptionRootFailsClosed, TestRegistryRejectsUnknownNullAndManualVerification | No separate finding |
| [internal/platformsettings/guide_catalog.json](</Users/macbookpro/Documents/Kredit.com/internal/platformsettings/guide_catalog.json>) | Encrypted settings, current authority and reviewed publication | No separate finding |
| [internal/platformsettings/owner_lifecycle_integration_test.go](</Users/macbookpro/Documents/Kredit.com/internal/platformsettings/owner_lifecycle_integration_test.go>) | Encrypted settings, current authority and reviewed publication; TestOwnerLifecycleDatabaseInvariant | No separate finding |
| [internal/platformsettings/registry.go](</Users/macbookpro/Documents/Kredit.com/internal/platformsettings/registry.go>) | Encrypted settings, current authority and reviewed publication; validateBool, ValidateKeyAndValue, init, validateNotificationConnector and 1 other declarations | No separate finding |
| [internal/platformsettings/runtime_connections.go](</Users/macbookpro/Documents/Kredit.com/internal/platformsettings/runtime_connections.go>) | Encrypted settings, current authority and reviewed publication; init, DecodeRuntimeConnection | No separate finding |
| [internal/platformsettings/store.go](</Users/macbookpro/Documents/Kredit.com/internal/platformsettings/store.go>) | Encrypted settings, current authority and reviewed publication; NewPostgresStore, GetAll, Get, get and 11 other declarations | No separate finding |
| [internal/platformsettings/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/platformsettings/store_test.go>) | Encrypted settings, current authority and reviewed publication; TestCryptoEncryptionDecryptionAndMasking, TestRegistryValidationRules, keys, TestPostgresStoreIntegration and 1 other declarations | No separate finding |
| [internal/platformsettings/website_content.go](</Users/macbookpro/Documents/Kredit.com/internal/platformsettings/website_content.go>) | Encrypted settings, current authority and reviewed publication; init, validateWebsiteText, ValidateWebsiteCopy, validateWebsiteContent and 4 other declarations | No separate finding |
| [internal/platformsettings/website_guides.go](</Users/macbookpro/Documents/Kredit.com/internal/platformsettings/website_guides.go>) | Encrypted settings, current authority and reviewed publication; init, IsGuidePage, ValidateWebsitePageCopy, validateEditorialExtras and 1 other declarations | No separate finding |
| [internal/platformsettings/website_guides_test.go](</Users/macbookpro/Documents/Kredit.com/internal/platformsettings/website_guides_test.go>) | Encrypted settings, current authority and reviewed publication; TestEditorialValidationAndLegacyHash | No separate finding |

## internal/providers

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/providers/mono/customer.go](</Users/macbookpro/Documents/Kredit.com/internal/providers/mono/customer.go>) | Mono request/response contracts, callbacks and uncertain outcomes; CreateCustomer, ValidateCustomerInput, CustomerIdentity | No separate finding |
| [internal/providers/mono/export_phase4_test.go](</Users/macbookpro/Documents/Kredit.com/internal/providers/mono/export_phase4_test.go>) | Mono request/response contracts, callbacks and uncertain outcomes; NewPhase4FixtureClient | No separate finding |
| [internal/providers/mono/fuzz_test.go](</Users/macbookpro/Documents/Kredit.com/internal/providers/mono/fuzz_test.go>) | Mono request/response contracts, callbacks and uncertain outcomes; FuzzWebhookParsingAfterAuthentication | No separate finding |
| [internal/providers/mono/hosted_url.go](</Users/macbookpro/Documents/Kredit.com/internal/providers/mono/hosted_url.go>) | Mono request/response contracts, callbacks and uncertain outcomes; validateHostedAuthorizationURL | No separate finding |
| [internal/providers/mono/hosted_url_test.go](</Users/macbookpro/Documents/Kredit.com/internal/providers/mono/hosted_url_test.go>) | Mono request/response contracts, callbacks and uncertain outcomes; TestHostedAuthorizationURLRejectsUnapprovedDestinations | No separate finding |
| [internal/providers/mono/mono.go](</Users/macbookpro/Documents/Kredit.com/internal/providers/mono/mono.go>) | Mono request/response contracts, callbacks and uncertain outcomes; New, NewLive, ReconciliationOnly, Name and 16 other declarations | [F008](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:41), [F009](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:45), [F010](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:49) |
| [internal/providers/mono/mono_test.go](</Users/macbookpro/Documents/Kredit.com/internal/providers/mono/mono_test.go>) | Mono request/response contracts, callbacks and uncertain outcomes; testClient, TestVariableSweepAuthorizationIsPending, TestActivationRequiresReadyFlag, TestPartialSweepUsesCollectedAmountNotRequestedAmount and 8 other declarations | No separate finding |
| [internal/providers/mono/phase4_contract_test.go](</Users/macbookpro/Documents/Kredit.com/internal/providers/mono/phase4_contract_test.go>) | Mono request/response contracts, callbacks and uncertain outcomes; TestPhase4DebitOutcomeMatrix, TestPhase4LookupRejectsUnconfirmedIdentity, RoundTrip, TestPhase4LostSubmissionResponseUsesOriginalReference and 5 other declarations | [F008](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:41) |
| [internal/providers/mono/phase4_persistence_test.go](</Users/macbookpro/Documents/Kredit.com/internal/providers/mono/phase4_persistence_test.go>) | Mono request/response contracts, callbacks and uncertain outcomes; serve, result, Record, phase4NewFixture and 10 other declarations | [F040](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:169) |
| [internal/providers/mono/webhook.go](</Users/macbookpro/Documents/Kredit.com/internal/providers/mono/webhook.go>) | Mono request/response contracts, callbacks and uncertain outcomes; ParseWebhook | [F026](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:113) |

## internal/publictoken

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/publictoken/fuzz_test.go](</Users/macbookpro/Documents/Kredit.com/internal/publictoken/fuzz_test.go>) | Signed public links, expiry, purpose and identity binding; FuzzTokenReferenceParsing | No separate finding |
| [internal/publictoken/token.go](</Users/macbookpro/Documents/Kredit.com/internal/publictoken/token.go>) | Signed public links, expiry, purpose and identity binding; Issue, Parse | No separate finding |
| [internal/publictoken/token_test.go](</Users/macbookpro/Documents/Kredit.com/internal/publictoken/token_test.go>) | Signed public links, expiry, purpose and identity binding; TestPurposeAndExpiryAreBound | No separate finding |

## internal/readiness

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/readiness/readiness.go](</Users/macbookpro/Documents/Kredit.com/internal/readiness/readiness.go>) | Reported capability versus configuration and actual evidence; Evaluate | No separate finding |
| [internal/readiness/readiness_test.go](</Users/macbookpro/Documents/Kredit.com/internal/readiness/readiness_test.go>) | Reported capability versus configuration and actual evidence; TestEvaluateRequiresEvidenceAndPositivePilotLimits, TestProductionReadinessRequiresRealProviders | No separate finding |

## internal/relationships

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/relationships/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/relationships/postgres_test.go>) | Buyer consent and sharing boundaries; TestPostgresStoreFailsClosedWithoutDatabase, TestStoreImplementsService | No separate finding |
| [internal/relationships/store.go](</Users/macbookpro/Documents/Kredit.com/internal/relationships/store.go>) | Buyer consent and sharing boundaries; NewStore, Record, List, NewPostgresStore and 1 other declarations | No separate finding |
| [internal/relationships/suppliers.go](</Users/macbookpro/Documents/Kredit.com/internal/relationships/suppliers.go>) | Buyer consent and sharing boundaries; Suppliers | No separate finding |
| [internal/relationships/suppliers_test.go](</Users/macbookpro/Documents/Kredit.com/internal/relationships/suppliers_test.go>) | Buyer consent and sharing boundaries; TestBuyerSupplierDirectoryIncludesUnactivatedLimitsAndHistoricalConsent | No separate finding |

## internal/reports

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/reports/analytics.go](</Users/macbookpro/Documents/Kredit.com/internal/reports/analytics.go>) | Amounts, due dates, cohorts, database visibility and honest empty states; PilotScorecard | [F025](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:109) |
| [internal/reports/payment_metrics.go](</Users/macbookpro/Documents/Kredit.com/internal/reports/payment_metrics.go>) | Amounts, due dates, cohorts, database visibility and honest empty states | [F025](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:109) |
| [internal/reports/payment_metrics_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/reports/payment_metrics_postgres_test.go>) | Amounts, due dates, cohorts, database visibility and honest empty states; TestPaymentMetricsRespectCompletionAndChronology | [F040](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:169) |
| [internal/reports/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/reports/postgres_test.go>) | Amounts, due dates, cohorts, database visibility and honest empty states; TestPostgresAnalyticsIsPrivacySafeAndRestartSafe, TestPostgresProductEventDeduplicationAndScorecardReconciliation, TestPostgresAuthoritativePaymentMandateEmitsCanonicalEvents | [F040](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:169) |
| [internal/reports/snapshot.go](</Users/macbookpro/Documents/Kredit.com/internal/reports/snapshot.go>) | Amounts, due dates, cohorts, database visibility and honest empty states; financialSnapshot, financialSnapshotTx | No separate finding |
| [internal/reports/snapshot_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/reports/snapshot_postgres_test.go>) | Amounts, due dates, cohorts, database visibility and honest empty states; TestReportRejectsMissingObligationProjection | No separate finding |
| [internal/reports/store.go](</Users/macbookpro/Documents/Kredit.com/internal/reports/store.go>) | Amounts, due dates, cohorts, database visibility and honest empty states; NewPostgresStore, NewStore, ReceivablesForSupplier, AgeingForSupplier and 18 other declarations | No separate finding |
| [internal/reports/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/reports/store_test.go>) | Amounts, due dates, cohorts, database visibility and honest empty states; TestReceivablesAndExportAreDerivedFromPayments, TestHistoryDoesNotEmitNonFiniteAverage, TestAnalyticsRejectsSensitiveMetadataAndUnversionedNames, TestForgivenBalanceDoesNotBecomeOnTimeRepaymentHistory and 10 other declarations | No separate finding |

## internal/schedules

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/schedules/adjustment.go](</Users/macbookpro/Documents/Kredit.com/internal/schedules/adjustment.go>) | Payment dates, allocation, unpaid balances and collection timing; ReducePrincipal | No separate finding |
| [internal/schedules/adjustment_test.go](</Users/macbookpro/Documents/Kredit.com/internal/schedules/adjustment_test.go>) | Payment dates, allocation, unpaid balances and collection timing; TestAcceptedCollectionInstantPreservedAcrossInstalments, TestPrincipalAdjustmentIsAtomicAndReducesLastUnpaidItems | No separate finding |
| [internal/schedules/fuzz_test.go](</Users/macbookpro/Documents/Kredit.com/internal/schedules/fuzz_test.go>) | Payment dates, allocation, unpaid balances and collection timing; fuzzPick, FuzzScheduleGeneration, FuzzPaymentAllocation | No separate finding |
| [internal/schedules/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/schedules/postgres_test.go>) | Payment dates, allocation, unpaid balances and collection timing; TestPostgresStoreRoundTrip | No separate finding |
| [internal/schedules/store.go](</Users/macbookpro/Documents/Kredit.com/internal/schedules/store.go>) | Payment dates, allocation, unpaid balances and collection timing; NewStore, NewPostgresStore, Create, CreateDefault and 23 other declarations | No separate finding |
| [internal/schedules/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/schedules/store_test.go>) | Payment dates, allocation, unpaid balances and collection timing; TestEqualMonthlyScheduleAllocatesEarlyPaymentInDueOrder, TestCustomScheduleValidatesSumAndGraceState, TestCustomScheduleRejectsOverflowingTotal, TestRejectedScheduleMutationsLeaveAllAmountsUnchanged and 3 other declarations | No separate finding |

## internal/support

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/support/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/support/postgres_test.go>) | Case authority, notes, transitions and history; TestPostgresStoreRoundTrip | No separate finding |
| [internal/support/store.go](</Users/macbookpro/Documents/Kredit.com/internal/support/store.go>) | Case authority, notes, transitions and history; NewStore, NewPostgresStore, Open, OpenWithNote and 10 other declarations | No separate finding |
| [internal/support/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/support/store_test.go>) | Case authority, notes, transitions and history; TestCaseTimelineIsAppendOnly | No separate finding |

## internal/tradelines

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/tradelines/postgres.go](</Users/macbookpro/Documents/Kredit.com/internal/tradelines/postgres.go>) | Customer limits, reservations, purchases, receipt and exposure; NewPostgresStore, NewPostgresStoreWithOutbox, SetTransactionalActivationHandler, local and 29 other declarations | [F024](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:105) |
| [internal/tradelines/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/tradelines/postgres_test.go>) | Customer limits, reservations, purchases, receipt and exposure; TestPostgresStoreRoundTrip | [F040](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:169) |
| [internal/tradelines/store.go](</Users/macbookpro/Documents/Kredit.com/internal/tradelines/store.go>) | Customer limits, reservations, purchases, receipt and exposure; NewStore, SetLineGuard, SetMaxDrawdownsPerLineDay, SetMaxActiveExposure and 30 other declarations | No separate finding |
| [internal/tradelines/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/tradelines/store_test.go>) | Customer limits, reservations, purchases, receipt and exposure; TestConcurrentReservationsCannotExceedLimit, TestDrawdownConfirmationActivationAndSuspension, TestDrawdownRejectsChangedTermsAndMissingActivationHandler, TestReceiptIssueDoesNotActivateAndCancellationReleasesLimit and 5 other declarations | No separate finding |

## internal/usercontrol

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/usercontrol/export.go](</Users/macbookpro/Documents/Kredit.com/internal/usercontrol/export.go>) | Account recovery, privacy requests, exports and retention holds; privacyExportPayload | No separate finding |
| [internal/usercontrol/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/usercontrol/postgres_test.go>) | Account recovery, privacy requests, exports and retention holds; TestPostgresRecoveryAndPrivacyControls | No separate finding |
| [internal/usercontrol/store.go](</Users/macbookpro/Documents/Kredit.com/internal/usercontrol/store.go>) | Account recovery, privacy requests, exports and retention holds; NewStore, NewPostgresStore, BindUser, GenerateRecoveryCodes and 35 other declarations | [F012](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:57) |
| [internal/usercontrol/store_test.go](</Users/macbookpro/Documents/Kredit.com/internal/usercontrol/store_test.go>) | Account recovery, privacy requests, exports and retention holds; TestRecoveryRequiresIndependentEvidenceReviewAndCoolingOff, TestPrivacyNeedsIndependentCompletionAndAppliesRestriction, TestUnknownRecoveryIsEnumerationSafe, TestRecoveryRateLimitCountsUnknownIdentifiers and 2 other declarations | No separate finding |

## internal/web

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/web/admin_insight_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/admin_insight_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; adminChangeContext, previewBusinessPolicy, policyJSON, policyMap and 2 other declarations | [F013](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:61) |
| [internal/web/admin_surfaces_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/admin_surfaces_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; adminSurfaceServer, statusForPath, TestUnenabledAdminSurfacesAreUnreachable, TestAdminSurfacesAllowListSupportsExplicitAll and 2 other declarations | No separate finding |
| [internal/web/admin_workflow_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/admin_workflow_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; adminCapabilities, adminRoles, reviewKinds, adminInbox and 9 other declarations | [F013](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:61) |
| [internal/web/audit_financial_regression_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/audit_financial_regression_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; verifyRevolvingRepayments | No separate finding |
| [internal/web/auth_contract_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/auth_contract_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestSignInAndAuthenticatorEnrollmentContracts | No separate finding |
| [internal/web/auth_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/auth_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; requestOTP, verifyOTP, logout, me and 10 other declarations | No separate finding |
| [internal/web/business_policy_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/business_policy_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; policyFailure, monoAdminStatus, businessPolicies, proposeBusinessPolicy and 2 other declarations | No separate finding |
| [internal/web/business_policy_handlers_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/business_policy_handlers_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestPublicPricingOnlyExposesRates | No separate finding |
| [internal/web/buyer_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/buyer_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; createBuyerInvitation, previewBuyerInvitation, requestBuyerInvitationOTP, acceptBuyerInvitation and 1 other declarations | [F020](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:89) |
| [internal/web/collection_jobs.go](</Users/macbookpro/Documents/Kredit.com/internal/web/collection_jobs.go>) | HTTP authorization, payloads, status handling and actual service wiring; EnqueueCollectionWork, enqueueCollectionPages, enqueueTenantCollectionPages, HandleCollectionJob | [F017](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:77) |
| [internal/web/collection_notice_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/collection_notice_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; acknowledgeCollectionNotice, collectionNotices | No separate finding |
| [internal/web/completion_handlers_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/completion_handlers_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestNotificationReceiptsAuthenticateExactBody, TestScrapeCredentialOnlyGrantsMetricsAccess, TestBuyerCanFindStandaloneBankPermission | No separate finding |
| [internal/web/credit_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/credit_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; createCreditRequest, listCreditRequests, getCreditRequest, updateDraftCreditRequest and 63 other declarations | [F022](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:97), [F023](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:101), [F024](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:105), [F030](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:129) |
| [internal/web/credit_terms_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/credit_terms_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; previewCreditTerms | No separate finding |
| [internal/web/customer_access_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/customer_access_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; ListCustomers, TestSalesCanSelectCustomersWithoutFinancialMutationAccess | No separate finding |
| [internal/web/customer_registration_recovery.go](</Users/macbookpro/Documents/Kredit.com/internal/web/customer_registration_recovery.go>) | HTTP authorization, payloads, status handling and actual service wiring; registrationFingerprint, listCustomerRegistrations, resolveCustomerRegistration | No separate finding |
| [internal/web/dashboard_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/dashboard_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; listOrganizationPayments, listOrganizationCollections, listOrganizationOverdue, listOrganizationCustomers and 3 other declarations | [F025](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:109) |
| [internal/web/document_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/document_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; uploadDocument, documentDownload | [F030](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:129) |
| [internal/web/document_upload_slot_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/document_upload_slot_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; completeDocumentUpload, createDocumentUploadSlot | No separate finding |
| [internal/web/feedback_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/feedback_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; submitProductFeedback | No separate finding |
| [internal/web/financial_reads.go](</Users/macbookpro/Documents/Kredit.com/internal/web/financial_reads.go>) | HTTP authorization, payloads, status handling and actual service wiring; readCreditForSupplier, readCreditForBuyer, readPayments, getPayment and 12 other declarations | No separate finding |
| [internal/web/financial_review_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/financial_review_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; financialReviews, decideFinancialReview | [F017](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:77) |
| [internal/web/fuzz_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/fuzz_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; FuzzProblemDetailMapping | No separate finding |
| [internal/web/http_helpers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/http_helpers.go>) | HTTP authorization, payloads, status handling and actual service wiring; decodeJSON, decodeJSONLimit, decodeJSONRequest, writeProblem and 2 other declarations | No separate finding |
| [internal/web/idempotency_middleware_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/idempotency_middleware_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestIdempotencyMiddlewareReplaysAndRejectsConflicts, TestFinancialMutationsRequireIdempotencyKey, TestFinancialMutationRouteMatrixRequiresIdempotencyKey, TestOneTimeCredentialsAreNotStoredForReplay and 3 other declarations | No separate finding |
| [internal/web/mandate_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/mandate_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; cancelBuyerMandate, restoreBuyerMandate, findBuyerMandate, applyMandateToBuyerResources | No separate finding |
| [internal/web/milestone1_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/milestone1_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestSupplierOnboardingAndTenantBoundaries, newTestClient, doJSON, decodeResponse | No separate finding |
| [internal/web/mono_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/mono_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; monoWebhook, HandleProviderNotice, createRepaymentCustomer | [F010](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:49), [F022](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:97), [F026](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:113) |
| [internal/web/mono_runtime_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/mono_runtime_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestDisabledMonoCannotCreateMockActiveMandate | No separate finding |
| [internal/web/notification_receipt_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/notification_receipt_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; notificationDeliveryReceipt | [F001](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:7) |
| [internal/web/notification_recipients.go](</Users/macbookpro/Documents/Kredit.com/internal/web/notification_recipients.go>) | HTTP authorization, payloads, status handling and actual service wiring; EmitNotification | No separate finding |
| [internal/web/onboarding_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/onboarding_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; getSupplierOnboarding, requestOnboardingContactOTP, verifyOnboardingContact, updateSupplierRepresentative and 13 other declarations | [F018](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:81), [F019](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:85) |
| [internal/web/optional_activity_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/optional_activity_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestOptionalActivityHonorsActorRestrictionAndReadFailure | No separate finding |
| [internal/web/organization_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/organization_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; listOrganizations, createOrganization, getOrganization, listMembers and 5 other declarations | No separate finding |
| [internal/web/outbox_notifications.go](</Users/macbookpro/Documents/Kredit.com/internal/web/outbox_notifications.go>) | HTTP authorization, payloads, status handling and actual service wiring; QueueOutboxNotification | No separate finding |
| [internal/web/outbox_notifications_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/outbox_notifications_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestOutboxNotificationQueuesOnceToActualBuyer, TestCollectionDiscoveryVisitsEveryPage | No separate finding |
| [internal/web/ownership_transfer.go](</Users/macbookpro/Documents/Kredit.com/internal/web/ownership_transfer.go>) | HTTP authorization, payloads, status handling and actual service wiring; transferOwnershipTx | [F014](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:65) |
| [internal/web/ownership_transfer_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/ownership_transfer_postgres_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestOwnershipTransferRechecksAuthorityAndPreservesEffectiveSuccessor | [F040](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:169) |
| [internal/web/payment_claim_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/payment_claim_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; createBuyerPaymentClaim, listBuyerPaymentClaims, listPaymentClaims, listOrganizationPaymentClaims and 5 other declarations | [F029](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:125) |
| [internal/web/platform_operations_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/platform_operations_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; requirePlatformAccess, operationsOverview, operationsAnalyticsScorecard, operationsJobs and 22 other declarations | [F015](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:69), [F016](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:73) |
| [internal/web/platform_operations_handlers_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/platform_operations_handlers_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestPlatformOperationsRequiresRoleAndStepUp | No separate finding |
| [internal/web/platform_settings_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/platform_settings_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; isFeatureEnabled, platformCapabilities, listPlatformSettings, updatePlatformSetting and 5 other declarations | No separate finding |
| [internal/web/platform_settings_handlers_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/platform_settings_handlers_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestPlatformSettingsEndpoints, GetGovernance, TestCapabilitiesReflectDisputeSwitch, GetBool | No separate finding |
| [internal/web/provider_readiness_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/provider_readiness_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestReadinessRejectsFailedConfiguredAdapterWithoutLeakingDetails | No separate finding |
| [internal/web/relationship_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/relationship_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; recordBuyerConsent, listBuyerConsents, buyerPermissionSuppliers | No separate finding |
| [internal/web/reports_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/reports_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; reportReceivables, providerStatus, readinessStatus, reportAgeing and 9 other declarations | No separate finding |
| [internal/web/risk_hold_scope_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/risk_hold_scope_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestSaleRiskHoldsRespectPartyScopeExpiryAndOutage | No separate finding |
| [internal/web/runtime.go](</Users/macbookpro/Documents/Kredit.com/internal/web/runtime.go>) | HTTP authorization, payloads, status handling and actual service wiring; DurableDomainReady, NewRuntime, NewRuntimeWithDB, csvSet and 1 other declarations | [F001](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:7), [F007](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:37), [F012](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:57), [F021](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:93) |
| [internal/web/runtime_schedule_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/runtime_schedule_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestAcceptedInstalmentTermsCreateTheActivatedSchedule, TestTradeLineReceiptCreatesItsOwnObligationScheduleAndBalancedLedger | [F040](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:169) |
| [internal/web/runtime_tradeline_postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/runtime_tradeline_postgres_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestPostgresTradeLineActivationCommitsAsOneFinancialTransaction | No separate finding |
| [internal/web/secure_link_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/secure_link_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; resolveSecureLink, safeSecureRedirect | No separate finding |
| [internal/web/secure_link_handlers_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/secure_link_handlers_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestResolveSecureLink, TestResolveSecureLinkRejectsExternalRedirect | No separate finding |
| [internal/web/server.go](</Users/macbookpro/Documents/Kredit.com/internal/web/server.go>) | HTTP authorization, payloads, status handling and actual service wiring; sensitiveRateLimitRoute, NewServer, NewServerWithRuntime, Handler and 36 other declarations | [F011](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:53) |
| [internal/web/server_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/server_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestDecodeJSONRequestWritesClearClientError, TestHealthEndpoint, TestInvitationTokenIsRedactedFromAccessPath, TestInvalidRequestIDIsReplacedAndSecurityHeadersArePresent and 1 other declarations | No separate finding |
| [internal/web/supplier_onboarding_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/supplier_onboarding_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestNewSupplierOwnerReachesPilotReadyAndCanInviteSales, TestIncompleteSupplierGateExplainsRecovery, TestOnboardingRoleViewsAndMFAFreshness, contains | No separate finding |
| [internal/web/support_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/support_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; openSupportCase, listSupportCases, transitionSupportCase | No separate finding |
| [internal/web/user_control_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/user_control_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; getNotificationPreferences, updateNotificationPreferences, regenerateRecoveryCodes, requestAccountRecovery and 12 other declarations | [F012](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:57) |
| [internal/web/user_control_handlers_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/user_control_handlers_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; TestNotificationPrivacyAndRecoverySelfService | No separate finding |
| [internal/web/website_content_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/website_content_handlers.go>) | HTTP authorization, payloads, status handling and actual service wiring; websiteContent, changeWebsiteContent, publishedGuides | No separate finding |
| [internal/web/website_content_handlers_test.go](</Users/macbookpro/Documents/Kredit.com/internal/web/website_content_handlers_test.go>) | HTTP authorization, payloads, status handling and actual service wiring; verifyWebsitePublication | No separate finding |

## internal/whatsapp

| File | Reviewed content | Result |
| --- | --- | --- |
| [internal/whatsapp/fuzz_test.go](</Users/macbookpro/Documents/Kredit.com/internal/whatsapp/fuzz_test.go>) | Inbound message identity, authentication and safe command handling; FuzzParseAmountNeverPanics, FuzzParseCommandNeverPanics | No separate finding |
| [internal/whatsapp/handler.go](</Users/macbookpro/Documents/Kredit.com/internal/whatsapp/handler.go>) | Inbound message identity, authentication and safe command handling; NewPostgresHandler, NewHandler, Sign, Verify and 4 other declarations | [F005](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:29) |
| [internal/whatsapp/handler_test.go](</Users/macbookpro/Documents/Kredit.com/internal/whatsapp/handler_test.go>) | Inbound message identity, authentication and safe command handling; TestSignedWebhookParsesStructuredCreateCommandAndDeduplicates, TestPaymentCommandNeverAcceptsCredentials, TestParseAmountUsesExactIntegerKoboArithmetic | No separate finding |
| [internal/whatsapp/postgres_test.go](</Users/macbookpro/Documents/Kredit.com/internal/whatsapp/postgres_test.go>) | Inbound message identity, authentication and safe command handling; TestPostgresWebhookDeduplicationSurvivesRestart | No separate finding |

## scripts

| File | Reviewed content | Result |
| --- | --- | --- |
| [scripts/api-lint.sh](</Users/macbookpro/Documents/Kredit.com/scripts/api-lint.sh>) | api-lint.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/backup.sh](</Users/macbookpro/Documents/Kredit.com/scripts/backup.sh>) | backup.sh: deployment, boundaries, failure handling and referenced paths | [F036](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:153) |
| [scripts/bootstrap.sh](</Users/macbookpro/Documents/Kredit.com/scripts/bootstrap.sh>) | bootstrap.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/check-tools.sh](</Users/macbookpro/Documents/Kredit.com/scripts/check-tools.sh>) | check-tools.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/ci.sh](</Users/macbookpro/Documents/Kredit.com/scripts/ci.sh>) | ci.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/configure-development-database.sh](</Users/macbookpro/Documents/Kredit.com/scripts/configure-development-database.sh>) | configure-development-database.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/content-audit.mjs](</Users/macbookpro/Documents/Kredit.com/scripts/content-audit.mjs>) | content-audit.mjs: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/data-inventory-check.sh](</Users/macbookpro/Documents/Kredit.com/scripts/data-inventory-check.sh>) | data-inventory-check.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/data-inventory-generate.sh](</Users/macbookpro/Documents/Kredit.com/scripts/data-inventory-generate.sh>) | data-inventory-generate.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/database-safety-test.sh](</Users/macbookpro/Documents/Kredit.com/scripts/database-safety-test.sh>) | database-safety-test.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/db-check.sh](</Users/macbookpro/Documents/Kredit.com/scripts/db-check.sh>) | db-check.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/db-reset.sh](</Users/macbookpro/Documents/Kredit.com/scripts/db-reset.sh>) | db-reset.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/db-rollback.sh](</Users/macbookpro/Documents/Kredit.com/scripts/db-rollback.sh>) | db-rollback.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/dev.sh](</Users/macbookpro/Documents/Kredit.com/scripts/dev.sh>) | dev.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/frontend-api-coverage.mjs](</Users/macbookpro/Documents/Kredit.com/scripts/frontend-api-coverage.mjs>) | frontend-api-coverage.mjs: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/fuzz.sh](</Users/macbookpro/Documents/Kredit.com/scripts/fuzz.sh>) | fuzz.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/generate.sh](</Users/macbookpro/Documents/Kredit.com/scripts/generate.sh>) | generate.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/implementation-plan-conformance-test.sh](</Users/macbookpro/Documents/Kredit.com/scripts/implementation-plan-conformance-test.sh>) | implementation-plan-conformance-test.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/implementation-plan-conformance.sh](</Users/macbookpro/Documents/Kredit.com/scripts/implementation-plan-conformance.sh>) | implementation-plan-conformance.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/launch-browser-check.mjs](</Users/macbookpro/Documents/Kredit.com/scripts/launch-browser-check.mjs>) | launch-browser-check.mjs: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/launch-outage-check.mjs](</Users/macbookpro/Documents/Kredit.com/scripts/launch-outage-check.mjs>) | launch-outage-check.mjs: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/lint.sh](</Users/macbookpro/Documents/Kredit.com/scripts/lint.sh>) | lint.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/load-env-test.sh](</Users/macbookpro/Documents/Kredit.com/scripts/load-env-test.sh>) | load-env-test.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/load-env.sh](</Users/macbookpro/Documents/Kredit.com/scripts/load-env.sh>) | load-env.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/load-smoke.sh](</Users/macbookpro/Documents/Kredit.com/scripts/load-smoke.sh>) | load-smoke.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/phase5-evidence-gate.sh](</Users/macbookpro/Documents/Kredit.com/scripts/phase5-evidence-gate.sh>) | phase5-evidence-gate.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/phase5_load.py](</Users/macbookpro/Documents/Kredit.com/scripts/phase5_load.py>) | phase5_load.py: deployment, boundaries, failure handling and referenced paths; redirect_request, validate_origin, percentile, read and 4 other declarations | No separate finding |
| [scripts/phase6-context-audit.py](</Users/macbookpro/Documents/Kredit.com/scripts/phase6-context-audit.py>) | phase6-context-audit.py: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/phase6-governance-test.sh](</Users/macbookpro/Documents/Kredit.com/scripts/phase6-governance-test.sh>) | phase6-governance-test.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/post-deploy-check.sh](</Users/macbookpro/Documents/Kredit.com/scripts/post-deploy-check.sh>) | post-deploy-check.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/product-contract-sync.mjs](</Users/macbookpro/Documents/Kredit.com/scripts/product-contract-sync.mjs>) | product-contract-sync.mjs: deployment, boundaries, failure handling and referenced paths; hasDynamicRouteChoice, visit | No separate finding |
| [scripts/readme-conformance.sh](</Users/macbookpro/Documents/Kredit.com/scripts/readme-conformance.sh>) | readme-conformance.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/recovery_fingerprint.py](</Users/macbookpro/Documents/Kredit.com/scripts/recovery_fingerprint.py>) | recovery_fingerprint.py: deployment, boundaries, failure handling and referenced paths; connection_environment, psql, quote, row_bytes and 2 other declarations | No separate finding |
| [scripts/release-certify.sh](</Users/macbookpro/Documents/Kredit.com/scripts/release-certify.sh>) | release-certify.sh: deployment, boundaries, failure handling and referenced paths | [F037](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:157) |
| [scripts/repository-audit.mjs](</Users/macbookpro/Documents/Kredit.com/scripts/repository-audit.mjs>) | repository-audit.mjs: deployment, boundaries, failure handling and referenced paths; fail, checkMarkdownLinks, checkStaticRouteLinks | No separate finding |
| [scripts/restore-drill.sh](</Users/macbookpro/Documents/Kredit.com/scripts/restore-drill.sh>) | restore-drill.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/security.sh](</Users/macbookpro/Documents/Kredit.com/scripts/security.sh>) | security.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/setup-vps.sh](</Users/macbookpro/Documents/Kredit.com/scripts/setup-vps.sh>) | setup-vps.sh: deployment, boundaries, failure handling and referenced paths | [F006](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:33) |
| [scripts/sqlc-check.sh](</Users/macbookpro/Documents/Kredit.com/scripts/sqlc-check.sh>) | sqlc-check.sh: deployment, boundaries, failure handling and referenced paths | [F037](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:157) |
| [scripts/sqlc-generate.sh](</Users/macbookpro/Documents/Kredit.com/scripts/sqlc-generate.sh>) | sqlc-generate.sh: deployment, boundaries, failure handling and referenced paths | [F037](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:157) |
| [scripts/test-e2e.sh](</Users/macbookpro/Documents/Kredit.com/scripts/test-e2e.sh>) | test-e2e.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/test-integration.sh](</Users/macbookpro/Documents/Kredit.com/scripts/test-integration.sh>) | test-integration.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |
| [scripts/test_phase5_backup.py](</Users/macbookpro/Documents/Kredit.com/scripts/test_phase5_backup.py>) | test_phase5_backup: fixtures, assertions and production equivalence; setUp, test_requested_bytes_pass, test_missing_checksum_fails, test_tampered_bytes_fail and 2 other declarations | No separate finding |
| [scripts/test_phase5_fingerprint.py](</Users/macbookpro/Documents/Kredit.com/scripts/test_phase5_fingerprint.py>) | test_phase5_fingerprint: fixtures, assertions and production equivalence; test_quote_identifier, test_numeric_precision_is_not_rounded, test_unicode_and_embedded_escapes_preserved, test_uri_becomes_explicit_libpq_parameters and 1 other declarations | No separate finding |
| [scripts/test_phase5_load.py](</Users/macbookpro/Documents/Kredit.com/scripts/test_phase5_load.py>) | test_phase5_load: fixtures, assertions and production equivalence; test_explicit_loopback_only, test_percentiles_are_nearest_rank, test_error_or_empty_buyer_payload_is_not_success, test_no_environment_acknowledgement_no_network | No separate finding |
| [scripts/test_phase5_release_evidence.py](</Users/macbookpro/Documents/Kredit.com/scripts/test_phase5_release_evidence.py>) | test_phase5_release_evidence: fixtures, assertions and production equivalence; setUp, check, test_structurally_complete_synthetic_manifest, test_pending_template_is_not_approval and 11 other declarations | No separate finding |
| [scripts/test_provider_evidence.py](</Users/macbookpro/Documents/Kredit.com/scripts/test_provider_evidence.py>) | test_provider_evidence: fixtures, assertions and production equivalence; setUp, test_complete_manifest, test_candidate_commit_must_match, test_incomplete_and_synthetic_evidence_rejected and 2 other declarations | No separate finding |
| [scripts/verify_backup.py](</Users/macbookpro/Documents/Kredit.com/scripts/verify_backup.py>) | verify_backup.py: deployment, boundaries, failure handling and referenced paths; verify | No separate finding |
| [scripts/verify_provider_evidence.py](</Users/macbookpro/Documents/Kredit.com/scripts/verify_provider_evidence.py>) | verify_provider_evidence.py: deployment, boundaries, failure handling and referenced paths; validate, main | No separate finding |
| [scripts/verify_release_evidence.py](</Users/macbookpro/Documents/Kredit.com/scripts/verify_release_evidence.py>) | verify_release_evidence.py: deployment, boundaries, failure handling and referenced paths; template, timestamp, validate, main | No separate finding |
| [scripts/web-test.sh](</Users/macbookpro/Documents/Kredit.com/scripts/web-test.sh>) | web-test.sh: deployment, boundaries, failure handling and referenced paths | No separate finding |

## tests/contract

| File | Reviewed content | Result |
| --- | --- | --- |
| [tests/contract/.gitkeep](</Users/macbookpro/Documents/Kredit.com/tests/contract/.gitkeep>) | .gitkeep: fixtures, assertions and production equivalence | No separate finding |
| [tests/contract/provider_contract_test.go](</Users/macbookpro/Documents/Kredit.com/tests/contract/provider_contract_test.go>) | provider_contract_test: fixtures, assertions and production equivalence; registeredAdapters, TestEveryAdapterDeclaresTheCapabilitiesTheProductRequires, TestNoAdapterAcceptsAForgedWebhook, TestMandateLifecycleContract | No separate finding |

## tests/e2e

| File | Reviewed content | Result |
| --- | --- | --- |
| [tests/e2e/.gitkeep](</Users/macbookpro/Documents/Kredit.com/tests/e2e/.gitkeep>) | .gitkeep: fixtures, assertions and production equivalence | No separate finding |
| [tests/e2e/README.md](</Users/macbookpro/Documents/Kredit.com/tests/e2e/README.md>) | README: fixtures, assertions and production equivalence | No separate finding |

## tests/integration

| File | Reviewed content | Result |
| --- | --- | --- |
| [tests/integration/persistence_test.go](</Users/macbookpro/Documents/Kredit.com/tests/integration/persistence_test.go>) | persistence_test: fixtures, assertions and production equivalence; TestMigratedDatabasePersistenceContractAndLedger | No separate finding |
| [tests/integration/tenant_isolation_test.go](</Users/macbookpro/Documents/Kredit.com/tests/integration/tenant_isolation_test.go>) | tenant_isolation_test: fixtures, assertions and production equivalence; TestFinancialRowsAreTenantBoundAtDatabaseLayer, TestWorkerIsTenantBoundAndCannotForgeBuyerEvidence, TestCollectionWorkerDiscoveryIsNarrowlyPrivileged, integrationPool and 4 other declarations | No separate finding |

## tests/load

| File | Reviewed content | Result |
| --- | --- | --- |
| [tests/load/smoke.js](</Users/macbookpro/Documents/Kredit.com/tests/load/smoke.js>) | smoke: fixtures, assertions and production equivalence | No separate finding |

## tests/performance

| File | Reviewed content | Result |
| --- | --- | --- |
| [tests/performance/.gitkeep](</Users/macbookpro/Documents/Kredit.com/tests/performance/.gitkeep>) | .gitkeep: fixtures, assertions and production equivalence | No separate finding |
| [tests/performance/acceptance.js](</Users/macbookpro/Documents/Kredit.com/tests/performance/acceptance.js>) | acceptance: fixtures, assertions and production equivalence | No separate finding |
| [tests/performance/portfolio_queries.sql](</Users/macbookpro/Documents/Kredit.com/tests/performance/portfolio_queries.sql>) | portfolio_queries: fixtures, assertions and production equivalence | No separate finding |

## tests/provider-simulators

| File | Reviewed content | Result |
| --- | --- | --- |
| [tests/provider-simulators/README.md](</Users/macbookpro/Documents/Kredit.com/tests/provider-simulators/README.md>) | README: fixtures, assertions and production equivalence | No separate finding |

## web

| File | Reviewed content | Result |
| --- | --- | --- |
| [web/package.json](</Users/macbookpro/Documents/Kredit.com/web/package.json>) | Dependency/toolchain pins, integrity, imports and available advisory evidence | No separate finding |
| [web/playwright.config.ts](</Users/macbookpro/Documents/Kredit.com/web/playwright.config.ts>) | playwright.config.ts: configuration/data and current consumers | No separate finding |
| [web/svelte.config.js](</Users/macbookpro/Documents/Kredit.com/web/svelte.config.js>) | svelte.config.js: configuration/data and current consumers | No separate finding |
| [web/tsconfig.json](</Users/macbookpro/Documents/Kredit.com/web/tsconfig.json>) | tsconfig.json: configuration/data and current consumers | No separate finding |
| [web/vercel.json](</Users/macbookpro/Documents/Kredit.com/web/vercel.json>) | vercel.json: configuration/data and current consumers | No separate finding |
| [web/vite.config.ts](</Users/macbookpro/Documents/Kredit.com/web/vite.config.ts>) | vite.config.ts: configuration/data and current consumers | No separate finding |

## web/src

| File | Reviewed content | Result |
| --- | --- | --- |
| [web/src/app.css](</Users/macbookpro/Documents/Kredit.com/web/src/app.css>) | app: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/app.d.ts](</Users/macbookpro/Documents/Kredit.com/web/src/app.d.ts>) | app.d: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/app.html](</Users/macbookpro/Documents/Kredit.com/web/src/app.html>) | app: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/hooks.server.ts](</Users/macbookpro/Documents/Kredit.com/web/src/hooks.server.ts>) | hooks.server: shared UI/data contract, safe rendering and consumers | [F011](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:53) |
| [web/src/lib/account-context.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/account-context.ts>) | account-context: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/admin-client.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/admin-client.ts>) | admin-client: shared UI/data contract, safe rendering and consumers; fail, request, adminGet, adminPost and 3 other declarations | No separate finding |
| [web/src/lib/api/client.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/api/client.ts>) | client: shared UI/data contract, safe rendering and consumers; csrfToken, csrfHeaders, idempotencyKey, signOut | No separate finding |
| [web/src/lib/api/mutation.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/api/mutation.ts>) | mutation: shared UI/data contract, safe rendering and consumers; canonical | No separate finding |
| [web/src/lib/api/onboarding-settings.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/api/onboarding-settings.ts>) | onboarding-settings: shared UI/data contract, safe rendering and consumers; settingsProfile, loadOnboardingSettings | [F027](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:117) |
| [web/src/lib/api/reliable.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/api/reliable.ts>) | reliable: shared UI/data contract, safe rendering and consumers; record, text, publicError, boundedFetch and 5 other declarations | No separate finding |
| [web/src/lib/attention.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/attention.ts>) | attention: shared UI/data contract, safe rendering and consumers; attentionItems | No separate finding |
| [web/src/lib/blog/articles.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/blog/articles.ts>) | articles: shared UI/data contract, safe rendering and consumers; validateArticleSlugs, articleForSlug, categorySlug, categoryForSlug | No separate finding |
| [web/src/lib/blog/first-wave-drafts.js](</Users/macbookpro/Documents/Kredit.com/web/src/lib/blog/first-wave-drafts.js>) | first-wave-drafts: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/blog/guides.js](</Users/macbookpro/Documents/Kredit.com/web/src/lib/blog/guides.js>) | guides: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/components/AdminAttention.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/AdminAttention.svelte>) | AdminAttention: shared UI/data contract, safe rendering and consumers; load | No separate finding |
| [web/src/lib/components/AuthGate.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/AuthGate.svelte>) | AuthGate: shared UI/data contract, safe rendering and consumers; verify | No separate finding |
| [web/src/lib/components/BankReturnNotice.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/BankReturnNotice.svelte>) | BankReturnNotice: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/components/CommandPalette.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/CommandPalette.svelte>) | CommandPalette: shared UI/data contract, safe rendering and consumers; show, close, onKeydown, pick and 1 other declarations | No separate finding |
| [web/src/lib/components/ConnectivityBanner.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/ConnectivityBanner.svelte>) | ConnectivityBanner: shared UI/data contract, safe rendering and consumers; update | No separate finding |
| [web/src/lib/components/DisputeDetail.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/DisputeDetail.svelte>) | DisputeDetail: shared UI/data contract, safe rendering and consumers; load, submitEvidence, decide | [F030](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:129) |
| [web/src/lib/components/DocumentLayout.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/DocumentLayout.svelte>) | DocumentLayout: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/components/DocumentUploader.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/DocumentUploader.svelte>) | DocumentUploader: shared UI/data contract, safe rendering and consumers; selectFile | No separate finding |
| [web/src/lib/components/FeedbackPrompt.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/FeedbackPrompt.svelte>) | FeedbackPrompt: shared UI/data contract, safe rendering and consumers; answer | No separate finding |
| [web/src/lib/components/HomeProof.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/HomeProof.svelte>) | HomeProof: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/components/Money.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/Money.svelte>) | Money: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/components/MotionObserver.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/MotionObserver.svelte>) | MotionObserver: shared UI/data contract, safe rendering and consumers; prepare, schedulePrepare | No separate finding |
| [web/src/lib/components/NotificationHistory.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/NotificationHistory.svelte>) | NotificationHistory: shared UI/data contract, safe rendering and consumers; timestamp, sentTime, load | No separate finding |
| [web/src/lib/components/OwnerDialog.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/OwnerDialog.svelte>) | OwnerDialog: shared UI/data contract, safe rendering and consumers; close | No separate finding |
| [web/src/lib/components/PaymentReview.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/PaymentReview.svelte>) | PaymentReview: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/components/PortalNav.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/PortalNav.svelte>) | PortalNav: shared UI/data contract, safe rendering and consumers; leaveAccount, current, menuGroups, closeMenus and 2 other declarations | No separate finding |
| [web/src/lib/components/ProtectedActionDialog.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/ProtectedActionDialog.svelte>) | ProtectedActionDialog: shared UI/data contract, safe rendering and consumers; close, proceed | No separate finding |
| [web/src/lib/components/PublishedLegalDocument.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/PublishedLegalDocument.svelte>) | PublishedLegalDocument: shared UI/data contract, safe rendering and consumers; pieces | No separate finding |
| [web/src/lib/components/ResourceNotice.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/ResourceNotice.svelte>) | ResourceNotice: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/components/ShareActions.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/ShareActions.svelte>) | ShareActions: shared UI/data contract, safe rendering and consumers; share | No separate finding |
| [web/src/lib/components/SiteFooter.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/SiteFooter.svelte>) | SiteFooter: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/components/SiteHeader.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/SiteHeader.svelte>) | SiteHeader: shared UI/data contract, safe rendering and consumers; closeMobileMenu | No separate finding |
| [web/src/lib/components/Skeleton.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/Skeleton.svelte>) | Skeleton: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/components/StatusPill.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/StatusPill.svelte>) | StatusPill: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/components/SystemBanner.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/SystemBanner.svelte>) | SystemBanner: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/components/VerifyIdentity.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/VerifyIdentity.svelte>) | VerifyIdentity: shared UI/data contract, safe rendering and consumers; verify | No separate finding |
| [web/src/lib/components/WorkspacePage.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/WorkspacePage.svelte>) | WorkspacePage: shared UI/data contract, safe rendering and consumers; refresh, read | No separate finding |
| [web/src/lib/datetime.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/datetime.ts>) | datetime: shared UI/data contract, safe rendering and consumers; localDateTime, readableDate, readableDateTime | No separate finding |
| [web/src/lib/demo-sale.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/demo-sale.ts>) | demo-sale: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/fee-terms.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/fee-terms.ts>) | fee-terms: shared UI/data contract, safe rendering and consumers; validFeeTerms, feeDisclosure, feeForKobo, baseFeeForKobo | No separate finding |
| [web/src/lib/financial-copy.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/financial-copy.ts>) | financial-copy: shared UI/data contract, safe rendering and consumers; disputeEffectCopy, hostedAuthorizationURL, acceptanceMessage | No separate finding |
| [web/src/lib/legal-defaults.json](</Users/macbookpro/Documents/Kredit.com/web/src/lib/legal-defaults.json>) | legal-defaults: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/money.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/money.ts>) | money: shared UI/data contract, safe rendering and consumers; parseNaira, exactKobo, sumKobo, formatKobo and 3 other declarations | No separate finding |
| [web/src/lib/product-language.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/product-language.ts>) | product-language: shared UI/data contract, safe rendering and consumers; privacyRequestLabel, productLabel | No separate finding |
| [web/src/lib/product-tools.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/product-tools.ts>) | product-tools: shared UI/data contract, safe rendering and consumers; writeLocal, removeLocal, rememberSaleItem, whatsappURL and 3 other declarations | No separate finding |
| [web/src/lib/records.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/records.ts>) | records: shared UI/data contract, safe rendering and consumers; kobo, organization, customer, saleView and 5 other declarations | No separate finding |
| [web/src/lib/sale-drafts.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/sale-drafts.ts>) | sale-drafts: shared UI/data contract, safe rendering and consumers; key, readDraft, saveDraft, deleteDraft | No separate finding |
| [web/src/lib/seo.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/seo.ts>) | seo: shared UI/data contract, safe rendering and consumers; seoForPath, jsonLd | No separate finding |
| [web/src/lib/server/guides.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/server/guides.ts>) | guides: shared UI/data contract, safe rendering and consumers; loadGuides | No separate finding |
| [web/src/lib/server/legal-config.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/server/legal-config.ts>) | legal-config: shared UI/data contract, safe rendering and consumers; loadLegalConfig, assertLaunchWebConfig | No separate finding |
| [web/src/lib/server/legal-content.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/server/legal-content.ts>) | legal-content: shared UI/data contract, safe rendering and consumers; loadLegalContent | No separate finding |
| [web/src/lib/server/legal-publication.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/server/legal-publication.ts>) | legal-publication: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/server/website-content.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/server/website-content.ts>) | website-content: shared UI/data contract, safe rendering and consumers; loadWebsiteCopy | No separate finding |
| [web/src/lib/trade-line-records.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/trade-line-records.ts>) | trade-line-records: shared UI/data contract, safe rendering and consumers; tradeLine, drawdown, tradeStatement | No separate finding |
| [web/src/lib/website-content.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/website-content.ts>) | website-content: shared UI/data contract, safe rendering and consumers; decodeWebsiteCopy, decodeWebsitePublication, decodeWebsiteRecord | No separate finding |
| [web/src/lib/website-defaults.json](</Users/macbookpro/Documents/Kredit.com/web/src/lib/website-defaults.json>) | website-defaults: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/lib/website-editor-content.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/website-editor-content.ts>) | website-editor-content: shared UI/data contract, safe rendering and consumers | No separate finding |
| [web/src/routes/+error.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/+error.svelte>) | Route /: +error.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/+layout.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/+layout.svelte>) | Route /: +layout.svelte; display, actions, scope, amounts and errors | [F035](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:149) |
| [web/src/routes/+page.server.ts](</Users/macbookpro/Documents/Kredit.com/web/src/routes/+page.server.ts>) | Route /: +page.server.ts; HTTP/SSR data and publication boundary | No separate finding |
| [web/src/routes/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/+page.svelte>) | Route /: +page.svelte; display, actions, scope, amounts and errors; storyKeys | No separate finding |
| [web/src/routes/admin/+layout.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/+layout.svelte>) | Route /admin: +layout.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/admin/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/+page.svelte>) | Route /admin: +page.svelte; display, actions, scope, amounts and errors; load | No separate finding |
| [web/src/routes/admin/analytics/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/analytics/+page.svelte>) | Route /admin/analytics: +page.svelte; display, actions, scope, amounts and errors; decodeScorecard, load | [F025](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:109) |
| [web/src/routes/admin/approvals/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/approvals/+page.svelte>) | Route /admin/approvals: +page.svelte; display, actions, scope, amounts and errors; write, snapshot, change, difference and 5 other declarations | No separate finding |
| [web/src/routes/admin/attention/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/attention/+page.svelte>) | Route /admin/attention: +page.svelte; display, actions, scope, amounts and errors; load | No separate finding |
| [web/src/routes/admin/audit/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/audit/+page.svelte>) | Route /admin/audit: +page.svelte; display, actions, scope, amounts and errors; happened, where, load | No separate finding |
| [web/src/routes/admin/cases/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/cases/+page.svelte>) | Route /admin/cases: +page.svelte; display, actions, scope, amounts and errors; load | No separate finding |
| [web/src/routes/admin/cases/[id]/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/cases/[id]/+page.svelte>) | Route /admin/cases/[id]: +page.svelte; display, actions, scope, amounts and errors; load, update | No separate finding |
| [web/src/routes/admin/controls/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/controls/+page.svelte>) | Route /admin/controls: +page.svelte; display, actions, scope, amounts and errors; payload, inspect, apply | No separate finding |
| [web/src/routes/admin/customer-registrations/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/customer-registrations/+page.svelte>) | Route /admin/customer-registrations: +page.svelte; display, actions, scope, amounts and errors; load, choose, resolve | No separate finding |
| [web/src/routes/admin/diagnostics/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/diagnostics/+page.svelte>) | Route /admin/diagnostics: +page.svelte; display, actions, scope, amounts and errors; load | No separate finding |
| [web/src/routes/admin/disputes/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/disputes/+page.svelte>) | Route /admin/disputes: +page.svelte; display, actions, scope, amounts and errors; load | No separate finding |
| [web/src/routes/admin/disputes/[id]/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/disputes/[id]/+page.svelte>) | Route /admin/disputes/[id]: +page.svelte; display, actions, scope, amounts and errors; load, validPrincipal, decide | [F030](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:129) |
| [web/src/routes/admin/history/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/history/+page.svelte>) | Route /admin/history: +page.svelte; display, actions, scope, amounts and errors; historyItem, difference, load, value | No separate finding |
| [web/src/routes/admin/inbox/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/inbox/+page.svelte>) | Route /admin/inbox: +page.svelte; display, actions, scope, amounts and errors; load, assign | No separate finding |
| [web/src/routes/admin/jobs/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/jobs/+page.svelte>) | Route /admin/jobs: +page.svelte; display, actions, scope, amounts and errors; load, input, preview, confirm | No separate finding |
| [web/src/routes/admin/money/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/money/+page.svelte>) | Route /admin/money: +page.svelte; display, actions, scope, amounts and errors; load | No separate finding |
| [web/src/routes/admin/mono/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/mono/+page.svelte>) | Route /admin/mono: +page.svelte; display, actions, scope, amounts and errors; load | No separate finding |
| [web/src/routes/admin/organizations/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/organizations/+page.svelte>) | Route /admin/organizations: +page.svelte; display, actions, scope, amounts and errors; load | No separate finding |
| [web/src/routes/admin/platform-settings/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/platform-settings/+page.svelte>) | Route /admin/platform-settings: +page.svelte; display, actions, scope, amounts and errors; clearConnector, readable, load, openEdit and 4 other declarations | No separate finding |
| [web/src/routes/admin/privacy/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/privacy/+page.svelte>) | Route /admin/privacy: +page.svelte; display, actions, scope, amounts and errors; privacyRecord, load, begin, confirm | No separate finding |
| [web/src/routes/admin/provider-events/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/provider-events/+page.svelte>) | Route /admin/provider-events: +page.svelte; display, actions, scope, amounts and errors; load, input, preview, confirm | No separate finding |
| [web/src/routes/admin/reconciliation/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/reconciliation/+page.svelte>) | Route /admin/reconciliation: +page.svelte; display, actions, scope, amounts and errors; money, review, load, decide | No separate finding |
| [web/src/routes/admin/recovery/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/recovery/+page.svelte>) | Route /admin/recovery: +page.svelte; display, actions, scope, amounts and errors; load, begin, confirm | No separate finding |
| [web/src/routes/admin/search/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/search/+page.svelte>) | Route /admin/search: +page.svelte; display, actions, scope, amounts and errors; search | No separate finding |
| [web/src/routes/admin/settings/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/settings/+page.svelte>) | Route /admin/settings: +page.svelte; display, actions, scope, amounts and errors; write, integer, policyData, impactData and 13 other declarations | No separate finding |
| [web/src/routes/admin/team/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/team/+page.svelte>) | Route /admin/team: +page.svelte; display, actions, scope, amounts and errors; intent, member, userResult, load and 4 other declarations | No separate finding |
| [web/src/routes/admin/users/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/users/+page.svelte>) | Route /admin/users: +page.svelte; display, actions, scope, amounts and errors; load | No separate finding |
| [web/src/routes/admin/website/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/website/+page.svelte>) | Route /admin/website: +page.svelte; display, actions, scope, amounts and errors; cloneCopy, load, save, restore | No separate finding |
| [web/src/routes/app/+layout.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/+layout.svelte>) | Route /app: +layout.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/app/+page.server.ts](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/+page.server.ts>) | Route /app: +page.server.ts; HTTP/SSR data and publication boundary | No separate finding |
| [web/src/routes/app/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/+page.svelte>) | Route /app: +page.svelte; display, actions, scope, amounts and errors; destination, masked, checkSession, requestCode and 2 other declarations | No separate finding |
| [web/src/routes/app/activity/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/activity/+page.svelte>) | Route /app/activity: +page.svelte; display, actions, scope, amounts and errors; item, decode, load, initialize and 1 other declarations | No separate finding |
| [web/src/routes/app/collections/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/collections/+page.svelte>) | Route /app/collections: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/app/credit/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/credit/+page.svelte>) | Route /app/credit: +page.svelte; display, actions, scope, amounts and errors; loadSales, load | No separate finding |
| [web/src/routes/app/credit/[id]/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/credit/[id]/+page.svelte>) | Route /app/credit/[id]: +page.svelte; display, actions, scope, amounts and errors; intentFor, entity, checkedSale, moneyRow and 14 other declarations | [F031](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:133), [F032](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:137) |
| [web/src/routes/app/credit/new/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/credit/new/+page.svelte>) | Route /app/credit/new: +page.svelte; display, actions, scope, amounts and errors; chooseBuyer, showError, saveDraft, clearDraft and 9 other declarations | [F031](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:133) |
| [web/src/routes/app/credit/new/+page.ts](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/credit/new/+page.ts>) | Route /app/credit/new: +page.ts; HTTP/SSR data and publication boundary | No separate finding |
| [web/src/routes/app/credit/quick/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/credit/quick/+page.svelte>) | Route /app/credit/quick: +page.svelte; display, actions, scope, amounts and errors; restoreForBusiness, loadCustomers, focusStep, next and 3 other declarations | No separate finding |
| [web/src/routes/app/customers/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/customers/+page.svelte>) | Route /app/customers: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/app/customers/[id]/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/customers/[id]/+page.svelte>) | Route /app/customers/[id]: +page.svelte; display, actions, scope, amounts and errors; decodeHistory, decodeStatement, loadCustomer, customerName and 1 other declarations | No separate finding |
| [web/src/routes/app/customers/new/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/customers/new/+page.svelte>) | Route /app/customers/new: +page.svelte; display, actions, scope, amounts and errors; load, submit, copy | No separate finding |
| [web/src/routes/app/disputes/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/disputes/+page.svelte>) | Route /app/disputes: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/app/disputes/[id]/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/disputes/[id]/+page.svelte>) | Route /app/disputes/[id]: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/app/help/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/help/+page.svelte>) | Route /app/help: +page.svelte; display, actions, scope, amounts and errors; decodeCase, intent, loadCases, initialize and 2 other declarations | No separate finding |
| [web/src/routes/app/notifications/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/notifications/+page.svelte>) | Route /app/notifications: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/app/onboarding/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/onboarding/+page.svelte>) | Route /app/onboarding: +page.svelte; display, actions, scope, amounts and errors; summary, load, mutate, saveRepresentative and 6 other declarations | No separate finding |
| [web/src/routes/app/overdue/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/overdue/+page.svelte>) | Route /app/overdue: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/app/overview/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/overview/+page.svelte>) | Route /app/overview: +page.svelte; display, actions, scope, amounts and errors; loadRequests, load, createOrganization | No separate finding |
| [web/src/routes/app/payments/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/payments/+page.svelte>) | Route /app/payments: +page.svelte; display, actions, scope, amounts and errors; load, decide, confirmDecision, start | No separate finding |
| [web/src/routes/app/reports/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/reports/+page.svelte>) | Route /app/reports: +page.svelte; display, actions, scope, amounts and errors; load, initialize, lagosDayStart, bucketName and 1 other declarations | [F033](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:141) |
| [web/src/routes/app/search/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/search/+page.svelte>) | Route /app/search: +page.svelte; display, actions, scope, amounts and errors; loadBusiness, initialize | No separate finding |
| [web/src/routes/app/settings/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/settings/+page.svelte>) | Route /app/settings: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/app/settings/billing/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/settings/billing/+page.svelte>) | Route /app/settings/billing: +page.svelte; display, actions, scope, amounts and errors; load, save | [F019](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:85), [F027](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:117) |
| [web/src/routes/app/settings/credit-policy/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/settings/credit-policy/+page.svelte>) | Route /app/settings/credit-policy: +page.svelte; display, actions, scope, amounts and errors; load, save | [F027](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:117) |
| [web/src/routes/app/settings/data/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/settings/data/+page.svelte>) | Route /app/settings/data: +page.svelte; display, actions, scope, amounts and errors; save | No separate finding |
| [web/src/routes/app/settings/notifications/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/settings/notifications/+page.svelte>) | Route /app/settings/notifications: +page.svelte; display, actions, scope, amounts and errors; preferences, load, save | No separate finding |
| [web/src/routes/app/settings/privacy/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/settings/privacy/+page.svelte>) | Route /app/settings/privacy: +page.svelte; display, actions, scope, amounts and errors; privacyRecord, load, submit, download | No separate finding |
| [web/src/routes/app/settings/security/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/settings/security/+page.svelte>) | Route /app/settings/security: +page.svelte; display, actions, scope, amounts and errors; load, beginEnrollment, verify, regenerateCodes | No separate finding |
| [web/src/routes/app/settings/settlement/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/settings/settlement/+page.svelte>) | Route /app/settings/settlement: +page.svelte; display, actions, scope, amounts and errors; load, save | [F018](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:81), [F027](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:117) |
| [web/src/routes/app/team/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/team/+page.svelte>) | Route /app/team: +page.svelte; display, actions, scope, amounts and errors; decodeMember, intent, load, invite and 1 other declarations | No separate finding |
| [web/src/routes/app/trade-lines/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/trade-lines/+page.svelte>) | Route /app/trade-lines: +page.svelte; display, actions, scope, amounts and errors; load, selectCustomer, switchBusiness, create | No separate finding |
| [web/src/routes/app/trade-lines/[id]/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/trade-lines/[id]/+page.svelte>) | Route /app/trade-lines/[id]: +page.svelte; display, actions, scope, amounts and errors; load, command, reserve, reduce and 1 other declarations | No separate finding |
| [web/src/routes/blog/+layout.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/blog/+layout.svelte>) | Route /blog: +layout.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/blog/+page.server.ts](</Users/macbookpro/Documents/Kredit.com/web/src/routes/blog/+page.server.ts>) | Route /blog: +page.server.ts; HTTP/SSR data and publication boundary | No separate finding |
| [web/src/routes/blog/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/blog/+page.svelte>) | Route /blog: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/blog/[slug]/+page.server.ts](</Users/macbookpro/Documents/Kredit.com/web/src/routes/blog/[slug]/+page.server.ts>) | Route /blog/[slug]: +page.server.ts; HTTP/SSR data and publication boundary | No separate finding |
| [web/src/routes/blog/[slug]/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/blog/[slug]/+page.svelte>) | Route /blog/[slug]: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/blog/rss.xml/+server.ts](</Users/macbookpro/Documents/Kredit.com/web/src/routes/blog/rss.xml/+server.ts>) | Route /blog/rss.xml: +server.ts; HTTP/SSR data and publication boundary | No separate finding |
| [web/src/routes/blog/topic/[topic]/+page.server.ts](</Users/macbookpro/Documents/Kredit.com/web/src/routes/blog/topic/[topic]/+page.server.ts>) | Route /blog/topic/[topic]: +page.server.ts; HTTP/SSR data and publication boundary | No separate finding |
| [web/src/routes/blog/topic/[topic]/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/blog/topic/[topic]/+page.svelte>) | Route /blog/topic/[topic]: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/buyer-invitations/+layout.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer-invitations/+layout.svelte>) | Route /buyer-invitations: +layout.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/buyer-invitations/[token]/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer-invitations/[token]/+page.svelte>) | Route /buyer-invitations/[token]: +page.svelte; display, actions, scope, amounts and errors; loadPreview, requestCode, accept | [F028](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:121) |
| [web/src/routes/buyer/+layout.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/+layout.svelte>) | Route /buyer: +layout.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/buyer/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/+page.svelte>) | Route /buyer: +page.svelte; display, actions, scope, amounts and errors; decodeDays, load | No separate finding |
| [web/src/routes/buyer/amendments/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/amendments/+page.svelte>) | Route /buyer/amendments: +page.svelte; display, actions, scope, amounts and errors; changeRecord, load, decide | No separate finding |
| [web/src/routes/buyer/credit-requests/[requestID]/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/credit-requests/[requestID]/+page.svelte>) | Route /buyer/credit-requests/[requestID]: +page.svelte; display, actions, scope, amounts and errors; intent, load, acceptSale, authorizeBank and 5 other declarations | [F029](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:125) |
| [web/src/routes/buyer/disputes/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/disputes/+page.svelte>) | Route /buyer/disputes: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/buyer/disputes/[id]/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/disputes/[id]/+page.svelte>) | Route /buyer/disputes/[id]: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/buyer/history/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/history/+page.svelte>) | Route /buyer/history: +page.svelte; display, actions, scope, amounts and errors; saleLabel, decodeHistory, load, askForCorrection | No separate finding |
| [web/src/routes/buyer/mandates/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/mandates/+page.svelte>) | Route /buyer/mandates: +page.svelte; display, actions, scope, amounts and errors; mandateRecord, load, command | No separate finding |
| [web/src/routes/buyer/notifications/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/notifications/+page.svelte>) | Route /buyer/notifications: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/buyer/obligations/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/obligations/+page.svelte>) | Route /buyer/obligations: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/buyer/obligations/[id]/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/obligations/[id]/+page.svelte>) | Route /buyer/obligations/[id]: +page.svelte; display, actions, scope, amounts and errors; decode, loadSale, acknowledgeNotice | No separate finding |
| [web/src/routes/buyer/payments/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/payments/+page.svelte>) | Route /buyer/payments: +page.svelte; display, actions, scope, amounts and errors; claimRecord, load | No separate finding |
| [web/src/routes/buyer/permissions/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/permissions/+page.svelte>) | Route /buyer/permissions: +page.svelte; display, actions, scope, amounts and errors; consentRecord, current, load, save | No separate finding |
| [web/src/routes/buyer/requests/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/requests/+page.svelte>) | Route /buyer/requests: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/buyer/settings/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/settings/+page.svelte>) | Route /buyer/settings: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/buyer/trade-lines/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/trade-lines/+page.svelte>) | Route /buyer/trade-lines: +page.svelte; display, actions, scope, amounts and errors; load, command | No separate finding |
| [web/src/routes/c/+layout.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/c/+layout.svelte>) | Route /c: +layout.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/c/[token]/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/c/[token]/+page.svelte>) | Route /c/[token]: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/c/[token]/+page.ts](</Users/macbookpro/Documents/Kredit.com/web/src/routes/c/[token]/+page.ts>) | Route /c/[token]: +page.ts; HTTP/SSR data and publication boundary; load | No separate finding |
| [web/src/routes/contact/+page.server.ts](</Users/macbookpro/Documents/Kredit.com/web/src/routes/contact/+page.server.ts>) | Route /contact: +page.server.ts; HTTP/SSR data and publication boundary | No separate finding |
| [web/src/routes/contact/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/contact/+page.svelte>) | Route /contact: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/demo/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/demo/+page.svelte>) | Route /demo: +page.svelte; display, actions, scope, amounts and errors; money, next, restart | No separate finding |
| [web/src/routes/faq/+page.server.ts](</Users/macbookpro/Documents/Kredit.com/web/src/routes/faq/+page.server.ts>) | Route /faq: +page.server.ts; HTTP/SSR data and publication boundary | No separate finding |
| [web/src/routes/faq/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/faq/+page.svelte>) | Route /faq: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/for-buyers/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/for-buyers/+page.svelte>) | Route /for-buyers: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/for-suppliers/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/for-suppliers/+page.svelte>) | Route /for-suppliers: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/glossary/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/glossary/+page.svelte>) | Route /glossary: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/how-it-works/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/how-it-works/+page.svelte>) | Route /how-it-works: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/legal/complaints/+page.server.ts](</Users/macbookpro/Documents/Kredit.com/web/src/routes/legal/complaints/+page.server.ts>) | Route /legal/complaints: +page.server.ts; HTTP/SSR data and publication boundary; load | No separate finding |
| [web/src/routes/legal/complaints/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/legal/complaints/+page.svelte>) | Route /legal/complaints: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/legal/privacy/+page.server.ts](</Users/macbookpro/Documents/Kredit.com/web/src/routes/legal/privacy/+page.server.ts>) | Route /legal/privacy: +page.server.ts; HTTP/SSR data and publication boundary; load | No separate finding |
| [web/src/routes/legal/privacy/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/legal/privacy/+page.svelte>) | Route /legal/privacy: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/legal/terms/+page.server.ts](</Users/macbookpro/Documents/Kredit.com/web/src/routes/legal/terms/+page.server.ts>) | Route /legal/terms: +page.server.ts; HTTP/SSR data and publication boundary; load | No separate finding |
| [web/src/routes/legal/terms/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/legal/terms/+page.svelte>) | Route /legal/terms: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/pay/+layout.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/pay/+layout.svelte>) | Route /pay: +layout.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/pay/[token]/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/pay/[token]/+page.svelte>) | Route /pay/[token]: +page.svelte; display, actions, scope, amounts and errors; load | [F029](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:125) |
| [web/src/routes/pricing/+page.server.ts](</Users/macbookpro/Documents/Kredit.com/web/src/routes/pricing/+page.server.ts>) | Route /pricing: +page.server.ts; HTTP/SSR data and publication boundary | No separate finding |
| [web/src/routes/pricing/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/pricing/+page.svelte>) | Route /pricing: +page.svelte; display, actions, scope, amounts and errors; loadRates | No separate finding |
| [web/src/routes/receipt/+layout.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/receipt/+layout.svelte>) | Route /receipt: +layout.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/receipt/[public_token]/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/receipt/[public_token]/+page.svelte>) | Route /receipt/[public_token]: +page.svelte; display, actions, scope, amounts and errors; load | No separate finding |
| [web/src/routes/recover/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/recover/+page.svelte>) | Route /recover: +page.svelte; display, actions, scope, amounts and errors; post, start, addRecoveryCode, requestContactCode and 4 other declarations | [F012](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:57) |
| [web/src/routes/robots.txt/+server.ts](</Users/macbookpro/Documents/Kredit.com/web/src/routes/robots.txt/+server.ts>) | Route /robots.txt: +server.ts; HTTP/SSR data and publication boundary; GET | [F034](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:145) |
| [web/src/routes/secure/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/secure/+page.svelte>) | Route /secure: +page.svelte; display, actions, scope, amounts and errors; load | No separate finding |
| [web/src/routes/security/+page.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/routes/security/+page.svelte>) | Route /security: +page.svelte; display, actions, scope, amounts and errors | No separate finding |
| [web/src/routes/sitemap.xml/+server.ts](</Users/macbookpro/Documents/Kredit.com/web/src/routes/sitemap.xml/+server.ts>) | Route /sitemap.xml: +server.ts; HTTP/SSR data and publication boundary; topicLastmod | No separate finding |
| [web/src/service-worker.ts](</Users/macbookpro/Documents/Kredit.com/web/src/service-worker.ts>) | service-worker: shared UI/data contract, safe rendering and consumers | No separate finding |

## web/static

| File | Reviewed content | Result |
| --- | --- | --- |
| [web/static/apple-touch-icon.png](</Users/macbookpro/Documents/Kredit.com/web/static/apple-touch-icon.png>) | Published image; brand, domain and consumers | No separate finding |
| [web/static/favicon.svg](</Users/macbookpro/Documents/Kredit.com/web/static/favicon.svg>) | favicon.svg: configuration/data and current consumers | No separate finding |
| [web/static/icon-192.png](</Users/macbookpro/Documents/Kredit.com/web/static/icon-192.png>) | Published image; brand, domain and consumers | No separate finding |
| [web/static/icon-512.png](</Users/macbookpro/Documents/Kredit.com/web/static/icon-512.png>) | Published image; brand, domain and consumers | No separate finding |
| [web/static/icon-maskable.svg](</Users/macbookpro/Documents/Kredit.com/web/static/icon-maskable.svg>) | icon-maskable.svg: configuration/data and current consumers | No separate finding |
| [web/static/icon.svg](</Users/macbookpro/Documents/Kredit.com/web/static/icon.svg>) | icon.svg: configuration/data and current consumers | No separate finding |
| [web/static/manifest.webmanifest](</Users/macbookpro/Documents/Kredit.com/web/static/manifest.webmanifest>) | manifest.webmanifest: configuration/data and current consumers | No separate finding |
| [web/static/og-source.svg](</Users/macbookpro/Documents/Kredit.com/web/static/og-source.svg>) | og-source.svg: configuration/data and current consumers | [F035](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:149) |
| [web/static/og.jpg](</Users/macbookpro/Documents/Kredit.com/web/static/og.jpg>) | Published image; brand, domain and consumers | [F035](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:149) |
| [web/static/og.png](</Users/macbookpro/Documents/Kredit.com/web/static/og.png>) | Published image; brand, domain and consumers | [F035](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:149) |

## web/tests

| File | Reviewed content | Result |
| --- | --- | --- |
| [web/tests/access-control.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/access-control.spec.ts>) | access-control.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/accessibility.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/accessibility.spec.ts>) | accessibility.spec: fixtures, assertions and production equivalence; mockAPI, expectNoSeriousViolations | No separate finding |
| [web/tests/account-safety-outage.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/account-safety-outage.spec.ts>) | account-safety-outage.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/admin-controls-retry.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/admin-controls-retry.spec.ts>) | admin-controls-retry.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/admin-outage.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/admin-outage.spec.ts>) | admin-outage.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/admin-section.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/admin-section.spec.ts>) | admin-section.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/admin-workflows.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/admin-workflows.spec.ts>) | admin-workflows.spec: fixtures, assertions and production equivalence; auth | No separate finding |
| [web/tests/article-drafts.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/article-drafts.spec.ts>) | article-drafts.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/audit-completion.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/audit-completion.spec.ts>) | audit-completion.spec: fixtures, assertions and production equivalence; signedIn, draftCount, saleDetail | No separate finding |
| [web/tests/audit-fixes.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/audit-fixes.spec.ts>) | audit-fixes.spec: fixtures, assertions and production equivalence | [F040](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:169) |
| [web/tests/audit-full-form.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/audit-full-form.spec.ts>) | audit-full-form.spec: fixtures, assertions and production equivalence; prepare | No separate finding |
| [web/tests/audit-product-journeys.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/audit-product-journeys.spec.ts>) | audit-product-journeys.spec: fixtures, assertions and production equivalence; signedIn | No separate finding |
| [web/tests/audit-reliability-unit.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/audit-reliability-unit.spec.ts>) | audit-reliability-unit.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/business-settings.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/business-settings.spec.ts>) | business-settings.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/completion-history.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/completion-history.spec.ts>) | completion-history.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/content-seo.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/content-seo.spec.ts>) | content-seo.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/customer-limit-recovery.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/customer-limit-recovery.spec.ts>) | customer-limit-recovery.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/demo-readiness.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/demo-readiness.spec.ts>) | demo-readiness.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/dispute-detail-outage.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/dispute-detail-outage.spec.ts>) | dispute-detail-outage.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/financial-completion.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/financial-completion.spec.ts>) | financial-completion.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/health.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/health.spec.ts>) | health.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/money.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/money.spec.ts>) | money.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/notification-history-outage.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/notification-history-outage.spec.ts>) | notification-history-outage.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/privacy-partial-approval.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/privacy-partial-approval.spec.ts>) | privacy-partial-approval.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/product-flows.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/product-flows.spec.ts>) | product-flows.spec: fixtures, assertions and production equivalence; mockOperationsCommand | No separate finding |
| [web/tests/product-quality.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/product-quality.spec.ts>) | product-quality.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/public-money-links.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/public-money-links.spec.ts>) | public-money-links.spec: fixtures, assertions and production equivalence | [F040](/Users/macbookpro/Documents/Kredit.com/docs/launch-audit-2026-09-12/FINDINGS.md:169) |
| [web/tests/public-security-content.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/public-security-content.spec.ts>) | public-security-content.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/publication-rendering.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/publication-rendering.spec.ts>) | publication-rendering.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/real-stack-financial.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/real-stack-financial.spec.ts>) | real-stack-financial.spec: fixtures, assertions and production equivalence; login | No separate finding |
| [web/tests/registration-recovery.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/registration-recovery.spec.ts>) | registration-recovery.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/sign-in.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/sign-in.spec.ts>) | sign-in.spec: fixtures, assertions and production equivalence; stubChallenge | No separate finding |
| [web/tests/website-editor.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/website-editor.spec.ts>) | website-editor.spec: fixtures, assertions and production equivalence | No separate finding |
| [web/tests/workspace-audit.spec.ts](</Users/macbookpro/Documents/Kredit.com/web/tests/workspace-audit.spec.ts>) | workspace-audit.spec: fixtures, assertions and production equivalence | No separate finding |
