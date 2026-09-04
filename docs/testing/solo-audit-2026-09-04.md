# Kredit solo audit — 4 September 2026

> Follow-up correction: the original article checks rewarded length and missed repeated boilerplate. The 100-article result below is historical evidence of that inadequate check, not an editorial endorsement. The prelaunch follow-up replaces that library and documents the remaining engineering work in `prelaunch-followup-2026-09-04.md`. The original 76-file scope remains unchanged; it was not an exhaustive manual review.


## Disposition

Kredit has a coherent visual direction and substantial financial safeguards, but **it must not be certified as entirely correct or production-ready**. This audit found additional defects beyond the earlier reports and repaired the concrete issues below. Database tenant isolation remains a verified architectural gap.

No agents or subagents were used. Existing working-tree changes and earlier reports were preserved. The review combines a file-by-file inventory and automated checks with focused source review, live PostgreSQL 18 tests, and browser inspection. The companion register distinguishes those methods; an automated check is not a claim of exhaustive manual review of every line or every business state.

## Repairs in this run

| ID | Problem and consequence | Repair |
| --- | --- | --- |
| S01 | The collection provenance trigger rejected legitimate reversals after collection completed. Returned bank payments could not restore debt. | Migration 080 validates the reversal against its original payment, including amount, tenant, currency, provider, fees and reversal identity, while retaining the original checks for new collections. |
| S02 | Worker credentials could insert/update buyer acknowledgement evidence after setting a buyer context. The prior report's read-only assertion was not enforced by grants. | Migration 079 separates buyer SELECT/INSERT policies, explicitly limits them to the API role, and revokes worker writes and API rewrites. Provisioning preserves these restrictions. |
| S03 | Startup still accepted a migration version below the new financial/security requirements. | Persistence readiness now requires migration 080 and the acknowledgement table. |
| S04 | A failed organization lookup appeared as an empty customer list; refresh failures and overlapping responses could leave misleading state. | Shared lists catch lookup/network/shape errors, retry the whole operation, clear stale records, abort old requests, and reject late responses. |
| S05 | Switching businesses after pagination could show no records despite available data. | Each load starts at the first page. |
| S06 | Collection/overdue links lost the selected organization; customer history searched all organizations and returned the first matching customer. | List links, customer history and repeat-sale links preserve the selected business. |
| S07 | Shared financial lists omitted amounts, and the buyer debt list included unaccepted sales with no obligation. | Cards display exact amounts; the debt list excludes records without an obligation. |
| S08 | Desktop navigation depended on JavaScript opening a hidden disclosure. Before hydration or without JavaScript, only the brand was available. | Desktop navigation is present in the server-rendered markup, with a separate native mobile disclosure sharing the same link snippet. |
| S09 | Sales could create a sale but could not load its customer picker because the directory required financial-read permission. | The directory uses organization-read permission for customer selection, and omits financial totals/counts unless the member also has financial-read permission. Suspended membership remains denied; financial-mutation grants are unchanged. |
| S10 | Payment confirmation promised nothing would be saved if the connection dropped. | Copy instructs the user to check payment history before retrying. |
| S11 | Large money amounts could produce invalid words above the billion scale. | Verbal amount formatting supports trillion and larger integer groups. |
| S12 | Buyer obligation loading/notice acknowledgement could leave a permanent loading/saving state after a network failure. | Explicit error, retry and finally paths restore usable state; displayed schedule dates consistently use Lagos time. |
| S13 | The Node container copied root dependencies but omitted the workspace link needed to resolve external openapi-fetch imports. | The runtime preserves root/workspace node_modules placement and includes a dependency-resolution build check. Docker context also excludes local data and secret-file patterns. |
| S14 | CI requires golangci-lint but did not install it. | The official installation action supplies pinned v2.13.2 before repository checks. See the [official action documentation](https://github.com/golangci/golangci-lint-action). |
| S15 | The dependency registry reported 22 frontend advisories, including six high-severity advisories. | Same-major patches installed for SvelteKit 2.70.2, Svelte 5.55.7, Vite 7.3.5 and Playwright 1.55.1. Explicit transitive overrides patch cookie 0.7.2 and esbuild 0.28.1; final registry audit reports zero advisories. |
| S16 | The privacy inventory omitted new columns and the actual jobs schema, leaving restricted job payloads unclassified. | The generator/checker now include jobs, classify sensitive payload fields, and describe acknowledgement writer restrictions. The regenerated inventory covers 1,083 fields. |
| S17 | Guide components captured initial route data, so a reused page could show the previous article and metadata. New compiler diagnostics also exposed stale navigation IDs and legal dates. | Derived values update when route data or labels change. A related-guide navigation regression checks the new heading and canonical URL. |
| S18 | Go vulnerability scanning found eight reachable advisories: seven in the old toolchain and one in the OTLP HTTP exporter. | The minimum Go version, CI toolchain and API container use 1.26.8; tracing SDK/exporter dependencies are aligned at 1.44.0. |
| S19 | Go lint reported 41 issues, including ignored database-test query errors and an overwritten privacy restriction insert error. | Errors are checked at their source, test environment cleanup uses t.Setenv, redundant unused declarations are removed, and mechanical formatting/simplification fixes are applied. Final lint results appear below. |
| S20 | The web container and CI Node pin predated multiple security updates. | Node 24.20.0 is pinned in the version file, package engine and container stages, based on the official release index and [Node security releases](https://nodejs.org/en/blog/vulnerability/july-2026-security-releases). |

Two older integration assertions also needed correction: notice eligibility now requires buyer acknowledgement, and the admin authorization test must use the rotated session token. The old token is explicitly checked for rejection. A first unseeded database run also exposed a missing analytics fixture; the final run uses the repository's migrate/roles/seed/integration sequence on a separate fresh database.

## Remaining findings

### O01 — Tenant row-level security remains ineffective in financial domains (P2, release blocker)

The permissive runtime-role policies in migrations 029, 031 and 035 allow all rows for kredit_app, regardless of the tenant context. On synthetic data, SET LOCAL ROLE kredit_app and an unrelated organization context still returned **40 obligations from other organizations**. This is a database defence-in-depth failure. It is not evidence that a specific HTTP endpoint presently leaks those rows; application authorization remains a separate boundary.

Removing these policies alone would break the current repositories. Remediation requires consistent transaction-local actor/tenant context, narrow worker capabilities, and negative cross-tenant tests for every affected repository. Documents have a stronger tenant policy, but that does not cover the rest of the financial schema.

### O02 — Request cancellation remains incomplete (P3)

Several current repository interfaces discard request context and create context.Background() transactions, including credit reads/writes and report snapshots. Some paths have explicit timeouts and the pool has server-side statement/lock timeouts, but these do not promptly cancel work after the caller disconnects. Propagating context requires a deliberate interface migration and cancellation tests, not a text replacement across background workers.

### O03 — Browser tests do not prove real provider or full stack behavior

Most account browser tests mock API responses. PostgreSQL integration tests validate a different boundary. Neither substitutes for an authenticated browser journey through the real API, nor live identity, mandate, collection, notification and object-storage certification. The full deployment and production credentials were not exercised.

### O04 — Manual and operational evidence remains incomplete

Automated accessibility and selected desktop/mobile visual inspection do not certify VoiceOver, TalkBack, forced colors or all 400% zoom states. The repository's release checklist also requires approved legal content, provider certification, penetration testing, target-environment load and monitoring evidence, recovery drills, and launch authorization. These remain external evidence requirements.

## Visual assessment

The cream background, cobalt actions, editorial serif headings, restrained borders and orange accents form a recognizable, coherent system. The homepage has clear reading order and distinct sections on desktop and mobile. This run preserves that direction and improves practical visual quality: visible amounts, resilient cards, consistent business navigation, and recoverable error states. Final desktop (1440 pixels) and mobile (390 pixels) homepage/pricing screenshots showed no horizontal overflow. The pricing preview clearly reports unavailable rates because no live API is attached. Screenshots are local evidence, not a claim that every account screen was manually inspected at every viewport.

## Verification

- Fresh PostgreSQL 18 database: all 80 migrations, runtime grants, repeatable seed and the complete integration script passed. The live role regression rejects worker acknowledgement writes.
- A checksum-verified backup restored into a separate empty database. Migration version 80 and runtime grants were checked successfully.
- Financial race checks passed for collections, credit, payments and reports before the final dependency/lint patches.
- Repository integrity, product contract sync (116 explicit frontend calls, 193 backend routes, 192 API operations), frontend coverage (190 routes), and content audit (100 articles, 203,810 words) passed.
- The frontend dependency registry reports zero advisories after the patches.
- All 32 shell scripts passed syntax checks; database-destructive-operation guards, README/implementation-plan conformance and their available self-tests passed.
- Final Go vulnerability scan: no vulnerabilities found.
- Final full Go package tests passed on Go 1.26.8.
- Svelte diagnostics passed with zero errors and zero warnings. The Node adapter production build passed on Node 24.20.0.
- The final seeded PostgreSQL integration rerun passed after the Go changes.
- The final permission-response and cleanup edits passed targeted PostgreSQL package checks for web, ledger, platform operations, relationships, Mono and reports. A frozen offline dependency installation passed.
- Unsuppressed golangci-lint passed with **0 issues**. The final full Go test run against the synthetic database passed; the last rollback-test cleanup was rechecked separately.
- Production-build browser suite: **87 passed, 1 configuration-specific skip**. The skipped legal-activation scenario then passed separately using synthetic company details: **88 distinct scenarios passed across the two configurations**. The main suite includes the seven new workspace/navigation regressions, accessibility, responsive layout, account permissions, financial journeys and SEO checks.

Docker was unavailable, so no container image was built. The complete strict security script was not run: gosec, OSV and Trivy coverage is not claimed; the individual Go and frontend vulnerability checks are reported above. The OpenAPI script ran its structural fallback because a full OpenAPI linter was not installed. The optional dedicated database-login matrix was not configured; SET ROLE tests were exercised. Browser tests primarily mock private APIs. No live provider credentials, production database, deployment, migration rollout or launch approval was used.

## File register

[Per-file scope and fingerprints](solo-audit-2026-09-04-files.csv) lists each owned file, SHA-256 fingerprint, component, review method, relevant finding and verification boundary. Dependency folders, build outputs, caches and temporary evidence are excluded. The register contains 705 files (706 owned files including the register itself). Seventy-six files have focused source review explicitly recorded; other files have applicable automated checks, not an exhaustive manual sign-off. The register excludes its own recursive fingerprint.


## Reproduction and evidence

The final evidence is saved under `.tmp/solo-audit/` in this workspace. Key logs are `go-complete.log`, `integration-verified.log`, `final-boundaries.log`, `rollback-test-final.log`, `lint-result.log`, `svelte-check-final.log`, `build-verified.log`, `browser-production.log`, `browser-legal.log`, `dependency-audit-final.json`, `go-vulnerabilities-final.log`, `data-inventory-verified.log` and `restore.log`.

Earlier attempt logs are retained separately: a pre-upgrade browser baseline was deliberately stopped; browser setup initially lacked the upgraded Chromium revision; disk exhaustion required removal of regenerable project cache; a development reload interrupted one accessibility scan. These attempts are not counted as successful verification. Final browser verification used a stable production build. Dependency and toolchain patches required rebuilding the Go cache.

[Desktop homepage preview](../../.tmp/solo-audit/home-desktop-final.png) · [Mobile homepage preview](../../.tmp/solo-audit/home-mobile-final.png) · [Mobile pricing preview](../../.tmp/solo-audit/pricing-mobile-final.png)
