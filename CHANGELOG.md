# Changelog

## Unreleased — decision register answered in draft

- Drafted a position for every row of the external decision register, and split
  the thirteen questions by who can actually answer them: five are questions for
  Mono, five are the founder's own risk decisions, and three need a Nigerian
  lawyer to review a draft rather than write one.
- Recorded that cross-supplier trade history sharing may fall under the Credit
  Reporting Act 2017, which reserves credit bureau operation to CBN-licensed
  entities. Sharing is disabled for the pilot while that is answered; buyers are
  shown only their own history, which loses nothing and defers the risk.
- Set draft pilot limits, halt thresholds and the operations surface list for a
  ten-supplier pilot at NGN500,000-2,000,000 per sale.
- Added the thirteen questions to put to Mono in writing, with the use-case
  question first: Sweep is written for lenders collecting loans, and Kredit is
  not the lender.

## Unreleased — product recommendations implemented

- Constrained deemed acceptance, the only path where buyer silence creates a
  collectable obligation. The window is now three days rather than one, it never
  applies to a buyer's first trade credit, and it requires an authenticated
  delivery receipt proving the goods-release notice reached the buyer and sat
  with them for the full period. Enforced in `internal/credit` and independently
  by a fail-closed database trigger (migration 070), documented as README
  section 8.3.1 and business rule 21.
- Sent the buyer a durable goods-release notice in the same transaction that
  records the release, stating the deadline and what silence will be taken to
  mean. No such notice was sent before, on any channel.
- Added four pilot scorecard measures that answer questions the existing set did
  not: mandate authorisation drop-off (the buyer proposition), voluntary payment
  share (the collection-fee incentive conflict), activations from buyer silence
  (residual wrongful-debit exposure), and manual touches per activated obligation
  (cost to serve against a one-hundred-basis-point margin).
- Pre-registered falsifiable pilot halt thresholds in
  `docs/product/pilot-kill-thresholds.md`. Targets still wait for evidence;
  stop conditions do not, because a threshold chosen after the data arrives is a
  rationalisation.
- Made the operations surface default-deny in production. `ADMIN_SURFACES`
  enumerates the surfaces a deployment operates; anything unlisted answers 404
  and is logged, and production refuses to start without the enumeration.
- Generalised the provider contract suite over a registry of adapters so a
  second collection adapter is a one-line registration, and added the
  certification and contingency plan for the single-provider dependency.
- Assessed cross-supplier trade history sharing in
  `docs/compliance/dpia-trade-history-sharing.md`, which recommends
  aggregate-only disclosure and records five outstanding decisions; the flow
  stays disabled until they are signed.

## Unreleased — audit recommendations implemented

- Enforced the linters the repository already configured: `scripts/ci.sh` now runs `scripts/lint.sh`, which fails closed in CI when `golangci-lint` is absent, and the workflow installs it. `errcheck`, `staticcheck`, `unused` and `ineffassign` previously never ran.
- Replaced the no-op fuzz gate (`go test ./... -run '^$'` matched nothing) with `scripts/fuzz.sh`, and added fuzz targets for reference parsing, provider webhook parsing past the signature layer, schedule generation, payment allocation, problem-details mapping and phone normalisation.
- Canonicalised telephone identifiers to E.164 so one subscriber is one account, with a fail-closed backfill that refuses to merge duplicates automatically (migration 067).
- Added a per-account MFA attempt lock and a session idle deadline alongside the absolute lifetime (migration 068).
- Gave authentication routes a rate-limit budget shared across API replicas; the in-process limiter alone multiplied the real budget by the replica count and reset on deploy (migration 069).
- Made the payment list report failure instead of returning an empty history during a database incident.
- Stopped serving stale credit aggregates from the process-local projection, and bounded that projection so it no longer grows with every request the process has served.
- Recorded [ADR 0005](docs/adr/0005-hand-written-http-and-sql.md): Go handlers and SQL are hand-written, the OpenAPI document stays the enforced contract, and the orphaned `api/generated`, `db/generated`, `db/queries` and `sqlc` pipeline are removed.
- Documented the fee rounding direction as the contractual term it is, and fixed problem-detail truncation splitting a UTF-8 rune.

