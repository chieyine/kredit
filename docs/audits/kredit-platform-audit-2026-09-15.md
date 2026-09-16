# Kredit platform review — findings and commands to run

Reviewed at commit `aa12001` ("Support unregistered informal trader verification…"), 15 September 2026.

## The short version

The platform is in good shape. The financial core — double-entry ledger, kobo integer money with overflow guards, the transactional outbox, idempotency fencing, the provider reconciliation model, the least-privilege PostgreSQL role design — is better than most production fintech code I've read. All 52 directly-callable `SECURITY DEFINER` functions pin `search_path` and are revoked from `PUBLIC`. The CI pipeline is strict and fails closed.

The problems cluster in one place: **the five commits made after the last full audit** (`311f89d`). Those are `bae6664` (R2 backup), `752b094` (docker logging), `9682878`/`346328f` (proxy and ingress changes), `69b132e` (WhatsApp Gemini AI), `5862e62` (consumer sales, DSA, WhatsApp OTP) and `aa12001` (unregistered traders). Consumer sales and DSA were built to the repo's usual standard — migrations, roles, OpenAPI, docs, tests. The others were not.

Separately, there is one long-standing database issue that predates all of that and is the most important item on this list.

---

## HIGH

### 1. Tenant isolation is not enforced at the database layer on 26 tables

PostgreSQL combines multiple **permissive** policies on a table with `OR`. The schema has **193 active policies and zero `RESTRICTIVE` policies**. On 26 tables a tenant-scoped policy sits alongside a blanket `current_user IN ('kredit_app','kredit_worker')` policy, so the effective rule is:

```
tenant_predicate  OR  current_user IN ('kredit_app','kredit_worker')
```

The application connects as exactly those roles (`internal/db/postgres.go:74`, `RuntimeParams["role"]`), so the right-hand side is always true and the tenant predicate never constrains anything.

Affected tables include `app.payment_mandates`, `app.payment_allocations`, `app.repayment_schedules`, `app.drawdowns`, `app.drawdown_reservations`, `app.disputes`, `app.correction_requests`, `app.correction_decisions`, `app.notifications`, `app.notification_preferences`, every `app.privacy_*`, every `app.account_recovery_*`, `app.processing_restrictions`, every `app.dsa_*` and the `app.consumer_*` tables.

A further set has only the blanket policy and no tenant policy at all: `app.schedule_items` (instalment amounts and collection dates), `app.agreement_versions` (the canonical agreement JSON), `app.dispute_evidence`, `app.dispute_decisions`, `app.support_cases`, `app.support_case_events`, `app.operation_actions` (write-offs and fee waivers), `app.mandate_events`, `app.provider_customer_bindings`, `app.notification_delivery_receipts`.

Migration `081_phase2_tenant_isolation.sql` handled this correctly for the core money tables — it **dropped** 38 blanket policies before creating 23 tenant-scoped ones. The pattern is understood; it just was not finished. Migration 081 does not mention schedules, disputes, mandates or notifications at all, and `tests/integration/tenant_isolation_test.go` asserts against `app.obligations`, one of the tables where the drop *was* done. So the Phase 2 gate passes green while these 26 remain open.

This is a defence-in-depth failure rather than a live exploit: the handlers do set tenant context on most paths. But it means the database will not stop a handler bug, and I found three paths that query these tables with no tenant context at all:

- `internal/schedules/store.go` — `getPostgres`, `allocatePostgres`, `evaluatePostgres`, `collectionTargetPostgres`, `markCollectedPostgres` all use `context.Background()` and select by `obligation_id`/`schedule_id` only.
- `internal/support/store.go` — `Get`, `Read`, `Timeline`, `TransitionContext` are keyed by `caseID` alone (the HTTP handlers do check the org afterwards, which is what currently saves it).
- `internal/platformops/store.go` `Search` — sets `app.current_user_id` but deliberately not the organisation, then runs a cross-tenant UNION.

