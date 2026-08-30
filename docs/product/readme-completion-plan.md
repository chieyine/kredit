# README completion implementation plan

Plan date: 29 August 2026

Status: repository-owned implementation complete; external release gates remain open.

This plan closes every repository-owned item in
`docs/operations/readme-gap-audit.md` and prepares the evidence required for the
external production gates in README sections 44–45. It does not treat a route,
table, or mock screen as completion. Every workstream must ship as a tested
vertical slice.

Current clause-by-clause evidence:
`docs/product/readme-completion-traceability.md`.

## 1. Outcome

Production V1 is ready for a limited pilot only when:

- the one-time, instalment, and recurring trade-line journeys are complete;
- every material action has domain validation, authorization, idempotency,
  audit evidence, notifications, and durable persistence;
- suppliers, buyers, and operations users can complete their required actions
  from the interface without database or command-line intervention;
- critical mobile and WCAG 2.2 AA checks pass;
- product analytics reconcile to financial source-of-truth records;
- all external legal, provider, security, recovery, and launch gates have
  dated owner approval and evidence.

## 2. Planning assumptions

The indicative sequence assumes one product squad: two backend engineers, one
frontend engineer, shared product/design, QA, security/compliance, and
operations support. Effort is expressed as delivery waves rather than fixed
calendar promises. A smaller team should preserve the sequence and reduce
parallel work.

No workstream may weaken the existing fail-closed production behavior. Money
remains integer kobo, financial mutations remain idempotent and transactional,
and provider timeouts remain unknown until reconciled.

## 3. Delivery sequence

| Wave | Workstreams | Primary outcome | Dependency |
| --- | --- | --- | --- |
| 0 | Foundation and contract lock | shared completion gates, feature flags, fixtures, and design decisions | none |
| 1 | Complete trade-line drawdown lifecycle | the final incomplete financially material journey works end to end | wave 0 |
| 2 | Supplier onboarding and operational readiness | a supplier can become genuinely ready to trade without manual database work | wave 0 |
| 3 | Notification preferences, account recovery, and privacy rights | safe user self-service and compliance operations | waves 0–2 |
| 4 | Operations controls and provider diagnostics | exceptional cases can be handled safely and visibly | waves 1–3 |
| 5 | Complete interactive web flows and WCAG | every required role can complete critical tasks accessibly | waves 1–4 |
| 6 | Product analytics and pilot evidence | measurable, reconciled product performance and signed launch evidence | waves 1–5 |

Workstreams in the same wave may proceed in parallel only when they do not
share a migration, aggregate, or acceptance-test fixture.

## 4. Wave 0 — foundation and contract lock

Workstream ID: `FOUNDATION-CONTRACT-LOCK`

### Scope

1. Convert every unresolved provider/legal question in
   `docs/product/open-questions.md` into an owned decision record with an owner,
   due date, evidence link, and unlocking feature flag.
2. Define shared state names, problem codes, audit action names, notification
   events, analytics events, and permission mappings before adding handlers.
3. Extend `docs/testing/test-matrix.md` with the acceptance scenarios in this
   plan and assign stable fixture identifiers.
4. Add a plan-conformance check that fails when an open item is marked closed
   without linked code and test evidence.
5. Freeze approved interface copy for financial confirmations, recovery,
   disputes, privacy requests, and operations actions.

### Exit criteria

- every workstream has a named product, engineering, compliance, and operations
  owner;
- all database and API changes planned for waves 1–3 have reviewed contracts;
- unresolved external decisions stay behind disabled feature flags;
- the gap register and this plan use the same stable workstream identifiers.

## 5. Wave 1 — complete recurring trade-line drawdowns

Workstream ID: `TL-DRAWDOWN`

### Required lifecycle

`reserved → buyer confirmed → goods released → received/no-issue → obligation activated`

Cancellation and expiry must release the reservation without creating an
obligation. An issue-at-receipt result must open or link a dispute and must not
silently activate the obligation.

### Schema and persistence

- Add immutable drawdown agreement/version records containing trade-line ID,
  buyer, supplier, principal, goods, invoice digest/reference, due/schedule
  terms, grace period, fee terms, mandate reference, and canonical hash.
- Add drawdown-specific release and receipt evidence, actor, timestamps, and
  states, either through constrained extensions of the shared evidence tables
  or dedicated tables with foreign keys.
- Add the resulting obligation and schedule relationship to the drawdown.
- Add database constraints for legal transitions and uniqueness of the active
  obligation per drawdown.
