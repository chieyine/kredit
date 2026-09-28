# Audit implementation — 27 September 2026

The user authorized the audit recommendations and clarified the product rule: **one operator per business, as few decisions as possible**. The user also confirmed the platform has not launched and only their testing account exists: no compatibility layers or historical staff subsystem are required. The implementation follows those rules. Code changes are local; migration 213 and deployment have not been executed. No agents were used and no tests were run. Obsolete feature-specific tests were removed with their implementations; no tests were added.

## Product and findings

| Reference | Implemented change |
|---|---|
| F01 | Essential setup and record links remain available. The initial restoration of staff/approval screens is superseded by the one-person rule: those screens, APIs and backend modules are deleted. The expanded-workspace switch and old-route redirect layer are also removed. |
| F02 | Correction to the audit: the customer HTTP handler already added authorized obligation balances; the directory type alone did not describe its response. The final implementation validates that response and computes it in one scoped database projection. It does not load the full sales list a second time. |
| F03 | Personal balances use complete summaries and exact integer arithmetic. Overdue totals are capped by outstanding principal, and due counts exclude inactive purchases. |
| F04–F06 | Business sales save a draft for fee/term review before Send. Personal sales require an explicit stock/delivery commitment. Grouped customer links open every matching purchase. |
| F07–F09 | Personal receipts and reported payments appear in Money in. Today retains independently available actions, including returns/refunds. Failed defaults, customer refresh, stale selection, MFA and saved-command recovery have explicit paths. |
| F10 | Current flow and route map describe this implementation; earlier audits and implementation notes retain their historical scope. |
| F11 | Bootstrap and development share a prerequisite step that waits for infrastructure and completes upload-bucket initialization before the host processes start. |
| F12 | Stored configuration checking loads bootstrap settings, reads saved connections, then performs full merged validation. Environment-only checking is unchanged. |
| F13 | Release certification and Terraform no longer require unused legal runtime metadata. A shared static checker validates the actual checked-in publication and matching backend versions. The separate legal-approval evidence gate remains. |
| F14 | Proxy maintenance validates every CIDR and the full candidate with the deployed Compose image before replacement. Generation is separate from explicit activation; activation retains a backup and returns failure after attempted restoration if restart fails. It uses the production environment path and release override. |
| F15 | Low-data mode keeps readable mobile navigation colors. |
| F16 | Existing credit-limit collection/detail routes remain reachable from Settings. The collection is a compact record list, with no new limit-creation form. |

## Maintainability recommendations

| Reference | Implemented change and boundary |
|---|---|
| M01 | Personal summaries aggregate money and claim counts without loading event histories; their schedule/balance rules are shared with detail. Detailed purchase lists fetch all event histories once per page. Customer balances use one tenant-scoped, repeatable-read projection. These are separate domain snapshots; the implementation does not claim a single atomic dashboard across domains. |
| M02 | Give is the only sale-entry implementation. Old business forms are deleted and all current links point to Give. Optional invoices, installment schedules and delivery terms remain available. The old device-draft storage module and its recovery-only UI are removed; business drafts are saved as actual sale records. |
| M03 | Credit models, views, hydration and persistence have separate files. Payment, collection, dispute, notice and credit-limit HTTP handlers have their own files. Governance, sale overview/document reading and optional sale fields have components that own their state. Stable financial transaction bodies were retained except the explicitly authorized owner credit-note change. Files were split by responsibility, not to reach an arbitrary size. |
| M04 | Admin reads use the shared bounded, decoded HTTP path. The generic workspace list requires a per-page decoder. Money fields use strict integer decoding; obsolete duplicate read/CSRF helpers were removed. Mutation intents remain the command/recovery mechanism. This is convergence of touched flows, not a claim that every old adapter was rewritten. |
| M05 | Owner-only authorization, deletion of staff APIs/modules, and migration 213 implement the product rule directly. There is no 410 compatibility handler or read-only staff-history API. Customer acceptance, identity, banking, ledger and platform-admin controls remain separate. |
| M06 | The authoritative current flow, route map and cutover guide are updated. Audit coverage files remain snapshots of the completed review, rather than being relabelled as fresh reads after implementation. |
| M07 | The HashiCorp Kubernetes provider lockfile is retained for Linux/macOS on amd64/arm64. CI initializes with a read-only lockfile. No provider was applied or deployment changed. Existing container/action pin baselines were not fabricated or silently refreshed. |
| M08 | Both API scripts share one route/operation inventory. It recognizes direct calls and statically resolvable mutation intents, excludes test source, and distinguishes operation checks from weaker route-to-screen mappings. Dynamic calls and response behavior remain outside static proof. Touched components use existing tokens; there is no new global override layer. Remaining global CSS can be migrated as its components change. |

## One-person migration

Migration 213 enforces one owner per business and removes unused staff/delegation/branch/internal-approval/proposed-terms-import tables and their history. Staff roles, invitation fields, staff MFA readiness, branch-only policies/triggers and shared legacy process status are removed. It rejects inconsistent ownership instead of choosing an owner or deleting accounts. Business identities, customer invitations, agreements, payments and ledger records remain.

Current owner checks still protect purchases, notifications and command replay. Credit notes keep draft/review/apply with one owner, immutable intent, ownership locking and transactional money checks. No database was reset and the testing account was not changed. The migration is forward-only; follow the [cutover guide](../../operations/production-cutover-runbook.md) before applying it to an existing installation.

The pre-launch legal defaults and fallback pages now describe the one-owner product directly, without a notice about former staff accounts. No external legal publication was performed. Runtime status requires identifiable process instances; old fee-disclosure matching and obsolete internal-approval error recovery were removed. Existing migration files and audit evidence remain development history, not supported older runtime behavior.

## Verification

See [CHECKS.md](CHECKS.md) for final compiler, static analysis and build results. SQL was reviewed against the migration chain but not executed. No tests, browser/device checks, clean-volume startup, provider behavior, database migration or production rollout is claimed. These are explicit verification limits, not evidence that those operations succeeded.
