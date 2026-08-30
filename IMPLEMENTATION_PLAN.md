# Kredit Production V1 Implementation Plan

Status: proposed  
Source of truth: `README.md` (baseline verified 16 August 2026)  
Starting point: specification-only repository; implementation begins at Milestone 0.

## 1. Outcome

Deliver a narrow, production-safe modular monolith for supplier-funded B2B trade credit in Nigeria. The first pilot must support the complete lifecycle:

1. verified supplier creates a credit request;
2. buyer reviews and accepts immutable terms;
3. buyer authorises a provider-neutral repayment mandate;
4. supplier confirms goods release and buyer confirms receipt or raises an issue;
5. Kredit activates and tracks the obligation;
6. voluntary and provider-backed repayments are recorded in a balanced ledger;
7. disputes, reversals, notifications, reports, and audit history are handled;
8. production capability is enabled only after legal, provider, security, and operational gates pass.

The product must remain supplier-funded. Do not introduce lending capital, wallets, cards, consumer lending, marketplace functionality, opaque credit scores, or other README non-goals.

## 2. Non-negotiable implementation principles

- **Financial correctness first:** store money as integer kobo; use PostgreSQL transactions, constraints, row locks, idempotency keys, and a rebuildable double-entry ledger for all material money actions.
- **Consent and evidence before release:** the buyer accepts the exact immutable agreement version, authorises the mandate, and the supplier records release before principal becomes active.
- **Authoritative state:** PostgreSQL is the domain source of truth; the ledger is authoritative for money-derived balances; provider systems are authoritative for their own mandate/debit/settlement states; browser state and WhatsApp messages are never authoritative.
- **Contract-first delivery:** update `docs/api/openapi.yaml` before changing API behaviour; generate Go and TypeScript clients; fail CI on generated-code drift.
- **Vertical slices:** complete one coherent workflow end to end before expanding breadth. Keep domain rules in Go modules, not HTTP handlers or Svelte components.
- **Provider neutrality:** use capability-based interfaces and deterministic simulators for KYC/KYB, mandates, collections, notifications, and storage scanning. Keep real integrations behind feature flags until approved.
- **Operational traceability:** every feature includes audit events, permissions, jobs, notifications, docs, normal-path tests, and failure-path tests.
- **Privacy and least privilege:** separate person, business, authority, consent, and mandate facts; mask restricted data; never log secrets, OTPs, raw tokens, BVNs, PINs, or credentials.

## 3. Target repository and runtime

Create the repository structure defined in the README:

- `cmd/api`, `cmd/worker`, `cmd/migrate`, `cmd/seed`, `cmd/reconcile`;
- Go domain modules under `internal/` for access, auth, organisations, buyers, identity, credit, agreements, mandates, payments, ledger, fees, collections, disputes, documents, notifications, reports, reputation, risk, admin, providers, settlements, and web;
- `api/openapi.yaml` plus generated Go/TypeScript artifacts;
- SQL-first Goose migrations and sqlc queries/generated code;
- SvelteKit app under `web/`;
- `docs/`, `infra/`, `tests/`, CI workflows, and local development files.

Deploy three processes from one modular monolith:

| Process | Responsibility |
| --- | --- |
| `kredit-web` | SvelteKit SSR/PWA and same-origin `/api` proxy |
| `kredit-api` | synchronous REST API, authentication, commands, queries, webhooks |
| `kredit-worker` | River jobs, reminders, collection schedules, retries, reconciliation, notifications, reports |

PostgreSQL 18 is the source of truth; S3-compatible storage holds documents/evidence; River uses PostgreSQL; MinIO, Mailpit, provider simulators, and an optional OpenTelemetry stack support local development.

## 4. Milestones and exit criteria

### Pre-milestone — Project foundation

**Build:** establish the repository as a reproducible project before feature work: create the required documentation trackers, record the first ADRs and open questions, define repository ownership/contribution conventions, add safe defaults and ignore rules, and scaffold the target directories without inventing domain behaviour.

**Exit:** a new contributor can identify the source of truth, current implementation status, unresolved external decisions, local environment expectations, and the first implementation command without relying on undocumented assumptions.

### Milestone 0 — Repository and guardrails

**Build:** pin Go/Node/pnpm/PostgreSQL and dependency versions; scaffold the repository; add `Taskfile.yml`, Docker Compose, typed configuration, Goose, sqlc, OpenAPI generation, SvelteKit shell, structured logging, OpenTelemetry skeleton, seed runner, docs folders, `IMPLEMENTATION_STATUS.md`, and CI.

