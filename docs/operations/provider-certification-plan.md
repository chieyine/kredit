# Collection provider certification and contingency

Kredit's collection mechanism is what makes trade credit collectable, and
therefore what makes a supplier willing to fund it from their own inventory.
Today that mechanism is a single adapter — Mono Sweep — which is sandbox-only,
with production disabled (`MONO_SWEEP_ENABLED`, `FEATURE_REAL_COLLECTIONS`).
Everything else in the provider layer is a simulator.

This is a scheduling risk more than an engineering one. Provider approvals run
on the provider's calendar, and no amount of finished code compresses them. It
belongs on the critical path, ahead of feature work.

It is a *concentration* risk rather than a *capability* risk. Other Nigerian
institutions can plausibly do the same thing; several simply do not document it
publicly, which makes discovery slow rather than impossible. The next section
explains why that distinction changes what to do about it.

## Critical path

1. **Start certification before the code is finished.** Sandbox certification
   and production access are sequential and externally paced. The application
   already refuses to enable real collections without written approval evidence
   (`PROVIDER_APPROVAL_REFERENCE`, `PROVIDER_APPROVED_BY`,
   `PROVIDER_APPROVED_AT`, validated in `internal/config`), so beginning the
   approval early cannot cause an accidental live debit.
2. **Record the approval as evidence, not configuration.** The approval
   reference, approver and approval time are read at startup and refused if
   absent, in the future, or placeholder. Keep the underlying written approval
   where the release owner can produce it.
3. **Complete the sandbox scenario matrix** before requesting production:
   success, pending, partial, retryable failure, non-retryable failure,
   settlement, reversal, duplicate event, out-of-order event, and signature
   forgery. `internal/collections/provider_contract_test.go` and
   `tests/contract/provider_contract_test.go` hold the shared assertions.
4. **Agree the provider SLA in writing**: submission latency, settlement
   timing, webhook delivery guarantees, retry semantics, and the support path
   for a stuck attempt. `provider_reliability` on the pilot scorecard measures
   the SLA once it exists; before that, it has nothing to be measured against.

## The rail underneath: NIBSS, not the aggregator

Before treating Mono as a single point of failure, it is worth being precise
about what would actually be lost. Nigerian direct debit runs on **NIBSS Direct
Debit (NDD)** with mandates held in the **Centralised Mandate Management System
(CMMS)**. NIBSS documents fixed mandates, **variable mandates**, and Global
Standing Instruction for BVN-linked recovery, and lists fintechs and payment
service providers among its participants alongside banks. Aggregators sit on top
of that rail; at least one competitor, Fincra, publicly documents a variable
recurring direct debit API described as powered by NIBSS e-mandate.

Three consequences, and they pull in different directions.

**Portability is better than it looks.** If the mandate semantics Kredit depends
on are NIBSS semantics rather than Mono semantics, an aggregator swap is much
closer to a configuration change than a rebuild. The test of that is whether
Kredit's mandate model is expressed in rail concepts — mandate reference,
variable mandate, amount ceiling, mandate status, cancellation — or in the shape
of one vendor's JSON. That is what the adapter registry in
`tests/contract/provider_contract_test.go` is really checking.

**A rail-level or regulatory constraint cannot be fixed by switching vendors.**
This is the part worth internalising. If the reason Kredit cannot collect is
that the scheme rules do not contemplate this biller relationship, every
aggregator returns the same answer, and discovering that after three integration
conversations is expensive. So the question list below is split accordingly.

**Capability is not the same as availability.** Several institutions can
plausibly do this and do not document it publicly. That does not make them
unavailable; it makes discovery slow, which is an argument for starting those
conversations in parallel now rather than sequentially after Mono answers.

### One encouraging finding

The CBN Regulation for the Direct Debit Scheme (2018) describes a biller as an
entity incorporated or registered to carry on business, onboarded by a bank or
payment service provider after due diligence. On that reading, **being a lender
is not a precondition for originating a direct debit** — a sponsoring bank or
PSP relationship is. That materially reduces the worry that supplier-funded
trade credit falls outside the scheme by construction. It does not settle it:
an individual provider's own terms may still be narrower, which is why question
1 below is still the first thing to ask.

### Two things the same regulation raises

