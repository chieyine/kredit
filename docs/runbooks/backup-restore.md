# Backup restore

Owner: infrastructure, with finance reviewing financial evidence. No restore was executed for the audit repairs or subsequent improvements.

## Prepare evidence and isolation

Record the incident/drill reference, named operator, source snapshot time, archive checksum, backup storage location, protected encryption-key location, target identity and start time. Do not record connection URLs or credentials in the report. Establish actual recovery-point and recovery-time objectives with the release owner; this document does not invent achieved RPO/RTO values.

Use a dedicated isolated PostgreSQL target with no production integrations, workers, email/SMS delivery or collection credentials. Provision the required named roles in the isolated cluster first. `scripts/restore-drill.sh` refuses a populated target and never uses `--clean`; the subsequent role restoration may affect cluster roles, which is another reason to use an isolated cluster.

Prepare a quiesced source fingerprint using `scripts/recovery_fingerprint.py capture PATH --quiesced` and the corresponding backup. Quiescence must be established operationally; the flag is an assertion, not a mechanism that pauses traffic. A fingerprint from a different source instant is not valid comparison evidence.

## Restore path

The existing drill accepts an absolute archive path plus its `.sha256` sidecar, `RESTORE_DATABASE_URL` and `RESTORE_EXPECTED_FINGERPRINT`. Optional `RESTORE_ROLE_ADMIN_URL` supplies isolated role-management credentials. Supply secrets through the approved environment mechanism.

Both `scripts/backup.sh` custom `.dump` archives and `cmd/backup-r2` `.dump.gz` archives are accepted. For compressed archives, the script verifies the original compressed bytes first and decompresses into a private temporary directory before invoking `pg_restore`. Decompression/restore failure aborts the operation. The temporary files are removed on exit.

The drill compares logical/security fingerprints and checks runtime grants. Capture its actual exit status and protected output. It does not certify production point-in-time recovery, offsite decryption, provider reconciliation or application behavior. Offsite availability must be demonstrated by retrieving the chosen object through the approved storage process, not by using an unrelated local copy.

## Release evidence

After a matching restore, capture restored schema version and counts, per-transaction ledger reconciliation, financial projection differences, source/target timestamps, elapsed time and operator review. Run application and tenant-isolation exercises only when authorized. Record failed stages honestly; a checksum match is not a successful restore.

Promote only after the applicable application, RLS and finance reviews. Never restore directly over the serving primary. If the drill fails, preserve diagnostic evidence, keep the target isolated and correct the recovery procedure. Do not connect the restored workers to live providers to see whether they work.
