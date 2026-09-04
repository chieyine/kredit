# Kredit file-by-file code audit — 4 September 2026

## Disposition

**The current working tree builds and its available automated suites are largely green, but it is not safe to call perfect or production-ready.** This audit records **22 actionable findings: 7 P1, 10 P2, and 5 P3**. The most serious defects are not cosmetic: supplier users can assert a worker-only collection payment source, buyers can over-block collection with nominal disputes, suppliers can adjudicate buyer disputes, a compliance role inherits destructive operations, and two external callback trust decisions can create financial evidence without an authoritative second check.

This was a **code audit**, not an audit of agents or prior work. Application behavior, database constraints, role boundaries, API contracts, provider handoffs, frontend routes, SEO, UI/accessibility code, product policy, deployment configuration, tests, and documentation were inspected as repository artifacts. No application code or configuration was changed. This report and its accompanying register are the only durable audit outputs.

The [file register](</Users/macbookpro/Documents/Kredit.com/docs/testing/code-audit-2026-09-04-files.csv>) accounts for **687 repository-owned files** in the audited snapshot, with path, section, line count, review method, status, linked finding IDs, and SHA-256. Dependencies, Git internals, caches, generated build output, Playwright output, and `web/node_modules` are excluded. A blank finding column means that no separate actionable finding was recorded under the stated review method; it is not a proof that every possible runtime path is defect-free.

## Scope by section

| Section | Files | Lines | Outcome |
| --- | ---: | ---: | --- |
| Backend services and commands | 239 | 42,769 | Builds/tests pass; authorization, payment-source, dispute, mandate, MFA, integration, and cancellation issues remain |
| Frontend, UI/UX, accessibility, SEO, and browser tests | 192 | 6,958 | Svelte check/build and 80 browser scenarios pass; legal sitemap/indexing conflict remains |
| Database, migrations, generated queries, and persistence | 106 | 9,150 | Static constraint/RLS/query review complete; PostgreSQL 18 execution unavailable locally |
| Product, legal, release, and technical documentation | 73 | 13,531 | Product-policy and certification drift found; legal/provider approvals remain external gates |
| DevOps, infrastructure, release, and scripts | 55 | 2,759 | Syntax and most structural checks pass; mutable scanner installs and repository-integrity failures remain |
| Dependencies and root configuration | 19 | 2,409 | Build inputs reviewed; online vulnerability intelligence unavailable |
| API contract and generated clients | 3 | 5,363 | 116 frontend calls, 191 backend routes, and 190 OpenAPI operations agree structurally |

Counts were taken before adding this report and register.

## Findings index

| ID | Priority | Section | Finding |
| --- | --- | --- | --- |
| K01 | P1 | Backend / financial integrity | Supplier users can forge worker-only collection payments |
| K02 | P1 | Authorization / product | Supplier finance users can adjudicate buyer disputes |
| K03 | P1 | Financial logic | Nominal or overlapping disputes can suppress the full collectible balance |
| K04 | P1 | Platform authorization | Compliance reviewers inherit destructive operations commands |
| K05 | P2 | Platform privacy | Support access is global rather than case-bound |
| K06 | P2 | Database security | Runtime-role policies nullify tenant RLS isolation |
| K07 | P1 | Documents / availability | Abandoned upload slots can starve the scanning queue |
| K08 | P2 | Organization authorization | Read-only members can create collection-blocking disputes |
| K09 | P2 | Authentication | Sensitive organization mutations accept stale MFA elevation |
| K10 | P2 | Authentication | MFA elevation does not rotate the bearer session |
| K11 | P2 | Authentication | TOTP codes can be replayed within the accepted time window |
| K12 | P2 | Cryptography / operations | One unversioned key spans unrelated security domains |
| K13 | P1 | Collections / integrations | Generic collection callbacks post money before authoritative reconciliation |
| K14 | P1 | Notifications / financial evidence | Connector receipts alone can authorize debit and deemed-acceptance evidence |
| K15 | P2 | Mandates / trade lines | A trade line can verify a mandate belonging to another supplier |
| K16 | P2 | Identity integration | Production identity results have no callback or polling completion path |
| K17 | P2 | Product / worker | Deemed acceptance is simultaneously forbidden, scheduled, and non-durable |
| K18 | P3 | SEO | Approved legal URLs are put in the sitemap while always declaring `noindex` |
| K19 | P3 | Backend / database reliability | Request cancellation is discarded by many repository operations |
| K20 | P3 | CI / supply chain | Security tools are installed from mutable `@latest` versions |
| K21 | P3 | Product / release documentation | Certification and implementation records describe an obsolete repository state |
| K22 | P3 | Repository hygiene | Repository integrity checks fail on CRLF data and trailing whitespace |

## P1 findings

### K01 — Supplier users can forge worker-only collection payments

