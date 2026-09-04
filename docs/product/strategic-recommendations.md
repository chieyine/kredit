# Kredit — Strategic Recommendations

**Prepared:** 3 September 2026
**Basis:** Full file-by-file audit of the Kredit codebase and README build contract, plus `docs/product/pilot-scorecard.md`, `docs/product/open-questions.md`, and the data model in `db/migrations/`.
**Scope:** Product, commercial and risk. Code-level defects and their fixes are recorded separately in `IMPLEMENTATION_STATUS.md`.

---

## Summary

The engineering discipline in this codebase is genuinely unusual for a pre-pilot product: an append-only balanced ledger, idempotency on every financial write, immutable agreement versions, row-level security as defence in depth, and a scorecard that refuses to invent targets it hasn't earned. That rigour is pointed at *correctness*.

My concern is that the two things most likely to end the pilot are not correctness problems. They are a 24-hour clock and a buyer who won't sign a sweep mandate.

| # | Recommendation | Type | When |
|---|---|---|---|
| 1 | Gate deemed acceptance on proven delivery; widen to 72h | Risk — wrongful debit | Before any real money moves |
| 2 | Instrument cost-to-serve per obligation | Commercial viability | Day one of pilot |
| 3 | Name and guard the collection-fee incentive | Incentive alignment | Before pilot |
| 4 | Start Mono production certification now; build a second adapter | Critical path | Now, in parallel |
| 5 | Make mandate-authorisation drop-off the headline metric; give buyers a real reason | Go-to-market | Before first invitation |
| 6 | Pre-register kill thresholds | Governance | Before pilot starts |
| 7 | DPIA the trade-history sharing design | Compliance / moat | Before demand depends on it |
| 8 | Dark-launch the admin surface | Security surface | Before pilot |

---

## 1. Deemed acceptance is the biggest wrongful-debit vector

**Current disposition.** The pilot now rejects automatic activation from buyer
silence at both the worker boundary and the database boundary. The earlier
24-hour auto-activation path is retained only as historical design context; it
is not scheduled and cannot create an obligation in the current pilot policy.

**Why it matters.** This is the one place in the system where *silence* creates a debit. The project's own stated priority order puts prevention of wrongful debit above everything else, including the README itself. This clause is the sharpest contradiction of that order in the product.

The cost of being wrong is not a support ticket. It is an unauthorised debit against a small business's operating account — precisely the failure mode that ends a payments pilot, loses a provider relationship, and attracts a regulator.

**What I'd do.**

- **Keep explicit buyer acknowledgement as the pilot rule.** Connector delivery
  receipts alone are not buyer consent. The current collection boundary requires
  independently authenticated buyer acknowledgement and the waiting period.
- **Widen the floor to 72 hours.** A pharmacy owner who receives goods on a Friday afternoon and closes Saturday will silently activate an obligation. Twenty-four hours encodes a same-day-business assumption that the buyer segment does not live in.
- **Require a positive signal for first-time buyers.** For a buyer's first obligation on the platform, do not deem acceptance at all. Make the first one explicit. The relationship has no history to draw on, and the first wrongful debit is the one that generates the story.

---

## 2. Unit economics are thin against a heavy manual load

**What's there.** One hundred basis points maximum: 0.5% service fee on activated principal, plus 0.5% collection fee charged only on what Kredit successfully collects. Against that revenue, the codebase carries an admin approval inbox, reconciliation cases, dispute review with evidence handling, account recovery, privacy requests, correction workflows, and support cases — roughly twenty-one admin screens of human-in-the-loop operation.

**Why it matters.** On a ₦2,000,000 trade credit, gross revenue is ₦20,000. One reconciliation case plus one support call consumes it. The model works only if the *median* obligation passes through with near-zero human touch, and the exceptions are genuinely rare.

Nobody knows yet whether that is true. It is the central commercial question and it is answerable in the pilot.

**What I'd do.**

- Instrument **cost-to-serve per obligation** from the first supplier: minutes of human time per activated obligation, broken down by workflow (approval, reconciliation, dispute, recovery, support).
- Treat it as a first-class scorecard metric alongside gross credit activated, not as an operations footnote.
- Decide the threshold in advance. If the pilot shows you cannot get below roughly fifteen minutes of human time per obligation, the model needs a higher fee, a narrower initial segment, or a larger minimum ticket. You want to learn that at supplier five, not supplier fifty.

---

## 3. The collection fee points the wrong way

**What's there.** Kredit earns 0.5% only on amounts it collects. A buyer who pays the supplier voluntarily and early therefore generates less revenue than one who drifts to the collection date and gets swept.

**Why it matters.** I do not think this is a live problem — nobody designed it that way on purpose, and the fee structure is otherwise defensible. But it is a structural incentive, and structural incentives shape a hundred small decisions nobody logs: how early reminders go out, how prominent the "pay now" affordance is, whether early settlement is frictionless or merely possible. Left unnamed, you will optimise into it without ever deciding to.

**What I'd do.**

- Write the conflict down explicitly, in the pricing documentation, where a future product manager will find it.
- Add **voluntary payment share vs. swept share** to the pilot scorecard as a guardrail metric.
- Require an explicit, recorded review for any change that reduces the voluntary share. Not a veto — a decision that someone has to make on the record.

---

## 4. Provider concentration is the critical path

**What's there.** Mono Sweep is the only real collection adapter. It is sandbox-only; production is disabled. Everything else in the provider layer is a simulator.

**Why it matters.** The entire collection mechanism — the thing that makes credit collectable, and therefore the thing that makes a supplier willing to fund it from their own inventory — is one provider approval away from existing. There is no product without it, and the approval runs on Mono's calendar rather than yours.

**What I'd do.**

