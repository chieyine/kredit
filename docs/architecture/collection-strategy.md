# Enforcement strategy

Researched 16 September 2026. Sources at the bottom. Claims that are unproven
negatives or that depend on a rule-versus-statute gap are marked as such.

## 1. What Kredit is

The seller decides who gets goods credit. He has always done this. Kredit does
not lend, does not underwrite, and does not control the order. Kredit's promise
to the seller is that he gets paid.

So Kredit is not a collections rail. It manufactures **consequence**. The seller
already owns the last lever — stop supplying — and does not need Kredit for it.
What he lacks is everything that makes the debt real *before* it reaches that
point.

That reframing matters because it decides what to build. A payment integration
that moves money is worth little here. An instrument that makes non-payment
expensive is worth a great deal.

## 2. Why cash-side collection cannot be the product

Every cash-side mechanism — direct debit mandates, GSI, sweeps, a dedicated
account with a split on inflows — assumes leverage over the buyer's money. A
buyer who knows an account will be debited moves his money, or tells his
customers to pay elsewhere. He can do it in a day, for free, from his phone.

This is the documented reason GSI under-delivers: borrowers move between banks,
neobanks and wallets faster than any recovery instruction follows them. Do not
build the product on the one asset the buyer can relocate instantly.

## 3. The five things Kredit puts on the table

### 3.1 Turn the notebook entry into a registered secured claim

This is the headline, and it is the thing no individual seller will ever do for
himself.

The Secured Transactions in Movable Assets Act 2017 defines a creditor as "the
person granting a facility on the back of a security interest created under
this Act" and a security interest as "a property right in collateral that is
created by agreement and secures payment or other performance of an
obligation". Collateral is "movable property, whether tangible or intangible".
It covers inventory — goods "held for sale or lease in the ordinary course of
business" — and receivables, and a security interest extends automatically to
"identifiable or traceable proceeds".

That last clause is the legal answer to *he will just move his money*. The
security follows the proceeds of the goods. A mandate cannot do that.

On default the Act gives repossession after 10 days' notice, sale after 10
working days' notice, and priority ranked by order of registration.

Registering a financing statement per invoice is administrative work no seller
would do for a single order and that a platform can do for every order. It
converts unsecured trade credit into a dated, priority-ranked, enforceable
claim.

**The constraint, stated plainly.** The Act says "a creditor" may register. The
CBN's operating rules for the National Collateral Registry do not: they say "it
is the responsibility of the financial institutions to register movable assets"
and list deposit money banks, finance companies, merchant banks, microfinance
banks, development finance institutions and non-bank financial institutions.
Suppliers cannot register directly today. Registration fees are ₦1,000 for
DMBs and finance companies, ₦500 for MFBs, DFIs and non-bank FIs, and ₦500 for
a public search.

So this instrument needs a financial-institution partner or licence. See
section 5. Do not promise sellers secured status until that is resolved with
NCR and with counsel.

### 3.2 Tell the seller who he is dealing with

He knows the buyer by face and market. He does not know the buyer owes four
other suppliers. An NCR search shows whether the buyer's inventory is already
encumbered and by whom, ranked by date. Kredit's own network record shows how
he pays other sellers on the platform. Bureau data shows formal exposure.

The seller still decides. He simply stops deciding blind. That is the product
he is buying, and it is worth more to him than any debit.

### 3.3 Do the chasing so he does not have to

A seller does not want to hound his own customer — he wants to keep selling to
him. A third party absorbing the unpleasantness preserves a commercial
relationship that direct pursuit would damage. This is a real service and it is
most of why enforcement-as-a-service sells at all.

### 3.4 Make the consequence multilateral

Default against one seller freezes the buyer across every seller on Kredit, and
is reported to a credit bureau. This converts a bilateral sanction the buyer can
absorb into a market-wide one he cannot.

Nigeria's Credit Reporting Act 2017 contemplates non-bank participants,
including suppliers of goods or services on credit, and section 9(2)(a) permits
a credit information provider to disclose to a bureau without the data
subject's prior consent, though the 2013 Guidelines recommend notice.
**Confirm data-provider eligibility and onboarding directly with CRC and
FirstCentral** — the statute is permissive but the bureaus run their own
admission. CreditChek's RecovaPRO offers opt-in bureau reporting as a
ready-made route.

### 3.5 Make payment unambiguous

A virtual account per obligation, auto-reconciled to that invoice. This kills
"I paid o" disputes and cash-at-delivery leakage inside the seller's own
operation — which in Nigerian distribution often costs the seller more than
actual buyer default does. Paystack dedicated virtual accounts, Monnify,
Fincra, Korapay and the Providus/Wema/Safe Haven products all do this, and
none requires a lending licence.

## 4. Where direct debit actually belongs

**A failed debit is not a failed collection. It is evidence.**

A dated, third-party-verified record that the buyer signed a mandate and then
had no funds is exactly the raw material an enforcement product needs: it
justifies escalation to bureau reporting, supports a repossession notice under
section 40, and makes a demand letter credible. It converts "he is avoiding me"
into a documented fact.

Three consequences:

1. **Mono-grade sweep is unnecessary.** Any aggregator produces the same
   evidence. Paying Mono's licence gate for a sweep feature the product does
   not rely on is paying for the wrong thing.
2. **Willingness to sign a mandate is itself the signal.** Sellers on the
   platform can offer better terms to buyers who sign one. Refusal is priced.
   The mandate earns its keep at underwriting whether or not it ever collects.
