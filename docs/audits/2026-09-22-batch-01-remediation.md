# Batch 1 source remediation — 22 September 2026

Base commit: `f8e5571d45a312698bd198bbaec50b086a8bb01e`.
Scope: K01–K10 from the first file-by-file audit. This is a source-change record, not a deployment certificate or a claim that the whole repository has been reviewed.

## Changes

| Finding | Source correction |
| --- | --- |
| K01 | API and worker use bootstrap-only validation before reading saved connections. The existing complete validation still runs on the effective configuration before runtime construction. Environment-only `config.Load` keeps its previous contract. |
| K02 | Explicit pool-size settings fail on invalid values. Timeout precedence is documented; URL overrides are validated, positive durations are normalized to milliseconds, startup `options` are refused, and each physical connection checks its actual timeout values. |
| K03 | Maintenance, collection discovery, financial reconciliation, outbox dispatch, notification discovery and document discovery have independent non-overlapping scheduling loops. Critical scheduling progress participates in worker readiness. |
| K04 | API and worker resource ownership moved into returning `run` functions. Fatal paths now unwind deferred cleanup before the outer `main` exits. Worker scheduling stops before River drains; cancellation provides the drain fallback. |
| K05 | PostgreSQL, MinIO, Mailpit and the provider simulator publish their development ports only on IPv4 loopback. API and web bindings are unchanged. |
| K06 | The balance helper verifies the stored obligation/request relationship and principal, requires each normalized update to return a row, and updates only a matching, structurally valid, current-version snapshot. It uses the version returned by the authoritative request update. |
| K07 | Every physical runtime connection checks both the login and effective role, including privileged memberships and ownership. Provisioning refuses privileged or login-enabled runtime-role drift. The intentional backup role is not weakened or changed. |
| K08 | Migration 204 adds per-boot heartbeat records without removing legacy slots. The watcher reports instance/build identity. The setup endpoint and UI display individual instances, identify legacy shared slots, and disclose incomplete lists. |
| K09 | A shared transaction-cleanup helper uses an independent five-second context, preserves rollback errors, and preserves panics. Audited database wrappers and settings transactions use it. No transaction or commit retry was added. |
| K10 | `infra/postgres/roles.sql` encloses its complete grant/revoke sequence in one transaction, preventing a later failure from committing an earlier partial permission policy. |

## Review and verification performed

The changed Go source was formatted with `gofmt` and parsed with the Go standard-library parser. The changed TypeScript script block was parsed with the installed TypeScript compiler. Compose YAML was parsed; all six intended infrastructure port bindings were checked. Patch context and whitespace were checked for locally reconstructed originals verified against their Git blob hashes. The credit-note approval caller and the text-keyed aggregate snapshot schema were read before strengthening the balance helper.

No tests were written or run. No database migration, grant script, production application, or financial operation was executed. A complete Go build and Svelte build were not performed: the local Go toolchain is 1.23.2 while the repository requests 1.26.8, and the repository's dependency tree and Svelte compiler were not available locally. Parsing is not semantic compilation or runtime verification.

The two new heartbeat-table grant entries were added to the inventory from the reviewed SQL. The older inventory entries were preserved, not regenerated or certified against a live database.

## Rollout requirements

Apply migration `204_runtime_process_instances.sql` before starting these API/worker binaries. Startup now refuses a database missing the new table. Apply `infra/postgres/roles.sql` as a standalone script with the migration/admin connection and error-stop enabled; the script owns its transaction. Do not embed it inside another transaction whose commit is owned by a caller.

Correct any runtime-role drift identified by the new checks before bringing the new instances online. Do not grant runtime users ownership, `BYPASSRLS`, or a path to an elevated role to bypass a startup failure. The dedicated backup role intentionally remains separate.

Replace connection-string `options`/`PGOPTIONS` with named supported connection parameters. Timeout values must be positive whole-number durations and cannot exceed 2,147,483,647 milliseconds. URL timeout values take precedence over environment defaults but must satisfy the same validation. Explicit invalid environment values are rejected even when a URL would otherwise override them.

`RUNTIME_INSTANCE_ID` is optional. When unset, the hostname is used for the display name; each process boot still receives its own random UUID. Build revision is taken from Go VCS build information when available and is otherwise reported as `unknown`.

Instance status describes observed boots, not the supervisor's authoritative replica inventory. The endpoint returns up to 500 records from the last 24 hours, including recent legacy shared slots, and reports truncation. Stale per-boot records older than seven days are pruned in bounded batches by the corresponding runtime role. A stopped or stale instance is not counted as proof that every expected replica is up to date.

Configuration auto-apply continues to depend on the external supervisor restarting a process after graceful shutdown. Worker readiness indicates local critical scheduling progress and database availability; it does not certify provider approval, completion of every queued job, or fleet-wide health.

## Financial boundary

The balance helper remains part of the caller's transaction. Callers must install the tenant context, hold the appropriate financial locks, and roll back the whole transaction on any returned error. The inspected credit-note approval path does this; other private balance-writing implementations are not consolidated by this change. A missing, stale or inconsistent snapshot now causes a refusal rather than a partial-success claim. Diagnose the inconsistency before retrying; an uncertain commit outcome is not an instruction to replay a financial mutation.

## Remaining audit scope

These changes address the first batch's source findings only. They do not close unrelated ledger, provider, authorization, migration, frontend, or deployment reviews that have not yet been completed. Merge and rollout remain separate actions.
