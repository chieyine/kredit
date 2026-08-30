# Persistence migration contract

Priority 4 makes the deployment boundary explicit: a PostgreSQL connection is
not the same thing as a durable application. API and worker startup now verify
the complete state-table contract, and staging readiness remains unavailable
while any financial aggregate is still process-local.

## Durable today

When a database pool is supplied, authentication users, OTP challenges,
sessions, and TOTP methods use PostgreSQL with encrypted recoverable targets and
secrets. The ledger transactions/postings, outbox, audit events, and HTTP
idempotency records also use PostgreSQL. River jobs and the provider
webhook/dead-letter inbox are stored durably. These paths have transaction,
uniqueness, and replay protections.

Organization records, memberships, and invitations now also use PostgreSQL
with request-local tenant context and RLS enforcement.

Buyer invitations, persons, businesses, representatives, verification cases,
consents, and bank-account references now use the PostgreSQL buyer adapter.
Invitation targets remain HMAC-addressable and encrypted for delivery.

Credit requests, payments, schedules, trade lines, collections, disputes,
operations, corrections, reports, notifications, messaging events, documents,
support cases, and relationship consent all use PostgreSQL repositories. Money
mutations commit normalized state, aggregate snapshots, ledger postings, and
outbox events in the same transaction where applicable.

Relationship consent records now use a PostgreSQL repository with request-local
buyer and supplier tenant context and explicit RLS policies.

Provider-neutral mandate state now has a PostgreSQL-backed repository. The
development provider remains deterministic; licensed provider API integration
and certification remain separate release gates.

## Remaining deployment work

There is no remaining process-local repository blocker in a database-backed
runtime. Deployment must still provide certified identity, collection,
messaging, object-storage and document-scanner connectors, plus strict security,
backup/restore, privacy, legal and launch evidence. Development keeps
deterministic in-memory adapters intentionally.

## Required sequence

1. Add repository interfaces at service boundaries without changing API
   semantics.
2. Implement and migration-test one aggregate at a time, beginning with auth,
   organisation, and buyer identity state.
3. Add tenant/RLS transaction coverage, concurrent-write tests, idempotency
   replay tests, and outbox recovery tests for each adapter.
4. Backfill and dual-read only behind an explicit, reversible feature flag;
   never dual-write money without a reconciliation proof.
5. Remove the corresponding in-memory store from the staging runtime, update
   the durability capability, and require a completed persistence-contract
   check in the release evidence.
6. Enable production only after all domain aggregates are durable and the
   signed security, provider, legal, backup/restore, and launch gates are
   complete.

There is no readiness or configuration bypass for this gate. Development may
continue to use deterministic in-memory stores, but staging and production
must fail closed rather than risk split-brain financial state across replicas.
