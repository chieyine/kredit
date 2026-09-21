# ERP comparison contract — audit correction

The ERP comparison endpoint compares **signed journal movements**, not current outstanding balances or a list of invoices. This is a deliberate correction to an ambiguous, debit-only implementation. External integrations must adopt the following contract before operational use.

Each external record has a unique `reference` equal to the Kredit **ledger transaction ID**, a signed integer `amount_kobo`, a `date` within the inclusive requested period, and an optional note. The internal amount is the transaction's debit minus credit on `TRADE_RECEIVABLE_CONTROL`. Originations increase receivables; payments and credit notes reduce them; payment reversals increase them. Supplier receivables only are included; consumer sales use their distinct ledger and are not mixed into this comparison.

Payment, dispute and credit-note transactions are linked through their domain records to authorized supplier obligations. Reference type is part of the join. Reads require current company-wide financial authority and transaction-local tenant identity, in one repeatable-read, read-only transaction. Missing authoritative persistence is an error, never a synthetic empty or principal-derived ledger.

Duplicate references, invalid periods/dates, oversized evidence and money overflow are rejected. Either side is limited to 10,000 movements; a larger period must be split, not silently truncated. Matching compares reference and signed amount within the selected interval. Differing timestamps inside that interval are retained in evidence but are not a separate timing-drift rule.

The v2 report hash binds the period, identities, signed amounts, dates, notes and result, in deterministic reference order. Generation time is excluded to make identical evidence reproducible. A hash is an integrity checksum, not a digital signature or independent audit certification. Comparison never changes balances or posts accounting entries.

Verification includes algorithm tests for duplicates, invalid intervals and overflow, reproducible evidence hashes, and a restricted-role PostgreSQL regression covering originations, payments, credit notes, payment reversal and rejected identities. Successful CI results must be checked on the actual candidate; adding these tests alone is not a pass.
