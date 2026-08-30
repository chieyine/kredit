# Product and external-dependency decision register

Last reviewed: 29 August 2026

These decisions must be supported by written evidence. A due date is a review
target, not permission to enable a capability. Missing or inconclusive evidence
leaves the corresponding feature gate disabled.

| ID | Decision required | Accountable owner | Review due | Evidence | Unlocking gate | Status |
| --- | --- | --- | --- | --- | --- | --- |
| EXT-001 | Which collection provider approves supplier-funded B2B trade credit? | Partnerships Lead | 11 Sep 2026 | Not supplied | `FEATURE_REAL_COLLECTIONS` | Open; disabled |
| EXT-002 | Which approved mandate structures support one-time, variable, recurring, and instalment collection? | Payments Lead | 18 Sep 2026 | Not supplied | `FEATURE_REAL_COLLECTIONS` | Open; disabled |
| EXT-003 | Is multi-account BVN-linked collection approved, and with what consent and revocation behavior? | Compliance Lead | 18 Sep 2026 | Not supplied | `FEATURE_MULTI_ACCOUNT_COLLECTIONS` | Open; disabled |
| EXT-004 | What cancellation, reversal, dispute, timeout, and reconciliation guarantees does the selected provider contractually supply? | Payments Lead | 18 Sep 2026 | Not supplied | `FEATURE_REAL_COLLECTIONS` | Open; disabled |
| EXT-005 | May collected funds settle directly to the supplier account, and what evidence proves final settlement? | Finance Operations Lead | 18 Sep 2026 | Not supplied | `FEATURE_DIRECT_SUPPLIER_SETTLEMENT` | Open; disabled |
| EXT-006 | How are Kredit fees billed, invoiced, taxed, reversed, and disclosed? | Finance Lead | 25 Sep 2026 | Not supplied | `FEATURE_LIVE_SUPPLIER_BILLING` | Open; disabled |
| EXT-007 | What person, business, authority, enhanced-review, and expiry requirements apply at each pilot threshold? | Compliance Lead | 11 Sep 2026 | Not supplied | `FEATURE_REAL_IDENTITY` | Open; disabled |
| EXT-008 | What retention periods, lawful bases, deletion exceptions, and trade-history wording are approved? | Data Protection Lead | 25 Sep 2026 | Not supplied | `FEATURE_APPROVED_RETENTION_POLICY` | Open; disabled |
| EXT-009 | Which industries, provider accounts, supplier counts, buyer counts, principal limits, exposure limits, and retry limits are approved for the pilot? | Risk Lead | 2 Oct 2026 | Not supplied | `FEATURE_PRODUCTION_PILOT` plus `PILOT_*` limits | Open; disabled |

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