## Unreleased — full-repository code audit fixes

- Indexed activated obligations by their own identifier so payment snapshots, collection state, ownership checks and rehydrated views resolve; a request-keyed index previously hid the obligation from every financial reader.
- Removed a self-deadlock in the durable auto-activation sweep and made it activate matured requests deterministically.
- Refused malformed and empty MFA codes and compared them in constant time; an undecodable stored secret previously matched an empty submitted code.
- Removed modulo bias from generated OTP codes and stopped issuing a clock-derived CSRF token when randomness is unavailable.
- Added Origin/Sec-Fetch-Site verification and constant-time token comparison to CSRF checks, and stopped the web proxy forwarding client-supplied forwarded-address headers that allowed rate-limit and OTP-throttle evasion.
- Implemented the documented "last day of month" instalment policy, which previously behaved as "cap" and silently shifted agreed dates.
- Stopped recording free-text release notes as a waybill number in goods-release evidence.
- Guarded a nil collection reservation on the successful-debit path, ordered ledger and payment reads deterministically, released a stuck idempotency reservation when a handler panics, and reported supplier onboarding OTP delivery failures.
- Fixed the Linux CI break in the implementation-plan self-test and reformatted four files so `gofmt -l` is clean.

## Unreleased — admin approvals and operating workflows

- Added a role-scoped approval inbox with ownership and deadlines, independent financial corrections, and buyer-accepted repayment-date amendments.
- Added specialist admin roles, policy impact previews, exact naira/percentage inputs, named actors, searchable change history and CSV exports.
- Added operational attention counts/details and preserved original agreement dates alongside current repayment schedules.
- Added migrations 064–066 and corrected write-offs to update unpaid instalment totals atomically. See docs/runbooks/admin-workflows.md and docs/testing/admin-workflows-2026-09-03.md.

## Unreleased — completion of financial code fixes

- Added consistent financial report snapshots, error-aware financial lists, checked totals and exact money displays.
- Made financial lifecycle notices transactional and required authenticated prior-notice delivery plus waiting time before real debit submission.
- Added durable financial metrics and owned reconciliation reviews, including provider reversal and missing-settlement evidence.
- Bound operations retries to exact intent and validated commands before provider calls; preserved later settlement updates during lookup.
- Added migrations 055–060, connector receipt/API contracts, an operator review page, and transaction/browser regression coverage. See docs/testing/code-completion-2026-09-02.md.

## Unreleased — direct code review fixes

- Closed stale collection eligibility and claim confirmation races; preserved valid payment reversals and forgiveness during balance rebuilds.
- Corrected retry identities, late provider results, cancellation races, notification routing/amounts, recurring job deduplication and paginated discovery.
- Rejected false second-approver assertions and unilateral replacement of accepted schedules; added exact naira parsing, waiver-aware fee reports and truthful repayment history.
- Corrected new drawdown fee disclosures while preserving accepted legacy hashes, OTP cooldowns, one-time credential replay and the web container build target.
- Added migrations 053–054 and focused regression tests. See docs/testing/code-review-2026-09-02.md for verification and remaining launch evidence.

## Unreleased — Mono Sweep sandbox backend

- Added a provider-isolated Variable Sweep adapter and transient customer registration with hosted buyer authorization, authoritative activation and supplier-scoped mandate reuse.
- Separated invoice acceptance from bank authorization and included collection policy in accepted terms.
- Commit debit reservations before network calls; reconcile unknown outcomes by saved reference; apply partial payments once per attempt and serialize manual payments/shared mandate capacity.
- Added bounded delayed retries, async authenticated webhook jobs, due/reconciliation workers, internal notices, immutable collection events and migration 052.
- Restricted persisted identity results to named verification facts; corrected worker lookup/ledger/job permissions and repayment tenant context.
- Defaulted all Sweep/automatic flags off and reject production activation pending actual Mono sandbox certification. Added local contract, concurrency, crash-recovery and revocation tests.


