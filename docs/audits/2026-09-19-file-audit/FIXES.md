# Sequential repair record

The original audit files and hashes describe the pre-repair snapshot and are retained unchanged. This record tracks subsequent fixes. No agents or tests are used.

## F001 — Implemented; source-reviewed only

- Changed `internal/ledger/store.go`, `sameTransactionIntent`, to compare a multiset of exact postings instead of matching slice positions.
- Account, debit, credit and duplicate counts must still match. Transaction event/reference checks and posting-count checks remain in place. Both memory and PostgreSQL stores use this comparison.
- Persisted posting order is unchanged. Initial writes and retries both load postings ordered by the same saved UUIDs before creating the outbox payload, preserving compatibility with existing outbox records. No migration or journal rewrite is required.
- Reviewed the changed code and both callers directly. No tests, builds, linters or database execution performed; runtime verification is not claimed.

## F002–F022 — Implemented; source-reviewed only

| Finding | Change |
| --- | --- |
| F002 | Authentication distinguishes database lookup failures from missing/expired sessions. Lookup failures and unsuccessful revocation return 503, preventing the browser from treating a storage outage as confirmed sign-out. |
| F003 | Backup retention only considers regular Kredit timestamped archives with regular checksum companions; unrelated files and symlinks are skipped. Invalid retention periods and deletion/listing errors fail explicitly. |
| F004 | Backup placeholders are rejected before creating an archive; checksum failures and unsuccessful offsite uploads fail the command. Retention runs only after upload confirmation. |
| F005 | Interrupted buyer bank authorization links to the existing `/contact` page. |
| F006 | Migration 155 aligns bank-enrollment read policy with the owner, administrator and compliance roles permitted to access the recovery queue. |
| F007 | A dedicated consumer-sale read permission includes sales staff and administrators, as well as existing financial readers. Get/List use it without changing financial-report permissions. |
| F008 | Consumer writes use the shared persistent mutation-intent implementation. Unknown outcomes, malformed successful responses and in-progress conflicts retain the request identity. Retries reuse the original payload in memory; browser storage holds only identity, timestamp and digest. Purchase and settings responses are decoded before clearing the intent. |
| F009 | Migration 155 adds the missing `days_to_payment` scorecard branch using the existing recognized-payment calculation. |
| F010 | Migration 155 counts system-acceptance evidence against activated obligations in the same window and organization. The scorecard description matches this calculation. |
| F011 | Analytics normalizes absent metadata to an empty object before validation and JSON serialization. |
| F012 | Terraform web deployment reads `FRONTEND_PROXY_SIGNING_KEY` from the same runtime secret used by the backend. |
| F013 | Feedback area and organization are normalized before handler authorization. PostgreSQL submission also checks active membership under explicit tenant context before recording seller feedback. |
| F014 | Supplier credit-list projections remove bank authorization URLs before returning views. |
| F015 | Hydrated mandates are indexed by both their internal ID and provider reference. |
| F016 | Mandate cancellation commits its local collection block before any remote call. Provider failure therefore leaves a durable local block for retry/recovery. |
| F017 | Monthly schedules default to preserving the selected day with shorter-month capping. Explicit last-day schedules require a month-end first date. Preview, creation, draft updates and send validate with the same schedule input/generator used at activation; custom first dates must match the agreement. |
| F018 | Already-released drawdowns may complete receipt after the line end date. Active-line, mandate, reserved-capacity and exposure checks remain; new drawdowns and releases retain expiry checks. |
| F019 | Buyers can confirm corrected delivery after reporting a problem. Sellers can accept returned goods and cancel the disputed purchase, releasing its reservation. PostgreSQL closes the dispute transactionally, preserves original issue evidence and records the resolution in the audit trail. Both interfaces expose the appropriate action. |
| F020 | Known pre-activation, declined and cancelled requests contribute zero to buyer balances. Missing obligations on activated or unknown states remain unconfirmed. |
| F021 | Buyer transfer reports and seller manual payments require the actual date/time, interpreted in Africa/Lagos and rejected if future-dated. Claim review labels that date explicitly and asks the seller to verify it. |
| F022 | Both sale forms load the selected business's saved payment days and grace hours. Saved zero-hour grace is preserved; restored/user-entered dates are retained. Failed settings reads block progression through the customer-loading error path. Sample dates use the saved defaults too. |

## Rollout and review limits

- All 22 confirmed findings have source fixes. Original audit evidence and hashes are unchanged.
- New migration: `db/migrations/155_audit_reporting_and_recovery.sql`. The startup persistence contract now requires migration 155. Apply it through the normal migration process before starting the updated application; it has not been executed here.
- The web runtime secret must contain the existing `FRONTEND_PROXY_SIGNING_KEY` shared with the API. No live secret or deployment was changed.
- Existing accepted agreements are not rewritten. Previously accepted invalid schedules require case-specific agreement recovery; new invalid terms are rejected before send. Historic manual-payment dates are not guessed or backfilled.
- Reviewed changed code, callers, response shapes, transaction paths, policies and migration differences directly. Go files were formatted. No agents, tests, builds, linters, browser checks, database commands or deployment were run. Runtime correctness is not claimed.


## Subsequent improvements

The follow-up implementation adds migration 156 and raises the final startup minimum to 156. See `docs/quality/2026-09-19-improvements.md` for recovery, monitoring and shared-input changes and their unverified runtime boundaries. The original audit snapshot remains unchanged.
