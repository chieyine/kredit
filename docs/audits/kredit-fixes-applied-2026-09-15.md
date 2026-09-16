# Kredit — fixes applied

Working tree at `aa12001`. 29 files changed, 8 added. Nothing committed; review
the diff before you commit.

## A correction to my earlier report first

I overstated finding 1. I classified a policy as an unconstrained blanket check
whenever it had no direct `current_organization_id` / `current_user_id`
reference. That was wrong for policies that filter through a subquery on another
RLS-protected table: `allocation_access` on `app.payment_allocations` reads
`app.obligations`, and `drawdown_line_access` on `app.drawdowns` reads
`app.trade_lines`. Those subqueries are themselves subject to the referenced
table's policies for the querying role, so they inherit its tenant scoping and
are fine.

Re-run with the distinction: 111 policies with a direct tenant predicate, 11
transitive, **55** true blanket role checks, and **14** tables — not 26 — where a
tenant policy is genuinely OR'd away. `app.payment_mandates`, `app.drawdowns`,
`app.drawdown_reservations`, `app.payment_allocations`, `app.consumer_*` and
`app.dsa_*` were on my original list and should not have been.

The finding is real and still the most serious one. It is smaller than I said.

## Fixed

**Backup TLS.** `InsecureSkipVerify` is gone from `cmd/backup-r2/main.go`;
uploads use the platform trust store with TLS 1.2 minimum. Also: the Postgres
container name is now `BACKUP_POSTGRES_CONTAINER` rather than a constant, and a
failed offsite upload exits non-zero instead of printing a warning and reporting
success — a local-only copy is not a backup.

**WhatsApp assistant no longer claims to record anything.** "✅ Sale Confirmed!
Invoice recorded" and "💰 Payment Recorded" are gone. Every reply now describes
what the seller still has to do and links to the authenticated surface. The
reply logic is extracted into `assistantReply` so the property is testable, and
`internal/web/meta_assistant_test.go` asserts across every intent — including an
intent the switch doesn't know — that no reply contains a completion claim, that
model-supplied text is never echoed back, and that amounts read back exactly.

**Gemini controls.** Off unless `FEATURE_WHATSAPP_ASSISTANT` is set; the runtime
builds no parser at all when it isn't. Production startup requires a valid key
*and* `WHATSAPP_ASSISTANT_TRANSFER_REFERENCE`. The key moved from the URL query
string to the `x-goog-api-key` header. Per-sender budget of 12 requests per 10
minutes, with a bounded sender map. Voice notes are no longer downloaded when
the assistant is unavailable — that used to cost a media fetch regardless.
Provider error bodies are no longer logged, because they echo the message.
Audio is capped and the response body is bounded.

**Phone sign-in dependency is now visible.** New `phone_sign_in_channel`
readiness gate fails in production when WhatsApp isn't configured, and the
runtime error tells the operator what to do instead of "notification channel is
disabled". I did not add a silent SMS fallback: pinning delivery evidence to one
channel is a deliberate decision in this codebase and reversing it is a product
call, not a bug fix.

**Client IP can no longer be chosen by the caller.** `API_TRUSTED_PROXIES` (IP
or CIDR, validated at load) now decides who may set `X-Real-IP`. Once
`FRONTEND_PROXY_SIGNING_KEY` is configured, an unsigned header from an
undeclared private peer is ignored. `internal/web/client_ip_test.go` covers all
three cases. I used an allowlist rather than simply refusing the header, because
refusing it outright would have collapsed all webhook traffic onto one proxy
address and rate-limited it.

**OTP purpose is bound at consumption.** This was worse than I first reported:
`/auth/otp/challenges` is unauthenticated and took the purpose straight from the
caller, so a code could be minted as one purpose and spent as another. The
public route is now restricted to `login` and `recovery`; `VerifyOTP` requires
`login`, `VerifyOTPForTarget` takes an explicit purpose, and
`VerifyAndAttachIdentifier` requires `supplier_contact_verification`. Both
stores enforce it.

**Identity document uploads.** The proxy limit matched only `/documents`, so the
singular `/document` identity route was capped at 2 MB of base64 — a real
ceiling of about 1.5 MB against a UI that promises 2 MB. Now matches both.

**Drawdown confirmation replay.** `ConfirmDrawdown` returned 200 for a cancelled
or expired drawdown that had previously been confirmed. It now only replays on
the confirmed path.

**A background context in a live write path.** `applyMandateToBuyerResources`
discarded the request context, so a cancelled mandate cancellation kept writing.
It now takes `ctx`.

## Found while fixing

