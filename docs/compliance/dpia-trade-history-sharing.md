# DPIA — cross-supplier trade history sharing

**Status:** draft for review. Every **DECISION** now carries a drafted position
so counsel reviews a proposal rather than a blank page. Nothing here is legal
advice, and the assessment is not complete until the positions are reviewed and
signed by someone qualified to sign them.

**Read section 0 first.** There is a licensing question underneath this
assessment that data protection analysis does not reach, and it may change the
answer to everything below it.

## 0. The prior question: is this credit reporting?

Nigeria's Credit Reporting Act 2017 reserves the operation of a credit bureau to
entities licensed by the Central Bank of Nigeria, and the regime reaches beyond
banks — utilities, telecommunications companies and cooperative societies appear
in it as participants. Under that Act, credit information providers may supply a
bureau without the data subject's prior consent, while disclosures from a bureau
to a user require consent or a data-exchange agreement.

Disclosing a buyer's payment record to other suppliers so that they can decide
whether to extend credit is, in substance, what that regime governs. Three
possible outcomes, none of which this document can choose between:

1. **Inside the regime.** Cross-supplier sharing requires a licence, or requires
   routing through a licensed bureau, and the product design changes.
2. **Outside the regime.** The activity is distinguishable — for instance
   because it is confined to a closed platform and initiated by the buyer — and
   only the NDPA analysis below applies.
3. **Structurable to sit outside it.** The most promising variant is
   **buyer-permissioned portability**: the buyer requests and presents their own
   record to a chosen supplier, rather than Kredit disclosing it to suppliers.
   That framing also strengthens every consent argument in section 3, because
   the buyer initiates rather than merely permits.

**Action:** put this to a Nigerian financial services lawyer before building any
cross-supplier disclosure. It is a licensing question, and a licensing question
answered wrongly is not a fine. Cross-supplier sharing is disabled for the pilot
(EXT-011) precisely so this can be answered without schedule pressure.

Recommended direction, pending that advice: design toward option 3.

**Why now.** Cross-supplier trade history is the durable moat: it is why a buyer
stays and why a second supplier joins. It is also the largest data-protection
exposure in the product, because it discloses one business's payment behaviour
to another business. The sequencing risk is what makes this urgent — once the
go-to-market story depends on history sharing, fixing the consent model means
unwinding the feature the business is defended by. Assess it while changing the
consent flow is still cheap.

`DPIA_REFERENCE` is already a production launch gate in `internal/config`. This
document is the assessment that reference should point at.

## 1. The processing

A supplier considering extending trade credit to a buyer is shown factual
history about that buyer drawn from that buyer's dealings with **other**
suppliers on Kredit. README section 8.9 defines the fields: verified-since date,
completed obligations, total completed principal, current active principal,
largest completed amount, on-time count and percentage, average days late,
unresolved overdue obligations, dispute counts and outcomes, mandate
cancellations while owing, partial recovery history, and repeat supplier
relationships.

Two properties make this materially different from the rest of the product:

- It is a **disclosure to a third party**, not processing for the buyer's own
  transaction. Every other data flow in Kredit serves the sale it belongs to.
- Several fields are **adverse inferences**. "Mandate cancellations while owing"
  and "unresolved overdue obligations" carry reputational weight and can cause a
  buyer to be refused credit.

## 2. Personal data in scope

The subject is nominally a business, but in the Nigerian SME segment the
business and the natural person are frequently inseparable: a sole
proprietorship's payment record is the proprietor's payment record.

**DECISION 1 — drafted: yes, treat it as personal data.** Trade history about a
sole proprietorship or single-owner company is treated as personal data under
the NDPA throughout this design, without waiting for a formal opinion.

The reasoning is that the cost of the assumption is low and asymmetric. Treating
business data as personal data adds consent records, access rights and an ageing
rule you would want anyway; treating personal data as business data removes
protections a regulator may later say were required. Counsel may narrow this
later; nothing is lost by starting from the protective position.

| Data | Source | Sensitivity |
|---|---|---|
| Verified-since date, completed obligation counts and amounts | `app.obligations`, `app.credit_requests` | Commercial |
| On-time percentage, average days late, unresolved overdue | `app.payments`, `app.schedule_items` | Adverse inference |
| Dispute counts and outcomes | `app.disputes`, `app.dispute_decisions` | Adverse inference |
| Mandate cancellations while owing | `app.mandates`, `app.mandate_events` | Adverse inference, high |
| Repeat supplier relationships | `app.trade_relationships` | Reveals the buyer's other suppliers |

