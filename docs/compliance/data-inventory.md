# Restricted-data inventory

The authoritative field-level register is generated at
`docs/compliance/data-inventory.tsv`. It records a technical classification for each known
column in the `app`, `ledger`, and `jobs` production schemas. The September repair additions were recorded from migration source; comparison with an applied database remains deferred with all tests. CI/database
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


Provider routing update: `app.message_routes` retains encrypted provider credentials for each original notification event so provider changes do not strand in-flight delivery evidence. These ciphertexts require the settings encryption root and must be included in controlled credential-retention and key-rotation procedures. This is an engineering inventory update, not a legal retention approval.

Platform fee billing adds itemised invoice lines, business names and addresses, approved payment instructions, verified bank movement references, amounts and operator evidence. Bank receipts and completed refunds are recorded by the platform owner; invoice issuance is tenant-scoped worker activity. Billing records remain separate from buyer repayments. The TSV includes the new fields; retention and lawful-basis approvals remain governed by the existing register.


## Native verification and fee billing additions

The inventory includes native identity sessions and preserved decision history, separate fee-bank consent, frozen settlement routes, fee allocations, bank debit attempts and completed bank receipts. Identity snapshots are restricted identity data; financial setup and settlement evidence are restricted financial data. BVN, NIN and OTP request values are transient and are not persisted in these records. Customer identity reconciliation uses a keyed fingerprint. Personal exports include the subject's verification results and fee consent, but exclude live authorization URLs, fingerprints and internal review diagnostics. Company-wide fee bills require the separate business permission.

Retention and transfer approvals remain business/legal decisions recorded through the approval workflow. Financial and verification evidence is preserved through forward recovery and cannot be silently erased during a provider change.
