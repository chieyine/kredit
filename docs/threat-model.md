# Kredit threat model

## Scope

The initial scope covers the first-party web application, Go API and worker,
PostgreSQL, object storage, provider webhooks, WhatsApp/email/SMS channels, and
operator access.

## Initial high-value assets

- accepted agreements and evidence;
- identity, authority, mandate, and account references;
- ledger postings, payment allocations, and collection attempts;
- audit history and support/compliance case data;
- provider credentials and webhook authenticity keys.

## Initial threats and controls

| Threat | Required control |
| --- | --- |
| forged invitation or replayed token | random single-use tokens, hashed at rest, expiry, audit |
| token leakage through logs or analytics | redact invitation token path from access logs; never log raw URLs/tokens |
| cross-tenant access | tenant context, permission checks, PostgreSQL RLS, isolation tests |
| wrongful debit | exact acceptance, active mandate, authoritative outstanding calculation, reservation, idempotency |
| duplicate/out-of-order webhook | provider event identity, state normalization, dedupe table, reconciliation |
| sensitive-data leakage | masking, structured logging review, secret scanning, restricted support access |
| malicious upload | content/type/size validation, object isolation, malware scanning state |
| operator abuse | least privilege, case-bound access, step-up auth, break-glass audit and review |

## Security boundaries

- The browser is untrusted and may replay, alter or omit every request field.
- The API is the authorization boundary; tenant membership is checked before
  organization-scoped reads and writes.
- Workers are separate trust zones with provider egress and no interactive
  browser session.
- PostgreSQL RLS and append-only audit tables are defense-in-depth controls,
  not substitutes for application authorization.
- Runtime database roles are defined in `infra/postgres/roles.sql`; the
  migration role is separate and audit rows are database-enforced append-only.

## Review triggers

Review this model before enabling a real provider, adding a restricted field,
changing an export, changing tenant roles, or after a security incident. Record
the reviewer and date in the release evidence.