`recordPayment` accepts `source_type`, provider, and provider reference from a supplier Finance or Collections user and passes them to the payment store. Selecting `kredit_collection` bypasses the database's unresolved-debit reservation guard, reduces authoritative outstanding, posts collection-settlement ledger entries, and may accrue a collection fee without a verified collection attempt. Remove `kredit_collection` from the supplier API and derive all collection facts from an authenticated, reconciled attempt.

Evidence: `internal/web/credit_handlers.go:517`, `internal/payments/postgres.go:66`, `db/migrations/052_sweep_collection_safety.sql:48`, `internal/payments/postgres.go:142`.

### K02 — Supplier finance users can adjudicate buyer disputes

The supplier-facing decision endpoint grants an interested Finance or Collections member authority to set the remaining disputed amount to zero. That resolves the buyer's dispute and removes the database collection block. Supplier users should submit evidence or a proposed response; a fresh-MFA, conflict-checked independent reviewer should own financial adjudication.

Evidence: `internal/web/credit_handlers.go:1488`, `internal/disputes/postgres.go:84`, `db/migrations/054_collection_eligibility_lock.sql:13`.

### K03 — Disputes can over-block collection

The buyer controls `collection_effect`. A one-kobo `FULL_BLOCK` dispute blocks the whole outstanding balance, while multiple overlapping `CONTESTED_ONLY` disputes are summed until the cap reaches the whole balance. Derive the effect from server policy, conserve aggregate active disputed principal under the obligation lock, and require review for full-balance blocks.

Evidence: `internal/web/credit_handlers.go:1369`, `internal/disputes/postgres.go:30`, `internal/disputes/postgres.go:215`, `internal/collections/engine.go:236`.

### K04 — Compliance reviewers inherit destructive commands

`PermissionProviderOperations` is granted to compliance reviewers, and the shared command endpoint uses it for user/organization suspension, collection retry/cancellation, job retry, webhook retry, and reconciliation. Only risk-hold commands receive an additional permission check. Split these into command-specific permissions and authorize the resolved command before external effects.

Evidence: `internal/access/roles.go:84`, `internal/web/platform_operations_handlers.go:175`, `internal/platformops/controls.go:235`.

### K07 — Abandoned upload slots can starve document scanning

Creating an upload slot immediately persists a `PENDING` document before an object exists. The worker repeatedly selects the oldest 100 pending rows, with no upload-completed predicate, lease, cursor, expiry, retry cap, or quarantine transition. A user can keep legitimate later documents from ever being scanned. Persist upload completion, expire abandoned slots, lease/cursor work, bound retries, and add user/tenant quotas.

Evidence: `internal/web/document_upload_slot_handlers.go:53`, `internal/documents/store.go:127`, `internal/documents/scanner.go:101`, `cmd/worker/main.go:264`.

### K13 — Collection callbacks post money before reconciliation

A validly signed generic callback directly records a collected payment and finalizes the attempt from callback fields. Theft of the callback secret or compromise of that connector can therefore create false collected state; finalized attempts fall out of normal pending reconciliation. The Mono path already demonstrates the safer design: callbacks should be signals followed by a server-to-server provider lookup.

Evidence: `internal/web/credit_handlers.go:1327`, `internal/collections/engine.go:345`, `internal/collections/engine.go:440`, `internal/collections/webhook_provider.go:71`.

### K14 — Notification receipts alone authorize financial evidence

An email/SMS/WhatsApp connector's HMAC receipt becomes immutable delivery evidence consumed by the prior-debit and deemed-acceptance database guards. A compromised connector already knows the notification and message identifiers it needs. One provider assertion should not be sufficient to create silence-based debt activation or debit authority; require an independent proof, acknowledgement, or manual review for this financially material evidence.

Evidence: `internal/web/notification_receipt_handlers.go:13`, `internal/notifications/receipts.go:20`, `db/migrations/057_collection_notice_delivery.sql:18`, `db/migrations/070_deemed_acceptance_evidence.sql:69`.

## P2 findings

### K05–K06 — Platform scope and database isolation

- Support roles can search global customer, organization, balance, reference, and case data without an assigned case or audited break-glass grant (`internal/web/platform_operations_handlers.go:293`, `internal/platformops/store.go:207`). This contradicts the repository threat model's case-bound support requirement.
- `kredit_app` and `kredit_worker` receive broad table grants, and later permissive `current_user` policies admit all tenant rows (`infra/postgres/roles.sql:19`, `db/migrations/031_atomic_payment_repository.sql:21`, `db/migrations/048_user_control_and_privacy.sql:150`). Application checks are therefore the only effective tenant boundary on these tables despite RLS being documented as defense in depth.

### K08 — Read-only members can mutate disputes

