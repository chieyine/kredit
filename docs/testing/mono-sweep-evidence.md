# Mono Sweep acceptance evidence — 2 September 2026

**Actual Mono sandbox proof: pending for every scenario below.** No Mono
credentials/Sweep access were available. Local HTTPS provider contract tests and
PostgreSQL integration tests are useful evidence, but do not certify Mono.
Use a separate sandbox database and the [runbook](../runbooks/mono-sweep.md).

| # | Required scenario | Repository evidence / remaining provider action |
| --- | --- | --- |
| 1 | Create supplier | Existing organization/onboarding tests; synthetic financial fixtures |
| 2 | Create buyer | Existing buyer/identity tests; synthetic user/business fixtures |
| 3 | Create ₦500,000 obligation | `collections/sweep_postgres_test.go`: 50,000,000 kobo fixtures |
| 4 | Buyer accepts | `credit/store_test.go`: acceptance before bank mandate; immutable accepted hash |
| 5 | Create Variable Sweep | `providers/mono/mono_test.go`: documented request contract |
| 6 | Receive authorization URL | Provider contract test; supplier views remove this URL |
| 7 | Complete hosted authorization | Requires buyer interaction in Mono sandbox; not simulated as certification |
| 8 | Confirm ACTIVE | Provider test requires approved and ready; database reuse/revocation tests |
| 9 | Make obligation due | Existing schedule/eligibility tests; independent worker discovers due items |
| 10 | Initiate debit | Database reservation committed before mock provider call |
| 11 | Process event | Secret authentication, allowlisted safe notice, River enqueue and aggregate lookup |
| 12 | Update ledger | Existing payment journal tests and PostgreSQL collected-payment tests |
| 13 | Update outstanding | Partial/manual payment integration tests verify exact kobo balance |
| 14 | Duplicate webhook | Five financial callbacks produce one attempt payment |
| 15 | Duplicate request | Concurrent database jobs produce one provider submission |
| 16 | Failed debit | Engine failure tests; Mono failed mapping is final unless a documented classification is added |
| 17 | Partial recovery | Contract maps collected amount; PostgreSQL test posts only recovered funds |
| 18 | Retry | Delay, retry budget, fresh-key bypass prevention and remaining-balance tests |
| 19 | Manual payment first | 30,000,000 kobo manual payment leaves 20,000,000 for collection |
| 20 | Revoke/expire | Durable cancellation, stale callback cannot reactivate, validity checks |
| 21 | Ambiguous result | Pending/timeout reservation retention, saved-reference lookup, crash-after-payment replay |

Additional evidence: restricted worker payment posting/reconciliation; fresh worker startup and external manual-payment visibility; shared mandate capacity across two obligations; conflicting
manual payment blocked while a debit is unresolved; immutable collection event
update rejected; local success/provider mismatch raised without changing the
ledger; disabled Mono cannot manufacture mock authorization; an unavailable payment-claim lookup blocks collection.

Validation performed:

- Fresh PostgreSQL 16 database migrated through 052 (production target remains PostgreSQL 18).
- `go test -timeout 180s ./...` passed.
- `bash scripts/test-integration.sh` passed, including repeatable seed and tagged persistence/ledger reconciliation.
- `KREDIT_INTEGRATION=1 go test -race -timeout 180s ./internal/collections ./internal/mandates ./internal/providers/mono ./internal/credit ./internal/payments` passed against an isolated local database.
- OpenAPI Go/TypeScript artifacts regenerated from `api/openapi.yaml`.
- Svelte/TypeScript check passed with zero errors and warnings.
- Field inventory check passed for all 943 fields.
- Completed reconciliation job can be scheduled again; duplicate active jobs collapse.
- Restricted worker posting/reconciliation, fresh state after restart/external payment, and fail-closed hold tests passed.
- Final targeted web/payment-claim integration regressions passed.

Before signing provider acceptance, record the sandbox run date, environment,
provider event/reference IDs in restricted evidence storage, expected/actual
amounts, each assertion, and reviewer. Do not place credentials, BVNs, bank
account inventories or hosted authorization links in this public runbook.
Confirm the documented retrieve-debit URL discrepancy and final partial-result
shape with Mono. Production configuration remains blocked until certification
is implemented and approved in a subsequent reviewed change.

Direct follow-up: [code review, fixed failure modes, verification and limits](code-review-2026-09-02.md). Local database migrations now run through 054. Actual provider certification remains pending.