## Unreleased — world-class product-quality closure

- Added a durable product-quality standard spanning complete journeys,
  financial clarity, navigation, visual consistency, accessibility, recovery,
  trust, SEO, index privacy, performance, installability, measurement,
  operations and release enforcement.
- Added route-correct search and social metadata, valid JSON-LD, canonical and
  language links, refreshed sitemap/robots controls, branded PNG/PWA icons,
  install shortcuts, public caching and private no-store responses.
- Added responsive current-page portal navigation, substantive supplier,
  buyer, security and support journeys, safe pre-launch legal boundaries, and
  a branded privacy-safe 404/500 recovery experience.
- Replaced browser prompts for protected job, webhook, privacy and recovery
  actions with accessible reasoned review dialogs and verified impact previews.
- Added browser-enforced metadata uniqueness, structured-data validity,
  responsive overflow, public-route WCAG, indexing, caching, error, sitemap and
  install-asset checks. Production web builds now block CI and certification.

## Wave 6 — product analytics and pilot evidence

- Added migrations 050–051 with versioned, privacy-minimised, deterministic
  product events emitted from authoritative lifecycle writes and backfilled
  safely, including payment mandates, collections, repeat sales and retention.
- Added the protected live pilot scorecard with three primary KPIs, funnel
  drivers, loss/dispute/provider/support/accessibility safety guardrails,
  definitions, filters, freshness and
  zero-tolerance source reconciliation.
- Added the event catalog, KPI measurement contract, compliance inventory
  coverage, replay/reconciliation integration tests and Wave 6 release evidence.

## Unreleased — semantic README audit

- Completed Wave 0 contract lock: accountable workstream and external-decision
  records, fail-closed evidence-backed feature gates, shared state/event/problem
  vocabulary, stable acceptance fixture IDs, frozen financial/trust copy, and
  an executable negative-tested completion-evidence gate.
- Completed Wave 3 user control and privacy: authenticated notification
  preferences with required/optional categories and quiet hours; hashed
  one-time recovery codes, enumeration-safe rate limits, independent review,
  cooling-off and session revocation; seven privacy-rights workflows with
  legal holds, restrictions, dual control and protected authoritative exports;
  and a generated field-level data inventory with schema-drift enforcement.
- Completed Wave 4 protected operations: immutable versioned commands and
  events; recent-MFA, permission, reason, idempotency and impact-preview gates;
  controlled job/webhook replay; user/organization suspension/restoration;
  expiring scoped buyer/supplier holds; reconciliation and provider-safe
  collection retry/cancellation; affected-user notices; redacted bounded
  diagnostics; operator runbooks; and five dedicated browser scenarios. The
  regenerated catalog inventory now covers all 924 persisted fields.
- Completed the repository-owned Wave 5 interface closure: team role and
  access-status controls; mandate-backed trade-line creation, suspension,
  restoration, safe reduction and amount-labelled drawdown actions; complete
  supplier/buyer dispute submission, evidence, decision and status surfaces;
  and permission/error/conflict/provider-state browser coverage.
- Added `@axe-core/playwright` as a blocking WCAG gate over twelve critical
  journeys. Serious and critical violations now fail CI; the companion suite
  verifies skip navigation, focus restoration, modal focus containment,
  reduced motion, touch sizing and 200% reflow. Real VoiceOver/TalkBack and
  target-device sign-off remain an explicit external release gate.
- Added a sequenced README completion implementation plan covering all ten open
  repository workstreams, their dependencies, vertical-slice deliverables,
  acceptance criteria, rollout stages, and external launch gates.
- Corrected the conformance record so structural file/route presence is no
  longer presented as proof that every product workflow is complete; added a
  prioritized repository-owned gap register.
- Made one-time, equal, and custom repayment schedule terms immutable before
  buyer acceptance and generate the activated schedule from those accepted
  terms.
