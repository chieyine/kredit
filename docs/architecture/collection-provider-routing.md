# Collection provider routing

`internal/collections` can now hold more than one collection provider and
choose between them. This document is the reasoning, because the rules are
mostly about what routing must refuse to do.

## The rule everything else follows from

A Nigerian direct debit mandate is an instruction one provider holds against
one bank account. No other provider can present it, and none can adopt one
created elsewhere. So routing has exactly two operations:

- **Selection** happens once, before a mandate exists, and decides which
  provider is asked to create it.
- **Resolution** is a lookup by the provider name already recorded against a
  mandate or a submitted attempt.

There is no third operation. Nothing moves an existing mandate to another
provider, and nothing retries a submitted debit somewhere else. A debit whose
outcome is unknown may already have moved the buyer's money; offering it to a
second provider is how a platform debits a customer twice. Failover belongs at
selection time or not at all.

This is enforced structurally rather than by convention:

- `Engine.Start` resolves the provider once, from the mandate, before it takes
  a lock, and uses that one value for the submission, the attempt record and
  the webhook signature.
- `Engine.debitRegistration` is a map lookup. It has no rank comparison, no
  health check and no second candidate. It returns nil when the named provider
  is missing or is reconcile-only, and the collection is refused.
- Reconciliation already used the attempt's persisted provider name
  (`providerFor`), and still does.
- `Registry.ForMandate` evaluates one provider and returns a refusal with
  reasons rather than a substitute.
- `Engine.SelectProvider` and `Registry.SelectForAuthorization` are the only
  methods that pick between providers, and nothing on the debit path calls
  either.

`TestFailedDebitIsNeverOfferedToASecondProvider` and
`TestMandateProviderIsUsedEvenWhenAnotherRanksAhead` are the tests that hold
this down.

## Roles

| Role | May be selected for a new authorization | May debit mandates it holds | May reconcile |
|---|---|---|---|
| `active` | yes | yes | yes |
| `reconcile_only` | no | no | yes |

`RegisterRetainedProvider` registers `reconcile_only`, and keeps the existing
guarantee that a saved connection is never chosen for new work.
`RegisterActiveProvider` is new and adds a selectable provider alongside the
deployment's configured one.

## Refusal reasons

Selection returns every candidate it considered, eligible or not, each with the
reasons it was passed over, because the reason is the part an operator needs.
`GET /organizations/{id}/provider-status` now carries this as `providers`; its
existing fields are unchanged.

| Reason | Meaning |
|---|---|
| `provider_not_registered` | The name recorded against the mandate is not configured here. Restore the connection; do not re-point the mandate. |
| `provider_reconcile_only` | Retained for existing work only. |
| `provider_not_approved` | The provider's `ApprovalRecord` is absent or invalid. |
| `provider_circuit_open` | The breaker is open. This blocks *selection*; it does not divert an existing mandate. |
| `capability_not_supported:<capability>` | Names the one capability that is missing. |
| `amount_below_provider_minimum` / `amount_above_provider_maximum` | Against the provider's declared bounds. |
| `collection_policy_not_supported` | `ValidatePolicy` refused the obligation's policy for these capabilities. |
| `currency_not_supported` | Only for a provider that declares `SupportedCurrencies`. |
| `no_eligible_collection_provider` | Nothing survived. |

## Determinism

Selection order is `(rank, registration sequence)`, never map order — ranging
over a map would let Go's iteration decide which provider takes a debit. The
deployment's configured active provider carries sequence 0, so it stays ahead
of anything registered at the same rank and a single-provider deployment
behaves exactly as it did.
`TestProviderSelectionOrderIsDeterministic` repeats the check so a map
dependency fails rather than flakes.

## Approval

`ApprovedAdapter` remains the authoritative gate: it refuses an unapproved
debit inside `Submit`. Routing checks approval as well, so an approval gap
shows up as a refusal reason instead of as a failed debit against a customer's
account.

Approval is read through wrappers. `ResilientProvider` exposes `Unwrap()`
rather than forwarding `Enabled()`, because a wrapper that forwards it cannot
distinguish "unapproved" from "no approval record exists" — and in production
those are the two cases that matter most.

`RequireApproval` refuses a provider that carries no record at all. It is off
in the runtime today: turning it on before every configured provider is wrapped
in an `ApprovedAdapter` would make the operations surface report an approved
provider (Mono, approved under `PROVIDER_CERTIFICATION_REFERENCE`) as
unapproved, and a status page that is wrong about approval is worse than one
that is silent. Turn it on with the first provider that is selected rather
than pinned.

## Adding a provider

Routing is provider-agnostic and currently inert: the deployment still has one
active provider, so selection has one candidate. Adding a real provider is:

1. Implement `collections.Provider`, plus `CapabilityProvider` declaring the
   real limits from that provider's documentation, and `WebhookSigner` and
   `CancellationProvider` where the provider supports them.
2. Wrap it in `NewApprovedAdapter` with a real `ApprovalRecord` — a written
   reference, an approver and a pilot limit. Without one it cannot submit.
3. `RegisterActiveProvider` it at a rank, or `RegisterRetainedProvider` if it
   only has history to reconcile.
4. Declare `MinimumAmountKobo` / `MaximumAmountKobo` honestly. These are
   refusal bounds at selection and clamping bounds in eligibility; a wrong
   value either blocks good collections or sends a provider an amount it will
   reject.

Nothing above needs credentials, which is why the routing layer was built
first.

## What is still true after this change

- Mandate creation still happens only on the active mandate account
  (`mandates.Router`), and every saved mandate still resolves locally to one
  original account before a provider request is made.
- A prepared submission still reserves before it can reach a bank.
  `reserveEveryProvider` now wraps every provider the debit could reach, not
  only the configured active one — with more than one provider, wrapping just
  the active one would have let a pinned debit reach a bank before its
  reservation committed.
