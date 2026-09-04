# Pilot kill thresholds

`docs/product/pilot-scorecard.md` deliberately invents no numerical targets:
every metric reports `baseline_required` until real volume supplies reviewed
evidence. That is the right posture for targets. It is the wrong posture for
**stop conditions**, because a threshold chosen after the data arrives is not a
threshold — it is a rationalisation.

The numbers below are **drafts, and they are meant to be argued with.** They are
reasoned from a pilot of up to ten suppliers and fifty buyers, with single sales
of ₦500,000 to ₦2,000,000 (EXT-009). Change any of them you disagree with. What
must not happen is launching with none.

This is a product governance artefact, not a code contract: no number here is
enforced by the application, and none should be, because halting a pilot is a
human decision made with context.

## How to use it

1. Before launch, the launch owner signs each row. A row may be changed; it may
   not be left unsigned.
2. During the pilot, the weekly review reads the current value from
   `/admin/analytics` and records it against the threshold.
3. A breached threshold triggers the stated action **that week**, not at the
   next planning cycle. The action is the point of the row.

A row whose threshold cannot be agreed is itself a finding: it means the team
does not yet share a view of what failure looks like.

## Thresholds

| # | Signal | Scorecard metric | Draft threshold | Action on breach | Owner | Agreed |
|---|---|---|---|---|---|---|
| 1 | Wrongful debit | none — incident record | **Any single confirmed instance** | Halt all collection immediately; no restart before root cause and written fix | Launch Owner | pending |
| 2 | Buyers will not authorise | `mandate_authorization_dropoff` | **Above 40%** | Stop onboarding new buyers; interview every drop-off before any further build | Launch Owner | pending |
| 3 | Collection does not work | `failed_collection_recovery` | **Below 50%** | Suspend new activations; escalate under the provider certification plan | Payments Lead | pending |
| 4 | Goods disputes | `dispute_rate` | **Above 5%** | Pause the affected supplier; review release evidence quality before resuming | Operations Lead | pending |
| 5 | Buyers are not answering | `deemed_acceptance_share` | **Above 10%** | Suspend deemed acceptance entirely; require explicit confirmation | Risk Lead | pending |
| 6 | Unit economics | manual handling cost as a share of fee revenue | **Above 30%** | Reprice, narrow the segment, or raise the minimum ticket before scaling | Launch Owner | pending |
| 7 | Incentive drift | `voluntary_payment_share` | **Below 60%, or a fall of 15 points between reviews** | Review every change to reminders and payment friction since the last reading | Launch Owner | pending |
| 8 | Recognised loss | `recognized_loss_rate` | **Above 2%** | Stop new credit for the affected supplier segment | Risk Lead | pending |
| 9 | Provider reliability | `provider_reliability` | **Below 95%** | Trigger the contingency in `docs/operations/provider-certification-plan.md` | Payments Lead | pending |
| 10 | Support load | `support_intervention_rate` | **Above 40 cases per 100 active suppliers per month** | Freeze supplier onboarding until load per supplier falls | Operations Lead | pending |

## Where the numbers come from

**Row 1 — one.** This row is different in kind from the others. Every other row
is a business signal where reasonable people disagree about the number. A
wrongful debit is money taken from a business that did not authorise it. The
threshold is one, and it is written down now precisely so that it is not argued
about in the moment it happens.

**Row 2 — 40%.** The buyer is asked for full verification and a variable-amount
debit authorisation in exchange for goods they previously received on a
handshake. If more than two in five refuse at that screen, the problem is the
proposition, not the copy, and no amount of supplier enthusiasm compensates.
Below 40% you have a funnel to optimise; above it you have a product question.

**Row 3 — 50%.** If fewer than half of failed collections are eventually
recovered, the mandate is not doing the job it exists to do, and the supplier's
willingness to fund credit rests on something that is not true.

**Row 4 — 5%.** One dispute in twenty is high for goods a buyer ordered and
received. Above that, the likely cause is release evidence quality — vague
delivery records, no waybill — rather than bad faith, and it is fixable before
it becomes a pattern.

**Row 5 — 10%.** With deemed acceptance unwired for the pilot (EXT-010) this
should be zero by construction. The row exists as a tripwire: a non-zero reading
means something reached that path that you did not expect, and that is worth
stopping for.

**Row 6 — 30% of fee revenue.** Kredit earns at most 100 basis points. On a
₦1,250,000 sale that is roughly ₦9,000 of realistic gross fee. Thirty percent of
that is about ₦2,700 — call it 45 minutes of a Lagos operations associate's
fully-loaded time. So: *if the average obligation consumes more than about
three-quarters of an hour of human time, the pricing is wrong at scale.*

Measure it by sampling, not by guessing: for two weeks, whoever handles a
support case, reconciliation, dispute, correction, recovery or privacy request
logs the minutes. Multiply by `manual_touches_per_obligation` from the
scorecard. A reading above **0.5 touches per obligation** should trigger the
sampling exercise even before the cost is known.

**Row 7 — 60% and 15 points.** Kredit earns its collection uplift only on money
it collects, so the pricing quietly rewards buyers drifting to the collection
date. The absolute floor catches a bad steady state; the 15-point drop catches
drift you introduced without noticing. The second is the more useful of the two.

**Row 8 — 2%.** Losses are the supplier's, not Kredit's, but a 2% loss rate ends
supplier willingness to fund credit regardless of who bears it, and Kredit's
reputation goes with it.

**Row 9 — 95%.** Provisional. Replace it with the actual SLA the moment Mono
commits to one in writing (EXT-004). A threshold you invented is much weaker
than a contractual commitment you can point at.

**Row 10 — 40 per 100 suppliers per month.** Roughly one support case per
supplier every two and a half months. Above that, either the product is
confusing or the segment is wrong, and both get worse with more suppliers.

## What is deliberately not here

- **Targets.** Aspirations belong in the scorecard once the baseline exists.
- **Enforcement in code.** Nothing in the application reads this file. A halt is
  operated through the existing controls: suspending activations, disabling
  automatic collection, and the operations command path.
- **Rows for capabilities that are off.** Multi-account collection, direct
  settlement and cross-supplier history sharing are disabled for the pilot
  (EXT-003, EXT-005, EXT-011). They need thresholds before they are enabled, not
  before launch.