- Persist transactional outbox events alongside every material state change.
- Update the persistence contract and migration/rollback tests.

### Domain and services

- Replace `ActivateDrawdown(drawdownID, obligationID)` with commands that create
  the obligation internally after valid release and receipt evidence.
- Validate supplier/buyer ownership, active line and mandate, accepted immutable
  terms, reservation expiry, available limit, pilot exposure, and state/version.
- Create the obligation, repayment schedule, base-fee ledger postings, exposure
  conversion, audit record, and outbox events in one transaction.
- Make reserve, confirm, release, receipt, cancellation, and activation replay
  safe with explicit idempotency keys.
- Handle concurrent drawdowns and late/duplicate commands without exceeding the
  line or duplicating money.

### API and interface

- Add OpenAPI commands for supplier release, buyer receipt/no-issue, buyer
  issue-at-receipt, and safe cancellation.
- Remove the external `obligation_id` activation input.
- Add supplier and buyer drawdown detail screens with exact terms, evidence,
  progress, available limit, and explicit amount-labelled actions.
- Generate a printable/hash-verified drawdown agreement.
- Send buyer-confirmation, safe-to-release, released, receipt, activation,
  cancellation, and expiry notifications.

### Tests and exit criteria

- Unit tests cover every transition and rejection path.
- PostgreSQL tests cover row locking, reservation expiry, rollback, restart,
  outbox atomicity, and a single resulting obligation.
- Contract tests prove retries and duplicate/out-of-order commands are safe.
- Browser tests complete the supplier/buyer flow on mobile and desktop.
- Ledger reconciliation and line exposure rebuild both equal source-of-truth
  records.
- No caller can supply or attach a pre-existing obligation.

## 6. Wave 2 — supplier onboarding and readiness

Workstream ID: `SUPPLIER-ONBOARDING`

### Schema and domain

- Add a versioned supplier onboarding profile for business identity, verified
  contacts, KYB case, settlement destination, billing method/reference,
  default credit policy, consent versions, and readiness state.
- Store only provider references or encrypted values for restricted settlement
  data; never return full bank details after capture.
- Define explicit readiness rules: organization created, owner verified, KYB
  approved, settlement destination verified, billing configured, current terms
  and privacy accepted, owner MFA active, and finance-role MFA active where
  applicable.
- Record rejection, resubmission, expiry, and provider-review states rather than
  using a single boolean.

### API and interface

- Add read/update endpoints for onboarding steps and a server-computed readiness
  summary. Sensitive steps require recent MFA.
- Build a resumable mobile-first onboarding wizard with clear progress,
  validation, saved drafts, and a final readiness review.
- Turn settlement, billing, security, and default credit policy settings into
  functional permission-aware screens.
- Prevent credit sending, goods release, or live collection when the relevant
  readiness requirement is not satisfied; explain the precise missing step.

### Audit, notifications, and tests

- Audit consent version, settlement changes, billing changes, KYB decisions,
  readiness transitions, and MFA completion without exposing restricted data.
- Notify the owner about verification outcomes, expiring requirements, and
  sensitive setting changes.
- Test role boundaries, cross-tenant isolation, MFA freshness, encrypted data,
  provider timeout/reconciliation, and all readiness gates.
- Browser acceptance: a new owner reaches pilot-ready status and invites a sales
  user; an incomplete organization is safely blocked with recovery guidance.

### Exit criteria

- no manual database change is required to onboard a pilot supplier;
- readiness is derived from durable evidence and cannot be asserted by a client;
- sales and finance roles see only the steps and settings they may manage.

### Completion evidence — 29 August 2026

Migration 047, `internal/onboarding`, the signed-in API, recent-MFA contact and
sensitive-setting commands, provider-hosted KYB, worker reconciliation,
readiness gates, resumable supplier/settings/team interfaces, and the evidence
manifest complete this wave. Unit and PostgreSQL tests prove derived readiness,
optimistic concurrency, restricted settlement storage, restart persistence,
rejection/resubmission/expiry, revision and outbox evidence, and provider
reconciliation. API and browser acceptance prove a new owner can become pilot
ready and invite sales, while incomplete suppliers receive precise recovery
steps and cannot send credit, release goods, or run collections.

## 7. Wave 3 — user control and compliance

### 7.1 Notification preferences

Workstream ID: `NOTIFICATION-PREFERENCES`

- Add authenticated read/update APIs over the existing preference store.
- Model mandatory transactional/security messages separately from optional
  reminders and product messages.
