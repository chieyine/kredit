# Phase 5 — Production assurance

Status: engineering controls implemented on a review branch; **production approval remains blocked**. Read exact-commit CI results before describing any check as passed. This phase implements the audit's K27–K35 and K62–K68 assurance work, not product implementation Milestone 5.

Dependency: the branch starts from Phase 4 candidate `5b3e7e3e7f8a11fcba04841d27a24a7fa580c3e8`. Phase 3 and Phase 4 PRs remain separate prerequisites. Do not merge a dependent PR into its parent to bypass the parent's repository-wide CI. No deployment, real debit, production load, real message, legal approval or penetration test is authorized by this branch.

## Delivered controls and evidence limits

| Audit scope | Implemented control | Evidence still required before pilot |
| --- | --- | --- |
| K27–K28 security and API authorization | Candidate-bound reviewed-evidence gate; scope below; existing auth/RLS/financial regression suites retained | Independent penetration test, route-level object/role review and critical/high finding closure |
| K29–K31 terms, mandates and privacy | Separate legal-terms, mandate-consent and DPIA approval gates; no fabricated legal text or approval values | Named Nigerian counsel/privacy reviewer, approved exact document versions and data/processor map |
| K32 recovery | Real logical backup/restore, requested-file checksum verification, source/restored data and security fingerprints, restricted-role check | Intended platform's encrypted backup/offsite restore, key recovery, PITR and measured RPO/RTO |
| K33 incident response | Action/approval/stop/reconciliation procedures below and incident-drill gate | Named operators, provider contacts, timed exercise, evidence of notification and escalation |
| K34–K35 financial monitoring | Migration 086 aggregate-only metrics, missing-signal alerts, tested alert firing/resolution, visible unknown outcomes | Production scraper, routing, on-call acknowledgements and alert-delivery drill |
| K62–K64 load and SLOs | Bounded real authenticated API/database read baseline; explicit provisional objectives | Production-shaped dataset, approved workload, concurrency/write paths, deployment sizing and accepted report |
| K65–K66 jobs and recurring recovery | Due-job/dead-letter gauges, queue ownership procedure, repeatable restore tooling | Operational ownership, retry budgets, scheduled backup and repeat drills |
| K67–K68 accessibility | Mandatory manual-assessment sign-off; task matrix below; existing automated UI checks retained | VoiceOver/TalkBack, keyboard, 400% zoom, high contrast and target-device evidence |

A passing Phase 5 workflow proves only the exercised engineering controls. It cannot supply any external evidence in the right-hand column.

## Release evidence gate

`scripts/release-certify.sh` now starts with `scripts/phase5-evidence-gate.sh`. It refuses absent approvals, a dirty checkout, a candidate mismatch, stale/future reviews, changed evidence bytes, unsafe paths, incomplete gate sets and unresolved critical/high findings. Existing runtime/configuration/local-test checks remain in place.

Generate the pending manifest **outside the public repository**:

```sh
python3 scripts/verify_release_evidence.py --write-template > /protected/release/manifest.json
```

The required gates are security review, API authorization, independent penetration test, legal terms, mandate consent, DPIA, actual provider certification, target-environment restore, target-environment load, monitoring drill, incident drill, manual accessibility, support training and final launch approval. Every record needs the required evidence type, an independent reviewer, check/review/expiry timestamps and a relative evidence path plus SHA-256. The current conservative freshness ceiling is 90 days, a project policy proposal rather than a legal rule. Material code/configuration changes invalidate the manifest even within that window.

Bind the review to all of:

- the exact 40-character commit in `RELEASE_CANDIDATE_SHA`;
- the intended environment in `RELEASE_TARGET_ENVIRONMENT`;
- the SHA-256 of a reviewed non-secret deployment configuration record in `RELEASE_ENVIRONMENT_SHA256` (image digest, database version, enabled capabilities, topology and policy values; never secret values);
- protected paths in `RELEASE_EVIDENCE_MANIFEST` and `RELEASE_EVIDENCE_ROOT`.

An environment variable containing a reference alone is no longer sufficient. The validator verifies **structure, binding, freshness and file integrity**, not report authenticity, the competence of a reviewer, regulatory status or a cryptographic signature. A trusted human must verify provenance and approve deployment through a protected environment. The repository's outstanding Phase 1 administration/branch-protection gate is not solved by a shell script.

Do not run release certification against production data: its legacy local-check commands are intended for the isolated release-verification workspace. Do not turn on live feature flags simply to satisfy validation. Actual provider approval remains tracked by issue #5.

## Security assessment scope

