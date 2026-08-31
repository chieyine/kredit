# Production-v1 certification report

## Current disposition

**Repository-owned README and product-quality implementation is complete;
production is not certified.** Waves 0–6, the interface closure, automated
accessibility gate and world-class product-quality gate are implemented. Human
assistive-technology review and the applicable legal, provider, security,
environment and launch approvals remain outstanding.

Run the complete gate with:

```text
bash scripts/release-certify.sh
```

The command intentionally fails when any local or external release gate is
missing. A green unit test run must never be interpreted as a production
certificate.

## Implemented in the latest product-completion pass

- Added a fail-closed production configuration executable to release
  certification, checking the exact provider, secret, TLS, database, approval
  and pilot-limit configuration the API will load.
- Added runtime-controlled legal activation. Production web startup now
  requires the approved operator, address, contacts, effective date and
  matching document versions. Approved pages become live and indexable;
  incomplete production legal configuration cannot serve.
- Added a go-live runbook for release freeze, staging evidence, migrations,
  DNS/TLS, low-value production proof, monitoring, stop conditions, rollback
  and first-day sign-off.

- Completed the launch-wide code audit across 552 owned files, 168 backend and
  OpenAPI operations, 165 screen-relevant endpoints, 100 long-form guides and
  every production database column. Product-contract, content, repository and
  data-inventory checks are now mandatory CI and release gates.
- Made caller-supplied environment values take priority over local `.env`
  defaults, preventing release checks and guarded database commands from
  silently selecting the wrong database. An executable regression test locks
  this behavior.
- Expanded the database integration gate from a hand-maintained package list
  to every PostgreSQL adapter, with a twice-applied deterministic seed and
  serial package isolation. This closed a previously unexecuted analytics
  scorecard path on clean databases.
- Added a public, no-sign-in interactive demo that takes a visitor through both
  sides of a credit sale—from choosing the amount to acceptance, delivery and
  a recorded payment. It uses sample-only data, plain Nigerian English and a
  direct path into the real product.
- Added atomic PostgreSQL payments spanning schedules, obligation/credit
  balances, ledger postings, fees and outbox events.
- Added restart-safe PostgreSQL trade-line and collection adapters plus
  provider-neutral identity, mandate, collection and notification connectors.
- Completed customer onboarding/directory, live reports/statements, signed
  secure links, same-origin production proxy and adapter-node containers.
- Passed all 51 migrations, every database-backed package, the tagged
  PostgreSQL suite, production API/worker/web builds, structural conformance,
  and all 60 Chromium product, access-control, admin, SEO and accessibility
  scenarios in one uninterrupted run.
- Added durable disputes, corrections, operations, notification delivery,
  analytics and messaging repositories; River notification/document workers;
  transactional outbox dispatch; malware-scanner integration; and a
  role/MFA-gated, privacy-safe operations console.
- Fixed the database-backed first-login stored function discovered by the live
  platform authorization proof. Migrations now pass through version 051,
  including versioned supplier-onboarding readiness, reconciliation, product
  analytics and rollback/reapply evidence.
- Completed recurring trade-line drawdowns with immutable terms, linked issue
  disputes, atomic obligation/schedule/ledger/outbox activation, safe expiry,
  printable agreements, and supplier/buyer browser flows.
- Added durable payment claims and review, signed public payment/receipt
  projections, mandate cancellation/restoration, trade-line limit reduction,
  buyer obligation detail, deployable Kubernetes/OpenTofu definitions, alert
  rules, and top-level contract/E2E/performance harnesses.
- Completed Wave 3 notification preferences, privileged account recovery,
  privacy-rights operations, authoritative protected exports, and catalog
  inventory drift enforcement across all 924 persisted fields.
- Completed Wave 4 protected operations commands and diagnostics with immutable
  command history, controlled retries, suspension/restoration, expiring holds,
  reconciliation, provider-confirmed cancellation, notifications, redaction,
  fresh PostgreSQL evidence, and five dedicated Chromium scenarios.
- Completed Wave 5 permission-aware interfaces for team lifecycle, recurring
  trade-line administration, supplier/buyer disputes, settings and financial
  commands; added amount-specific confirmation labels and provider/conflict
  browser cases.
- Added an axe-core WCAG 2.2 AA gate across twelve critical journeys and
  automated keyboard, skip-link, modal focus containment/restoration,
  reduced-motion, touch-target and 200% reflow checks. Manual device evidence
  is tracked separately and remains release-blocking.
- Added responsive public and portal navigation, safe protected-action dialogs,
  branded error recovery, substantive acquisition/trust/support journeys,
  route-correct SEO/social metadata and JSON-LD, strict index/cache privacy,
  PWA icons/shortcuts, and a browser gate across every indexable route.

## Outstanding gates

- Manual VoiceOver/TalkBack, 400% zoom, forced-colors, and provider-hosted
  accessibility reviews recorded in `wave5-accessibility-evidence.md`.
- k6 performance evidence, backup/restore drills, strict scanner output and
  observability alert-routing tests in the target infrastructure.
- Real provider certification plus signed security, privacy, legal, support,
  backup/restore, and launch approvals.