**Safety:** add generation-drift checks, dependency/security scanning, migration checks, linting, race-test entry points, and secret scanning.

**Exit:** `task bootstrap` then `task dev` starts web, API, worker, PostgreSQL, object storage, mail viewer, and simulators; `task ci` reproduces CI locally; generated code is clean.

### Milestone 1 — Authentication and organisations

**Build:** OTP login and controls; secure sessions/cookies; CSRF; MFA for owner/finance; organisations, memberships, invitations, roles, permission checks, audit events, tenant context, and PostgreSQL RLS baseline.

**Exit:** supplier owner creates an organisation, invites a sales user, and performs role-appropriate actions; cross-tenant reads/writes and restricted actions are denied and tested.

### Milestone 2 — Buyer and identity skeleton

**Build:** secure invitation links with hashed tokens and expiry; buyer portal; person/business entities; phone verification; mock KYC/KYB and authority provider interfaces; consent/version records; provider-neutral account/mandate references.

**Exit:** an invited buyer completes mock identity, business, authority, and account verification without exposing sensitive credentials; support/compliance access is masked and audited.

### Milestone 3 — One-time credit vertical slice

**Build:** draft credit request; buyer selection/invitation; principal, goods, invoice, due date, grace period, collection time; immutable agreement version and acceptance evidence; mock mandate; release and receipt confirmation; activation policy; base-fee posting; statement/PDF summary.

**Exit:** the complete request-to-active flow works for the acceptance dataset, with valid state transitions, audit timeline, permission tests, and no activation before all required pre-release conditions.

### Milestone 4 — Ledger and voluntary payments

**Build:** chart of accounts, journal transactions, balanced postings, immutable financial events, payment sources, allocations, partial/full/early payments, supplier-recorded payments, reversals, receipts, fee engine, cached-balance rebuild, and reconciliation views.

**Exit:** outstanding balances rebuilt from postings match projections across normal, concurrent, partial, reversal, and duplicate-request tests; base fee is charged only on activated principal.

### Milestone 5 — Instalments

**Build:** equal/custom schedules; weekly, fortnightly, monthly, and custom dates; month-end rules; schedule-item state machine; early-payment allocation; grace/due calculations; reminders; authorised collection amount/ceiling per schedule.

**Exit:** the six-instalment demo passes end to end, including early payment, partial collection, final statement, and schedule-level audit history.

### Milestone 6 — Recurring trade lines

**Build:** approved limit, exposure, availability, cadence, grace period, start/end dates, active-mandate requirement, buyer-confirmed drawdowns, allocation, suspension, limit changes, and recurring statements.

**Exit:** concurrent drawdowns cannot exceed available limit; a cancelled/inactive mandate blocks new drawdowns while preserving existing obligations.

### Milestone 7 — Collection engine with mock provider

**Build:** eligibility calculation from authoritative outstanding amount; collection reservation; submission idempotency; provider adapter and simulator; webhooks; timeouts/unknown states; partial success; configurable retries; polling/reconciliation; collection fee posting; multi-account capability flag.

**Exit:** provider contract suite passes for success, partial success, timeout, cancellation, retry, reconciliation, and three duplicate webhook deliveries; exactly one financial effect is recorded per provider event.

### Milestone 8 — Disputes and operational controls

**Build:** standard dispute reasons; full/partial dispute amount; evidence and timeline; supplier response; human review; partial collection block; ledger adjustments; support/compliance/dispute console; controlled fee waivers, write-offs, corrections, retries, and break-glass access.

**Exit:** the partial-dispute demo collects only the permitted undisputed amount, produces the correct final ledger, and records every decision, adjustment, and operator access.

### Milestone 9 — WhatsApp and notifications

**Build:** channel-neutral event system; templates and locale/content versioning; WhatsApp webhook verification and command parser; secure confirmation links; invitation/status/reminder/receipt/dispute messages; email and SMS fallback; deduplication and quiet hours.

**Exit:** a supplier can create a structured request from WhatsApp and the buyer completes the web flow; no sensitive payment credential is requested or accepted in chat; delivery/retry events are observable.

### Milestone 10 — Reporting and factual history

**Build:** supplier dashboards, receivables/ageing, due/overdue views, fees, mandate issues, disputes, customer statements, exports, buyer factual repayment history, corrections/appeals, and privacy-safe product analytics.

**Exit:** reports reconcile to ledger and read models; history is interpretable and factual (not a score); correction workflows are permissioned and audited.

### Milestone 11 — Real provider adapter

