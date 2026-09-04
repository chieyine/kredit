# Product and external-dependency decision register

Last reviewed: 3 September 2026

These decisions must be supported by written evidence. A due date is a review
target, not permission to enable a capability. Missing or inconclusive evidence
leaves the corresponding feature gate disabled.

## How to read this register

Every row below now carries a **draft position** so that reviewing is easier
than starting from nothing. A draft position is not an answer. A row becomes
**Approved** only when the accountable owner records the evidence described
under "Decision evidence requirements".

The rows are not the same kind of question, and treating them as one list is
why they are hard to start:

- **Ask the provider, and ask more than one (EXT-001, 002, 004, 014).** These
  are not yours to decide. Some are not even Mono's to decide: Nigerian direct
  debit runs on the NIBSS rail under a CBN scheme regulation, so a scheme-level
  "no" is a no from every aggregator.
  `docs/operations/provider-certification-plan.md` splits the questions into
  scheme questions and aggregator questions, and lists who to send them to.
- **Decide yourself (EXT-003, 005, 006, 009, 010, 012, 013).** These are risk
  appetite and operating choices. Draft positions below are deliberately
  conservative: for a first pilot, the cheapest correct answer to most
  capability questions is "not yet".
- **Have a Nigerian lawyer check (EXT-007, 008, 011, and the tax half of 006).**
  Drafts below exist so counsel reviews a position rather than writing one, which
  is much cheaper. **EXT-011 must not be enabled on the strength of this
  document alone** — see the licensing note in that row.

A pattern runs through the draft positions: **almost every capability question
is answered "off for the pilot."** That is not timidity. Each disabled gate is a
class of failure that cannot happen while you are learning whether the core
product works at all, and each can be turned on later with evidence you do not
have yet.

| ID | Decision required | Accountable owner | Review due | Draft position | Unlocking gate | Status |
| --- | --- | --- | --- | --- | --- | --- |
| EXT-001 | Which collection provider approves supplier-funded B2B trade credit? | Partnerships Lead | 11 Sep 2026 | Mono Sweep as the first adapter, with the same question put to at least two others in parallel; the scheme itself appears not to require the collector to be a lender | `FEATURE_REAL_COLLECTIONS` | Open; awaiting provider |
| EXT-002 | Which approved mandate structures support one-time, variable, recurring, and instalment collection? | Payments Lead | 18 Sep 2026 | One mandate per trade relationship with a total ceiling; confirm ceiling and instalment semantics with Mono | `FEATURE_REAL_COLLECTIONS` | Open; awaiting provider |
| EXT-003 | Is multi-account BVN-linked collection approved, and with what consent and revocation behavior? | Compliance Lead | 18 Sep 2026 | **No. Not for the pilot.** Keep disabled | `FEATURE_MULTI_ACCOUNT_COLLECTIONS` | Draft decided; disabled |
| EXT-004 | What cancellation, reversal, dispute, timeout, and reconciliation guarantees does the selected provider contractually supply? | Payments Lead | 18 Sep 2026 | Unknown until Mono answers the SLA question list | `FEATURE_REAL_COLLECTIONS` | Open; awaiting provider |
| EXT-005 | May collected funds settle directly to the supplier account, and what evidence proves final settlement? | Finance Operations Lead | 18 Sep 2026 | **No. Not for the pilot.** Settle through the platform account and reconcile | `FEATURE_DIRECT_SUPPLIER_SETTLEMENT` | Draft decided; disabled |
| EXT-006 | How are Kredit fees billed, invoiced, taxed, reversed, and disclosed? | Finance Lead | 25 Sep 2026 | Supplier pays; invoice manually, monthly in arrears, during the pilot; VAT treatment confirmed by an accountant | `FEATURE_LIVE_SUPPLIER_BILLING` | Draft decided; automation disabled |
| EXT-007 | What person, business, authority, enhanced-review, and expiry requirements apply at each pilot threshold? | Compliance Lead | 11 Sep 2026 | Full verification for every buyer, no tiering, during the pilot | `FEATURE_REAL_IDENTITY` | Draft; needs legal review |
| EXT-008 | What retention periods, lawful bases, deletion exceptions, and trade-history wording are approved? | Data Protection Lead | 25 Sep 2026 | Draft periods and bases below | `FEATURE_APPROVED_RETENTION_POLICY` | Draft; needs legal review |
| EXT-009 | Which industries, provider accounts, supplier counts, buyer counts, principal limits, exposure limits, and retry limits are approved for the pilot? | Risk Lead | 2 Oct 2026 | Full `PILOT_*` values below | `FEATURE_PRODUCTION_PILOT` plus `PILOT_*` limits | Draft decided |
| EXT-010 | Should buyer silence ever activate an obligation, and is auto-activation wired to a scheduled worker for the pilot? | Risk Lead | Before pilot launch | Yes in principle, **not wired for the pilot**; chase unanswered receipts by hand | `DEEMED_ACCEPTANCE_MIN_HOURS` plus a scheduled worker operation | Draft decided; unreachable by design |
| EXT-011 | What field set may cross-supplier trade history disclose, on what lawful basis, and with what ageing rule for adverse events? | Data Protection Lead | Before sharing is enabled | **No cross-supplier sharing in the pilot.** Collect the data; show each buyer only their own history | `DPIA_REFERENCE`; see `docs/compliance/dpia-trade-history-sharing.md` | Draft; licensing question open |
| EXT-012 | Which operations surfaces does the pilot deployment enable, and who approves adding one? | Operations Lead | Before pilot launch | `ADMIN_SURFACES` list below; the launch owner approves additions | `ADMIN_SURFACES` | Draft decided |
| EXT-013 | What are the falsifiable halt thresholds for the pilot, and who calls a halt? | Launch Owner | Before the first buyer invitation | Ten thresholds in `docs/product/pilot-kill-thresholds.md` | `docs/product/pilot-kill-thresholds.md` | Draft decided |
| EXT-014 | Does the ten-business-day pre-debit notice in the CBN direct debit regulation apply to a first collection, or does the collection date accepted in the agreement satisfy the "as agreed with the payer" limb? | Payments Lead | Before real collections | The accepted collection date is the agreed notice; `COLLECTION_NOTICE_MIN_HOURS` stays at 24 | `COLLECTION_NOTICE_MIN_HOURS` and the schedule design | Open; must be confirmed |

