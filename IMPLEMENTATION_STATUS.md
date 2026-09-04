# Implementation Status

Last updated: 3 September 2026

## Audit recommendations implemented (3 September 2026)

Every recommendation from the repository code audit is implemented except the
request-context refactor, which is deliberately staged (below). Migrations now
run through **069**: canonical E.164 phone identifiers with a fail-closed
backfill, session idle expiry and per-account MFA locking, and a rate-limit
budget shared across API replicas for the authentication routes. The linters the
repository had configured but never executed now run in CI and fail closed, and
the fuzz gate — previously a no-op that matched no test — runs eight targets
covering seven of the eight areas README section 40.3 names. The eighth, a CSV
import parser, has no counterpart in v1: CSV is exported, never imported.
[ADR 0005](docs/adr/0005-hand-written-http-and-sql.md) records that Go handlers
and SQL are hand-written and removes the orphaned generated code and `sqlc`
pipeline that nothing compiled.

**Not implemented: request-scoped context propagation.** Repository methods take
no `context.Context`, so threading the request context is a signature change
across roughly fifty methods and every call site and test. Database-side
`statement_timeout`, `lock_timeout` and `idle_in_transaction_session_timeout`
already bound the work; what is missing is cancellation when a client
disconnects. This is a mechanical refactor that should be done with a compiler
in the loop, in its own change, and is recorded here rather than attempted
blind.

## Repository code audit (3 September 2026)

Every owned file in the repository was reviewed against this README. Ten defects
were corrected and locked in with regression tests: an obligation index that made
activated obligations unreachable to payments, collections and disputes; a
self-deadlocking auto-activation sweep; an MFA comparison that accepted an empty
code against an undecodable secret; a biased OTP generator and a guessable CSRF
fallback token; missing Origin/Sec-Fetch-Site verification (README section 21.6);
a web proxy that forwarded client-supplied forwarded-address headers and so
allowed rate-limit and OTP-throttle evasion; a "last day of month" instalment
policy that silently behaved as "cap" (README section 26.2); release notes stored
as waybill evidence; a nil reservation dereference on the successful-debit path;
and a Linux-incompatible CI self-test that made `scripts/ci.sh` fail before it
reached the test suite. No schema change was required. Open recommendations that
were **not** applied, because each needs a product decision, are recorded in the
audit summary: per-user MFA attempt throttling, session idle expiry, an error-
returning payment list API, and stricter phone normalisation.

## Admin workflow completion (3 September 2026)

All seven recommended admin improvements are implemented: unified role-scoped review inbox; independently approved corrections; buyer-consented repayment-date amendments; exact naira/percentage settings and administrator names; policy impact previews; specialist permissions; attention views; searchable/exportable retained history. Migrations now run through **066**. Changes remain local and are not deployed. See [operating guide](docs/runbooks/admin-workflows.md) and [verification](docs/testing/admin-workflows-2026-09-03.md).

## Direct code review (2 September 2026)

Financial race conditions, payment-claim atomicity, report/disclosure accuracy, notification delivery routing and recurring work have received additional fixes. Migrations now run through 060. Financial reports use consistent snapshots; financial notices are transactional; real debits require verified prior-notice delivery; reconciliation cases and durable metrics are implemented. See [the completion record](docs/testing/code-completion-2026-09-02.md) for final changes and tests. Direct accepted-schedule replacement and unverified high-value adjustments remain blocked; the new independently approved amendment/correction workflows provide the supported path. See [review scope, file coverage and verification](docs/testing/code-review-2026-09-02.md). This is code-level evidence, not regulatory or provider certification.

## Mono Sweep backend extension (2 September 2026)

The provider-independent backend now has a Mono Variable Sweep sandbox adapter,
separate buyer acceptance and bank authorization, reusable supplier/business
mandates, durable debit reservations, partial repayment handling, bounded retries,
async authenticated webhooks, and worker-driven reconciliation/notifications.
Migration 052 protects shared mandate capacity and manual-payment races and adds
immutable collection events. Customer registration retains only the provider
reference and consent version; BVN and raw bank details are not retained.

