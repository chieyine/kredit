# Production security checklist

Complete this checklist for every production promotion and attach evidence to
the release record.

- [ ] `APP_ENV=production` startup validation passes with managed secrets.
- [ ] Session, OTP, token-hash and provider secrets are at least 32 bytes,
      unique per environment, owned and within their rotation window.
- [ ] Database and object-storage endpoints use private/TLS connections; no
      localhost, development credentials or `sslmode=disable` remain.
- [ ] Security review, DPIA, legal approval, penetration test, backup-restore,
      provider certification, support training and launch approval references
      are attached.
- [ ] Tenant-isolation, authorization, webhook replay and financial
      idempotency tests pass.
- [ ] `SECURITY_STRICT=1 bash scripts/security.sh` passes with all scanners
      installed; dependency and container findings are triaged.
- [ ] Backups are encrypted and the latest restore exercise meets RPO/RTO.
- [ ] Monitoring and alert routing are tested without logging restricted data.
- [ ] Incident contacts, rollback owner and break-glass audit reviewer are
      available for the release window.