`RoleViewer` has `PermissionReadFinancial`; the dispute creation and evidence POST handlers authorize with that read permission. A viewer can create a `FULL_BLOCK` dispute and alter collection eligibility. Require a mutation permission and add explicit viewer-denial tests.

Evidence: `internal/access/roles.go:133`, `internal/web/credit_handlers.go:1341`, `internal/web/credit_handlers.go:1460`.

### K09–K12 — Authentication and cryptographic lifecycle

- Sensitive organization permissions check only `AAL2`, not `MFAVerifiedAt`, so an elevated session remains usable through a 14-day idle/30-day absolute lifetime (`internal/web/organization_handlers.go:204`, `internal/auth/store.go:275`).
- TOTP verification upgrades the current bearer token in place. A copied AAL1 token becomes AAL2 when the victim verifies MFA (`internal/web/auth_handlers.go:134`, `internal/auth/postgres.go:480`). Rotate the token and CSRF state atomically.
- TOTP validation does not persist or compare the last accepted counter even though `last_used_at` exists, permitting same-window code replay (`internal/auth/postgres.go:447`, `db/migrations/002_milestone1_auth_org.sql:67`).
- `TOKEN_HASH_KEY` is the effective root for sessions, OTPs, encrypted MFA/contact data, recovery/invitation capabilities, public links, notification links, and WhatsApp signatures. The separately required OTP and field-encryption settings have no runtime consumers, and there is no versioned/dual-read rotation path (`internal/config/config.go:128`, `internal/web/runtime.go:404`, `internal/web/runtime.go:474`).

### K15 — Mandate supplier scope is omitted

Trade-line mandate resolution proves the mandate and buyer identities but does not compare the mandate's supplier organization with the supplier creating the line. The line can be represented as mandate-active and goods may be released before a later collection trigger rejects the mismatch. Include supplier ID in resolution and enforce it when the line is inserted.

Evidence: `internal/web/credit_handlers.go:754`, `internal/mandates/provider.go:267`, `db/migrations/045_trade_line_mandate_integrity.sql:11`, `db/migrations/052_sweep_collection_safety.sql:29`.

### K16 — Identity verification cannot complete asynchronously

The README advertises `POST /v1/webhooks/identity/{provider}`, and the provider interface implements both `GetVerification` and `VerifyWebhook`, but the server registers no identity callback route and no production caller polls `GetVerification`. A provider response that is initially pending has no repository-owned path to update onboarding. Register an authenticated callback/inbox/worker flow or add durable polling and idempotent state application.

Evidence: `README.md:1832`, `internal/identity/provider.go:74`, `internal/identity/webhook.go:65`, `internal/web/server.go:329`.

### K17 — Deemed acceptance is contradictory and non-durable

Pilot policy explicitly says automatic activation is not wired and is unreachable, yet the worker calls it at startup and every maintenance interval. The PostgreSQL implementation then selects candidates only from a process-local cache instead of the database, so behavior depends on cache residency and fresh workers miss durable pending records. Remove the scheduling for the pilot or place it behind an explicit disabled feature gate; if approved later, query and lock eligible database rows authoritatively.

Evidence: `docs/product/open-questions.md:51`, `docs/product/open-questions.md:289`, `cmd/worker/main.go:170`, `internal/credit/postgres.go:994`, `IMPLEMENTATION_STATUS.md:466`.

## P3 findings

### K18 — Sitemap and robots directives conflict

When legal approval is active, `/legal/privacy` and `/legal/terms` are added to the sitemap, but both paths remain in `nonIndexablePaths`, so their pages always emit `noindex,nofollow`. The production-legal browser scenario is skipped without approval data, and the default sitemap test expects both URLs to be absent. Make the indexability rule use the same legal-activation state as sitemap generation and test the approved configuration.

Evidence: `web/src/lib/seo.ts:13`, `web/src/lib/seo.ts:86`, `web/src/routes/sitemap.xml/+server.ts:6`, `web/src/routes/+layout.svelte:22`.

### K19 — Database work often outlives request cancellation

Many production repositories replace the incoming request context with `context.Background()` or expose mutation interfaces with no context parameter. Client disconnects and request deadlines therefore cannot stop those database operations; work continues until the database statement timeout. Carry request/worker contexts through service and repository boundaries, retaining background contexts only for intentional detached work.

Examples: `internal/corrections/postgres.go:30`, `internal/ledger/postgres.go:104`, `internal/payments/postgres.go:40`, `internal/schedules/store.go:591`, `internal/tradelines/postgres.go:58`.

### K20–K22 — Reproducibility and repository hygiene