This extension is **not complete under the required provider acceptance gate**:
Mono sandbox credentials/Sweep access are unavailable, and buyer authorization
must be completed in the hosted provider flow. All 21 scenarios must still be
proved against Mono. Production enablement is rejected by configuration.
See [setup, limitations and official sources](docs/runbooks/mono-sweep.md) and
[scenario evidence](docs/testing/mono-sweep-evidence.md).

## README audit correction (29 August 2026)

The earlier structural audit was too broad, so every named flow was re-audited
as a tested vertical slice. The repository-owned gaps in
`docs/operations/readme-gap-audit.md` are now closed. Production V1 is not yet
approved: external legal/provider/security/environment/manual-accessibility
evidence and launch signatures remain fail-closed.

This audit immediately closed three high-risk gaps:

- instalment and custom schedule terms are now part of the immutable canonical
  agreement before buyer acceptance; activation creates the schedule from
  those accepted terms;
- activated agreements now have a hash-verified printable representation with
  parties, terms, acceptance, mandate, goods evidence, schedule, and dispute
  instructions, available to both supplier and buyer;
- trade-line creation no longer trusts client-provided mandate state: it
  verifies mandate ownership, ACTIVE state, and ceiling server-side, while the
  database enforces the active-line mandate invariant.

The audit also wired invoice upload into credit creation and stopped treating
the invoice field as API-only capability.

Detailed requirement matrix: [README conformance audit](docs/operations/readme-conformance.md).
Implementation sequence: [README completion plan](docs/product/readme-completion-plan.md).
Ongoing product-quality contract: [World-class product standard](docs/product/world-class-product-standard.md).

The world-class product-quality pass is complete in repository scope. Public,
supplier, buyer and operations shells now have responsive current-page
navigation and reachable support; protected operations use accessible reasoned
review dialogs instead of browser prompts; the product has safe branded error
recovery, substantive supplier/buyer/trust/support journeys, a complete PWA
manifest and icon set, route-correct search/social metadata and structured
data, strict index/privacy boundaries, and public-versus-private cache policy.
The browser quality gate enforces unique metadata, JSON-LD validity, sitemap
coverage, mobile overflow, public-route WCAG checks, noindex/no-store controls,
404 recovery and install assets. Draft legal copy remains deliberately noindex
until approved, and external accessibility/security/provider/environment/legal
evidence remains fail-closed.

Wave 0 (`FOUNDATION-CONTRACT-LOCK`) is complete. The repository now has
role-owned external decision records with due dates and fail-closed flags, a
locked states/problems/audit/notification/analytics/permission contract for
waves 1–3, stable acceptance fixture IDs, reviewed financial and trust copy,
and an executable evidence manifest. The conformance test rejects a workstream
marked complete without both implementation and test evidence. External
decisions remain open and their capabilities remain disabled.

Wave 1 (`TL-DRAWDOWN`) is complete. Recurring purchases now move through
reserved → exact-hash buyer confirmation → evidenced goods release → buyer
no-issue receipt → internally created obligation. One PostgreSQL transaction
commits exposure conversion, the normalized credit aggregate, one obligation,
one repayment schedule, balanced ledger postings, and outbox events. A forced
post-write failure proves full rollback; retry proves exactly-once activation.
Receipt issues open a linked dispute without activating money, cancellation
and worker expiry release capacity, and both portals expose exact terms,
evidence, progress, printable hash-verified agreements, and amount-labelled
actions.

Wave 2 (`SUPPLIER-ONBOARDING`) is complete. A durable, versioned profile now
derives pilot readiness from organization and representative identity, both
verified owner contacts, provider KYB evidence, a verified masked settlement
destination, billing configuration, default credit policy, current consent
versions, owner MFA, and MFA for every active finance member. Rejection,
resubmission, provider review, timeout, and expiry remain explicit states.
Recent MFA protects sensitive changes; role-filtered views limit sales and
finance access. Credit sending, goods release, and both manual and automated
collection are blocked with precise recovery guidance until pilot ready.
Mobile onboarding, settings, and team-invitation flows are functional.

