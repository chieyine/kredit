# Restricted-data inventory

The authoritative field-level register is generated at
`docs/compliance/data-inventory.tsv`. It contains one reviewed row for every
column in the `app`, `ledger`, and `river` production schemas. CI/database
certification runs `scripts/data-inventory-check.sh`; schema drift without an
inventory row fails the check.

Lawful-basis and retention entries remain explicitly marked
`pending_legal_approval` until the Data Protection Lead approves the
environment retention register. Production readiness must remain fail-closed
while that approval is pending.

This inventory is reviewed at least quarterly and whenever a new provider,
field, export or retention rule is introduced.

| Data class | Examples | Purpose | Access boundary | Control |
| --- | --- | --- | --- | --- |
| Restricted identity | identity identifiers, authority evidence, documents | verify identity and signing authority | case-bound operations and approved support | field encryption/object isolation, audit, no logs |
| Restricted authentication | session tokens, OTP material, MFA secrets | authenticate a user | auth service only | hashed/opaque storage, HttpOnly cookies, redacted logs |
| Restricted financial | account tokens, mandates, ledger postings, settlement references | execute and reconcile repayment | tenant-scoped finance and provider workers | exact kobo arithmetic, RLS, idempotency, audit |
| Confidential commercial | agreements, obligations, disputes, exposure | operate trade credit | tenant membership and least privilege | access checks, encrypted storage, audit |
| Internal operations | job status, health and non-sensitive metrics | operate the platform | operators and on-call | structured logs, retention and access review |
| Product analytics | hashed subject/organisation IDs, versioned event names, bounded state metadata | measure the essential pilot funnel and reliability | AAL2 compliance operations | first-party storage, deterministic deduplication, field allowlist, live source reconciliation |

Retention and deletion periods are approved by legal and recorded in the
environment-specific retention register. Test fixtures must be synthetic and
must never be copied from production.
