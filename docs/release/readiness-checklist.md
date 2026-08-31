# Production readiness checklist

The machine-checkable certification report is maintained in
`docs/release/certification-report.md`; run `bash scripts/release-certify.sh`
before treating this checklist as complete.

Use `docs/release/go-live-runbook.md` for staging, cutover, rollback and the
first-day review.

- [ ] All v1 definition-of-done items pass.
- [ ] No unresolved critical/high security defect.
- [ ] Legal, privacy, provider, fee/tax, and mandate approvals recorded.
- [ ] Approved operator name, service address, legal/privacy contacts,
      effective date and document versions are deployed; live legal pages no
      longer show the pre-launch notice.
- [ ] `kredit.com.ng` DNS, TLS renewal, canonical origin and HTTPS redirects are
      verified from outside the production network.
- [ ] `scripts/post-deploy-check.sh` passes against the public production
      origin and approved legal entity.
- [ ] Google Search Console domain ownership is verified, the live
      `https://kredit.com.ng/sitemap.xml` is submitted, and its first fetch
      succeeds without sitemap or robots errors.
- [ ] The live home page and one guide pass Google Rich Results and URL
      Inspection checks; canonical URL, rendered HTML and index permission
      match what the application publishes.
- [ ] Home-page and guide links have been previewed in WhatsApp and at least
      one other social service to confirm the title, description and 1200×630
      image are readable.
- [ ] Production analytics receives page, sign-in, demo, sale-created and
      payment-completed events without storing private sale or payment data in
      marketing tools.
- [ ] Threat model, data map, retention rules, and runbooks reviewed.
- [ ] Backup restore and reconciliation drills completed.
- [ ] WCAG 2.2 AA critical flows checked. Automated serious/critical gate is
      green; attach the manual evidence listed in
      `docs/release/wave5-accessibility-evidence.md` before checking this item.
- [ ] Provider sandbox/certification tests pass.
- [ ] Pilot limits and feature flags are configured outside code.
- [ ] Support/compliance training and escalation paths are ready.
- [ ] Rollback and capability-disable procedure tested.
- [ ] `docs/security/production-security-checklist.md` completed with evidence.
- [ ] `SECURITY_STRICT=1 bash scripts/security.sh` passes with every required
      scanner installed.
- [ ] Observability scrape and alert routing are tested against
      `docs/operations/observability.md` objectives.
- [ ] A private, checksummed backup is created and an isolated restore drill
      verifies the latest Goose schema version.
- [ ] `db.CheckPersistenceContract` passes for every state-bearing table and
      all domain aggregates are backed by PostgreSQL repositories; no staging
      or production process-local store remains.
- [ ] `bash scripts/release-certify.sh` passes in an environment with the
      production-like database, frontend/browser dependencies, and protected
      approval references configured.

## Evidence references and controlled limits

The following references and limits must be supplied through protected environment configuration, never hard-coded:

- `SECURITY_REVIEW_REFERENCE`
- `DPIA_REFERENCE`
- `LEGAL_APPROVAL_REFERENCE`
- `PEN_TEST_REFERENCE`
- `BACKUP_RESTORE_REFERENCE`
- `PROVIDER_CERTIFICATION_REFERENCE`
- `SUPPORT_TRAINING_REFERENCE`
- `LAUNCH_APPROVAL_REFERENCE`
- `PILOT_MAX_SUPPLIER_ORGANIZATIONS`
- `PILOT_MAX_BUYER_BUSINESSES`
- `PILOT_MAX_PRINCIPAL_KOBO`
- `PILOT_MAX_ACTIVE_EXPOSURE_KOBO`
- `PILOT_MAX_DRAWDOWNS_PER_LINE_DAY`
- `PILOT_MAX_COLLECTION_RETRIES`
- `PILOT_ENHANCED_REVIEW_KOBO`
- `PILOT_ALLOWED_PROVIDER_ACCOUNTS`
- `PILOT_ALLOWED_INDUSTRIES`

`GET /api/v1/organizations/{organizationID}/readiness` exposes gate names and missing evidence without exposing the underlying approval references. Production configuration is rejected unless all evidence and positive limits are present.

## Exercises

- `scripts/backup.sh` creates a PostgreSQL custom-format backup; the configured backup directory/object store must be encrypted and access-restricted.
- `scripts/restore-drill.sh /absolute/path/to/backup.dump` restores only into the explicitly supplied isolated database and verifies connectivity.
- `scripts/load-smoke.sh` runs the k6 health/API smoke profile when k6 is installed.