Wave 4 (`OPS-CONTROLS`) is complete. Platform operators can preview and apply
versioned, reasoned, permission-scoped commands for failed jobs/webhooks,
user/organization suspension and exact-status restoration, expiring
buyer/supplier risk holds, reconciliation, unknown submissions, and bounded
collection retry/cancellation. Recent MFA, idempotency, immutable command
events, affected-user notifications, provider capability checks, and
optimistic concurrency protect every mutation. The diagnostics workspace
reports bounded provider latency/errors/timeouts, webhook duplicate/order/lag,
unknown and overdue reconciliation, queue/dead-letter/backlog, ledger/report
drift, mandate and settlement signals with redacted correlations. Five browser
scenarios and fresh PostgreSQL migration/integration evidence cover the flow.

Wave 5 interface closure (`INTERACTIVE-FLOWS`) is complete. Owners can invite,
change, suspend, restore, and remove non-owner members with MFA/RBAC/audit
protection. Suppliers can create mandate-verified trade lines, suspend/restore
them, safely reduce limits, and complete amount-labelled drawdown actions.
Supplier and buyer dispute opening, evidence, status, decisions, settlement,
billing, buyer review, settings, recovery, privacy, and operations actions are
reachable through permission-aware interfaces with browser coverage for
success and safe provider/conflict failure states. The automated `WCAG-AA`
slice is implemented: axe-core blocks serious/critical violations on twelve
critical journeys, with keyboard, focus restoration, native modal trapping,
skip-link, reduced-motion, touch-target, and 200% reflow checks. The workstream
remains in progress until a human reviewer attaches real VoiceOver/TalkBack,
400% zoom, forced-colors, and target-device evidence documented in
`docs/release/wave5-accessibility-evidence.md`.

Wave 6 (`PRODUCT-ANALYTICS`) is complete in repository scope. Versioned,
privacy-minimised product events now come from authoritative lifecycle writes,
use deterministic replay protection, and reconcile at zero tolerance. The
protected AAL2 compliance scorecard calculates three primary pilot KPIs,
drivers and safety guardrails live from domain records, with definitions,
sources, date/supplier filters, freshness and explicit baseline-dependent
targets. Migration 051 completes authoritative payment-mandate, collection,
repeat-sale and supplier-retention events plus recognised-loss,
support-intervention-rate and accessibility-defect guardrails. Legal approval
of retention/lawful basis and pilot target approval remain external fail-closed
gates.

Latest conformance closure: the README structure is now enforced by
`scripts/readme-conformance.sh` and CI. Local development includes a
deterministic provider simulator for identity, mandates, collections,
notifications, and document scanning. The seed now creates the named supplier,
supplier team, verified Royal Pharmacy buyer, active/cancelled mandates,
recurring trade line, scenarios A/B/D/E as durable credit aggregates, and the
scenario-F duplicate-webhook proof. Draft amendment with optimistic
concurrency, supplier cancellation, and buyer decline are implemented across
the OpenAPI contract, generated clients, Go domain/API, Svelte flows, audit
events, unit tests, and Playwright coverage. The complete migration chain and seed
were validated on clean PostgreSQL databases, including a repeated seed run.

The August implementation pass added durable off-platform payment claims with bounded
collection holds and supplier review, explicit payment-source taxonomy, signed
public payment/receipt projections, mandate cancellation and fresh restoration,
safe trade-line limit reductions, buyer obligation detail, production-shaped
Kubernetes/OpenTofu and monitoring definitions, and repository-owned contract,
browser, integration, and performance harnesses. Those capabilities remain
valid, but the 29 August semantic audit supersedes the earlier claim of complete
README closure.

