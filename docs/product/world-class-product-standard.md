# World-class product standard

Status date: 29 August 2026

This is the durable quality contract for Kredit. A feature is not considered
finished because its happy path renders. It must be understandable, safe,
responsive, accessible, discoverable where appropriate, private where not,
observable, recoverable, and covered at the boundaries it crosses.

## Product bar

| Pillar | Required product behaviour | Repository evidence | Status |
| --- | --- | --- | --- |
| Complete journeys | Supplier, buyer, operations, payment, receipt, invitation, dispute, recovery, privacy, settings and support routes have entry, progress, success, empty, blocked and recoverable-error states. | `web/src/routes`, `web/tests/product-flows.spec.ts`, `web/tests/accessibility.spec.ts` | Implemented |
| Financial clarity | Exact amount, date, fee, grace, authority, evidence and consequence precede material confirmation. Provider uncertainty never masquerades as failure. | `docs/product/interface-copy.md`, financial components and integration tests | Implemented |
| Information architecture | Public, supplier, buyer and operations shells use distinct navigation, current-page state, responsive menus, search where scale requires it, and reachable support. | `SiteHeader.svelte`, `PortalNav.svelte`, `CommandPalette.svelte` | Implemented |
| Visual system | Shared colour, type, spacing, radius, elevation, status and focus tokens; responsive compositions; light/dark schemes; reduced-motion behaviour; consistent cards, forms, actions and feedback. | `web/src/app.css`, shared components, visual browser inspection | Implemented |
| Accessibility | Semantic landmarks, one page heading, skip navigation, labelled inputs, live status/error feedback, keyboard dialogs, focus restoration, 200% reflow, mobile touch sizes and automated WCAG 2.2 AA checks. | `web/tests/accessibility.spec.ts`, `web/tests/product-quality.spec.ts` | Automated scope implemented; real assistive-device sign-off external |
| Error recovery | Branded privacy-safe 404/500 handling, retryable data failures, no duplicate financial submission, offline warning and explicit safe destinations. | `web/src/routes/+error.svelte`, `SystemBanner.svelte`, idempotency tests | Implemented |
| Trust and control | Least privilege, MFA, auditable impact previews, bounded disputes/collection, privacy requests, mandate cancellation, security centre and safe complaint route. | domain tests, `security/+page.svelte`, `ProtectedActionDialog.svelte` | Implemented |
| SEO | Unique titles/descriptions, canonical and language links, route-correct Open Graph/Twitter cards, valid JSON-LD, indexable sitemap, robots controls and branded share artwork. | `web/src/lib/seo.ts`, `sitemap.xml`, `robots.txt`, `og.png`, product-quality browser gate | Implemented |
| Index privacy | Authenticated, tokenised, recovery and draft legal routes are `noindex`; private HTML is `no-store`; public content has bounded shared caching. | root layout, `hooks.server.ts`, browser response tests | Implemented |
| Performance | System fonts, no third-party runtime tags, SSR HTML, precompressed production build, bounded public caching, small shared assets and no sensitive document caching. | SvelteKit adapter/build, service worker, response tests | Implemented |
| Installability | Branded 192/512/Apple/maskable icons, complete manifest, useful shortcuts, theme/background colours and safe offline boundary. | `web/static/manifest.webmanifest`, icons, `service-worker.ts` | Implemented |
| Measurement | Privacy-minimised authoritative events, replay protection, reconciled KPI/driver/guardrail scorecard and explicit baseline-dependent targets. | migrations 050–051, analytics catalog and pilot scorecard | Implemented |
| Reliability and operations | Structured privacy-safe logs, request correlation, metrics/traces, alerts, durable jobs/outbox, provider reconciliation, backups/restores and operator runbooks. | `docs/operations`, `docs/runbooks`, integration and release gates | Implemented in repository scope |
| Quality enforcement | Clean database migration/seed, backend/integration/race/vet/security, frontend diagnostics/build, route/browser/accessibility/SEO checks and fail-closed release approvals. | `scripts/ci.sh`, `scripts/release-certify.sh`, CI workflow | Implemented |

## Design principles

1. Make the next safe action obvious; never hide the consequence.
2. Show facts and provenance instead of an unexplained score.
3. Preserve the user’s place and data when a dependency fails.
4. Use plain language before domain terminology, and explain necessary terms.
5. Design mobile-first for time-constrained operators without weakening
   desktop information density.
6. Do not manufacture testimonials, customer logos, approval badges, security
   claims, launch dates or performance numbers.
7. Keep public projections minimal and authenticated records complete.
8. Every new public route joins the metadata, sitemap, responsive and
   accessibility quality gates before release.

## External evidence that code cannot create

The following remain fail-closed release gates rather than hidden product gaps:

- approved production terms, privacy notice, lawful bases, retention and
  processor disclosures;
- real VoiceOver, TalkBack, forced-colour, 400% zoom and target-device review;
- independent penetration test and security sign-off;
- live identity, mandate, payment, collection, notification and settlement
  provider certification;
- target-environment load, backup/restore, alert, incident and recovery drills;
- approved pilot pricing/limits/targets, support training and launch signatures;
- factual customer proof only after real pilot consent and evidence exist.

Draft legal pages remain useful and reachable but are deliberately `noindex`
and cannot capture production acceptance until approved versions are supplied.