- Added hash-verified printable agreement documents for suppliers and buyers,
  including acceptance, mandate, goods evidence, schedule, support guidance,
  and a browser print/save-as-PDF action.
- Added private invoice upload to credit creation and bound the scanned
  document digest and reference into the canonical agreement.
- Removed client control over trade-line mandate state. Trade-line creation now
  verifies the stored mandate's buyer/business ownership, ACTIVE state, and
  ceiling server-side; migration 045 enforces mandate referential integrity and
  prevents an ACTIVE line without a mandate.
- Completed Wave 1 recurring trade-line drawdowns: immutable exact-term hashes,
  replay-safe reserve/confirm/release/receipt/cancel commands, linked receipt
  disputes, internally created obligations and schedules, and hash-verified
  printable supplier/buyer agreements.
- Consolidated drawdown exposure conversion, credit normalization, balanced
  activation postings, repayment schedule, and outbox events into one
  PostgreSQL transaction; forced-rollback and retry tests prove no partial
  financial state or duplicate obligation can escape.
- Made reservation expiry atomically release line capacity and emit its outbox
  event, and added amount-labelled mobile/desktop supplier and buyer flows.
- Completed Wave 2 supplier onboarding with versioned, server-derived readiness;
  owner email/phone verification; provider-hosted KYB; masked/reference-only
  settlement storage; billing, default-credit-policy, consent, owner-MFA, and
  finance-MFA evidence; explicit review/rejection/resubmission/expiry states;
  and immutable revision/outbox history.
- Added recent-MFA, optimistic-concurrency, role-filtered onboarding APIs and
  functional mobile onboarding, settlement, billing, credit-policy, and team
  invitation interfaces. Financial activity returns precise recovery steps
  until pilot ready, including automated collection eligibility.
- Added clean migration-047 rollback/reapply proof, full tagged PostgreSQL
  verification, and browser acceptance for pilot readiness, sales invitation,
  and incomplete-supplier recovery.

## Unreleased — README code-requirement closure

- Added durable buyer payment claims with supplier confirmation/rejection,
  bounded collection holds, evidence references, atomic payment application,
  explicit payment-source reporting, audit events, and notifications.
- Added signed, expiring, privacy-minimized public payment and receipt
  projections plus buyer and supplier interfaces for the complete workflow.
- Added mandate cancellation and fresh authorization restoration, dependent
  trade-line suspension, safe unused-limit reduction, and buyer obligation
  detail across the domain, API contract, generated clients, and browser UI.
- Added production-shaped Kubernetes/OpenTofu deployment definitions,
  Prometheus alert rules, privacy-filtered OTLP collection, provider contract
  tests, browser acceptance coverage, and performance harnesses.
- Validated all 49 migrations, idempotent seed execution, tagged PostgreSQL
  integration, Go unit/race/vet checks, Svelte diagnostics/build, OpenAPI
  generation, and 24 Chromium product-flow scenarios.

## Unreleased — README conformance audit

- Added an executable README conformance gate covering required living docs,
  task commands, processes, route map, UI primitives, database model, provider
  simulator, PWA boundary, and acceptance fixtures; CI now runs this gate.
- Added the deterministic local provider simulator for identity, mandate,
  collection, reconciliation, notification-idempotency, and document-scanning
  scenarios and wired it into bootstrap, development, Compose, and containers.
- Expanded the deterministic seed with Royal Pharmacy identity/authority data,
  mandates, recurring trade line, README scenarios A–F, and duplicate-webhook
  evidence; clean migration and repeated seed execution now pass.
- Added safe draft amendment with optimistic version checking, supplier
  cancellation, and buyer decline across the contract, generated types,
  backend domain/API, supplier/buyer interfaces, audit history, and tests.
- Declared and pinned `openapi-typescript` so TypeScript contract generation no
  longer silently depends on a global tool.

- Added the README-specified network and payment-mandate/event schema with
  encrypted account-token storage and persistence-contract checks.