- **Pre-debit notice.** The regulation describes a minimum of ten business days'
  advance notice before a first payment or a change to amount or due date, *or
  as agreed with the payer*. `COLLECTION_NOTICE_MIN_HOURS` currently defaults to
  24. The likely reading is that the collection date the buyer accepted in the
  agreement **is** the agreed notice, which is exactly why Kredit makes that date
  explicit at acceptance — but this must be confirmed, because if a standalone
  ten-business-day notice applies to a first debit, short-dated trade credit is
  squeezed and the schedule design changes.
- **Amendment requires a new mandate.** The regulation states that any change to
  the terms of a mandate requires cancelling the existing mandate and issuing a
  new one. Kredit already behaves this way — a restored authorisation gets a new
  provider identifier, and `tests/contract/provider_contract_test.go` asserts it
  — and it aligns with business rule 4, where a materially changed agreement
  requires a new immutable version and fresh acceptance. Keep both.

## Candidates to approach in parallel

Send the questionnaire to more than one. The marginal cost of a second and third
conversation is an email; the cost of discovering the answer sequentially is
months. A second provider also gives negotiating position and the design input
the second adapter needs.

Approach, in rough order of how much is publicly knowable today:

1. **Mono** — current adapter, sandbox integration already built.
2. **A provider that publicly documents variable direct debit on NIBSS
   e-mandate** — Fincra documents one; there may be others by the time this is
   read.
3. **A bank or PSP sponsor directly** — the biller relationship has to be
   sponsored by one regardless, and a sponsor can answer the rail-level
   questions authoritatively rather than through an aggregator's interpretation.

Record what each says against the same question list so the answers are
comparable rather than anecdotal.

## The questions to put to every provider, in writing

EXT-001, EXT-002 and EXT-004 in the decision register cannot be answered
internally. They are answered by this list, from someone empowered to answer it.
Verbal reassurance is not evidence; ask for email.

Questions 1 to 7 are **rail and scheme questions**: if the answer is no, it is
probably no everywhere, and a different aggregator will not change it. Ask these
first, of whoever can answer them soonest — a provider, a sponsor bank, or
NIBSS. Questions 8 to 17 are **aggregator questions** whose answers legitimately
differ between vendors, and are what you actually compare providers on.

**Question 1 is the one that matters most. Ask it first, on its own, before
investing further engineering effort.**

### Rail and scheme questions

1. **Use case.** Sweep and direct-debit products are generally written for a
   lender collecting a loan. Kredit is not the lender: a supplier extends trade
   credit from its own inventory and Kredit collects on the supplier's behalf.
   Is supplier-funded B2B trade credit a permitted use? If it requires a
   specific agreement or classification, which?

2. **Who is the biller?** The scheme puts the mandate between a biller and a
   payer. Should Kredit be the biller collecting on the supplier's behalf, or
   should the supplier be the biller with Kredit as technical operator? This
   changes the mandate wording the buyer sees, who carries the indemnity, and
   who the payer's bank holds responsible.

3. **Sponsorship.** Which bank or payment service provider sponsors the biller
   relationship, what due diligence do they require, and how long does
   onboarding take? This is the item most likely to sit on the critical path.

4. **Pre-debit notice.** The CBN direct debit regulation describes a minimum of
   ten business days' advance notice before a first payment or a change of
   amount or date, *or as agreed with the payer*. Does the collection date the
   buyer accepts inside the Kredit agreement satisfy the "as agreed" limb, or
   does a separate ten-business-day notice apply to a first debit? *(If the
   latter, short-dated trade credit is squeezed and `COLLECTION_NOTICE_MIN_HOURS`
   and the schedule design both change.)*

5. **Mandate ceiling semantics.** Does one authorisation carry a total ceiling
   across all debits, or a per-debit ceiling? What happens when a schedule would
   exceed the remaining ceiling — refusal, partial debit, or error?

6. **Instalments.** Can a single authorisation carry a schedule of separate
   debits on different dates, or does each instalment require its own
   authorisation? *(If the latter, the instalment feature is materially more
   expensive than it looks and that changes the roadmap.)*

7. **Variable amounts.** Can the debited amount differ from the amount shown at
   authorisation, within the ceiling? What must the buyer have been told for
   that to be permitted, and must the mandate be explicitly marked variable?

### Aggregator questions

8. **Cancellation and restoration.** When a buyer cancels, does the
   authorisation identifier become unusable permanently? Can it be restored, or
   must a new authorisation be created? How quickly does a cancellation take
   effect against an in-flight debit, and does it take effect immediately or at
   the end of the current cycle?

9. **Reversal.** Under what circumstances can a settled debit be reversed, by
   whom, within what window, and how is Kredit notified?

