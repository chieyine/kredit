# Financial reporting and reversal audit

## Verified defects and corrections

Enterprise reporting previously queried `business_branches.region`, a nonexistent column, and discarded query errors. It then used raw database queries without request tenant context, so missing branch facts could be presented as headquarters exposure. The revised path uses the actual `territory` field, checks every read/scan error, verifies current financial authority and reads money, branch metadata and assignments in one repeatable-read transaction. Inactive branches with historical exposure remain visible under the existing branch policies.

Ageing is calculated from each remaining instalment, not by assigning an entire obligation to one incompatible bucket label. Current/future and up-to-30-day amounts share `0-30`; the other buckets are 31–60, 61–90 and over 90 days since the collection deadline. Classified instalments must exactly reconcile to the obligation outstanding amount. All money accumulation and basis-point ratios reject invalid amounts and avoid integer overflow.

The published on-time field retains its originated-principal denominator for compatibility. It is a conservative collected-principal share from obligations without late payment/current overdue flags, not an independent credit score or a mature-payment timeliness rate. Late fully paid obligations no longer count as on-time. An empty portfolio is `NO_DATA`, not `EXCELLENT`. Existing nonempty rating thresholds are retained pending product review of metric definitions.

Report hashing now covers the full published summary, health fields and sorted branch exposure, excluding generation time. CSV output treats untrusted branch text as text. Checksums are not digital signatures; recipients should import text columns as text.

A new restricted-role ERP lifecycle regression exposed a payment-reversal failure: `ReverseSplitTx` attempted DELETE on immutable split-fee allocation evidence. The correction keeps allocation records, requires and locks a reversed original payment, checks amount accumulation, decrements fee collections and posts the undo journal atomically. Replays validate the existing undo journal and cannot subtract twice. Existing bank/reward readers already exclude reversed payments from active totals. No DELETE grant or relaxation of evidence guards was introduced.

## Verification

Local isolated algorithm tests pass for signed ERP movements, duplicate evidence, money overflow, ageing boundaries, late-payment handling, report hash stability and CSV handling. They exercise the actual pure calculation source with extracted production data types/money helper, not a local PostgreSQL or full application run.

New PostgreSQL regressions exercise current branch territory, retained inactive-branch exposure, rejected tenant overrides, payment reversal without split fees, preserved split-allocation evidence and repeated reversal. Check actual CI results for the current candidate before claiming these pass. The full isolated database workflow also supplies legacy restricted-login aliases and a separate named native-provider fixture database instead of relaxing production role or test-database guards.

All changes remain in draft PR #16. No production financial data, provider account, main branch, merge or deployment is changed by these tests.
