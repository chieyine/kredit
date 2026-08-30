# Backup restore

Owner: infrastructure. Confirm the incident, approved restore point, backup
checksum, encryption key access, and an isolated target. Restore into an
isolated environment first, run migration and ledger reconciliation checks,
and capture the schema version and evidence.

Promote only after application smoke tests, RLS checks, and finance sign-off.
Do not restore over the primary directly. If validation fails, discard the
isolated target and keep the primary serving; all restore attempts are audited.
