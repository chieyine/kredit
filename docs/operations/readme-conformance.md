# README conformance audit

Re-audited 29 August 2026 across `internal/`, `cmd/`, `db/`, `infra/`,
`scripts/`, `api/`, and `web/src/`.

Status: **repository-owned requirements conformant**. External legal,
provider, manual-accessibility, security, environment-exercise and launch
evidence remains fail-closed in `docs/release/readiness-checklist.md`.

| README area | Evidence | Result |
| --- | --- | --- |
| Repository/toolchain/CI | `Taskfile.yml`, pinned Go/Node/pnpm, `scripts/ci.sh`, OpenAPI checks | Implemented |
| Executable README contract | `scripts/readme-conformance.sh`, `task readme:check`, CI invocation | Implemented |
| Local provider simulation | `cmd/provider-simulator`, Compose/bootstrap/dev wiring, contract test | Implemented for all provider-neutral connector families |
| Demo and acceptance dataset | verified supplier/buyer, mandates, trade line, durable scenarios A–F in `db/seeds/001_demo.sql` | Implemented and idempotent on clean PostgreSQL |
| API safety | middleware, CSRF, rate/body limits, problem details, idempotency replay/conflict | Implemented |
| Auth and organisations | PostgreSQL adapters, migrations 020–021, RLS context, MFA/OTP tests | Implemented |
| Buyer identity and provider abstraction | buyer store, PostgreSQL buyer adapter, identity mock, invitation/portal handlers and routes | Implemented with mock provider; real providers externally gated |
| Credit, payments, schedules, collections, disputes | PostgreSQL repositories, balanced ledger, API handlers, unit/race/integration tests | Core paths implemented; accepted schedule terms corrected in the 29 August audit |
| Trade lines | migration 046, PostgreSQL row locking and atomic activation, immutable drawdown hashes/documents, release/receipt/dispute evidence, obligation/schedule/ledger/outbox creation, expiry worker, supplier/buyer interfaces, integration and browser tests | Implemented |
| Supplier onboarding | migration 047, server-derived versioned readiness, verified contacts, provider KYB states, masked settlement references, billing/policy/consent evidence, owner/finance MFA rules, worker expiry, operational gates, mobile onboarding/settings/team flows | Implemented; live provider approval remains an external gate |
| User communications and recovery | migration 048, versioned preferences, mandatory/optional categories, quiet hours/fallback, hashed recovery codes, uniform rate limits, independent evidence/review, cooling-off, cancellation and revocation | Implemented |
| Privacy rights and data inventory | seven request types, legal holds/restrictions, dual-control completion, tenant-bound authoritative exports, compliance UI, 924-field generated catalog register and drift check | Implemented in repository scope; legal approval remains external |
| Payment claims and payment-source truth | migration 044, `internal/paymentclaims`, atomic confirmation, bounded collection holds, supplier/buyer interfaces | Implemented and PostgreSQL-tested |
| Network and mandates | migrations 022/025/045, RLS policies, ciphertext account-token column, `mandates.PostgresProvider`, server-side trade-line mandate ownership/state/ceiling verification, contract checks | Runtime state persistence and trade-line authorization integrity implemented with mock provider; real provider certification externally gated |
| Mandate customer controls | provider cancel/fresh-restore contract, persisted credit snapshots, dependent trade-line suspension, buyer interface | Implemented; restoration never reactivates the cancelled authorization |
| Relationship consent persistence | `relationships.PostgresStore`, tenant context, RLS policies, migration 015 | Implemented |
| Jobs/outbox/reconciliation | River queues, inbox/dead letters, outbox dispatcher, delivery/document workers, `cmd/reconcile` | Implemented with crash recovery and integration proof |
| Reports/corrections/privacy | live reconciled dashboards/CSV, customer history/statements, durable corrections, versioned privacy-minimised product events and authoritative pilot KPI scorecard | Implemented |
| Documents/support/schedules | private object storage, authenticated scanner connector, durable scan discovery, case timelines, PostgreSQL schedules | Implemented; production scanner credentials are fail-closed |
| Frontend route map | supplier/buyer/admin layouts, dynamic routes, reusable workspace states, functional onboarding/settings/team, buyer review/decline/accept, disputes, trade lines/drawdowns and protected operations flows | Implemented; target-device evidence remains an external release gate |
| UI primitives/accessibility/PWA | reusable financial primitives, tokens/focus/reduced-motion/touch styles, manifest, service worker, offline banner | Implemented |
| Public product experience and SEO | shared responsive header/footer, conversion hierarchy, signed minimal payment/receipt projections, structured metadata, robots, sitemap, social metadata | Implemented; production analytics validation is deployment evidence |
| Infrastructure and monitoring | Kubernetes/OpenTofu API/worker/web stack, autoscaling, disruption/network controls, ingress/TLS, Prometheus rules, privacy-filtered OTLP collector | Implemented; target-environment provisioning and alert routing remain external |
| Operations/runbooks | controlled versioned commands, immutable events, suspension/restoration, scoped expiring holds, provider-safe replay/reconciliation/cancellation, redacted bounded diagnostics, and procedure-specific runbooks | Implemented |
| Generated SQLC artifacts | pinned Go fallback in `scripts/sqlc-generate.sh`, drift checker in `scripts/sqlc-check.sh`, checked `db/generated` output | Implemented |
| Browser/integration/performance verification | 46 Playwright scenarios, tagged DB integration, provider contract tests, k6/SQL performance harnesses | Chromium and PostgreSQL suites passed; load evidence remains environment-dependent |
| External provider/legal/security gates | provider approvals, credentials, live certification, signed launch evidence | Intentionally external release gates |

The API and worker remain fail-closed when the persistence contract,
scanner/provider connectors, or external release evidence is incomplete.
