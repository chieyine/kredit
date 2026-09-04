# Pilot KPI scorecard

The protected application-evidence view at `/admin/analytics` is a live,
date-filtered and optional supplier-filtered view. Compliance-review permission
and AAL2 step-up are required; every read is audited. Its first section is a
print-ready snapshot for accelerator and funding applications. The full pilot
scorecard remains available below it.

## KPI hierarchy

Primary KPIs:

- gross trade credit activated — sum of authoritative obligation principal;
- active suppliers — distinct suppliers activating an obligation;
- time to first accepted credit sale — onboarding record to first immutable
  agreement acceptance.

Drivers cover invitation-to-verification and sent-to-acceptance conversion,
mandate authorisation drop-off, acceptance-to-release and release-to-receipt
time, days to payment, repeat-sale rate, trade-line utilisation and retained
active suppliers.

Guardrails cover on-time payment, failed-collection recovery, recognised loss,
dispute rate, issue-at-receipt rate, activations from buyer silence, voluntary
payment share, manual touches per activated obligation, collection-provider
reliability, support interventions per 100 active suppliers, and open
accessibility defects. Metric cards display their exact definition and source.

### Four metrics that answer questions the others do not

- **Mandate authorisation drop-off** is the buyer proposition in one number.
  The buyer is asked for full verification and a variable-amount debit
  authorisation in exchange for goods they previously received on a handshake.
  Suppliers can be enthusiastic while this number says the product does not
  work; no amount of supplier demand compensates for buyers who will not sign.

- **Voluntary payment share** exists because Kredit earns its collection uplift
  only on money it collects, so the pricing quietly rewards a buyer drifting to
  the collection date rather than paying early. Nobody designed that incentive
  on purpose, and it will shape reminder timing and payment friction unless it
  is watched. Any change that reduces this share is reviewed explicitly and the
  decision recorded, rather than shipped as a conversion improvement.

- **Activations from buyer silence** tracks the residual wrongful-debit
  exposure. Deemed acceptance is constrained by delivered-notice evidence
  (README section 8.3.1), but a rising share still means buyers are not
  answering, which is a risk signal long before it becomes a dispute.

- **Manual touches per activated obligation** is the cost-to-serve signal.
  Kredit earns at most one hundred basis points on activated principal, and the
  operating model carries an approval inbox, reconciliation, dispute review,
  account recovery, privacy requests, corrections and support. On a ₦2,000,000
  sale, gross revenue is ₦20,000: one reconciliation case plus one support call
  consumes it. Minutes per touch is not invented in code — it is sampled during
  the pilot and applied to this ratio.

The snapshot also shows direct clarity feedback from authenticated sellers and
customers: total Yes, Partly and No answers, split by seller and customer. No
free-form comment or contact detail is collected with this signal.

No numerical targets are invented in code. Every metric reports
`baseline_required` until pilot volume, risk appetite, provider SLAs and the
launch owners supply reviewed target evidence. This keeps a descriptive pilot
baseline from being presented as an approved commercial or risk threshold.

Targets wait for evidence; **halt thresholds do not**. A threshold chosen after
the data is in is not a threshold, so the falsifiable stop conditions are
pre-registered separately in `docs/product/pilot-kill-thresholds.md` and are
agreed before the first buyer invitation is sent.

## Reconciliation and freshness

Money and status metrics are queried live from domain tables. Product events
only provide transition evidence. The scorecard compares reconstructable event
counts with buyer invitations, credit requests, acceptances, releases,
receipts, obligations, payments, trade lines and disputes at zero tolerance.
Any difference sets `reconciliation_ok=false` and the interface shows “Review
required.” The response includes generation time, selected window, live-query
mode, latest event time, definitions, filters and every reconciliation row.

Weekly pilot review should record the selected dates, supplier filter, metric
snapshot, reconciliation state, known data-quality limitations and owner
decisions. Analytics must never be used to repair financial state.