- Support channel preference, quiet hours in Africa/Lagos, fallback rules, and
  accessible explanations of messages that cannot be disabled.
- Add supplier and buyer settings interfaces, audit events, delivery-worker
  integration, and browser tests.
- Prove a preference update affects future delivery without mutating historical
  notification evidence.

### 7.2 Privileged account recovery

Workstream ID: `ACCOUNT-RECOVERY`

- Issue one-time hashed recovery codes at MFA enrollment; display them once and
  support secure regeneration that invalidates the old set.
- Add a recovery request/case aggregate with risk facts, attempts, decision,
  reviewer separation, cooling-off time, and restricted action scope.
- Require at least two independent factors or manual identity/business evidence;
  phone possession alone is insufficient.
- Revoke existing sessions, notify verified contacts, block sensitive financial
  changes during cooling-off, and provide a cancellation route for the genuine
  account owner.
- Add rate limits, enumeration-safe responses, immutable audit events, operations
  review UI, and adversarial tests for replay, takeover, and reviewer bypass.

### 7.3 Data-subject requests

Workstream ID: `PRIVACY-RIGHTS`

- Add durable access, correction, deletion, restriction, objection, consent
  withdrawal, and portability requests with identity binding, due dates,
  decisions, reasons, holds, and audit history.
- Build user submission/status interfaces and a restricted compliance review
  queue with dual control for destructive decisions.
- Produce a privacy-filtered export from authoritative stores.
- Implement retention/legal-hold evaluation and restrictions without deleting
  financial records that must be retained.
- Notify requesters of receipt, clarification, decision, completion, and appeal
  routes without leaking request content.
- Test identity binding, tenancy, deadlines, retention conflicts, export access,
  restriction enforcement, and audit completeness.

### 7.4 Field-level data inventory

Workstream ID: `DATA-INVENTORY`

- Replace the class-level map with one row per persisted field: table/field,
  classification, subject, source, purpose, lawful basis, readers, writers,
  encryption/tokenization, retention, deletion/hold behavior, processor,
  location/transfer considerations, and owner.
- Generate the database-column portion from migrations/catalog metadata and fail
  CI when a new column lacks inventory coverage.
- Link every privacy export/restriction/deletion behavior to the inventory.
- Require compliance approval for retention and lawful-basis values before the
  production gate can pass.

### Wave 3 exit criteria

- users can manage communications and submit privacy requests without support;
- recovery cannot be completed with control of one channel alone;
- every production table column is covered by the maintained data inventory.

Completion evidence (29 August 2026): migration 048 adds versioned preference
categories, hashed one-time recovery codes and rate limits, independent recovery
evidence/review/cooling-off, durable privacy requests, legal holds,
restrictions, dual-control completion, and protected authoritative exports.
Authenticated supplier/buyer settings and restricted operations queues are
functional. PostgreSQL, API, adversarial unit, and browser tests pass. The
generated inventory covers all 924 fields in the `app`, `ledger`, and `river`
schemas and the drift check fails on any uncovered column. Lawful-basis and
retention approval remains an explicit external production gate.

## 8. Wave 4 — operations commands and provider diagnostics

Workstream ID: `OPS-CONTROLS`

### Commands

- Controlled retry for failed jobs and provider webhooks.
- User and organization suspend/restore with scoped consequences.
- Buyer/supplier risk holds with expiry and reason.
- Reconciliation request and unknown-submission resolution.
- Safe collection retry/cancel where provider capability permits.

Every command requires a structured reason, current version, permission check,
recent MFA for sensitive actions, idempotency key, preview of impact, immutable
audit event, and notification to affected users where safe.

### Diagnostics

- Provider latency/error/timeout by operation and bounded time window.
- Webhook lag, duplicate/out-of-order counts, and oldest unprocessed event.
- Unknown submissions and reconciliation age.
- Ledger/report drift, queue age, dead letters, notification backlog, scanner
  backlog, mandate mismatch, and settlement mismatch.
- Redacted correlation identifiers that allow an operator to trace a case
  without exposing credentials or restricted payloads.

### Tests and exit criteria

- Permission, MFA, reason, idempotency, concurrency, and tenant-isolation tests
  cover every command.
- Retry tests prove financial effects cannot duplicate.
- Browser tests cover at least one failed job, webhook, unknown provider result,
  suspension/restoration, and risk hold.