- **Start production certification now**, ahead of the code being finished. Nigerian payments approvals are a lead-time problem, not an engineering problem, and they do not compress.
- **Build a second real adapter before launch**, even if no volume ever routes through it. Not for redundancy — for proof. An abstraction that has only ever met one API is a guess. The second integration is where you discover which parts of the mandate model were quietly Mono-specific, and it is far cheaper to discover that before you have live obligations depending on it.
- Have a written answer to "Mono suspends us on a Tuesday" that does not begin with "we'd need to build."

---

## 5. The buyer side is the hard problem, and the metrics don't yet say so

**What's there.** The asymmetry is stark. The supplier receives a funded trade credit, structured terms, and portable trade history. The buyer is asked for full KYC/KYB, a bank account reference, and a **variable-amount recurring debit mandate** — in exchange for goods they were previously receiving on a handshake.

**Why it matters.** That is a large ask, and the mandate authorisation screen is where the funnel will break. The scorecard tracks invitation-to-verification and sent-to-acceptance conversion, which is right as far as it goes, but the mandate step is the moment a buyer decides whether the whole proposition is worth it.

Today the honest version of the buyer pitch is: *same goods, more paperwork.* That is not a proposition; it is a favour the buyer does for their supplier. Favours do not scale to fifty buyers.

**What I'd do.**

- Promote **mandate authorisation drop-off** to the headline pilot metric. If it is above roughly 40%, the buyer proposition is wrong and no amount of supplier enthusiasm will fix it.
- Give the buyer something concrete and immediate that they cannot get by asking the supplier nicely: a **larger limit**, or **longer terms**, available only through Kredit. The supplier is funding it either way; make the differential explicit and visible to the buyer at the moment of decision.
- Make trade history **explicitly portable** and say so in the buyer flow. The durable reason to accept a sweep mandate is that a buyer's Kredit history unlocks credit at the *next* supplier. That is worth signing for. Convenience is not.
- Interview the first ten buyers who drop off at the mandate screen. That is the highest-value research in the pilot and it will not happen unless someone owns it.

---

## 6. Pre-register kill thresholds before the pilot starts

**What's there.** `docs/product/pilot-scorecard.md` deliberately ships no numerical targets: "No numerical targets are invented in code. Every metric reports `baseline_required` until pilot volume, risk appetite, provider SLAs and the launch owners supply reviewed target evidence."

**Why it matters.** That is intellectually honest and I would keep it — for *targets*. But it leaves a gap: there are no **kill thresholds** either. Targets can wait for evidence. Thresholds cannot, because a threshold chosen after you have seen the data is not a threshold; it is a rationalisation.

**What I'd do.** Before the first invitation goes out, write down and commit to the repository:

- The dispute rate at which the pilot stops.
- The failed-collection recovery rate below which the mandate model is judged not to work.
- The mandate authorisation drop-off above which the buyer proposition is judged wrong.
- The cost-to-serve above which the pricing is judged wrong.
- The wrongful-debit count that triggers an immediate halt. (I would suggest one.)

Pick the numbers while nobody is emotionally invested in the answer. Name the person who calls it.

---

## 7. DPIA the trade-history sharing before demand depends on it

**What's there.** `relationship_consents`, a maintained data map, and a data inventory with its own conformance gate. The bones of a defensible consent model are present.

**Why it matters.** Cross-supplier trade history is the durable moat — it is why a buyer stays and why the second supplier joins. It is also the largest NDPA exposure in the product, because it means disclosing one business's payment behaviour to another business, and the consent model has to carry that weight under scrutiny.

The risk is sequencing. If the go-to-market story comes to depend on history sharing before the sharing design has been assessed, then fixing the consent flow means unwinding the feature that makes the business defensible.

**What I'd do.**

- Run a proper DPIA on the sharing design **now**, while changing the consent flow is cheap.
- Be specific about what is shared: aggregate payment behaviour is a very different disclosure from itemised transaction history, and the moat probably only needs the former.
- Give buyers visibility and control over what their history shows — this is both the compliance answer and, handled well, part of the buyer proposition in §5.

---

## 8. Dark-launch the admin surface

Twenty-one admin screens is a large privileged surface for a pilot of five to ten suppliers. Every admin screen is a path that has to be access-reviewed, audit-logged, and defended.

Put the subset the pilot actually needs behind flags and leave the rest dormant. The code can stay; the routes should not be reachable. Re-enable them as real operational demand appears, which also tells you which ones were speculative.

---

## What is already strong

Worth stating plainly, because it should not be traded away under pilot pressure:

- **The ledger.** Append-only and balanced, with fees rounded down in the payer's favour and a test that enforces it. This is the correct foundation and it is rare to see it built first.
- **Evidence discipline.** Immutable agreement versions, acceptance records, goods release and receipt confirmation as distinct events. When a dispute arrives, you will be able to reconstruct what happened. Most early payments products cannot.
- **Idempotency everywhere on financial writes.** This is what prevents the double-debit incident that would otherwise be inevitable.
- **The refusal to invent numbers.** The scorecard's `baseline_required` posture is the right instinct. Extend it to kill thresholds (§6) rather than diluting it.

---

## Open questions I'd want the founders to answer

1. What is the buyer's answer to "why should I sign a debit mandate?" — in one sentence, in the buyer's own words?
2. If Mono declines or delays production access by six months, what is the plan?
3. Who owns the wrongful-debit decision — the halt call — and what is their number?
4. Is the first pilot segment chosen for ticket size (protecting unit economics) or for relationship density (protecting the trade-history moat)? Those pull in different directions and the answer shapes onboarding.
5. What does Kredit do when a supplier wants to release goods to a buyer whose mandate has failed verification? Today the system is clear; commercially, this is where the first pressure will come from.