---

## EXT-001 — Which collection provider approves supplier-funded B2B trade credit?

**This is not a decision, it is a question — and it goes to more than one
recipient.** The crux is easy to miss: most sweep and direct-debit products are
written for a **lender** collecting a **loan**. Kredit is not the lender. The
supplier funds the credit from its own inventory and Kredit collects on the
supplier's behalf.

Two things sharpen this. First, Nigerian direct debit runs on the NIBSS rail
under a CBN scheme regulation, so part of the answer is not the aggregator's to
give: a scheme-level "no" is a no everywhere, and asking three vendors would
just buy the same refusal three times. Second, the CBN direct debit regulation
describes a biller as an entity incorporated to carry on business and onboarded
by a bank or payment service provider after due diligence — which suggests
**being a lender is not a precondition**, and a sponsoring relationship is. That
is encouraging, but a provider's own commercial terms can still be narrower than
the scheme permits.

So: ask Mono in writing, ask at least one other aggregator the same question, and
if a sponsor bank or PSP conversation is available, ask them too — they can answer
the scheme-level part authoritatively rather than through a vendor's
interpretation. `docs/operations/provider-certification-plan.md` separates the
scheme questions from the aggregator questions for exactly this reason.

Draft position: Mono Sweep is the first adapter, contingent on written
confirmation, with at least one alternative conversation running in parallel.

## EXT-002 — Approved mandate structures

Partly scheme, partly vendor. NIBSS documents fixed and variable mandates, so
variable-amount collection is a rail capability rather than a Mono feature —
which is good news for portability. What differs by vendor is how that is
exposed, and the answer that most affects the product is instalments: whether a
single authorisation can carry a schedule of separate debits, or whether each
instalment needs its own authorisation. If it is the latter, the instalment
feature is materially more expensive than it looks.

Note one rail-level constraint already confirmed in the regulation: changing the
terms of a mandate requires cancelling it and issuing a new one. Kredit already
behaves this way, and it aligns with business rule 4.

Draft position, to be confirmed: one mandate per trade relationship, carrying a
total ceiling, against which individual debits are drawn.

## EXT-003 — Multi-account BVN-linked collection

**Draft decision: no. Keep `FEATURE_MULTI_ACCOUNT_COLLECTIONS` disabled.**

You can decide this today, and it is the safest decision in the register.
Sweeping across every account linked to a buyer's BVN is the most aggressive
capability in the product. It is the hardest to explain to a buyer at the moment
they authorise it, the most likely to produce a wrongful debit from an account
the buyer did not have in mind, and the most damaging if it goes wrong.

