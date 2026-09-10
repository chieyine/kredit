# Local implementation checkpoint — 7 September 2026

This is an **incomplete release checkpoint**, not acceptance of the full replacement brief.
The authoritative baseline is in `baseline.txt`; `changes.patch` and `source-manifest.json`
identify the delivered working tree. The owner's replacement brief remains unmodified.

## Implemented

- Guest sign-in skips `/me` without a session cookie. An optional existing-session lookup
  cannot block the sign-in form. OTP and protected API checks remain in place.
- Pricing is read during server rendering from the real public business-policy endpoint,
  with no-store caching. It uses the same policy revision and integer calculation helpers.
  The isolated database returned 50 basis points for both rates; no fees were invented.
  A missing source still produces a scoped unavailable state, not free pricing.
- Terms, privacy and complaints use the supplied incorporated company details and general
  contact. Repetitive summary panels were removed while underlying substantive clauses
  remain. A shared legal document layout provides mobile disclosure and desktop anchors.
- The initial publication has stable version/date metadata. New onboarding and new sale
  records use the new document identifiers. Existing database agreements were not rewritten.
- Settings reject unknown registry keys, null values and manual `verified` status writes.
  Secret writes reject missing/short independent encryption roots. Decryption failures no
  longer return encrypted payloads as if they were plaintext.
- Setting and secret edits require the reviewed version; a transaction lock protects the
  comparison/write. History responses redact secret payloads. Migration 088 prevents history
  update/delete and revokes those runtime grants.
- Public capabilities no longer return launch/governance/owner diagnostics. Trade-line and
  drawdown creation UI starts closed until an affirmative capability response.
- Owner bootstrap now requires a recent AAL2 session for an existing active identity,
  refuses reuse under a table lock, and requires the audit write to succeed.

## Verified limits and remaining work

The full supplier/buyer/owner redesign and end-to-end acceptance matrix are NOT complete.
`route-state-inventory.csv` is a route inventory, not proof of those journeys. The settings
inventory explicitly leaves consumer tracing unverified.

The following remain implementation/review work, not requests for the owner to approve an
unfinished release:

1. Connect every supported integration setting to API/provider/worker construction and
   preserve reconciliation across credential rotation. Current general settings saving
   must not be treated as proof of a changed adapter.
2. Replace the initial code-backed legal publication with the required owner-managed draft,
   preview, publish, history and restore workflow. Current metadata is not that workflow.
3. Complete atomic multi-setting changes, versioned rollback, measured provider evidence,
   cross-instance propagation and all governance failure paths.
4. Finish and test owner transfer/recovery, suspension/expiry and last-owner concurrency.
   The revised bootstrap command is built/tested only to the extent stated in the test log;
   do not use bootstrap as a recovery bypass.
5. Review all signed-in populated/exception states, workers, notifications, payment journeys,
   organisation races, accessibility, browser engines, mobile devices, load and restore.
6. Production optional-provider startup separation and final deployment configuration remain
   unverified. No external provider, financial, mailbox or message delivery call was made.

Public routes inspected work in the isolated local development runtime. Enabled customer
journeys and live bank collection are not release-certified. Physical devices, Firefox,
WebKit and assistive-technology use were not tested. Race instrumentation did not start with
this installed Go toolchain; see `evidence/race.log`.

## Reproduce the local environment

Do not load the owner's `.env` for these checks. The disposable cluster created for this task
is `/tmp/kredit-launch-v2-pg`, PostgreSQL 18, loopback port 55432, database `kredit_launch_v2`.
It contains synthetic integration-test data only and uses local test credentials.

Start the existing isolated cluster if it is stopped:

```sh
/usr/local/opt/postgresql@18/bin/pg_ctl -D /tmp/kredit-launch-v2-pg -l /tmp/kredit-launch-v2-pg/server.log -o '-h 127.0.0.1 -p 55432 -k /tmp' start
```

From this project, run the API in a separate terminal (the task API may already use 8080):

```sh
APP_ENV=development API_ADDR=127.0.0.1:8080 DATABASE_URL='postgres://kredit_app_login:local-isolated-app@127.0.0.1:55432/kredit_launch_v2?sslmode=disable' RIVER_DATABASE_URL='postgres://kredit_worker_login:local-isolated-worker@127.0.0.1:55432/kredit_launch_v2?sslmode=disable' go run ./cmd/api
```

Run the web app:

```sh
pnpm --dir web exec vite --host 127.0.0.1 --port 5173
```

Build and serve the Node web release locally:

```sh
KREDIT_WEB_ADAPTER=node pnpm --dir web build
HOST=127.0.0.1 PORT=3000 ORIGIN=http://127.0.0.1:3000 API_INTERNAL_URL=http://127.0.0.1:8080 node web/build
```

The API above deliberately uses development adapters. It is not a production financial
runtime. Workers were not started for financial execution. Do not point this fixture setup
at production or reuse these local credentials.

## Owner access

The actual bootstrap entry point is `go run ./cmd/bootstrap-owner -identifier '<verified identity>' -reason '<reason>'`,
with the intended database owner connection supplied via `DATABASE_DIRECT_URL`.
The identity must first sign in at `/app` and complete MFA at `/app/settings/security`
within ten minutes. Bootstrap is one-time; it no longer creates an unverified account.
The owner console is `/admin/platform-settings`; fee policies are `/admin/settings`.
No real owner identity, MFA secret or recovery credential was created during this work.
The complete owner recovery procedure remains unverified; this checkpoint is not sufficient
for live owner activation.

`SETTINGS_ENCRYPTION_KEY` must be an independently generated secret of at least 32 bytes.
There is no TOKEN_SECRET/TOKEN_HASH_KEY fallback. Keep existing approved encryption material
when upgrading an environment with stored secrets; do not rotate it casually. A key-ID and
legacy ciphertext migration procedure remains to be completed.