10. **Disputes.** What is the buyer's route to dispute a debit with the provider
    or their bank, what is the timeline, how does that interact with the CBN
    dispute resolution mechanism, and what is Kredit's obligation and liability?

11. **Timeouts and uncertainty.** If a debit submission times out, what is the
    authoritative way to determine whether it was applied? Is there an
    idempotency key or an unambiguous reference lookup?

12. **Webhook guarantees.** At-least-once or at-most-once? Are events ordered?
    What is the retry schedule, and how long are undelivered events retained?

13. **Settlement.** What evidence constitutes final settlement rather than
    provisional success, and what is the timing?

14. **Reconciliation.** Is there a statement or reconciliation endpoint that
    lists all debits and settlements for a period, and at what granularity?

15. **Bank coverage.** Which banks are supported, and which are not? A buyer
    whose bank is unsupported cannot be onboarded at all, and that constraint
    belongs in the pilot segment decision rather than discovered per buyer.

16. **Support and SLA.** What are the committed submission latency, webhook
    delivery and support response times, and what is the escalation path for a
    stuck attempt affecting a real buyer?

17. **Suspension.** Under what circumstances would access be suspended, with what
    notice, and what happens to authorisations already in force?

Record the answers as the written evidence for EXT-001, EXT-002 and EXT-004.
Question 16's answer becomes the real threshold for row 9 of
`docs/product/pilot-kill-thresholds.md`, replacing the provisional 95%.

## Building the second adapter before launch

Build a second real adapter before launch even if no volume ever routes through
it. The reason is not redundancy. It is proof.

An abstraction that has only ever met one API is a guess. The second
integration is where it becomes clear which parts of the mandate model were
quietly Mono-specific — identifier shape, authorisation session semantics,
whether a cancelled mandate can be restored or must be recreated, how partial
recovery is reported, what a webhook signs over. Discovering that with live
obligations in flight is materially more expensive than discovering it now.

`tests/contract/provider_contract_test.go` is the mechanism: adapters are
registered in one table and the shared assertions run against all of them.
Adding an adapter should be a one-line registration plus its own package tests.
If it needs more than that, the boundary leaked, and that leak is the finding
the exercise was for.

## Contingency: the provider suspends or delays us

The answer must not begin with "we would need to build."

- **Delay before launch.** Collections stay disabled; the product still
  functions for suppliers who accept voluntary payment, because the payment
  sources model already supports supplier-recorded transfers, buyer payment
  claims and cash records (`app.payments.source_type`). The pilot proposition
  narrows to structured terms, evidence and trade history — weaker, but real.
  Say this to design partners before they discover it.
- **Suspension after launch.** Existing obligations remain valid and
  collectable by other means; a mandate suspension does not erase an accepted
  obligation (business rule 11). Automatic collection is disabled through
  configuration, buyers are told through the existing notification path, and
  outstanding balances move to voluntary settlement.
- **Permanent loss.** The second adapter, already written and contract-tested,
  becomes the primary. Buyers must re-authorise: a mandate is bound to the biller
  relationship and cannot simply be repointed. That re-authorisation campaign is
  the real cost, and its expected drop-off is the number worth estimating in
  advance — `mandate_authorization_dropoff` from the pilot is the best available
  estimate. Whether an existing CMMS mandate can survive an aggregator change is
  itself worth asking under question 8; if it can, this contingency becomes much
  cheaper and that is worth knowing before you need it.

## Ownership

Certification has a named owner with a date. `provider_reliability` and the
row 9 threshold in `docs/product/pilot-kill-thresholds.md` are reviewed weekly
against the agreed SLA once the SLA exists.

## Sources for the rail description

- [NIBSS Direct Debit (NDD) and the Centralised Mandate Management System](https://nibss-plc.com.ng/nibss-direct-debit-ndd/)
- [CBN Regulation for the Direct Debit Scheme in Nigeria, 2018 — summary](https://www.tekedia.com/details-of-the-revised-cbn-regulations-for-the-direct-debit-scheme-in-nigeria/)
- [CBN Regulation for the Direct Debit Scheme in Nigeria, 2018 (primary)](https://www.cbn.gov.ng/out/2018/bpsd/regulation%20for%20direct%20debit%20scheme%20in%20nigeria%202018%20(revised).pdf)
- [Fincra direct debit and mandate management documentation](https://docs.fincra.com/docs/direct-debit)

The regulation summary above is secondary. Read the primary regulation before
relying on the notice-period and biller-eligibility points, and confirm both
with whoever sponsors the biller relationship.