Use a written authorization, named tester, isolated staging target, test accounts, approved rate limits, stop conditions and evidence handling. Align the review with [OWASP ASVS 5.0](https://owasp.org/www-project-application-security-verification-standard/); this document does not claim ASVS compliance.

The route-level review must trace **authentication → role permission → object ownership → tenant DB context → material side effect**, including negative cases. Cover supplier owners/finance/sales, buyers, platform operators, unauthenticated users and workers. Include login/OTP enumeration and replay, session recovery/MFA cooling-off, CSRF, cross-tenant and same-tenant wrong-role access, supplier/buyer financial reads, money writes/reversals, mandates/consents, admin corrections, uploads/storage URLs, webhook replay/identity, idempotency collisions, SSRF and restricted data in errors/logs. For each route record method/path, required actor, permission, ownership lookup, database enforcement, test reference and outcome. An HTTP 200 or a code scan alone is not approval.

The external report must identify the tested commit and environment, every finding, reproduction evidence, remediation and retest result. Critical/high findings cannot be waived by entering `accepted_risk`. Secrets and exploitation evidence stay in restricted storage, not public issues or build artifacts.

## Legal and privacy review packet

The legal reviewer must approve Kredit's actual supplier-funded trade-credit model, not a hypothetical lending model. Provide exact Terms/Privacy/consent versions and effective dates; supplier/buyer agreement evidence; mandate ceiling, cancellation, debit notice and dispute workflows; fee/tax/settlement treatment; provider contracts and allocations of responsibility. Never equate commercial agreement acceptance with bank-debit authorization.

The DPIA packet must include the existing data inventory/map, purposes and lawful-basis analysis, persons/businesses/authority distinctions, identity and bank-reference processing, data minimization, retention/deletion exceptions for financial history, access/correction requests, processors/subprocessors, transfers, security controls and incident obligations. Unresolved legal interpretation goes to counsel; this code change neither asserts compliance nor activates draft legal pages.

## Monitoring and migration rollout

Migration 086 introduced the metrics reader. For the current application, apply every current migration (through 096 at this audit checkpoint), then apply `infra/postgres/roles.sql`. The helper returns fourteen fixed **aggregate gauges**, not customer records. It is SECURITY DEFINER with fixed search path, no caller-supplied SQL and no PUBLIC execute grant. `row_security=off` makes insufficient owner visibility fail visibly rather than silently filtering the counts. It does not grant table-wide runtime access or weaken financial RLS.

The isolated integration test checks that the unscoped app role sees zero obligation rows while monitoring counts the seeded active portfolio correctly. It also verifies missing database handling, complete gauge output and the helper's PUBLIC restriction. Keep the existing authenticated/secret-protected Prometheus endpoint; do not expose metrics publicly.

The rule file now uses only supported durable financial gauges. It adds stale unresolved collections, negative balances, due queues and discarded jobs, and removes speculative failure-rate/revocation counters that were not emitted by the durable exporter. The `promtool` tests cover healthy/missing signals, firing, hold periods and recovery. Each production Prometheus deployment must attach environment/cluster labels and isolate rules per environment; a gauge from staging must never conceal missing production instrumentation. Add per-instance `up` checks when there are multiple replicas. Current in-process latency is a bounded sample, not a fleet-wide latency histogram or a contractual SLO measurement.

Configure the actual scrape token and route critical alerts to a named responder. Test a controlled missing-scrape/unknown-result scenario, alert delivery, acknowledgement, escalation and resolution. A valid rule file does not prove alerts reach a human.

## Proposed pilot objectives — approval required

These are starting acceptance targets, not achieved results or provider promises: API availability 99.9% over a rolling 30-day window; authenticated API read p95 below 1 second and p99 below 2 seconds under the **approved** workload; known financial discrepancies zero; unknown outcomes reconciled or assigned immediately; webhook age above 5 minutes and unresolved collection age above 15 minutes investigated. Adapt provider timing thresholds to documented bank/provider behavior before enabling production paging. Candidate recovery objectives are RPO at most 15 minutes and RTO at most 60 minutes; only intended-platform drills/PITR can establish them.

Define eligible requests, client/server/provider latency boundaries, planned maintenance, error-budget ownership and what happens when a target is breached. Do not use the 64-read CI baseline as proof of any of these production objectives.

## Load baseline and target-environment test

`phase5_load.py` authenticates seeded supplier/buyer sessions using the actual development OTP endpoints. It runs 64 financial reads with four threads, checks HTTP 200 **and** data shape, reports per-route p95/p99/failures, and verifies that unauthenticated financial/metrics reads are denied. The small request budget deliberately stays below the normal limiter; no spoofed forwarding headers, disabled rate controls, browser agents or real provider calls are used.

```sh
APP_ENV=development KREDIT_PHASE5_LOCAL=1 \
  python3 scripts/phase5_load.py --output /protected/test/ci-baseline.json
```

The harness refuses non-loopback targets and requires an explicit development acknowledgement. It is a CI regression baseline only. A separate approved staging campaign must use production-shaped records, representative active suppliers/buyers and row distributions, cold/warm caches, report/dashboard queries, OTP limits, scheduled jobs and webhook backlogs. Test both throughput and concurrent financial writes under locks; require intact balances/idempotency after the run. Record infrastructure, versions, dataset scale, rates, test duration, p95/p99, saturation, errors, database contention and post-run reconciliation. Use the existing Phase 3/4 financial proofs as complementary correctness evidence, not throughput measurements. No automated real debit is authorized by this workload.

## Recovery procedure and repeat cadence

The CI workflow uses PostgreSQL 18 for server **and client tools**, an empty isolated target, a quiesced synthetic source and named runtime roles. A fresh archive retains ACLs. The sidecar's filename is never trusted to choose what gets hashed: the actual requested dump bytes must match the digest.

Before a protected drill: obtain authorization and an isolated destination; provision the same named roles; pause application workers, schedulers and DDL; record PostgreSQL version/extensions and encryption/key access; ensure no application can attach to the restore target. Capture a source fingerprint, back up, restore and compare:

```sh
python3 scripts/recovery_fingerprint.py capture /protected/test/source.json --quiesced
bash scripts/backup.sh
# Use the exact absolute archive path printed by backup.sh.
RESTORE_EXPECTED_FINGERPRINT=/protected/test/source.json \
  RESTORE_DATABASE_URL="$ISOLATED_RESTORE_URL" \
  bash scripts/restore-drill.sh /protected/backups/exact-backup.dump
```

The comparison includes table row counts and row hashes, schema migration records, RLS policies/flags, PUBLIC access to SECURITY DEFINER functions and balanced ledger postings. Restricted app-role access is rechecked in CI on the restored database. This is not a complete ACL/ownership or physical database verifier; separately review role memberships, function owners, grants and deployment credentials. Preserve protected evidence, not raw rows or a public archive. A target with existing tables is rejected, including aliases of the source. The script never runs `--clean` or drops a target.

A source fingerprint captured separately from a backup is valid only while writers/DDL remain paused. It is not a live point-in-time consistency protocol. Archive checksums provide integrity, not encryption. Before production, repeat using the actual encrypted offsite/cloud backup, recover keys in an isolated account, prove PITR and measure RPO/RTO. Proposed cadence: monthly, after material schema/storage/key changes, and before each pilot expansion. Have an operations owner approve the cadence and maintain an evidence-expiry reminder.

## Incident response and job handling

For suspected incorrect/duplicate debit: record time and correlation in protected incident storage; assign incident lead; disable **new initiation** for the affected capability through the approved controls; preserve provider reconciliation and evidence; do not resubmit an unknown outcome or edit a balance manually. Verify the original reference with the provider, compare reservations/payments/ledger/settlement facts and open reconciliation cases. Refund/reversal/customer notices require the authorized finance/support process and provider-confirmed evidence. Re-enable only after root cause, reconciliation and independent approval.

For provider outage/backlog: keep reservations for uncertain submissions; retain bounded retries and the original identities; assign an operator to every stale or discarded financial job; distinguish unknown from confirmed failure. Avoid bulk retry, job deletion or a new idempotency key as a workaround. Review backlog age, provider availability, queue workers and database locks. Resolve cases only after authoritative disagreement is fixed. Test restart and backlog recovery in the approved staging exercise.

For suspected data exposure: stop affected access paths, preserve logs without copying secrets into tickets, involve security/privacy leads, rotate compromised credentials under a reviewed plan and have counsel determine notification duties. For each exercise record detection, acknowledgement, containment, restoration, customer/support handling and sign-off times. A written playbook alone is not an incident drill.

## Manual accessibility evidence

The reviewer must perform login/recovery, credit review/acceptance, mandate authorization handoff, exact amount confirmation, pending/failed payment recovery, transaction history, disputes and operational confirmation dialogs. Test keyboard-only operation, visible focus, screen-reader labels/live status changes, dialog focus trapping/restoration, 400% zoom/reflow, high-contrast/forced-colors, reduced motion and touch target operation on target devices. Use VoiceOver and TalkBack with actual device/browser/OS versions recorded. Confirm amount, party, consequence and next step remain understandable without vision or a mouse.

Retain per-task findings, fixes and retests. Existing Axe/Playwright checks remain useful but cannot constitute this manual assessment. No contrast/accessibility test is suppressed by Phase 5.

## External closure

All external gate records start pending. Independent reviewers, actual platform/provider access and human participation are required. Do not create fake passing reports, run unapproved scans against production, commit credentials, or present a CI artifact as regulatory certification. Phase 5 can be reviewed as engineering work while external approval remains blocked; it cannot be described as a production release sign-off until the protected evidence and prerequisite PR/CI gates are satisfied.

References: [Prometheus rule testing](https://prometheus.io/docs/prometheus/latest/configuration/unit_testing_rules/), [PostgreSQL row security](https://www.postgresql.org/docs/current/ddl-rowsecurity.html), [OWASP ASVS](https://owasp.org/www-project-application-security-verification-standard/). These guide the controls; they are not evidence that Kredit has passed independent assessment.
