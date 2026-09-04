# Admin business settings implementation

Implemented an 18-field, database-backed policy configuration centre at `/admin/settings`, with immutable proposals, independent approval, effective dates, optimistic revision checks, conflict-safe proposal IDs, cancellation before activation, and retained decision history. Both proposed and effective values are validated against deployment approval boundaries.

The change connects policy values to collection workers, debit reservations, receipt-notice waiting periods, payment claims, business/principal/exposure/drawdown limits, review flags, correction thresholds, and new-offer fee snapshots. Fee disclosures, public pricing, activation journals and fee-rate accounting were updated together. Existing offers keep their recorded terms. Ordinary receipt activation now writes its journal, obligation, schedule and aggregate in one transaction.

Relevant implementation:
- `internal/businesspolicy`: policy catalog, validation, effective-value reads and approval workflow.
- `db/migrations/061_business_policies.sql`: durable proposals/history, role-safe admin verification and effective policy selection.
- `db/migrations/062_policy_enforcement_and_fee_terms.sql`: offer fee terms and financial/database policy guards.
- `db/migrations/063_preserve_existing_commitments.sql`: apply exposure limits before new goods commitments while preserving recognition of already released sales and existing drawdowns.
- `internal/web/business_policy_handlers.go` and `web/src/routes/admin/settings/+page.svelte`: protected API and admin UI.
- `internal/ledger/fee_terms.go`, credit/trade-line/payment repositories: accepted-rate calculation and persistence.
- `docs/runbooks/business-settings.md`: operating procedure and deployment boundaries.

Tests include complete input validation, approval by a different administrator, inactive/stale proposals, history immutability, application/worker database roles, global policy writer serialization, database business/principal/exposure limits, immutable accepted rates, actual collection fee metadata, public-pricing disclosure, and admin/pricing browser journeys. Existing financial and worker suites are also run with the new schema.

Provider credentials, certification, legal approvals, individually approved large corrections, and accepted-schedule amendment workflows are unchanged external or unavailable capabilities. This implementation does not claim that every deployment setting or future product workflow is editable through admin.

## Verification results

- All Go packages passed their seeded PostgreSQL/unit tests. The cross-domain suite was rerun after migration 063 and passed; its earlier run correctly rejected the temporarily older schema during implementation.
- Final schema: race-enabled policy, collection, web and persistence tests passed. The final policy regression was rerun after the legacy NULL fee-term compatibility fix and passed, including release-time exposure accounting and recognition after limits tighten.
- Two new browser journeys passed: proposing/independently approving settings, and current pricing/unavailable-pricing handling. The mobile admin navigation regression also passed with the new settings link.
- Svelte checking passed with zero errors and warnings. The Vercel-adapter production build passed.
- OpenAPI Go/TypeScript generation and whitespace checks passed. The database-derived data inventory includes the new policy and offer-term records.
- Migrations 061–062 were rolled back and reapplied before policy use; migration 063 was applied and its commitment rules verified. A rollback attempt against a disposable database containing a custom-priced offer was correctly refused by migration 062.

Detailed local logs: `.tmp/policy-final-integration.log`, `.tmp/policy-tagged-final.log`, `.tmp/policy-final-race.log`, `.tmp/policy-final-commitments.log`, `.tmp/policy-browser.log`, `.tmp/policy-mobile-browser.log`, `.tmp/policy-web-final.log`, `.tmp/policy-web-build.log`, and `.tmp/policy-used-rollback.log`.

No deployment, real bank debit, provider message, or live approval was performed. The changes remain in the working tree for review; apply migrations 061–063 and the runtime role grants when deploying.