The last row deserves separate attention: the set of suppliers a buyer trades
with is commercially sensitive on its own, independent of payment behaviour.

**DECISION 2 — drafted: counts only, never identities.** No supplier is ever
told which other suppliers a buyer trades with, and no field permits that to be
inferred — so no dates, amounts or sequences specific enough to match against a
known transaction.

A supplier's legitimate need is to know whether this buyer is established and
pays. It is not to learn who their competitors are selling to, and a product
that leaks that will lose supplier trust faster than it gains buyer coverage.

## 3. Lawful basis and the consent model

`app.relationship_consents` records, per buyer user and supplier organisation, a
`consent_type`, a `version`, an `evidence_hash`, a `granted` flag and a
timestamp, and `internal/relationships` refuses to record one without an
existing `invited` or `active` trade relationship. Consent is therefore
per-relationship, versioned, evidenced and revocable in structure.

Three gaps stand between that structure and a defensible basis:

1. **Consent to what, exactly.** The schema records a `consent_type` string. It
   does not constrain the field set that type authorises. A consent granted for
   "trade history" today and a widened field set tomorrow are indistinguishable
   in the record.
   **Recommendation:** bind each `consent_type`/`version` pair to an explicit,
   documented field list, and treat widening the list as a new version
   requiring fresh consent — the same rule the product already applies to
   materially changed agreements (business rule 4).
2. **Freely given.** Consent obtained at the moment a buyer needs goods, from a
   supplier they depend on, is under commercial pressure. That weakens consent
   as a basis.
   **DECISION 3 — drafted: neither, as currently framed. Make the buyer the
   discloser.** Rather than choosing between weak consent and a legitimate
   interest that a buyer would struggle to contest, restructure so the buyer
   requests their own record and presents it to a supplier they choose. Consent
   given by someone who initiated the disclosure is meaningfully freely given;
   consent extracted at the moment they need goods is not.

   This is the same direction section 0 recommends for the licensing question,
   which is what makes it worth the extra product work rather than a
   compromise. If counsel prefers a conventional basis, the fallback is consent
   for the disclosure plus legitimate interest for the underlying processing,
   with the balancing test recorded here before enablement.
3. **Revocation semantics.** `granted` is recorded per consent row and
   `internal/relationships` appends rather than mutates, so revocation is
   expressible. What is not yet defined is the effect: a supplier who has
   already seen a history cannot unsee it.
   **DECISION 4 — drafted: revocation is prospective, and the buyer is told so
   in those words.** On revocation, no further disclosure occurs; what a
   supplier already saw cannot be recalled. The consent screen says this
   plainly — not in a linked policy — because a buyer who believes revocation
   is retroactive has not given informed consent, and discovering that later is
   worse than a slightly harder consent screen now.

   Suggested wording to be tested with real buyers: *"You can withdraw this at
   any time. Suppliers you have already shared with will keep what they have
   already seen."*

## 4. Minimisation — the recommendation that most reduces risk

The moat needs a buyer to be able to prove they are good for it. It does not
need the receiving supplier to hold an itemised record of that buyer's dealings
with their competitors.

**Recommendation:** disclose **aggregates and bands**, not itemised history.
Concretely: on-time percentage as a band rather than a figure; completed
obligation and dispute counts rather than individual disputes; largest completed
amount as a band; counterparty identities never disclosed; no dates that allow a
specific transaction to be reconstructed.

This preserves the commercial value — a new supplier learns the buyer is
established and pays — while removing most of the re-identification and
competitive-intelligence risk. It also makes DECISION 2 straightforward.

## 5. Buyer visibility and control

A buyer should be able to see exactly what a supplier sees about them, from
their own portal, before they consent — not a description of it, the actual
rendered view — and to see which suppliers have viewed it and when.

This is both the compliance answer and part of the commercial answer. The buyer
is being asked for full verification and a variable-amount debit authorisation
in exchange for goods they previously received on a handshake. Portable,
visible, buyer-controlled history is the strongest thing on the buyer's side of
that trade. Building it as a transparency control and building it as the buyer
proposition are the same work.

`app.audit_events` already exists for the access log; the buyer-facing view of
it does not.

## 6. Retention and accuracy