None of that risk buys anything during a pilot. A buyer with one nominated
account and a working mandate is the case you need to prove.

Revisit only if failed-collection recovery is poor **and** the failures are
demonstrably "wrong account had no money" rather than "buyer cannot pay". Those
are different problems and multi-account only helps the first.

## EXT-004 — Provider contractual guarantees

The provider's answer, and the section of the question list where vendors
genuinely differ — reversal windows, webhook guarantees, timeout resolution,
reconciliation, bank coverage and support response. This is what a provider
comparison is actually made of.

Until those guarantees exist in writing, `provider_reliability` on the scorecard
has nothing to be measured against and threshold 9 in the kill-threshold table is
a guess rather than an SLA breach.

Ask one question here early even though it sits in the vendor block: whether an
existing CMMS mandate survives a change of aggregator. If it does, the
permanent-loss contingency becomes far cheaper, and that is worth knowing before
you need it rather than during.

## EXT-005 — Direct settlement to the supplier account

**Draft decision: no for the pilot. Keep `FEATURE_DIRECT_SUPPLIER_SETTLEMENT`
disabled.**

Collected funds settle to the platform account and are reconciled before onward
payment. This is slower for the supplier and it is the right choice at this
stage: it gives you one reconciliation point and one ledger truth, which is what
you want while you are still discovering how the provider behaves.

Two things to be aware of. First, this means Kredit briefly holds money that
belongs to suppliers — **ask a lawyer whether that has licensing implications in
Nigeria before the first naira moves**, because the answer may push you back
toward direct settlement for regulatory reasons rather than operational ones.
Second, the manual settlement step is real work; count it in the cost-to-serve
measurement.

Evidence of final settlement, once enabled: the provider's settlement reference
recorded against the `app.settlement_events` row, reconciled to the bank
statement line, not merely the provider's webhook.

## EXT-006 — Fees: billing, invoicing, tax, reversal, disclosure

**Draft decision: the supplier pays. Invoice manually during the pilot. Keep
`FEATURE_LIVE_SUPPLIER_BILLING` disabled.**

- **Who pays.** The supplier, never the buyer. The buyer's obligation is the
  principal. Adding a buyer-side fee would change what the buyer accepted and
  is a different product.
- **What is charged.** 0.5% of activated principal, plus 0.5% of amounts Kredit
  successfully collects. Rounded down, in the payer's favour (README section 7.3).
- **When.** Monthly in arrears.
- **How, during the pilot.** By hand. At ten suppliers this is under an hour a
  month, and building automated billing before the pricing is proven is
  building the wrong thing carefully. Turn the gate on when manual invoicing
  becomes the constraint.
- **Reversal.** A reversed payment reverses the collection fee on that amount.
  The activation fee is not reversed, because the service — structuring,
  agreement, mandate, evidence — was delivered.
- **Disclosure.** The exact fee appears on the offer the supplier accepts,
  before acceptance, and on the monthly invoice. No fee is ever introduced after
  acceptance.

**Needs an accountant, not this document:** VAT treatment of the service fee and
the collection fee, whether withholding tax applies, and how the invoice must be
formatted for the supplier to claim it. Get this from a Nigerian accountant
before the first invoice, not after.

## EXT-007 — Identity, business and authority requirements

**Draft position: full verification for every buyer, at every size, during the
pilot. No tiering.**

Requirements for every buyer business before any credit is offered:

- business registration verified against CAC;
- the representative's identity verified;
- evidence that the representative may bind the business;
- a bank account reference in the business's name, matched to the business.

Enhanced review above ₦2,000,000 (`PILOT_ENHANCED_REVIEW_KOBO`), meaning a named
human looks at the case before the offer is sent.

Tiered requirements — lighter checks for smaller amounts — are an optimisation
you earn once you know your fraud and default patterns. You do not know them
yet, and a tiered model designed in ignorance is the kind of thing that is
discovered by a regulator rather than by you.

Re-verification: annually, or on any change of representative or bank account.

**Needs legal review.** Specifically: whether Kredit is a reporting entity with
customer due diligence obligations under Nigerian AML rules. Since Kredit is not
the lender and does not hold deposits, the likely answer is no — but "likely" is
not a compliance position, and this is cheap to ask.

## EXT-008 — Retention, lawful bases, deletion exceptions

