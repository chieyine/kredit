# Kredit file-by-file audit register

Project-file review complete. Every one of the 1,147 project paths in the original inventory has a recorded disposition: 1,140 tracked files and seven local untracked files. No agents, tests, builds, linters, application execution or database execution were used. Application files were not changed.

| Review method | Files |
| --- | ---: |
| read-complete | 1070 |
| visual-review-complete | 24 |
| structured-artifact-review | 47 |
| metadata-only-secret-exclusion | 6 |

“read-complete” means the file text was read, not that every behavior is proved correct. Test source was read without running it. Structured artifacts were reviewed through their complete records, formats, provenance and diagnostic summaries; archived patch inspection covers file/hunk inventory, not every obsolete code body. Images were opened individually. Environment/SSH content was excluded; filenames, permissions and exclusion rules were inspected.

The full physical inventory also accounts for 41,832 generated/temporary paths, 5,820 installed-dependency paths, and 6,492 Git metadata paths. These 54,144 paths were inventoried, not individually source-audited. Third-party implementation code, caches, local service databases and remote production configuration are outside the source review.

Notes were recorded as the audit progressed. Earlier references to pending cross-file work are preserved as chronological notes, not pending file reads or claims that a concern was fixed. The final confirmed findings are in [FINDINGS.md](FINDINGS.md); unresolved hypotheses and limits are in [REVIEW-LIMITS.md](REVIEW-LIMITS.md). No-separate-finding is not a guarantee of correctness.


## Root files

| File | Review | Notes |
| --- | --- | --- |
| [.dockerignore](../../../.dockerignore) | read-complete | Read all: key and env exclusion. Generated web/.vercel and root .svelte-kit are not excluded, but no confirmed runtime disclosure traced. |
| `.env` | metadata-only-secret-exclusion | Filename, size (2698 bytes), permissions (0o644) and Git/Docker exclusion inspected. Contents deliberately not opened or copied into audit. Private keys are owner-only; .env is readable by other local users and public-key/known-host files are world-writable: tighten local permissions if this workstation has other users. No credential-use or exposure claim. |
| [.env.example](../../../.env.example) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |

## .github/workflows

| File | Review | Notes |
| --- | --- | --- |
| [ci.yml](../../../.github/workflows/ci.yml) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [phase2-tenant-isolation.yml](../../../.github/workflows/phase2-tenant-isolation.yml) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [phase3-financial-proof.yml](../../../.github/workflows/phase3-financial-proof.yml) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [phase4-provider-verification.yml](../../../.github/workflows/phase4-provider-verification.yml) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [phase5-production-assurance.yml](../../../.github/workflows/phase5-production-assurance.yml) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [phase6-context-audit.yml](../../../.github/workflows/phase6-context-audit.yml) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [product-audit.yml](../../../.github/workflows/product-audit.yml) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |

## Root files

| File | Review | Notes |
| --- | --- | --- |
| [.gitignore](../../../.gitignore) | read-complete | Read all: secrets, local keys, generated output and test artifacts excluded from version control. |
| [.go-version](../../../.go-version) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |
| [.golangci.yml](../../../.golangci.yml) | read-complete | Read all: lint/format configuration. No lint executed. |
| [.node-version](../../../.node-version) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |
| [.pnpm-version](../../../.pnpm-version) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |
| [CHANGELOG.md](../../../CHANGELOG.md) | read-complete | Read complete historical status/change record. September 9 header and earlier completed-wave claims are stale relative to current migration 154 and findings; current evidence must come from this audit, not these historical test claims. |

## Claude outputs

| File | Review | Notes |
| --- | --- | --- |
| ci.yml (`Claude outputs/ci.yml`, since removed) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |

## Root files

| File | Review | Notes |
| --- | --- | --- |
| [IMPLEMENTATION_PLAN.md](../../../IMPLEMENTATION_PLAN.md) | read-complete | Read complete documentation; distinguished historical plans/check results from current implementation and this source-only audit. No documented command executed. |
| [IMPLEMENTATION_STATUS.md](../../../IMPLEMENTATION_STATUS.md) | read-complete | Read complete historical status/change record. September 9 header and earlier completed-wave claims are stale relative to current migration 154 and findings; current evidence must come from this audit, not these historical test claims. |
| [KREDIT-CODE-AUDIT.md](../../../KREDIT-CODE-AUDIT.md) | read-complete | Read all 136 historical observations, business model, plan, suggested fixes and coverage appendix. Several entries are preferences, explicitly unverified suspicions or now-removed source; not adopted as current findings. Cookie existence is not authenticated session proof (K033); historical fee nil semantics preserve old accepted contracts; current adapters and migrations materially exceed this snapshot. |
| [KREDIT-LAUNCH-HANDOVER.md](../../../KREDIT-LAUNCH-HANDOVER.md) | read-complete | Read complete documentation; distinguished historical plans/check results from current implementation and this source-only audit. No documented command executed. |
| [KREDIT-TEST-FIXES-ROUND-2.md](../../../KREDIT-TEST-FIXES-ROUND-2.md) | read-complete | Read complete documentation; distinguished historical plans/check results from current implementation and this source-only audit. No documented command executed. |
| [KREDIT-TEST-FIXES.md](../../../KREDIT-TEST-FIXES.md) | read-complete | Read complete documentation; distinguished historical plans/check results from current implementation and this source-only audit. No documented command executed. |
| [Kredit_Local_Launch_Master_Prompt_v2_2026-09-07.md](../../../Kredit_Local_Launch_Master_Prompt_v2_2026-09-07.md) | read-complete | Read complete historical implementation brief as an audit artifact, not a new instruction to implement/test/deploy. Requirements for actual paid dates, completion across terminal states, operator controls and source-bound evidence remain useful comparison points. Latest user audit-only/no-tests/no-agents request governs this work. |
| [README.local.md](../../../README.local.md) | read-complete | Reviewed startup steps and explicit in-progress audit/provider approval caveat. |
| [README.md](<README.md>) | read-complete | Read entire master specification. Current contract needs reconciliation: consumer lending excluded while consumer module is implemented; docs/api/openapi.yaml described as an index but contains stale duplicate specification; milestone 9 WhatsApp creation exit conflicts with section 29 read-back-only safety boundary. Representative endpoint/data-model examples are design requirements, not an accurate live API inventory. Explicit receipt, payment-date, trade-line and immutable-term requirements substantiate findings F017–F022. No tests or cited external standards verified. |
| [SECURITY.md](../../../SECURITY.md) | read-complete | Reviewed reporting, secret handling, production gates and scope. Private reporting contact is described generically without a direct address. |
| [Taskfile.yml](../../../Taskfile.yml) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |

## api

| File | Review | Notes |
| --- | --- | --- |
| [openapi.yaml](../../../api/openapi.yaml) | read-complete | Read all 5,089 lines: endpoints, security declarations, idempotency parameters and schemas. Many operations lack concrete response/request schemas, so route coverage alone cannot prove payload compatibility. FeeTerms omits min_fee_kobo; legal acceptance preview has no shape. |

## cmd/api

| File | Review | Notes |
| --- | --- | --- |
| [main.go](../../../cmd/api/main.go) | read-complete | Read all: database/schema startup, connection settings, HTTP timeouts, shutdown and local health probe. Probe tests only liveness, not configured-provider readiness; deployment impact pending. |
| [main_test.go](../../../cmd/api/main_test.go) | read-complete | Read in full as source only; no tests executed. Reviewed command setup and assertions. |

## cmd/backup-r2

| File | Review | Notes |
| --- | --- | --- |
| [main.go](../../../cmd/backup-r2/main.go) | read-complete | Read all: dump streaming, local permissions, checksums, upload and retention. Confirmed unscoped deletion of old files in configured directory; placeholder upload configuration exits successfully. |

## cmd/bootstrap-owner

| File | Review | Notes |
| --- | --- | --- |
| [main.go](../../../cmd/bootstrap-owner/main.go) | read-complete | Read all: one-use table lock, active recently elevated account, atomic role/governance/audit writes. No confirmed defect. |
| [main_test.go](../../../cmd/bootstrap-owner/main_test.go) | read-complete | Read in full as source only; no tests executed. Reviewed command setup and assertions. |

## cmd/configcheck

| File | Review | Notes |
| --- | --- | --- |
| [main.go](../../../cmd/configcheck/main.go) | read-complete | Read all: environment/stored settings and readiness checks. No execution. |
| [main_test.go](../../../cmd/configcheck/main_test.go) | read-complete | Read in full as source only; no tests executed. Reviewed command setup and assertions. |

## cmd/migrate

| File | Review | Notes |
| --- | --- | --- |
| [main.go](../../../cmd/migrate/main.go) | read-complete | Read all: migration connection selection, guarded down command, Goose and River sequencing. No confirmed defect in this pass. |
| [main_test.go](../../../cmd/migrate/main_test.go) | read-complete | Read in full as source only; no tests executed. Reviewed command setup and assertions. |

## cmd/provider-simulator

| File | Review | Notes |
| --- | --- | --- |
| [main.go](../../../cmd/provider-simulator/main.go) | read-complete | Read in full as source only. Simulator request bounds, scenario handling, stable identifiers and replay behavior reviewed. |
| [main_test.go](../../../cmd/provider-simulator/main_test.go) | read-complete | Read in full as source only; no tests executed. Reviewed command setup and assertions. |

## cmd/reconcile

| File | Review | Notes |
| --- | --- | --- |
| [main.go](../../../cmd/reconcile/main.go) | read-complete | Read all: worker-only connection and imbalance exit status. No execution. |
| [main_test.go](../../../cmd/reconcile/main_test.go) | read-complete | Read in full as source only; no tests executed. Reviewed command setup and assertions. |

## cmd/seed

| File | Review | Notes |
| --- | --- | --- |
| [main.go](../../../cmd/seed/main.go) | read-complete | Read all: development-only seed and policy initialization. Seed SQL remains pending. |
| [main_test.go](../../../cmd/seed/main_test.go) | read-complete | Read in full as source only; no tests executed. Reviewed command setup and assertions. |

## cmd/worker

| File | Review | Notes |
| --- | --- | --- |
| [main.go](../../../cmd/worker/main.go) | read-complete | Read all: worker/provider initialization, job handlers, periodic discovery, outbox, readiness and shutdown. Shared serial maintenance loop can delay other discovery while external lookups wait; impact pending. |
| [main_test.go](../../../cmd/worker/main_test.go) | read-complete | Read in full as source only; no tests executed. Reviewed command setup and assertions. |

## db/migrations