**Fix:** for each affected table either drop the `*_runtime*` / `*_worker*` / `*_access` blanket policy the way 081 did, or re-declare it `AS RESTRICTIVE`. Then extend `tests/integration/tenant_isolation_test.go` to cover `schedule_items`, `payment_mandates`, `disputes` and `notifications`, not just `obligations`.

### 2. Production database backups upload over unverified TLS

`cmd/backup-r2/main.go:120-125`:

```go
tr := &http.Transport{
    TLSClientConfig: &tls.Config{
        InsecureSkipVerify: true,
    },
}
```

This is the client that uploads a full `pg_dump` of the production database to Cloudflare R2. Certificate verification is disabled, so any party able to intercept the connection receives the entire database — users, mandates, the ledger, encrypted identity fields and all. R2 presents a valid public certificate; there is no reason for this flag.

Line 59 also hardcodes `docker exec kredit-prod-postgres-1 …`, which will silently break if the Compose project name or service index changes.

**Fix:** delete the custom transport and use the default `http.Client`. If a specific CA is genuinely needed, load it into `RootCAs`.

### 3. The WhatsApp assistant tells sellers a sale was recorded when nothing was recorded

`internal/web/meta_handlers.go`, the Gemini branch. On `IntentConfirm` the bot replies:

> ✅ *Sale Confirmed!* Invoice recorded. A notification and payment link have been dispatched to the buyer.

And on `IntentRecordPayment`:

> 💰 *Payment Recorded:* …

Neither statement is true. The handler parses the message, writes a row to `app.messaging_events` for replay suppression, and sends a reply. `s.runtime.WhatsApp.Handle(...)`'s return value is discarded (`_, _ =`). No credit request is created, no payment is recorded, no notification is sent to the buyer. I checked every reference to `IntentCreateCredit`, `IntentConfirm` and `IntentRecordPayment` in the repo — `internal/whatsapp/ai.go` and this handler are the only ones.

A seller who releases goods on the strength of "Sale Confirmed! Invoice recorded" has no agreement, no mandate and no obligation. In a trade-credit product that is the worst possible false statement to make.

The non-AI fallback path in the same handler is honest — it says "Review and confirm in Kredit" and links to the app. README §29.2 also states the rule: "Before a financial action, show a confirmation summary." The AI path shows the summary and then claims the action happened.

**Fix:** change the AI replies to describe a draft and link to the app for confirmation, matching the fallback path — or wire the intent through to the real credit-creation flow behind authentication. The first is a one-hour change; the second is a project.

### 4. Gemini is an undeclared cross-border sub-processor

`internal/whatsapp/ai.go` sends WhatsApp message text — buyer names, amounts, goods descriptions — and raw voice-note audio to `https://generativelanguage.googleapis.com`. Checks I ran:

- `docs/compliance/data-inventory.tsv` has `processor` and `location_transfer` columns and lists `meta` 12 times. There is **no entry for Google or Gemini**.
- Zero mentions of Gemini, Google generative AI, voice notes or audio processing anywhere in `README.md`, `docs/` or `api/openapi.yaml`.
- `GEMINI_API_KEY` appears only in `.env.example`. `internal/config/config.go` reads it with `envOr(...)` and applies no production validation.
- The API key is passed in the URL query string (`?key=%s`), so it lands in any proxy or request log on the path.
- There is no per-sender rate limit. Every inbound WhatsApp message triggers a Gemini call, and voice notes trigger a media download plus an audio-model call. Anyone who can message the business number can drive unbounded spend.

The reason this got through is structural, not careless — see finding 9.

**Fix:** add the Gemini flow to the data inventory with a lawful basis and a transfer assessment; document it in the privacy notice; move the key to an `x-goog-api-key` header; add a per-sender rate limit; and gate the whole feature behind a feature flag that is off by default.

### 5. Phone sign-in and account recovery now depend on WhatsApp alone, with no configuration check

`internal/notifications/store.go`:

```go
// SendOTP sends email codes by email and phone codes by WhatsApp.
func (s *Store) SendOTP(ctx context.Context, recipient, channel, code string) error {
    if channel == "phone" { channel = ChannelWhatsApp }
```