**Draft position, for legal review:**

| Data | Retention | Draft lawful basis |
| --- | --- | --- |
| Financial records: obligations, ledger, payments, fees, invoices | 7 years from closure | Legal obligation (tax and company record-keeping) |
| Agreements, acceptances, release and receipt evidence | 7 years from closure | Legal obligation; contract performance |
| Identity and KYB evidence | 12 months after the relationship ends | Contract performance; legal obligation while active |
| Bank account references | Deleted on relationship end | Contract performance |
| Mandate records and events | 7 years | Legal obligation; evidence of authorisation |
| Audit events | Life of the platform | Legitimate interest (security and accountability) |
| Notifications and delivery receipts | 24 months | Legitimate interest (evidence a person was told) |
| Support cases | 24 months from closure | Legitimate interest |
| Analytics events | 24 months | Legitimate interest |

Deletion exceptions: an active legal hold, an open dispute, and any record
inside a statutory retention period override an erasure request. The
`app.legal_holds` and `app.processing_restrictions` tables already implement
this; the policy above is what they should be configured to.

**Also needs doing, and probably applies to you:** the NDPC treats an
organisation processing the personal data of **more than 200 data subjects in
six months** as a data controller of major importance, which carries
registration obligations. Fifty buyer businesses plus their representatives plus
supplier staff will approach that quickly. Check whether registration and a
designated data protection officer are required before launch rather than after.

**Trade-history wording** is deliberately not drafted here. See EXT-011.

## EXT-009 — Pilot limits

**Draft decision, based on suppliers whose typical sale is ₦500,000 to
₦2,000,000.** These are ceilings, not targets — nothing should get near them.

```
FEATURE_PRODUCTION_PILOT=true
PILOT_MAX_SUPPLIER_ORGANIZATIONS=10
PILOT_MAX_BUYER_BUSINESSES=50
PILOT_MAX_PRINCIPAL_KOBO=400000000          # ₦4,000,000 per single sale
PILOT_MAX_ACTIVE_EXPOSURE_KOBO=800000000    # ₦8,000,000 outstanding per buyer business
PILOT_ENHANCED_REVIEW_KOBO=200000000        # ₦2,000,000 triggers named human review
PILOT_MAX_DRAWDOWNS_PER_LINE_DAY=3
PILOT_MAX_COLLECTION_RETRIES=2
PILOT_ALLOWED_INDUSTRIES=pharmaceutical_distribution,fmcg_wholesale
PILOT_ALLOWED_PROVIDER_ACCOUNTS=<the specific Mono account identifiers approved>
```

Reasoning:

- **₦4m per sale** is twice the top of the expected range. A sale at the cap is
  an exception that should be noticed, not a routine transaction.
- **₦8m outstanding per buyer** allows roughly four concurrent large orders. Note
  that this limit is **per buyer business across all suppliers**, which is the
  right shape: it caps what a single buyer can owe the platform's suppliers
  collectively, which is the exposure that actually matters.
- **₦2m enhanced review** puts a human in front of the top quarter of sales.
- **Two collection retries.** Three failed attempts against one buyer's account
  is not a collection problem, it is a conversation.
- **Fifty buyers** keeps the population small enough that you can call any of
  them, which is the real pilot control.
- **Industries**: pick the two your first suppliers are actually in and narrow
  this to one if you can. The seed fixtures use pharmaceutical distribution.

## EXT-010 — Should buyer silence ever activate an obligation?

**Draft decision: yes in principle, but do not wire it for the pilot.**

The constraints are now in place (README section 8.3.1, migration 070): a
delivered notice, a buyer with history, and a full 72-hour window. That work was
worth doing, because it means the safe version is what gets switched on if it
ever is.

But at ten suppliers and fifty buyers, an unanswered receipt confirmation is a
phone call, not an automation. Chasing by hand is cheap at this scale, and it
also produces the single most valuable pilot artefact: you find out **why**
buyers do not respond. An automation would convert that signal into silence.

Do not schedule the worker until two things are true: manual chasing has become
the operational constraint, and the candidate-selection defect recorded in
`IMPLEMENTATION_STATUS.md` is fixed (the Postgres sweep currently scans a
process-local cache rather than the database).

`DEEMED_ACCEPTANCE_MIN_HOURS=72` stays configured so the guard is in force the
moment anything does reach that path.

## EXT-011 — Cross-supplier trade history

