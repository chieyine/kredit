# Completing Phase 2 tenant isolation

## What is wrong

PostgreSQL combines **permissive** policies with `OR`. The schema currently has
193 active policies and no `RESTRICTIVE` policy anywhere. On the tables listed in
`docs/compliance/rls-permissive-baseline.txt`, a tenant-scoped policy sits beside
a blanket `current_user IN ('kredit_app','kredit_worker')` policy, so the rule the
database actually evaluates is:

```
tenant_predicate  OR  current_user IN ('kredit_app','kredit_worker')
```

The application connects as exactly those roles — `internal/db/postgres.go` sets
`RuntimeParams["role"]` — so the right-hand side is always true and the tenant
predicate never constrains anything. Tenant isolation on those tables is enforced
only by the Go handlers.

Fourteen tables are affected. A policy that filters through a subquery on another
RLS-protected table is **not** affected: `allocation_access` on
`app.payment_allocations` reads `app.obligations`, and `drawdown_line_access` on
`app.drawdowns` reads `app.trade_lines`. Those subqueries are themselves subject
to the referenced table's policies for the querying role, so they inherit its
tenant scoping. Only a literal `current_user IN (...)` with no tenant predicate
is unconstrained.

Migration `081_phase2_tenant_isolation.sql` did this correctly for the core money
tables: it **dropped** 38 blanket policies before creating 23 tenant-scoped ones.
The tables below are the ones that conversion has not reached.

## Why this is not one migration

The blanket policies are load-bearing today. Three classes of caller rely on
them, and each has to be given a narrow replacement before its table can be
converted. Converting first and fixing after would take the platform down.

**1. Stores that never set tenant context at all.**

| Package | Evidence |
|---|---|
| `internal/schedules` | Every Postgres method uses `context.Background()` and sets no `app.current_*`. `getPostgres`, `allocatePostgres`, `evaluatePostgres`, `collectionTargetPostgres`, `markCollectedPostgres`. |
| `internal/support` | `Get`, `Read`, `Timeline`, `TransitionContext` are keyed by `caseID` alone. |
| `internal/notifications` | No file in the package sets tenant context. |

**2. Worker discovery that is legitimately cross-tenant.**

`referrals.Refresh` opens its transaction with `s.begin(ctx, "", "")` — empty user
and organisation — and `consumer.EnqueueReminders` queries
`app.consumer_reminder_work()` across every tenant. `schedules.evaluatePostgres`
scans all schedule items globally.

**3. Admin reads that are legitimately cross-tenant**, running as `kredit_app`:
`usercontrol.ListRecoveries`, `usercontrol.ListPrivacyReview`,
`platformops.Search`, and the `ops/disputes` and `ops/cases` surfaces.

## The per-table procedure

Do one table per change, in this order. Each step is independently deployable
and independently revertable.

1. **Give the worker a narrow route.** Replace the worker's direct table read
   with a `SECURITY DEFINER` function that returns identifiers only and is
   granted to `kredit_worker`, exactly as `app.collection_due_work_page`,
   `app.drawdown_expiry_tenants` and `app.notification_event_identity` already
   do. Pin `search_path`; revoke from `PUBLIC`.
2. **Give the admin surface a narrow route.** Same pattern, granted to
   `kredit_app`, following `app.admin_user_directory` and
   `app.admin_money_activity`. Revoke from `PUBLIC`.
3. **Make the request path set tenant context.** Thread the request context
   through the store, using `db.SetTenantContext` or, where the obligation is the
   anchor, `db.SetObligationContext`. `tradelines.PostgresStore.ForContext` is the
   pattern to copy; it keeps the signature change to one method.
4. **Drop the blanket policy** for that table in a migration, exactly as 081 did:

   ```sql
   DROP POLICY IF EXISTS <blanket_policy_name> ON app.<table>;
   ```

   Prefer dropping over `AS RESTRICTIVE`. A dropped policy makes the remaining
   permissive tenant policy the whole rule, which is easier to reason about than
   an AND of two predicates.
5. **Remove the table's line** from `docs/compliance/rls-permissive-baseline.txt`.
   `scripts/rls-policy-shape-check.sh` fails if a table is fixed but left in the
   baseline, so the list can only shrink.
6. **Extend the proof.** Add the table to
   `tests/integration/tenant_isolation_test.go` so a future change cannot quietly
   reopen it. That test currently asserts against `app.obligations`, which is one
   of the tables 081 already fixed — which is why this gap stayed green.

## Suggested order

Start where the caller set is smallest.

| Order | Tables | What it needs |
|---|---|---|
| 1 | `app.privacy_requests`, `app.privacy_exports`, `app.privacy_request_events` | Step 2 only. The subject path already sets `app.current_user_id`; the blocker is `usercontrol.ListPrivacyReview`, one admin queue. |
| 2 | `app.account_recovery_requests`, `_codes`, `_events`, `_evidence`, `app.processing_restrictions` | Step 2 only, for `usercontrol.ListRecoveries`. Same shape as above. |
| 3 | `app.correction_requests`, `app.correction_decisions` | Step 3. `corrections/postgres.go` has five direct pool queries and sets no context. |
| 4 | `app.disputes` | Step 3. `disputes/postgres.go` sets context on its three transactions but has seven direct pool queries that bypass them. |
| 5 | `app.repayment_schedules` | Step 3. Every `schedules` Postgres method uses `context.Background()`; `evaluatePostgres` also needs step 1. |
| 6 | `app.notifications`, `app.notification_preferences` | Step 3. No file in `internal/notifications` sets tenant context, and delivery is worker-driven, so it needs step 1 too. |

## Verifying

```sh
bash scripts/rls-policy-shape-check.sh
go test -p 1 -tags=integration ./tests/integration -run TenantIsolation -count=1 -v
```

And the empirical check — as `kredit_app` with a bogus organisation set, a
converted table returns zero rows:

```sh
psql "$APP_DATABASE_URL" -X -c "
SELECT set_config('app.current_organization_id','00000000-0000-0000-0000-000000000000',false),
       set_config('app.current_user_id','00000000-0000-0000-0000-000000000000',false);
SELECT count(*) FROM app.drawdowns;"
```
