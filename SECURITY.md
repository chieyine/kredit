# Security policy

Kredit handles restricted identity, financial and authentication data. This
policy applies to the API, worker, web application, migrations and deployment
configuration in this repository.

## Reporting

Do not open a public issue for a suspected vulnerability. Send a private report
to the organisation's security contact with the affected component, impact,
reproduction steps and a safe proof of concept. Remove real customer data and
credentials from reports. The security owner acknowledges reports within two
business days, triages severity, and coordinates a fix and disclosure timeline.

## Required handling

- Never commit production secrets, tokens, OTPs, identity numbers, account
  numbers, documents or provider payloads.
- Use the development environment and synthetic fixtures for reproduction.
- Rotate any credential that appears in a log, issue, build artifact or report.
- Production startup must fail closed when required secrets, TLS endpoints,
  approvals or security evidence are missing.
- Run `SECURITY_STRICT=1 bash scripts/security.sh` before a production release;
  this requires every configured scanner instead of silently skipping it.

## Scope and assumptions

The threat model in `docs/threat-model.md`, data inventory in
`docs/compliance/data-inventory.md`, and release checklist are the source of
truth for security controls and residual risk. A successful automated check is
not a substitute for provider certification, a penetration test, legal review,
or an incident-response exercise.
