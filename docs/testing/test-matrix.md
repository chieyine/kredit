# Required test matrix

| Area | Minimum scenarios |
| --- | --- |
| bootstrap | migrations, seed, generated-code drift, config validation |
| auth/access | OTP, MFA, CSRF, session expiry, RBAC, RLS, break-glass |
| agreements | draft amendment/version conflict, cancellation, decline, immutable version, exact acceptance, release/receipt evidence, token replay |
| money | kobo arithmetic, balanced postings, allocations, reversals, rebuild, concurrency, explicit payment sources, confirmed/rejected off-platform claims |
| providers | timeout, mandate cancellation/fresh restoration, partial success, duplicate/out-of-order webhook, reconciliation |
| collections | eligibility, reservation, bounded buyer-claim hold, retry policy, no-over-debit, fee only on success |
| disputes | partial block, evidence, review, adjustment, audit |
| channels | WhatsApp safety, template dedupe, fallback, quiet hours |
| UX | mobile critical flows, responsive navigation, error recovery and WCAG 2.2 AA checks |
| product quality | unique SEO/social metadata, valid JSON-LD, sitemap/robots, index privacy, caching, install assets and public-route accessibility |
| operations | failed jobs/webhooks, controlled retry, reports, backup restore, deployment definitions, alert-rule validation |

Acceptance scenarios A–F from `README.md` are represented by deterministic
seed records plus domain/provider/integration assertions. Browser acceptance
also covers create, amend, cancel, decline, team lifecycle, trade-line
creation/administration, dispute evidence, operations, public, mobile, and
indexing flows. Axe-core blocks serious or critical violations across the
twelve Wave 5 journeys; keyboard/focus, reduced-motion, touch-target and 200%
reflow assertions run alongside it. Live-provider and real
assistive-technology certification remain fail-closed pilot gates.

## Stable acceptance fixture identifiers

These identifiers are permanent. Tests and seed comments should cite them so a
scenario can be traced without relying on a display name.

| Fixture ID | README scenario | Stable primary record | Required assertion |
| --- | --- | --- | --- |
| `FIX-README-A-ONE-TIME` | A — one-time credit | credit request `00000000-0000-7000-8000-000000000101` | exact terms, activation, full payment, zero outstanding, balanced ledger |
| `FIX-README-B-INSTALMENTS` | B — instalments | credit request `00000000-0000-7000-8000-000000000102` | accepted six-part schedule, partial allocation, schedule/outstanding reconciliation |
| `FIX-README-C-TRADE-LINE` | C — recurring trade line | trade line `00000000-0000-7000-8000-000000000040` | drawdowns and repayments reconcile to exposure and available limit |
| `FIX-README-D-MANDATE-CANCEL` | D — mandate cancellation | credit request `00000000-0000-7000-8000-000000000104` | cancellation blocks new release/collection without erasing debt |
| `FIX-README-E-PARTIAL-DISPUTE` | E — partial dispute | credit request `00000000-0000-7000-8000-000000000105` | only the approved disputed amount is blocked or adjusted |
| `FIX-README-F-DUPLICATE-WEBHOOK` | F — duplicate webhook | provider event `scenario-f-success-event` | three deliveries produce one stored event and one financial effect |

## README completion acceptance scenarios