- Runbooks name the corresponding command, expected result, rollback/mitigation,
  escalation path, and evidence to attach.
- Operators can resolve all README-defined deterministic failure scenarios
  without direct database writes.

## 9. Wave 5 — complete interfaces and accessibility

Workstream IDs: `INTERACTIVE-FLOWS`, `WCAG-AA`

### Interface closure

Complete the command surfaces currently represented by list/static pages:

- team invitations, role changes, suspensions, and removals;
- settlement destination verification/change;
- billing setup and billing status;
- buyer request review/decline/accept where applicable;
- supplier and buyer dispute submission, evidence, and status;
- trade-line creation, drawdown release/receipt/cancel, suspension, safe limit
  reduction, and statement access;
- notification, security, recovery, and privacy settings;
- operations controls from wave 4.

All screens need loading, empty, validation, authorization, conflict, offline,
unknown-provider, retry-safe, and success states. Financial confirmation buttons
must include the amount and action rather than generic “Continue” text.

### Accessibility

- Add an automated accessibility engine to Playwright and run it against login,
  onboarding, credit creation, buyer acceptance, goods release/receipt,
  payments, disputes, trade-line drawdowns, settings, recovery, privacy, and
  operations commands.
- Treat serious/critical automated violations as CI failures.
- Test keyboard-only operation, focus restoration, dialog focus trapping,
  skip links, error summaries, status announcements, table alternatives,
  zoom/reflow, contrast, reduced motion, and touch targets.
- Complete manual VoiceOver/TalkBack plus iOS/Android device evidence for the
  critical flows and record known limitations with owners.

### Exit criteria

- every action in the role journeys can be completed through the interface;
- no serious/critical automated accessibility violation remains;
- keyboard, screen-reader, zoom, and mobile evidence is attached to the release
  checklist.

## 10. Wave 6 — product analytics and pilot evidence

Workstream ID: `PRODUCT-ANALYTICS`

### Event model

Instrument privacy-minimized events for:

- onboarding started/step completed/ready;
- customer invited/verified;
- credit drafted/sent/viewed/accepted/declined;
- mandate started/active/failed/cancelled;
- goods released/receipt confirmed/issue raised;
- obligation activated/payment link created/payment claimed/payment confirmed;
- payment due/late/collected/failed/recovered;
- trade line created/drawdown reserved/confirmed/released/activated/expired;
- dispute opened/resolved;
- repeat credit sale and supplier retention.

Events must use opaque or privacy-hashed subject identifiers, explicit purpose,
versioned schemas, server-side timestamps for authoritative actions, and no
invoice text, contact data, bank data, provider tokens, or free-form reasons.

### Metrics

- time to first accepted credit sale;
- invitation-to-verification and sent-to-acceptance conversion;
- acceptance-to-release and release-to-receipt time;
- on-time payment rate and days to payment;
- failed-collection recovery rate;
- dispute and issue-at-receipt rate;
- repeat-sale rate and trade-line utilization;
- supplier activation and retained active suppliers;
- support/operations intervention rate.

### Reconciliation and exit criteria

- Financial funnel counts reconcile to domain source-of-truth tables within a
  documented tolerance; analytics never become the financial source of truth.
- Duplicate command/webhook delivery does not duplicate authoritative events.
- Dashboards display freshness, definitions, filters, and reconciliation status.
- Event inventory and retention appear in the field-level data map.
- Pilot owners have a weekly scorecard with guardrails for loss, disputes,
  provider reliability, support burden, and accessibility defects.

## 11. Cross-cutting definition of done for every workstream

A workstream is not complete until all applicable items exist:

- reviewed migration and rollback/forward-fix strategy;
- domain types, invariants, and state transitions;
- repository queries and transactional command boundary;
- OpenAPI contract and regenerated clients;
- handlers with CSRF, authentication, authorization, idempotency, limits, and
  safe problem details;
- audit events and privacy-safe logs;
- outbox jobs and notifications;
- supplier, buyer, or operations interface;
- unit, permission, integration, concurrency, restart, failure-path, and browser
  tests as applicable;
- runbook and user-facing support guidance;
- updates to `IMPLEMENTATION_STATUS.md`, the test matrix, data inventory,
  changelog, and gap audit;
- evidence links reviewed by the feature owner.

## 12. Release and rollout gates

### Internal alpha

- waves 0–4 complete;
- deterministic providers only;
- synthetic data;
- ledger, exposure, and reports reconcile;
- no open critical/high security defect.

### Staff-assisted supplier beta

