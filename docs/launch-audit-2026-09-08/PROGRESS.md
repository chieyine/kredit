# Current checkpoint

The original direct file read is complete. See [the final code review checkpoint](FILE-BY-FILE.md) for current fixes, verification and remaining deployment/provider requirements. The dated progress notes below are historical and do not describe the latest code state.

# Launch code audit — 8 September 2026

Status: in progress; no launch sign-off. This is a direct source-code audit and verification pass by the primary assistant. No subagents were used. The full file-by-file review is not yet complete. Passing repository checks must not be confused with manual review of every file.

The checkout contained extensive existing edits before this pass. Those edits are preserved. Earlier audit reports were treated as context, not current verification evidence.

## Confirmed defects addressed in this pass

- Fresh installation failed at migration 088 because runtime roles are provisioned after migrations. Conditional grants/revokes also repair migration 089's corresponding dependency.
- Re-running database role setup restored broad privileges on the owner guard and settings history. Explicit restrictions now follow baseline grants. Regression checks cover both application and worker roles.
- Public receipt links used a tenant-authenticated payment lookup and failed without tenant context against PostgreSQL. A narrowly scoped receipt projection is now used after signed-token verification; ordinary payment reads remain tenant protected.
- Public receipt UI described reversed payments as money received. It now distinguishes reversals and does not claim they reduce the debt.
- Public payment and receipt support buttons linked to a placeholder WhatsApp number. They now use the already-published support email.
- Public payment links with zero outstanding amount still encouraged payment. The payment action is now conditional on a positive verified amount and points at the corresponding sale.
- Public financial bearer tokens were not redacted from HTTP access-log paths. The sanitizer now covers receipt, payment-intent and public sale link paths.
- Admin jobs, provider events, recovery and privacy queues converted API failure into a successful empty state. They now reject failed/malformed reads and offer retry.
- Admin business/user directories and case lists could remain loading after transport failure; older search results could replace newer ones. Reads are bounded and invalidated when superseded.
- Admin overview, money, search and diagnostics reads could hang or accept incomplete responses. They now expose recoverable errors; search no longer displays “no match” while still searching.
- The shared admin client accepted malformed successful responses as success. It now checks JSON and bounds body reads, and identifies uncertain writes as unconfirmed.
- Settings listing discarded governance lookup failures; the UI assumed solo-owner approval before any successful read. Both now fail closed on unavailable governance.
- Owner dialogs could close during a pending write and hide errors behind an inert modal background. Busy dialogs retain context; errors appear inside the dialog; settings history failure is distinct from no history.
- Shared legacy reads had no timeout; public-link and reported-transfer pages lacked failure recovery. These paths now use bounded reads and retry controls.
- The OpenAPI maintenance task referenced a deleted script. It now calls the current generator.
- Release certification expected old document versions. It now agrees with the published Go and frontend version constants.
- Database readiness stopped at migration 80 and omitted the owner/settings tables and functions. It now requires the current runtime contract including the restricted receipt projection.

## Current verification evidence

