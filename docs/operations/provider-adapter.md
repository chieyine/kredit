# Provider adapter release gate

Milestone 11 keeps real-money collection disabled until an approved provider contract and written compliance approval exist.

## Current development adapter

- `mock-collection` is deterministic and used by unit/contract tests.
- `FEATURE_REAL_COLLECTIONS` defaults to `false`.
- Non-development runtimes use an explicit unavailable identity provider until
  a certified KYC/KYB/authority adapter is configured; the mock identity
  provider is development-only.
- The API exposes `/api/v1/organizations/{organizationID}/provider-status` so operations can see the provider name, capabilities, and feature gate.
- A real provider must implement the provider-neutral `collections.Provider` contract and webhook signature verification. Provider SDK types must not leak into credit, payment, or ledger code.

## Enablement checklist

1. Record the provider contract, licensed account types, mandate scope, supported collection modes, settlement states, reversal semantics, webhook guarantees, and reconciliation API.
2. Obtain written provider/compliance approval and record its reference and approver.
3. Configure `COLLECTION_PROVIDER`, `PROVIDER_APPROVAL_REFERENCE`, and `PROVIDER_APPROVED_BY` through the protected environment. Keep `FEATURE_REAL_COLLECTIONS=false` until sandbox tests pass.
4. Run the common contract suite: authorization/mandate normalization, success, pending-then-webhook success, partial success, retryable failure, final failure, duplicate webhook, invalid signature, out-of-order settlement, settlement reconciliation, and reversal.
5. Verify exact kobo amounts, stable external references, idempotency, reservation release, collection-fee posting, and no “settled” notification before settlement confirmation.
6. Configure a controlled pilot limit. Any amount above the approved limit must be rejected by the adapter.
7. Obtain a human release approval. Production configuration rejects mock
   providers and requires both real-provider feature gates; provider
   certification and pilot enablement remain Milestone 12 release gates.

## Incident and reconciliation rules

- A provider timeout is `UNKNOWN`, never a failure assumption.
- A duplicate provider event is acknowledged without a second financial effect.
- A debit success is not supplier settlement; settlement remains separately observable.
- Reconciliation mismatches remain visible to operations and are never silently discarded.
- Mandate cancellation, reversal, disputed blocks, and provider refusal stop future attempts while preserving the obligation and audit trail.