| File | Review | Notes |
| --- | --- | --- |
| [001_initial.sql](../../../db/migrations/001_initial.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [002_milestone1_auth_org.sql](../../../db/migrations/002_milestone1_auth_org.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [003_milestone2_buyers_identity.sql](../../../db/migrations/003_milestone2_buyers_identity.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [004_milestone3_credit_ledger.sql](../../../db/migrations/004_milestone3_credit_ledger.sql) | read-complete | Read all: original credit/mandate/agreement/ledger tables and initial RLS. Random posting UUID default confirms F001. Later migration overrides not fully reviewed. |
| [005_milestone4_payments.sql](../../../db/migrations/005_milestone4_payments.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [006_milestone5_schedules.sql](../../../db/migrations/006_milestone5_schedules.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [007_milestone6_trade_lines.sql](../../../db/migrations/007_milestone6_trade_lines.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [008_milestone7_collections.sql](../../../db/migrations/008_milestone7_collections.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [009_milestone8_disputes_operations.sql](../../../db/migrations/009_milestone8_disputes_operations.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [010_milestone9_notifications_whatsapp.sql](../../../db/migrations/010_milestone9_notifications_whatsapp.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [011_milestone10_reporting_history.sql](../../../db/migrations/011_milestone10_reporting_history.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [012_milestone11_provider_adapter.sql](../../../db/migrations/012_milestone11_provider_adapter.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [013_milestone12_release_readiness.sql](../../../db/migrations/013_milestone12_release_readiness.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [014_river_jobs_schema.sql](../../../db/migrations/014_river_jobs_schema.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [015_missing_domain_tables.sql](../../../db/migrations/015_missing_domain_tables.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [016_financial_core_hardening.sql](../../../db/migrations/016_financial_core_hardening.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [017_worker_operations.sql](../../../db/migrations/017_worker_operations.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [018_security_hardening.sql](../../../db/migrations/018_security_hardening.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [019_runtime_role_policies.sql](../../../db/migrations/019_runtime_role_policies.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [020_auth_persistence_primitives.sql](../../../db/migrations/020_auth_persistence_primitives.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [021_organization_persistence_primitives.sql](../../../db/migrations/021_organization_persistence_primitives.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [022_network_and_mandates.sql](../../../db/migrations/022_network_and_mandates.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [023_buyer_persistence_primitives.sql](../../../db/migrations/023_buyer_persistence_primitives.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [024_credit_aggregate_snapshots.sql](../../../db/migrations/024_credit_aggregate_snapshots.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [025_mandate_lookup_function.sql](../../../db/migrations/025_mandate_lookup_function.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [026_mandate_lookup_owner.sql](../../../db/migrations/026_mandate_lookup_owner.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [027_credit_snapshot_tenant_functions.sql](../../../db/migrations/027_credit_snapshot_tenant_functions.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [028_outbox_processing_lease.sql](../../../db/migrations/028_outbox_processing_lease.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [029_runtime_domain_repository_policies.sql](../../../db/migrations/029_runtime_domain_repository_policies.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [030_schedule_repository_runtime_policy.sql](../../../db/migrations/030_schedule_repository_runtime_policy.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [031_atomic_payment_repository.sql](../../../db/migrations/031_atomic_payment_repository.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [032_supplier_customer_directory.sql](../../../db/migrations/032_supplier_customer_directory.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [033_fix_ledger_balance_trigger.sql](../../../db/migrations/033_fix_ledger_balance_trigger.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [034_trade_line_runtime_policy.sql](../../../db/migrations/034_trade_line_runtime_policy.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [035_collection_aggregate_repository.sql](../../../db/migrations/035_collection_aggregate_repository.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [036_corrections_runtime_policy.sql](../../../db/migrations/036_corrections_runtime_policy.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [037_dispute_operation_runtime_policy.sql](../../../db/migrations/037_dispute_operation_runtime_policy.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [038_notification_delivery_repository.sql](../../../db/migrations/038_notification_delivery_repository.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [039_analytics_runtime_policy.sql](../../../db/migrations/039_analytics_runtime_policy.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [040_messaging_runtime_policy.sql](../../../db/migrations/040_messaging_runtime_policy.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [041_notification_delivery_recovery.sql](../../../db/migrations/041_notification_delivery_recovery.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [042_platform_operations_roles.sql](../../../db/migrations/042_platform_operations_roles.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [043_fix_auth_user_upsert_ambiguity.sql](../../../db/migrations/043_fix_auth_user_upsert_ambiguity.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [044_payment_sources_and_claims.sql](../../../db/migrations/044_payment_sources_and_claims.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [045_trade_line_mandate_integrity.sql](../../../db/migrations/045_trade_line_mandate_integrity.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [046_trade_line_drawdown_lifecycle.sql](../../../db/migrations/046_trade_line_drawdown_lifecycle.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [047_supplier_onboarding_readiness.sql](../../../db/migrations/047_supplier_onboarding_readiness.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [048_user_control_and_privacy.sql](../../../db/migrations/048_user_control_and_privacy.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [049_operations_controls.sql](../../../db/migrations/049_operations_controls.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [050_product_analytics.sql](../../../db/migrations/050_product_analytics.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [051_analytics_contract_completion.sql](../../../db/migrations/051_analytics_contract_completion.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [052_sweep_collection_safety.sql](../../../db/migrations/052_sweep_collection_safety.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [053_payment_reversal_integrity.sql](../../../db/migrations/053_payment_reversal_integrity.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [054_collection_eligibility_lock.sql](../../../db/migrations/054_collection_eligibility_lock.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [055_durable_financial_notices.sql](../../../db/migrations/055_durable_financial_notices.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [056_financial_reconciliation.sql](../../../db/migrations/056_financial_reconciliation.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [057_collection_notice_delivery.sql](../../../db/migrations/057_collection_notice_delivery.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [058_operations_intent_integrity.sql](../../../db/migrations/058_operations_intent_integrity.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [059_adjustment_notification_evidence.sql](../../../db/migrations/059_adjustment_notification_evidence.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [060_provider_reversal_review.sql](../../../db/migrations/060_provider_reversal_review.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [061_business_policies.sql](../../../db/migrations/061_business_policies.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [062_policy_enforcement_and_fee_terms.sql](../../../db/migrations/062_policy_enforcement_and_fee_terms.sql) | read-complete | Read all SQL, including rollback. Reviewed schema constraints, trigger conditions, function grants and access policies. Final-schema cross-file review remains in progress. |
| [063_preserve_existing_commitments.sql](../../../db/migrations/063_preserve_existing_commitments.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [064_admin_workflows.sql](../../../db/migrations/064_admin_workflows.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [065_admin_attention_details.sql](../../../db/migrations/065_admin_attention_details.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [066_admin_support_review_queue.sql](../../../db/migrations/066_admin_support_review_queue.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [067_canonical_phone_identifiers.sql](../../../db/migrations/067_canonical_phone_identifiers.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [068_session_idle_and_mfa_throttle.sql](../../../db/migrations/068_session_idle_and_mfa_throttle.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [069_shared_authentication_rate_limits.sql](../../../db/migrations/069_shared_authentication_rate_limits.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [070_deemed_acceptance_evidence.sql](../../../db/migrations/070_deemed_acceptance_evidence.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [071_audit_financial_integrity.sql](../../../db/migrations/071_audit_financial_integrity.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [072_repair_audit_legacy_state.sql](../../../db/migrations/072_repair_audit_legacy_state.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [073_audit_security_invariants.sql](../../../db/migrations/073_audit_security_invariants.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [074_mfa_replay_and_rotation.sql](../../../db/migrations/074_mfa_replay_and_rotation.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [075_document_upload_lifecycle.sql](../../../db/migrations/075_document_upload_lifecycle.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [076_buyer_notice_acknowledgement.sql](../../../db/migrations/076_buyer_notice_acknowledgement.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [077_document_tenant_rls.sql](../../../db/migrations/077_document_tenant_rls.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [078_runtime_role_idempotency.sql](../../../db/migrations/078_runtime_role_idempotency.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [079_notice_acknowledgement_permissions.sql](../../../db/migrations/079_notice_acknowledgement_permissions.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [080_collected_payment_reversal_provenance.sql](../../../db/migrations/080_collected_payment_reversal_provenance.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [081_phase2_tenant_isolation.sql](../../../db/migrations/081_phase2_tenant_isolation.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [082_phase2_collection_tenant_isolation.sql](../../../db/migrations/082_phase2_collection_tenant_isolation.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [083_phase2_buyer_evidence_guard.sql](../../../db/migrations/083_phase2_buyer_evidence_guard.sql) | read-complete | Read all SQL and rollback; traced commitment preservation, role policies, evidence provenance and successive replacements. Final-schema verification remains in progress. |
| [084_phase2_collection_worker_boundaries.sql](../../../db/migrations/084_phase2_collection_worker_boundaries.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [085_phase2_provider_collection_identity.sql](../../../db/migrations/085_phase2_provider_collection_identity.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [086_phase5_financial_monitoring.sql](../../../db/migrations/086_phase5_financial_monitoring.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [087_owner_super_admin_controls.sql](../../../db/migrations/087_owner_super_admin_controls.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [088_settings_history_immutable.sql](../../../db/migrations/088_settings_history_immutable.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [089_owner_lifecycle_guard.sql](../../../db/migrations/089_owner_lifecycle_guard.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [090_governance_fail_closed.sql](../../../db/migrations/090_governance_fail_closed.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [091_retire_unread_platform_settings.sql](../../../db/migrations/091_retire_unread_platform_settings.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [092_public_payment_receipt.sql](../../../db/migrations/092_public_payment_receipt.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [093_admin_notification_connectors.sql](../../../db/migrations/093_admin_notification_connectors.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [094_admin_runtime_connections.sql](../../../db/migrations/094_admin_runtime_connections.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [095_delivery_issue_blocks_deemed_acceptance.sql](../../../db/migrations/095_delivery_issue_blocks_deemed_acceptance.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [096_close_legacy_tenant_bypasses.sql](../../../db/migrations/096_close_legacy_tenant_bypasses.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [097_collection_notice_versions.sql](../../../db/migrations/097_collection_notice_versions.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [098_provider_webhook_claims.sql](../../../db/migrations/098_provider_webhook_claims.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [099_notification_send_identity.sql](../../../db/migrations/099_notification_send_identity.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [100_preserve_agreement_bytes.sql](../../../db/migrations/100_preserve_agreement_bytes.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [101_owner_website_content.sql](../../../db/migrations/101_owner_website_content.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [102_relationship_consent_evidence.sql](../../../db/migrations/102_relationship_consent_evidence.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [103_solo_owner_privacy_completion.sql](../../../db/migrations/103_solo_owner_privacy_completion.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [104_versioned_legal_publications.sql](../../../db/migrations/104_versioned_legal_publications.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [105_buyer_supplier_directory.sql](../../../db/migrations/105_buyer_supplier_directory.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [106_consent_relationship_scope.sql](../../../db/migrations/106_consent_relationship_scope.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [107_notification_intent_fingerprint.sql](../../../db/migrations/107_notification_intent_fingerprint.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [108_admin_document_scan_recovery.sql](../../../db/migrations/108_admin_document_scan_recovery.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [109_guide_and_contact_publication.sql](../../../db/migrations/109_guide_and_contact_publication.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [110_notification_recipient_scope.sql](../../../db/migrations/110_notification_recipient_scope.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [111_team_invitation_lifecycle.sql](../../../db/migrations/111_team_invitation_lifecycle.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [112_system_acceptance_evidence.sql](../../../db/migrations/112_system_acceptance_evidence.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [113_document_orphan_recovery.sql](../../../db/migrations/113_document_orphan_recovery.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [114_system_acceptance_admin_controls.sql](../../../db/migrations/114_system_acceptance_admin_controls.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [115_atomic_domain_activity.sql](../../../db/migrations/115_atomic_domain_activity.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [116_system_acceptance_settings_access.sql](../../../db/migrations/116_system_acceptance_settings_access.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [117_customer_registration_recovery.sql](../../../db/migrations/117_customer_registration_recovery.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [118_notification_delivery_lookup.sql](../../../db/migrations/118_notification_delivery_lookup.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [119_launch_financial_boundaries.sql](../../../db/migrations/119_launch_financial_boundaries.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [120_admin_read_projections.sql](../../../db/migrations/120_admin_read_projections.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [121_drawdown_expiry_discovery.sql](../../../db/migrations/121_drawdown_expiry_discovery.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [122_scorecard_projections.sql](../../../db/migrations/122_scorecard_projections.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [123_invoice_and_mandate_evidence.sql](../../../db/migrations/123_invoice_and_mandate_evidence.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [124_recovery_account_lookup.sql](../../../db/migrations/124_recovery_account_lookup.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [125_buyer_verification_intents.sql](../../../db/migrations/125_buyer_verification_intents.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [126_mandate_authorization_intents.sql](../../../db/migrations/126_mandate_authorization_intents.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [127_dispute_document_evidence.sql](../../../db/migrations/127_dispute_document_evidence.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [128_billing_verification_required.sql](../../../db/migrations/128_billing_verification_required.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [129_provider_operations_projection.sql](../../../db/migrations/129_provider_operations_projection.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [130_native_message_submissions.sql](../../../db/migrations/130_native_message_submissions.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [131_notification_provider_routes.sql](../../../db/migrations/131_notification_provider_routes.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [132_runtime_process_status.sql](../../../db/migrations/132_runtime_process_status.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [133_settlement_registration.sql](../../../db/migrations/133_settlement_registration.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [134_base_fee_accrual.sql](../../../db/migrations/134_base_fee_accrual.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [135_fee_billing.sql](../../../db/migrations/135_fee_billing.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [136_fee_invoice_refunds.sql](../../../db/migrations/136_fee_invoice_refunds.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [137_native_identity.sql](../../../db/migrations/137_native_identity.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [138_collection_settlement_routes.sql](../../../db/migrations/138_collection_settlement_routes.sql) | read-complete | Read all SQL and rollback. Reviewed restricted discovery functions, audit evidence, provider intents, notification routes, governance, financial projections and billing constraints as applicable. Cross-file application review remains pending. |
| [139_named_mono_boundaries.sql](../../../db/migrations/139_named_mono_boundaries.sql) | read-complete | Read all. Reviewed SQL/application contract, permissions, error propagation and reporting definitions. See F009/F010 for scorecard defects; other cross-file review ongoing. |
| [140_split_fee_allocation.sql](../../../db/migrations/140_split_fee_allocation.sql) | read-complete | Read all. Reviewed SQL/application contract, permissions, error propagation and reporting definitions. See F009/F010 for scorecard defects; other cross-file review ongoing. |
| [141_authorized_fee_billing.sql](../../../db/migrations/141_authorized_fee_billing.sql) | read-complete | Read all. Reviewed SQL/application contract, permissions, error propagation and reporting definitions. See F009/F010 for scorecard defects; other cross-file review ongoing. |
| [142_fee_bank_reconciliation.sql](../../../db/migrations/142_fee_bank_reconciliation.sql) | read-complete | Read all. Reviewed SQL/application contract, permissions, error propagation and reporting definitions. See F009/F010 for scorecard defects; other cross-file review ongoing. |
| [143_fee_setup_review.sql](../../../db/migrations/143_fee_setup_review.sql) | read-complete | Read all. Reviewed SQL/application contract, permissions, error propagation and reporting definitions. See F009/F010 for scorecard defects; other cross-file review ongoing. |
| [144_identity_review_history.sql](../../../db/migrations/144_identity_review_history.sql) | read-complete | Read all. Reviewed SQL/application contract, permissions, error propagation and reporting definitions. See F009/F010 for scorecard defects; other cross-file review ongoing. |
| [145_fee_evidence_guards.sql](../../../db/migrations/145_fee_evidence_guards.sql) | read-complete | Read all. Reviewed SQL/application contract, permissions, error propagation and reporting definitions. See F009/F010 for scorecard defects; other cross-file review ongoing. |
| [146_fee_notice_routing.sql](../../../db/migrations/146_fee_notice_routing.sql) | read-complete | Read all. Reviewed SQL/application contract, permissions, error propagation and reporting definitions. See F009/F010 for scorecard defects; other cross-file review ongoing. |
| [147_personal_fee_consent_export.sql](../../../db/migrations/147_personal_fee_consent_export.sql) | read-complete | Read all. Reviewed SQL/application contract, permissions, error propagation and reporting definitions. See F009/F010 for scorecard defects; other cross-file review ongoing. |
| [148_unpaid_split_fee_bills.sql](../../../db/migrations/148_unpaid_split_fee_bills.sql) | read-complete | Read all. Reviewed SQL/application contract, permissions, error propagation and reporting definitions. See F009/F010 for scorecard defects; other cross-file review ongoing. |
| [149_consumer_sales.sql](../../../db/migrations/149_consumer_sales.sql) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [150_consumer_acceptance_guards.sql](../../../db/migrations/150_consumer_acceptance_guards.sql) | read-complete | Read all: customer-only acceptance/receipt and retailer release checks; superseded readiness function traced through 151/152. |
| [151_consumer_retailer_readiness.sql](../../../db/migrations/151_consumer_retailer_readiness.sql) | read-complete | Read all: locked retailer readiness and acceptance guard. Readiness replaced by 152. |
| [152_consumer_automatic_activation.sql](../../../db/migrations/152_consumer_automatic_activation.sql) | read-complete | Read all: exceptional restrictions, registration-backed bank readiness and eligibility lock. Consumer-domain caller review pending. |
| [153_dsa_referrals.sql](../../../db/migrations/153_dsa_referrals.sql) | read-complete | Read all: referral/reward/payout tables, RLS, qualification facts, commission caps, attribution and agent immutability. Runtime cross-check read; broader predecessor schema review pending. |
| [154_bank_debit_enrollment.sql](../../../db/migrations/154_bank_debit_enrollment.sql) | read-complete | Read all: encrypted enrollment table, forced RLS, vendor uniqueness and grants. F006: operator read policy permits platform_owner only. |

## db/seeds

| File | Review | Notes |
| --- | --- | --- |
| [001_demo.sql](../../../db/seeds/001_demo.sql) | read-complete | Read all synthetic seed statements and conflict handling. Payment reduces trade-line exposure via migration 071 obligation trigger; initial 275m exposure is therefore not a defect. Fixture agreements are synthetic and do not establish production acceptance evidence. Never executed. |

## Root files

| File | Review | Notes |
| --- | --- | --- |
| [docker-compose.yml](../../../docker-compose.yml) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |

## docs/adr

| File | Review | Notes |
| --- | --- | --- |
| [0001-modular-monolith.md](../../adr/0001-modular-monolith.md) | read-complete | Read architecture decision, scope, rationale and consequences. Handwritten HTTP/SQL and canonical API ownership are explicit; historical route/query counts are checkpoints. |
| [0002-financial-source-of-truth.md](../../adr/0002-financial-source-of-truth.md) | read-complete | Read architecture decision, scope, rationale and consequences. Handwritten HTTP/SQL and canonical API ownership are explicit; historical route/query counts are checkpoints. |
| [0003-provider-neutral-identity.md](../../adr/0003-provider-neutral-identity.md) | read-complete | Read architecture decision, scope, rationale and consequences. Handwritten HTTP/SQL and canonical API ownership are explicit; historical route/query counts are checkpoints. |
| [0004-mono-sweep-collection-boundary.md](../../adr/0004-mono-sweep-collection-boundary.md) | read-complete | Read architecture decision, scope, rationale and consequences. Handwritten HTTP/SQL and canonical API ownership are explicit; historical route/query counts are checkpoints. |
| [0005-hand-written-http-and-sql.md](../../adr/0005-hand-written-http-and-sql.md) | read-complete | Read architecture decision, scope, rationale and consequences. Handwritten HTTP/SQL and canonical API ownership are explicit; historical route/query counts are checkpoints. |
| [README.md](../../adr/README.md) | read-complete | Read architecture decision, scope, rationale and consequences. Handwritten HTTP/SQL and canonical API ownership are explicit; historical route/query counts are checkpoints. |

## docs/api

| File | Review | Notes |
| --- | --- | --- |
| [README.md](../../api/README.md) | read-complete | Claims adjacent openapi.yaml is an index pointer, but it is a stale 3,856-line full specification. Canonical API location is correctly linked. |
| [openapi.yaml](../../api/openapi.yaml) | read-complete | Reviewed all content by exact line comparison to the fully read canonical API file: 3,824 matching lines plus 32 separately read differing lines. Stale full specification, not an index pointer: settlement still accepts provider reference/masked account instead of bank_code/account_number; invitation acceptance omits required consents/legal versions; notification callback contract is obsolete. Replace with actual pointer or generated synchronized copy. |

## docs/architecture

| File | Review | Notes |
| --- | --- | --- |
| [collection-provider-routing.md](../../architecture/collection-provider-routing.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |
| [collection-strategy.md](../../architecture/collection-strategy.md) | read-complete | Read complete strategy proposal. Secured claims, bureau reporting, cross-seller sanctions and licensing recommendations are not implemented-product or legal-approval evidence. External legal/provider assertions were not revalidated in this code audit. |
| [product-engineering-principles.md](../../architecture/product-engineering-principles.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |

## docs/audits

| File | Review | Notes |
| --- | --- | --- |
| [kredit-fixes-applied-2026-09-15.md](../kredit-fixes-applied-2026-09-15.md) | read-complete | Read complete historical fix report. Correctly retracts transitive-RLS false positives in companion audit. Backup nonzero-failure claim still misses placeholder credentials F004; current controls assessed independently. Report commands not executed. |
| [kredit-platform-audit-2026-09-15.md](../kredit-platform-audit-2026-09-15.md) | read-complete | Read complete historical audit and command examples, none executed. Its RLS 26-table allegation is explicitly corrected to 14 in companion fix report; do not repeat original list or blanket restrictive-policy fix. Several findings were fixed since revision; historical claims not imported as present findings. |

## docs/compliance

| File | Review | Notes |
| --- | --- | --- |
| [README.md](../../compliance/README.md) | read-complete | Read complete documentation; distinguished historical plans/check results from current implementation and this source-only audit. No documented command executed. |
| [data-inventory.md](../../compliance/data-inventory.md) | read-complete | Read complete documentation; distinguished historical plans/check results from current implementation and this source-only audit. No documented command executed. |
| [data-inventory.tsv](../../compliance/data-inventory.tsv) | structured-artifact-review | Reviewed all 1,424 schema/table/field classifications and unique policy annotations. No duplicate coordinates. Entire lawful-basis and retention columns remain pending legal approval; this is a draft catalog, not evidence of approved retention or actual encryption. Heuristic classifications mislabel job counts/provider names as identity and classify otp_challenges.code_hmac as commercial; opaque aggregate/input/payload fields require content-aware classification. No database/schema execution. |
| [dpia-trade-history-sharing.md](../../compliance/dpia-trade-history-sharing.md) | read-complete | Read full draft, proposed disclosure/consent/ageing controls and outstanding approvals. Clearly marked draft; recommendations are not implementation or legal sign-off. |
| [rls-permissive-baseline.txt](../../compliance/rls-permissive-baseline.txt) | read-complete | Read complete documented 14-table baseline; historical policy list is not treated as proof of current effective RLS. Final migration and caller tracing required. |
| [sub-processors.md](../../compliance/sub-processors.md) | read-complete | Read outbound processor register and assistant controls. Statements that named providers keep data entirely in Nigeria are assertions requiring contractual evidence, not facts established by code or host allowlisting. No legal/hosting verification performed. |

## docs/content

| File | Review | Notes |
| --- | --- | --- |
| [article-research-2026-09-04.csv](../../content/article-research-2026-09-04.csv) | read-complete | Read all 100 topic rows and 15 columns. Candidates, merge destinations and specialist validation gates are explicit; all volumes unmeasured. Cluster-level source leads are not substantiation of every proposed article; no fabricated traffic results. |
| [editorial-standard.md](../../content/editorial-standard.md) | read-complete | Read complete documentation; distinguished historical plans/check results from current implementation and this source-only audit. No documented command executed. |
| [search-research-2026-09-04.md](../../content/search-research-2026-09-04.md) | read-complete | Complete editorial research read, including source leads and explicit unmeasured demand/validation limitations. Historical code observations and external Google/provider/legal facts were not independently reverified; no publishing action authorized or performed. |
| [topic-backlog.md](../../content/topic-backlog.md) | read-complete | Read all unpublished topic ideas; slugs and titles are editorial backlog, not published article evidence. |

## docs

| File | Review | Notes |
| --- | --- | --- |
| [data-map.md](../../data-map.md) | read-complete | Reviewed storage/access/retention mapping and historical migration checkpoints. Supplier URL omission claim needs correction alongside F014; legal retention decisions remain explicitly pending. |

## docs/launch-audit-2026-09-08

| File | Review | Notes |
| --- | --- | --- |
| [ADMIN-CONNECTIONS.md](../../launch-audit-2026-09-08/ADMIN-CONNECTIONS.md) | read-complete | Historical launch checkpoint read in full; commands and verification results are historical, not executed or adopted in this audit. Sequential checkpoint updates supersede older entries. Provider-adapter and migration counts are obsolete for the current tree; current source reviewed independently. |
| [FILE-BY-FILE.md](../../launch-audit-2026-09-08/FILE-BY-FILE.md) | read-complete | Historical launch checkpoint read in full; commands and verification results are historical, not executed or adopted in this audit. Sequential checkpoint updates supersede older entries. Provider-adapter and migration counts are obsolete for the current tree; current source reviewed independently. |
| [PROGRESS.md](../../launch-audit-2026-09-08/PROGRESS.md) | read-complete | Historical launch checkpoint read in full; commands and verification results are historical, not executed or adopted in this audit. Sequential checkpoint updates supersede older entries. Provider-adapter and migration counts are obsolete for the current tree; current source reviewed independently. |
| [admin-connections-mobile-actions.png](../../launch-audit-2026-09-08/admin-connections-mobile-actions.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [admin-connections-mobile.png](../../launch-audit-2026-09-08/admin-connections-mobile.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [file-by-file-audit.json](../../launch-audit-2026-09-08/file-by-file-audit.json) | structured-artifact-review | Historical register: parsed records/schema, inspected status/method/verification distributions, path completeness/uniqueness, hash formats and provenance. Markdown inventory reconciled with JSON paths. Historical review notes are not current findings or fresh review of the named files; current project source read independently. No execution. |
| [file-review-index.json](../../launch-audit-2026-09-08/file-review-index.json) | structured-artifact-review | Historical register: parsed records/schema, inspected status/method/verification distributions, path completeness/uniqueness, hash formats and provenance. Markdown inventory reconciled with JSON paths. Historical review notes are not current findings or fresh review of the named files; current project source read independently. No execution. |
| [home-desktop.png](../../launch-audit-2026-09-08/home-desktop.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [home-mobile.png](../../launch-audit-2026-09-08/home-mobile.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [initial-browser.log](../../launch-audit-2026-09-08/initial-browser.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [signin-desktop.png](../../launch-audit-2026-09-08/signin-desktop.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |

## docs/launch-audit-2026-09-12

| File | Review | Notes |
| --- | --- | --- |
| [FILE-BY-FILE.md](../../launch-audit-2026-09-12/FILE-BY-FILE.md) | structured-artifact-review | Historical register: parsed records/schema, inspected status/method/verification distributions, path completeness/uniqueness, hash formats and provenance. Markdown inventory reconciled with JSON paths. Historical review notes are not current findings or fresh review of the named files; current project source read independently. No execution. |
| [FINDINGS.md](../../launch-audit-2026-09-12/FINDINGS.md) | read-complete | Read complete historical narrative. Findings describe the September 12 snapshot; later repair record supersedes repair status. Provider/advisory research is historical and not independently refreshed or adopted as current certification. Current implementation audited separately. |
| [FIXES.md](../../launch-audit-2026-09-12/FIXES.md) | read-complete | Read complete historical narrative. Findings describe the September 12 snapshot; later repair record supersedes repair status. Provider/advisory research is historical and not independently refreshed or adopted as current certification. Current implementation audited separately. |
| [PROVIDERS.md](../../launch-audit-2026-09-12/PROVIDERS.md) | read-complete | Read complete historical narrative. Findings describe the September 12 snapshot; later repair record supersedes repair status. Provider/advisory research is historical and not independently refreshed or adopted as current certification. Current implementation audited separately. |
| [README.md](../../launch-audit-2026-09-12/README.md) | read-complete | Read complete historical narrative. Findings describe the September 12 snapshot; later repair record supersedes repair status. Provider/advisory research is historical and not independently refreshed or adopted as current certification. Current implementation audited separately. |
| [file-review-index.json](../../launch-audit-2026-09-12/file-review-index.json) | structured-artifact-review | Historical register: parsed records/schema, inspected status/method/verification distributions, path completeness/uniqueness, hash formats and provenance. Markdown inventory reconciled with JSON paths. Historical review notes are not current findings or fresh review of the named files; current project source read independently. No execution. |
| [file-review.csv](../../launch-audit-2026-09-12/file-review.csv) | structured-artifact-review | Historical register: parsed records/schema, inspected status/method/verification distributions, path completeness/uniqueness, hash formats and provenance. Markdown inventory reconciled with JSON paths. Historical review notes are not current findings or fresh review of the named files; current project source read independently. No execution. |
| [findings.json](../../launch-audit-2026-09-12/findings.json) | structured-artifact-review | All 42 detail bodies match the fully read FINDINGS.md verbatim; unique IDs and fields checked. Historical open status differs from later repair-status.json by design, not evidence that these defects all persist today. |
| [repair-status.json](../../launch-audit-2026-09-12/repair-status.json) | structured-artifact-review | Historical register: parsed records/schema, inspected status/method/verification distributions, path completeness/uniqueness, hash formats and provenance. Markdown inventory reconciled with JSON paths. Historical review notes are not current findings or fresh review of the named files; current project source read independently. No execution. |
| [scope.json](../../launch-audit-2026-09-12/scope.json) | structured-artifact-review | Historical register: parsed records/schema, inspected status/method/verification distributions, path completeness/uniqueness, hash formats and provenance. Markdown inventory reconciled with JSON paths. Historical review notes are not current findings or fresh review of the named files; current project source read independently. No execution. |

## docs/launch-readiness

| File | Review | Notes |
| --- | --- | --- |
| [OPERATING-NOTES.md](../../launch-readiness/OPERATING-NOTES.md) | read-complete | Historical launch checkpoint read in full; commands and verification results are historical, not executed or adopted in this audit. Sequential checkpoint updates supersede older entries. Provider-adapter and migration counts are obsolete for the current tree; current source reviewed independently. |
| [RESUME.md](../../launch-readiness/RESUME.md) | read-complete | Historical launch checkpoint read in full; commands and verification results are historical, not executed or adopted in this audit. Sequential checkpoint updates supersede older entries. Provider-adapter and migration counts are obsolete for the current tree; current source reviewed independently. |
| [TEST-RESULTS.md](../../launch-readiness/TEST-RESULTS.md) | read-complete | Historical launch checkpoint read in full; commands and verification results are historical, not executed or adopted in this audit. Sequential checkpoint updates supersede older entries. Provider-adapter and migration counts are obsolete for the current tree; current source reviewed independently. |
| [baseline.txt](../../launch-readiness/baseline.txt) | read-complete | Historical launch checkpoint read in full; commands and verification results are historical, not executed or adopted in this audit. Sequential checkpoint updates supersede older entries. Provider-adapter and migration counts are obsolete for the current tree; current source reviewed independently. |
| [changed-files.txt](../../launch-readiness/changed-files.txt) | read-complete | Historical launch checkpoint read in full; commands and verification results are historical, not executed or adopted in this audit. Sequential checkpoint updates supersede older entries. Provider-adapter and migration counts are obsolete for the current tree; current source reviewed independently. |
| [changes.patch](../../launch-readiness/changes.patch) | structured-artifact-review | Historical unified patch: inspected provenance and complete file/hunk inventory (37 files, 76 hunks, 409 additions, 334 deletions). Not applied; not a separate line-by-line audit of obsolete implementations. Current counterparts already read independently. |
| [evidence-manifest.json](../../launch-readiness/evidence-manifest.json) | structured-artifact-review | All 16 path/hash records parsed and matched retained evidence bytes. Matching hashes prove file identity only, not validity of historical claims or current runtime behavior. |

## docs/launch-readiness/evidence

| File | Review | Notes |
| --- | --- | --- |
| [after-complaints.png](../../launch-readiness/evidence/after-complaints.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [after-home.png](../../launch-readiness/evidence/after-home.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [after-pricing.png](../../launch-readiness/evidence/after-pricing.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [after-privacy.png](../../launch-readiness/evidence/after-privacy.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [after-signin.png](../../launch-readiness/evidence/after-signin.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [after-terms-mobile-closed.png](../../launch-readiness/evidence/after-terms-mobile-closed.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [after-terms-mobile-open.png](../../launch-readiness/evidence/after-terms-mobile-open.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [after-terms.png](../../launch-readiness/evidence/after-terms.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [all-go-final.log](../../launch-readiness/evidence/all-go-final.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [all-go-unsandboxed.log](../../launch-readiness/evidence/all-go-unsandboxed.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [all-go.log](../../launch-readiness/evidence/all-go.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [api.log](../../launch-readiness/evidence/api.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [before-home.png](../../launch-readiness/evidence/before-home.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [before-pricing.png](../../launch-readiness/evidence/before-pricing.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [before-privacy.png](../../launch-readiness/evidence/before-privacy.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [before-signin.png](../../launch-readiness/evidence/before-signin.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [before-terms.png](../../launch-readiness/evidence/before-terms.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [browser-check.log](../../launch-readiness/evidence/browser-check.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [crypto-test.log](../../launch-readiness/evidence/crypto-test.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [db-roles.log](../../launch-readiness/evidence/db-roles.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [focused-go.log](../../launch-readiness/evidence/focused-go.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [migrations-final.log](../../launch-readiness/evidence/migrations-final.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [migrations.log](../../launch-readiness/evidence/migrations.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [node-preview.log](../../launch-readiness/evidence/node-preview.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [outage-check.log](../../launch-readiness/evidence/outage-check.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [outage-preview.log](../../launch-readiness/evidence/outage-preview.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [owner-api-integration.log](../../launch-readiness/evidence/owner-api-integration.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [owner-baseline-migrations.log](../../launch-readiness/evidence/owner-baseline-migrations.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [owner-fixed-migrations.log](../../launch-readiness/evidence/owner-fixed-migrations.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [owner-lifecycle-after.log](../../launch-readiness/evidence/owner-lifecycle-after.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [owner-lifecycle-before.log](../../launch-readiness/evidence/owner-lifecycle-before.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [postgres-init.log](../../launch-readiness/evidence/postgres-init.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [pricing-source-outage.png](../../launch-readiness/evidence/pricing-source-outage.png) | visual-review-complete | Individually opened historical screenshot. Inspected visible layout/content and recorded viewport state; long full-page evidence is scaled. No new rendering, interactive verification or current-browser assurance. Historical legal/domain/pricing content may differ from current source. |
| [public-playwright.log](../../launch-readiness/evidence/public-playwright.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [race.log](../../launch-readiness/evidence/race.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [release-api.log](../../launch-readiness/evidence/release-api.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [release-browser-check.log](../../launch-readiness/evidence/release-browser-check.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [release-capabilities.json](../../launch-readiness/evidence/release-capabilities.json) | read-complete | Read historical JSON response. Pricing 50/50 basis points and disabled capability flags are synthetic snapshot evidence, not proof of current production configuration. |
| [release-pricing.json](../../launch-readiness/evidence/release-pricing.json) | read-complete | Read historical JSON response. Pricing 50/50 basis points and disabled capability flags are synthetic snapshot evidence, not proof of current production configuration. |
| [settings-integration-final.log](../../launch-readiness/evidence/settings-integration-final.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [settings-integration.log](../../launch-readiness/evidence/settings-integration.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [web-build-final.log](../../launch-readiness/evidence/web-build-final.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [web-build.log](../../launch-readiness/evidence/web-build.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [web-check-final.log](../../launch-readiness/evidence/web-check-final.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |
| [web-check.log](../../launch-readiness/evidence/web-check.log) | structured-artifact-review | Historical log: parsed full text and reviewed line profiles, distinct diagnostic classes, first/final outcomes and provenance. Contains prior execution evidence only; no commands from the log executed. Earlier failures, environment errors and later success are separate observations, not current test results. |

## docs/launch-readiness

| File | Review | Notes |
| --- | --- | --- |
| [route-state-inventory.csv](../../launch-readiness/route-state-inventory.csv) | read-complete | Historical launch checkpoint read in full; commands and verification results are historical, not executed or adopted in this audit. Sequential checkpoint updates supersede older entries. Provider-adapter and migration counts are obsolete for the current tree; current source reviewed independently. |
| [settings-inventory.csv](../../launch-readiness/settings-inventory.csv) | read-complete | Historical launch checkpoint read in full; commands and verification results are historical, not executed or adopted in this audit. Sequential checkpoint updates supersede older entries. Provider-adapter and migration counts are obsolete for the current tree; current source reviewed independently. |
| [source-manifest.json](../../launch-readiness/source-manifest.json) | structured-artifact-review | All 808 path/hash records parsed; valid SHA256 formats. Many paths removed and 416 surviving files differ, establishing that the manifest is historical rather than current coverage. No prior test result adopted. |

## docs/operations

| File | Review | Notes |
| --- | --- | --- |
| [PRODUCTION-DEPLOYMENT-GUIDE.md](../../operations/PRODUCTION-DEPLOYMENT-GUIDE.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. Opening says API has no published host port, while later ingress section correctly describes loopback 8080 publication. Mono-only setup guidance predates native alternatives. |
| [admin-surface-enablement.md](../../operations/admin-surface-enablement.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |
| [observability.md](../../operations/observability.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |
| [persistence-migration.md](../../operations/persistence-migration.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |
| [provider-adapter.md](../../operations/provider-adapter.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |
| [provider-certification-plan.md](../../operations/provider-certification-plan.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. Opening claim that only Mono is implemented is obsolete after native Paystack/Flutterwave/Monnify additions. Shared tests/contract registry still lists only mock and Mono; native package checks are separate. |
| [readme-conformance.md](../../operations/readme-conformance.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |
| [readme-gap-audit.md](../../operations/readme-gap-audit.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |
| [rls-phase2-completion.md](../../operations/rls-phase2-completion.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. Documented -run TenantIsolation matches none of the test names in tests/integration/tenant_isolation_test.go, so that invocation does not run the referenced isolation scenarios. |
| [unregistered-traders.md](../../operations/unregistered-traders.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |

## docs/payments

| File | Review | Notes |
| --- | --- | --- |
| [native-collectors.md](../../payments/native-collectors.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. Cancellation-pauses-before-bank claim is undermined by preliminary remote lookup before local block (F016). Existing fixture claims are historical, not rerun. |

## docs/platform-improvements

| File | Review | Notes |
| --- | --- | --- |
| [CONSUMER-SALES.md](../../platform-improvements/CONSUMER-SALES.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |
| [DSA-REFERRALS.md](../../platform-improvements/DSA-REFERRALS.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |
| [ESSENTIAL-CHECKS.md](../../platform-improvements/ESSENTIAL-CHECKS.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |
| [IMPLEMENTATION.md](../../platform-improvements/IMPLEMENTATION.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |
| [PROVIDER-CONTRACTS.md](../../platform-improvements/PROVIDER-CONTRACTS.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. Earlier remaining-work paragraph contradicts later completed native identity/settlement/fee sections. Consolidate current contract guidance. |

## docs/product

| File | Review | Notes |
| --- | --- | --- |
| [analytics-event-catalog.md](../../product/analytics-event-catalog.md) | read-complete | Complete event/purpose/metadata/retention contract read. Seller membership claim has whitespace bypass F013; nullable empty metadata conflicts with object constraint F011. Catalog is not runtime evidence. |
| [interface-copy.md](../../product/interface-copy.md) | read-complete | Complete trust-copy contract read. Receipt-issue review/resolution promise exceeds implemented trade-line issue terminal path (F019). |
| [kredit-interface-standard.md](../../product/kredit-interface-standard.md) | read-complete | Complete read of interface, truthfulness, accessibility and financial-state requirements; missing-data guidance needs distinction between unknown active balance and legitimately unactivated terminal offers (F020). |
| [open-questions.md](../../product/open-questions.md) | read-complete | Complete 14-item decision register read. Draft provider/legal/retention positions explicitly lack approval; did not independently verify external law/contracts. Summary EXT-010 still permits silence in principle while current migration forbids it; body explains this requires future implementation. EXT-012 summary predates owner all-surfaces decision; fee defaults are not current runtime policy. |
| [pilot-kill-thresholds.md](../../product/pilot-kill-thresholds.md) | read-complete | Complete governance proposal read. Row 5 says above 10% but explanation says any nonzero forbidden silence activation is a tripwire; requires consistent threshold. Draft unsigned rows are not automated controls, and F009 currently blocks scorecard. |
| [pilot-scorecard.md](../../product/pilot-scorecard.md) | read-complete | Complete KPI definitions and reconciliation policy read. Says deemed acceptance is constrained by delivered notice, contradicting current prohibition; current endpoint fails on unsupported days_to_payment metric F009 and silence metric F010. |
| [public-launch-and-admin-controls.md](../../product/public-launch-and-admin-controls.md) | read-complete | Complete owner/solo-governance, hidden optional feature, runtime settings, secrets, publication, infrastructure and acceptance requirements read. Historical implementation sections must not be mistaken for current capability inventory. |
| [readme-completion-plan.md](../../product/readme-completion-plan.md) | read-complete | Complete six-wave historical requirements and completion claims read. Current implementation has drawdown expiry/issue recovery gaps F018/F019 and nonfunctional scorecard F009; old generated-client and 924-field claims are historical. |
| [readme-completion-traceability.md](../../product/readme-completion-traceability.md) | read-complete | Historical table read completely. Generated clients were removed; old field counts and passed test totals cannot certify current tree. Complete drawdown/analytics claims contradicted by F018/F019 and F009/F010. |
| [strategic-recommendations.md](../../product/strategic-recommendations.md) | read-complete | Historical recommendations read completely. Mono-only adapter statement is obsolete; deemed-acceptance discussion mixes prohibited current behavior with historical recommendations. Commercial thresholds are proposals, not measured outcomes. |
| [wave0-contracts.md](../../product/wave0-contracts.md) | read-complete | Complete planned states/permissions/events read. Drawdown receipt issue is specified without a resolution transition, matching F019. Planned contract vocabulary is not the current exhaustive API schema. |
| [workstream-evidence.tsv](../../product/workstream-evidence.tsv) | read-complete | Read every workstream row and linked-evidence list. Static complete flags reference historical tests and code paths; existing files are not proof of full workflow correctness. Analytics/drawdown rows need current findings attached. |
| [world-class-product-standard.md](../../product/world-class-product-standard.md) | read-complete | Complete aspirational quality table read. Dated implementation claims are historical, not present proof; current findings contradict complete-journey, analytics and error-recovery claims. |

## docs/release

| File | Review | Notes |
| --- | --- | --- |
| [certification-report.md](../../release/certification-report.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |
| [go-live-runbook.md](../../release/go-live-runbook.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |
| [readiness-checklist.md](../../release/readiness-checklist.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |
| [wave5-accessibility-evidence.md](../../release/wave5-accessibility-evidence.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. |
| [wave6-analytics-evidence.md](../../release/wave6-analytics-evidence.md) | read-complete | Read complete document; compared stated architecture, operating boundaries and evidence qualifications with reviewed source. Historical success claims are not verification of this working tree; referenced commands were not executed. Earlier analytics success claims do not cover current scorecard projection failure F009. |

## docs/runbooks

| File | Review | Notes |
| --- | --- | --- |
| [README.md](../../runbooks/README.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. |
| [admin-workflows.md](../../runbooks/admin-workflows.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. Deployment text pins obsolete schema 148; later appended sections contradict earlier claims that native settlement/billing remain unfinished. Consolidate current operating guidance. |
| [backup-restore.md](../../runbooks/backup-restore.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. |
| [break-glass-access.md](../../runbooks/break-glass-access.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. |
| [business-settings.md](../../runbooks/business-settings.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. Deployment text pins obsolete schema 153; repository now includes migration 154. |
| [dispute-adjustment.md](../../runbooks/dispute-adjustment.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. |
| [failed-jobs.md](../../runbooks/failed-jobs.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. |
| [failed-webhooks.md](../../runbooks/failed-webhooks.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. |
| [financial-operations.md](../../runbooks/financial-operations.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. |
| [mandate-cancellation.md](../../runbooks/mandate-cancellation.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. |
| [mono-sweep.md](../../runbooks/mono-sweep.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. Earlier notification section treats connector callback as delivery evidence, conflicting with current authoritative lookup design. Supplier authorization URL omission claim is affected by F014. Historical production-refusal statements and schema 153 need alignment with current configuration. |
| [operations-controls.md](../../runbooks/operations-controls.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. |
| [payment-reversal.md](../../runbooks/payment-reversal.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. |
| [phase5-production-assurance.md](../../runbooks/phase5-production-assurance.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. Historical branch/candidate document, including obsolete schema 128 reference; external evidence explicitly remains unverified. |
| [pilot-scorecard.md](../../runbooks/pilot-scorecard.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. |
| [production-pilot.md](../../runbooks/production-pilot.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. |
| [provider-reconciliation.md](../../runbooks/provider-reconciliation.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. |
| [provider-simulator.md](../../runbooks/provider-simulator.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. |
| [worker-operations.md](../../runbooks/worker-operations.md) | read-complete | Read complete procedure, prerequisites, authority, recovery steps and evidence limitations. Commands were reviewed only, not executed. |

## docs/security

| File | Review | Notes |
| --- | --- | --- |
| [production-security-checklist.md](../../security/production-security-checklist.md) | read-complete | Reviewed release checklist; unchecked evidence requirements are not proof of completed assurance. |

## docs/testing

| File | Review | Notes |
| --- | --- | --- |
| [admin-settings-2026-09-03.md](../../testing/admin-settings-2026-09-03.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. Independent-only approval, 18 fields and provider exclusion are historical; solo-owner/runtime provider controls have since been implemented. |
| [admin-workflows-2026-09-03.md](../../testing/admin-workflows-2026-09-03.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. Dated revisions, environments and pending external evidence must remain distinct from current source findings. |
| [audit-2026-09-06-implementation.md](../../testing/audit-2026-09-06-implementation.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. K-06 historical logout claim misses current 401/revocation ambiguity (F002). |
| [audit-2026-09-06-refinement.md](../../testing/audit-2026-09-06-refinement.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. Dated revisions, environments and pending external evidence must remain distinct from current source findings. |
| [audit-2026-09-06-verification.md](../../testing/audit-2026-09-06-verification.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. Dated revisions, environments and pending external evidence must remain distinct from current source findings. |
| [audit-2026-09-07-completion.md](../../testing/audit-2026-09-07-completion.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. Preserving mutation creation time as paid_at preserves replay identity but does not capture actual payment date (F021). |
| [code-audit-2026-09-03-files.csv](../../testing/code-audit-2026-09-03-files.csv) | structured-artifact-review | Historical audit ledger: parsed all 664 records, inspected schema, unique review/validation/status/finding annotations, row completeness, path uniqueness and hash format. Historical execution claims are not current verification; no tests run or underlying source review inferred from this ledger. |
| [code-audit-2026-09-03.md](../../testing/code-audit-2026-09-03.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. Dated revisions, environments and pending external evidence must remain distinct from current source findings. |
| [code-audit-2026-09-04-files.csv](../../testing/code-audit-2026-09-04-files.csv) | structured-artifact-review | Historical audit ledger: parsed all 687 records, inspected schema, unique review/validation/status/finding annotations, row completeness, path uniqueness and hash format. Historical execution claims are not current verification; no tests run or underlying source review inferred from this ledger. |
| [code-audit-2026-09-04.md](../../testing/code-audit-2026-09-04.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. Historical 22 findings and 100-article praise superseded by subsequent fixes/editorial rewrite. Temporary security-report link is not durable evidence. |
| [code-audit-fixes-2026-09-04.md](../../testing/code-audit-fixes-2026-09-04.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. A11 initially claims silence activation fixed, later K17 correctly disables it; read together with superseding policy. |
| [code-completion-2026-09-02.md](../../testing/code-completion-2026-09-02.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. Dated revisions, environments and pending external evidence must remain distinct from current source findings. |
| [code-review-2026-09-02-files.tsv](../../testing/code-review-2026-09-02-files.tsv) | structured-artifact-review | Historical audit ledger: parsed all 618 records, inspected schema, unique review/validation/status/finding annotations, row completeness, path uniqueness and hash format. Historical execution claims are not current verification; no tests run or underlying source review inferred from this ledger. |
| [code-review-2026-09-02.md](../../testing/code-review-2026-09-02.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. Dated revisions, environments and pending external evidence must remain distinct from current source findings. |
| [mono-sweep-evidence.md](../../testing/mono-sweep-evidence.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. Supplier URL removal assertion is false for durable list path F014; old schema/test counts and production prohibition are historical. |
| [phase4-provider-evidence.template.json](../../testing/phase4-provider-evidence.template.json) | read-complete | Read schema, adapter/source binding, contract-confirmation booleans and all 21 pending scenarios. Empty references and false approvals intentionally prevent this template from claiming certification. |
| [phase4-provider-verification.md](../../testing/phase4-provider-verification.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. Dated revisions, environments and pending external evidence must remain distinct from current source findings. |
| [phase6-hardening.md](../../testing/phase6-hardening.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. Dated revisions, environments and pending external evidence must remain distinct from current source findings. |
| [prelaunch-followup-2026-09-04.md](../../testing/prelaunch-followup-2026-09-04.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. Dated revisions, environments and pending external evidence must remain distinct from current source findings. |
| [solo-audit-2026-09-04-files.csv](../../testing/solo-audit-2026-09-04-files.csv) | structured-artifact-review | Historical audit ledger: parsed all 705 records, inspected schema, unique review/validation/status/finding annotations, row completeness, path uniqueness and hash format. Historical execution claims are not current verification; no tests run or underlying source review inferred from this ledger. |
| [solo-audit-2026-09-04.md](../../testing/solo-audit-2026-09-04.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. Dated revisions, environments and pending external evidence must remain distinct from current source findings. |
| [test-matrix.md](../../testing/test-matrix.md) | read-complete | Read full historical report/requirements, source references and verification limitations. No commands run and no historical pass claims adopted as current audit evidence. Matrix stops issue-at-receipt coverage at opening, with no resolution/return transition (F019). Seed warning is historical; current migration 071 exposure synchronization handles the seed payment. |

## docs

| File | Review | Notes |
| --- | --- | --- |
| [threat-model.md](../../threat-model.md) | read-complete | Reviewed assets, boundaries, attack controls and Mono assumptions. Supplier authorization-URL redaction claim is contradicted by list projection finding F014. |

## Root files

| File | Review | Notes |
| --- | --- | --- |
| [go.mod](../../../go.mod) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |
| [go.sum](../../../go.sum) | read-complete | Read all module/version/checksum records. Pinned River families and OpenTelemetry packages internally aligned; checksum strings are integrity metadata, not vulnerability evidence. No downloads or module verification executed. |

## infra/containers

| File | Review | Notes |
| --- | --- | --- |
| [.gitkeep](../../../infra/containers/.gitkeep) | read-complete | Empty directory placeholder containing only a newline; no code. |
| [Dockerfile.api](../../../infra/containers/Dockerfile.api) | read-complete | Read all: build stages, nonroot runtime, explicit simulator target and migration packaging. No confirmed defect in isolation. |
| [Dockerfile.web](../../../infra/containers/Dockerfile.web) | read-complete | Read all: frozen lock installation, Node adapter build, production dependency stage, nonroot runtime. Configuration and lockfile cross-check pending. |
| [README.md](../../../infra/containers/README.md) | read-complete | Read all. Reviewed deployment wiring, database grants, monitoring/privacy settings or configuration contract as applicable. F012 affects Terraform production web deployment. No scripts or rules were executed. |

## infra/environments

| File | Review | Notes |
| --- | --- | --- |
| [.gitkeep](../../../infra/environments/.gitkeep) | read-complete | Empty directory placeholder containing only a newline; no code. |
| [Caddyfile.prod](../../../infra/environments/Caddyfile.prod) | read-complete | Read all: proxy trust ranges, TLS, security headers and client IP forwarding. No confirmed defect without deployment configuration. |
| [README.md](../../../infra/environments/README.md) | read-complete | Read all. Reviewed deployment wiring, database grants, monitoring/privacy settings or configuration contract as applicable. F012 affects Terraform production web deployment. No scripts or rules were executed. |
| [docker-compose.prod.yml](../../../infra/environments/docker-compose.prod.yml) | read-complete | Read all: TLS database connections, role provisioning, service dependencies, loopback API publication and optional ingress. Healthcheck uses API liveness only. |
| [env.production.example](../../../infra/environments/env.production.example) | read-complete | Read all. Reviewed deployment wiring, database grants, monitoring/privacy settings or configuration contract as applicable. F012 affects Terraform production web deployment. No scripts or rules were executed. |
| [env.runtime.example](../../../infra/environments/env.runtime.example) | read-complete | Read all. Reviewed deployment wiring, database grants, monitoring/privacy settings or configuration contract as applicable. F012 affects Terraform production web deployment. No scripts or rules were executed. |
| [main.tf](../../../infra/environments/main.tf) | read-complete | Read all. Reviewed deployment wiring, database grants, monitoring/privacy settings or configuration contract as applicable. F012 affects Terraform production web deployment. No scripts or rules were executed. |
| [outputs.tf](../../../infra/environments/outputs.tf) | read-complete | Read all. Reviewed deployment wiring, database grants, monitoring/privacy settings or configuration contract as applicable. F012 affects Terraform production web deployment. No scripts or rules were executed. |
| [variables.tf](../../../infra/environments/variables.tf) | read-complete | Read all. Reviewed deployment wiring, database grants, monitoring/privacy settings or configuration contract as applicable. F012 affects Terraform production web deployment. No scripts or rules were executed. |
| [versions.tf](../../../infra/environments/versions.tf) | read-complete | Read all. Reviewed deployment wiring, database grants, monitoring/privacy settings or configuration contract as applicable. F012 affects Terraform production web deployment. No scripts or rules were executed. |

## infra/monitoring

| File | Review | Notes |
| --- | --- | --- |
| [.gitkeep](../../../infra/monitoring/.gitkeep) | read-complete | Empty directory placeholder containing only a newline; no code. |
| [README.md](../../../infra/monitoring/README.md) | read-complete | Read all. Reviewed deployment wiring, database grants, monitoring/privacy settings or configuration contract as applicable. F012 affects Terraform production web deployment. No scripts or rules were executed. |
| [otel-collector.yaml](../../../infra/monitoring/otel-collector.yaml) | read-complete | Read all. Reviewed deployment wiring, database grants, monitoring/privacy settings or configuration contract as applicable. F012 affects Terraform production web deployment. No scripts or rules were executed. |
| [phase5-rules.test.yaml](../../../infra/monitoring/phase5-rules.test.yaml) | read-complete | Read all. Reviewed deployment wiring, database grants, monitoring/privacy settings or configuration contract as applicable. F012 affects Terraform production web deployment. No scripts or rules were executed. |
| [prometheus-rules.yaml](../../../infra/monitoring/prometheus-rules.yaml) | read-complete | Read all. Reviewed deployment wiring, database grants, monitoring/privacy settings or configuration contract as applicable. F012 affects Terraform production web deployment. No scripts or rules were executed. |

## infra/postgres

| File | Review | Notes |
| --- | --- | --- |
| [development-logins.sql](../../../infra/postgres/development-logins.sql) | read-complete | Read all. Reviewed deployment wiring, database grants, monitoring/privacy settings or configuration contract as applicable. F012 affects Terraform production web deployment. No scripts or rules were executed. |
| [pg_hba.prod.conf](../../../infra/postgres/pg_hba.prod.conf) | read-complete | Read all: local administrative trust and TLS+SCRAM network connections; database not host-published by production Compose. |
| [roles.sql](../../../infra/postgres/roles.sql) | read-complete | Read all. Reviewed deployment wiring, database grants, monitoring/privacy settings or configuration contract as applicable. F012 affects Terraform production web deployment. No scripts or rules were executed. |

## internal/access

| File | Review | Notes |
| --- | --- | --- |
| [admin_roles_test.go](../../../internal/access/admin_roles_test.go) | read-complete | Test source only; not executed. Reviewed organization/platform permission assertions and separation of duties. |
| [authority.go](../../../internal/access/authority.go) | read-complete | Read all: transaction-scoped operator identity, lifecycle advisory lock, active-user/role checks, row locks and permission aggregation. No confirmed defect in this file. |
| [roles.go](../../../internal/access/roles.go) | read-complete | Read all: supplier/platform permission matrix, role validity and step-up policy. No confirmed defect in isolation; endpoint permission coverage remains pending. |
| [roles_test.go](../../../internal/access/roles_test.go) | read-complete | Test source only; not executed. Reviewed organization/platform permission assertions and separation of duties. |

## internal/agreementdocs

| File | Review | Notes |
| --- | --- | --- |
| [document.go](../../../internal/agreementdocs/document.go) | read-complete | Reviewed canonical hash checks, use of accepted terms, HTML escaping and printable current schedule/drawdown evidence. |
| [document_test.go](../../../internal/agreementdocs/document_test.go) | read-complete | Test source only; not executed. Reviewed canonical hash checks, use of accepted terms, HTML escaping and printable current schedule/drawdown evidence. |

## internal/audit

| File | Review | Notes |
| --- | --- | --- |
| [audit_postgres_test.go](../../../internal/audit/audit_postgres_test.go) | read-complete | Test source only; not executed. Reviewed append/history scope, metadata sanitization, persistence errors and transaction activity assertions. |
| [domain_activity_postgres_test.go](../../../internal/audit/domain_activity_postgres_test.go) | read-complete | Test source only; not executed. Reviewed append/history scope, metadata sanitization, persistence errors and transaction activity assertions. |
| [store.go](../../../internal/audit/store.go) | read-complete | Reviewed append/history scope, metadata sanitization, persistence errors and transaction activity assertions. |
| [store_test.go](../../../internal/audit/store_test.go) | read-complete | Test source only; not executed. Reviewed append/history scope, metadata sanitization, persistence errors and transaction activity assertions. |

## internal/auth

| File | Review | Notes |
| --- | --- | --- |
| [audit_postgres_test.go](../../../internal/auth/audit_postgres_test.go) | read-complete | Test source only; not executed. Read remaining authentication test source: OTP target binding, session rotation/expiry, MFA throttles/recovery and encrypted persisted targets. See F002 for logout error handling. |
| [fuzz_test.go](../../../internal/auth/fuzz_test.go) | read-complete | Test source only; not executed. Read remaining authentication test source: OTP target binding, session rotation/expiry, MFA throttles/recovery and encrypted persisted targets. See F002 for logout error handling. |
| [postgres.go](../../../internal/auth/postgres.go) | read-complete | Read all: database transactions, OTP row locking, MFA replay/lockout, encrypted fields, session rotation. Session lookup conflates infrastructure failure with invalid credentials; tracing caller impact. |
| [postgres_test.go](../../../internal/auth/postgres_test.go) | read-complete | Test source only; not executed. Read remaining authentication test source: OTP target binding, session rotation/expiry, MFA throttles/recovery and encrypted persisted targets. See F002 for logout error handling. |
| [store.go](../../../internal/auth/store.go) | read-complete | Read all: OTP purpose/target binding, cooldown, attempt caps, session idle/absolute expiry, TOTP replay and rotation. No confirmed defect in this pass. |
| [store_test.go](../../../internal/auth/store_test.go) | read-complete | Test source only; not executed. Read remaining authentication test source: OTP target binding, session rotation/expiry, MFA throttles/recovery and encrypted persisted targets. See F002 for logout error handling. |
| [target_test.go](../../../internal/auth/target_test.go) | read-complete | Test source only; not executed. Read remaining authentication test source: OTP target binding, session rotation/expiry, MFA throttles/recovery and encrypted persisted targets. See F002 for logout error handling. |

## internal/billing

| File | Review | Notes |
| --- | --- | --- |
| [authorized.go](../../../internal/billing/authorized.go) | read-complete | Read all. Reviewed separate seller-fee consent, immutable provider identity, invoice calculations, split fee allocation/reversal, bank evidence and reconciliation as applicable. No newly confirmed finding in this pass. Test source was read only, not executed. |
| [bank.go](../../../internal/billing/bank.go) | read-complete | Read all. Reviewed separate seller-fee consent, immutable provider identity, invoice calculations, split fee allocation/reversal, bank evidence and reconciliation as applicable. No newly confirmed finding in this pass. Test source was read only, not executed. |
| [connector.go](../../../internal/billing/connector.go) | read-complete | Read all. Reviewed separate seller-fee consent, immutable provider identity, invoice calculations, split fee allocation/reversal, bank evidence and reconciliation as applicable. No newly confirmed finding in this pass. Test source was read only, not executed. |
| [invoices.go](../../../internal/billing/invoices.go) | read-complete | Read all. Reviewed separate seller-fee consent, immutable provider identity, invoice calculations, split fee allocation/reversal, bank evidence and reconciliation as applicable. No newly confirmed finding in this pass. Test source was read only, not executed. |
| [invoices_test.go](../../../internal/billing/invoices_test.go) | read-complete | Read all. Reviewed separate seller-fee consent, immutable provider identity, invoice calculations, split fee allocation/reversal, bank evidence and reconciliation as applicable. No newly confirmed finding in this pass. Test source was read only, not executed. |
| [provider.go](../../../internal/billing/provider.go) | read-complete | Read all. Reviewed separate seller-fee consent, immutable provider identity, invoice calculations, split fee allocation/reversal, bank evidence and reconciliation as applicable. No newly confirmed finding in this pass. Test source was read only, not executed. |
| [splits.go](../../../internal/billing/splits.go) | read-complete | Read all. Reviewed separate seller-fee consent, immutable provider identity, invoice calculations, split fee allocation/reversal, bank evidence and reconciliation as applicable. No newly confirmed finding in this pass. Test source was read only, not executed. |

## internal/businesspolicy

| File | Review | Notes |
| --- | --- | --- |
| [policy.go](../../../internal/businesspolicy/policy.go) | read-complete | Reviewed complete policy validation, deployment ceilings, proposal locking/replay, independent decisions, effective dates and immutable history. |
| [policy_test.go](../../../internal/businesspolicy/policy_test.go) | read-complete | Test source only; not executed. Reviewed complete policy validation, deployment ceilings, proposal locking/replay, independent decisions, effective dates and immutable history. |
| [store.go](../../../internal/businesspolicy/store.go) | read-complete | Reviewed complete policy validation, deployment ceilings, proposal locking/replay, independent decisions, effective dates and immutable history. |
| [store_test.go](../../../internal/businesspolicy/store_test.go) | read-complete | Test source only; not executed. Reviewed complete policy validation, deployment ceilings, proposal locking/replay, independent decisions, effective dates and immutable history. |

## internal/buyers

| File | Review | Notes |
| --- | --- | --- |
| [audit_postgres_test.go](../../../internal/buyers/audit_postgres_test.go) | read-complete | Test source only; not executed. Reviewed invitation acceptance, encrypted targets, identity reuse, tenant-scoped portal reads, verification send fences and recovery/version checks. Confirmed stale fixture: AcceptInput at lines 56 and 65 omits ConsentsAccepted, TermsVersion, PrivacyVersion and IdentityNoticeVersion; PostgresStore.Accept rejects it before identity reuse assertions. Update fixture to current consent contract. Source inference only. |
| [postgres.go](../../../internal/buyers/postgres.go) | read-complete | Reviewed invitation acceptance, encrypted targets, identity reuse, tenant-scoped portal reads, verification send fences and recovery/version checks. |
| [postgres_test.go](../../../internal/buyers/postgres_test.go) | read-complete | Test source only; not executed. Reviewed invitation acceptance, encrypted targets, identity reuse, tenant-scoped portal reads, verification send fences and recovery/version checks. |
| [store.go](../../../internal/buyers/store.go) | read-complete | Reviewed invitation acceptance, encrypted targets, identity reuse, tenant-scoped portal reads, verification send fences and recovery/version checks. |
| [store_test.go](../../../internal/buyers/store_test.go) | read-complete | Test source only; not executed. Reviewed invitation acceptance, encrypted targets, identity reuse, tenant-scoped portal reads, verification send fences and recovery/version checks. |
| [verification.go](../../../internal/buyers/verification.go) | read-complete | Reviewed invitation acceptance, encrypted targets, identity reuse, tenant-scoped portal reads, verification send fences and recovery/version checks. |
| [verification_recovery.go](../../../internal/buyers/verification_recovery.go) | read-complete | Reviewed invitation acceptance, encrypted targets, identity reuse, tenant-scoped portal reads, verification send fences and recovery/version checks. |

## internal/collections

| File | Review | Notes |
| --- | --- | --- |
| [adapter.go](../../../internal/collections/adapter.go) | read-complete | Read complete: collection reservations, submission and reconciliation, retry eligibility, provider identity and routing, settlement freezing and persistence reviewed against callers and SQL guards. |
| [adapter_test.go](../../../internal/collections/adapter_test.go) | read-complete | Read complete as source only; reviewed scenarios and assertions for collection/routing/persistence behavior. Not executed. |
| [admin_workflows_postgres_test.go](../../../internal/collections/admin_workflows_postgres_test.go) | read-complete | Read complete as source only; reviewed scenarios and assertions for collection/routing/persistence behavior. Not executed. |
| [audit_regression_test.go](../../../internal/collections/audit_regression_test.go) | read-complete | Read complete as source only; reviewed scenarios and assertions for collection/routing/persistence behavior. Not executed. |
| [billing_postgres_test.go](../../../internal/collections/billing_postgres_test.go) | read-complete | Read complete as source only; reviewed scenarios and assertions for collection/routing/persistence behavior. Not executed. |
| [completion_postgres_test.go](../../../internal/collections/completion_postgres_test.go) | read-complete | Read complete as source only; reviewed scenarios and assertions for collection/routing/persistence behavior. Not executed. |
| [engine.go](../../../internal/collections/engine.go) | read-complete | Read complete: collection reservations, submission and reconciliation, retry eligibility, provider identity and routing, settlement freezing and persistence reviewed against callers and SQL guards. |
| [engine_test.go](../../../internal/collections/engine_test.go) | read-complete | Read complete as source only; reviewed scenarios and assertions for collection/routing/persistence behavior. Not executed. |
| [fee_launch_test.go](../../../internal/collections/fee_launch_test.go) | read-complete | Read complete as source only; reviewed scenarios and assertions for collection/routing/persistence behavior. Not executed. |
| [fee_policy_postgres_test.go](../../../internal/collections/fee_policy_postgres_test.go) | read-complete | Read complete as source only; reviewed scenarios and assertions for collection/routing/persistence behavior. Not executed. |
| [financial_audit_postgres_test.go](../../../internal/collections/financial_audit_postgres_test.go) | read-complete | Read complete as source only; reviewed scenarios and assertions for collection/routing/persistence behavior. Not executed. |
| [policy.go](../../../internal/collections/policy.go) | read-complete | Read complete: collection reservations, submission and reconciliation, retry eligibility, provider identity and routing, settlement freezing and persistence reviewed against callers and SQL guards. |
| [policy_postgres_test.go](../../../internal/collections/policy_postgres_test.go) | read-complete | Read complete as source only; reviewed scenarios and assertions for collection/routing/persistence behavior. Not executed. |
| [postgres.go](../../../internal/collections/postgres.go) | read-complete | Read complete: collection reservations, submission and reconciliation, retry eligibility, provider identity and routing, settlement freezing and persistence reviewed against callers and SQL guards. |
| [postgres_test.go](../../../internal/collections/postgres_test.go) | read-complete | Read complete as source only; reviewed scenarios and assertions for collection/routing/persistence behavior. Not executed. |
| [provider.go](../../../internal/collections/provider.go) | read-complete | Read complete: collection reservations, submission and reconciliation, retry eligibility, provider identity and routing, settlement freezing and persistence reviewed against callers and SQL guards. |
| [provider_contract_test.go](../../../internal/collections/provider_contract_test.go) | read-complete | Read complete as source only; reviewed scenarios and assertions for collection/routing/persistence behavior. Not executed. |
| [registry.go](../../../internal/collections/registry.go) | read-complete | Read complete: collection reservations, submission and reconciliation, retry eligibility, provider identity and routing, settlement freezing and persistence reviewed against callers and SQL guards. |
| [resilient.go](../../../internal/collections/resilient.go) | read-complete | Read complete: collection reservations, submission and reconciliation, retry eligibility, provider identity and routing, settlement freezing and persistence reviewed against callers and SQL guards. |
| [resilient_test.go](../../../internal/collections/resilient_test.go) | read-complete | Read complete as source only; reviewed scenarios and assertions for collection/routing/persistence behavior. Not executed. |
| [routing.go](../../../internal/collections/routing.go) | read-complete | Read complete: collection reservations, submission and reconciliation, retry eligibility, provider identity and routing, settlement freezing and persistence reviewed against callers and SQL guards. |
| [routing_test.go](../../../internal/collections/routing_test.go) | read-complete | Read complete as source only; reviewed scenarios and assertions for collection/routing/persistence behavior. Not executed. |
| [settlement_postgres_test.go](../../../internal/collections/settlement_postgres_test.go) | read-complete | Read complete as source only; reviewed scenarios and assertions for collection/routing/persistence behavior. Not executed. |
| [settlement_route.go](../../../internal/collections/settlement_route.go) | read-complete | Read complete: collection reservations, submission and reconciliation, retry eligibility, provider identity and routing, settlement freezing and persistence reviewed against callers and SQL guards. |
| [sweep_postgres_test.go](../../../internal/collections/sweep_postgres_test.go) | read-complete | Read complete as source only; reviewed scenarios and assertions for collection/routing/persistence behavior. Not executed. |
| [webhook_provider.go](../../../internal/collections/webhook_provider.go) | read-complete | Read complete: collection reservations, submission and reconciliation, retry eligibility, provider identity and routing, settlement freezing and persistence reviewed against callers and SQL guards. |

## internal/config

| File | Review | Notes |
| --- | --- | --- |
| [admin_connections.go](../../../internal/config/admin_connections.go) | read-complete | Read all stored-connection merge and validation paths, credential redaction/preservation and endpoint retarget checks. No confirmed defect in this pass. |
| [admin_connections_test.go](../../../internal/config/admin_connections_test.go) | read-complete | Read full existing test source only. Traced fixture setup, gate checks, expected outcomes and production call dependencies; no tests executed. |
| [config.go](../../../internal/config/config.go) | read-complete | Read all load/validation/helpers. Environment, feature gates, secrets, URLs, provider selections and bounds reviewed. No newly confirmed issue; saved-setting application still traced separately. |
| [config_test.go](../../../internal/config/config_test.go) | read-complete | Read full existing test source only. Traced fixture setup, gate checks, expected outcomes and production call dependencies; no tests executed. |
| [mono_production_test.go](../../../internal/config/mono_production_test.go) | read-complete | Read full existing test source only. Traced fixture setup, gate checks, expected outcomes and production call dependencies; no tests executed. |
| [native_collectors_test.go](../../../internal/config/native_collectors_test.go) | read-complete | Read full existing test source only. Traced fixture setup, gate checks, expected outcomes and production call dependencies; no tests executed. |
| [retained_providers.go](../../../internal/config/retained_providers.go) | read-complete | Read all retained collection/identity account validation, name uniqueness, active-account matching and credential requirements. External API contracts not verified. |
| [watch_connections.go](../../../internal/config/watch_connections.go) | read-complete | Read all polling, status persistence and graceful restart request conditions. No confirmed defect in this pass. |

## internal/consumer

| File | Review | Notes |
| --- | --- | --- |
| [actions.go](../../../internal/consumer/actions.go) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [model.go](../../../internal/consumer/model.go) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [model_test.go](../../../internal/consumer/model_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |
| [notices.go](../../../internal/consumer/notices.go) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [postgres_test.go](../../../internal/consumer/postgres_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |
| [store.go](../../../internal/consumer/store.go) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |

## internal/corrections

| File | Review | Notes |
| --- | --- | --- |
| [history.go](../../../internal/corrections/history.go) | read-complete | Reviewed request transitions, separate reviewer requirement, durable decision evidence and buyer/organization history. |
| [postgres.go](../../../internal/corrections/postgres.go) | read-complete | Reviewed request transitions, separate reviewer requirement, durable decision evidence and buyer/organization history. |
| [postgres_test.go](../../../internal/corrections/postgres_test.go) | read-complete | Read test source and assertions only; not executed. Reviewed request transitions, separate reviewer requirement, durable decision evidence and buyer/organization history. |
| [store.go](../../../internal/corrections/store.go) | read-complete | Reviewed request transitions, separate reviewer requirement, durable decision evidence and buyer/organization history. |
| [store_test.go](../../../internal/corrections/store_test.go) | read-complete | Read test source and assertions only; not executed. Reviewed request transitions, separate reviewer requirement, durable decision evidence and buyer/organization history. |

## internal/credit

| File | Review | Notes |
| --- | --- | --- |
| [agreement_bytes.go](../../../internal/credit/agreement_bytes.go) | read-complete | Read complete, without execution. Reviewed agreement serialization, state transitions, activation transactions, projection refresh and authorization. Supplier list redaction and restored mandate alias defects recorded as F014/F015; remaining cross-domain tracing continues. |
| [agreement_bytes_test.go](../../../internal/credit/agreement_bytes_test.go) | read-complete | Read complete, without execution. Reviewed agreement serialization, state transitions, activation transactions, projection refresh and authorization. Supplier list redaction and restored mandate alias defects recorded as F014/F015; remaining cross-domain tracing continues. |
| [calendar.go](../../../internal/credit/calendar.go) | read-complete | Read complete, without execution. Reviewed agreement serialization, state transitions, activation transactions, projection refresh and authorization. Supplier list redaction and restored mandate alias defects recorded as F014/F015; remaining cross-domain tracing continues. |
| [calendar_test.go](../../../internal/credit/calendar_test.go) | read-complete | Read complete, without execution. Reviewed agreement serialization, state transitions, activation transactions, projection refresh and authorization. Supplier list redaction and restored mandate alias defects recorded as F014/F015; remaining cross-domain tracing continues. |
| [postgres.go](../../../internal/credit/postgres.go) | read-complete | Read complete, without execution. Reviewed agreement serialization, state transitions, activation transactions, projection refresh and authorization. Supplier list redaction and restored mandate alias defects recorded as F014/F015; remaining cross-domain tracing continues. |
| [postgres_test.go](../../../internal/credit/postgres_test.go) | read-complete | Read complete, without execution. Reviewed agreement serialization, state transitions, activation transactions, projection refresh and authorization. Supplier list redaction and restored mandate alias defects recorded as F014/F015; remaining cross-domain tracing continues. |
| [reads.go](../../../internal/credit/reads.go) | read-complete | Read complete, without execution. Reviewed agreement serialization, state transitions, activation transactions, projection refresh and authorization. Supplier list redaction and restored mandate alias defects recorded as F014/F015; remaining cross-domain tracing continues. |
| [store.go](../../../internal/credit/store.go) | read-complete | Read complete, without execution. Reviewed agreement serialization, state transitions, activation transactions, projection refresh and authorization. Supplier list redaction and restored mandate alias defects recorded as F014/F015; remaining cross-domain tracing continues. |
| [store_test.go](../../../internal/credit/store_test.go) | read-complete | Read complete, without execution. Reviewed agreement serialization, state transitions, activation transactions, projection refresh and authorization. Supplier list redaction and restored mandate alias defects recorded as F014/F015; remaining cross-domain tracing continues. |
| [system_acceptance_postgres_test.go](../../../internal/credit/system_acceptance_postgres_test.go) | read-complete | Read complete, without execution. Reviewed agreement serialization, state transitions, activation transactions, projection refresh and authorization. Supplier list redaction and restored mandate alias defects recorded as F014/F015; remaining cross-domain tracing continues. |

## internal/db

| File | Review | Notes |
| --- | --- | --- |
| [financial_context.go](../../../internal/db/financial_context.go) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [notice_permissions_test.go](../../../internal/db/notice_permissions_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |
| [owner_permissions_test.go](../../../internal/db/owner_permissions_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |
| [persistence_contract.go](../../../internal/db/persistence_contract.go) | read-complete | Read all. Reviewed SQL/application contract, permissions, error propagation and reporting definitions. See F009/F010 for scorecard defects; other cross-file review ongoing. |
| [persistence_contract_test.go](../../../internal/db/persistence_contract_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |
| [postgres.go](../../../internal/db/postgres.go) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [postgres_test.go](../../../internal/db/postgres_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |
| [relationship_permissions_test.go](../../../internal/db/relationship_permissions_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |
| [schedule_adjustment.go](../../../internal/db/schedule_adjustment.go) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [tenant_context.go](../../../internal/db/tenant_context.go) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |

## internal/disputes

| File | Review | Notes |
| --- | --- | --- |
| [ledger_posting_test.go](../../../internal/disputes/ledger_posting_test.go) | read-complete | Read test source and assertions only; not executed. Reviewed contested-only collection holds, evidence and decision transitions, adjustment bounds and transactional ledger/schedule/balance changes. |
| [postgres.go](../../../internal/disputes/postgres.go) | read-complete | Reviewed contested-only collection holds, evidence and decision transitions, adjustment bounds and transactional ledger/schedule/balance changes. |
| [postgres_test.go](../../../internal/disputes/postgres_test.go) | read-complete | Read test source and assertions only; not executed. Reviewed contested-only collection holds, evidence and decision transitions, adjustment bounds and transactional ledger/schedule/balance changes. |
| [store.go](../../../internal/disputes/store.go) | read-complete | Reviewed contested-only collection holds, evidence and decision transitions, adjustment bounds and transactional ledger/schedule/balance changes. |
| [store_test.go](../../../internal/disputes/store_test.go) | read-complete | Read test source and assertions only; not executed. Reviewed contested-only collection holds, evidence and decision transitions, adjustment bounds and transactional ledger/schedule/balance changes. |

## internal/documents

| File | Review | Notes |
| --- | --- | --- |
| [cleanup.go](../../../internal/documents/cleanup.go) | read-complete | Read all. Traced ownership, immutable object keys, upload completion/hash validation, scan state and download gating. No additional confirmed defect in this pass; external scanner/storage behavior unverified. |
| [cleanup_postgres_test.go](../../../internal/documents/cleanup_postgres_test.go) | read-complete | Read full test source only, without execution. Reviewed synthetic object/scanner fixtures, assertions and conditional database paths; no runtime verification claimed. |
| [postgres_test.go](../../../internal/documents/postgres_test.go) | read-complete | Read full test source only, without execution. Reviewed synthetic object/scanner fixtures, assertions and conditional database paths; no runtime verification claimed. |
| [s3.go](../../../internal/documents/s3.go) | read-complete | Read all. Traced ownership, immutable object keys, upload completion/hash validation, scan state and download gating. No additional confirmed defect in this pass; external scanner/storage behavior unverified. |
| [s3_test.go](../../../internal/documents/s3_test.go) | read-complete | Read full test source only, without execution. Reviewed synthetic object/scanner fixtures, assertions and conditional database paths; no runtime verification claimed. |
| [scanner.go](../../../internal/documents/scanner.go) | read-complete | Read all. Traced ownership, immutable object keys, upload completion/hash validation, scan state and download gating. No additional confirmed defect in this pass; external scanner/storage behavior unverified. |
| [scanner_test.go](../../../internal/documents/scanner_test.go) | read-complete | Read full test source only, without execution. Reviewed synthetic object/scanner fixtures, assertions and conditional database paths; no runtime verification claimed. |
| [store.go](../../../internal/documents/store.go) | read-complete | Read all. Traced ownership, immutable object keys, upload completion/hash validation, scan state and download gating. No additional confirmed defect in this pass; external scanner/storage behavior unverified. |
| [store_test.go](../../../internal/documents/store_test.go) | read-complete | Read full test source only, without execution. Reviewed synthetic object/scanner fixtures, assertions and conditional database paths; no runtime verification claimed. |

## internal/feedback

| File | Review | Notes |
| --- | --- | --- |
| [postgres_test.go](../../../internal/feedback/postgres_test.go) | read-complete | Read all source only. Reviewed authority, input handling and persistence calls. Feedback store normalization participates in F013; other domain implementations still need cross-file tracing. Tests, where present, were read without execution. |
| [store.go](../../../internal/feedback/store.go) | read-complete | Read all source only. Reviewed authority, input handling and persistence calls. Feedback store normalization participates in F013; other domain implementations still need cross-file tracing. Tests, where present, were read without execution. |
| [store_test.go](../../../internal/feedback/store_test.go) | read-complete | Read all source only. Reviewed authority, input handling and persistence calls. Feedback store normalization participates in F013; other domain implementations still need cross-file tracing. Tests, where present, were read without execution. |

## internal/idempotency

| File | Review | Notes |
| --- | --- | --- |
| [postgres_test.go](../../../internal/idempotency/postgres_test.go) | read-complete | Test source only; not executed. Reviewed reserve/complete assertions, conflicting intent, expired/completed versus unfinished reservations and response isolation. |
| [store.go](../../../internal/idempotency/store.go) | read-complete | Read all: reserve/complete and memory/Postgres parity. Pending cross-file review of reservation recovery and session-dependent HTTP scopes. |
| [store_test.go](../../../internal/idempotency/store_test.go) | read-complete | Test source only; not executed. Reviewed reserve/complete assertions, conflicting intent, expired/completed versus unfinished reservations and response isolation. |

## internal/identifier

| File | Review | Notes |
| --- | --- | --- |
| [id.go](../../../internal/identifier/id.go) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [id_test.go](../../../internal/identifier/id_test.go) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |

## internal/identity

| File | Review | Notes |
| --- | --- | --- |
| [names.go](../../../internal/identity/names.go) | read-complete | Reviewed saved provider routing, safe identity evidence, native lookup consent/version fences, OTP expiry, manual review and evidence ownership. |
| [native.go](../../../internal/identity/native.go) | read-complete | Reviewed saved provider routing, safe identity evidence, native lookup consent/version fences, OTP expiry, manual review and evidence ownership. |
| [native_actions.go](../../../internal/identity/native_actions.go) | read-complete | Reviewed saved provider routing, safe identity evidence, native lookup consent/version fences, OTP expiry, manual review and evidence ownership. |
| [native_test.go](../../../internal/identity/native_test.go) | read-complete | Test source only; not executed. Reviewed saved provider routing, safe identity evidence, native lookup consent/version fences, OTP expiry, manual review and evidence ownership. |
| [provider.go](../../../internal/identity/provider.go) | read-complete | Reviewed saved provider routing, safe identity evidence, native lookup consent/version fences, OTP expiry, manual review and evidence ownership. |
| [provider_test.go](../../../internal/identity/provider_test.go) | read-complete | Test source only; not executed. Reviewed saved provider routing, safe identity evidence, native lookup consent/version fences, OTP expiry, manual review and evidence ownership. |
| [routing.go](../../../internal/identity/routing.go) | read-complete | Reviewed saved provider routing, safe identity evidence, native lookup consent/version fences, OTP expiry, manual review and evidence ownership. |
| [safe_result.go](../../../internal/identity/safe_result.go) | read-complete | Reviewed saved provider routing, safe identity evidence, native lookup consent/version fences, OTP expiry, manual review and evidence ownership. |
| [safe_result_test.go](../../../internal/identity/safe_result_test.go) | read-complete | Test source only; not executed. Reviewed saved provider routing, safe identity evidence, native lookup consent/version fences, OTP expiry, manual review and evidence ownership. |
| [webhook.go](../../../internal/identity/webhook.go) | read-complete | Reviewed saved provider routing, safe identity evidence, native lookup consent/version fences, OTP expiry, manual review and evidence ownership. |

## internal/jobs

| File | Review | Notes |
| --- | --- | --- |
| [client.go](../../../internal/jobs/client.go) | read-complete | Reviewed worker queue routing, operation dispatch, periodic uniqueness, transactional expiry, webhook identity/lease fencing and dead-letter handling. |
| [client_test.go](../../../internal/jobs/client_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |
| [collection_requeue_test.go](../../../internal/jobs/collection_requeue_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |
| [dead_letter.go](../../../internal/jobs/dead_letter.go) | read-complete | Reviewed worker queue routing, operation dispatch, periodic uniqueness, transactional expiry, webhook identity/lease fencing and dead-letter handling. |
| [provider_inbox_postgres_test.go](../../../internal/jobs/provider_inbox_postgres_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |
| [reconciliation_postgres_test.go](../../../internal/jobs/reconciliation_postgres_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |
| [telemetry_test.go](../../../internal/jobs/telemetry_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |

## internal/ledger

| File | Review | Notes |
| --- | --- | --- |
| [consumer.go](../../../internal/ledger/consumer.go) | read-complete | Read all: pre/post-release payments, returns, price reductions and refunds. Domain input guards remain to be reviewed. |
| [fee_terms.go](../../../internal/ledger/fee_terms.go) | read-complete | Read all: legacy fee defaults, bounded basis points, overflow-safe rate arithmetic and posting generation. No confirmed defect. |
| [fee_terms_test.go](../../../internal/ledger/fee_terms_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |
| [postgres.go](../../../internal/ledger/postgres.go) | read-complete | Read all: transaction insert/conflict, postings, outbox and financial posting helpers. F001: random UUID posting order breaks positional replay comparison. |
| [reconcile.go](../../../internal/ledger/reconcile.go) | read-complete | Read all: one-snapshot per-transaction balancing, includes empty headers. No confirmed defect. |
| [reconcile_test.go](../../../internal/ledger/reconcile_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |
| [referrals.go](../../../internal/ledger/referrals.go) | read-complete | Read all: signed commission corrections and payout postings. No confirmed defect with bounded referral inputs. |
| [store.go](../../../internal/ledger/store.go) | read-complete | Read all: balanced journal, checked sums, immutable copies, idempotency matching and ordered digest. Payment compensation replay interaction pending. |
| [store_test.go](../../../internal/ledger/store_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |

## internal/legalpublication

| File | Review | Notes |
| --- | --- | --- |
| [current.go](../../../internal/legalpublication/current.go) | read-complete | Reviewed published document integrity checks, default version selection and error propagation. |
| [current_test.go](../../../internal/legalpublication/current_test.go) | read-complete | Test source only; not executed. Reviewed published document integrity checks, default version selection and error propagation. |
| [versions.go](../../../internal/legalpublication/versions.go) | read-complete | Reviewed published document integrity checks, default version selection and error propagation. |

## internal/mandates

| File | Review | Notes |
| --- | --- | --- |
| [connector_contract_test.go](../../../internal/mandates/connector_contract_test.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |
| [paused.go](../../../internal/mandates/paused.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |
| [paused_test.go](../../../internal/mandates/paused_test.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |
| [provider.go](../../../internal/mandates/provider.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |
| [provider_test.go](../../../internal/mandates/provider_test.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |
| [reads.go](../../../internal/mandates/reads.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |
| [recovery.go](../../../internal/mandates/recovery.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |
| [routing.go](../../../internal/mandates/routing.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |
| [routing_test.go](../../../internal/mandates/routing_test.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |
| [sweep_postgres_test.go](../../../internal/mandates/sweep_postgres_test.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |
| [unavailable.go](../../../internal/mandates/unavailable.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |
| [webhook.go](../../../internal/mandates/webhook.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |

## internal/notifications

| File | Review | Notes |
| --- | --- | --- |
| [adapters.go](../../../internal/notifications/adapters.go) | read-complete | Read full source: notification adapter, routing, receipt identity and persistence behavior; existing assertions inspected where applicable. No test executed. |
| [configured.go](../../../internal/notifications/configured.go) | read-complete | Read entire source: live configuration fails closed, enabled gate precedes pinned route, event-specific status lookup. |
| [configured_test.go](../../../internal/notifications/configured_test.go) | read-complete | Read full source: notification adapter, routing, receipt identity and persistence behavior; existing assertions inspected where applicable. No test executed. |
| [mesaj.go](../../../internal/notifications/mesaj.go) | read-complete | Read full source: notification adapter, routing, receipt identity and persistence behavior; existing assertions inspected where applicable. No test executed. |
| [meta.go](../../../internal/notifications/meta.go) | read-complete | Read entire source: template/text construction, phone identity, callback HMAC, uncertain send handling. Media download trusts provider metadata URL and truncates at 16 MiB; follow caller handling during WhatsApp review. |
| [meta_receipts.go](../../../internal/notifications/meta_receipts.go) | read-complete | Read full source: notification adapter, routing, receipt identity and persistence behavior; existing assertions inspected where applicable. No test executed. |
| [otp_routing_test.go](../../../internal/notifications/otp_routing_test.go) | read-complete | Read full source: notification adapter, routing, receipt identity and persistence behavior; existing assertions inspected where applicable. No test executed. |
| [postgres_test.go](../../../internal/notifications/postgres_test.go) | read-complete | Read full source: notification adapter, routing, receipt identity and persistence behavior; existing assertions inspected where applicable. No test executed. |
| [receipts.go](../../../internal/notifications/receipts.go) | read-complete | Read full source: notification adapter, routing, receipt identity and persistence behavior; existing assertions inspected where applicable. No test executed. |
| [routes.go](../../../internal/notifications/routes.go) | read-complete | Read full source: notification adapter, routing, receipt identity and persistence behavior; existing assertions inspected where applicable. No test executed. |
| [sendly.go](../../../internal/notifications/sendly.go) | read-complete | Read full source: notification adapter, routing, receipt identity and persistence behavior; existing assertions inspected where applicable. No test executed. |
| [store.go](../../../internal/notifications/store.go) | read-complete | Read entire source: preferences and consent rechecks, event fingerprints, encrypted destinations, quiet-hour scheduling, lease ownership, retry exhaustion and secure-link refresh. Recovery call sites supply request-specific links; empty-link collision is not reachable there. |
| [store_test.go](../../../internal/notifications/store_test.go) | read-complete | Read full source: notification adapter, routing, receipt identity and persistence behavior; existing assertions inspected where applicable. No test executed. |
| [submission.go](../../../internal/notifications/submission.go) | read-complete | Read entire source: durable STARTED fence, payload HMAC, accepted replay, permanent rejection and unknown-outcome no-retransmit handling. |
| [webhook.go](../../../internal/notifications/webhook.go) | read-complete | Read full source: notification adapter, routing, receipt identity and persistence behavior; existing assertions inspected where applicable. No test executed. |
| [webhook_test.go](../../../internal/notifications/webhook_test.go) | read-complete | Read full source: notification adapter, routing, receipt identity and persistence behavior; existing assertions inspected where applicable. No test executed. |

## internal/observability

| File | Review | Notes |
| --- | --- | --- |
| [financial.go](../../../internal/observability/financial.go) | read-complete | Reviewed durable metric completeness checks, bounded duration samples, aggregate exposition and trace propagation. |
| [phase5_financial_integration_test.go](../../../internal/observability/phase5_financial_integration_test.go) | read-complete | Test source only; not executed. Reviewed durable metric completeness checks, bounded duration samples, aggregate exposition and trace propagation. |
| [phase5_financial_test.go](../../../internal/observability/phase5_financial_test.go) | read-complete | Test source only; not executed. Reviewed durable metric completeness checks, bounded duration samples, aggregate exposition and trace propagation. |
| [store.go](../../../internal/observability/store.go) | read-complete | Reviewed durable metric completeness checks, bounded duration samples, aggregate exposition and trace propagation. |
| [store_test.go](../../../internal/observability/store_test.go) | read-complete | Test source only; not executed. Reviewed durable metric completeness checks, bounded duration samples, aggregate exposition and trace propagation. |
| [tracer.go](../../../internal/observability/tracer.go) | read-complete | Reviewed durable metric completeness checks, bounded duration samples, aggregate exposition and trace propagation. |
| [tracer_test.go](../../../internal/observability/tracer_test.go) | read-complete | Test source only; not executed. Reviewed durable metric completeness checks, bounded duration samples, aggregate exposition and trace propagation. |

## internal/onboarding

| File | Review | Notes |
| --- | --- | --- |
| [postgres.go](../../../internal/onboarding/postgres.go) | read-complete | Reviewed readiness derivation, legal consent versions, KYB/settlement reference binding, billing approval, MFA synchronization and durable profile revisions. |
| [postgres_test.go](../../../internal/onboarding/postgres_test.go) | read-complete | Test source only; not executed. Reviewed readiness derivation, legal consent versions, KYB/settlement reference binding, billing approval, MFA synchronization and durable profile revisions. |
| [store.go](../../../internal/onboarding/store.go) | read-complete | Reviewed readiness derivation, legal consent versions, KYB/settlement reference binding, billing approval, MFA synchronization and durable profile revisions. |
| [store_test.go](../../../internal/onboarding/store_test.go) | read-complete | Test source only; not executed. Reviewed readiness derivation, legal consent versions, KYB/settlement reference binding, billing approval, MFA synchronization and durable profile revisions. |
| [trader_test.go](../../../internal/onboarding/trader_test.go) | read-complete | Test source only; not executed. Reviewed readiness derivation, legal consent versions, KYB/settlement reference binding, billing approval, MFA synchronization and durable profile revisions. |

## internal/operations

| File | Review | Notes |
| --- | --- | --- |
| [approvals.go](../../../internal/operations/approvals.go) | read-complete | Reviewed cumulative approval thresholds, independent approval and buyer consent, snapshot revalidation, schedule date changes, fee waivers and durable idempotency. |
| [ledger_posting_test.go](../../../internal/operations/ledger_posting_test.go) | read-complete | Read test source and assertions only; not executed. Reviewed cumulative approval thresholds, independent approval and buyer consent, snapshot revalidation, schedule date changes, fee waivers and durable idempotency. |
| [postgres.go](../../../internal/operations/postgres.go) | read-complete | Reviewed cumulative approval thresholds, independent approval and buyer consent, snapshot revalidation, schedule date changes, fee waivers and durable idempotency. |
| [postgres_test.go](../../../internal/operations/postgres_test.go) | read-complete | Read test source and assertions only; not executed. Reviewed cumulative approval thresholds, independent approval and buyer consent, snapshot revalidation, schedule date changes, fee waivers and durable idempotency. |
| [store.go](../../../internal/operations/store.go) | read-complete | Reviewed cumulative approval thresholds, independent approval and buyer consent, snapshot revalidation, schedule date changes, fee waivers and durable idempotency. |
| [store_test.go](../../../internal/operations/store_test.go) | read-complete | Read test source and assertions only; not executed. Reviewed cumulative approval thresholds, independent approval and buyer consent, snapshot revalidation, schedule date changes, fee waivers and durable idempotency. |

## internal/organizations

| File | Review | Notes |
| --- | --- | --- |
| [audit_postgres_test.go](../../../internal/organizations/audit_postgres_test.go) | read-complete | Test source only; not executed. Reviewed organization creation limits, initial onboarding atomicity, membership authority locking, owner protection and invitation lifecycle. |
| [invitation_lifecycle_test.go](../../../internal/organizations/invitation_lifecycle_test.go) | read-complete | Test source only; not executed. Reviewed organization creation limits, initial onboarding atomicity, membership authority locking, owner protection and invitation lifecycle. |
| [limit_postgres_test.go](../../../internal/organizations/limit_postgres_test.go) | read-complete | Test source only; not executed. Reviewed organization creation limits, initial onboarding atomicity, membership authority locking, owner protection and invitation lifecycle. |
| [postgres.go](../../../internal/organizations/postgres.go) | read-complete | Reviewed organization creation limits, initial onboarding atomicity, membership authority locking, owner protection and invitation lifecycle. |
| [postgres_test.go](../../../internal/organizations/postgres_test.go) | read-complete | Test source only; not executed. Reviewed organization creation limits, initial onboarding atomicity, membership authority locking, owner protection and invitation lifecycle. |
| [store.go](../../../internal/organizations/store.go) | read-complete | Reviewed organization creation limits, initial onboarding atomicity, membership authority locking, owner protection and invitation lifecycle. |
| [store_test.go](../../../internal/organizations/store_test.go) | read-complete | Test source only; not executed. Reviewed organization creation limits, initial onboarding atomicity, membership authority locking, owner protection and invitation lifecycle. |

## internal/outbox

| File | Review | Notes |
| --- | --- | --- |
| [dispatcher.go](../../../internal/outbox/dispatcher.go) | read-complete | Read all: durable publication, claim acknowledgement, bounded exponential delay and extended retry. No confirmed defect. |
| [dispatcher_postgres_test.go](../../../internal/outbox/dispatcher_postgres_test.go) | read-complete | Read only, not executed: committed publication, conflicting payload replay and stale publisher fencing assertions. |
| [dispatcher_test.go](../../../internal/outbox/dispatcher_test.go) | read-complete | Read only, not executed: backoff bounds assertions. |
| [store.go](../../../internal/outbox/store.go) | read-complete | Read all: semantic JSON conflict check, transaction append, SKIP LOCKED claims, expired claims and fenced completion. No confirmed defect. |

## internal/paymentclaims

| File | Review | Notes |
| --- | --- | --- |
| [postgres.go](../../../internal/paymentclaims/postgres.go) | read-complete | Reviewed claim intent matching, ownership scope, hold expiry and conservative errors, obligation locking and atomic payment confirmation. |
| [postgres_test.go](../../../internal/paymentclaims/postgres_test.go) | read-complete | Read test source and assertions only; not executed. Reviewed claim intent matching, ownership scope, hold expiry and conservative errors, obligation locking and atomic payment confirmation. |
| [store.go](../../../internal/paymentclaims/store.go) | read-complete | Reviewed claim intent matching, ownership scope, hold expiry and conservative errors, obligation locking and atomic payment confirmation. |
| [store_test.go](../../../internal/paymentclaims/store_test.go) | read-complete | Read test source and assertions only; not executed. Reviewed claim intent matching, ownership scope, hold expiry and conservative errors, obligation locking and atomic payment confirmation. |

## internal/payments

| File | Review | Notes |
| --- | --- | --- |
| [ledger_posting_test.go](../../../internal/payments/ledger_posting_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |
| [postgres.go](../../../internal/payments/postgres.go) | read-complete | Read all payment recording, allocation, reversal, rebuilding, context, snapshots, journal and outbox transaction paths. No newly confirmed defect; callers and billing remain under review. |
| [postgres_test.go](../../../internal/payments/postgres_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |
| [receipt.go](../../../internal/payments/receipt.go) | read-complete | Read all narrow public receipt projections. Signed-token requirement documented; full handler authorization tracing ongoing. |
| [receipt_postgres_test.go](../../../internal/payments/receipt_postgres_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |
| [store.go](../../../internal/payments/store.go) | read-complete | Read all: amount/currency/provenance/idempotency validation, allocation, compensation and reversal. Potential failed-operation retry divergence in development memory store; production adapter pending. |
| [store_test.go](../../../internal/payments/store_test.go) | read-complete | Read source assertions only; not executed. Reviewed scope, fixture assumptions and error/concurrency coverage. |

## internal/platform/logging

| File | Review | Notes |
| --- | --- | --- |
| [logging.go](../../../internal/platform/logging/logging.go) | read-complete | Read all: handler wrapping, attribute resolution, grouped redaction and sanitized errors. No confirmed defect in reviewed callers. |
| [sanitize.go](../../../internal/platform/logging/sanitize.go) | read-complete | Read all: credential redaction, sensitive route segments, metadata filtering. Cross-file metadata-key implementation pending. |
| [sanitize_test.go](../../../internal/platform/logging/sanitize_test.go) | read-complete | Test source only; not executed. Reviewed redaction assertions for URLs, secrets, opaque financial links and nested log values. |

## internal/platformops

| File | Review | Notes |
| --- | --- | --- |
| [controls.go](../../../internal/platformops/controls.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [controls_test.go](../../../internal/platformops/controls_test.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [directory_postgres_test.go](../../../internal/platformops/directory_postgres_test.go) | read-complete | Read full source only. Users now requires db.TenantFromContext but this fixture calls Users(t.Context()) without setting an actor, so enabled execution returns authenticated-operator-context error before its assertions. |
| [external_commands.go](../../../internal/platformops/external_commands.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [external_commands_test.go](../../../internal/platformops/external_commands_test.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [financial_review.go](../../../internal/platformops/financial_review.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [provider_work.go](../../../internal/platformops/provider_work.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [store.go](../../../internal/platformops/store.go) | read-complete | Read full source: admin read projections, actor context, directory pagination, role grants/revocation. RevokeRole lacks transactional authority recheck unlike GrantRole; trace handler/DB authority before treating as a finding. Owner lifecycle protected by migration 089. |

## internal/platformsettings

| File | Review | Notes |
| --- | --- | --- |
| [connector_test.go](../../../internal/platformsettings/connector_test.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [crypto.go](../../../internal/platformsettings/crypto.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [crypto_root_test.go](../../../internal/platformsettings/crypto_root_test.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [guide_catalog.json](../../../internal/platformsettings/guide_catalog.json) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [owner_lifecycle_integration_test.go](../../../internal/platformsettings/owner_lifecycle_integration_test.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [registry.go](../../../internal/platformsettings/registry.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [runtime_connections.go](../../../internal/platformsettings/runtime_connections.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [store.go](../../../internal/platformsettings/store.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [store_test.go](../../../internal/platformsettings/store_test.go) | read-complete | Read full source only. Integration connector fixture omits Adapter: connector; current email default is sendly, so custom endpoint/token fail validation before encrypted-storage assertions. No execution. |
| [website_content.go](../../../internal/platformsettings/website_content.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [website_guides.go](../../../internal/platformsettings/website_guides.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [website_guides_test.go](../../../internal/platformsettings/website_guides_test.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |

## internal/providers/bankdebit

| File | Review | Notes |
| --- | --- | --- |
| [enrollment.go](../../../internal/providers/bankdebit/enrollment.go) | read-complete | Read all: validation, encrypted bank details, durable pre-send fence, versioned confirmation, draft cancellation and identity routing. No confirmed defect in this pass. |
| [enrollment_integration_test.go](../../../internal/providers/bankdebit/enrollment_integration_test.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [money.go](../../../internal/providers/bankdebit/money.go) | read-complete | Read all: exact provider decimals via rational arithmetic. Negative Naira input is malformed; callers reviewed require positive amounts. |
| [money_test.go](../../../internal/providers/bankdebit/money_test.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [notice.go](../../../internal/providers/bankdebit/notice.go) | read-complete | Read all: normalized event and native-client contracts. |
| [recovery.go](../../../internal/providers/bankdebit/recovery.go) | read-complete | Read all: operator authorization, pending discovery, locked review and versioned recovery. F006: pending discovery RLS excludes permitted non-owner operators. |

## internal/providers/flutterwave

| File | Review | Notes |
| --- | --- | --- |
| [banks.go](../../../internal/providers/flutterwave/banks.go) | read-complete | Read all: hardcoded debit bank directory. Currency of provider-supported banks not verified. |
| [flutterwave.go](../../../internal/providers/flutterwave/flutterwave.go) | read-complete | Read all: tokenization fence, amount and identity checks, mandate state, charge reconciliation, notice authentication and recovery. External contract not verified. |
| [flutterwave_test.go](../../../internal/providers/flutterwave/flutterwave_test.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [settlement.go](../../../internal/providers/flutterwave/settlement.go) | read-complete | Read all: account resolution and subaccount response binding. External contract not verified. |

## internal/providers/monnify

| File | Review | Notes |
| --- | --- | --- |
| [banks.go](../../../internal/providers/monnify/banks.go) | read-complete | Read all: bank-directory request and shared validation. |
| [monnify.go](../../../internal/providers/monnify/monnify.go) | read-complete | Read all: auth-token cache, durable authorization, mandate identity/amount bounds, cancellation, debit verification and webhook HMAC. External contract not verified. |
| [monnify_test.go](../../../internal/providers/monnify/monnify_test.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [settlement.go](../../../internal/providers/monnify/settlement.go) | read-complete | Read all: destination validation, account/contract fingerprint and provider response identity. |

## internal/providers/mono

| File | Review | Notes |
| --- | --- | --- |
| [customer.go](../../../internal/providers/mono/customer.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [export_phase4_test.go](../../../internal/providers/mono/export_phase4_test.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [fee_billing.go](../../../internal/providers/mono/fee_billing.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [fee_billing_test.go](../../../internal/providers/mono/fee_billing_test.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [fuzz_test.go](../../../internal/providers/mono/fuzz_test.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [hosted_url.go](../../../internal/providers/mono/hosted_url.go) | read-complete | Read all: exact HTTPS host, no credentials/port and nonempty path. No confirmed defect. |
| [hosted_url_test.go](../../../internal/providers/mono/hosted_url_test.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [mono.go](../../../internal/providers/mono/mono.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [mono_test.go](../../../internal/providers/mono/mono_test.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [phase4_contract_test.go](../../../internal/providers/mono/phase4_contract_test.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [phase4_persistence_test.go](../../../internal/providers/mono/phase4_persistence_test.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [settlement.go](../../../internal/providers/mono/settlement.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [settlement_test.go](../../../internal/providers/mono/settlement_test.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [webhook.go](../../../internal/providers/mono/webhook.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |

## internal/providers/paystack

| File | Review | Notes |
| --- | --- | --- |
| [paystack.go](../../../internal/providers/paystack/paystack.go) | read-complete | Read all: adapter configuration, authorization, charge/reconciliation, amount/customer/mode binding, signed notices and operator recovery. External provider contract correctness not verified. |
| [paystack_test.go](../../../internal/providers/paystack/paystack_test.go) | read-complete | Read full source. Reviewed provider request/response identity, amount and environment binding, authorization lifecycle, fee isolation, settlement routing, and synthetic contract/persistence assertions as applicable. Test source inspected only; nothing executed. |
| [settlement.go](../../../internal/providers/paystack/settlement.go) | read-complete | Read all: bank listing, credential fingerprint, account resolution and subaccount result binding. External provider contract not verified. |

## internal/publictoken

| File | Review | Notes |
| --- | --- | --- |
| [fuzz_test.go](../../../internal/publictoken/fuzz_test.go) | read-complete | Test source only; not executed. Reviewed purpose, signature and expiry assertions and parser fuzz source. |
| [token.go](../../../internal/publictoken/token.go) | read-complete | Read all: purpose-bound HMAC, constant-time signature check, token length limit and exclusive expiry. No confirmed defect. |
| [token_test.go](../../../internal/publictoken/token_test.go) | read-complete | Test source only; not executed. Reviewed purpose, signature and expiry assertions and parser fuzz source. |

## internal/readiness

| File | Review | Notes |
| --- | --- | --- |
| [readiness.go](../../../internal/readiness/readiness.go) | read-complete | Reviewed production gates, evidence references, provider/channel requirements and positive pilot limits. |
| [readiness_test.go](../../../internal/readiness/readiness_test.go) | read-complete | Test source only; not executed. Reviewed production gates, evidence references, provider/channel requirements and positive pilot limits. |

## internal/referrals

| File | Review | Notes |
| --- | --- | --- |
| [postgres_test.go](../../../internal/referrals/postgres_test.go) | read-complete | Read full existing test source only. Traced fixture setup, gate checks, expected outcomes and production call dependencies; no tests executed. |
| [read.go](../../../internal/referrals/read.go) | read-complete | Read all: self/admin visibility, balances and cursor lists. Financial values decoded through generic maps; safe frontend range handling not reviewed. |
| [rewards.go](../../../internal/referrals/rewards.go) | read-complete | Read all: paged refresh, CAC deduplication, activation window, reward deltas, payout hold and completed-transfer recording. No confirmed defect in this pass. |
| [store.go](../../../internal/referrals/store.go) | read-complete | Read all: enrollment, bank-change guards, immutable attribution, owner controls and atomic audits. No confirmed defect in reviewed paths. |

## internal/relationships

| File | Review | Notes |
| --- | --- | --- |
| [postgres_test.go](../../../internal/relationships/postgres_test.go) | read-complete | Test source only; not executed. Reviewed relationship-scoped consent history, withdrawal after relationship ends, reminder selection and supplier directory scope. |
| [store.go](../../../internal/relationships/store.go) | read-complete | Reviewed relationship-scoped consent history, withdrawal after relationship ends, reminder selection and supplier directory scope. |
| [suppliers.go](../../../internal/relationships/suppliers.go) | read-complete | Reviewed relationship-scoped consent history, withdrawal after relationship ends, reminder selection and supplier directory scope. |
| [suppliers_test.go](../../../internal/relationships/suppliers_test.go) | read-complete | Test source only; not executed. Reviewed relationship-scoped consent history, withdrawal after relationship ends, reminder selection and supplier directory scope. |

## internal/reports

| File | Review | Notes |
| --- | --- | --- |
| [analytics.go](../../../internal/reports/analytics.go) | read-complete | Read all. Reviewed SQL/application contract, permissions, error propagation and reporting definitions. See F009/F010 for scorecard defects; other cross-file review ongoing. |
| [payment_metrics.go](../../../internal/reports/payment_metrics.go) | read-complete | Read all. Reviewed SQL/application contract, permissions, error propagation and reporting definitions. See F009/F010 for scorecard defects; other cross-file review ongoing. |
| [payment_metrics_postgres_test.go](../../../internal/reports/payment_metrics_postgres_test.go) | read-complete | Read full existing test source only. Traced fixture setup, gate checks, expected outcomes and production call dependencies; no tests executed. |
| [postgres_test.go](../../../internal/reports/postgres_test.go) | read-complete | Read full existing test source only. Traced fixture setup, gate checks, expected outcomes and production call dependencies; no tests executed. |
| [snapshot.go](../../../internal/reports/snapshot.go) | read-complete | Read all repeatable-read snapshot construction, tenant context and input validation. No newly confirmed defect. |
| [snapshot_postgres_test.go](../../../internal/reports/snapshot_postgres_test.go) | read-complete | Read full existing test source only. Traced fixture setup, gate checks, expected outcomes and production call dependencies; no tests executed. |
| [store.go](../../../internal/reports/store.go) | read-complete | Read all summaries, fees, history, CSV, analytics and instalment/timeliness calculations. F011: nil analytics metadata serialized as JSON null violates the database object constraint. |
| [store_test.go](../../../internal/reports/store_test.go) | read-complete | Read full existing test source only. Traced fixture setup, gate checks, expected outcomes and production call dependencies; no tests executed. |

## internal/schedules

| File | Review | Notes |
| --- | --- | --- |
| [adjustment.go](../../../internal/schedules/adjustment.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |
| [adjustment_test.go](../../../internal/schedules/adjustment_test.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |
| [fuzz_test.go](../../../internal/schedules/fuzz_test.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |
| [postgres_test.go](../../../internal/schedules/postgres_test.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |
| [store.go](../../../internal/schedules/store.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |
| [store_test.go](../../../internal/schedules/store_test.go) | read-complete | Read complete as source, without execution. Reviewed mandate routing, recovery and cancellation or schedule date generation, allocation and reversal. Schedule runtime role policies permit unscoped repository reads; cancellation outage impact remains under tracing. |

## internal/settlement

| File | Review | Notes |
| --- | --- | --- |
| [connector.go](../../../internal/settlement/connector.go) | read-complete | Reviewed destination validation, provider connection binding, durable send fencing, settlement receipt reconciliation, amount limits and ledger posting. |
| [provider.go](../../../internal/settlement/provider.go) | read-complete | Reviewed destination validation, provider connection binding, durable send fencing, settlement receipt reconciliation, amount limits and ledger posting. |
| [receipts.go](../../../internal/settlement/receipts.go) | read-complete | Reviewed destination validation, provider connection binding, durable send fencing, settlement receipt reconciliation, amount limits and ledger posting. |
| [store.go](../../../internal/settlement/store.go) | read-complete | Reviewed destination validation, provider connection binding, durable send fencing, settlement receipt reconciliation, amount limits and ledger posting. |

## internal/support

| File | Review | Notes |
| --- | --- | --- |
| [postgres_test.go](../../../internal/support/postgres_test.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [store.go](../../../internal/support/store.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [store_test.go](../../../internal/support/store_test.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |

## internal/tradelines

| File | Review | Notes |
| --- | --- | --- |
| [postgres.go](../../../internal/tradelines/postgres.go) | read-complete | Read entire source: immutable agreement hashes, capacity reservation, lifecycle transitions, atomic activation, durable aggregate loading and outbox identity. See F018/F019 for receipt lifecycle dead ends. |
| [postgres_test.go](../../../internal/tradelines/postgres_test.go) | read-complete | Read entire source: immutable agreement hashes, capacity reservation, lifecycle transitions, atomic activation, durable aggregate loading and outbox identity. See F018/F019 for receipt lifecycle dead ends. |
| [store.go](../../../internal/tradelines/store.go) | read-complete | Read entire source: immutable agreement hashes, capacity reservation, lifecycle transitions, atomic activation, durable aggregate loading and outbox identity. See F018/F019 for receipt lifecycle dead ends. |
| [store_test.go](../../../internal/tradelines/store_test.go) | read-complete | Read entire source: immutable agreement hashes, capacity reservation, lifecycle transitions, atomic activation, durable aggregate loading and outbox identity. See F018/F019 for receipt lifecycle dead ends. |

## internal/usercontrol

| File | Review | Notes |
| --- | --- | --- |
| [export.go](../../../internal/usercontrol/export.go) | read-complete | Read entire source: account recovery evidence, approval, cooling-off, session revocation, privacy decisions and subject-bound export columns; assertions inspected only. |
| [postgres_test.go](../../../internal/usercontrol/postgres_test.go) | read-complete | Read entire source: account recovery evidence, approval, cooling-off, session revocation, privacy decisions and subject-bound export columns; assertions inspected only. |
| [store.go](../../../internal/usercontrol/store.go) | read-complete | Read entire source: account recovery evidence, approval, cooling-off, session revocation, privacy decisions and subject-bound export columns; assertions inspected only. |
| [store_test.go](../../../internal/usercontrol/store_test.go) | read-complete | Read entire source: account recovery evidence, approval, cooling-off, session revocation, privacy decisions and subject-bound export columns; assertions inspected only. Source defect: delivery-failure fixture uses seven-character reason "verified", rejected by the eight-character minimum before the delivery callback; no execution. |

## internal/web

| File | Review | Notes |
| --- | --- | --- |
| [admin_insight_handlers.go](../../../internal/web/admin_insight_handlers.go) | read-complete | Read all. Reviewed SQL/application contract, permissions, error propagation and reporting definitions. See F009/F010 for scorecard defects; other cross-file review ongoing. |
| [admin_surfaces_test.go](../../../internal/web/admin_surfaces_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [admin_workflow_handlers.go](../../../internal/web/admin_workflow_handlers.go) | read-complete | Read complete. Reviewed authentication, role and MFA checks, CSRF, state handling and persistence/provider handoffs. Remaining domain tracing explicitly pending; no additional confirmed defect in isolation. |
| [audit_financial_regression_test.go](../../../internal/web/audit_financial_regression_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [auth_contract_test.go](../../../internal/web/auth_contract_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [auth_handlers.go](../../../internal/web/auth_handlers.go) | read-complete | Read all: public OTP, login/logout, MFA cookies, CSRF and origin checks. Investigating logout response on revocation storage failure. |
| [bank_authorization_handlers.go](../../../internal/web/bank_authorization_handlers.go) | read-complete | Read all: buyer mandate ownership, provider switch gating, bank details, native webhook enqueue and authoritative reconciliation. |
| [billing_handlers.go](../../../internal/web/billing_handlers.go) | read-complete | Read all. Invoice access, owner approval, fresh MFA/CSRF and receipt failure handling reviewed. Rollbacks use request context; domain implementations previously reviewed. |
| [business_policy_handlers.go](../../../internal/web/business_policy_handlers.go) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |
| [business_policy_handlers_test.go](../../../internal/web/business_policy_handlers_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [buyer_handlers.go](../../../internal/web/buyer_handlers.go) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |
| [client_ip_test.go](../../../internal/web/client_ip_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [collection_jobs.go](../../../internal/web/collection_jobs.go) | read-complete | Read complete. Reviewed request authorization, state transitions, runtime wiring and provider/persistence handoffs where applicable. Cross-file domain tracing remains in progress; no additional confirmed defect from this read. |
| [collection_notice_handlers.go](../../../internal/web/collection_notice_handlers.go) | read-complete | Read all. Buyer ownership, obligation lock, exact delivered notification evidence and acknowledgement dedupe reviewed. No additional confirmed defect. |
| [completion_handlers_test.go](../../../internal/web/completion_handlers_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [consumer_handlers.go](../../../internal/web/consumer_handlers.go) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [credit_handlers.go](../../../internal/web/credit_handlers.go) | read-complete | Read complete. Reviewed request authorization, state transitions, runtime wiring and provider/persistence handoffs where applicable. Cross-file domain tracing remains in progress; no additional confirmed defect from this read. |
| [credit_terms_handlers.go](../../../internal/web/credit_terms_handlers.go) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |
| [customer_access_test.go](../../../internal/web/customer_access_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [customer_registration_recovery.go](../../../internal/web/customer_registration_recovery.go) | read-complete | Read all. Provider-pinned lookup, identity HMAC match, authority lock, maturity and ownership recheck reviewed. Nil-database behavior needs runtime route context; not reported as production defect. |
| [dashboard_handlers.go](../../../internal/web/dashboard_handlers.go) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |
| [dispute_document_handlers.go](../../../internal/web/dispute_document_handlers.go) | read-complete | Read all. Traced ownership, immutable object keys, upload completion/hash validation, scan state and download gating. No additional confirmed defect in this pass; external scanner/storage behavior unverified. |
| [document_handlers.go](../../../internal/web/document_handlers.go) | read-complete | Read all. Traced ownership, immutable object keys, upload completion/hash validation, scan state and download gating. No additional confirmed defect in this pass; external scanner/storage behavior unverified. |
| [document_upload_slot_handlers.go](../../../internal/web/document_upload_slot_handlers.go) | read-complete | Read all. Traced ownership, immutable object keys, upload completion/hash validation, scan state and download gating. No additional confirmed defect in this pass; external scanner/storage behavior unverified. |
| [due_worklist.go](../../../internal/web/due_worklist.go) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |
| [fee_authorization_handlers.go](../../../internal/web/fee_authorization_handlers.go) | read-complete | Read all. Supplier/owner route authority, fresh MFA/CSRF and action dispatch reviewed. Pause/resume return empty authorization with saved flag; caller handling pending. |
| [fee_operations_handlers.go](../../../internal/web/fee_operations_handlers.go) | read-complete | Read all. Read role and owner-only reconciliation writes reviewed; service error mapping and action validation traced. |
| [feedback_handlers.go](../../../internal/web/feedback_handlers.go) | read-complete | Read all. F013: raw area membership check occurs before store whitespace normalization; authenticated cross-business feedback attribution is possible. |
| [financial_reads.go](../../../internal/web/financial_reads.go) | read-complete | Read all. Read adapter selection reviewed; payments prefer caller context, trade lines use context-bound service. Dispute legacy reads lack caller context; implementation tracing pending. |
| [financial_review_handlers.go](../../../internal/web/financial_review_handlers.go) | read-complete | Read all source only. Reviewed authority, input handling and persistence calls. Feedback store normalization participates in F013; other domain implementations still need cross-file tracing. Tests, where present, were read without execution. |
| [fuzz_test.go](../../../internal/web/fuzz_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [http_helpers.go](../../../internal/web/http_helpers.go) | read-complete | Read all: strict single-value JSON decoding, body limits and public-error redaction. No confirmed defect in isolation. |
| [idempotency_middleware_test.go](../../../internal/web/idempotency_middleware_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [mandate_handlers.go](../../../internal/web/mandate_handlers.go) | read-complete | Read complete. Reviewed authentication, role and MFA checks, CSRF, state handling and persistence/provider handoffs. Remaining domain tracing explicitly pending; no additional confirmed defect in isolation. |
| [mandate_recovery_handlers.go](../../../internal/web/mandate_recovery_handlers.go) | read-complete | Read all: operator permission, MFA/CSRF recovery and native attempt routing. Confirms F006. |
| [message_recovery_handlers.go](../../../internal/web/message_recovery_handlers.go) | read-complete | Read all. Operator access, age threshold, resolution atomicity and audit reviewed. Message submissions have no RLS so unscoped list does not reproduce mandate queue defect. |
| [meta_assistant_test.go](../../../internal/web/meta_assistant_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [meta_handlers.go](../../../internal/web/meta_handlers.go) | read-complete | Read complete. Reviewed request authorization, state transitions, runtime wiring and provider/persistence handoffs where applicable. Cross-file domain tracing remains in progress; no additional confirmed defect from this read. |
| [milestone1_test.go](../../../internal/web/milestone1_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [mono_handlers.go](../../../internal/web/mono_handlers.go) | read-complete | Read complete. Reviewed authentication, role and MFA checks, CSRF, state handling and persistence/provider handoffs. Remaining domain tracing explicitly pending; no additional confirmed defect in isolation. |
| [mono_runtime_test.go](../../../internal/web/mono_runtime_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [native_collector_test.go](../../../internal/web/native_collector_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [native_identity_documents.go](../../../internal/web/native_identity_documents.go) | read-complete | Read all. Traced ownership, immutable object keys, upload completion/hash validation, scan state and download gating. No additional confirmed defect in this pass; external scanner/storage behavior unverified. |
| [native_identity_handlers.go](../../../internal/web/native_identity_handlers.go) | read-complete | Read all source only. Reviewed authority, input handling and persistence calls. Feedback store normalization participates in F013; other domain implementations still need cross-file tracing. Tests, where present, were read without execution. |
| [notification_receipt_handlers.go](../../../internal/web/notification_receipt_handlers.go) | read-complete | Read all. Bounded bodies, connector resolution and HMAC validation reviewed. Callbacks intentionally do not establish delivery; worker lookup pending review. |
| [notification_recipients.go](../../../internal/web/notification_recipients.go) | read-complete | Read complete. Reviewed request authorization, state transitions, runtime wiring and provider/persistence handoffs where applicable. Cross-file domain tracing remains in progress; no additional confirmed defect from this read. |
| [onboarding_handlers.go](../../../internal/web/onboarding_handlers.go) | read-complete | Read complete. Reviewed authentication, role and MFA checks, CSRF, state handling and persistence/provider handoffs. Remaining domain tracing explicitly pending; no additional confirmed defect in isolation. |
| [optional_activity_test.go](../../../internal/web/optional_activity_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [organization_handlers.go](../../../internal/web/organization_handlers.go) | read-complete | Read complete. Reviewed authentication, role and MFA checks, CSRF, state handling and persistence/provider handoffs. Remaining domain tracing explicitly pending; no additional confirmed defect in isolation. |
| [outbox_notifications.go](../../../internal/web/outbox_notifications.go) | read-complete | Read complete. Reviewed durable notification enqueueing, recipient context and authenticated provider callback reconciliation. No additional confirmed finding in this file. |
| [outbox_notifications_test.go](../../../internal/web/outbox_notifications_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [ownership_transfer.go](../../../internal/web/ownership_transfer.go) | read-complete | Read complete. Reviewed authentication, role and MFA checks, CSRF, state handling and persistence/provider handoffs. Remaining domain tracing explicitly pending; no additional confirmed defect in isolation. |
| [ownership_transfer_postgres_test.go](../../../internal/web/ownership_transfer_postgres_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [payment_claim_handlers.go](../../../internal/web/payment_claim_handlers.go) | read-complete | Read all: supplier/buyer authorization, claim decisions, signed receipt/payment links. Durable claim internals and notification failure recovery pending. |
| [paystack_provider.go](../../../internal/web/paystack_provider.go) | read-complete | Read complete. Reviewed durable notification enqueueing, recipient context and authenticated provider callback reconciliation. No additional confirmed finding in this file. |
| [platform_operations_handlers.go](../../../internal/web/platform_operations_handlers.go) | read-complete | Read complete. Reviewed authentication, role and MFA checks, CSRF, state handling and persistence/provider handoffs. Remaining domain tracing explicitly pending; no additional confirmed defect in isolation. |
| [platform_operations_handlers_test.go](../../../internal/web/platform_operations_handlers_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [platform_settings_handlers.go](../../../internal/web/platform_settings_handlers.go) | read-complete | Read complete. Reviewed authentication, role and MFA checks, CSRF, state handling and persistence/provider handoffs. Remaining domain tracing explicitly pending; no additional confirmed defect in isolation. |
| [platform_settings_handlers_test.go](../../../internal/web/platform_settings_handlers_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [provider_readiness_test.go](../../../internal/web/provider_readiness_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [provider_work.go](../../../internal/web/provider_work.go) | read-complete | Read complete. Reviewed request authorization, state transitions, runtime wiring and provider/persistence handoffs where applicable. Cross-file domain tracing remains in progress; no additional confirmed defect from this read. |
| [referral_handlers.go](../../../internal/web/referral_handlers.go) | read-complete | Read all: public code lookup, cursor validation, owner operations, MFA and consent. No confirmed defect in reviewed paths. |
| [relationship_handlers.go](../../../internal/web/relationship_handlers.go) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |
| [reports_handlers.go](../../../internal/web/reports_handlers.go) | read-complete | Read all. Reviewed SQL/application contract, permissions, error propagation and reporting definitions. See F009/F010 for scorecard defects; other cross-file review ongoing. |
| [risk_hold_scope_test.go](../../../internal/web/risk_hold_scope_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [runtime.go](../../../internal/web/runtime.go) | read-complete | Read complete. Reviewed request authorization, state transitions, runtime wiring and provider/persistence handoffs where applicable. Cross-file domain tracing remains in progress; no additional confirmed defect from this read. |
| [runtime_schedule_test.go](../../../internal/web/runtime_schedule_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [runtime_tradeline_postgres_test.go](../../../internal/web/runtime_tradeline_postgres_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [secure_link_handlers.go](../../../internal/web/secure_link_handlers.go) | read-complete | Read all: signature expiry delegation and same-origin path allowlist, URL decoding, traversal/control/backslash checks reviewed. |
| [secure_link_handlers_test.go](../../../internal/web/secure_link_handlers_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [seller_settlement_handlers.go](../../../internal/web/seller_settlement_handlers.go) | read-complete | Read all source only. Reviewed authority, input handling and persistence calls. Feedback store normalization participates in F013; other domain implementations still need cross-file tracing. Tests, where present, were read without execution. |
| [server.go](../../../internal/web/server.go) | read-complete | Read all sections, including re-reading truncated route block. Idempotency scope/replay authorization, rate budgets, middleware, routes, observability and headers reviewed. Cross-domain replay effects remain open. |
| [server_test.go](../../../internal/web/server_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [settlement_handlers.go](../../../internal/web/settlement_handlers.go) | read-complete | Read all source only. Reviewed authority, input handling and persistence calls. Feedback store normalization participates in F013; other domain implementations still need cross-file tracing. Tests, where present, were read without execution. |
| [settlement_review_handlers.go](../../../internal/web/settlement_review_handlers.go) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |
| [setup_handlers.go](../../../internal/web/setup_handlers.go) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |
| [supplier_onboarding_test.go](../../../internal/web/supplier_onboarding_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [support_handlers.go](../../../internal/web/support_handlers.go) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |
| [trader_onboarding_test.go](../../../internal/web/trader_onboarding_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [user_control_handlers.go](../../../internal/web/user_control_handlers.go) | read-complete | Read complete. Reviewed authentication, role and MFA checks, CSRF, state handling and persistence/provider handoffs. Remaining domain tracing explicitly pending; no additional confirmed defect in isolation. |
| [user_control_handlers_test.go](../../../internal/web/user_control_handlers_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |
| [verification_recovery_handlers.go](../../../internal/web/verification_recovery_handlers.go) | read-complete | Read all. Operator permission, MFA/CSRF and recovery interface dispatch reviewed; buyer store tracing pending. |
| [website_content_handlers.go](../../../internal/web/website_content_handlers.go) | read-complete | Read complete source/configuration. Reviewed input and role boundaries, error handling and dependent calls where applicable. No additional confirmed defect in this pass; no commands or tests executed. |
| [website_content_handlers_test.go](../../../internal/web/website_content_handlers_test.go) | read-complete | Read complete as source only; not executed. Reviewed assertions, fixtures, authorization boundaries and failure coverage. Integration assertions are not runtime evidence. |

## internal/whatsapp

| File | Review | Notes |
| --- | --- | --- |
| [ai.go](../../../internal/whatsapp/ai.go) | read-complete | Read full source: sender budget, bounded audio, header-only provider key and structured result parsing. Prompt requests relative date calculation without providing the date; output is advisory only, to trace against UI behavior. No live model invocation. |
| [ai_test.go](../../../internal/whatsapp/ai_test.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [fuzz_test.go](../../../internal/whatsapp/fuzz_test.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [handler.go](../../../internal/whatsapp/handler.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [handler_test.go](../../../internal/whatsapp/handler_test.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |
| [postgres_test.go](../../../internal/whatsapp/postgres_test.go) | read-complete | Read entire source: validation, state transitions, authorization boundaries, persistence and error handling as applicable. Existing test source read without execution. |

## Root files

| File | Review | Notes |
| --- | --- | --- |
| [package.json](../../../package.json) | read-complete | Read all: package manager, engine and command mappings. No commands executed. |
| [pnpm-lock.yaml](../../../pnpm-lock.yaml) | structured-artifact-review | Read importer settings, overrides, all package/version/engine/peer/platform and snapshot dependency records. Inspected all 193 integrity records structurally (SHA-512 with 64-byte decoded digests); no custom resolution URL. Two esbuild versions correspond to Vite override and Vercel adapter. No installation, advisory scan or package-source audit. |
| [pnpm-workspace.yaml](../../../pnpm-workspace.yaml) | read-complete | Read all: workspace, esbuild build permission and dependency overrides. No advisory validation. |
| [redocly.yaml](../../../redocly.yaml) | read-complete | Read all: API lint configuration and warning/error severities. |

## scripts

| File | Review | Notes |
| --- | --- | --- |
| [api-lint.sh](../../../scripts/api-lint.sh) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [backup.sh](../../../scripts/backup.sh) | read-complete | Read all: restrictive umask, unique archive directory, ACL retention and checksum. No confirmed defect in isolation. |
| [bootstrap.sh](../../../scripts/bootstrap.sh) | read-complete | Read all: local tool/dependency/service/migration sequence. No execution. |
| [check-tools.sh](../../../scripts/check-tools.sh) | read-complete | Read all: required executable checks and version printouts. No execution. |
| [ci.sh](../../../scripts/ci.sh) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [configure-development-database.sh](../../../scripts/configure-development-database.sh) | read-complete | Read all: explicit development/test guard and role login initialization. No execution. |
| [content-audit.mjs](../../../scripts/content-audit.mjs) | read-complete | Read all source only. Static route/content scanners have text-matching limits; explicit downstream call-site review still required. Browser scripts were not executed. |
| [data-inventory-check.sh](../../../scripts/data-inventory-check.sh) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [data-inventory-generate.sh](../../../scripts/data-inventory-generate.sh) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [database-safety-test.sh](../../../scripts/database-safety-test.sh) | read-complete | Read full test source only; no tests executed. Reviewed fixtures and assertion scope; these synthetic cases do not establish runtime or provider correctness. |
| [db-check.sh](../../../scripts/db-check.sh) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [db-reset.sh](../../../scripts/db-reset.sh) | read-complete | Read all: explicit authorization/environment guard and destructive schema reset. No execution. |
| [db-rollback.sh](../../../scripts/db-rollback.sh) | read-complete | Read all: explicit authorization/environment guard and rollback command. No execution. |
| [dev.sh](../../../scripts/dev.sh) | read-complete | Read all: local services and background process cleanup. No execution. |
| [frontend-api-coverage.mjs](../../../scripts/frontend-api-coverage.mjs) | read-complete | Read all source only. Static route/content scanners have text-matching limits; explicit downstream call-site review still required. Browser scripts were not executed. |
| [fuzz.sh](../../../scripts/fuzz.sh) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [generate.sh](../../../scripts/generate.sh) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [implementation-plan-conformance-test.sh](../../../scripts/implementation-plan-conformance-test.sh) | read-complete | Read full test source only; no tests executed. Reviewed fixtures and assertion scope; these synthetic cases do not establish runtime or provider correctness. |
| [implementation-plan-conformance.sh](../../../scripts/implementation-plan-conformance.sh) | read-complete | Read all source without execution. Reviewed inputs, failure paths, bounds and command sequencing; no additional confirmed defect. |
| [launch-browser-check.mjs](../../../scripts/launch-browser-check.mjs) | read-complete | Read all source only. Static route/content scanners have text-matching limits; explicit downstream call-site review still required. Browser scripts were not executed. |
| [launch-outage-check.mjs](../../../scripts/launch-outage-check.mjs) | read-complete | Read all source only. Static route/content scanners have text-matching limits; explicit downstream call-site review still required. Browser scripts were not executed. |
| [lint.sh](../../../scripts/lint.sh) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [load-env-test.sh](../../../scripts/load-env-test.sh) | read-complete | Read full test source only; no tests executed. Reviewed fixtures and assertion scope; these synthetic cases do not establish runtime or provider correctness. |
| [load-env.sh](../../../scripts/load-env.sh) | read-complete | Read all: non-evaluating parser and caller environment precedence. No execution. |
| [load-smoke.sh](../../../scripts/load-smoke.sh) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [phase5-evidence-gate.sh](../../../scripts/phase5-evidence-gate.sh) | read-complete | Read all source without execution. Reviewed inputs, failure paths, bounds and command sequencing; no additional confirmed defect. |
| [phase5_load.py](../../../scripts/phase5_load.py) | read-complete | Read all source without execution. Reviewed inputs, failure paths, bounds and command sequencing; no additional confirmed defect. |
| [phase6-context-audit.py](../../../scripts/phase6-context-audit.py) | read-complete | Read all source without execution. Reviewed inputs, failure paths, bounds and command sequencing; no additional confirmed defect. |
| [phase6-governance-test.sh](../../../scripts/phase6-governance-test.sh) | read-complete | Read full test source only; no tests executed. Reviewed fixtures and assertion scope; these synthetic cases do not establish runtime or provider correctness. |
| [post-deploy-check.sh](../../../scripts/post-deploy-check.sh) | read-complete | Read all: public HTTPS origin, temporary responses, headers and publication checks. No requests executed. |
| [product-contract-sync.mjs](../../../scripts/product-contract-sync.mjs) | read-complete | Read all source only. Static route/content scanners have text-matching limits; explicit downstream call-site review still required. Browser scripts were not executed. |
| [readme-conformance.sh](../../../scripts/readme-conformance.sh) | read-complete | Read all source without execution. Reviewed inputs, failure paths, bounds and command sequencing; no additional confirmed defect. |
| [recovery_fingerprint.py](../../../scripts/recovery_fingerprint.py) | read-complete | Read all source without execution. Reviewed inputs, failure paths, bounds and command sequencing; no additional confirmed defect. |
| [release-certify.sh](../../../scripts/release-certify.sh) | read-complete | Read all source without execution. Reviewed inputs, failure paths, bounds and command sequencing; no additional confirmed defect. |
| [repository-audit.mjs](../../../scripts/repository-audit.mjs) | read-complete | Read all source only. Static route/content scanners have text-matching limits; explicit downstream call-site review still required. Browser scripts were not executed. |
| [restore-drill.sh](../../../scripts/restore-drill.sh) | read-complete | Read all: checksum/source fingerprint, empty-target guard, ACL-preserving restore and role checks. No commands executed. |
| [rls-policy-shape-check.sh](../../../scripts/rls-policy-shape-check.sh) | read-complete | Read all source without execution. Reviewed inputs, failure paths, bounds and command sequencing; no additional confirmed defect. |
| [security.sh](../../../scripts/security.sh) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [setup-vps.sh](../../../scripts/setup-vps.sh) | read-complete | Read all: root installation, firewall, directories and systemd composition. Optional ingress profile requires deployment-guide cross-check. |
| [sub-processor-check.sh](../../../scripts/sub-processor-check.sh) | read-complete | Read all source without execution. Reviewed inputs, failure paths, bounds and command sequencing; no additional confirmed defect. |
| [test-e2e.sh](../../../scripts/test-e2e.sh) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [test-integration.sh](../../../scripts/test-integration.sh) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |
| [test_phase5_backup.py](../../../scripts/test_phase5_backup.py) | read-complete | Read full test source only; no tests executed. Reviewed fixtures and assertion scope; these synthetic cases do not establish runtime or provider correctness. |
| [test_phase5_fingerprint.py](../../../scripts/test_phase5_fingerprint.py) | read-complete | Read full test source only; no tests executed. Reviewed fixtures and assertion scope; these synthetic cases do not establish runtime or provider correctness. |
| [test_phase5_load.py](../../../scripts/test_phase5_load.py) | read-complete | Read full test source only; no tests executed. Reviewed fixtures and assertion scope; these synthetic cases do not establish runtime or provider correctness. |
| [test_phase5_release_evidence.py](../../../scripts/test_phase5_release_evidence.py) | read-complete | Read full test source only; no tests executed. Reviewed fixtures and assertion scope; these synthetic cases do not establish runtime or provider correctness. |
| [test_provider_evidence.py](../../../scripts/test_provider_evidence.py) | read-complete | Read full test source only; no tests executed. Reviewed fixtures and assertion scope; these synthetic cases do not establish runtime or provider correctness. |
| [verify_backup.py](../../../scripts/verify_backup.py) | read-complete | Read all source without execution. Reviewed inputs, failure paths, bounds and command sequencing; no additional confirmed defect. |
| [verify_provider_evidence.py](../../../scripts/verify_provider_evidence.py) | read-complete | Read all source without execution. Reviewed inputs, failure paths, bounds and command sequencing; no additional confirmed defect. |
| [verify_release_evidence.py](../../../scripts/verify_release_evidence.py) | read-complete | Read all source without execution. Reviewed inputs, failure paths, bounds and command sequencing; no additional confirmed defect. |
| [web-test.sh](../../../scripts/web-test.sh) | read-complete | Read all source without execution: configuration, command sequencing, failure handling and environment assumptions reviewed. No additional confirmed defect in this file; downstream implementation review remains pending. |

## ssh

| File | Review | Notes |
| --- | --- | --- |
| id_ed25519 (`ssh/id_ed25519`, since removed) | metadata-only-secret-exclusion | Filename, size (444 bytes), permissions (0o600) and Git/Docker exclusion inspected. Contents deliberately not opened or copied into audit. Private keys are owner-only; .env is readable by other local users and public-key/known-host files are world-writable: tighten local permissions if this workstation has other users. No credential-use or exposure claim. |
| id_ed25519.pub (`ssh/id_ed25519.pub`, since removed) | metadata-only-secret-exclusion | Filename, size (95 bytes), permissions (0o666) and Git/Docker exclusion inspected. Contents deliberately not opened or copied into audit. Private keys are owner-only; .env is readable by other local users and public-key/known-host files are world-writable: tighten local permissions if this workstation has other users. No credential-use or exposure claim. |
| known_hosts (`ssh/known_hosts`, since removed) | metadata-only-secret-exclusion | Filename, size (568 bytes), permissions (0o666) and Git/Docker exclusion inspected. Contents deliberately not opened or copied into audit. Private keys are owner-only; .env is readable by other local users and public-key/known-host files are world-writable: tighten local permissions if this workstation has other users. No credential-use or exposure claim. |
| kredit_prod (`ssh/kredit_prod`, since removed) | metadata-only-secret-exclusion | Filename, size (464 bytes), permissions (0o600) and Git/Docker exclusion inspected. Contents deliberately not opened or copied into audit. Private keys are owner-only; .env is readable by other local users and public-key/known-host files are world-writable: tighten local permissions if this workstation has other users. No credential-use or exposure claim. |
| kredit_prod.pub (`ssh/kredit_prod.pub`, since removed) | metadata-only-secret-exclusion | Filename, size (100 bytes), permissions (0o666) and Git/Docker exclusion inspected. Contents deliberately not opened or copied into audit. Private keys are owner-only; .env is readable by other local users and public-key/known-host files are world-writable: tighten local permissions if this workstation has other users. No credential-use or exposure claim. |

## tests/contract

| File | Review | Notes |
| --- | --- | --- |
| [.gitkeep](../../../tests/contract/.gitkeep) | read-complete | Read complete acceptance/contract/integration/performance source or suite-location documentation; no execution. Synthetic/mocked scenarios do not establish live provider certification. |
| [provider_contract_test.go](../../../tests/contract/provider_contract_test.go) | read-complete | Read complete acceptance/contract/integration/performance source or suite-location documentation; no execution. Synthetic/mocked scenarios do not establish live provider certification. |

## tests/e2e

| File | Review | Notes |
| --- | --- | --- |
| [.gitkeep](../../../tests/e2e/.gitkeep) | read-complete | Read complete acceptance/contract/integration/performance source or suite-location documentation; no execution. Synthetic/mocked scenarios do not establish live provider certification. |
| [README.md](../../../tests/e2e/README.md) | read-complete | Read complete acceptance/contract/integration/performance source or suite-location documentation; no execution. Synthetic/mocked scenarios do not establish live provider certification. |

## tests/integration

| File | Review | Notes |
| --- | --- | --- |
| [persistence_test.go](../../../tests/integration/persistence_test.go) | read-complete | Read complete acceptance/contract/integration/performance source or suite-location documentation; no execution. Synthetic/mocked scenarios do not establish live provider certification. |
| [tenant_isolation_test.go](../../../tests/integration/tenant_isolation_test.go) | read-complete | Read complete acceptance/contract/integration/performance source or suite-location documentation; no execution. Synthetic/mocked scenarios do not establish live provider certification. |

## tests/load

| File | Review | Notes |
| --- | --- | --- |
| [smoke.js](../../../tests/load/smoke.js) | read-complete | Read complete acceptance/contract/integration/performance source or suite-location documentation; no execution. Synthetic/mocked scenarios do not establish live provider certification. |

## tests/performance

| File | Review | Notes |
| --- | --- | --- |
| [.gitkeep](../../../tests/performance/.gitkeep) | read-complete | Read complete acceptance/contract/integration/performance source or suite-location documentation; no execution. Synthetic/mocked scenarios do not establish live provider certification. |
| [acceptance.js](../../../tests/performance/acceptance.js) | read-complete | Read complete acceptance/contract/integration/performance source or suite-location documentation; no execution. Synthetic/mocked scenarios do not establish live provider certification. |
| [portfolio_queries.sql](../../../tests/performance/portfolio_queries.sql) | read-complete | Read complete acceptance/contract/integration/performance source or suite-location documentation; no execution. Synthetic/mocked scenarios do not establish live provider certification. |

## tests/provider-simulators

| File | Review | Notes |
| --- | --- | --- |
| [README.md](../../../tests/provider-simulators/README.md) | read-complete | Read complete acceptance/contract/integration/performance source or suite-location documentation; no execution. Synthetic/mocked scenarios do not establish live provider certification. |

## web

| File | Review | Notes |
| --- | --- | --- |
| [package.json](../../../web/package.json) | read-complete | Read all: scripts and exact/ranged dependencies. No install or advisory scan. |
| [playwright.config.ts](../../../web/playwright.config.ts) | read-complete | Reviewed browser test configuration and environment gating; no execution. |

## web/src

| File | Review | Notes |
| --- | --- | --- |
| [app.css](../../../web/src/app.css) | read-complete | Reviewed all global styles, component overrides, responsive rules, focus and reduced-motion handling; no visual/browser tests run. |
| [app.d.ts](../../../web/src/app.d.ts) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |
| [app.html](../../../web/src/app.html) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |
| [hooks.server.ts](../../../web/src/hooks.server.ts) | read-complete | Read all: same-origin API proxy, forwarding-header stripping/signing, request limits, deadline and private response cache rules. Deployment/client-IP configuration cross-check pending. |

## web/src/lib

| File | Review | Notes |
| --- | --- | --- |
| [account-context.ts](../../../web/src/lib/account-context.ts) | read-complete | Read all: shared account context identifier and readonly user identity. |
| [admin-client.ts](../../../web/src/lib/admin-client.ts) | read-complete | Read all: admin request deadline/body validation, per-call idempotency default, Lagos time formatting. Mutation callers pending. |

## web/src/lib/api

| File | Review | Notes |
| --- | --- | --- |
| [client.ts](../../../web/src/lib/api/client.ts) | read-complete | Read all: bounded read retries, CSRF headers, random keys and logout. Investigating accepting every 401 as confirmed logout. |
| [mutation.ts](../../../web/src/lib/api/mutation.ts) | read-complete | Read all: canonical request digest, retained intent, uncertain outcomes and 15-minute recovery block. Session rotation and storage-clear edge cases pending. |
| [onboarding-settings.ts](../../../web/src/lib/api/onboarding-settings.ts) | read-complete | Reviewed organization selection, settings version validation and permission payload. |
| [reliable.ts](../../../web/src/lib/api/reliable.ts) | read-complete | Read all: response decoders, deadlines, request invalidation, storage clearing and redirect validation. No confirmed defect in isolation. |

## web/src/lib

| File | Review | Notes |
| --- | --- | --- |
| [attention.ts](../../../web/src/lib/attention.ts) | read-complete | Reviewed dispute/payment/overdue/delivery priority ordering, customer naming and destination construction. |

## web/src/lib/blog

| File | Review | Notes |
| --- | --- | --- |
| [articles.ts](../../../web/src/lib/blog/articles.ts) | read-complete | Reviewed guide source/rendering, draft exclusion, categories, related links, metadata and safe text output. External citations were not fact-checked as part of this code audit. |
| [first-wave-drafts.js](../../../web/src/lib/blog/first-wave-drafts.js) | read-complete | Reviewed guide source/rendering, draft exclusion, categories, related links, metadata and safe text output. External citations were not fact-checked as part of this code audit. |
| [guides.js](../../../web/src/lib/blog/guides.js) | read-complete | Reviewed guide source/rendering, draft exclusion, categories, related links, metadata and safe text output. External citations were not fact-checked as part of this code audit. |

## web/src/lib/components

| File | Review | Notes |
| --- | --- | --- |
| [AdminAttention.svelte](../../../web/src/lib/components/AdminAttention.svelte) | read-complete | Reviewed attention load/error handling and API-supplied links. |
| [AuthGate.svelte](../../../web/src/lib/components/AuthGate.svelte) | read-complete | Reviewed account verification, request cancellation, context identity and 401 cleanup. |
| [BankReturnNotice.svelte](../../../web/src/lib/components/BankReturnNotice.svelte) | read-complete | Reviewed user-scoped storage and bounded return reference/expiration. |
| [CommandPalette.svelte](../../../web/src/lib/components/CommandPalette.svelte) | read-complete | Reviewed filtering, selection, modal keyboard handling and navigation. |
| [ConnectivityBanner.svelte](../../../web/src/lib/components/ConnectivityBanner.svelte) | read-complete | Reviewed connectivity messages, event listeners and timer cleanup. |
| [ConsumerPurchase.svelte](../../../web/src/lib/components/ConsumerPurchase.svelte) | read-complete | Reviewed acceptance, payment claims/receipts, delivery, returns, refunds and retry flow. Route wrappers key component by sale ID, preventing retained state across purchases. F008 applies to shared mutation helper. |
| [ConsumerSales.svelte](../../../web/src/lib/components/ConsumerSales.svelte) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [DSA.svelte](../../../web/src/lib/components/DSA.svelte) | read-complete | Reviewed enrollment/bank changes, reward rules, pagination, payout review and same-request retry. Shared consumer mutation caveat F008 applies. |
| [DisputeDetail.svelte](../../../web/src/lib/components/DisputeDetail.svelte) | read-complete | Reviewed scoped requests, evidence upload/scan checks, mutation intents and decision amount inputs. |
| [DocumentLayout.svelte](../../../web/src/lib/components/DocumentLayout.svelte) | read-complete | Reviewed legal metadata, contents anchors, observer cleanup and print presentation. |
| [DocumentUploader.svelte](../../../web/src/lib/components/DocumentUploader.svelte) | read-complete | Reviewed file selection callback; size/type/scan enforcement delegated to callers and server. |
| [FeeAuthorization.svelte](../../../web/src/lib/components/FeeAuthorization.svelte) | read-complete | Reviewed separate fee consent, sensitive input clearing, mandate lifecycle controls and HTTPS links. Organization changes have no stale-load guard and leave old items visible; needs scope tracing. |
| [FeeInvoices.svelte](../../../web/src/lib/components/FeeInvoices.svelte) | read-complete | Reviewed invoice money/date decoding, credits/refunds and stale-response generation guard. |
| [FeeOperations.svelte](../../../web/src/lib/components/FeeOperations.svelte) | read-complete | Reviewed fee-bank reconciliation inputs, amount/time validation and debit review actions. Generation suppresses stale loads, but previous items remain while organization changes; follow-up with parent scope. |
| [FeedbackPrompt.svelte](../../../web/src/lib/components/FeedbackPrompt.svelte) | read-complete | Reviewed user/organization local-storage scope, same-answer retry and stale-response guards. |
| HomeProof.svelte (`web/src/lib/components/HomeProof.svelte`, since removed) | read-complete | Reviewed static product assertions and links against implemented workflow. |
| [IdentityChecks.svelte](../../../web/src/lib/components/IdentityChecks.svelte) | read-complete | Reviewed identity actions, consent/OTP handling, document limits and review controls. |
| [Money.svelte](../../../web/src/lib/components/Money.svelte) | read-complete | Reviewed exact amount display and optional integer-naira compact presentation. |
| [MotionObserver.svelte](../../../web/src/lib/components/MotionObserver.svelte) | read-complete | Reviewed reduced-motion behavior, intersection observer and cleanup. |
| [NotificationHistory.svelte](../../../web/src/lib/components/NotificationHistory.svelte) | read-complete | Reviewed history decoding, status counts, ordering, filtering and distinction between sent and delivered. |
| [OwnerDialog.svelte](../../../web/src/lib/components/OwnerDialog.svelte) | read-complete | Reviewed modal lifecycle, focus restoration and busy-action protection. |
| [PaymentReview.svelte](../../../web/src/lib/components/PaymentReview.svelte) | read-complete | Reviewed receipt-check acknowledgment and busy-guarded confirmation dialog. |
| [PortalNav.svelte](../../../web/src/lib/components/PortalNav.svelte) | read-complete | Reviewed mobile/desktop navigation, modal focus handling and sign-out error retention. |
| [ProtectedActionDialog.svelte](../../../web/src/lib/components/ProtectedActionDialog.svelte) | read-complete | Reviewed reason requirement, preview-before-confirm sequence and locked reason after preview. |
| [PublishedLegalDocument.svelte](../../../web/src/lib/components/PublishedLegalDocument.svelte) | read-complete | Reviewed escaped publication rendering, section anchors and restricted linkification. |
| [RepaymentCustomer.svelte](../../../web/src/lib/components/RepaymentCustomer.svelte) | read-complete | Reviewed consent-gated registration, BVN clearing and result confirmation. |
| [ResourceNotice.svelte](../../../web/src/lib/components/ResourceNotice.svelte) | read-complete | Reviewed explicit load/error display and retry callback. |
| [RetainedAccounts.svelte](../../../web/src/lib/components/RetainedAccounts.svelte) | read-complete | Reviewed provider account editing, credential masking inputs, adapter options and retention warning. |
| [SaleCosts.svelte](../../../web/src/lib/components/SaleCosts.svelte) | read-complete | Reviewed pinned fee presentation, principal and agreed payment timing. |
| [SaleProgress.svelte](../../../web/src/lib/components/SaleProgress.svelte) | read-complete | Reviewed evidence timeline and next-step rendering. |
| SalesFlowNav.svelte (`web/src/lib/components/SalesFlowNav.svelte`, since removed) | read-complete | Reviewed buyer/seller flow links and route classification. |
| [SellerSettlements.svelte](../../../web/src/lib/components/SellerSettlements.svelte) | read-complete | Reviewed payout/return inputs, saved bank destination presentation and distinction from buyer payment. |
| [ShareActions.svelte](../../../web/src/lib/components/ShareActions.svelte) | read-complete | Reviewed explicit sharing controls, URL encoding and share/copy outcomes. |
| [SiteFooter.svelte](../../../web/src/lib/components/SiteFooter.svelte) | read-complete | Reviewed static navigation and date display. |
| [SiteHeader.svelte](../../../web/src/lib/components/SiteHeader.svelte) | read-complete | Reviewed static navigation, mobile disclosure and active links. |
| [Skeleton.svelte](../../../web/src/lib/components/Skeleton.svelte) | read-complete | Reviewed loading placeholders and reduced-motion styles. |
| [StatusPill.svelte](../../../web/src/lib/components/StatusPill.svelte) | read-complete | Reviewed status label formatting and class sanitization. |
| [SystemBanner.svelte](../../../web/src/lib/components/SystemBanner.svelte) | read-complete | Reviewed escaped message and tone class presentation. |
| [VerifyIdentity.svelte](../../../web/src/lib/components/VerifyIdentity.svelte) | read-complete | Reviewed TOTP input, enrollment check and AAL2 response verification. |
| [WorkspacePage.svelte](../../../web/src/lib/components/WorkspacePage.svelte) | read-complete | Reviewed organization selection, stale-request suppression, error states, search, pagination and row presentation. |

## web/src/lib

| File | Review | Notes |
| --- | --- | --- |
| [consumer.ts](../../../web/src/lib/consumer.ts) | read-complete | Read complete in source-only audit. Cross-file and final-schema conclusions remain in progress; see FINDINGS.md for confirmed issues. |
| [datetime.ts](../../../web/src/lib/datetime.ts) | read-complete | Read all: Lagos display and local browser datetime inputs. Caller interpretation pending. |
| [demo-sale.ts](../../../web/src/lib/demo-sale.ts) | read-complete | Reviewed explicit example labels and constant balance arithmetic. |
| [fee-terms.ts](../../../web/src/lib/fee-terms.ts) | read-complete | Read all: fee rate validation, integer floor calculation and cap at principal. No confirmed arithmetic mismatch with ledger/fee_terms.go. |
| [financial-copy.ts](../../../web/src/lib/financial-copy.ts) | read-complete | Read all: collection/dispute disclosure, hosted authorization URL allowlist and sale state copy. No confirmed defect in isolation. |
| [legal-defaults.json](../../../web/src/lib/legal-defaults.json) | read-complete | Read all bundled terms/privacy/complaints copy; checked product descriptions, version assumptions and internal links. This is source review, not legal advice or legal compliance certification. |
| [money.ts](../../../web/src/lib/money.ts) | read-complete | Read all: exact integer parsing, safe range handling, BigInt formatting and verbalization. No confirmed defect for supported int64 kobo range. |
| [product-language.ts](../../../web/src/lib/product-language.ts) | read-complete | Reviewed status/role/privacy vocabulary and fallback behavior. |
| [product-tools.ts](../../../web/src/lib/product-tools.ts) | read-complete | Reviewed optional local storage, recent sale items, sharing and display preference handling. Local JSON is cast without shape validation, so malformed saved items can break the optional recent-item helper. |
| [records.ts](../../../web/src/lib/records.ts) | read-complete | Reviewed money/date validation, sale projection, evidence timeline and list decoders. |
| [sale-drafts.ts](../../../web/src/lib/sale-drafts.ts) | read-complete | Read all: opt-in versioned draft keys, user/business scope, TTL, validation and deletion. No confirmed defect. |
| [sale-progress.ts](../../../web/src/lib/sale-progress.ts) | read-complete | Reviewed state-to-next-step mapping and settled-balance precedence. |
| [seo.ts](../../../web/src/lib/seo.ts) | read-complete | Reviewed public URL inventory, canonical origin, metadata and JSON-LD escaping. |

## web/src/lib/server

| File | Review | Notes |
| --- | --- | --- |
| [guides.ts](../../../web/src/lib/server/guides.ts) | read-complete | Reviewed publication overlay, copy decoding, fallbacks and related guide resolution. |
| [legal-config.ts](../../../web/src/lib/server/legal-config.ts) | read-complete | Read all. Reviewed deployment wiring, database grants, monitoring/privacy settings or configuration contract as applicable. F012 affects Terraform production web deployment. No scripts or rules were executed. |
| [legal-content.ts](../../../web/src/lib/server/legal-content.ts) | read-complete | Reviewed version-specific legal lookup, fail-closed fetch errors and initial publication fallback. |
| [legal-publication.ts](../../../web/src/lib/server/legal-publication.ts) | read-complete | Reviewed fixed initial legal publication metadata and version identifiers. |
| [website-content.ts](../../../web/src/lib/server/website-content.ts) | read-complete | Reviewed publication fetch timeout and static fallback behavior. |

## web/src/lib

| File | Review | Notes |
| --- | --- | --- |
| [trade-line-records.ts](../../../web/src/lib/trade-line-records.ts) | read-complete | Reviewed trade-line/drawdown/statement validation and line ownership matching. |
| [website-content.ts](../../../web/src/lib/website-content.ts) | read-complete | Reviewed copy/publication/record validation, guide links, contact fields and legal-page classification. |
| [website-defaults.json](../../../web/src/lib/website-defaults.json) | read-complete | Reviewed static marketing and FAQ copy against product flow; provider/legal claims not externally verified. |
| [website-editor-content.ts](../../../web/src/lib/website-editor-content.ts) | read-complete | Reviewed guide defaults and editor page map composition. |

## web/src/routes

| File | Review | Notes |
| --- | --- | --- |
| [+error.svelte](../../../web/src/routes/+error.svelte) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |
| [+layout.svelte](../../../web/src/routes/+layout.svelte) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |
| [+page.server.ts](../../../web/src/routes/+page.server.ts) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |
| [+page.svelte](../../../web/src/routes/+page.svelte) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |

## web/src/routes/admin

| File | Review | Notes |
| --- | --- | --- |
| [+layout.svelte](../../../web/src/routes/admin/+layout.svelte) | read-complete | Reviewed account gate, pathname-keyed subtree, navigation and sign-out integration; backend remains authority for platform roles. |
| [+page.svelte](../../../web/src/routes/admin/+page.svelte) | read-complete | Reviewed overview decoding, queue links and attention summary. |

## web/src/routes/admin/agents

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/agents/+page.svelte) | read-complete | Reviewed admin DSA wrapper. |

## web/src/routes/admin/analytics

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/analytics/+page.svelte) | read-complete | Reviewed scorecard decoding, date/org filters, reconciliation warnings and printable metrics. F009 prevents backend scorecard response; F010 affects source metric semantics. |

## web/src/routes/admin/approvals

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/approvals/+page.svelte) | read-complete | Reviewed governance/capability loads, immutable proposal IDs, amount/date inputs, self-approval versus second reviewer and buyer date acceptance. |

## web/src/routes/admin/attention

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/attention/+page.svelte) | read-complete | Reviewed expiring mandate, unclear debit and failed notice lists and control links. |

## web/src/routes/admin/audit

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/audit/+page.svelte) | read-complete | Reviewed organization scope, latest-response guard, local result search and event rendering. Historical navigation is limited by the single returned result set; no older-page control is present. |

## web/src/routes/admin/billing

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/billing/+page.svelte) | read-complete | Reviewed billing selection, approval/receipt intents and financial confirmation UI. |

## web/src/routes/admin/cases

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/cases/+page.svelte) | read-complete | Reviewed filtered case fetch, latest-request guards and detail links. |

## web/src/routes/admin/cases/[id]

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/cases/[id]/+page.svelte) | read-complete | Reviewed ID-bound case loading, scoped mutation, timeline and state update form. |

## web/src/routes/admin/consumer-sales

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/consumer-sales/+page.svelte) | read-complete | Reviewed admin consumer list wrapper. |

## web/src/routes/admin/consumer-sales/[saleID]

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/consumer-sales/[saleID]/+page.svelte) | read-complete | Reviewed keyed purchase component wrapper and route scope. |

## web/src/routes/admin/controls

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/controls/+page.svelte) | read-complete | Reviewed immutable impact preview payload, current-version binding, command result validation and mutation intent. |

## web/src/routes/admin/customer-registrations

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/customer-registrations/+page.svelte) | read-complete | Reviewed interrupted registration linking versus not-created resolution with evidence and mutation intent. |

## web/src/routes/admin/diagnostics

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/diagnostics/+page.svelte) | read-complete | Reviewed diagnostics response checks, loading/errors and operational metrics display. |

## web/src/routes/admin/disputes

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/disputes/+page.svelte) | read-complete | Reviewed dispute status filter, stale-response guard, amount display and detail links. |

## web/src/routes/admin/disputes/[id]

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/disputes/[id]/+page.svelte) | read-complete | Reviewed ID-bound load, exact amounts, evidence download, decision mutation and form outcomes. |

## web/src/routes/admin/history

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/history/+page.svelte) | read-complete | Reviewed audit history validation, filtering/pagination, CSV link and before/after money differences. |

## web/src/routes/admin/inbox

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/inbox/+page.svelte) | read-complete | Reviewed approval pagination, actor assignment, deadline conversion and review actions. |

## web/src/routes/admin/jobs

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/jobs/+page.svelte) | read-complete | Reviewed retry preview, expected attempt version and retained action idempotency key. |

## web/src/routes/admin/mandate-authorizations

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/mandate-authorizations/+page.svelte) | read-complete | Reviewed original mandate reference recovery and no-replacement guidance; F006 limits eligible operator queue reads. |

## web/src/routes/admin/message-submissions

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/message-submissions/+page.svelte) | read-complete | Reviewed uncertain-send resolution, explicit evidence acknowledgment and acceptance-versus-delivery distinction. |

## web/src/routes/admin/money

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/money/+page.svelte) | read-complete | Reviewed platform totals and recent activity presentation; amounts use shared exact-money formatter. |

## web/src/routes/admin/mono

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/mono/+page.svelte) | read-complete | Reviewed configured-versus-live readiness disclosure, provider flags and operational setup links. |

## web/src/routes/admin/organizations

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/organizations/+page.svelte) | read-complete | Reviewed organization search, stale-request guard, current-version checks and protected-control links. |

## web/src/routes/admin/platform-settings

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/platform-settings/+page.svelte) | read-complete | Read all 1098 lines. Reviewed secret/runtime configuration editing, expected versions, provider form branches, history guards, governance switch and ownership transfer. Default WhatsApp adapter selection omits Meta-specific fields; selecting explicit Meta reveals them. No provider contracts externally verified. |

## web/src/routes/admin/privacy

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/privacy/+page.svelte) | read-complete | Reviewed version-bound decisions/completion, recorded work disclosures and idempotency handling. |

## web/src/routes/admin/provider-events

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/provider-events/+page.svelte) | read-complete | Reviewed failed event replay, preview and expected-attempt version. |

## web/src/routes/admin/provider-work

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/provider-work/+page.svelte) | read-complete | Reviewed connection state versus live evidence, work filtering and recovery links. |

## web/src/routes/admin/reconciliation

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/reconciliation/+page.svelte) | read-complete | Reviewed exact discrepancy values, case history, claim/resolve actions and per-case mutation intents. |

## web/src/routes/admin/recovery

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/recovery/+page.svelte) | read-complete | Reviewed recovery queue, approve/reject intent and cooling-off disclosures. |

## web/src/routes/admin/search

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/search/+page.svelte) | read-complete | Reviewed reference validation, loading/error behavior and supported detail links. |

## web/src/routes/admin/seller-settlements

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/seller-settlements/+page.svelte) | read-complete | Reviewed admin settlement wrapper. |

## web/src/routes/admin/settings

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/settings/+page.svelte) | read-complete | Reviewed policy values/ranges, fee-unit scaling, impact preview, scheduled changes and governance-aware approvals. |

## web/src/routes/admin/settlement-review

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/settlement-review/+page.svelte) | read-complete | Reviewed destination ownership approval, original provider reference, evidence and uncertain-registration recovery. |

## web/src/routes/admin/setup

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/setup/+page.svelte) | read-complete | Reviewed configuration gates, API/worker freshness/version matching, polling cleanup and explicit live-operation distinction. |

## web/src/routes/admin/team

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/team/+page.svelte) | read-complete | Reviewed account search, role-effect preview, immutable grant payload, role revocation and request guards. |

## web/src/routes/admin/users

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/users/+page.svelte) | read-complete | Reviewed directory search, stale-request guard, version validation and control links. |

## web/src/routes/admin/verification-requests

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/verification-requests/+page.svelte) | read-complete | Reviewed verification reference recovery and embedded interactive identity review. |

## web/src/routes/admin/website

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/admin/website/+page.svelte) | read-complete | Reviewed saved draft/publish separation, immutable version checks, copy validation, reason capture and historical draft restore. |

## web/src/routes/agents

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/agents/+page.svelte) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |

## web/src/routes/app

| File | Review | Notes |
| --- | --- | --- |
| +layout.svelte (`web/src/routes/app/+layout.svelte`, since removed) | read-complete | Reviewed authenticated seller layout, navigation and pathname-keyed session gate. |
| +page.server.ts (`web/src/routes/app/+page.server.ts`, since removed) | read-complete | Reviewed cookie-presence login loader; authentication remains enforced by server endpoints. |
| +page.svelte (`web/src/routes/app/+page.svelte`, since removed) | read-complete | Reviewed OTP phone/email login, expiry, resend, device label and safe next-page handling. |

## web/src/routes/app/activity

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/activity/+page.svelte`, since removed) | read-complete | Reviewed partial resource failures, correction decisions, provider/readiness status and activity list. |

## web/src/routes/app/collections

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/collections/+page.svelte`, since removed) | read-complete | Reviewed bank collection list and sale links; backend deliberately returns request ID as row ID. |

## web/src/routes/app/consumer-sales

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/consumer-sales/+page.svelte`, since removed) | read-complete | Reviewed seller consumer-sales component wrapper. |

## web/src/routes/app/consumer-sales/[saleID]

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/consumer-sales/[saleID]/+page.svelte`, since removed) | read-complete | Reviewed keyed purchase component wrapper and route scope. |

## web/src/routes/app/credit

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/credit/+page.svelte`, since removed) | read-complete | Reviewed sale list, draft and detail mutations, customer scope, timing preview, invoice hashing, receipt release, payments, disputes and collection controls. F017 covers monthly schedule activation; F021 covers manual payment dates. |

## web/src/routes/app/credit/[id]

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/credit/[id]/+page.svelte`, since removed) | read-complete | Reviewed sale list, draft and detail mutations, customer scope, timing preview, invoice hashing, receipt release, payments, disputes and collection controls. F017 covers monthly schedule activation; F021 covers manual payment dates. |

## web/src/routes/app/credit/new

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/credit/new/+page.svelte`, since removed) | read-complete | Reviewed sale list, draft and detail mutations, customer scope, timing preview, invoice hashing, receipt release, payments, disputes and collection controls. F017 covers monthly schedule activation; F021 covers manual payment dates. |
| +page.ts (`web/src/routes/app/credit/new/+page.ts`, since removed) | read-complete | Reviewed sale list, draft and detail mutations, customer scope, timing preview, invoice hashing, receipt release, payments, disputes and collection controls. F017 covers monthly schedule activation; F021 covers manual payment dates. |

## web/src/routes/app/credit/quick

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/credit/quick/+page.svelte`, since removed) | read-complete | Reviewed sale list, draft and detail mutations, customer scope, timing preview, invoice hashing, receipt release, payments, disputes and collection controls. F017 covers monthly schedule activation; F021 covers manual payment dates. |

## web/src/routes/app/customers

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/customers/+page.svelte`, since removed) | read-complete | Reviewed customer selection, organization scoping, invitation mutation and customer history/statement display. New-customer page selects the first organization rather than the supplied organization query parameter. |

## web/src/routes/app/customers/[id]

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/customers/[id]/+page.svelte`, since removed) | read-complete | Reviewed customer selection, organization scoping, invitation mutation and customer history/statement display. New-customer page selects the first organization rather than the supplied organization query parameter. |

## web/src/routes/app/customers/new

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/customers/new/+page.svelte`, since removed) | read-complete | Reviewed customer selection, organization scoping, invitation mutation and customer history/statement display. New-customer page selects the first organization rather than the supplied organization query parameter. |

## web/src/routes/app/disputes

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/disputes/+page.svelte`, since removed) | read-complete | Reviewed dispute list/detail component wiring and required organization context. |

## web/src/routes/app/disputes/[id]

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/disputes/[id]/+page.svelte`, since removed) | read-complete | Reviewed dispute list/detail component wiring and required organization context. |

## web/src/routes/app/help

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/help/+page.svelte`, since removed) | read-complete | Reviewed scoped support-case creation and status transitions. |

## web/src/routes/app/notifications

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/notifications/+page.svelte`, since removed) | read-complete | Reviewed notification history component wrapper. |

## web/src/routes/app/onboarding

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/onboarding/+page.svelte`, since removed) | read-complete | Reviewed readiness consistency checks, scoped setup, contact challenges, consent versions and permission gates. |

## web/src/routes/app/overdue

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/overdue/+page.svelte`, since removed) | read-complete | Reviewed overdue list presentation and request navigation. |

## web/src/routes/app/overview

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/overview/+page.svelte`, since removed) | read-complete | Reviewed independent dashboard resource states, scoped totals, attention list and business creation; no runtime checks. |

## web/src/routes/app/payments

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/payments/+page.svelte`, since removed) | read-complete | Reviewed payment/claim totals, stable decision identities, confirmation modal and business-switch gating. F021 covers displayed transfer date. |

## web/src/routes/app/referral

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/referral/+page.svelte`, since removed) | read-complete | Reviewed referral lookup, consent, local referral storage and attribution mutation; shares F008 consumer mutation retry behavior. |

## web/src/routes/app/reports

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/reports/+page.svelte`, since removed) | read-complete | Reviewed exact-kobo sums, reporting decoders, Lagos period filtering, CSV download and request race guards. |

## web/src/routes/app/search

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/search/+page.svelte`, since removed) | read-complete | Reviewed organization-scoped parallel reads and bounded customer/sale/payment matching. |

## web/src/routes/app/settings

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/settings/+page.svelte`, since removed) | read-complete | Reviewed settings loading, permissions, input controls and mutations; sale defaults are saved here but creation pages use hardcoded values. |

## web/src/routes/app/settings/billing

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/settings/billing/+page.svelte`, since removed) | read-complete | Reviewed settings loading, permissions, input controls and mutations; sale defaults are saved here but creation pages use hardcoded values. |

## web/src/routes/app/settings/credit-policy

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/settings/credit-policy/+page.svelte`, since removed) | read-complete | Reviewed settings loading, permissions, input controls and mutations; sale defaults are saved here but creation pages use hardcoded values. |

## web/src/routes/app/settings/data

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/settings/data/+page.svelte`, since removed) | read-complete | Reviewed settings loading, permissions, input controls and mutations; sale defaults are saved here but creation pages use hardcoded values. |

## web/src/routes/app/settings/notifications

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/settings/notifications/+page.svelte`, since removed) | read-complete | Reviewed settings loading, permissions, input controls and mutations; sale defaults are saved here but creation pages use hardcoded values. |

## web/src/routes/app/settings/privacy

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/settings/privacy/+page.svelte`, since removed) | read-complete | Reviewed settings loading, permissions, input controls and mutations; sale defaults are saved here but creation pages use hardcoded values. |

## web/src/routes/app/settings/security

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/settings/security/+page.svelte`, since removed) | read-complete | Reviewed settings loading, permissions, input controls and mutations; sale defaults are saved here but creation pages use hardcoded values. |

## web/src/routes/app/settings/settlement

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/settings/settlement/+page.svelte`, since removed) | read-complete | Reviewed settings loading, permissions, input controls and mutations; sale defaults are saved here but creation pages use hardcoded values. |

## web/src/routes/app/team

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/team/+page.svelte`, since removed) | read-complete | Reviewed scoped membership reads and protected invite/role/status mutations. |

## web/src/routes/app/trade-lines

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/trade-lines/+page.svelte`, since removed) | read-complete | Reviewed capability gating, organization scope, reservation, release/cancel, limit changes and unresolved mutation handling. F018/F019 cover backend receipt lifecycle gaps. |

## web/src/routes/app/trade-lines/[id]

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/app/trade-lines/[id]/+page.svelte`, since removed) | read-complete | Reviewed capability gating, organization scope, reservation, release/cancel, limit changes and unresolved mutation handling. F018/F019 cover backend receipt lifecycle gaps. |

## web/src/routes/blog

| File | Review | Notes |
| --- | --- | --- |
| [+layout.svelte](../../../web/src/routes/blog/+layout.svelte) | read-complete | Reviewed guide source/rendering, draft exclusion, categories, related links, metadata and safe text output. External citations were not fact-checked as part of this code audit. |
| [+page.server.ts](../../../web/src/routes/blog/+page.server.ts) | read-complete | Reviewed guide source/rendering, draft exclusion, categories, related links, metadata and safe text output. External citations were not fact-checked as part of this code audit. |
| [+page.svelte](../../../web/src/routes/blog/+page.svelte) | read-complete | Reviewed guide source/rendering, draft exclusion, categories, related links, metadata and safe text output. External citations were not fact-checked as part of this code audit. |

## web/src/routes/blog/[slug]

| File | Review | Notes |
| --- | --- | --- |
| [+page.server.ts](../../../web/src/routes/blog/[slug]/+page.server.ts) | read-complete | Reviewed guide source/rendering, draft exclusion, categories, related links, metadata and safe text output. External citations were not fact-checked as part of this code audit. |
| [+page.svelte](../../../web/src/routes/blog/[slug]/+page.svelte) | read-complete | Reviewed guide source/rendering, draft exclusion, categories, related links, metadata and safe text output. External citations were not fact-checked as part of this code audit. |

## web/src/routes/blog/rss.xml

| File | Review | Notes |
| --- | --- | --- |
| [+server.ts](../../../web/src/routes/blog/rss.xml/+server.ts) | read-complete | Reviewed guide source/rendering, draft exclusion, categories, related links, metadata and safe text output. External citations were not fact-checked as part of this code audit. |

## web/src/routes/blog/topic/[topic]

| File | Review | Notes |
| --- | --- | --- |
| [+page.server.ts](../../../web/src/routes/blog/topic/[topic]/+page.server.ts) | read-complete | Reviewed guide source/rendering, draft exclusion, categories, related links, metadata and safe text output. External citations were not fact-checked as part of this code audit. |
| [+page.svelte](../../../web/src/routes/blog/topic/[topic]/+page.svelte) | read-complete | Reviewed guide source/rendering, draft exclusion, categories, related links, metadata and safe text output. External citations were not fact-checked as part of this code audit. |

## web/src/routes/buyer-invitations

| File | Review | Notes |
| --- | --- | --- |
| [+layout.svelte](../../../web/src/routes/buyer-invitations/+layout.svelte) | read-complete | Reviewed simple public route wrapper. |

## web/src/routes/buyer-invitations/[token]

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/buyer-invitations/[token]/+page.svelte) | read-complete | Reviewed invitation preview, OTP, legal-version acceptance, stable mutation and token-change race handling. |

## web/src/routes/buyer

| File | Review | Notes |
| --- | --- | --- |
| +layout.svelte (`web/src/routes/buyer/+layout.svelte`, since removed) | read-complete | Reviewed pathname-keyed authentication gate, buyer navigation, bank-return notice and sign-out. |
| +page.svelte (`web/src/routes/buyer/+page.svelte`, since removed) | read-complete | Reviewed independently loaded account/balances/payment dates, stale-request suppression and totals. F020: legitimate zero-debt states poison whole balance as unconfirmed. |

## web/src/routes/buyer/amendments

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/amendments/+page.svelte`, since removed) | read-complete | Reviewed exact original/proposed schedule linkage, unpaid amounts, consent, expiry presentation and decision replay. |

## web/src/routes/buyer/bank-authorization/[reference]

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/bank-authorization/[reference]/+page.svelte`, since removed) | read-complete | Read all: enrollment states, mutation intent, amount disclosure and bank link allowlist. F005: interrupted-request support link targets missing route. |

## web/src/routes/buyer/credit-requests/[requestID]

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/credit-requests/[requestID]/+page.svelte`, since removed) | read-complete | Reviewed acceptance and bank-permission separation, exact agreement fields, receipt/report actions, request-scope guards and balance links. Payment claim paid_at comes from mutation creation rather than actual bank-transfer date; F021. |

## web/src/routes/buyer/disputes

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/disputes/+page.svelte`, since removed) | read-complete | Reviewed dispute list mapping, amount and detail links. |

## web/src/routes/buyer/disputes/[id]

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/disputes/[id]/+page.svelte`, since removed) | read-complete | Reviewed buyer-scoped dispute detail wrapper. |

## web/src/routes/buyer/history

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/history/+page.svelte`, since removed) | read-complete | Reviewed historical financial totals, correction request identity, amount validation and preserved decision notes. |

## web/src/routes/buyer/mandates

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/mandates/+page.svelte`, since removed) | read-complete | Reviewed mandate state validation, cancellation result confirmation and pending cancellation display; F016 backend cancellation limitation applies. |

## web/src/routes/buyer/notifications

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/notifications/+page.svelte`, since removed) | read-complete | Reviewed shared notification history wrapper. |

## web/src/routes/buyer/obligations

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/obligations/+page.svelte`, since removed) | read-complete | Reviewed obligation ID links and balance list filter. |

## web/src/routes/buyer/obligations/[id]

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/obligations/[id]/+page.svelte`, since removed) | read-complete | Reviewed schedule/notice decoding, acknowledged-notice retry identity and payment navigation. |

## web/src/routes/buyer/payments

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/payments/+page.svelte`, since removed) | read-complete | Reviewed payment-claim states/dates, ordering and temporary hold deadline display. |

## web/src/routes/buyer/permissions

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/permissions/+page.svelte`, since removed) | read-complete | Reviewed current consent selection, evidence digest, user/seller mutation scope and response matching. |

## web/src/routes/buyer/purchases

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/purchases/+page.svelte`, since removed) | read-complete | Reviewed buyer consumer-purchase list wrapper. |

## web/src/routes/buyer/purchases/[saleID]

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/purchases/[saleID]/+page.svelte`, since removed) | read-complete | Reviewed keyed purchase component wrapper and route scope. |

## web/src/routes/buyer/requests

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/requests/+page.svelte`, since removed) | read-complete | Reviewed sent/reviewing filters and seller/goods links; accepted requests excluded intentionally, but mandate-renewal link points here. |

## web/src/routes/buyer/settings

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/settings/+page.svelte`, since removed) | read-complete | Reviewed shared account settings and recovery navigation. |

## web/src/routes/buyer/trade-lines

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/buyer/trade-lines/+page.svelte`, since removed) | read-complete | Reviewed line/statement identity matching, bounded parallel reads, accepted hashes and drawdown receipt/cancel actions; F018/F019 surface here. |

## web/src/routes/c

| File | Review | Notes |
| --- | --- | --- |
| [+layout.svelte](../../../web/src/routes/c/+layout.svelte) | read-complete | Reviewed simple public route wrapper. |

## web/src/routes/c/[token]

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/c/[token]/+page.svelte) | read-complete | Reviewed invitation redirect placeholder. |
| [+page.ts](../../../web/src/routes/c/[token]/+page.ts) | read-complete | Reviewed encoded invitation token redirect. |

## web/src/routes/contact

| File | Review | Notes |
| --- | --- | --- |
| [+page.server.ts](../../../web/src/routes/contact/+page.server.ts) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |
| [+page.svelte](../../../web/src/routes/contact/+page.svelte) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |

## web/src/routes/demo

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/demo/+page.svelte) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |

## web/src/routes/faq

| File | Review | Notes |
| --- | --- | --- |
| [+page.server.ts](../../../web/src/routes/faq/+page.server.ts) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |
| [+page.svelte](../../../web/src/routes/faq/+page.svelte) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |

## web/src/routes/for-buyers

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/for-buyers/+page.svelte`, since removed) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |

## web/src/routes/for-suppliers

| File | Review | Notes |
| --- | --- | --- |
| +page.svelte (`web/src/routes/for-suppliers/+page.svelte`, since removed) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |

## web/src/routes/glossary

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/glossary/+page.svelte) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |

## web/src/routes/how-it-works

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/how-it-works/+page.svelte) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |

## web/src/routes/join

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/join/+page.svelte) | read-complete | Reviewed referral code and login destination construction. |

## web/src/routes/legal/complaints

| File | Review | Notes |
| --- | --- | --- |
| [+page.server.ts](../../../web/src/routes/legal/complaints/+page.server.ts) | read-complete | Reviewed versioned legal-content loading and escaped fallback document rendering/links. Legal prose read for source consistency; not a legal compliance opinion. |
| [+page.svelte](../../../web/src/routes/legal/complaints/+page.svelte) | read-complete | Reviewed versioned legal-content loading and escaped fallback document rendering/links. Legal prose read for source consistency; not a legal compliance opinion. |

## web/src/routes/legal/privacy

| File | Review | Notes |
| --- | --- | --- |
| [+page.server.ts](../../../web/src/routes/legal/privacy/+page.server.ts) | read-complete | Reviewed versioned legal-content loading and escaped fallback document rendering/links. Legal prose read for source consistency; not a legal compliance opinion. |
| [+page.svelte](../../../web/src/routes/legal/privacy/+page.svelte) | read-complete | Reviewed versioned legal-content loading and escaped fallback document rendering/links. Legal prose read for source consistency; not a legal compliance opinion. |

## web/src/routes/legal/terms

| File | Review | Notes |
| --- | --- | --- |
| [+page.server.ts](../../../web/src/routes/legal/terms/+page.server.ts) | read-complete | Reviewed versioned legal-content loading and escaped fallback document rendering/links. Legal prose read for source consistency; not a legal compliance opinion. |
| [+page.svelte](../../../web/src/routes/legal/terms/+page.svelte) | read-complete | Reviewed versioned legal-content loading and escaped fallback document rendering/links. Legal prose read for source consistency; not a legal compliance opinion. |

## web/src/routes/pay

| File | Review | Notes |
| --- | --- | --- |
| [+layout.svelte](../../../web/src/routes/pay/+layout.svelte) | read-complete | Reviewed simple public route wrapper. |

## web/src/routes/pay/[token]

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/pay/[token]/+page.svelte) | read-complete | Reviewed public balance presentation, request race guard and authenticated buyer navigation. |

## web/src/routes/pricing

| File | Review | Notes |
| --- | --- | --- |
| [+page.server.ts](../../../web/src/routes/pricing/+page.server.ts) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |
| [+page.svelte](../../../web/src/routes/pricing/+page.svelte) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |

## web/src/routes/receipt

| File | Review | Notes |
| --- | --- | --- |
| [+layout.svelte](../../../web/src/routes/receipt/+layout.svelte) | read-complete | Reviewed simple public route wrapper. |

## web/src/routes/receipt/[public_token]

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/receipt/[public_token]/+page.svelte) | read-complete | Reviewed public receipt states, money/date validation and redacted presentation. |

## web/src/routes/recover

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/recover/+page.svelte) | read-complete | Reviewed public recovery request, evidence, completion and cancellation forms and token extraction. |

## web/src/routes/robots.txt

| File | Review | Notes |
| --- | --- | --- |
| [+server.ts](../../../web/src/routes/robots.txt/+server.ts) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |

## web/src/routes/secure

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/secure/+page.svelte) | read-complete | Reviewed secure-link resolution and same-origin HTTP/HTTPS destination validation. |

## web/src/routes/security

| File | Review | Notes |
| --- | --- | --- |
| [+page.svelte](../../../web/src/routes/security/+page.svelte) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |

## web/src/routes/sitemap.xml

| File | Review | Notes |
| --- | --- | --- |
| [+server.ts](../../../web/src/routes/sitemap.xml/+server.ts) | read-complete | Reviewed complete public route/source: state, rendering, navigation, metadata, sample-only interactions and responsive styles as applicable; no executable verification. |

## web/src

| File | Review | Notes |
| --- | --- | --- |
| [service-worker.ts](../../../web/src/service-worker.ts) | read-complete | Read all: cache restricted to manifest static assets; document/API requests excluded. No confirmed defect. |

## web/static

| File | Review | Notes |
| --- | --- | --- |
| [apple-touch-icon.png](../../../web/static/apple-touch-icon.png) | visual-review-complete | Viewed complete raster asset. Brand icon or 1200×630 social image is legible and consistent with SVG source; binary asset has no source-code lines. |
| [favicon.svg](../../../web/static/favicon.svg) | read-complete | Read complete static manifest/SVG source; reviewed local asset links, display metadata, geometry and absence of executable/external embedded content. |
| [icon-192.png](../../../web/static/icon-192.png) | visual-review-complete | Viewed complete raster asset. Brand icon or 1200×630 social image is legible and consistent with SVG source; binary asset has no source-code lines. |
| [icon-512.png](../../../web/static/icon-512.png) | visual-review-complete | Viewed complete raster asset. Brand icon or 1200×630 social image is legible and consistent with SVG source; binary asset has no source-code lines. |
| [icon-maskable.svg](../../../web/static/icon-maskable.svg) | read-complete | Read complete static manifest/SVG source; reviewed local asset links, display metadata, geometry and absence of executable/external embedded content. |
| [icon.svg](../../../web/static/icon.svg) | read-complete | Read complete static manifest/SVG source; reviewed local asset links, display metadata, geometry and absence of executable/external embedded content. |
| [manifest.webmanifest](../../../web/static/manifest.webmanifest) | read-complete | Read complete static manifest/SVG source; reviewed local asset links, display metadata, geometry and absence of executable/external embedded content. |
| og-source.svg (`web/static/og-source.svg`, since removed) | read-complete | Read complete static manifest/SVG source; reviewed local asset links, display metadata, geometry and absence of executable/external embedded content. |
| [og.jpg](../../../web/static/og.jpg) | visual-review-complete | Viewed complete raster asset. Brand icon or 1200×630 social image is legible and consistent with SVG source; binary asset has no source-code lines. |
| [og.png](../../../web/static/og.png) | visual-review-complete | Viewed complete raster asset. Brand icon or 1200×630 social image is legible and consistent with SVG source; binary asset has no source-code lines. |

## web

| File | Review | Notes |
| --- | --- | --- |
| [svelte.config.js](../../../web/svelte.config.js) | read-complete | Read all: adapter selection, aliases and CSP. No confirmed defect in isolation. |

## web/tests

| File | Review | Notes |
| --- | --- | --- |
| [access-control.spec.ts](../../../web/tests/access-control.spec.ts) | read-complete | Read all test source and mock responses; reviewed assertions, isolation, mutation retry and failure handling where applicable. Not executed; mocked success does not verify backend behavior. |
| [accessibility.spec.ts](../../../web/tests/accessibility.spec.ts) | read-complete | Read all test source and mock responses; reviewed assertions, isolation, mutation retry and failure handling where applicable. Not executed; mocked success does not verify backend behavior. |
| [account-safety-outage.spec.ts](../../../web/tests/account-safety-outage.spec.ts) | read-complete | Read all test source and mock responses; reviewed assertions, isolation, mutation retry and failure handling where applicable. Not executed; mocked success does not verify backend behavior. |
| [admin-controls-retry.spec.ts](../../../web/tests/admin-controls-retry.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [admin-outage.spec.ts](../../../web/tests/admin-outage.spec.ts) | read-complete | Read all test source and mock responses; reviewed assertions, isolation, mutation retry and failure handling where applicable. Not executed; mocked success does not verify backend behavior. |
| [admin-section.spec.ts](../../../web/tests/admin-section.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [admin-workflows.spec.ts](../../../web/tests/admin-workflows.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [article-drafts.spec.ts](../../../web/tests/article-drafts.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [audit-completion.spec.ts](../../../web/tests/audit-completion.spec.ts) | read-complete | Read all test source and mock responses; reviewed assertions, isolation, mutation retry and failure handling where applicable. Not executed; mocked success does not verify backend behavior. |
| [audit-fixes.spec.ts](../../../web/tests/audit-fixes.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. Invitation fixtures omit required legal versions and identity-notice fields; current decoder rejects them before OTP. Acceptance scenario also omits required consent. |
| [audit-full-form.spec.ts](../../../web/tests/audit-full-form.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. Upload-success fixture provides sha256 without document id; current form requires id and will not create the request asserted by this scenario. |
| [audit-product-journeys.spec.ts](../../../web/tests/audit-product-journeys.spec.ts) | read-complete | Read all test source and mock responses; reviewed assertions, isolation, mutation retry and failure handling where applicable. Not executed; mocked success does not verify backend behavior. Logout failure scenario covers 503 but not backend revocation failure mapped to 401 (F002). |
| [audit-reliability-unit.spec.ts](../../../web/tests/audit-reliability-unit.spec.ts) | read-complete | Read all test source and mock responses; reviewed assertions, isolation, mutation retry and failure handling where applicable. Not executed; mocked success does not verify backend behavior. |
| [business-settings.spec.ts](../../../web/tests/business-settings.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [completion-history.spec.ts](../../../web/tests/completion-history.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [content-seo.spec.ts](../../../web/tests/content-seo.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [customer-limit-recovery.spec.ts](../../../web/tests/customer-limit-recovery.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [demo-readiness.spec.ts](../../../web/tests/demo-readiness.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [dispute-detail-outage.spec.ts](../../../web/tests/dispute-detail-outage.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [financial-completion.spec.ts](../../../web/tests/financial-completion.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. Negative empty-state assertion uses obsolete wording, although the positive error assertion still provides protection. |
| [health.spec.ts](../../../web/tests/health.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [money.spec.ts](../../../web/tests/money.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [notification-history-outage.spec.ts](../../../web/tests/notification-history-outage.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [privacy-partial-approval.spec.ts](../../../web/tests/privacy-partial-approval.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [product-flows.spec.ts](../../../web/tests/product-flows.spec.ts) | read-complete | Read all test source and mock responses; reviewed assertions, isolation, mutation retry and failure handling where applicable. Not executed; mocked success does not verify backend behavior. Drawdown receipt-issue scenario ends after reporting; it does not cover resolution of the stuck reservation (F019). Payment reporting asserts amount/reference but not accuracy of payment date (F021). |
| [product-quality.spec.ts](../../../web/tests/product-quality.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [public-money-links.spec.ts](../../../web/tests/public-money-links.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [public-security-content.spec.ts](../../../web/tests/public-security-content.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [publication-rendering.spec.ts](../../../web/tests/publication-rendering.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [real-stack-financial.spec.ts](../../../web/tests/real-stack-financial.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [registration-recovery.spec.ts](../../../web/tests/registration-recovery.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [sign-in.spec.ts](../../../web/tests/sign-in.spec.ts) | read-complete | Read all test source and mock responses; reviewed assertions, isolation, mutation retry and failure handling where applicable. Not executed; mocked success does not verify backend behavior. |
| [website-editor.spec.ts](../../../web/tests/website-editor.spec.ts) | read-complete | Read complete test source, fixtures, expected behavior and failure paths; not executed. |
| [workspace-audit.spec.ts](../../../web/tests/workspace-audit.spec.ts) | read-complete | Read all test source and mock responses; reviewed assertions, isolation, mutation retry and failure handling where applicable. Not executed; mocked success does not verify backend behavior. |

## web

| File | Review | Notes |
| --- | --- | --- |
| [tsconfig.json](../../../web/tsconfig.json) | read-complete | Reviewed TypeScript configuration and inherited Svelte settings. |
| [vercel.json](../../../web/vercel.json) | read-complete | Read all: SvelteKit framework and URL normalization settings. |
| [vite.config.ts](../../../web/vite.config.ts) | read-complete | Read all: development API proxy and environment override. No execution. |