- Full Go unit and database integration suites passed. PostgreSQL 18 migration 092 applied successfully, and the revised development seed was applied twice. The integration suite passed again against the corrected seed.
- Go vet and command builds passed. Strict Go lint initially found three unchecked deferred rollbacks in owner integration tests; after correction, it reports zero issues.
- Repository integrity, frontend/backend contract synchronization, frontend API coverage and content checks passed. Final Svelte check: zero errors and zero warnings, including the demo readiness fix.
- API schema lint: valid, with 377 warnings including missing descriptions, license metadata and method-disambiguated admin paths. This is not a warning-free specification.
- All 31 Python script tests passed. Database destructive-operation guard tests, environment-loader tests, governance tests, context audit and README structural conformance passed after stale verification assumptions were corrected.
- Initial 162-test browser run: 150 passed, nine failed, three real-backend tests skipped. All nine failures passed in a stable-preview rerun.
- Added outage/public-link tests passed. Real supplier and buyer journeys passed, including opening a persisted sale. Expanding these checks exposed inconsistent demo financial snapshots; the seed was corrected rather than weakening tenant checks.
- Fresh full 175-test browser run with real-backend journeys enabled: 174 passed, one demo early-click failure. After fixing readiness, all six targeted public/demo checks passed, including the failed original test and a new delayed-script regression. The full 176-test suite has not been rerun as one clean run after that final change.
- Production Vercel build passed earlier; final Node-adapter build passed. The Node bundle started from an isolated directory without development dependencies. All 19 selected browser checks against that standalone compiled bundle passed (public pages, delayed hydration, admin/account outages and public money links).
- Financial race checks passed for collections, credit, payments and reports.
- Local PostgreSQL backup/restore drill passed: source and restored data/security fingerprints matched, and runtime grants passed. This is an isolated logical restore, not production or point-in-time recovery certification. The backup script now uses a portable random directory; the new archive checksum passed.
- Docker container execution unavailable: the Docker daemon is not running. Container files have been reviewed but images have not been built here.
- Current dependency advisory scan not run: automatic approval review rejected sending dependency metadata to `https://registry.npmjs.org/` without explicit authorization. No alternate route was used to send that metadata.
- Local Node is 24.11.0 while the project pins 24.20.0. Local success does not verify the exact pinned container toolchain.

## Additional defects addressed while continuing

- Admin team and support-case operations could remain busy after network failure. Bounded reads/writes, recovery controls and explicit uncertain-write messages now retain a usable screen.
- Buyer reminder permissions treated unavailable consent data as stopped reminders. The page now hides unverified controls and supports retry.
- Account safety and message preferences could imply default settings after failed reads. Controls now require verified settings, and writes release their busy states after failure.
- Multiple collection attempts for the same sale shared a keyed UI identity, causing duplicate-key failures. The shared list now renders each attempt; a regression test covers two attempts for one sale.
- A provider without cancellation support could consume the circuit breaker's only recovery probe and strand later calls. Unsupported actions are checked before reserving a probe.
- Readiness ignored adapters that were configured but failed construction. It now rejects that runtime without revealing private provider diagnostics.
- Public capabilities always advertised disputes as enabled even when the action handlers rejected them under the owner switch. They now read the same flag/default.
- Demo financial snapshots claimed four non-existent obligations while omitting the three actual drawdown obligations. Portal projections now come from normalized records, with matching schedules. Only the known orphan development fixtures are removed during reseeding. The processed-webhook fixture remains, without treating its payload as proof of actual financial effects.
- Demo buttons accepted clicks before hydration attached their handlers. They are now disabled until the component mounts.
- The web Dockerfile required removed `openapi-fetch` at image build time. The stale dependency check was removed; empty production dependency directories are handled explicitly.
- PostgreSQL 18's local named volume mounted the obsolete directory. It now mounts `/var/lib/postgresql`, per [the official image documentation](https://hub.docker.com/_/postgres). No existing Docker volumes were modified or reset.
- The destructive-database guard test lacked the fingerprint prerequisite, and README conformance required retired component files and scenario labels. The tests now exercise the current contracts. Release-evidence type annotations also now import on the installed Python version; all 31 script tests pass.

The [file review index](file-review-index.json) records focused direct inspections separately from pending manual review. A focused inspection is not a claim of exhaustive verification of every line. No complete file-by-file sign-off has been issued.

## User constraints and launch gaps

The user plans to connect external services after deployment. SMS/email/identity providers have not yet been selected. They want operational configuration through admin rather than source edits.