| Scenario ID | Workstream | Minimum acceptance behavior | Required layers |
| --- | --- | --- | --- |
| `ACC-TL-001` | `TL-DRAWDOWN` | reserve → buyer confirm → release → no-issue receipt creates exactly one obligation and schedule | domain, PostgreSQL, API, browser |
| `ACC-TL-002` | `TL-DRAWDOWN` | issue-at-receipt opens/links a dispute and does not activate an obligation | domain, PostgreSQL, API, browser |
| `ACC-TL-003` | `TL-DRAWDOWN` | cancellation/expiry releases reserved capacity without money postings | domain, PostgreSQL, worker |
| `ACC-TL-004` | `TL-DRAWDOWN` | concurrent drawdowns, retries, and duplicate/out-of-order commands cannot exceed the limit or duplicate money | race, PostgreSQL, contract |
| `ACC-TL-005` | `TL-DRAWDOWN` | accepted canonical terms and printable document match the activated obligation | domain, document, browser |
| `ACC-ONB-001` | `SUPPLIER-ONBOARDING` | new owner completes identity, KYB, settlement, billing, consent, and MFA readiness | domain, provider contract, PostgreSQL, browser |
| `ACC-ONB-002` | `SUPPLIER-ONBOARDING` | incomplete/expired readiness blocks only protected actions and explains recovery | permission, API, browser |
| `ACC-ONB-003` | `SUPPLIER-ONBOARDING` | sales and finance roles can access only their permitted steps and data | RBAC, RLS, browser |
| `ACC-NOT-001` | `NOTIFICATION-PREFERENCES` | user changes channel, quiet hours, and optional categories and future delivery obeys them | domain, PostgreSQL, worker, browser |
| `ACC-NOT-002` | `NOTIFICATION-PREFERENCES` | required security/transactional events cannot be disabled and use safe fallback | domain, worker, browser |
| `ACC-REC-001` | `ACCOUNT-RECOVERY` | recovery code plus independent evidence enters cooling-off, revokes sessions, and completes safely | auth, PostgreSQL, browser |
| `ACC-REC-002` | `ACCOUNT-RECOVERY` | phone-only, replayed, rate-limited, conflicted, or self-approved recovery is rejected without enumeration | security, permission, API |
| `ACC-PRV-001` | `PRIVACY-RIGHTS` | identity-bound access/portability request produces a protected, auditable export | domain, PostgreSQL, jobs, browser |
| `ACC-PRV-002` | `PRIVACY-RIGHTS` | deletion/restriction honors financial retention and legal holds while completing allowed actions | domain, PostgreSQL, operations |
| `ACC-DATA-001` | `DATA-INVENTORY` | every persisted column has inventory coverage and new uncovered columns fail CI | schema inspection, CI |
| `ACC-OPS-001` | `OPS-CONTROLS` | controlled job/webhook retry requires permission, reason, MFA, version, and idempotency | permission, PostgreSQL, browser |
| `ACC-OPS-002` | `OPS-CONTROLS` | suspend/restore and scoped risk holds have previewed consequences, notifications, and audit evidence | domain, permission, browser |
| `ACC-OPS-003` | `OPS-CONTROLS` | unknown provider submission can be traced and reconciled without direct database writes | provider contract, PostgreSQL, browser |
| `ACC-UI-001` | `INTERACTIVE-FLOWS` | each README supplier, buyer, and operations command is executable through a permission-aware interface | browser, mobile |
| `ACC-A11Y-001` | `WCAG-AA` | critical flows have no serious/critical automated accessibility violations | browser, CI |
| `ACC-A11Y-002` | `WCAG-AA` | keyboard, screen-reader, zoom/reflow, reduced-motion, and mobile checks have reviewed evidence | manual acceptance |
| `ACC-ANA-001` | `PRODUCT-ANALYTICS` | authoritative funnel events are privacy-minimized, deduplicated, versioned, and reconcile to domain records | domain, PostgreSQL, analytics |
| `ACC-ANA-002` | `PRODUCT-ANALYTICS` | pilot scorecard exposes freshness, definitions, reconciliation state, and approved guardrails | report, browser |
| `ACC-QLT-001` | product quality | every indexable route has unique canonical search/social metadata and valid structured data | browser, CI |
| `ACC-QLT-002` | product quality | public routes have no mobile overflow or serious/critical automated WCAG defects | browser, CI |
| `ACC-QLT-003` | product quality | private/draft routes are noindex, private HTML is no-store, sitemap/robots exclude them, and 404/PWA assets remain valid | browser, CI |
