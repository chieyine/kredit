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
acceptance-to-release and release-to-receipt time, days to payment, repeat-sale
rate, trade-line utilisation and retained active suppliers.

Guardrails cover on-time payment, failed-collection recovery, recognised loss,
dispute rate, issue-at-receipt rate, collection-provider reliability, support
interventions per 100 active suppliers, and open accessibility defects. Metric
cards display their exact definition and source.

The snapshot also shows direct clarity feedback from authenticated sellers and
customers: total Yes, Partly and No answers, split by seller and customer. No
free-form comment or contact detail is collected with this signal.

No numerical targets are invented in code. Every metric reports
`baseline_required` until pilot volume, risk appetite, provider SLAs and the
launch owners supply reviewed target evidence. This keeps a descriptive pilot
baseline from being presented as an approved commercial or risk threshold.

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