- Added executable ledger reconciliation (global and per-transaction), tagged
  database integration coverage, and explicit runbooks for failed jobs,
  webhooks, provider reconciliation, mandates, reversals, disputes, restore,
  and break-glass access.
- Added the complete documented SvelteKit route map, buyer/public/admin
  layouts, reusable financial/accessibility primitives, PWA manifest/service
  worker, and an explicit offline financial-action banner.
- Expanded mutation idempotency coverage to buyer acceptance, release/receipt,
  drawdown confirmation, member changes, scheduling, evidence, and send flows.
- Added a PostgreSQL buyer repository for encrypted invitations, buyer persons,
  businesses, representatives, verification cases, consents, and bank-account
  references, with migration 023 and RLS-safe token lookup functions.
- Added the PostgreSQL credit aggregate boundary and migration 024, including
  restart hydration and tenant-scoped persistence for lifecycle and balance
  mutations.

## Unreleased — Priority 4 persistence contract

- Added a complete PostgreSQL state-table contract and explicit runtime
  durability capabilities.
- Added a PostgreSQL authentication repository with encrypted OTP targets,
  encrypted TOTP secrets, transactionally consumed challenges, session
  revocation, and MFA step-up support.
- Added a PostgreSQL organization repository with tenant-scoped memberships,
  invitations, invitation activation, role changes, UUIDv7 identifiers, and a
  protected organization-count capability for pilot limits.
- Added fail-closed persistence checks to staging API startup, API readiness,
  and worker startup, plus the repository migration runbook.
- Kept production disabled until every state-bearing domain aggregate is backed
  by a tenant-safe, transactionally tested PostgreSQL repository.

## Unreleased — Priority 3 observability and reliability

- Added bounded request counters/duration summaries, protected Prometheus
  metrics, OTLP HTTP tracing with W3C trace propagation, response
  status/latency logging, explicit API timeouts, checksummed backups, isolated
  restore verification, and the observability/SLO contract.

## Unreleased — Priority 2 security hardening

- Added fail-closed production secret/TLS endpoint validation, PII-safe
  structured logs and audit metadata, safe request IDs and problem details,
  security-denial auditing, cross-origin security headers, strict scanner mode,
  and security/privacy governance artifacts.

## Unreleased — durable foundation

- Added a pgx PostgreSQL pool boundary and fail-closed staging/production health checks.
- Replaced raw migration replay with Goose version tracking and River migration support.
- Added River-backed maintenance jobs and worker startup/shutdown handling.
- Added dedicated River financial/provider/collections/notifications/documents/reports/maintenance queues, per-job retry budgets, scheduled ledger reconciliation, provider webhook inbox de-duplication, terminal dead-letter persistence, and provider circuit-breaker health reporting.
- Added generated Go and TypeScript OpenAPI contract artifacts and an `openapi-fetch` client helper.
- Added recovery, rate limiting, request-size limits, CSP/HSTS/security headers, and safer client-IP handling.
- Added document/object-storage, support-case, relationship-consent, idempotency, and River schema foundations.
- Added fail-closed staging/worker schema checks and removed non-development silent fallback to in-memory object storage.
- Added HTTP idempotency replay/conflict persistence, support-case and buyer-consent routes, pilot supplier/buyer/industry/provider controls, and enhanced-review marking.
- Corrected factual repayment history so completed late payments remain classified as late.
- Added executable RLS identity helpers, tenant-scoped PostgreSQL transaction context, a durable PostgreSQL ledger implementation with deferred balance enforcement, transactional outbox primitives, mandatory financial idempotency headers, and exact integer-kobo WhatsApp amount parsing.

All notable user-visible changes will be recorded here.

## Unreleased

- Added a polished public product shell with sticky navigation, conversion
  calls-to-action, responsive footer, refined visual tokens, richer homepage
  storytelling, accessible skip navigation, structured metadata, robots rules,
  and an XML sitemap.