`internal/notifications/otp_routing_test.go` asserts this deliberately, including "failed phone authentication must not silently switch channels". Mesaj SMS is configured and validated in production config but is never used for authentication.

Three consequences:

- `internal/config/config.go:515` only requires `NOTIFICATION_WHATSAPP_*` when `FEATURE_WHATSAPP` is on. A production deployment with WhatsApp off and SMS/email correctly configured **passes startup validation and then returns 503 on every phone sign-in** ("notification channel is disabled"). Nothing warns you.
- If Meta has an outage or a user's number is not on WhatsApp, every phone-identified user is locked out of both sign-in and account recovery, with no fallback by design.
- README §29.4 says "never request OTP, PIN, or online-banking password" in WhatsApp and "fall back to email/SMS for critical events". The implementation now does the opposite on both counts. An OTP in a WhatsApp notification preview on a lock screen is exactly the exposure §29.4 warns about.

**Fix:** at minimum, make production config validation refuse to start when phone identifiers are accepted and WhatsApp is not configured. Better: restore SMS as the fallback for authentication codes, and reconcile §29.4 with whatever you decide.

---

## MEDIUM

### 6. The auth rate limiter's client-IP key is spoofable from inside the network

`internal/web/server.go:829-852`. `clientIP` prefers the HMAC-signed `X-Kredit-Client-IP` from the SvelteKit proxy, which is sound. But it then falls back to trusting a plain `X-Real-IP` header from **any** peer whose remote address is private or loopback.

The comment above the function states the safety argument:

```go
// Caddy overwrites X-Real-IP for direct API traffic; the API has no public port.
```

That is no longer true. Commit `346328f` moved the `caddy` service behind a `standalone` Compose profile (so it does not start by default) and published the API on `127.0.0.1:8080` for 1Panel. The actual ingress is now 1Panel's proxy, whose configuration is not in this repo.

`clientIP` is the only key for the per-replica 120/minute limiter, the cross-replica `app.record_rate_limit_attempt` budget (20 per 10 minutes on auth routes), and the account-recovery throttle in `user_control_handlers.go:95` (5 per hour). Anything that can reach the API on the Docker network or on loopback can rotate `X-Real-IP` per request and defeat all three — which turns the OTP brute-force limit and the recovery-request limit into no limit at all. Whether that is reachable from the internet depends entirely on whether 1Panel's proxy strips the header.

**Fix:** restrict the `X-Real-IP` fallback to an explicit trusted-proxy allowlist, or require the signed header when `FRONTEND_PROXY_SIGNING_KEY` is set. Either way, update the stale comment.

### 7. The deployment guide and the production Compose file disagree

`docs/operations/PRODUCTION-DEPLOYMENT-GUIDE.md` says "There is no host `127.0.0.1:8080` listener in this stack" and documents the Caddy + Cloudflare origin configuration as the ingress. `infra/environments/docker-compose.prod.yml` publishes `127.0.0.1:8080:8080` on the api service and puts caddy behind `profiles: [standalone]`.

The consequence is that the hardening you can verify from this repo — `trusted_proxies_strict` with the Cloudflare ranges, `header_up -CF-Connecting-IP`, `-True-Client-IP`, `-Forwarded`, HSTS preload — is not what runs. Whatever 1Panel is doing is unreviewed and unreviewable from here. Finding 6 is a direct consequence.

**Fix:** decide which ingress is real, make the Compose file and the guide agree, and check the 1Panel proxy config into the repo (or move back to the Caddy profile).

### 8. Identity-document uploads above roughly 1.5 MB fail with a confusing error

`web/src/hooks.server.ts` sizes the proxy body limit by path suffix:

```ts
size > (event.url.pathname.endsWith('/documents') ? 3 : 2) * 1024 * 1024
```