**Build only after written approval:** implement approved KYC/KYB, mandate, debit, settlement, and notification adapters; sandbox contract tests; provider webhook/reconciliation handling; capability flags; operations runbook; settlement and fee-billing behaviour.

**Exit:** approved sandbox scenarios and provider certification pass; production capability remains disabled until the release gate is signed.

### Milestone 12 — Production hardening and pilot

**Build:** security review and penetration test; DPIA/legal review; dependency and secret review; load/performance tests; backup/restore drill; observability dashboards and alerts; incident/runbooks; support training; pilot limits; feature-flag configuration; release checklist.

**Exit:** no unresolved critical/high security defect; WCAG 2.2 AA critical-flow checks pass; backups restore; provider/legal/compliance approvals are recorded; signed readiness checklist enables a limited pilot.

## 5. Cross-cutting work required in every milestone

For each feature, deliver all of the following together:

- migration/schema and database constraints;
- domain entities, value objects, state transitions, and policy rules;
- sqlc queries/repositories and transactional command service;
- OpenAPI contract, generated client/server code, handler, and error mapping;
- authentication, authorisation, tenant-isolation, and step-up checks;
- audit events and restricted-data logging review;
- River jobs, webhook/idempotency handling, and notification events where relevant;
- Svelte route/feature flow with mobile-first accessible UI;
- unit, property/invariant, integration, provider-contract, and relevant E2E tests;
- documentation updates to `IMPLEMENTATION_STATUS.md`, ADRs, threat model, data map, runbooks, test matrix, and open questions.

## 6. Test and verification gates

Maintain a test matrix covering:

- authentication, MFA, CSRF, session expiry/recovery, RBAC, RLS, and break-glass access;
- agreement immutability, exact acceptance, release/receipt evidence, invitation-token security;
- ledger balance invariants, integer-kobo calculations, fee policy, allocation, concurrency, reversals, and rebuilds;
- every state machine transition and invalid transition;
- mandate/provider timeout, cancellation, duplicate/out-of-order webhooks, partial success, retries, and reconciliation;
- drawdown limit races, dispute partial blocking, correction/write-off approvals, and audit completeness;
- notification deduplication, fallback, quiet hours, and WhatsApp safety;
- mobile responsive behaviour, WCAG 2.2 AA critical flows, and end-to-end acceptance scenarios A–F from the README;
- race detector, fuzzing for money/date/state logic, dependency scanning, load tests, backup restore, and generated-code drift.

No milestone is complete if a financially material path has only a happy-path test.

## 7. Release and launch controls

Use feature flags and controlled configuration for all provider-backed capabilities and pilot limits. Before enabling real collections, record:

- provider approval and supported mandate/account capabilities;
- legal/regulatory decision on Kredit's role and fee/tax treatment;
- completed threat model, DPIA, data map, retention policy, and security review;
- backup restore evidence and operational runbooks;
- support/compliance training and escalation paths;
- configured caps for suppliers, buyers, principal, exposure, drawdowns, retries, enhanced review, and allowed industries;
- signed release readiness checklist and rollback/disable procedure.

## 8. Open questions that block real-provider enablement

Track answers in `docs/product/open-questions.md`; do not invent provider or legal behaviour. The most important gates are mandate structures, multi-account collection, cancellation semantics, direct settlement, fee deduction, enhanced-KYC bands, webhook/reconciliation guarantees, reversal rules, licensing/regulated-partner structure, activation legal meaning, evidence of authority, retention periods, fee taxes, pilot industry, and approved pilot limits.

Until an answer is approved, implement a typed interface, deterministic mock, tests, documentation, and a disabled feature flag so unrelated core work can continue safely.

## 9. First implementation sequence

Start with these concrete repository changes, in order:

1. Scaffold the repository and local services from the target structure.
2. Add version/toolchain checks, typed configuration, Taskfile commands, and CI.
3. Add initial Goose migration, sqlc/OpenAPI generation, and generated-drift checks.
4. Create documentation trackers (`IMPLEMENTATION_STATUS.md`, ADR index, threat model, data map, test matrix, runbooks, open questions, readiness checklist).
5. Implement auth/session/tenant primitives and RLS before business entities.
6. Implement organisation/membership/RBAC flows and seed accounts.
7. Implement buyer invitation, identity/authority abstractions, and secure token handling.
8. Implement the one-time credit slice through activation with a mock mandate.
9. Add the ledger before adding any payment or collection UI.
10. Add acceptance dataset scenarios and run the complete vertical slice in CI.

At the end of each milestone, update implementation status with completed work, tests run, limitations, open questions, migration notes, and the next milestone. A feature is not considered shipped until the corresponding docs and evidence are updated.