**`task ci` was already failing at `HEAD`.** `scripts/phase6-context-audit.py`
flagged three files that are unmodified in your working tree. Two of the five
hits were legitimate — `defer tx.Rollback(context.Background())` must survive
client disconnect — but the file-level check meant the one real defect
(`mandate_handlers.go`, above) was hidden behind them. The audit is now
line-precise: it permits a rollback and reports everything else with a line
number. It passes, and it is stricter than before. I verified it still fails on
a planted violation.

**The secret scan passed or failed depending on which tool you had installed.**
`scripts/security.sh` prefers ripgrep, which honours `.gitignore` and so skips
your local `ssh/` keys. The `grep` fallback does not, and would have failed on
your machine. The fallback now mirrors the ignored paths.

## Not fixed, and why

**The remaining RLS work.** I did not write the migration. The blanket policies
are load-bearing right now, and converting them without first replacing what
depends on them would take the platform down:

- `internal/schedules`, `internal/support` and `internal/notifications` set no
  tenant context anywhere — every Postgres method in `schedules` uses
  `context.Background()`.
- `referrals.Refresh` opens its transaction with `s.begin(ctx, "", "")` and
  `consumer.EnqueueReminders` queries across every tenant, both deliberately.
- `usercontrol.ListRecoveries`, `usercontrol.ListPrivacyReview` and
  `platformops.Search` are admin cross-tenant reads running as `kredit_app`.

Each table needs its worker and admin callers moved to a narrow `SECURITY
DEFINER` route first — the pattern this codebase already uses well in
`app.collection_due_work_page` and `app.admin_user_directory`. That is a staged
programme, and I can't compile or run a test here, so writing it blind into a
payment platform is not something I'm willing to do.

What I did instead:

- `scripts/rls-policy-shape-check.sh` reads `pg_policies` and fails if the set of
  affected tables grows, or if a converted table is left in the baseline — so the
  list can only shrink and no new table can repeat the pattern silently. Wired
  into `ci.sh` behind `DATABASE_URL`.
- `docs/compliance/rls-permissive-baseline.txt` records the accurate 14.
- `docs/operations/rls-phase2-completion.md` gives the per-table procedure, the
  evidence for each blocker, and a suggested order starting with the two that
  need only an admin route.
- `TENANT-ISOLATION-PHASE2` is now a tracked workstream, so `plan:check` keeps it
  visible instead of it sitting outside every gate.

**The transfer assessment for Gemini.** I recorded where it belongs and made
startup refuse without it. Writing it is your call, not mine.

## New gates

| Gate | What it catches |
|---|---|
| `scripts/sub-processor-check.sh` | A new external host in Go source with no entry in `docs/compliance/sub-processors.md`. This is the gap that let Gemini through: the field-level data inventory diffs database columns, and a new outbound transfer adds none. Verified it fails on a planted host. |
| `scripts/rls-policy-shape-check.sh` | A tenant policy newly OR'd away by a permissive blanket policy. |
| `internal/web/meta_assistant_test.go` | Any assistant reply that claims a financial record was written. |
| `internal/web/client_ip_test.go` | A rate-limit key that the caller can choose. |
| `internal/whatsapp/ai_test.go` | Replaced the live-network test that always skipped with deterministic budget and bounds tests. |

Both new scripts are in `Taskfile.yml` (`task data:subprocessors:check`,
`task db:rls:check`) and required by `readme-conformance.sh`.

## What I could verify, and what I couldn't

`proxy.golang.org` is blocked by this environment's egress policy and your Mac
has no Go toolchain, so I could not build or run tests. What I did run:

- `gofmt -e` on all 19 changed Go files — no parse errors. Two files needed
  formatting and were formatted, so `scripts/lint.sh` won't trip.
- `readme-conformance.sh`, `implementation-plan-conformance-test.sh`,
  `phase6-context-audit.py`, `sub-processor-check.sh`, `load-env-test.sh` — all
  pass.
- `bash -n` on every script I touched.
- Negative tests on both new gates and on the sharpened context audit, to prove
  they fail closed.

**Everything else needs your machine.** In particular the Go compiler has not
seen the `VerifyOTPForTarget` signature change, which touches the `auth.Service`
interface, both stores and three call sites.

```sh
cd ~/Documents/Kredit.com
go build ./... && go vet ./...
go test ./internal/auth/... ./internal/web/... ./internal/whatsapp/... ./internal/config/...
task ci
```

Then, with a database up, the two gates that need one:

```sh
bash scripts/rls-policy-shape-check.sh
bash scripts/data-inventory-check.sh
```

## One housekeeping item

`_to_delete/zz_probe.go.txt` is a throwaway file I created to prove the
sub-processor gate fails closed. I could not delete it — file deletion in that
folder is blocked for this session — so I moved it there. Remove the
`_to_delete/` folder when convenient.
