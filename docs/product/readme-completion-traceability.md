# README completion-plan traceability

Review date: 29 August 2026

This matrix traces every repository-owned requirement in
`readme-completion-plan.md` to vertical-slice evidence. “Implemented” means the
code/schema/interface and applicable automated evidence exist. It does not
convert external approvals or real-device reviews into repository evidence.

## Workstreams

| Plan requirement | Implementation evidence | Acceptance evidence | Result |
| --- | --- | --- | --- |
| Wave 0 owned decisions, locked vocabulary, stable fixtures, copy and negative conformance gate | `open-questions.md`, `wave0-contracts.md`, `interface-copy.md`, feature gates and evidence manifest | plan-conformance self-test deliberately rejects unsupported completion | Implemented; unresolved external decisions remain disabled |
| Wave 1 immutable drawdown terms/evidence, constrained lifecycle, internally created obligation/schedule, ledger/exposure/outbox atomicity and replay safety | migration 046, `internal/tradelines`, transactional credit/ledger/schedule integration, agreement documents, supplier/buyer commands | transition, concurrency, expiry, rollback/retry, restart, contract and browser tests | Implemented |
| Wave 1 release/receipt/issue/cancel APIs, exact detail screens, printable hash verification and notifications | OpenAPI/generated client, credit handlers, supplier/buyer trade-line pages, agreement document generator | generated-contract drift, document hash/tamper and Playwright journeys | Implemented |
| Wave 2 versioned derived supplier readiness with provider references/masked settlement and explicit review/rejection/expiry states | migration 047, `internal/onboarding`, provider reconciliation worker and readiness gates | unit/PostgreSQL/API/browser role, MFA, restart and recovery-path tests | Implemented |
| Wave 2 resumable permission-aware onboarding/settings/team interfaces and precise financial gating | onboarding, settlement, billing, credit-policy, security and team pages | pilot-ready and incomplete-supplier browser journeys | Implemented |
| Wave 3 mandatory/optional notification preferences with channels, quiet hours, fallback, audit and future-delivery enforcement | migration 048, notification store/worker, authenticated APIs and supplier/buyer settings | unit/PostgreSQL/API/browser tests | Implemented |
| Wave 3 recovery codes, independent evidence, separation, cooling-off, cancellation, revocation, uniform rate limiting and restricted operations review | migration 048, `internal/usercontrol`, auth/session integration, recovery and admin pages | adversarial unit/PostgreSQL/API/browser tests | Implemented |
| Wave 3 seven privacy-right request types, identity binding, holds/restrictions, dual control, protected authoritative export and notifications | migration 048, privacy services/APIs, user and compliance pages | retention-conflict, tenancy, export, separation and browser tests | Implemented |
| Wave 3 field-per-column inventory and CI drift gate | generated `data-inventory.tsv`, catalog generator/checker, analytics-specific purpose/readers/owner | database catalog comparison covers all 924 fields | Implemented; lawful basis/retention approval external |
| Wave 4 protected retries, suspension/restoration, scoped holds, reconciliation, unknown submission and provider-safe collection controls | migration 049, `internal/platformops`, protected handlers and operations pages | permission/MFA/reason/idempotency/concurrency/PostgreSQL and five browser scenarios | Implemented |
| Wave 4 bounded redacted diagnostics and operator procedures | diagnostics queries/pages and operation-specific runbooks | provider/queue/integrity assertions and browser coverage | Implemented |
| Wave 5 every supplier/buyer/operator command reachable with material UI states and amount-labelled financial actions | completed workspace/settings/dispute/trade-line/operations surfaces | 28 product/health browser journeys | Implemented |
| Wave 5 automated WCAG engine and interaction safeguards | axe-core suite, shared focus/reflow/motion/touch/offline behavior | 15/15 automated accessibility journeys; zero serious/critical findings | Implemented in repository scope |
| Wave 5 real VoiceOver/TalkBack, 400% zoom, forced-colors and target iOS/Android evidence | owner/evidence register in `wave5-accessibility-evidence.md` | requires human reviewer and real target devices | External evidence open; `WCAG-AA` remains in progress |
| Wave 6 versioned privacy-minimised authoritative event model | migrations 050–051 and event catalog; hashed subjects/orgs, explicit purpose, server times, metadata denylist and deterministic keys | privacy validation, restart, replay and authoritative payment-mandate tests | Implemented |
| Wave 6 complete lifecycle vocabulary including collections, repeat sales and supplier retention | migration 051 contract alignment and trigger/backfill functions | canonical event assertions against seeded PostgreSQL | Implemented |
| Wave 6 KPI hierarchy, conversions/times/payment/recovery/dispute/repeat/utilisation/activation/retention/intervention metrics | `internal/reports/analytics.go` | PostgreSQL scorecard calculation test | Implemented |
| Wave 6 loss, dispute, provider, support and accessibility weekly guardrails | scorecard guardrail queries, protected UI and weekly review runbook | metric-presence, permission/AAL2, filter and browser tests | Implemented; numerical targets require owner approval |
| Wave 6 freshness, definitions, filters and zero-tolerance source reconciliation | scorecard API/UI and `pilot-scorecard.md` | seeded exact reconciliation and mismatch-visible UI behavior | Implemented |

## Cross-cutting definition of done

| Control | Evidence | Result |
| --- | --- | --- |
| Migration and forward/rollback strategy | migrations 046–051 include down or explicit append-only forward-fix disposition; clean schema migration tested | Implemented |
| Domain invariants and transactional repositories | domain stores plus PostgreSQL repositories, row locks, outbox and ledger boundaries | Implemented |
| OpenAPI, handlers and generated clients | `api/openapi.yaml`, generated Go/TypeScript clients, authentication/CSRF/RBAC/idempotency middleware | Implemented |
| Audit, privacy-safe logs, notifications/jobs | audit/outbox/notification services and redaction controls | Implemented |
| Role interfaces and support guidance | supplier, buyer and operations routes plus runbook index | Implemented |
| Unit, permission, integration, concurrency, restart, failure and browser evidence | Go/PostgreSQL suite and 46 Playwright journeys | Implemented |
| Status, matrix, inventory, changelog, gap audit and owner evidence | maintained repository documents and executable manifest check | Implemented |

## External release gates

The following plan items cannot truthfully be implemented by source changes:

- approved legal wording, retention periods and lawful bases;
- written real-provider approval and live mandate/collection/settlement/reversal certification;
- independent strict security/dependency/penetration evidence;
- real VoiceOver/TalkBack, 400% zoom, forced-colors and target-device review;
- target-environment backup/restore, load, alert-routing and incident exercises;
- support training, approved pilot targets/limits and cross-functional launch signatures.

Their required reference fields, owners, due dates and fail-closed feature gates
are present. Until evidence is supplied, staff beta/production pilot readiness
must remain unchecked even though repository implementation is complete.