Implementation update: repository-wide README conformance audit, Priority 6
organization persistence, mandate/network schema, reconciliation tooling, PWA
offline shell, complete marketing/legal/operations route map, and operational
runbooks are now present. Priority 4 persistence-contract/fail-closed deployment hardening, Priority 3 observability/reliability hardening, Priority 2 security/privacy hardening, durable PostgreSQL/River infrastructure, versioned
migration execution, generated OpenAPI artifacts, production HTTP hardening,
object-storage adapters, document/support/relationship APIs, request
idempotency replay/conflict handling, configurable pilot-limit enforcement, and
the factual-history late-payment correction are now present. Priority 0 now
also includes executable RLS identity helpers, a durable PostgreSQL ledger
boundary with deferred balance checks, tenant transaction context, and a
transactional outbox primitive. Priority 1 now adds dedicated River queues,
retry budgets, provider webhook inbox/de-duplication, terminal job dead-letter
capture, ledger reconciliation scheduling, provider circuit-breaker health,
and fail-closed object-storage/schema startup checks. Payments, schedules,
trade lines and collections now join credit, mandates, identity, relationships,
documents, support, authentication, organizations, audit, idempotency, ledger
and outbox behind PostgreSQL runtime adapters. Production launch remains
fail-closed until external provider certification, approved legal text, and
signed environment-specific release evidence are supplied.

Priority 2 now adds fail-closed production secret/TLS endpoint validation,
centralized PII-safe structured logging and audit metadata sanitization, safe
request identifiers and problem details, security-denial audit events, cross-
origin headers, a strict scanner gate, and security/privacy governance artifacts.

Priority 3 now adds bounded request metrics and duration summaries, protected
Prometheus exposition, OTLP HTTP tracing with W3C trace-context propagation,
response status/latency logs, API process timeouts, backup checksums, isolated
restore verification, and an operational observability/SLO contract.

Priority 4 now adds a complete state-table persistence contract, explicit
runtime durability capabilities, fail-closed staging/worker checks, and the
completed repository migration sequence.

Priority 5 begins the repository migration with PostgreSQL-backed
authentication: OTP challenges, sessions, users, and TOTP methods now have a
transactional adapter, encrypted recoverable values, and narrowly scoped SQL
security-definer lookup functions. Priority 6 extends that migration to
organizations, memberships, invitations, tenant-scoped reads, role changes,
and buyer identity/invitation state. Later migrations complete the financial
and operational adapters.

The latest hardening pass centralizes AAL2 step-up enforcement for sensitive
organization permissions, verifies credit-request tenant ownership before send,
binds drawdown activation to an active obligation belonging to the same buyer
and supplier, adds optimistic-version protection to credit snapshots, repairs
persisted mandate ownership lookup, wires idempotency keys through critical
browser mutations, and keeps non-development workers fail-closed until their
domain handlers are configured. These changes reduce unsafe partial behavior;
they do not satisfy the remaining durable-repository or external approval gates.

## Current state