- wave 5 automated and manual critical-flow accessibility evidence complete;
- approved legal wording installed and versioned;
- support and operations runbooks exercised;
- recovery, privacy, and incident paths rehearsed;
- strict security and dependency scans pass.

### Limited production pilot

- wave 6 metrics and guardrails operational;
- written provider approval and live certification complete;
- settlement and reversal behavior certified;
- backup/restore, load, alert routing, and incident exercises have dated evidence;
- pilot limits, industry/amount restrictions, kill switches, and feature flags
  are reviewed;
- legal, compliance, security, engineering, operations, and launch owners sign
  the readiness checklist.

### Expansion

- pilot metrics remain within approved guardrails for the agreed observation
  window;
- reconciliation exceptions and support burden are understood;
- no expansion occurs while unresolved money drift, mandate integrity,
  settlement uncertainty, or critical accessibility/security defects exist.

## 13. Recommended execution order inside each wave

1. Confirm product/legal/provider decisions and failure behavior.
2. Update OpenAPI and state-transition documentation.
3. Add migration and database constraints.
4. Implement domain logic and unit tests.
5. Implement transactional persistence and integration tests.
6. Add handlers, permissions, idempotency, audit, and outbox events.
7. Add notifications and interfaces.
8. Add browser/accessibility tests and operational runbooks.
9. Run full migration, seed, Go, generated-code, frontend, and browser checks.
10. Close the gap only after evidence review.

## 14. Completion scoreboard

| Workstream | Wave | Status | Closure evidence |
| --- | --- | --- | --- |
| `FOUNDATION-CONTRACT-LOCK` | 0 | Complete | `docs/product/wave0-contracts.md`, `docs/product/interface-copy.md`, `docs/product/workstream-evidence.tsv`, and executable conformance checks |
| `TL-DRAWDOWN` | 1 | Complete | migration 046; atomic rollback/retry PostgreSQL proof; domain, document, API, generated contract, supplier/buyer UI, worker expiry, and Playwright evidence in `docs/product/workstream-evidence.tsv` |
| `SUPPLIER-ONBOARDING` | 2 | Complete | migration 047; durable derived readiness and provider reconciliation; recent-MFA APIs; masked/reference-only settlement storage; permission-aware onboarding/settings/team UI; PostgreSQL and browser evidence in `docs/product/workstream-evidence.tsv` |
| `NOTIFICATION-PREFERENCES` | 3 | Complete | migration 048; authenticated versioned settings; required-category enforcement; PostgreSQL delivery and browser evidence |
| `ACCOUNT-RECOVERY` | 3 | Complete | migration 048; hashed one-time codes; enumeration-safe rate limits; independent evidence/review; cooling-off; session revocation; adversarial/PostgreSQL/browser evidence |
| `PRIVACY-RIGHTS` | 3 | Complete | migration 048; seven request types; legal-hold/retention decisions; restrictions; dual control; tenant-bound authoritative exports; PostgreSQL/browser evidence |
| `DATA-INVENTORY` | 3 | Complete | generated 924-field register and catalog drift check; legal approval remains an external gate |
| `OPS-CONTROLS` | 4 | Complete | migration 049; immutable versioned command/event records; provider-safe job/webhook/collection controls; suspension/restoration; expiring scoped holds; redacted diagnostics; permission/MFA/PostgreSQL/browser/runbook evidence |
| `INTERACTIVE-FLOWS` | 5 | Complete | permission-aware supplier, buyer and operations command surfaces; loading/empty/error/conflict/offline/success handling; 28 product/health browser journeys and interface evidence manifest |
| `WCAG-AA` | 5 | In progress — external evidence | automated serious/critical axe gate plus keyboard, focus, dialog, skip-link, error-summary, status, reduced-motion, touch-target, 200% reflow and offline checks pass; real VoiceOver/TalkBack, 400% zoom, forced-colors and target-device reviews remain external |
| `PRODUCT-ANALYTICS` | 6 | Complete | migrations 050–051; locked versioned privacy-minimised event vocabulary including authoritative mandates, collections, repeat sales and retention; live authoritative KPI scorecard; loss/dispute/provider/support/accessibility guardrails; definitions, filters, freshness and zero-tolerance reconciliation; PostgreSQL/permission/UI/runbook evidence |

The scoreboard may move to “Complete” only when the linked acceptance evidence
meets section 11. Partial backend, API, or interface delivery must be recorded
as “In progress,” never “Complete.”
