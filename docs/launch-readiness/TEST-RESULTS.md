# Executed local verification

Environment: macOS Intel; Node 24.11.0 (below the manifest's >=24.20 requirement),
pnpm 11.17.0, Go 1.27.1, PostgreSQL 18. Synthetic database on 127.0.0.1:55432.
No production environment file was loaded. Chromium executable was the installed
`chromium-1234` Google Chrome for Testing binary. Physical devices were not used.

| Command / check | Outcome | Evidence |
|---|---|---|
| Initial missing/short-root regression on baseline crypto implementation | Exit 1; both new tests failed as expected before correction | Tool output; corrected tests in source |
| `go test ./...` | Exit 0; 50 packages with passing tests | evidence/all-go-final.log |
| `pnpm --dir web check` | Exit 0; zero errors and zero Svelte warnings | evidence/web-check-final.log |
| `KREDIT_WEB_ADAPTER=node pnpm --dir web build` | Exit 0 | evidence/web-build-final.log |
| `go build` API, worker, bootstrap-owner | Exit 0; binaries in `.tmp/kredit-launch-*` | Build artifacts |
| `go run ./cmd/migrate` on fresh isolated PostgreSQL | Exit 0; clean schema through 87 | evidence/migrations.log |
| `go run ./cmd/migrate` from 87 to 88 | Exit 0 | evidence/migrations-final.log |
| `KREDIT_INTEGRATION=1 ... go test ./internal/platformsettings -count=1` | Exit 0; persistence, crypto, stale-edit rejection, history redaction and mutation rejection | evidence/settings-integration-final.log |
| `KREDIT_INTEGRATION=1 ... go test ./internal/web -run TestPlatformSettingsEndpoints -count=1` | Exit 0; actual API/DB test, including owner permissions and secret rotation | evidence/owner-api-integration.log |
| `playwright test content-seo.spec.ts public-security-content.spec.ts` | Exit 0; 11 passed; real local pages, not API-mocked | evidence/public-playwright.log |
| `PLAYWRIGHT_BASE_URL=http://127.0.0.1:3000 node scripts/launch-browser-check.mjs` | Exit 0; six routes, seven legal viewport widths, anchors/disclosure, guest session count and protected redirect | evidence/release-browser-check.log |
| `node scripts/launch-outage-check.mjs` | Exit 0; real unavailable upstream at port 65530, six public routes, failed OTP request, scoped pricing outage and protected API 503 | evidence/outage-check.log |
| `go test -race ./internal/platformsettings ./internal/credit ./internal/payments ./internal/onboarding` | Could not start race instrumentation: `runtime/race: package testmain: cannot find package` | evidence/race.log |
| `git diff --check` | Exit 0 | Terminal verification |

The initial unrestricted Go rerun encountered a compile error in the added history test
(variable redeclaration). It was corrected and the full suite rerun successfully. An earlier
sandboxed Go run could not open loopback HTTP test servers; the successful final run had
that permission. Those failed logs remain preserved; they are not passing evidence.

The full Go run does NOT mean all database integration suites passed: suites gated by
KREDIT_INTEGRATION were only explicitly enabled for the two entries above. Financial
workers, all provider contracts, all signed-in browser suites, restore/load/security tools,
Firefox/WebKit and race checks are not certified by these results.

## Browser evidence

`before-{home,pricing,terms,privacy,signin}.png` came from the original local application,
1440×1000 CSS viewport, guest state. The matching `after-*` files came from the final Node
build, same viewport and guest state. These are actual full-page screenshots, so output
height varies with document content. Mobile terms screenshots use 390×900 CSS pixels and
show the disclosure closed/open. Other tested legal widths: 320, 360, 430, 768, 1024, 1440.
`pricing-source-outage.png` records the deliberately failed upstream state.

No signed-in supplier/buyer/owner before/after evidence was captured. The broad release
brief remains incomplete; see OPERATING-NOTES.md and RESUME.md.
