# Kredit go-live runbook

This is the controlled path from a release candidate to the first production
pilot. A green code build cannot replace a legal, provider, security or
operational approval.

## 1. Freeze the release candidate

1. Record the Git commit and immutable API, worker and web image digests.
2. Confirm the release contains the matching migrations and OpenAPI contract.
3. Stop feature changes. Only release-blocking fixes may enter the candidate.
4. Record the release, rollback, incident and compliance owners.

## 2. Supply approved public legal details

Set these values from approved company documents. They are public values, but
must still move through reviewed deployment configuration:

- `LEGAL_DOCUMENTS_ACTIVE=true`
- `LEGAL_ENTITY_NAME` — the exact registered entity operating Kredit.
- `LEGAL_SERVICE_ADDRESS` — the approved address for notices.
- `LEGAL_CONTACT_EMAIL` — a monitored legal/support address.
- `PRIVACY_CONTACT_EMAIL` — a monitored privacy-rights address.
- `LEGAL_EFFECTIVE_DATE` — `YYYY-MM-DD`.
- `TERMS_VERSION=supplier-terms-v1`
- `PRIVACY_VERSION=privacy-v1`

Production web startup fails when a value is missing or invalid. Once active,
the legal pages show the effective date, operator, address and contacts and
become indexable. The versions must match supplier onboarding.

## 3. Configure production

1. Use `APP_ENV=production`, `ORIGIN=https://kredit.com.ng`,
   `PUBLIC_BASE_URL=https://kredit.com.ng` and
   `APP_BASE_URL=https://kredit.com.ng`.
2. Inject backend secrets from the managed secret store. Never put them in a
   Terraform variable file, image, repository file or support note.
3. Use managed PostgreSQL with TLS, private object storage, encrypted backups,
   a real document scanner, OTLP collector and monitored message connectors.
4. Enable only certified identity and collection providers. Keep separately
   gated capabilities disabled until their written approvals exist.
5. Set conservative positive pilot limits and explicit provider-account and
   industry allow-lists.
6. Set `FEATURE_APPROVED_RETENTION_POLICY=true` and
   `FEATURE_PRODUCTION_PILOT=true` only after their signed approval references
   are attached. Production configuration rejects either missing gate.

## 4. Certify staging

1. Deploy the exact candidate to staging and apply every migration once.
2. Run `SECURITY_STRICT=1 bash scripts/security.sh` with every scanner.
3. Complete provider sandbox, webhook replay and out-of-order event tests.
4. Complete backup and isolated restore drills; record checksum, RPO and RTO.
5. Trigger every production alert and confirm the on-call person receives it.
6. Complete the real-device checks in `wave5-accessibility-evidence.md`.
7. Run `APP_ENV=production bash scripts/release-certify.sh` with the protected
   evidence references. It must end with `Release certification PASSED`.

Do not waive a failed gate by changing the script or recording approval in an
informal message.

## 5. Production cutover

1. Create and checksum a final pre-deployment backup.
2. Apply migrations through the migration job before starting the new API.
3. Deploy API and worker, wait for `/healthz` and `/readyz`, then deploy web.
4. Confirm `https://kredit.com.ng`, the two legal pages, sign-in, a private
   invitation, payment link and receipt link over HTTPS.
   Run `BASE_URL=https://kredit.com.ng LEGAL_ENTITY_NAME="Approved company"
   bash scripts/post-deploy-check.sh` for the repeatable public-origin check.
5. Confirm legal pages show the approved entity and effective date, not the
   pre-launch notice.
6. Run one controlled low-value transaction through the certified provider.
   Verify acceptance, goods confirmation, payment, ledger, fee, receipt,
   notification and reconciliation records.
7. Enable the pilot only for approved organisations and limits.
8. Watch readiness, errors, queue age, provider health, reconciliation,
   notification delivery and support continuously during the launch window.

## 6. Stop and rollback conditions

Stop new financial actions immediately for a duplicate or unexplained debit,
ledger imbalance, reconciliation failure, cross-business exposure, identity or
webhook verification failure, sustained readiness failure, or incorrect legal,
fee or mandate wording.

Disable the affected capability first and preserve audit, provider and database
evidence. Roll back only when the previous image is compatible with the applied
schema; otherwise use a reviewed forward fix. Never reverse a financial
migration or delete evidence during an incident.

## 7. First-day closure

Record transaction totals, failures, reconciliations, complaints, incidents,
provider status and pilot-limit usage. The release and compliance owners must
sign the launch record before the pilot expands.
