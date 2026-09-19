# Verification — 19 September 2026

The user explicitly authorized execution after the earlier source-only audit. No agents were used. All database writes used a newly initialized PostgreSQL cluster under `/private/tmp/kredit-verification-20260919-pg`, on loopback port 55439. No deployed environment, customer data, provider credentials, or real payment provider was used.

## Environment

- Go 1.27.1, PostgreSQL 18.6, Node 24.11.0, pnpm 11.17.0, installed Google Chrome. Final isolated browser checks used the cached Chromium 140 headless shell (revision 1193), matching Playwright 1.55.1.
- Node is below the repository's declared `>=24.20.0 <25` range. Results describe this local environment, not certification on the supported production runtime.
- Commands used clean environments. Application and worker database logins remained separate and restricted. Fixture setup used a separate administrator in the disposable cluster.
- Local logs and synthetic recovery artifacts: `.tmp/verification-20260919/` (ignored by Git).

## Result

All 220 browser cases have passing evidence across the full initial run and targeted corrected reruns, including the five initially gated cases and the new business-name regression. Backend, database integration, race, frontend build/type, metrics, and local restore checks passed under the conditions below. This is local verification, not production certification.

## Completed evidence

| Check | Result | Evidence |
| --- | --- | --- |
| Go unit suite | Passed after correcting two early-collection date fixtures | `go-unit-final.log` |
| Go binaries | Passed | `go-build.log` |
| Ledger replay and schedule regressions | Passed | `audit-regressions.log` |
| Frontend type/Svelte check | Zero errors and warnings | `web-check.log`, `web-check-final.log` |
| Frontend production build, including verification fixes | Passed | `web-build.log`, `web-build-final.log` |
| Fresh migrations, through 156 | Passed | `migrations.log`, `migrations-final.log` |
| Development seed, repeated twice | Passed with administrative migration/fixture ownership | `integration-admin.log` |
| PostgreSQL integration suite | All packages passed across the corrected suite and final reports-package rerun | `integration-final.log`, `reports-final.log` |
| Go race detector | No races reported; reports package was rerun after a fixture import changed during compilation | `go-race-final.log`, `reports-race-final.log` |
| Financial metrics and tenant isolation | Passed; 19 distinct nonnegative metrics, no unscoped obligation access, recovery function not PUBLIC-executable | `financial-metrics.log`, `metric-permissions.log` |
| RLS policy shape | Matches existing baseline; 14 listed tables remain pending stricter policy conversion | `rls-shape.log` |
| Recovery helper unit checks | 27 passed | `recovery-unit.log` |
| Compressed backup and isolated restore | Passed checksum, complete data/security fingerprint comparison, runtime grants, and balanced-ledger checks | `backup.log`, `restore.log`, `source-fingerprint.json` |
| Browser suite and corrected reruns | 169 initial passes, then 42 corrected rerun passes; the last three failures passed in the isolated final run | `browser-suite.log`, `browser-rerun.log`, `browser-final.log` |
| New business-name regression | Passed | `browser-final.log` |
| Real API browser journeys | All three passed: supplier payment reads, persisted buyer sales, frontend/API readiness | `browser-final.log` |
| Published content rendering | Both gated cases passed: guide publication, listings/feed/metadata, contact email links and mobile layout | `browser-publication.log` |
| Real API startup | Health and readiness both HTTP 200 against a disposable database clone | `local-api.log`, `api-health.json`, `api-readiness.json` |

The browser results combine the initial suite with targeted reruns; they are not a single clean full-suite run. The first pass used an API-unconfigured production preview and older fixtures. Corrected reruns used the final build and the isolated API. Two remaining Axe scans timed out in system Chrome but passed unchanged with matching Chromium and one worker. Mobile and desktop buyer screenshots were captured, and the mobile result was visually inspected (`buyer-mobile.png`, `buyer-desktop.png`).

The restore verifier first rejected deliberately unbalanced integration-test fixtures, as intended. The successful drill used a separate, freshly migrated and seeded, quiescent source database. It proves local logical restoration; it does not prove offsite R2 delivery, production point-in-time recovery, or production RPO/RTO.

## Corrections made during verification

- Credit tests now specify collection dates on or after their agreed payment dates.
- Database fixtures now provide the required consent versions, operator authority, tenant context, explicit connector adapter, and worker role. Unscoped application reconciliation remains rejected.
- New regressions cover reordered ledger replay without losing posting multiplicity and pre-agreement schedule validation.
- Business settings now fall back to the legal name when the optional display name is absent; previously the strict string decoder threw before reaching the fallback.
- The overview referral and sale-progress verification links have larger touch targets.
- An empty advanced-sale amount no longer displays the parser sentinel as a negative one-kobo charge.
- Browser fixtures now supply explicit payment timestamps, consent notices, saved payment defaults, bank lists, current navigation, access-change confirmation, complete mutation responses, and required business type. Accessibility fixtures wait for the intended forms to load.
- The financial-review browser journey now also releases and reclaims ownership, checking that handover leaves the discrepancy unchanged.

## Open boundaries

- An ordinary, non-bypass migration owner cannot seed the current schema: `app.record_domain_activity()` fails the forced RLS policy on `app.audit_events`. Administrative ownership succeeds. This setup difference remains unresolved and must not be disguised by granting bypass privileges to the application or worker.
- PostgreSQL 18 refuses to demote its bootstrap superuser. The development login script's `demote_owner=true` fails if `kredit` is that bootstrap role. The verification cluster used a distinct bootstrap/fixture administrator instead.
- `promtool` and a Docker daemon were unavailable. Prometheus alert execution/delivery has not been demonstrated.
- Live provider behavior, production deployment, historical customer corrections, and sessions with real users remain outside this local verification.

## Cleanup

The verification API, browser previews, editorial fixture and temporary PostgreSQL cluster were stopped. Synthetic databases, logs, backup and screenshots were retained for inspection. `cleanup.log` records the PostgreSQL shutdown; `local-api.log` records the API shutdown. No production configuration or deployment was changed. Final `git diff --check` passed.