- Replaced the new-credit placeholder with a responsive, validated workflow
  that loads the supplier organisation, enforces required commercial terms,
  preserves integer-kobo API semantics, includes CSRF credentials, and submits
  directly to the versioned credit-request endpoint.
- Replaced supplier, buyer, and operations placeholder pages with a reusable
  responsive workspace surface providing consistent headings, actions,
  highlights, loading/error/empty states, and record presentation.
- Added authenticated organisation selection, session-expiry recovery,
  API-backed record loading, filtering, refresh controls, client pagination,
  and live wiring for credit requests, trade lines, disputes, members, audit
  events, and buyer consents.
- Added branded favicon and maskable PWA artwork plus expanded Playwright
  coverage for public routes, mobile navigation, and private-page indexing.

- Added a fail-closed `release:certify` gate covering code, SQLC drift,
  database contract, browser dependencies, runtime durability, and protected
  approval evidence.
- Added a PostgreSQL relationship-consent repository with buyer/supplier RLS
  policies and wired it into database-backed runtimes.
- Added a PostgreSQL-backed provider-neutral mandate state repository while
  keeping the deterministic mock provider for development.

- Added the pre-milestone project foundation and Milestone 0 repository scaffolding.
- Added Milestone 1 authentication, organisation onboarding, RBAC/MFA step-up, audit events, and the PostgreSQL auth/organisation/RLS schema baseline.
- Added Milestone 2 buyer invitations, buyer portal onboarding, person/business/authority records, consent tracking, and provider-neutral mock identity verification.
- Added Milestone 3 one-time credit lifecycle, immutable agreement hashes, mock mandate authorization, release and receipt evidence, active obligations, balanced activation ledger postings, SQL schema, API routes, and buyer review UI.
- Added Milestone 4 voluntary/collected payments, payment allocations, collection-fee postings, reversal events, reconciliation/balance rebuild, payment API routes, SQL schema, and buyer payment history.
- Added Milestone 5 repayment schedule generation, cadence/month-end policies, grace and overdue states, schedule-aware allocations/reversals, schedule APIs, and schedule SQL schema.
- Added Milestone 6 recurring trade lines, locked drawdown reservations, buyer confirmation, exposure/availability calculations, suspension controls, statements, API routes, and SQL schema.
- Added Milestone 7 mock collection engine, eligibility reason codes, collection reservations, provider submission, signed webhooks, partial success/retry handling, timeout reconciliation, collection-fee integration, API routes, and SQL schema.
- Added Milestone 8 structured disputes, evidence and decisions, partial/full collection blocking, balanced dispute adjustments, audited write-offs/fee waivers, operations history, API routes, and SQL schema.
- Added Milestone 9 channel-neutral notifications, versioned templates, fallback delivery, quiet hours, secure links, signed WhatsApp webhooks, safe command parsing, API routes, and SQL schema.
- Added Milestone 10 reconciled receivables, ageing and fee reports, CSV exports, buyer factual repayment history without scoring, append-only correction requests and decisions, privacy-safe analytics events, API routes, UI views, and SQL schema.
- Added Milestone 11 provider-neutral approved-adapter gates, provider capability/status reporting, settlement/reversal state tracking, signed webhook support through adapters, written-approval configuration checks, sandbox contract tests, provider SQL schema, and the operations enablement runbook. Real collection remains disabled by default.
- Added Milestone 12 production-readiness evidence gates, configurable pilot limits, authenticated readiness and metrics endpoints, backup/restore and k6 load-smoke tooling, optional security/dependency/container scans, race checks, launch checklist, and pilot runbook. Production remains blocked until all gates and durable infrastructure requirements are complete.

### Admin business policies (2026-09-03)
- Added governed admin settings for collection automation, notices, limits, review flags, correction thresholds and fee rates.
- Preserved recorded offer rates across future policy changes and used those rates in journals, fee accounting and disclosures.
- Made ordinary credit receipt activation transactional through schedule creation, and added schema downgrade guards for policy history/custom fee terms.
