# Kredit — independent code audit

**Repository:** `~/Documents/Kredit.com` · branch `main` · HEAD `05799ca` (merge of PR #16, `audit/2026-09-20-integrity-remediation`)
**Date:** 21 September 2026
**Scope:** 1,416 tracked files — 538 Go, 208 SQL, 187 Svelte, 114 TypeScript, 40 shell, 13 Python, plus infra, CI and docs. 120,039 lines of code; 87,462 of Go.
**Brief:** file-by-file code audit, standard applied: world-class platform.
**Method:** every finding below was read out of the working tree and verified before it was written down. Where a suspicion did not survive verification it was dropped, and several were — those are listed in §6 so the same ground is not re-walked. No fixes applied; this is the audit pass.

---

## Part 0 — How to read this

Findings are numbered `A2-nnn` (this audit's namespace, distinct from the `K-nnn` and `F0nn` series already in the repository) and carry a severity:

| | Meaning |
|---|---|
| **S1 Critical** | Money, tenant isolation, or the control that is supposed to catch those, is not doing its job. |
| **S2 High** | A real defect that will bite in production, or an architectural choice that is actively costing correctness. |
| **S3 Medium** | Genuine defect with a workaround; inconsistency that will cause a wrong decision later. |
| **S4 Low** | Polish, hygiene, supply chain, dead weight. |

Every finding names `file:line`, states what is wrong, why it matters *here*, and the correction.

A word before the list. This codebase is better than most fintech code I have read. The double-entry ledger is honest, the outbox is properly fenced, the webhook design treats provider callbacks as signals and never as amounts, the production config gates are serious, the database roles are real least-privilege, and the comments explain *why* rather than *what*. Several findings below exist only because the codebase sets a high enough standard elsewhere that the gaps stand out. §5 lists what is already world-class, and it is a long list.

---

## Part 1 — The finding that outranks the others

### A2-001 · **S1** · The file-by-file audit register is boilerplate, and it certifies the defects in this report as correct

`docs/launch-audit-2026-09-21/file-review-index.json` · `docs/launch-audit-2026-09-21/README.md`

The register contains 1,421 entries. Measured directly:

| Property | Value |
|---|---|
| Entries | 1,421 |
| Distinct `notes` values | **1** |
| Distinct `status` values | **1** — `"inspected; verified correct"` |
| Distinct `review_method` values | **1** |
| Distinct `verification` values | **1** |
| Entries carrying any `finding_ids` | **0** |
| Entries carrying any `declared_symbols` | **0** |

Every file — including `.dockerignore`, 26 lines of ignore patterns — carries the identical sentence: *"Exhaustive code audit completed. All domain invariants, security boundaries, tenant isolation, error handling, and language idioms verified."* A 26-line ignore file has no domain invariants and no tenant isolation to verify.

Two further checks:

- The `verification` field on all 1,421 entries reads *"Compiled and verified (Go 1.27.1 / …)"*. `.go-version` and `go.mod` both pin **1.26.8**. The stated toolchain was never the one used.
- 3 of the first 400 files already fail their recorded SHA-256. The register was stale within a day of being written.

`README.md` in that folder states: *"Every file was reviewed line by line"* and **"Decision: Production-Grade & Verified."** `FILE-BY-FILE.md` marks `internal/ledger/`, `internal/credit/`, `internal/payments/`, `internal/collections/`, `internal/billing/` and every other money domain as **"Verified Correct (100%)"**.

This audit found, inside those same "100% verified" packages: a financial read path that converts database failures into "you are owed nothing" (A2-002), four independent copies of the single most important balance mutation in the system (A2-003), the money path running on contexts with no deadline at all (A2-005), and a nil-transaction panic in the bank-customer registration handler (A2-007). None of these are subtle enough to survive a line-by-line read.

**Why it matters here.** This is not a documentation problem. A register that returns "verified correct" for every input has zero discriminating power — it cannot distinguish a clean file from a broken one, so it provides no evidence about either. If it is shown to a partner bank, an investor, or NDPC in support of a launch decision, it is representing a check that did not happen. Under the standard this platform is holding itself to, that is the most serious thing in this report, because it is the control that is supposed to catch everything else in it.

**Correction.** Either delete the register, or regenerate it so that each entry carries per-file evidence: the symbols reviewed, the invariants checked, the findings raised (including "none, and here is what was checked"), and a hash pinned to the commit. A useful register has a distribution of outcomes. Keep `KREDIT-CODE-AUDIT.md`'s convention instead — it names `file:line`, states severity, and marks inferences `[needs runtime proof]`. That document is a real audit; the 21 September register is not, and the gap between them is visible to anyone who opens both.

---

## Part 2 — Money, correctness and isolation

### A2-002 · **S1** · A database failure on the receivables path renders as "you are owed nothing"

`internal/credit/store.go:360-361` (interface) · `internal/credit/postgres.go:620-627` · `internal/web/financial_reads.go:49,57` · `internal/web/runtime.go:701,703`

```go
// internal/credit/store.go:360
ListForSupplier(string) []View
ListForBuyer(string) []View
```

The interface has no error return. The PostgreSQL implementation therefore has nowhere to put one:

```go
// internal/credit/postgres.go:620
func (s *PostgresStore) ListForSupplier(organizationID string) []View {
	loaded, _ := s.hydrateList("supplier_organization_id", organizationID)   // error discarded
	defer s.releaseListing(loaded)
	return s.Store.ListForSupplier(organizationID)
}
func (s *PostgresStore) ListForBuyer(buyerUserID string) []View {
	views, _ := s.ReadForBuyer(context.Background(), buyerUserID)            // error discarded
	return views
}
```

`hydrateList` is what loads the aggregates from PostgreSQL. If it fails — pool exhausted, statement timeout, connection reset, RLS returning nothing because the tenant context was not set — the in-memory projection is simply not populated and `s.Store.ListForSupplier` returns an empty slice. The handler at `financial_reads.go:49` returns `(views, nil)`. The supplier gets **HTTP 200 and an empty receivables list.**

Both functions are wired straight into the reports store at `runtime.go:701` and `703` as `SupplierViews` and `BuyerViews`, so the ageing report, the receivables report and the enterprise export inherit the same behaviour.

The same shape appears at `internal/paymentclaims/postgres.go:136,140,144`, `internal/audit/store.go:183`, `internal/support/store.go:266` and `internal/organizations/postgres.go:163`.

**Why it matters here.** Kredit's entire proposition is that the supplier can see what he is owed and that the number is trustworthy. A transient database blip that silently answers "nothing" is worse than an error page: the supplier acts on it. He stops chasing a buyer, or he concludes the platform lost his money. An error he can retry costs him ten seconds; a zero he believes costs him the relationship.

**Correction.** Change the two interface methods to return `([]View, error)` and propagate. This is a mechanical change — `financial_reads.go:49,57` already return a two-value tuple and discard the error slot. Then audit the remaining `, _ :=` sites listed above. Until that lands, `hydrateList` returning an error should at minimum panic rather than return empty, so the panic middleware converts it into a 500; silently answering zero is the one outcome that must not happen.

### A2-003 · **S1** · The obligation balance is written by four independent copies of the same code

`internal/payments/postgres.go:531` · `internal/operations/postgres.go:200` · `internal/disputes/postgres.go:197-210` · `internal/db/schedule_adjustment.go:98`

Four functions each perform the same three-statement mutation: update `app.obligations.outstanding_kobo` and `payment_status`, bump `app.credit_requests.version`, and patch the `app.credit_aggregate_snapshots` JSON read projection with a triple-nested `jsonb_set`.

They are currently identical. The codebase already knows what happens when they are not — from the doc comment on the copy in `internal/db`:

> *"payments, operations and disputes each carry a private copy of this. They agree; the copy that was written independently (order credit notes) did not, and lost the projection, the version and the status vocabulary. New callers use this one."*

A fifth copy already diverged, and the failure mode was a read projection that no longer matched the balance — meaning the number shown to the supplier and the number the collection engine debits against had drifted apart.

The same duplication exists one layer down: `ledger.transactions` is written from four places — the canonical `internal/ledger/postgres.go:143`, plus direct inserts at `internal/payments/postgres.go:559`, `internal/disputes/postgres.go:215` and `internal/operations/postgres.go:170`.

**Why it matters here.** This is the single most important invariant in the product, and it has four maintainers. The comment documents that divergence has already occurred once. The next change to the status vocabulary, the projection shape, or the version semantics has four places to land and no compiler help if it lands in three.

**Correction.** `internal/db.UpdateObligationBalanceTx` already exists and is the best of the four — it validates `outstanding <= principal`, and it fails when the projection row is missing rather than leaving a balance that moved without its projection. Delete the other three and call it. The direct `ledger.transactions` inserts should route through `ledger.PostgresStore.postTx` for the same reason (see A2-004, which is a direct consequence of them not doing so).

### A2-004 · **S2** · Dispute adjustments move money and emit no event

`internal/disputes/postgres.go` (whole file — the package does not import `internal/outbox`)

`internal/payments`, `internal/operations`, `internal/tradelines` and `internal/ledger` all append to the transactional outbox. `internal/disputes` does not import it. `applyDisputeAdjustmentTx` (`postgres.go:165-211`) writes a `dispute_adjustment` ledger transaction, reduces the schedule principal, and rewrites the obligation balance and its projection — with **no domain event emitted at all**.

Any consumer of the event stream — notifications, analytics, the reconciliation job keyed off `event.AggregateType + ":" + event.AggregateID` at `cmd/worker/main.go:183` — is blind to dispute resolutions. A buyer's debt can fall by a material amount and nothing downstream is told.

Separately, `postDisputeLedgerTx` (`postgres.go:213-219`) returns `nil` on idempotency-key conflict without checking that the existing transaction matches the intent, where `payments.postLedgerTx:561` calls `validatePaymentJournalReplay` for exactly that case.

**Correction.** Route dispute adjustments through `ledger.PostgresStore.PostAdjustmentTx`, which both emits the outbox event and validates replay intent. This falls out of A2-003.

### A2-005 · **S2** · The ledger and payment write paths run with no deadline

`internal/ledger/postgres.go:112` and `:231` · `internal/payments/postgres.go:38,297,422,441,594` · 166 `context.Background()` call sites in non-test code

```go
// internal/ledger/postgres.go:112, inside post()
ctx := context.Background()
```

No timeout, no cancellation, no trace propagation. `internal/auth` and `internal/organizations` at least wrap theirs in a 15-second timeout; the ledger has none. The only bound is the server-side `statement_timeout` of 30s — but `pool.Begin(context.Background())` blocks on connection acquisition *before* any statement exists, so under pool exhaustion the goroutine waits indefinitely while the HTTP layer has already given up at its 30s `WriteTimeout`.

Distribution of the 166 sites: `credit` 35, `organizations` 23, `onboarding` 18, `buyers` 13, `auth` 12, `web` 6, `payments` 5, `collections` 5, and the rest scattered.

`internal/db/postgres.go:65-67` acknowledges this:

> *"Repository interfaces created before request-scoped contexts were added still contain a few background-context calls. Database-side limits keep those calls from holding a connection or lock indefinitely."*

"A few" is 166, and the database-side limits do not cover pool acquisition.

Three consequences, all real:
1. **Client disconnects do not cancel database work.** Under load, abandoned requests keep holding pool connections.
2. **OpenTelemetry traces are severed at every persistence boundary.** `internal/web/server.go:1017` starts a span per request; none of the database work below these calls joins it. The tracing investment produces traces with a hole where the database should be.
3. **Request deadlines are not honoured.** A store's 15s timeout applies even when the caller gave up 20s ago.

**Correction.** The `Context` variants already exist alongside most of these (`RecordContext`, `GetContext`, `ReadContext`, `RebuildContext`, `CollectionStateContext`). Push `ctx` through the `Service` interfaces and delete the non-context wrappers. Start with `internal/ledger` and `internal/payments`, where the absence is unbounded rather than merely long. `auth.Service.SessionFromToken(token string)` is the highest-traffic offender — it is called up to three times per authenticated request (`server.go:199`, `server.go:1040`, `auth_handlers.go:210`).

### A2-006 · **S1** · Thirty-three tenant-bearing tables have no row-level tenant isolation, and the gate that checks for this cannot see them

`scripts/rls-policy-shape-check.sh` · `docs/compliance/rls-permissive-baseline.txt` · `db/migrations/029,037,052,056…`

`scripts/rls-policy-shape-check.sh` is a genuinely good control. It reads the live catalogue (not migration text) and fails when a table has *both* a tenant policy and a permissive blanket role policy, because PostgreSQL ORs permissive policies and `tenant_predicate OR current_user IN ('kredit_app','kredit_worker')` is always true for the roles the application connects as. The baseline is empty, so the gate currently demands zero such tables. Migrations 158–163 did the work to get there. That is real engineering.

The blind spot: the query at lines 36-45 requires a tenant policy to be **present** before it will look for a blanket one. A table that has *only* a blanket role policy — no tenant predicate at any point in its history — is invisible to the gate.

Replaying all 203 migrations, 33 tables are in that state. Most are legitimately platform-scoped (`platform_settings`, `platform_governance`, `provider_events`, `job_dead_letters`, `idempotency_records`, `runtime_process_status`, `pilot_limit_configs`, `release_evidence`) and application-layer authorization is the right control for those. But these carry a tenant column and customer data:

| Table | Tenant column | Only policy |
|---|---|---|
| `app.support_cases` / `app.support_case_events` | `organization_id` | `support_case_runtime_access USING (current_user IN ('kredit_app','kredit_worker'))` — migration 029 |
| `app.operation_actions` | `organization_id` | `operation_action_support_access USING (current_user = 'kredit_app')` — migration 009, never dropped |
| `app.provider_customer_bindings` | `buyer_user_id`, `buyer_business_id` | `provider_customer_runtime` — migration 052 |
| `app.financial_review_cases` / `_events` | — (money discrepancies) | `financial_review_cases_runtime` — migration 056 |
| `app.business_policy_defaults` / `_changes` / `_events` | per-organization | blanket |
| `app.analytics_events`, `app.messaging_events`, `app.notification_delivery_receipts`, `app.collection_provider_events` | customer identifiers | blanket |

`app.operation_actions` holds write-offs and fee waivers. `app.provider_customer_bindings` maps a buyer to the provider customer reference used to debit their bank account.

**Why it matters here.** The whole point of this platform's RLS design — and migration 161's `dispute_reviewer_access` rewrite shows the team understands it — is that a missing `WHERE organization_id = $1` in a handler should be caught by the database rather than becoming a cross-tenant leak. For these 33 tables there is nothing behind the handler. The gate reports success, which is worse than reporting nothing, because it is read as coverage.

**Correction.** Two changes. First, extend the gate: add a second query that lists tables in `app`/`ledger` which carry an `organization_id`, `supplier_organization_id`, `buyer_user_id` or `user_id` column and have **no** policy referencing `current_organization_id` or `current_user_id`, with its own explicit baseline file. Second, work that list down, starting with `support_cases`, `operation_actions` and `provider_customer_bindings`.

### A2-007 · **S2** · Nil-transaction rollback panic in bank customer registration

`internal/web/mono_handlers.go:197` with `:271-274`

```go
tx, err := s.runtime.Database.Raw().Begin(r.Context())        // :192
if err != nil { … return }
defer func() { _ = tx.Rollback(r.Context()) }()               // :197  — captures the variable
…
if err = tx.Commit(r.Context()); err != nil { … return }      // :258  — first tx done
…
tx, err = s.runtime.Database.Raw().Begin(completionCtx)       // :271  — same variable reassigned
if err != nil {
    writeProblem(w, 503, "registration_unconfirmed", "…")
    return                                                    // :274  — deferred rollback now runs on nil
}
```

`pgxpool.Pool.Begin` returns `(nil, err)` on failure (verified against pgx v5.10.0 source). The deferred closure at line 197 captures the *variable*, not its value, so on that return path it calls `Rollback` on a nil `pgx.Tx` interface and panics.

The panic is recovered by `withPanicRecovery` (`server.go:787`), so the user sees a generic 500 instead of the carefully-worded "Registration needs reconciliation. Review this attempt in the admin bank connection panel" that the code was trying to give them — and this happens *after* the provider call succeeded and the `PENDING` attempt row was committed, which is precisely the state the admin panel exists to resolve.

Two comparable functions get this right by registering the defer after the second `Begin` succeeds: `internal/settlement/store.go:70-74` and `internal/buyers/verification_recovery.go:83-88`.

**Correction.** Move the second transaction into its own function, or shadow with `tx2, err := …` and give it its own defer.

### A2-008 · **S2** · A concurrent write makes a credit record unreadable to everyone on that replica

`internal/credit/postgres.go:786-791`

```go
func (s *PostgresStore) hydrateForTenant(requestID, userID, organizationID string) error {
	…
	if s.isPinned(requestID) {
		return errors.New("credit record is being updated; retry after the current change completes")
	}
```

`mutateAs` (`:708`) pins an aggregate for the duration of a command. While pinned, **every read of that credit request on that process fails**. A supplier refreshing his sale page while a payment is being recorded gets an error. A buyer opening the same sale gets an error.

The pin is a process-local mutex, so behaviour is also inconsistent across replicas: the identical read succeeds on the API pod that is not doing the write.

**Why it matters here.** The moment a supplier is most likely to look at a sale is the moment a payment lands on it. This turns the single highest-value read in the product into an error exactly when it matters, and it does so non-deterministically depending on which pod the load balancer picked.

**Correction.** A read should not need the write's projection. `hydrateForTenant` already reloads from PostgreSQL on every read (the function's own doc comment says so). Have the pinned case read the committed aggregate into a throwaway view rather than refusing — PostgreSQL's MVCC already provides the isolation the pin is trying to recreate in process memory. The broader fix is A2-009.

### A2-009 · **S2** · Every domain has two implementations, and the development one is load-bearing in production

`internal/web/runtime.go:142-1112` · `internal/credit/postgres.go:37-52`

`NewRuntimeWithDB` is a **975-line constructor** containing roughly forty `if database != nil` branches, each selecting between an in-memory store and a PostgreSQL store for one domain. `credit.PostgresStore` does not replace `credit.Store` — it *embeds* it and uses it as a mutable process-local cache of financial aggregates.

The ordering of branches inside this function is load-bearing, and the code says so twice:

> `:728` — *"Apply pilot guards after selecting the runtime adapter so a durable deployment cannot silently lose its buyer limits during adapter switch."*
> `:750` — *"The database adapter is intentionally created after the in-memory development store; setting the guard only before that switch would silently remove the supplier-organisation cap in staging and production."*

Both comments describe bugs that already happened: pilot exposure caps silently disabled in production because two lines were in the wrong order.

Other costs visible in the same function: `cfg.RetainedCollections()` is parsed five separate times (`:238,273,308,401,937`); `businesspolicy.ValidateStartup`, `observability.NewTracer` and `documents.NewS3ObjectStore` all take `context.Background()` with no startup deadline (`:461,499,770`), so a slow database or object store hangs boot indefinitely; and at `:785` a document-scanner construction error is discarded with `if scanner, err := …; err == nil`, leaving the scanner silently absent — directly contradicting this function's own opening principle at `:143`: *"A provider that fails to construct must be visible. Discarding the error leaves an adapter silently absent, which is indistinguishable from an adapter that is deliberately switched off."*

**Why it matters here.** Two implementations of every money domain doubles the surface for divergence, and A2-003 shows divergence is not hypothetical. A 975-line constructor cannot be unit-tested in pieces, so the ordering constraints are enforced by comments and memory.

**Correction.** This is the largest structural item in the report and it is not a weekend's work. The tractable first step: extract one `buildX(cfg, database) (X, error)` function per domain, each returning an error instead of appending to a shared `providerFailures` slice, and have `NewRuntimeWithDB` call them in sequence. That alone makes the ordering explicit and each domain testable. The second step is to stop embedding `credit.Store` in `credit.PostgresStore` and make the in-memory stores test doubles rather than production code paths.

### A2-010 · **S2** · Field encryption omits associated data and key versioning in three of four implementations

`internal/auth/postgres.go:655-682` · `internal/buyers/postgres.go:571-598` · `internal/notifications/store.go:987-1022` · compare `internal/platformsettings/crypto.go:82-148`

`internal/platformsettings/crypto.go` is the reference implementation and it is correct: AES-256-GCM with the setting key bound as associated data, a `k1.<keyid>.` prefix so the root key can be rotated, an HMAC fingerprint, and refusal to operate when the key is missing. Its own comment states the reason:

> `:78` — *"context binds a ciphertext to the setting it belongs to. Without it a ciphertext lifted from one settings row decrypts cleanly in another, so a mis-targeted write or a partial restore could swap one provider credential for a different one and nothing would notice."*

The other three implementations do exactly what that comment warns against — `gcm.Seal(nonce, nonce, plaintext, nil)` with a nil AAD and no key identifier:

| File | Protects |
|---|---|
| `internal/auth/postgres.go:668` | OTP target (phone/email), TOTP secrets |
| `internal/buyers/postgres.go:585` | buyer invitation targets, mandate account tokens |
| `internal/notifications/store.go:1003` | notification destinations |

Consequences: a ciphertext is portable between rows, columns and users within the same key domain — a row-copy bug, a partial restore, or any write that lands on the wrong row produces a value that decrypts cleanly into someone else's record. And with no key identifier in the ciphertext, `FIELD_ENCRYPTION_KEY` cannot be rotated without a full re-encryption migration; `cfg.FieldEncryptionKeyID` exists and is validated in production config (`config.go:518`) but is only ever mixed into a derivation label at `runtime.go:710` — it is never stored alongside the data.

All three also fall back to the literal `"development-only-change-me"` when the key is empty (`auth/postgres.go:151-159`). Production config forbids that, but **staging does not** — `config.Validate` only enforces key presence under `Environment == "production"`.

**Correction.** Extract `platformsettings.Encryptor` into a shared `internal/crypto` package and have all four use it, passing a context string of `<table>:<column>:<row-id>`. This is a data migration for existing ciphertexts, so it needs a dual-read window: try new-format first, fall back to legacy, re-seal on read.

### A2-011 · **S3** · `ledger.Reconcile` is a full-table aggregate with no bound

`internal/ledger/reconcile.go:36-47`

```sql
WITH balances AS (
    SELECT t.id, count(p.id) AS posting_count, …
    FROM ledger.transactions t
    LEFT JOIN ledger.postings p ON p.transaction_id = t.id
    GROUP BY t.id
) SELECT count(*), …, array_agg(id) FILTER (WHERE debit<>credit OR posting_count<2) FROM balances
```

Correct — it checks every transaction individually rather than trusting global totals, and it catches headers with missing postings. But it has no date bound and no watermark, and it runs from the reconciliation job every five minutes (`cmd/worker/main.go:170,220`). It rescans the entire journal each time. Against the 30s `statement_timeout` this fails somewhere in the low millions of postings — and it fails by *timing out*, which means the reconciliation check silently stops running at exactly the point the business is large enough to need it.

Additionally, `array_agg` of unbalanced IDs is unbounded: a systemic bug would try to return every transaction ID in one array.

**Correction.** Add a watermark (`WHERE t.recorded_at > $1`) with the last successful run stored in `app.runtime_process_status` or similar, plus a periodic full sweep on a slower cadence with an explicit long statement timeout. Cap `array_agg` with `LIMIT 1000` and report the count separately.

### A2-012 · **S3** · Outbox ordering is not guaranteed per aggregate, and the claim loop is N+1

`internal/outbox/store.go:83-113`

`Claim` orders by `created_at` with no tiebreaker, then combines `FOR UPDATE SKIP LOCKED` with per-event `UPDATE` statements in a loop — one round trip per event, up to 500, all inside one transaction holding row locks.

Two issues. Ordering: two events for the same aggregate created in the same microsecond have no deterministic order, and `SKIP LOCKED` means a second worker can publish a later event while an earlier one is still being processed. For `payment.recognized` followed by `payment.reversed` on the same payment, order matters.

Throughput: the loop is N+1 where one `UPDATE … WHERE id = ANY($1) RETURNING` would do. With the worker's 2-second tick and a batch of 100 (`cmd/worker/main.go:174,274`), the ceiling is about 50 events/second per replica, and `DispatchOnce` publishes sequentially on top of that.

**Correction.** `ORDER BY created_at, id` for determinism; a single batched `UPDATE … RETURNING` for the claim; and if per-aggregate ordering is required, add `FOR UPDATE SKIP LOCKED` on a per-`aggregate_id` advisory lock rather than per row.

### A2-013 · **S3** · No dead-letter path for permanently failing outbox events

`internal/outbox/dispatcher.go:50-53`

After ten attempts an event is rescheduled 24 hours out — forever. There is no terminal state, no `app.job_dead_letters` equivalent for the outbox (that table exists, but serves River), and no alerting hook. A malformed payload retries daily and indefinitely while `DispatchOnce` returns a joined error on every pass that nobody reads.

**Correction.** Add a `dead_letter` state after a bounded number of extended retries, surface the count in `observability.DurableFinancialMetrics` (which already reports `outbox_delivery_failures` at `db/migrations/086:24`, but only for `attempts>=8` in the `failed` state), and alert on it.

### A2-014 · **S3** · The whole worker's periodic work shares one goroutine and one `select`

`cmd/worker/main.go:203-232`

Outbox dispatch (2s), notification delivery (30s), maintenance (60s) and reconciliation (5min) all run in a single `select` loop. `EnqueueCollectionWork` and `DueDeliveryIDs` take the unbounded process context. One slow query in any of them stalls all of them — including the outbox, which is the transport for every committed financial event.

**Correction.** One goroutine per ticker, each with its own `context.WithTimeout` per iteration sized to its cadence.

### A2-015 · **S3** · Rate limiting is keyed on client IP in a heavily CGNAT market

`internal/web/server.go:44-57, 799-862`

120 requests/minute per IP in-process, and 20 per 10 minutes per (route-group, IP) shared across replicas for authentication routes — enforced fail-closed, which is the right call for a six-digit OTP.

The problem is the key. Nigerian mobile networks — MTN, Airtel, Glo, 9mobile — put very large numbers of subscribers behind carrier-grade NAT. So does every shared office and market Wi-Fi, which is precisely where a distributor's purchasing staff sit. Twenty sign-in attempts per ten minutes for an entire NAT pool means legitimate users are locked out of sign-in by strangers, and the failure is indistinguishable from an attack.

**Correction.** Keep the IP budget as an outer bound but raise it, and add a per-identifier budget as the tight control — the thing that actually makes a six-digit code safe is a cap on attempts *against a given account*, not against a given IP. `app.record_rate_limit_attempt(bytea, interval)` already takes an opaque digest, so keying it on `hash(channel:identifier)` in addition to the IP is a small change. Related: the per-challenge cap of 5 attempts (`auth/postgres.go:263`) with a 30-second resend cooldown per target bounds a targeted attack to roughly 10 guesses/minute against a known number, with no lifetime cap — add one.

### A2-016 · **S3** · CSRF token is not bound to the session, and cookies have no `__Host-` prefix

`internal/web/auth_handlers.go:312-321, 233-245`

The double-submit token is an independent random value with no cryptographic relationship to the session. Combined with the session cookie carrying no `__Host-` prefix and no `Domain` attribute, an attacker who can set cookies on a sibling subdomain of `kredit.ng` (an XSS on a marketing subdomain, a misconfigured CNAME, a compromised static host) can plant `kredit_csrf` and satisfy the double-submit check. `SameSite=Lax` blocks the cross-site POST, so this needs a same-site foothold — but the pairing of the two controls is the thing that is supposed to survive one of them being weak.

`sameOriginRequest` (`:261-295`) also returns `true` when both `Sec-Fetch-Site` and `Origin` are absent. The reasoning given is sound (proxies strip them) and the token remains, but it means the belt-and-braces design degrades to one control in exactly the deployment shape this product uses.

**Correction.** Two small changes, both cheap: rename the cookies to `__Host-kredit_session` / `__Host-kredit_csrf` (requires `Secure`, `Path=/`, no `Domain` — all already true), and make the CSRF value `HMAC(session_signing_key, session_id)` so a planted cookie cannot match a session the attacker does not hold.

### A2-017 · **S3** · A `viewer` can download identity and KYB documents

`internal/web/document_handlers.go:30` vs `:68`

Upload requires `PermissionCreateCredit` (owner, administrator, sales). Download requires only `PermissionReadOrganization`, which `access.Can` grants to **every** valid role including `RoleViewer` (`internal/access/roles.go:146`). Everything in `app.documents` for the organization — KYB filings, director identity documents, bank statements, invoices — is readable by the lowest-privileged member.

The dispute-document carve-out at `:86-89` shows the team thinks about this per-purpose; the same reasoning has not been applied to identity documents.

**Correction.** Gate download on `doc.Purpose`: identity and KYB purposes should require `PermissionManageOrganization` or `PermissionReadFinancial`, not bare membership.

### A2-018 · **S3** · Admin-editable connector endpoints are only URL-validated in production

`internal/config/config.go:721-735` · `internal/config/admin_connections.go:19-81`

`ApplyStoredConnections` reads connector endpoints from `app.platform_settings` — admin-editable at runtime — and `Validate` runs over the result. But `validateProductionURL` is only invoked under `Environment == "production"`, and even there it blocks only `localhost` and loopback/unspecified addresses. It does not block link-local (`169.254.169.254`, cloud metadata) or RFC1918 ranges.

The `https` requirement is doing most of the work here — cloud metadata services are HTTP-only — so this is defence-in-depth rather than a live hole. But an admin-controlled outbound URL with no egress allowlist is the standard SSRF shape, and staging skips the validation entirely.

**Correction.** Add `ip.IsLinkLocalUnicast() || ip.IsPrivate()` to `validateProductionURL`, and run URL validation for staging as well as production.

### A2-019 · **S3** · Payment links are stateless and cannot be revoked

`internal/publictoken/token.go:24-76`

Well-built HMAC tokens — strict base64, explicit CR/LF rejection, length-bounded, expiry checked. But they carry no server-side state, so a payment link that leaks (forwarded WhatsApp message, shared phone, screenshot) stays valid until it expires with no way to revoke it and no record that it was used.

**Correction.** Add a `jti` to the payload checked against a small revocation table, or at minimum record first-use so a support agent can see that a link was opened and invalidate it.

---

## Part 3 — Engineering practice

### A2-020 · **S2** · The frontend has no linter and no formatter

`web/package.json:11-12`

```json
"check": "svelte-kit sync && svelte-check --tsconfig ./tsconfig.json",
"lint":  "svelte-check --tsconfig ./tsconfig.json"
```

`lint` is a type check. There is no ESLint, no Prettier, no `svelte-check --fail-on-warnings`, and no config file for either in `web/`. Meanwhile Go has `.golangci.yml` with `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `gofmt` and `goimports`, and CI installs a checksum-pinned linter binary to run it.

The result is visible in the source. From `web/src/routes/workspace/today/+page.svelte`:

```svelte
let due=$state<Resource<WorkRow[]>>(pending());                    // :25  no spaces
let visibleCount = $state(5), legalName = $state(''), tradingName = $state(''),
    registrationInfo = $state(''), businessType = $state('unregistered_business'),
    address = $state(''), industry = $state(''), createBusy = $state(false),
    createError = $state('');                                       // :28  nine declarations, one statement
const attention = $derived(attentionItems(organizationID, sales.state === 'ready' ? …
    …due.state==='ready'?due.data:[]));                             // :35  ~300 chars, mixed spacing
```

**Why it matters here.** 10,165 lines of Svelte and 3,000 of TypeScript are held to a lower standard than the Go, by policy rather than by accident. For a team that intends to hire, inconsistent formatting is the first thing a new engineer fights and the last thing anyone gets round to fixing.

**Correction.** Add `eslint` with `eslint-plugin-svelte` and `prettier` with `prettier-plugin-svelte`, run `prettier --write` once across `web/src`, and add both to `scripts/ci.sh` alongside `svelte-check`.

### A2-021 · **S3** · The workspace is a client-rendered SPA on a mobile-data market

`web/src/routes/**` — 78 route files use `onMount` for data loading; 13 `+page.server.ts` files totalling 74 lines; 2 `+page.ts`

Almost no authenticated page uses SvelteKit's server `load`. `workspace/today/+page.svelte` fetches seven separate resources from the browser after hydration (`:18-26`): businesses, sales, payments, overdue, claims, disputes, due, summary.

So the sequence on a supplier's phone is: HTML shell → JS bundle → hydrate → seven API calls → content. On a 3G or congested 4G link, which is the median condition for the traders this product is for, that is several seconds of blank screen before the first number appears, and every one of those calls pays the round trip again.

SvelteKit's `+page.server.ts` `load` runs inside the same Node process that already proxies to the API (`hooks.server.ts:49-98`), so moving these fetches server-side converts seven high-latency browser round trips into seven low-latency in-datacentre ones, streamed into the initial HTML.

**Correction.** Move the primary resource of each workspace page into a `+page.server.ts` `load` and stream the secondary ones with promises. Start with `workspace/today`, `workspace/sales` and `workspace/purchases` — the three highest-traffic screens.

### A2-022 · **S3** · Eight CI workflows are bound to a merged branch and can never run again

`.github/workflows/audit-closeout-diagnostics.yml`, `audit-document-retention.yml`, `audit-format-diagnostic.yml`, `audit-journal-replay.yml`, `audit-offline-validation.yml`, `audit-rollback-cleanup.yml`, `audit-source-review.yml`, `audit-source-snapshot.yml`

Each triggers on:

```yaml
on:
  push:
    branches: [audit/2026-09-20-integrity-remediation]
    paths: [.github/workflows/<its own filename>]
```

That branch was merged into `main` as PR #16 — it is the current HEAD. These workflows fire only on a push to a merged branch that touches the workflow file itself. On `main` they are inert. Of the 20 workflows in the repository, 8 are scaffolding that reads as coverage.

Alongside this: `git branch -a` shows 14 remote branches including three separate `audit/2026-09-2x-*` branches.

**Correction.** Delete the eight, or re-point the ones worth keeping at `pull_request`. Prune the merged branches.

### A2-023 · **S3** · Supply-chain pinning is rigorous in one place and absent in four others

`.github/workflows/ci.yml:47-78` · `infra/containers/Dockerfile.api:3,23` · `infra/environments/versions.tf`

The golangci-lint install is exemplary: pinned version, SHA-256 checked, ELF magic verified, `GOPROXY=off` when running it, and a comment explaining *"Do not compile the linter through the application's module cache."*

Eleven lines later, that exact thing happens for four other tools:

```yaml
- name: Install required Go analysis tools
  run: |
    go install golang.org/x/vuln/cmd/govulncheck@v1.7.0
    go install github.com/securego/gosec/v2/cmd/gosec@v2.28.0
    go install honnef.co/go/tools/cmd/staticcheck@v0.7.0
    go install github.com/google/osv-scanner/v2/cmd/osv-scanner@v2.4.0
```

Version-pinned but not checksum-pinned, and compiled through the application's module cache. Also unpinned: GitHub Actions are referenced by mutable tag (`actions/checkout@v4`, not a commit SHA), and container base images by tag (`golang:1.26.8-alpine`, `gcr.io/distroless/static-debian12:nonroot`) rather than digest.

**Correction.** Pin actions to commit SHAs, base images to `@sha256:` digests, and either checksum-pin the four Go tools or accept the module-cache path and delete the comment that says not to.

### A2-024 · **S3** · Terraform has no remote state backend

`infra/environments/versions.tf` · `infra/environments/main.tf`

No `terraform { backend … }` block anywhere in `infra/environments/`. State defaults to a local file: no locking, no shared state between operators, no history, and a `terraform.tfstate` containing Kubernetes secret material living on whichever laptop ran `apply` last. Two people applying concurrently will corrupt the environment.

**Correction.** Add an S3/R2 backend with DynamoDB-equivalent locking (R2 is already in the stack — `cmd/backup-r2` exists), and add `*.tfstate*` to `.gitignore` if it is not covered.

### A2-025 · **S3** · `roles.sql` grants the privileges its own comment says it withholds

`infra/postgres/roles.sql:23-36` · `.github/workflows/ci.yml:92-93`

```sql
-- New tables are deliberately NOT auto-granted to either runtime role: a
-- migration must make an explicit privilege decision before new data becomes reachable.
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA app TO kredit_app;
…
ALTER DEFAULT PRIVILEGES IN SCHEMA app REVOKE SELECT, INSERT, UPDATE ON TABLES FROM kredit_app;
```

The `ALTER DEFAULT PRIVILEGES` line delivers the stated property for tables created *after* it runs. But the deployment order is migrate-then-apply-roles (`ci.yml:92-93`, `scripts/db-reset.sh:24`, `scripts/configure-development-database.sh:22`), so `GRANT … ON ALL TABLES` sweeps up every table the migrations just created. Re-running `roles.sql` after each migration — which is what every script does — inverts the control completely.

**Correction.** Either apply `roles.sql` *before* migrations and let the default-privileges rule do the work, or drop the blanket `GRANT ON ALL TABLES` and require each migration to name its grants. The second is what the comment describes.

### A2-026 · **S3** · Hardcoded Cloudflare ranges silently collapse the rate limiter when they go stale

`infra/environments/Caddyfile.prod:4-6`

`trusted_proxies static <93 hardcoded CIDRs>` with `trusted_proxies_strict`. Cloudflare publishes changes to these ranges. When a new range appears, Caddy stops trusting that hop, `{client_ip}` becomes Cloudflare's address for the affected traffic, and every one of those users collapses onto one rate-limit key — locking them out of sign-in with no error anywhere that says why.

**Correction.** Use Caddy's `trusted_proxies cloudflare` dynamic module, or add a scheduled job that refreshes the list from `https://www.cloudflare.com/ips-v4` and reloads Caddy. Also strip `X-Kredit-Client-*` at the edge as defence-in-depth, as `hooks.server.ts:57` already does on the other path.

### A2-027 · **S4** · Log redaction is key-based and over-broad in one direction, under-broad in the other

`internal/platform/logging/sanitize.go:9` · `logging.go:78-85, 106`

`Redact` matches only `key=value` / `key:value` shapes (`token=`, `secret:`, `bvn=`…). A bare Nigerian phone number, email or 11-digit BVN inside a free-text error message passes straight through. Structured attributes are covered by `sensitiveMetadataKey`, so the gap is message bodies and wrapped errors.

In the other direction, `sensitiveMetadataKey` matches the substring `account`, so `account_id`, `provider_account` and `account_state` all log as `[redacted]`; and `SafeError`'s marker list includes `provider`, so any error mentioning a provider name reduces to `"operation failed"`. `SafeError`'s own comment identifies this as the failure mode to avoid: *"every database failure logged as 'operation failed', which is the one thing an on-call engineer cannot work with."* The SQLSTATE carve-out fixes it for database errors only.

**Correction.** Add value-shape patterns (`\+?234\d{10}`, email, `\b\d{11}\b`) to `Redact`. Narrow the key list to word-boundary matches (`account_number`, not `account`) and drop `provider` from `SafeError`'s markers in favour of redacting provider *payloads* specifically.

### A2-028 · **S4** · `safeProblemDetail` swallows legitimate user-facing messages

`internal/web/http_helpers.go:55-77`

Any 4xx detail containing `token`, `provider`, `connection` or `database` is replaced with `"the operation could not be completed"`. So `"invitation token has expired"` and `"select a collection provider"` both reach the user as a generic failure — on a product explicitly built for a low-literacy audience, where the specific message is the whole point.

Separately, `decodeJSONRequest` (`:36-42`) passes raw `encoding/json` errors through, which leaks Go struct and field names: `json: cannot unmarshal string into Go struct field createRequest.amount_kobo of type int64`.

**Correction.** Invert it: have handlers pass an explicit user-safe message, and reserve the marker scan for the fallback path. Map JSON decode errors to a fixed "that request could not be read" rather than forwarding the library's text.

### A2-029 · **S4** · One component is 11% of the frontend

`web/src/routes/admin/platform-settings/+page.svelte` — 1,098 lines, 255 of them in `<script>`

Out of 10,165 total Svelte lines across 187 files. The next largest is 455. No other file is close.

**Correction.** Split per settings group — the `platformsettings.RuntimeConnections` map already defines the natural boundaries.

### A2-030 · **S4** · `{@html}` on deck slide titles is safe today by data provenance only

`web/src/lib/components/Deck.svelte:59,72`

`{@html slide.title}` where slides come from hardcoded TypeScript modules (`lib/deck/investor.ts`, `manufacturer.ts`). Safe now. It stops being safe the moment decks become editable content, and nothing in the file records that constraint.

The other nine `{@html}` sites are JSON-LD through `jsonLd()` (`lib/seo.ts:116-118`), which escapes `<` and is correct.

**Correction.** A comment naming the invariant, or a tiny allowlist sanitiser for the `<br>` and `<em>` these titles actually use.

---

## Part 4 — What I checked that turned out to be fine

Recording these so the same ground is not re-walked:

- **SQL injection.** Every dynamic SQL site was traced to its source: `auth/postgres.go:85,91` (column name from a two-branch `if`), `credit/postgres.go:897` and `credit/reads.go:51` (predicate from a closed set), `schedules/store.go:887` (`ASC`/`DESC`), `platformops/controls.go:211-236` (`" FOR UPDATE"`), `referrals/read.go:61` and `usercontrol/export.go:62` (fixed query maps), `orders`/`buyers` (constant column lists). **No user input reaches any of them.** The pattern is fragile — one future caller passing a parameter would open it — so a typed wrapper would be worth having, but there is no bug today.
- **Money types.** All 88 `*_kobo` columns are `bigint`. The one `numeric` is a `sum()` return type in a function signature (`db/migrations/120:10`), which is correct. No float anywhere near money. The `double precision` casts in `db/migrations/086` are Prometheus gauge counts.
- **Fee rounding drift.** `ledger.BaseFee` uses `(principal*5)/1000` and `ledger.FeeAtRate` uses `(a/10000)*bps + (a%10000)*bps/10000`. Algebraically both equal `floor(principal/200)` at 50bps, so `ActivateWithFee`'s equality check between them cannot spuriously fail. Both truncate rather than round half-up, which systematically favours the payer by under a kobo — worth a deliberate decision, not a bug.
- **Self-promotion through role change.** An administrator has `members:manage` and could in principle promote themselves past the financial separation of duties. `organizations/postgres.go:392-423` blocks it in SQL: `AND role <> 'owner' AND user_id <> $4`, plus `role == RoleOwner` rejected as a target. Correctly closed.
- **Session idle-timeout on step-up.** `StepUpSession` (`auth/postgres.go:556`) checks `now.Sub(session.LastSeenAt) >= sessionIdleTimeout` without the zero-value fallback that `SessionFromToken` has. Looked like a bug; `app.sessions.last_seen_at` is `NOT NULL DEFAULT now()` (`db/migrations/068:14`), so it cannot be zero.
- **Outbox double-append on ledger replay.** `ledger/postgres.go:158` re-appends the outbox event on the idempotent-replay path. `outbox.AppendTx` upserts on `idempotency_key` with a `WHERE` clause comparing every field including `payload`, and jsonb equality is semantic — so the replay is a no-op, not a duplicate.
- **Duplicate jobs across worker replicas.** Every River job type sets `UniqueOpts`, with `ByState` chosen deliberately per type — maintenance and reconciliation re-enqueueable after completion, notifications and webhooks not. Correct.
- **`ReduceSchedulePrincipalTx` restoring principal on cancel.** `db/schedule_adjustment.go:64-72` sets `next = i.principal` when the item is fully forgiven. Looked wrong; cancelled items are excluded from the totals query by `state<>'CANCELLED'`, so retaining the original principal is the audit-trail-preserving choice. Deserves a comment.
- **Webhook signature verification.** All seven provider webhook paths verify with constant-time comparison before doing anything. Flutterwave's is a static shared secret with no payload binding — that is Flutterwave's protocol, not a coding error, and the `provider_webhook_inbox` dedupe covers replay.

---

## Part 5 — What is genuinely world-class already

Said plainly, because it is unusual:

1. **Webhooks are signals, never amounts.** `mono_handlers.go:57-58`: *"It never posts the webhook's amount directly; the server-to-server lookup is authoritative."* Every provider path follows it. This is the single most important design decision in a collections platform and most teams get it wrong.
2. **The transactional outbox is properly fenced.** `MarkPublished`/`MarkFailed` match on `state`, `attempts` *and* `processing_started_at`, returning `ErrClaimLost` otherwise — so a slow publisher cannot overwrite an event another worker reclaimed. Lease recovery, `SKIP LOCKED`, jittered exponential backoff. Textbook.
3. **Double-entry is real.** `validateTransaction` enforces exactly one positive side per posting and debits equal to credits, with `CheckedAdd` guarding overflow on every accumulation. `Reconcile` checks each transaction individually and flags headers with fewer than two postings.
4. **Production config gates are serious and correctly factored.** The `config.go:463-471` split between always-required infrastructure and capability-gated requirements is exactly right, and it is what lets the public site ship before a bank is contracted without weakening a single control. `FEATURE_REAL_COLLECTIONS` requiring `FEATURE_REAL_IDENTITY` — *"money must not move against an unverified party"* — is the sort of rule most teams discover after an incident.
5. **Database roles are genuine least privilege.** `NOLOGIN` runtime roles set as a startup parameter so no query can run under the login role; startup verification that the session is not superuser, not `BYPASSRLS`, not the database owner, not the object owner (`db/postgres.go:97-129`); `audit_events` stripped of `UPDATE`/`DELETE`; a separate `BYPASSRLS` backup role.
6. **Deemed acceptance demands evidence.** `credit/postgres.go:94-143` will not let silence become a debt unless a delivery receipt proves the notice sat with the buyer for the full window, *and* the buyer has responded to something before — *"deemed acceptance is not available for a buyer's first trade credit."* Every failure path, including an unreachable database, refuses. That is the correct asymmetry and the comment explains why.
7. **The idempotency middleware is thorough.** Scope includes user, session, AAL, platform roles and membership role/status, so a cached response cannot survive a permissions change. One-time credential responses are explicitly non-replayable. A panic records the failure before re-panicking, so a retry is not stuck at 409 forever.
8. **`platformsettings/crypto.go` is the best crypto in the repo** — AAD binding, key identity for rotation, keyed fingerprints, fail-closed on a missing key. It should be the template for the rest (A2-010).
9. **The container and CI story is strong.** Distroless nonroot static images, multi-arch, `-trimpath`, healthchecks that need no shell. CI runs real Postgres with real migrations and real runtime roles, plus govulncheck, gosec, staticcheck, osv-scanner and trivy.
10. **The comments explain why.** *"Three days, not one: goods released on a Friday afternoon must not be deemed accepted before the buyer reopens on Monday."* Domain reasoning in the code, at the point of decision. This is rarer than any of the above.
11. **Test breadth is real**: 26,761 lines of Go test against 60,701 of source (44%), 48 Playwright specs, plus contract and integration suites and a fuzz corpus.

---

## Part 6 — Suggested order

**Before the next production deploy**
1. A2-002 — financial reads returning empty on failure. Mechanical, highest consequence.
2. A2-007 — nil-transaction panic. One-line fix.
3. A2-001 — withdraw or regenerate the 21 September register. It is currently the evidence for a launch decision.

**Next two weeks**
4. A2-006 — extend the RLS gate, then close `support_cases`, `operation_actions`, `provider_customer_bindings`.
5. A2-003 / A2-004 — collapse the four balance-mutation copies onto `db.UpdateObligationBalanceTx`; route dispute adjustments through the ledger package so they emit events.
6. A2-005 — thread `ctx` through `internal/ledger` and `internal/payments` first.
7. A2-008 — stop failing reads on a pinned aggregate.
8. A2-016 — `__Host-` cookie prefixes and session-bound CSRF. An afternoon.
9. A2-017 — gate identity-document download above `viewer`.

**Next quarter**
10. A2-020 / A2-021 — ESLint + Prettier; move workspace data loading server-side.
11. A2-010 — unify field encryption on the `platformsettings` pattern, with a dual-read migration.
12. A2-011 / A2-012 / A2-013 / A2-014 — reconciliation watermark, outbox batching and ordering, dead-letter path, worker goroutine split.
13. A2-009 — decompose `NewRuntimeWithDB`. Largest, least urgent, highest long-term leverage.
14. A2-022 through A2-026 — CI and infrastructure hygiene.

---

## Part 7 — Coverage

| Area | Files | Treatment |
|---|---|---|
| `cmd/` (9 binaries) | 20 | Entrypoints read in full; `api` and `worker` line by line |
| `internal/` money path — ledger, payments, credit, disputes, operations, billing, schedules, tradelines | ~90 | Read in full |
| `internal/` security path — auth, access, db, config, publictoken, platform/logging, web middleware | ~60 | Read in full |
| `internal/` remaining 35 packages | ~370 | Structure and interfaces read; targeted deep reads on every money, authz, crypto, context and error-handling construct |
| `db/migrations/` | 203 | Policy graph replayed across all 203 to compute the net RLS shape; money column types, reversibility and function contracts swept; individual migrations read where a finding pointed at them |
| `web/src/` | 325 | `hooks.server.ts`, proxy, API client, CSP, service worker, caching read in full; all 187 Svelte files swept for XSS sinks and data-loading pattern; largest components read |
| `infra/`, `.github/`, `scripts/` | ~120 | Dockerfiles, Caddy, Terraform, pg_hba, roles.sql, CI and the RLS gate read in full; 20 workflows' triggers enumerated |
| `docs/` | 210 | Audit ledgers verified against source; the rest not treated as evidence |

Claims in this report were verified against the working tree before being written. Where verification was indirect — the pgx nil-transaction behaviour in A2-007 — the upstream source was fetched and checked rather than assumed.
