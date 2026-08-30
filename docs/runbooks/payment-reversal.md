# Payment reversal

Owner: finance operations. Verify the original payment, reversal reason,
provider evidence, and that the request is authorized and idempotent.

Create an immutable reversal event and balanced compensating ledger postings.
Rebuild affected cached balances and schedule allocations, then reconcile the
obligation. Do not delete or overwrite the original payment. Escalate
duplicate or over-limit reversals to finance leadership.
