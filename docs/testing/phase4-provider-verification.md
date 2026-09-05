# Phase 4 — External-provider verification

Status: **started; not certified or approved for production**.

This is the external-dependency phase of the audit, not implementation Milestone 4. It starts on `phase4-provider-verification` from Phase 3 candidate `82bfd4f868d41ed4e085e8b437611bd39a1b0d2f`. Phase 3 PR #3 is still unmerged; its repository-wide CI must pass before the dependent work is promoted to main. Starting this branch does not close Phase 3 or bypass its gate.

## First slice

The new `internal/providers/mono/phase4_contract_test.go` exercises the real adapter code with synthetic responses and local HTTPS servers:

- full, failed, processing, unknown, partial and malformed monetary outcomes;
- reference and mandate identity binding during reconciliation;
- a lost submission response followed by repeated reads using the original reference, without another POST;
- approval and ready-to-debit flags, pause, suspension, cancellation, expiry and rejection mappings;
- refusal to reinstate a cancelled mandate without fresh authorization;
- all twelve accepted callback types, duplicate receipt identity, restricted-data discard and authentication;
- correlation requirements for aggregate and individual partial-debit notifications.

The first review found two fail-open validation gaps and a contradictory-result case. Negative provider amounts could fall through to a positive requested amount; individual debit-attempt notices could omit their correlation reference; a full-success result with positive pending amount was accepted. The guards now classify malformed monetary evidence as pending with zero recognized money and reject uncorrelated debit-attempt notices. They do not turn ambiguity into a retryable failure.

Run the same focused gate locally:

```sh
go test -count=1 -race -timeout 180s \
  ./internal/providers/mono ./internal/collections ./internal/mandates \
  ./internal/identity ./internal/notifications ./internal/web ./internal/config
```

The `Phase 4 Provider Verification / provider-contracts` workflow runs this gate without provider credentials. CI results must be read from the exact commit being reviewed. This document does not claim a run passed before it has completed.

## Evidence levels and outstanding work

| Layer | What it establishes | Phase 4 status |
| --- | --- | --- |
| Synthetic adapter tests | How Kredit handles specified provider-shaped inputs | First slice implemented; consult exact-commit CI |
| Persisted worker/ledger integration | Reservations, deduplication and tenant boundaries survive asynchronous delivery and restart | Existing Phase 2/3 evidence retained; targeted Mono notice-to-ledger replay still required |
| Actual Mono sandbox | What the provider really accepts and returns for this account | Pending; not run in this slice |
| Provider/legal/operational approval | Whether the integration may be enabled for a controlled pilot | Pending; no enablement changes |

Next required provider evidence remains the scenario list in [Mono Sweep acceptance evidence](mono-sweep-evidence.md) and [the sandbox runbook](../runbooks/mono-sweep.md). Capture successful, failed, pending, partial and timeout/unknown outcomes; duplicate, delayed and out-of-order callbacks; mandate pause/cancel/expiry/rejection and fresh authorization; and authoritative debit/settlement reconciliation. Do not equate parser-level duplicate identity with proof of exactly-once ledger effects. Do not treat status mapping alone as proof that an old ready callback cannot reactivate a persisted cancelled mandate.

Before running against Mono, confirm sandbox Sweep access and separate Partial Sweep entitlement, use an isolated database and approved test identities, and store sandbox credentials in secret management. Do not request or commit real BVNs, account inventories, private provider references, authorization URLs or tokens. The hosted buyer consent step and provider certification review require human participation. No real debit, external notification, deployment or provider feature flag was enabled by this slice.

Resolve these contract questions through actual sandbox evidence rather than guessing: the singular/plural retrieve-debit URL discrepancy; final partial-result fields and field presence; request versus provider references; mandate date formats and validity; cancellation/reinstatement semantics; partial-sweep entitlement; settlement and refund evidence. The current adapter remains sandbox-only. Existing production rejection and kill-switch behavior are unchanged.

## Official reference check — 5 September 2026

- [Mono Partial Sweep](https://docs.mono.co/docs/payments/direct-debit/mono-sweep/partial-sweep) documents processing, individual attempt notices, aggregate partial/full results and the collected/pending amounts. Partial Sweep requires separate account access.
- [Mono direct-debit events](https://docs.mono.co/docs/payments/direct-debit/webhook-events) documents mandate and debit callbacks. Callbacks remain signals for server-to-server reconciliation, never direct authority for posting money.
- [Mono Sweep integration](https://docs.mono.co/docs/payments/direct-debit/mono-sweep/integration-guide) distinguishes mandate approval from readiness to debit.

The references support test inputs and the verification checklist; they are not proof that Kredit has passed provider certification.