- **Retention. DECISION 5 — drafted ageing rule.** Trade history has no natural
  end date, and an adverse inference that never expires is disproportionate.

  | Event | Surfaces for | Then |
  | --- | --- | --- |
  | Completed obligation, on-time payment | Indefinitely | Positive history does not age out |
  | Late payment | 24 months | Drops out of the displayed history |
  | Unresolved overdue obligation | While unresolved, then 24 months from resolution | Drops out |
  | Dispute resolved in the buyer's favour | Never surfaces | Not displayed at all |
  | Dispute resolved against the buyer | 12 months | Drops out |
  | Mandate cancellation while owing | 24 months from settlement of the amount owed | Drops out |
  | Recognised loss / write-off | 36 months | Drops out |

  The asymmetry is deliberate: good history is permanent, adverse history
  expires. A buyer who defaulted three years ago and has since paid twenty
  obligations on time should not be judged by the first one, and a system that
  never forgives has no route back for the buyers who most need one.

  Underlying records are retained per the schedule in EXT-008 regardless; this
  rule governs what is *displayed*, not what is *kept*.
- **Accuracy.** `app.correction_requests` and `app.correction_decisions` give a
  buyer a route to challenge a fact. That route must be reachable from the
  history view itself, and a contested fact should be visibly marked as
  contested while under review rather than presented as settled.

## 7. Risks and treatments

| Risk | Treatment | Status |
|---|---|---|
| Disclosure of a buyer's supplier relationships to a competitor | Aggregate-only disclosure; no counterparty identity | Recommended, §4 |
| Consent widened silently by adding fields | Bind field set to `consent_type`/`version`; new version requires fresh consent | Recommended, §3 |
| Consent not freely given under commercial pressure | Buyer initiates the disclosure; buyer-visible preview before consent | Drafted, §3 DECISION 3 |
| Adverse inference persisting indefinitely | Ageing rule: adverse events expire, positive history does not | Drafted, §6 DECISION 5 |
| Activity falls inside the Credit Reporting Act 2017 licensing regime | Legal opinion before any cross-supplier disclosure; design toward buyer-permissioned portability | Open, §0 |
| Inaccurate history causing credit refusal | Correction route reachable from the history view; contested facts marked | Recommended, §6 |
| Buyer unaware of who has seen their history | Buyer-facing access log from `app.audit_events` | Recommended, §5 |
| Sharing enabled before this assessment completes | Feature stays disabled until the decisions are signed | Required |

## 8. Outcome

**Cross-supplier sharing is disabled for the pilot** (EXT-011). Trade history
still accumulates from day one, and each buyer is shown their own record. That
sequencing costs almost nothing and defers every risk in this assessment until
it can be answered without schedule pressure.

Before any cross-supplier disclosure to a real buyer, all of the following must
be true:

1. A Nigerian financial services lawyer has answered the licensing question in
   section 0, in writing.
2. Draft DECISIONs 1–5 have been reviewed and signed by someone qualified to
   sign them, or replaced with better answers.
3. Disclosure is aggregate-only per section 4, with no counterparty identities.
4. The buyer-visible preview and access log in section 5 exist and work.
5. The ageing rule in section 6 is implemented, not merely documented.

The engineering foundations — per-relationship versioned consent, row-level
security, audit events, a correction workflow — are already in place. What is
missing is the scoping decision about what is actually disclosed, and the legal
answer about whether Kredit may disclose it at all.

Review this assessment before any change that widens the disclosed field set,
changes the lawful basis, or extends sharing beyond supplier organisations.

## Sources consulted for section 0

- [Analysis of the Credit Reporting Act 2017 — practical issues arising](https://www.mondaq.com/nigeria/consumer-credit/757320/a-critical-evaluation-of-the-credit-reporting-act-2017-practical-issues-arising)
- [A critical analysis of credit information sharing under Nigeria's Credit Reporting Act 2017](https://www.financierworldwide.com/a-critical-analysis-of-credit-information-sharing-under-nigerias-credit-reporting-act-2017)
- [NDPC guidance notice on registration of data controllers and processors of major importance](https://kpmg.com/ng/en/home/insights/2024/03/nigeria-data-protection-commissions-guidance-notice-on-registration-of-data-processors-controllers-of-major-importance.html)

These are secondary summaries, not the legislation, and are cited to show what
prompted the question — not as a substitute for the Act or for advice on it.
