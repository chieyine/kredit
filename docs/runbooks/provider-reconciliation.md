# Provider reconciliation

Owner: payments operations. Export the provider settlement window and compare
provider references, amounts, currencies, and states with the internal ledger.
Investigate every missing, duplicate, partial, or unknown event before marking
the window complete.

Use the reconciliation command and append a settlement event for each
decision. Do not force-match amounts or delete provider events. Escalate
unresolved differences to the provider and finance owner; corrections require
an approved, balanced ledger adjustment and an audit trail.