**Draft decision: do not enable cross-supplier sharing in the pilot. Collect the
data; show each buyer only their own history.**

This defers the hardest question in the register while losing almost nothing.
The history accumulates from day one either way. What is deferred is the
disclosure to a third party, which is where all the risk sits. Meanwhile,
showing a buyer their own record is straightforwardly useful, is the transparency
control the DPIA recommends anyway (section 5), and is the honest start of the
buyer proposition.

**The licensing question, which is why this must not go live on my draft.**
Nigeria's Credit Reporting Act 2017 reserves credit bureau operation to entities
licensed by the Central Bank of Nigeria, and its scope reaches beyond banks —
utilities, telecoms and cooperatives are named as participants in the regime.
Sharing a buyer's payment record with other suppliers so they can decide whether
to extend credit is, in substance, credit reporting. Whether Kredit's design
falls inside that regime, outside it, or can be structured to sit outside it —
for example as buyer-permissioned portability, where the buyer presents their own
record rather than Kredit disclosing it — is a question for a Nigerian financial
services lawyer. It is not a question a draft document can close, and getting it
wrong is a licensing problem, not a fine.

The five NDPA decisions in `docs/compliance/dpia-trade-history-sharing.md` carry
draft answers. Read them together with the licensing question, because the answer
to the licensing question may make several of them moot.

## EXT-012 — Operations surfaces

**Draft decision:**

```
ADMIN_SURFACES=overview,attention,capabilities,approval-inbox,cases,disputes,money,search,users,organizations,jobs,provider-events,audit,analytics,account-recovery,privacy-requests,admin-changes,review-assignments,change-context,change-history,financial-reconciliation,team,metrics,diagnostics
```

Deferred: `business-policies` and `commands`. Policy proposal workflows assume an
operating scale a first pilot does not have, and the operations command path is
the highest-privilege surface in the product — enable it the day you first
genuinely need it, with a record of who asked and why.

The dual-control admin change workflow stays enabled. It is a safety control, not
surface area, and disabling it to shrink the attack surface would enlarge the
risk it exists to manage.

Additions are approved by the launch owner and recorded with the same evidence as
any other privileged grant. See `docs/operations/admin-surface-enablement.md`.

## EXT-013 — Halt thresholds

**Draft decision: the ten thresholds in `docs/product/pilot-kill-thresholds.md`,
with the launch owner calling a halt.**

Every number there is arguable and should be argued with. What is not negotiable
is having a number before the data arrives, because a threshold agreed
afterwards is a rationalisation.

## EXT-014 — Pre-debit notice period

**Draft position: the collection date the buyer accepted in the agreement is the
agreed notice. `COLLECTION_NOTICE_MIN_HOURS` stays at 24. Confirm before real
collections.**

The CBN direct debit regulation describes a minimum of ten business days'
advance notice before a first payment, or before a change to amount or due date
— *or as agreed with the payer*. Kredit's design makes the collection date
explicit in the agreement the buyer accepts, and sends a separate pre-debit
notice before the debit itself, which is a strong fit for the "as agreed" limb.

But it must be confirmed, because the alternative reading is expensive. If a
standalone ten-business-day notice applies to a first debit regardless of what
was agreed, then roughly fourteen calendar days of any credit term are consumed
before collection is permitted, and short-dated trade credit — the common case —
stops working as designed. That would change `COLLECTION_NOTICE_MIN_HOURS`, the
minimum viable term, and the schedule generator's earliest permitted collection
date.

Ask this as question 4 in the provider certification plan. It is cheap to ask
and expensive to discover late.

---

## Decision evidence requirements

To change a row to **Approved**, add a repository-safe evidence reference or an
approved internal record reference and record:

- decision and effective date;
- accountable approver and reviewing functions;
- scope, limitations, expiry/review date, and revocation path;
- provider capability or legal wording relied upon;
- exact configuration or feature flag unlocked;
- required contract tests, monitoring, and runbook changes.

Credentials, full contracts, identity documents, bank information, and other
restricted material must not be committed. Store only the approved reference
and safe decision summary.

## Gate ownership

Engineering owns fail-closed enforcement. The accountable business owner
supplies evidence; Compliance and Security review it; Operations confirms the
runbook. No single function can both supply and solely approve its own evidence.

Where one person currently holds several of these roles, the discipline that
matters is the written record: the decision, the date, and what it was based on.
A draft position in this document is a starting point for that record, not a
substitute for it.
