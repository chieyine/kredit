# README implementation gap audit

Audit date: 29 August 2026

Implementation sequence: `docs/product/readme-completion-plan.md`.

This register compares the binding requirements in `README.md` with executable
repository evidence. A route, table, or page name alone is not accepted as
proof. “Open” means repository-owned work remains; “external gate” means the
interface and fail-closed control may be complete while protected approval or
deployment evidence is necessarily supplied outside the repository.

## Closed during this audit

| README requirement | Evidence | Result |
| --- | --- | --- |
| 8.4, 19.1, 26.1 — buyer accepts the exact instalment schedule | schedule terms are carried by `credit.CreditRequest`, validated before creation/amendment, serialized into canonical agreement JSON, displayed in buyer review, and used by the activation hook | Closed; unit/runtime tests added |
| 19.4 — generated PDF or printable agreement representation | `internal/agreementdocs`, supplier/buyer agreement-document endpoints, hash verification, print/save-as-PDF browser action | Closed; tamper and content tests added |
| 8.3 — invoice upload in one-time credit | credit-creation UI uploads a scanned private document and binds its SHA-256 digest and reference into exact terms | Closed for the credit-creation flow |
| 23, 26.5 — mandate integrity for trade-line creation | the handler resolves the internal mandate server-side through `app.trade_line_mandate`, verifies buyer/business ownership, ACTIVE state, and an adequate ceiling; migration 045 adds the active-line invariant and foreign key | Closed; domain, mock-provider, PostgreSQL migration, and handler tests pass |
| 8.5, 17.6, 26.4 — recurring trade-line drawdown lifecycle | migration 046, immutable hashes and printable documents, supplier release/buyer receipt evidence, linked receipt disputes, internally created obligations and schedules, balanced ledger/outbox writes, atomic rollback/retry proof, expiry capacity rebuild, and supplier/buyer browser flows | Closed; `TL-DRAWDOWN` evidence manifest reviewed 29 August 2026 |
| 8.1 — complete supplier onboarding and readiness | migration 047, versioned readiness evidence/revisions, verified owner contacts, provider-hosted KYB, masked/reference-only settlement, billing and credit policy, versioned consent, owner/finance MFA rules, operational gates, reconciliation/outbox, permission-aware mobile onboarding/settings/team flows | Closed; `SUPPLIER-ONBOARDING` evidence manifest reviewed 29 August 2026 |
| 21.5 — privileged account recovery | migration 048, hashed one-time codes, uniform rate limiting, independent verified evidence, reviewer separation, 24-hour cooling-off, cancellation, session/code revocation, audit and operations UI | Closed; `ACCOUNT-RECOVERY` evidence manifest reviewed 29 August 2026 |
| 36.4 — data-subject requests | seven identity-bound request types, due dates/history, compliance decisions, legal holds, processing restrictions, dual control, protected authoritative exports and user/operations UI | Closed; `PRIVACY-RIGHTS` evidence manifest reviewed 29 August 2026 |
| 36.3 — field-level data inventory (`DATA-INVENTORY`) | generated 924-field TSV plus catalog drift check covering all application, ledger and job columns | Closed for repository coverage; retention/lawful-basis approval remains an external production gate |
| 30, 8.1 — notification preferences | authenticated versioned preferences, required/optional categories, channel fallback, Africa/Lagos quiet hours, audit receipt, delivery enforcement and supplier/buyer UI | Closed; `NOTIFICATION-PREFERENCES` evidence manifest reviewed 29 August 2026 |
| 33.3–33.5 — operations commands and provider diagnostics | migration 049, immutable command/events, version/reason/idempotency/recent-MFA/permission gates, impact previews, notifications, controlled job/webhook/collection actions, suspension/restoration, expiring scoped holds, reconciliation, redacted bounded diagnostics and operator runbooks | Closed; `OPS-CONTROLS` PostgreSQL and 5-scenario browser evidence reviewed 29 August 2026 |
| 13.4, 45 — complete interactive web flows | team role/status controls, settlement/billing settings, buyer accept/decline, supplier/buyer dispute opening/evidence/status, mandate-backed trade-line creation, drawdown release/receipt/cancel, line suspend/restore/limit/statement, user settings and operations controls | Closed; `INTERACTIVE-FLOWS` browser evidence reviewed 29 August 2026 |
| 40, 45 — automated WCAG 2.2 AA checks | axe-core serious/critical CI gate across 12 critical journeys plus skip/focus/dialog/reduced-motion/touch-target/200%-reflow checks | Repository automation closed; real VoiceOver/TalkBack, 400% zoom, forced-colors, and provider-hosted reviews remain external sign-off gates |
| 42 — product analytics (`PRODUCT-ANALYTICS`) | migrations 050–051, locked privacy-minimised event catalog, transactional authoritative mandate/collection/repeat/retention instrumentation, deterministic replay protection, live KPI scorecard, loss/dispute/provider/support/accessibility guardrails, definitions/freshness/filters and zero-tolerance reconciliation | Closed; Wave 6 PostgreSQL, permission, interface and runbook evidence reviewed 29 August 2026 |

## Open repository-owned requirements

No repository-owned README requirements remain open in this audit. External
release gates below still apply and prevent a production-complete claim.

## External gates (not code shortcuts)

- `WCAG-AA` manual closure: real VoiceOver/TalkBack, 400% zoom,
  forced-colors, and provider-hosted interface evidence assigned in
  `docs/release/wave5-accessibility-evidence.md`;
- approved supplier terms, buyer agreement wording, privacy notice, complaints
  policy, and data-processing agreements;
- written identity, mandate, collection, and settlement provider approval plus
  live certification;
- target-environment backup/restore, load, alert routing, independent security,
  support training, and launch-owner signatures;
- target-device browser validation and legal/compliance approval of retention
  periods and lawful bases.

Until the applicable external gates are closed, the release must remain
fail-closed and must not be represented as production V1 complete.