- Pre-milestone — **complete**: source-of-truth documents, ADR/open-question trackers, repository conventions, local configuration contract, and target directory skeleton are present.
- Milestone 0 — **complete (repository scope)**: executable API/worker/migration/seed/reconciliation processes, PostgreSQL, Goose, River, local stack, generated API artifacts, hardened middleware, transactional outbox dispatch, PWA, production Node build, containers, CI and browser checks are present. External launch evidence remains a release gate.
- Milestone 1 — **complete (repository scope)**: OTP authentication, opaque sessions, CSRF, MFA/TOTP step-up, supplier organisation onboarding, memberships, RBAC, audit events, SQL-first auth/org schema, RLS policies, and PostgreSQL auth/organization adapters are implemented.
- Milestone 2 — **complete (repository scope)**: secure buyer invitations, single-use hashed public tokens, buyer portal access, person/business/authority records, consent records, bank-account reference model, identity-provider abstraction, deterministic simulator, API/OpenAPI flows, buyer portal routes, and the PostgreSQL buyer adapter are implemented. Real identity providers remain externally gated.
- Milestone 3 — **complete (repository scope)**: the complete one-time credit lifecycle, draft amendment/cancellation/decline, and PostgreSQL aggregate/normalized state are implemented and tested. Real identity/mandate certification remains external.
- Milestone 4 — **complete (repository scope)**: payments atomically commit allocations, schedules, obligation/aggregate balances, ledger entries, fees and outbox events in PostgreSQL, with restart/idempotency integration coverage. Provider settlement certification remains external.
- Milestone 5 — **complete (repository scope)**: schedules and allocation/reversal/collection accounting are PostgreSQL-backed and restart-safe; quiet-hour reminders use encrypted destinations, durable leases, bounded retries, and River delivery.
- Milestone 6 — **complete (repository scope)**: trade lines, reservations, drawdowns, exposure, suspension and statements are PostgreSQL-backed with concurrency and restart integration coverage. Real mandate certification remains external.
- Milestone 7 — **complete (repository scope)**: collection state, attempts/reservations, webhook deduplication, retry/reconciliation, and exactly-once payment effects are PostgreSQL-backed and restart-tested. Real provider certification remains external.
- Milestone 8 — **complete (repository scope)**: disputes, evidence, decisions, collection blocking, atomic financial adjustments, write-offs/waivers, support timelines, scanned documents, and the role/MFA-gated operations console are durable and integration-tested.
- Milestone 9 — **complete (repository scope)**: notification preferences/templates/delivery state and WhatsApp event deduplication are durable; delayed delivery is recovered through River with encrypted destinations, stable connector idempotency, leases, and bounded retries. Messaging certification remains external.
- Milestone 10 — **complete (repository scope)**: reconciled reports/statements/CSV, correction workflows, and privacy-hashed analytics persist across restarts. Cross-supplier sharing remains consent-gated by design.
- Milestone 11 — **complete (repository scope)**: provider-neutral approved adapter seam, capability/status endpoint, written-approval and pilot-limit gates, settlement/reversal state tracking, signed adapter webhooks, sandbox contract tests, provider approval/event/reconciliation SQL schema, and operations runbook are implemented. The deterministic mock remains available in development; real-provider credentials, approval, certification, and production enablement remain external gates.
- Milestone 12 — **complete (repository scope)**: all state-bearing runtime aggregates use PostgreSQL when configured; production readiness fails closed on schema, scanner/provider configuration, strict security tooling, recovery proof, and signed legal/security/provider/launch evidence.

## Priority 1 architecture evidence

- River queues are separated by financial, provider, collections, notifications, documents, reports, and maintenance concerns with per-class retry budgets and unique job arguments.
- Ledger reconciliation is scheduled by the worker and terminal River failures are copied into `app.job_dead_letters` for replay/audit workflows.
- Provider webhook events enter a durable `app.provider_webhook_inbox` with provider/event uniqueness before asynchronous processing.
- The production worker schedules maintenance and ledger reconciliation, drains
  the transactional outbox into River, delivers quiet-hour notifications, and
  discovers/scans pending documents with recoverable leases and retry budgets.
- Collection providers expose circuit-breaker health state and the provider-status endpoint now reports it.
- Staging/API/worker startup checks the migrated schema, and non-development object-storage failures fail closed instead of silently using process memory.

## Priority 2 security evidence

- Production configuration rejects short or placeholder secrets, local database
  endpoints, disabled database TLS, non-HTTPS service URLs, and missing object
  storage/encryption configuration.
- Structured logs, route paths, request IDs, audit metadata and problem details
  are sanitized to prevent credentials, authentication material, restricted
  identifiers, provider payloads and internal connection errors from leaking.
- Authentication, CSRF and organization-authorization denials create safe,
  request-correlated security audit events.
- `SECURITY_STRICT=1 bash scripts/security.sh` fails closed when required
  vulnerability, static-analysis, dependency and filesystem scanners are absent.
- `SECURITY.md`, the threat model, restricted-data inventory and production
  security checklist provide reviewable handling and release gates.
- Migrations 018–019 enforce append-only audit history, runtime-role RLS on
  worker/provider operational tables, and the least-privilege role template is
  captured in `infra/postgres/roles.sql`.

## Priority 3 reliability evidence

