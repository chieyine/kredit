# Weekly pilot scorecard review

Owner: Growth Product Lead  
Control reviewers: Data Platform Lead, Data Protection Lead, Supplier Operations Lead  
Escalation: Financial Systems Lead for money reconciliation; Security and Accessibility Reviewer for their guardrails

## Prerequisites

- Use an AAL2 session with compliance-review permission.
- Confirm the selected date range and optional supplier UUID are the intended
  pilot cohort. Never paste customer contact, bank or invoice data into notes.
- Confirm the scorecard says `live query` and record its generation timestamp.

## Procedure

1. Open `/admin/analytics`, select the weekly UTC-inclusive dates, and apply the
   approved supplier filter if the review is cohort-specific.
2. Stop the review if reconciliation says **Review required**. Record the event,
   source count, event count and window; do not edit financial records or event
   rows. Follow provider reconciliation for provider events and escalate source
   drift to the Financial Systems Lead.
3. Record the three primary KPIs: activated trade-credit volume, active
   suppliers and time to first accepted sale.
4. Review conversion, fulfilment time, days to payment, repeat sale,
   trade-line utilisation and supplier retention drivers.
5. Review every guardrail: on-time payment, failed-collection recovery,
   recognised loss, disputes, receipt issues, provider reliability, support
   interventions per 100 active suppliers, and open accessibility defects.
6. Compare only against an approved target reference. While a metric says
   `baseline_required`, record the observation but do not label it pass/fail.
7. Attach the filter/window, generated timestamp, reconciliation result,
   metric snapshot, target reference, decisions, owners and due dates to the
   protected weekly pilot record.

## Mitigation and rollback

- Money or source drift: suspend expansion and affected financial capability;
  preserve immutable evidence and reconcile from domain/ledger records.
- Provider reliability or unknown state: disable the affected live provider
  capability and follow `provider-reconciliation.md`.
- Loss, dispute or support burden outside an approved guardrail: hold cohort
  expansion and invoke the production-pilot kill switch review.
- Critical accessibility defect: block the affected journey from expansion and
  attach the defect to the Wave 5 manual evidence register.

Analytics are never used to repair balances, agreements, exposure or provider
state. Corrections happen through the relevant versioned domain command.

## Required evidence

- review date, owners and approved target reference;
- selected window and supplier filter;
- generated and latest-event timestamps;
- reconciliation rows and status;
- KPI/driver/guardrail snapshot;
- mitigation, decision, owner and follow-up date.
