# Phase 4 — External-provider verification

Status: **engineering verification implemented; actual Mono sandbox certification remains pending**.

This is the external-dependency phase of the audit, not implementation Milestone 4. The branch is `phase4-provider-verification`, stacked on Phase 3 candidate `82bfd4f868d41ed4e085e8b437611bd39a1b0d2f`. Phase 3 PR #3 remains an upstream merge prerequisite. Neither a passing focused test nor this document waives the repository-wide CI gate.

## Adapter contract coverage

`internal/providers/mono/phase4_contract_test.go` exercises the real adapter with synthetic responses and local HTTPS servers. It covers full, failed, processing, unknown, partial and malformed monetary outcomes; mandate/reference identity binding; loss of the submission response followed by reads using the original reference; mandate readiness and lifecycle mappings; refusal to restore a cancelled mandate without fresh authorization; all twelve accepted callback types; authentication, duplicate receipt identity and restricted-data discard.

Negative provider monetary values and invalid requested amounts remain pending with zero recognized money. Contradictory full-success results with positive pending amounts also remain unresolved. Aggregate and individual debit-attempt notices must contain a correlation reference. These are conservative validation rules, not claims that every fixture represents an observed provider response.

## Persisted provider-boundary coverage

`internal/providers/mono/phase4_persistence_test.go` uses the actual Mono adapter, PostgreSQL 18, a dedicated non-owner `kredit_worker_login`, transaction-local tenant context, production collection/payment repositories, `ProviderWebhookWorker.Work`, and `Runtime.HandleProviderNotice`.

The tests prove, for their fixtures:

- a reservation and original reference are committed before the outbound debit request;
- individual partial-debit notices do not themselves recognize money;
- five deliveries of the same final notice produce one payment and four recorded inbox duplicates;
- distinct late processing/failure/success notices reconcile against the authoritative lookup rather than overwrite a completed payment from their labels;
- partial results change balances only by the collected amount, failed results recognize zero, and unknown results retain the reservation;
- a lost submission response recovers after reconstructing the worker pool and repositories, without another POST;
- a foreign mandate identity cannot recognize money, and the same failed inbox item can be retried after a matching authoritative result;
- cancellation/expiry survive repository restart and cannot be reactivated by a subsequent lookup;
- an unauthenticated job cannot persist an inbox item or post a payment;
- payment totals, allocations and outstanding balances agree, ledger transactions remain balanced, and durable receipts do not contain the private fixture payload fields.

The active obligation/mandate, due function and eligibility facts are synthetic fixtures. These tests do not prove hosted consent, the full authorization/notice-gate lifecycle, real notification delivery, real settlement/refund behavior or the Mono service. They invoke the production worker directly; they do not prove HTTP ingress or the River dispatcher end to end. The first run exposed an incomplete test fixture: mandate lookup needs the real buyer business/owner relationship, not merely metadata. That fixture was corrected without weakening production ownership checks.

## Repeatable gates

The permanent `Phase 4 Provider Verification` workflow has three checks: `provider-contracts`, `provider-persistence`, and `evidence-tooling`. Read the result for the exact proposed commit before merging. Synthetic persistence logs are retained as CI artifacts for 14 days; they are not uploaded as provider certification evidence.

```sh
go test -count=1 -race -timeout 180s \
  ./internal/providers/mono ./internal/collections ./internal/mandates \
  ./internal/identity ./internal/notifications ./internal/web ./internal/config

# Against a fresh isolated database migrated with all current migrations,
# infra/postgres/roles.sql and infra/postgres/development-logins.sql:
go test -tags=integration -count=1 -race -timeout 180s \
  ./internal/providers/mono -run '^TestPhase4Persisted' -v

python3 -m unittest discover -s scripts -p 'test_provider_evidence.py' -v
```

The integration test requires `DATABASE_URL` for fixture setup and `RIVER_DATABASE_URL` for the dedicated restricted worker. It fails rather than silently skipping when either is absent. Never point the fixture suite at production or at a database containing customer data.

## Actual provider certification: blocked pending external evidence

All 21 real scenarios in [Mono Sweep acceptance evidence](mono-sweep-evidence.md) still require actual sandbox execution. Follow [the sandbox runbook](../runbooks/mono-sweep.md), using all current migrations rather than its historical migration counts. Required external inputs are a Mono payments sandbox with Sweep enabled, separate Partial Sweep entitlement where applicable, approved test identities, an isolated database, a reachable HTTPS callback, and hosted buyer authorization. Store `MONO_SECRET_KEY`, `MONO_WEBHOOK_SECRET` and `MONO_REDIRECT_URL` in approved secret/environment management; never in this repository, CI logs or chat.

Record the exact adapter commit, run date, expected and actual assertions, provider references and reviewer in restricted evidence storage. The [pending manifest template](phase4-provider-evidence.template.json) intentionally contains no passes. Copy it outside the repository; do not commit a completed pack with private references, authorization links, BVNs, account inventories or credentials.

```sh
python3 scripts/verify_provider_evidence.py /restricted/phase4/manifest.json \
  --evidence-dir /restricted/phase4
```

This validator checks completeness, required confirmations and SHA-256 consistency of local evidence files. It cannot authenticate provenance or distinguish a forged assertion from an actual provider observation; a human reviewer must inspect the evidence and provider confirmation. A valid manifest does not enable features or constitute production approval. A pending, missing, incomplete, synthetic-labelled or hash-mismatched pack exits unsuccessfully.

Resolve the singular/plural retrieve-debit URL discrepancy, final partial-result fields, reference identity, mandate validity/date formats and cancellation semantics with actual sandbox/provider evidence. Never add an alternate debit-submission retry or guess that a callback is proof of settlement. Pilot approval also requires the independent legal/security/operational gates.

## Completion boundary

| Evidence | Status |
| --- | --- |
| Synthetic adapter behavior | Implemented; exact-commit CI is authoritative |
| Persisted inbox/reconciliation/payment behavior | Implemented; exact-commit CI is authoritative |
| Evidence-pack validator | Implemented; cannot certify authenticity |
| Actual Mono sandbox scenarios and human authorization | Pending |
| Provider confirmation and human certification review | Pending |
| Upstream Phase 3 merge and repository-wide green CI | Still required |
| Production collection enablement | Unchanged and not approved by this phase |

## Official reference check

The official [Partial Sweep guide](https://docs.mono.co/docs/payments/direct-debit/mono-sweep/partial-sweep), [webhook events](https://docs.mono.co/docs/payments/direct-debit/webhook-events) and [retrieve-debit reference](https://docs.mono.co/api/direct-debit/account/retrieve-a-debit) were reviewed on 6 September 2026. The retrieve-debit operation URL and cURL example still differ. These sources guide the contract checks; they are not evidence that Kredit passed provider certification.