- Every API response now receives a request ID and trace ID; valid W3C
  `traceparent` IDs are propagated into privacy-safe OTLP HTTP spans without
  logging restricted values.
- Request status, total count and bounded duration samples feed the protected
  JSON and Prometheus metrics endpoints.
- API read, write, idle and header timeouts are explicit; backup artifacts are
  private, checksummed and restore drills verify the Goose schema version.
- `docs/operations/observability.md` records the initial service objectives,
  alert thresholds and telemetry privacy contract.

## Priority 4 persistence evidence

- `internal/db/persistence_contract.go` checks every state-bearing table across
  auth, identity, credit, payments, schedules, collections, disputes,
  notifications, reporting, documents, support, ledger, outbox, and River,
  plus the SQL functions/columns required by the auth adapter.
- Runtime capability status marks the database-backed aggregate boundary durable
  only when the complete persistence contract is present.
- API staging startup and readiness fail closed when the contract is incomplete
  or domain aggregates are not durable; worker startup also requires the full
  state-table contract before consuming jobs.
- `docs/operations/persistence-migration.md` defines the adapter sequence,
  encryption requirements, tenant/RLS tests, reconciliation proof, and release
  gate for enabling production.

## Priority 5/6 persistence evidence

- `internal/auth/postgres.go` implements the authentication service contract
  against PostgreSQL, including consumed OTP challenges, attempt accounting,
  hashed session tokens, revocation, encrypted TOTP secrets, and AAL2 elevation.
- Migration 020 adds encrypted OTP target storage, scoped user/session lookup
  functions, and membership visibility needed for authenticated organisation
  queries without disabling tenant RLS.
- `internal/organizations/postgres.go` implements PostgreSQL organization
  creation, membership reads, invitation activation, and role changes with
  UUIDv7 identifiers and request-local RLS context.
- Migration 021 adds the least-privilege organization-count function used by
  pilot limits; the persistence contract validates it before deployment.
- Migration 022 adds the README-specified supplier-buyer relationship and
  payment-mandate/event tables. Provider account references are stored only as
  encrypted ciphertext.
- Migration 023 and `internal/buyers/postgres.go` add durable invitations,
  buyer profiles, representatives, verification cases, consents, and bank
  references with encrypted invitation targets and token-safe lookup.
- Migration 024 and `internal/credit/postgres.go` add a PostgreSQL-backed credit
  aggregate boundary with restart hydration, tenant-scoped snapshots, and
  persisted lifecycle/payment adjustments. Migrations 027–028 tighten tenant
  context for snapshot lookup and add crash recovery leases for outbox claims.
- Migrations 029–030 and the PostgreSQL document/support/schedule repository paths persist
  document metadata/scan state and support case timelines transactionally;
  schedule creation, allocation, reversal, collection accounting, and
  restart-safe reads are also backed by PostgreSQL. Integration round-trip
  tests run against PostgreSQL when `DATABASE_URL` is provided. Migrations
  031–051 complete the remaining financial, operational, onboarding, privacy,
  and analytics persistence contracts.
- SQLC generation now produces the checked `db/generated` package; the
  provider-adapter query name collision was removed and the generator detects
  a Go-installed SQLC binary before attempting a network download.
- A clean PostgreSQL migration run now succeeds through Goose/River version 51.
  The baseline includes a portable `uuidv7()` fallback, corrected schedule RLS
  qualification, and explicit Goose statement boundaries around procedural SQL.
- Production configuration now rejects mock identity/collection providers and
  the migration/reconciliation binaries refuse to fall back to a local
  database when `DATABASE_URL` is missing. Provider webhook inbox processing
  is serialized with a row lock to prevent duplicate concurrent handling.
- `cmd/reconcile` and `internal/ledger.Reconcile` verify global and per-
  transaction journal balance without exposing ledger detail.
- The PWA manifest/service worker caches only the static shell; API, identity,
  financial, invitation, document, and admin responses are explicitly online-
  only.

## Milestone 0 evidence

