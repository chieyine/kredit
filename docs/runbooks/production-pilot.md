# Production pilot runbook

1. Confirm the readiness endpoint reports every gate as passed.
2. Confirm provider certification and account-type scope match the configured pilot values.
3. Run a fresh private, checksummed backup, record the backup identifier, and complete a restore drill against an isolated database; verify the restored Goose schema version.
4. Configure a private scrape of `/api/v1/ops/metrics/prometheus` with step-up credentials, then run the load smoke profile against staging and review p95 latency, error rate, provider call volume, and database connections.
5. Verify alert routing for provider webhook failures, reconciliation mismatches, ledger imbalance, authentication anomalies, and backup failures.
6. Train support on disputes, correction requests, provider timeouts, mandate cancellation, and capability-disable procedures.
7. Enable only the approved feature flags and conservative pilot limits. Never change limits in source code or bypass the readiness gate.
8. During the pilot, follow `pilot-scorecard.md` weekly and review active
   exposure, collection success/partial/failure, recognised loss, disputes,
   reconciliation, provider reliability, support burden, retention and
   accessibility defects daily where alerts require it.
9. If a gate fails, disable the capability, preserve obligations and ledger history, notify affected parties, and follow the incident runbook.

The durable PostgreSQL/River runtime is implemented and production remains
fail-closed. Real provider certification, legal/retention approval, manual
accessibility evidence, target-environment exercises and launch signatures are
external gates and must not be bypassed.