The identity route is `POST /api/v1/identity/checks/{provider}/{caseID}/document` — singular (`internal/web/server.go:529`), so it gets the 2 MB limit. The Go handler accepts a 3 MB JSON body carrying a 2 MB file, and `IdentityChecks.svelte` tells the user "Choose a PDF or image up to 2 MB". Base64 inflates by 4/3, so a genuine 2 MB document becomes ~2.7 MB of JSON and the proxy returns 413 before the API sees it.

Real ceiling is about 1.5 MB. This hits unregistered traders uploading identity evidence — the newest onboarding flow.

**Fix:** match on `/document` as well as `/documents`, or size the limit from the route table rather than a suffix.

### 9. The verification gates are frozen at the pre-audit feature set

This is why findings 3, 4 and 5 shipped green.

- `scripts/readme-conformance.sh` is a fixed allow-list of files, routes, tables and strings that must exist. It has no mechanism to require that a *new* feature be documented.
- `scripts/implementation-plan-conformance.sh` requires evidence for exactly 11 hardcoded workstream IDs. Consumer sales, DSA and the WhatsApp AI are not among them; `docs/product/workstream-evidence.tsv` contains zero rows mentioning any of them.
- `scripts/data-inventory-check.sh` diffs the TSV against `information_schema.columns`. It catches a new *column*; it cannot catch a new *outbound data flow to a new processor*, which is exactly what Gemini is. It also requires `DATABASE_URL`, so `ci.sh` skips it entirely without one.
- `/meta/webhook` has no entry in `api/openapi.yaml`, so `api-lint.sh` has nothing to check.

**Fix:** add a gate that fails when a new external host appears in Go source without a matching data-inventory processor row, and add workstream rows for the three new features so `plan:check` starts requiring evidence for them.

### 10. The WhatsApp AI has no effective test coverage

`internal/whatsapp/ai_test.go` is the only test, and it begins:

```go
apiKey := os.Getenv("GEMINI_API_KEY")
if apiKey == "" { t.Skip("GEMINI_API_KEY is not set, skipping live test") }
```

CI never sets it, so the test always skips. Even when it runs it asserts exact field values from a non-deterministic LLM, so it would be flaky rather than useful. `metaWebhook` — the handler that contains the false-confirmation replies — has **no test at all**, in either Go or Playwright. Every other package in this repo has real coverage; `internal/web` alone has 25 test files.

**Fix:** test `metaWebhook` against a stubbed parser, table-driven over each intent, asserting the reply text. That single test would have caught finding 3.

### 11. `/api/v1/ops/search` probably returns nothing useful

`internal/platformops/store.go` `Search` runs a cross-tenant UNION over credit requests, documents, payments, collection attempts, support cases and disputes, with only `app.current_user_id` set and no organisation. Today it works because of the blanket policies in finding 1 — which means fixing finding 1 will break it. Worth deciding now which behaviour you want.

### 12. Background contexts in the schedule and support stores

`internal/schedules/store.go` and `internal/support/store.go` use `context.Background()` for pool-level queries, so request cancellation and deadlines do not propagate and no tenant context is set. `scripts/phase6-context-audit.py` only audits `internal/web`, so these are outside the gate.

---

## LOW

- **OTP purpose is not bound at consumption.** `app.otp_challenges.purpose` is stored but neither `verifyOTP` nor `VerifyAndAttachIdentifier` checks it, so a code issued for one purpose can in principle be redeemed for another. The challenge ID is required and single-use, which limits the practical impact.
- **`credit.GetPublic` reads only the bounded 512-entry in-process projection**, so a public payment link can 404 after enough churn or a restart.
- **`tradelines.ConfirmDrawdown` returns 200 for a cancelled drawdown** that was previously confirmed — the replay branch checks `State != DrawdownPending && !BuyerConfirmedAt.IsZero()` without excluding `CANCELLED`/`EXPIRED`. No state changes; the response is just misleading.
- **19 tables have no RLS at all**, including `ledger.transactions`, `ledger.postings`, `ledger.accounts`, `app.mandates`, `app.agreement_acceptances`, `app.goods_releases`, `app.receipt_confirmations`, `app.outbox_events` and `app.otp_challenges`. Largely mitigated: `infra/postgres/roles.sql` grants the runtime roles `SELECT, INSERT` only on the ledger schema, so the journal is append-only. Cross-tenant *reads* remain possible from a buggy path.
- **Only `app.audit_events` uses `FORCE ROW LEVEL SECURITY`.** Fine while the runtime roles are not the table owner, which is currently the case — but it is one `ALTER TABLE … OWNER TO` away from silently disabling every policy.
- **`referrals.Read` returns full bank account numbers** for DSA agents and payouts. Agents see their own, which is reasonable; platform owners see everyone's. Compare with the settlement path, which deliberately stores only `account_last4`.

