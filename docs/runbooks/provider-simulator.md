# Provider simulator runbook

Kredit includes a deterministic local provider simulator for development, contract testing, failure rehearsal, and operator training. It is **not** provider certification and must never be treated as evidence that a live provider has approved Kredit.

Production configuration rejects mock providers. Do not weaken that boundary to make a demo or test pass.

## Start and verify

From the repository root:

```bash
go run ./cmd/provider-simulator
```

The simulator listens on `:8090` by default. To use another local address:

```bash
SIMULATOR_ADDR=127.0.0.1:8091 go run ./cmd/provider-simulator
```

With the simulator running, the built-in health check must return exit code 0:

```bash
go run ./cmd/provider-simulator -healthcheck
```

The HTTP health endpoint is `GET /healthz`.

## Supported provider surfaces

The simulator exposes deterministic local equivalents for:

- person, business, and authority verification;
- mandate creation, lookup, cancellation, and restoration;
- collection submission and lookup;
- notification submission with an `Idempotency-Key`;
- document scanning.

IDs are stable for the same deterministic input where the production contract expects idempotent behaviour.

## Failure and edge-case rehearsal

Use the `X-Simulator-Scenario` header or `scenario` query parameter to request non-happy-path responses. Useful scenarios include `pending`, `failed`, `partial`, `cancelled`, `quarantine`, and `timeout` where the endpoint supports that state.

Example collection rehearsal:

```bash
curl -sS \
  -H 'Content-Type: application/json' \
  -H 'X-Simulator-Scenario: partial' \
  -d '{"external_reference":"training-collection-001","amount_kobo":125000}' \
  http://127.0.0.1:8090/collections
```

For a later collection lookup, the `state` query parameter can simulate a subsequent provider state transition without modifying production data.

## What an operator should prove

For each financial/provider workflow being rehearsed, record only non-sensitive test evidence showing that Kredit:

1. treats provider failures and unknown/pending states as non-success;
2. does not invent a successful collection amount;
3. preserves idempotency for repeated notification submissions;
4. surfaces provider and queue problems through the existing admin provider-events, reconciliation, jobs, attention, and diagnostics views;
5. can recover through the intended protected controls rather than a database edit or a weakened configuration gate.

Run the normal repository test and provider-verification workflows after changing simulator contracts.

## Production boundary

Simulator evidence is engineering evidence only. Before live collections, the release still requires the actual provider certification/approval reference accepted by production configuration, the external Phase 5 approvals, and the exact production-environment checks tracked by the release process.

Never place live credentials, real bank details, real customer identities, production webhook payloads, or provider secrets in simulator fixtures, screenshots, logs, issues, or this repository.