3. **Choose the aggregator on onboarding speed, not capability.** Which is what
   the routing layer in `internal/collections` is for.

### Provider facts for adapter `Capabilities`

| Provider | Minimum | Personal max | Corporate max | Banks | Gate |
|---|---|---|---|---|---|
| Mono | NGN 200 | NGN 25m | NGN 250m | 23 | Licence table |
| RecovaPRO | NGN 250 | NGN 20m | NGN 200m | — | Onboarding only |
| Flutterwave | — | — | — | 23 | Request access from support |
| Paystack | — | — | — | 23 | None stated in docs |
| Monnify | — | — | — | 26 | Activation email |
| Fincra | — | — | — | 25 | None stated in docs |

Mono's licence table is Mono's own KYB policy, not a rail requirement, and it
reads "Money Lenders License, FCCPC certification, **or** Certificate of
compliance" — alternatives, not a stack.

The CBN *Guidelines on Nigeria Direct Debit Scheme* define an Originator as
"an organisation that is able to make a Direct Debit Transfer", requiring only
that it "must be a customer of a bank whose responsibility it would be to
support the application" and that it "must execute an Indemnity, obtained from
the Originator's Bank". No lending licence appears in the regulation.

Mandate activation is never instant: Mono says 5 minutes to 24 hours and
occasionally past 48; Paystack says up to 24 hours and sometimes longer;
Flutterwave says typically under 3 hours for new customers. Create mandates at
onboarding, never at the point of collection.

## 5. The consolidation: one licence replaces eight integrations

The two most valuable instruments — registering security interests, and being a
direct debit Originator in Kredit's own right — are both gated on being a
financial institution, not on payment integrations.

The **CBN Finance Company licence** is the cheapest wrapper that clears both.
Minimum share capital ₦100 million, application fee ₦100,000, licensing fee
₦250,000. Permitted activities include "local and international trade finance,
debt factoring, debt securitization, debt administration ... warehouse receipt
finance" — which is Kredit's business described in a licence category.

For comparison: Tier 2 Unit MFB ₦50 million, Tier 1 Unit MFB ₦200 million,
State MFB ₦1 billion, National MFB ₦5 billion.

What the licence unlocks in one move:

- NCR registration rights, so section 3.1 becomes real.
- Debt factoring, which is the endgame product: the seller gets paid now and
  Kredit holds the receivable. That is what sellers actually want, and it is
  worth far more than a collection fee.
- A path into GSI if CBN's stated expansion to fintech lenders and MFIs lands —
  described in February 2026 as "underway, with phased completion expected by
  2026", though as of that date fintechs were still excluded in practice and
  only NIRSAL MFB was cited as having deployed it. Do not plan around this.
- It settles the Mono question permanently, and makes Kredit an Originator
  rather than a sub-merchant of somebody else's participation.

Until then, a partnership with an existing finance company or MFB can register
NCR interests on sellers' behalf.

## 6. Sequencing

**Now, no licence needed:**

1. Virtual account per obligation, auto-reconciled. Highest yield per unit of
   engineering effort in this entire document.
2. The network record and the cross-seller freeze. This is the enforcement
   product and it is pure software.
3. One direct debit aggregator behind the routing layer, for signal and
   evidence — whichever onboards fastest, with a real `ApprovalRecord`.
4. Credit bureau reporting, once eligibility is confirmed.

**Next:**

5. A finance company or MFB partner to register NCR security interests for
   sellers, with the per-invoice registration flow automated.

**Endgame:**

6. Finance Company licence, then factoring.

## 7. What not to count on

- **GSI.** Restricted to CBN-licensed institutions connected to NIP; fintechs
  still excluded in practice as of February 2026.
- **Open banking.** Did not go live in August 2025 as announced. In October
  2025 a CBN payments policy official said on record that it "hasn't gone live
  yet". No evidence of a live registry or any registered non-bank API consumer
  as of September 2026.
- **Okra.** Wound down May 2025. Not a viable counterparty.
- **Balance checks.** Among Paystack, Flutterwave, Monnify, Fincra, RecovaPRO,
  Adjutor and Zeeh, none publicly documents a pre-debit confirmation of funds;
  Mono does. That is an unproven negative — do not state Mono's uniqueness as
  fact externally.
- **Repossession speed.** The statutory right is real; Nigerian enforcement is
  slow. Its value is as a credible threat that resolves matters before it is
  exercised.

## 8. Open questions that are not engineering

- Can a non-financial creditor register at NCR in any form today, or is a
  financial-institution partner mandatory? The Act and the CBN's registry rules
  disagree. Resolve with NCR and counsel before promising sellers secured
  status.
- Does registering security interests on sellers' behalf make Kredit an agent
  with duties to them, and what does that require?
- Is Kredit a money lender under Nigerian law when the seller is the creditor
  and Kredit carries no credit risk? This changes once factoring begins.

## Sources

- Secured Transactions in Movable Assets Act 2017, sections 9, 12, 23, 40, 45, 63
- CBN, National Collateral Registry guidelines; NCR portal
- CBN, Guidelines on Nigeria Direct Debit Scheme (and Revised Version 0.2)
- Credit Reporting Act 2017, section 9(2)(a); CRC and FirstCentral
- CBN licensing requirements for banks and other financial institutions
- NIBSS, NIBSS Direct Debit (NDD); NIBSS GSI explainer
- Provider docs: Mono, Paystack, Flutterwave, Monnify, Fincra, CreditChek
- Nairametrics (Feb 2026) on GSI gaps; Technext (Oct 2025) on open banking