- `GOCACHE=/tmp/kredit-gocache go test ./...` — passed.
- `GOCACHE=/tmp/kredit-gocache go vet ./...` — passed.
- `bash scripts/ci.sh` — passed; formatting, backend, race, vet, best-effort security and Svelte checks passed.
- `docker compose config --quiet` — passed.
- API smoke test — passed for `/api/v1/healthz` and `/api/v1/meta` using the local Go process.
- Milestone 1 unit/handler tests — passed for OTP login, session revocation, TOTP elevation, organisation tenant boundaries, invitation activation, RBAC, CSRF, audit events, step-up denial, and successful post-MFA invitation.
- Milestone 2 unit/handler tests — passed for target-bound OTP, secure invitation preview, single-use acceptance, buyer portal creation, person/business/authority verification, consent creation, and token-safe access logging paths.
- Milestone 3 unit tests — passed for optimistic draft amendment, stale-version rejection, cancellation/decline terminal states, immutable agreement hashing, mandate-bound buyer acceptance, release/receipt evidence, idempotent activation, balanced ledger postings, base fee arithmetic, and issue receipts that do not activate obligations.
- Milestone 4 unit tests — passed for voluntary and collected allocation, partial payment, duplicate idempotency keys, collection-fee accrual, overpayment rejection, reversal events, and cached-balance rebuild.
- Milestone 5 unit tests — passed for equal six-item generation, monthly month-end capping, custom schedule sum/date validation, grace/overdue state evaluation, early allocation order, schedule-item reversal reopening, and schedule-aware payment allocation.
- Milestone 6 unit tests — passed for concurrent reservation contention, available-limit enforcement, buyer confirmation, activation exposure, outstanding exposure updates, suspension blocking, mandate gating, and resume behavior.
- Milestone 7 unit tests — passed for explicit eligibility reasons, exact reservation amounts, signed webhook processing, duplicate webhook idempotency, partial collection and retry, timeout followed by provider reconciliation, and collection fee integration through the payment ledger.
- Milestone 8 unit tests — passed for partial dispute blocking, evidence and decision history, balanced adjustment postings, full-block behavior, collection eligibility on the undisputed amount, high-value write-off approval, and audited fee waivers.
- Milestone 9 unit tests — passed for critical fallback delivery, notification deduplication, quiet-hour suppression, secure-link generation, signed WhatsApp webhook verification, structured credit/payment command parsing, and rejection of unsupported sensitive-credential commands.
- Milestone 10 unit tests — passed for reconciled receivables summaries, payment-source fee/export projections, factual history without score output, and append-only correction decision history.
- Milestone 11 unit tests — passed for written-approval and pilot-limit gates, sandbox capability reporting, configuration refusal without approval metadata, and provider settlement-state plumbing.
- Milestone 11 race test — `go test -race ./internal/collections` passed.
- Milestone 12 unit tests — passed for readiness evidence/limit evaluation and configurable pilot gates.
- `ruby -e 'require "yaml"; YAML.load_file("api/openapi.yaml")'` — passed.
- `task ci` — requires the Task runner; equivalent shell checks are available under `scripts/ci.sh`.
- Clean PostgreSQL migration and deterministic seed — passed through migration 051; migrations 046 and 047 were independently rolled back and reapplied on isolated databases, and the expanded acceptance seed passed repeated idempotent runs.
- Tagged PostgreSQL integration suite — passed through migration 051, including supplier onboarding, atomic drawdown activation/rollback, payment-claim persistence, trade-line mandate integrity, analytics reconciliation, the persistence contract, and global/per-transaction ledger reconciliation.
- `pnpm --dir web build` — passed with `@sveltejs/adapter-node`.
- Playwright — 60 Chromium checks cover complete financial/product flows,
  protected operations, mobile navigation and reflow, critical accessibility,
  every indexable acquisition route, SEO/social metadata, structured data,
  index/cache privacy, error recovery, sitemap/robots and PWA assets.

## Known limitations

