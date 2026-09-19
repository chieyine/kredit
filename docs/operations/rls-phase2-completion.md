# Phase 2 tenant isolation

The fourteen-table permissive-policy backlog is closed in migrations 158–163.
Migration 164 preserves narrowly authorized operations counts, exact dispute
reference lookup and recovery of a specific pending notification.

| Tables | Authority |
|---|---|
| `privacy_requests`, `privacy_request_events`, `privacy_exports`, `processing_restrictions` | Subject identity or active privacy reviewer |
| `account_recovery_codes`, `account_recovery_requests`, `account_recovery_evidence`, `account_recovery_events` | Subject identity; active recovery reviewers can review requests and events, not code hashes or evidence |
| `correction_requests`, `correction_decisions` | Authorized organization; requester access for personal history |
| `disputes` | Buyer, supplier membership, or active dispute reviewer |
| `repayment_schedules` | Visible obligation |
| `notification_preferences`, `notifications` | Recipient identity |

The broad policies on `schedule_items`, `dispute_evidence` and
`dispute_decisions` are also removed. Their policies inherit the parent boundary.

`db.ScopedDatabase` establishes both user and organization settings inside each
transaction, including single-statement reads. Legacy financial stores receive
request context through immutable `ForContext` views. Collection due-date reads
now carry context. There is no shared mutable request identity on a pool.

Background notification discovery uses bounded functions with a pinned search
path and no PUBLIC execution privilege. Claims retain their lease and retry
checks and run under recipient context. Receipt lookup returns only the fields
needed to match the original provider send. Global operations counts and exact
reference lookup check active platform roles. Message recovery rechecks and
locks operator authority before selecting the pending send's recipient.

## Verification

- `internal/db/isolation_completion_test.go` populates every table and checks
  subject access, unrelated/missing identity denial, and denied cross-subject
  updates for both runtime roles, plus active/revoked operations projections.
- `internal/db/scoped_database_test.go` checks identity clearing, transaction
  release and cancellation with a one-connection pool.
- Repository integration tests exercise notification send/deduplication and
  worker recovery, schedule allocation/reversal, correction review and dispute
  adjustment through restricted runtime logins.
- `internal/usercontrol/isolation_postgres_test.go` covers recovery factors,
  cooling-off, token rejection, private exports and reviewer revocation through
  the restricted application login.
- `scripts/rls-policy-shape-check.sh` now expects zero baseline entries. An empty
  baseline is valid; a reintroduced blanket policy fails the gate.

Tests requiring separate administrator/runtime connections accept
`KREDIT_TEST_APP_DATABASE_URL` and `KREDIT_TEST_WORKER_DATABASE_URL`. Synthetic
integration fixtures belong in a disposable database, never production.

## Deployment

Apply migrations through 164 and reapply `infra/postgres/roles.sql` using the
migration administrator, then deploy the matching API and worker code. Stop or
drain old workers before removing their blanket policies: older repositories
without context propagation are incompatible with the stricter schema. Runtime
startup requires migration 164 and the new SQL capabilities.

Migration 157 repairs forced-RLS audit appends by the exact activity-trigger
owner without granting runtime bypass. Development provisioning preserves the
PostgreSQL bootstrap administrator (OID 10), which PostgreSQL cannot demote,
and provisions separate non-superuser application and worker logins. Ordinary
migration-owner logins are still demoted when requested.

Do not restore blanket policies or grant runtime BYPASSRLS to work around a
failed request. Identify its missing authorized context and use a forward fix.
Local verification does not prove the production deployment has these changes.
