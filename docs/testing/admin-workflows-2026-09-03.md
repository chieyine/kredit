# Admin workflow implementation and verification — 3 September 2026

## Implemented scope

1. Role-scoped approval inbox covering policy, financial-change, dispute, financial-review, support, recovery and privacy work; recorded owner/deadline changes.
2. Immutable correction proposals and independent approval; exact repayment-date proposals with separate buyer consent. Financial effects, decision records and durable notices commit together.
3. Settings accept naira and percentages with exact integer conversion, retain safe large values, and display administrator names.
4. Policy previews provide changed-field effects and current population/workload counts. Current revision and execution-time controls remain authoritative.
5. Financial operator, policy manager, approver and access-administrator roles; existing support/compliance/dispute roles retained. Self-approval and conflicted financial approvers are rejected.
6. Overview attention cards and detailed operational work lists with direct actions.
7. Paginated search across retained change history, original/proposed values, named actors, reasons and CSV page exports.

Migrations 064–066 applied successfully to the disposable `kredit_sweep_release_check` database. Readiness requires schema 66 and the new tables/views/functions. New financial service tests use `kredit_app` to exercise actual runtime row policies and grants, rather than relying on fixture-owner privilege.

## Verification

- All Go packages passed unit checks. The API self-healthcheck initially could not open a local listener under the sandbox; rerunning that package with local networking allowed passed.
- Final authenticated HTTP checks passed for the impact preview, administrator-name lookup, inbox ordering, CSV/history reads, and specialist role restrictions.

- Focused final database/race checks passed for operations, collections, businesspolicy, web, access and db. Covered concurrent double approval (one journal), exact/conflicting proposal retries, disallowed proposer/approver roles, buyer identity, stale payment snapshots, unresolved debits, revoked proposing/approving roles, buyer rejection, exact buyer acceptance, and cancellation (but not application) after an obligation closes.
- Broader race/regression run passed collections, businesspolicy, web, access and db. Its old operations fixture lacked the schedule created by production activation; the new consistency guard correctly rejected it. Added a realistic schedule to that fixture and reran operations successfully with race checking.
- Eight browser journeys passed: admin sections, mobile navigation, admin team, independent correction, buyer consent, review ownership/history, business settings approval and public pricing.
- Final frontend static checking passed with zero errors and warnings. The production build completed successfully using the Vercel adapter, including the current-date and history refinements.
- OpenAPI Go and TypeScript generation succeeded.
- Data inventory regenerated: 1,021 fields.
- Earlier targeted tests caught and resolved a repayment-date SQL timestamp inference error and a write-off that left schedule totals stale. Write-offs now reduce the latest unpaid instalments while retaining allocation references needed for reversals; schedule and obligation balances must agree before a new write-off.

## Practical limits

The disposable PostgreSQL instance was stopped cleanly after verification. No deployment, real debits, provider messages, live Mono authorization or external approvals were performed. Mono credentials/Sweep access remain unavailable. Date amendments preserve instalment amounts and grace periods; they do not restructure amounts, renew mandates or rewrite original agreements. Impact previews are estimates based on current records, not forecasts of provider success. History exports contain the selected page of up to 100 records; older records are accessible by pagination. This is implementation and test evidence, not a claim that every possible loophole or regulatory requirement has been eliminated.


## Evidence logs

- `.tmp/admin-final-focused.log`: final affected-domain race checks.
- `.tmp/admin-final-endpoints.log`: final authenticated admin reads and policy impact preview.
- `.tmp/admin-closed-proposal.log`: closed-obligation cancellation regression.
- `.tmp/admin-browser.log`: eight passing browser journeys.
- `.tmp/admin-all-go.log` and `.tmp/admin-api-unit.log`: all-package unit check plus successful local-listener retry.
- `.tmp/admin-codegen.log`, `.tmp/admin-data-inventory.log`: generated contract and inventory evidence.
- `.tmp/admin-migration.log`, `.tmp/admin-migration65.log`, `.tmp/admin-migration66.log`: successful local migrations.

Final frontend check/build ran in the terminal: zero errors/warnings, successful production Vercel-adapter output. `git diff --check` passed.