---

## What I actually reviewed

Complete line-by-line reads: every non-test Go file under `internal/` (~45 packages) and `internal/web/` (54 handler files including the 1,820-line `credit_handlers.go`), all of `cmd/`, `infra/postgres/roles.sql`, `infra/containers/*`, `infra/environments/*`, every CI gate script, all seven GitHub workflows, the frontend core (`hooks.server.ts`, `lib/api/*`, `lib/money.ts`, `lib/records.ts`, `lib/seo.ts`, `service-worker.ts`, `admin-client.ts`) and the new-feature components, and the foundational migrations.

Programmatic analysis across all 153 migrations: every `CREATE`/`DROP POLICY` resolved in positional order to the 193 active policies; all `SECURITY DEFINER` functions checked for `search_path` pinning and `PUBLIC` revocation; all `GRANT`/`REVOKE` statements; RLS enablement per table.

Structural rather than line-by-line: the 183 Go test files (mapped by package and checked for coverage of the new features), the remaining ~200 Svelte route files (scanned for `@html`, storage, credential and XSS patterns), and the 111 markdown documents (searched against the specific claims each finding depends on).

I could not compile or run anything: `proxy.golang.org` is blocked by the egress policy in this environment, the linked Mac has no Go toolchain, and the cloud container has Go 1.24.7 against your required 1.26.8. Everything below is for you to run locally.

---

## Commands to run on your Mac

From `/Users/macbookpro/Documents/Kredit.com`.

### The full gate

```sh
task ci
```

That is `scripts/ci.sh`: API lint, phase-6 governance and context audits, README and plan conformance, env loading, `pnpm run audit`, `go test ./...`, `go test -race` on the financial packages, `scripts/lint.sh`, `scripts/security.sh`, and the frontend check/build/test. It skips the integration and data-inventory steps unless `DATABASE_URL` is set.

### Individually, if you want to see where it stops

```sh
go build ./...
go vet ./...
gofmt -l .

go test ./...
go test -race ./internal/collections ./internal/credit ./internal/payments ./internal/reports

bash scripts/lint.sh          # needs golangci-lint
bash scripts/security.sh      # see below for strict mode
bash scripts/api-lint.sh
bash scripts/readme-conformance.sh
bash scripts/implementation-plan-conformance-test.sh
python3 scripts/phase6-context-audit.py
bash scripts/phase6-governance-test.sh

pnpm install --frozen-lockfile
pnpm run audit
pnpm --dir web check
pnpm --dir web build
pnpm --dir web test
```

### Security scanners, failing closed

`scripts/security.sh` silently skips any scanner that is not installed unless you force it. CI sets `SECURITY_STRICT=1`; locally you should too:

```sh
go install golang.org/x/vuln/cmd/govulncheck@v1.7.0
go install github.com/securego/gosec/v2/cmd/gosec@v2.28.0
go install honnef.co/go/tools/cmd/staticcheck@v0.7.0
go install github.com/google/osv-scanner/v2/cmd/osv-scanner@v2.4.0
brew install golangci-lint trivy
npm install --global @redocly/cli@2.51.2

SECURITY_STRICT=1 OPENAPI_LINT_STRICT=1 bash scripts/security.sh
```

### The database gates

