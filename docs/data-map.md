# Kredit data map

| Data class | Examples | Primary store | Access notes |
| --- | --- | --- | --- |
| identity | phone, email, verification state | PostgreSQL | masked outside authorised flows |
| business/KYB | legal name, registration, address, industry | PostgreSQL | tenant and compliance scoped |
| authority/consent | representative, acceptance, consent version | PostgreSQL + immutable evidence | append-only history |
| financial | principal, kobo amounts, payments, ledger | PostgreSQL | ledger-only financial writes |
| mandate/provider | provider reference, status, capability | PostgreSQL | no raw credentials or PINs |
| documents | invoices, evidence, generated summaries | S3-compatible storage | signed access, retention class, scan state |
| operations | audit, support cases, jobs, webhooks | PostgreSQL | case-bound sensitive access |
| buyer identity | person, business, representative, authority state | PostgreSQL | separate facts; tenant/relationship scoped |
| verification | provider reference, level, state, safe result | PostgreSQL | no raw BVN, biometrics, credentials, or OTP |
| invitation | hashed public token, target hash, expiry, status | PostgreSQL | raw token is transient and never logged |
| consent | consent type/version/timestamp/evidence hash | PostgreSQL | append-only history |

The authoritative per-field register is `docs/compliance/data-inventory.tsv`;
its catalog check currently covers all 924 persisted production fields.
Retention and lawful-basis values remain marked pending until legal/compliance
approval, which keeps the production release gate fail-closed.