- CI installs `govulncheck`, `gosec`, `staticcheck`, and `osv-scanner` using mutable `@latest` versions. Pin reviewed versions and update them deliberately (`.github/workflows/ci.yml:47`).
- The certification report still claims 552 files, 51 migrations, and 60 browser scenarios; the audited snapshot contains 687 files, 72 numbered migrations, and 81 declared browser scenarios. `IMPLEMENTATION_STATUS.md` also says the worker does not call deemed acceptance when it now does. Treat these documents as generated release evidence or require a drift check (`docs/release/certification-report.md:21`, `IMPLEMENTATION_STATUS.md:466`).
- `node scripts/repository-audit.mjs` fails because `docs/testing/code-audit-2026-09-03-files.csv` contains CRLF/stray carriage returns. `git diff --check` also reports an extra blank line at `infra/monitoring/prometheus-rules.yaml:69`. Normalize the CSV and remove the whitespace error.

## Section conclusions

### Frontend, UI/UX, design, and accessibility

The frontend type/semantic check and production build pass with zero warnings. The complete Chromium run passed 80 scenarios; one production-legal scenario was skipped because approval values were not present. Automated axe coverage found no serious or critical violations on the tested journeys, and keyboard focus, error-summary focus, 200% reflow, reduced motion, touch targets, offline behavior, mobile navigation, and private-page indexing tests pass. Component styling is centralized and journeys are coherent. The remaining code-level frontend defect is K18. Manual VoiceOver/TalkBack, 400% zoom, forced-colors, and provider-hosted accessibility evidence remain release gates, so accessibility cannot yet be certified.

### Backend and API

Go formatting and vet pass. The package suite passes except for three listener-based tests that the managed sandbox cannot execute because binding local ports is prohibited; no code assertion failed in those packages before the bind. The API/frontend/OpenAPI method-path contract agrees. The P1/P2 findings above show that compile success does not establish safe authorization or financial semantics.

### Database and financial integrity

Migrations, grants, policies, functions, constraints, and repository queries were cross-referenced file by file. Atomic payment/reversal logic, idempotency, obligation locking, reservation logic, and ledger balancing are substantial strengths. PostgreSQL 18 migration and integration execution was not repeated because the available local server is PostgreSQL 14; database conclusions in this report are source/schema traces rather than a live PostgreSQL 18 proof.

### Security and integrations

The completed [Codex Security report](</private/var/folders/c_/kx1_v4v57lvf9sclw0sgtzwm0000gn/T/codex-security-scans-TkEMbA/Kredit.com/14b4c5cb16ac4414ec54147d2ece0368af43ca5f_20260904T105533Z_qs3e3ugg/report.md>) contains the canonical threat model, 15 validated security findings, fingerprints, and remediation details. No evidence-backed SQL injection, stored/reflected XSS, open redirect, hardcoded production credential, unauthenticated webhook forgery, or cross-tenant object-reference path was found in the reviewed surfaces. External providers, object-store policy, scanner internals, KMS, egress policy, and deployed role identities were not independently verified.

### Product, SEO, and release

Public content coverage is unusually complete: 100 guide articles and 203,810 blog words pass structural checks, and route/navigation metadata is broadly coherent. K17 and K18 are real contract conflicts. Separately, production remains intentionally uncertified until legal, privacy, security, provider, backup/restore, performance, monitoring, and launch approvals/evidence are completed.

## Validation record

| Check | Result | Limitation |
| --- | --- | --- |
| Go format and shell syntax | Pass | Style/syntax only |
| `go vet ./...` | Pass | Static analysis, not execution |
| `go test ./...` | All non-listener packages pass | Three `httptest` listener tests were blocked by sandbox `EPERM` |
| Focused Go race run | Pass: auth, access, credit, disputes, collections, documents, notifications, platform operations | Not every package was race-tested |
| Svelte check | Pass: 0 errors, 0 warnings | Does not prove API runtime values |
| Production frontend build | Pass: 455 modules, Vercel adapter | No deployed environment |
| Playwright Chromium suite | 80 passed, 1 skipped | Production legal activation lacked signed approval data |
| Product/API contract checks | Pass: 116 frontend calls, 191 routes, 190 OpenAPI operations | Structural method/path agreement only |
| Frontend API coverage | Pass: 188 routes; 16 intentionally screenless | Accounting rather than behavior proof |
| Content audit | Pass: 100 articles, 203,810 words | No legal or factual certification |
| Repository audit | Fail | K22: legacy CSV line endings and monitoring whitespace |
| `git diff --check` | Fail | K22: monitoring file blank-line error |
| Dependency vulnerability databases | Not run | No online scanner/database access in this environment |
| PostgreSQL 18 integration | Not run | Local PostgreSQL is 14.20 |

## Release recommendation

Do not certify production or enable real collections until all P1 findings are fixed and exercised against PostgreSQL 18 with provider reconciliation evidence. P2 authorization, MFA, mandate, identity, and product-policy issues should also be closed before pilot money movement. P3 items should be completed before calling the repository clean and reproducible. After fixes, rerun the same file register, security scan, database integration gate, race subset, full Chromium suite, and external release evidence checklist.