Current code has Mono support and provider-neutral connector contracts for notifications, identity verification and document scanning. It does not provide native adapters for arbitrary vendors. The connector contracts require a compatible implementation; a vendor API key cannot make an arbitrary provider conform to those contracts. The owner settings registry now exposes three feature flags and encrypted email/SMS/WhatsApp connector overrides consumed on each non-development delivery. Identity, document scanning, Mono, the existing other-bank connector and launch approvals now also have encrypted admin controls consumed by API and worker startup. These changes require a coordinated API/worker restart; bootstrap database, storage and root keys remain deployment settings. See ADMIN-CONNECTIONS.md for the supported contract and initial sign-in dependency. Real provider success, production delivery, deployment configuration, production backup recovery and live money movement are not verified by local tests.

No deployment, live-provider mutation, production database mutation or external message was performed.


## Admin continuation

- The explicitly approved npm registry audit reported zero vulnerabilities (191 total dependencies in its report).
- Added atomic encrypted notification connector settings, strict HTTPS/token validation, per-delivery configuration resolution across API/workers, explicit disable, safe status metadata, and masked history. Redirects are refused to keep message bodies from following a connector redirect.
- Added migration 093 to permit only the three consumed integration keys as encrypted integration settings; runtime readiness now requires version 093.
- Protected controls now invalidate a preview when its form changes and submit the reviewed payload with a stable retry identity. Expiry input explicitly uses Lagos time.
- Remaining direct admin fetches now use bounded reads/writes. Disputes, reconciliation, audit, attention, approvals, inbox and history no longer turn malformed or failed reads into empty-work claims. Financial history subtraction uses exact integer amounts.
- Admin credential rotation/disable/fallback tests passed. PostgreSQL persistence tests passed, proving encrypted values at rest, hidden history values and readback by the runtime. All 23 unique admin/access/workflow browser cases now have passing results, including a clean 14-case run after the modal fixes. The interrupted financial approval test was rerun after removing a simultaneous build that reloaded Vite. Svelte checks report zero errors/warnings; the full Go unit suite and final affected backend checks pass; lint reports zero issues. The mobile connector dialog was visually inspected and its bounds/save-button reachability checked at 390×844. The final Node production build passed after mobile dialog and text-contrast fixes. Temporary Vite and PostgreSQL services were stopped after verification.

- Worker startup now rejects configured provider initialization failures before processing jobs. Notification connection errors no longer include endpoint URLs, and saved connector URLs reject query credentials.
- Setup instructions and the remaining deployment-only integration scope are in [ADMIN-CONNECTIONS.md](ADMIN-CONNECTIONS.md). No real message or live collection was sent.


## Completion of supported admin connection controls

- Added owner-editable identity, Mono, document scanner, other-bank connector and launch approval/limit settings, with generated fields from a strict server allowlist. Migration 094 permits exactly these encrypted integration keys.
- API and worker startup load and validate the saved configuration before constructing providers. Release certification includes stored settings as well. No source or environment-file edit is needed to replace supported provider credentials.
- The editor prefills ordinary values, hides credentials, preserves blank password fields, offers explicit credential removal, and distinguishes pending restart from configuration applied to the API. Worker restart and live-provider verification remain explicit requirements.
- Pausing Mono preserves reconciliation credentials; scanner disable removes it from the effective runtime without disclosing its stored secret. Shared transactional validation prevents conflicting cross-connection saves. Rejected candidates leave no history mutation.
- Configuration unit tests and focused PostgreSQL/owner endpoint tests passed. The full Go suite passed. All 24 admin/access/approval browser tests passed in one run, including the new Mono editor. The full PostgreSQL integration suite passed at migration 094. Final targeted tests also passed after requiring explicit token entry when an enabled connector address changes.

- Final frontend checks reported zero errors and warnings, the Node production build passed, and Go lint reported zero issues. Repository, product-contract, frontend-route coverage and content checks passed. No live provider call, message or money movement was performed.


## Direct file-by-file continuation

The new per-file audit and fixes are tracked in [FILE-BY-FILE.md](FILE-BY-FILE.md) and [file-by-file-audit.json](file-by-file-audit.json). This is an ongoing review; prior focused-review labels are not treated as exhaustive coverage.
