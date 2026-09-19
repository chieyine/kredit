# Audit repair and recovery rollout

Prepared 2026-09-19. This is a deployment procedure, not a completed deployment record. No database, provider, build or test commands were executed in preparing it.

## Order

1. Record the release commit, API/worker/web image digests, environment and named rollout/finance/on-call owners. Preserve the prior images and the matching schema baseline.
2. Verify a usable pre-release backup and its key/role access. Follow `docs/runbooks/backup-restore.md`; a checksum alone is not restore evidence.
3. Apply migrations **155 and 156** through the existing migration job before starting the new API/worker. Startup now requires 156. Migration 155 repairs scorecard branches and bank-recovery read roles; 156 adds aggregate recovery-queue metrics and an index for exact request-reference lookup. Allow for index-build locking in the migration window. The original monitoring function remains unchanged; a separate recovery function preserves older collectors during rollout. These migrations are forward-only.
4. Ensure the web and API use the same managed `FRONTEND_PROXY_SIGNING_KEY`. Deploy the matching web bundle with the new financial-review response contract. Keep old alert rules until old application instances have drained, then load the updated required-metric rules.
5. Observe readiness, authenticated metric scrapes, financial-review and bank-recovery pages, and each role's permitted actions. Missing new metrics are an error, not zero cases. Alert receivers and escalation owners must be configured externally.
6. Record the outcome and any hold. Do not claim certification from source review. Local runtime verification was subsequently authorized and completed as recorded in docs/quality. Production provider and receiver checks remain deployment-specific.

## Historical recovery

Identify the release's actual cutoff timestamp from deployment evidence. Do not use the audit date as an assumed production rollout time.

- **Monthly agreements:** review pre-cutoff equal monthly sales with `last_day` policy where the selected first due date was not month end. Also review custom schedules whose first date differs from the agreement. Compare the accepted canonical agreement and release evidence, not just today's snapshot. Drafts may be corrected before sending; already accepted or released agreements require an authorized case-specific correction and any necessary renewed agreement. Never silently rewrite accepted dates or hashes. The new code prevents new invalid terms; it does not repair existing agreements automatically.
- **Manual payment dates:** review seller-recorded transfer, cash and buyer-claim payments entered before the corrected interfaces were deployed. Compare `paid_at` with actual bank/receipt evidence. Similarity to `recognized_at` is a candidate signal, not proof of a wrong date. Preserve original receipts; no inferred date backfill is provided. Correct through the approved financial workflow only when supporting evidence exists.
- **Delivery issues:** locate open drawdown receipt disputes and contact the actual parties through support. Corrected delivery and accepted returns now have explicit application paths. Do not bulk-close them.
- **Failed cancellation:** reconcile provider references whose local cancellation was saved but provider confirmation remains unavailable. Retain the local block; do not restore authorization to dismiss the case.

For each candidate record capture only its reference, finding type, current state, case owner, evidence location, agreed disposition and closure evidence. Query/export under the authorized business scope in a restricted workspace; do not export authorization URLs, full bank data or request bodies into this repository.

## Rollback

Do not run Down migrations or restore the primary database over financial activity. Migration 156 leaves the old metrics result unchanged, so older collectors retain their original contract. Revert the new required-metric alert rules when reverting the application, or they will correctly report the new signals missing. Review compatibility of the other changes before selecting a prior image. Keep collection containment and original evidence until reconciliation completes.

## Tenant-isolation rollout update

The matching API and worker now require migrations through 164 and the updated role grants. Drain old workers and coordinate the schema/application rollout because older unscoped repositories cannot operate after blanket policies are removed. See `docs/operations/rls-phase2-completion.md`. These local changes have not been deployed by this verification task.