- PostgreSQL/pgx, Goose, River, AWS S3, and OpenAPI generator dependencies are now declared and the Go/TypeScript OpenAPI artifacts are generated. They are not vendored.
- The locked frontend dependency tree, Svelte diagnostics, adapter-node build
  and Chromium suite pass.
- Development may intentionally use deterministic process-local adapters; every
  database-backed runtime selects durable repositories and advertises
  `DomainAggregatesDurable` only for that configuration.
- Migrations execute cleanly through version 052 on PostgreSQL.
- Non-development supports provider-neutral HTTPS connectors for identity,
  mandates, collections and notifications and rejects incomplete/mock setup.
- Real provider integrations remain disabled by feature flags and unresolved product/legal questions.
- Backup/restore and k6 load-smoke scripts are present; signed execution evidence remains deployment-specific and fail-closed in release certification.
- OTLP exporter wiring is implemented and unit-tested, but delivery to a live
  collector and alert routing require deployment infrastructure and were not
  exercised here.
- Optional gosec, staticcheck, OSV, Trivy, and govulncheck checks are skipped
  for local convenience when binaries are unavailable; release validation must
  use `SECURITY_STRICT=1` and therefore fails closed.

## Wave 3 user control and privacy

Wave 3 is complete in repository scope. Users can configure notification
channels, optional categories, and Africa/Lagos quiet hours without disabling
required security or transactional messages. MFA enrollment issues hashed
one-time recovery codes; recovery is enumeration-safe and rate-limited, needs
independent evidence and a separate reviewer, enforces cooling-off and blocks
sensitive actions, and revokes sessions/codes on completion. Seven durable
privacy-rights request types support deadlines, legal holds, restrictions,
dual-control completion, and expiring tenant-bound exports generated from
authoritative stores. The catalog-generated field inventory covers 943 fields.

Browser acceptance now covers the complete Chromium product suite, including notification
preferences, identity-bound privacy submission, enumeration-safe recovery,
protected operations, accessibility and the product-quality gate.

## Next milestone

No repository-owned README gap remains in the current audit. Complete the
external pilot gates: manual assistive-technology/device evidence, legal and
provider approvals, target-environment resilience/security proof, reviewed KPI
targets, support readiness and launch-owner signatures.

## Admin business settings — 2026-09-03

Added `/admin/settings` with 18 business controls, independent approval, scheduled effective dates and immutable history. Values are enforced by workers and financial database boundaries. Fees are recorded on new offers and preserved for existing agreements; public pricing and offer displays use the corresponding rates. Requires migrations 061–063. See `docs/runbooks/business-settings.md` for protected deployment controls and forward-only repair once custom fee terms or policy history exist.

## Product recommendations implemented — 2026-09-03

Seven product and risk recommendations from the full-repository audit are
implemented across code, configuration and recorded product decisions.

**Deemed acceptance.** Pilot policy forbids converting buyer silence into an
activated obligation. Worker scheduling has been removed and the database now
rejects the legacy auto-activation issue reason, so the path remains unreachable
even if stale application code attempts it. Any future enablement requires a new
approved product decision and an authoritative durable implementation.

**Admin surface.** `ADMIN_SURFACES` makes the operations namespace default-deny
in production. This is a reachability control layered over the existing
permission checks, not a replacement for them, and safety controls such as the
dual-control admin change workflow must not be disabled to shrink the surface.

**Provider abstraction.** `tests/contract/provider_contract_test.go` now runs
its assertions over a registry of adapters, and asserts at compile time that the
Mono client satisfies both provider interfaces. A second adapter should be a
one-line registration; if it is not, the boundary leaked.

**Not done.** Building the second collection adapter itself, the buyer-facing
trade-history preview and access log recommended by the DPIA, and the
aggregate-only history disclosure. Each needs product decisions recorded in
`docs/product/open-questions.md` (EXT-010 to EXT-013) before it is worth
writing.

**Verification limits.** Repository-owned checks and the Go/frontend suites are
run from the current working tree. PostgreSQL 18 integration and external
provider, legal and deployment evidence remain release gates and must be
recorded by the certification workflow before merging or enabling collections.