```sh
docker compose up -d postgres

export DATABASE_URL='postgres://kredit:kredit@127.0.0.1:5432/kredit?sslmode=disable'
export DATABASE_DIRECT_URL="$DATABASE_URL"
export APP_DATABASE_URL='postgres://kredit_app_login:kredit-app-development-only@127.0.0.1:5432/kredit?sslmode=disable'
export RIVER_DATABASE_URL='postgres://kredit_worker_login:kredit-worker-development-only@127.0.0.1:5432/kredit?sslmode=disable'

go run ./cmd/migrate
psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -f infra/postgres/roles.sql
psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 \
  -v app_login_password=kredit-app-development-only \
  -v worker_login_password=kredit-worker-development-only \
  -v backup_login_password=kredit-backup-development-only \
  -v demote_owner=false \
  -f infra/postgres/development-logins.sql

bash scripts/test-integration.sh
bash scripts/data-inventory-check.sh
bash scripts/database-safety-test.sh
```

Then the whole pipeline with the database in play:

```sh
CI=true CI_REQUIRE_DATABASE=1 SECURITY_STRICT=1 OPENAPI_LINT_STRICT=1 bash scripts/ci.sh
```

### Confirm finding 1 yourself

With the database up, this prints every table where a tenant policy is being OR'd away by a blanket role check:

```sh
psql "$DATABASE_URL" -X -c "
SELECT schemaname||'.'||tablename AS table_name,
       count(*) FILTER (WHERE qual LIKE '%current_organization_id%'
                           OR qual LIKE '%current_user_id%') AS tenant_policies,
       count(*) FILTER (WHERE permissive = 'PERMISSIVE'
                           AND qual LIKE '%current_user%kredit_app%') AS permissive_blanket
FROM pg_policies
WHERE schemaname IN ('app','ledger')
GROUP BY 1
HAVING count(*) FILTER (WHERE qual LIKE '%current_organization_id%'
                           OR qual LIKE '%current_user_id%') > 0
   AND count(*) FILTER (WHERE permissive = 'PERMISSIVE'
                           AND qual LIKE '%current_user%kredit_app%') > 0
ORDER BY 1;"
```

And the empirical proof — as `kredit_app` with a bogus organisation set, these should return zero rows and will not:

```sh
psql "$APP_DATABASE_URL" -X -c "
SELECT set_config('app.current_organization_id','00000000-0000-0000-0000-000000000000',false),
       set_config('app.current_user_id','00000000-0000-0000-0000-000000000000',false);
SELECT 'schedule_items' AS t, count(*) FROM app.schedule_items
UNION ALL SELECT 'repayment_schedules', count(*) FROM app.repayment_schedules
UNION ALL SELECT 'payment_mandates',    count(*) FROM app.payment_mandates
UNION ALL SELECT 'disputes',            count(*) FROM app.disputes
UNION ALL SELECT 'notifications',       count(*) FROM app.notifications;"
```

### Other checks

```sh
bash scripts/fuzz.sh
bash scripts/test-e2e.sh
bash scripts/restore-drill.sh        # needs BACKUP_DIR and RESTORE_DATABASE_URL
python3 -m unittest discover -s scripts -p 'test_phase5_*.py' -v
bash scripts/release-certify.sh      # fail-closed production certification
```

---

## Suggested order

1. Finding 3 — change the two WhatsApp reply strings. Under an hour, and it is the one that can cost a seller real goods.
2. Finding 2 — delete `InsecureSkipVerify`. Minutes.
3. Finding 5 — add the production config check so a WhatsApp-less deployment cannot silently lock out phone sign-in.
4. Finding 1 — drop or restrict the 26 blanket policies, then extend the Phase 2 integration test. A day, and it needs care.
5. Finding 4 — either document Gemini properly or turn it off until you have.
6. Findings 6 and 7 — settle what the production ingress actually is, then fix `clientIP` to match.
7. Findings 9 and 10 — add the gates that would have caught 3 and 4, so the next feature cannot repeat this.
