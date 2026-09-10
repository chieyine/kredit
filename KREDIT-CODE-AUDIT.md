> Current result: [10 September code review checkpoint](docs/launch-audit-2026-09-08/FILE-BY-FILE.md). The report below is historical; its “no fixes applied” statement does not describe the current working tree.

# Kredit — full platform code audit

> Historical source assessment of the baseline named below. Its appendix
> distinguishes close source review from pattern sweeps and inventory. It is
> not an exhaustive per-line review or current verification evidence. The
> [September 9 direct ledger](docs/launch-audit-2026-09-08/file-by-file-audit.json)
> tracks the current review, fixes and unresolved follow-ups.

**Repository:** `~/Documents/Kredit.com` · branch `main` · HEAD `247b36d` plus an uncommitted working tree (28 modified, 11 untracked)
**Auditor's brief:** understand the platform completely, produce an audit plan, then audit every file by code. Standard applied: world-class in design, frontend, backend and content; not overwhelming for a low-literacy audience; free of generative-AI tells.
**Method:** every finding below was read out of the current working tree. Prior audit reports in `docs/testing/` were deliberately not used as evidence. Where a claim was checkable without running the product (compiled CSS/JS in `web/.vercel/output`, `git` state, cross-file greps) it was checked.
**Fixes:** none applied. This is the audit pass; corrections follow in a second pass.

---

## Part 0 — How to read this

Findings are numbered `K-nnn` and carry a severity:

| | Meaning |
|---|---|
| **S1 Critical** | Money, security, privacy or tenant isolation can go wrong; or a primary journey is broken. |
| **S2 High** | A stated product goal fails — the owner's named complaints, world-class design, or the low-literacy audience. |
| **S3 Medium** | Real defect with a workaround; inconsistency a careful user will notice. |
| **S4 Low** | Polish, dead code, maintainability. |

Every finding names `file:line` and states what is wrong, why it matters here, and the correction. Where a finding is an inference that could not be proved by reading alone, it is marked **[needs runtime proof]**.

---

## Part 1 — What Kredit actually is

### 1.1 The business

Kredit is Nigerian B2B trade-credit software for **KREDIT TECHNOLOGIES LIMITED** (RC 9834452, Maiduguri, Borno State). A **supplier** sells goods to a **buyer** on credit. Kredit records the agreement, carries the buyer's acceptance, holds delivery evidence, tracks the outstanding balance, and — where a bank mandate exists — collects repayment through **Mono** (direct-debit "sweep"). Kredit does not lend, does not hold the goods, and does not guarantee repayment.

Two facts govern every design decision and are the source of most findings in this report:

1. **It is a financial system of record.** An amount that is wrong, double-counted, or silently defaulted to zero is a real loss for a real trader.
2. **Its users are low-literacy Nigerian traders on cheap Android phones over intermittent mobile data.** Density, jargon, giant display type, tooltips, and any screen that needs interpretation are failures, not style choices.

There is a third fact the code keeps forgetting: **the owner operates alone.** Anything that assumes a second approver, a support team, or an editorial team is fiction.

### 1.2 The stack

| Layer | Technology | Size |
|---|---|---|
| API + workers | Go 1.26.8, `net/http` `ServeMux` (Go 1.22 routing), pgx v5, River (jobs), goose (migrations), sqlc, OpenTelemetry | 41,351 lines non-test |
| Go tests | in-package + `tests/` | 21,538 lines |
| Web | SvelteKit 2.70 / Svelte 5 (runes), TypeScript 5.9, Vite 7, adapter-vercel (default) or adapter-node | 8,310 lines hand-written (+18,567 generated `schema.ts`/`schema.d.ts`) |
| Database | PostgreSQL, 90 migrations, RLS + restricted runtime roles | 108 `.sql` files |
| Infra | Docker, Terraform, Prometheus rules, OTel collector | |

**Binaries:** `cmd/api`, `cmd/worker`, `cmd/migrate`, `cmd/seed`, `cmd/reconcile`, `cmd/configcheck`, `cmd/bootstrap-owner`, `cmd/provider-simulator`.

### 1.3 Shape of the system

```
Browser
  └─ SvelteKit (Node/Vercel)
       ├─ hooks.server.ts : security headers, cookie gate, same-origin /api proxy
       └─ routes/ : public site · /app supplier · /buyer · /admin owner console
                          │
                          ▼  (same-origin, cookie auth)
                    Go API  internal/web/server.go
                      middleware: requestContext → panicRecovery → securityHeaders
                                  → rateLimit → idempotency → bodyLimit → adminSurfaces
                      ~190 routes over 4 audiences
                          │
        ┌─────────────────┼──────────────────────────┐
        ▼                 ▼                          ▼
  internal/* domains   River jobs (cmd/worker)   providers/mono
  (credit, ledger,     collections, notices,     mandates, sweep,
   payments, schedules, reminders, outbox        webhooks
   collections, disputes,
   tradelines, platformsettings…)
        │
        ▼
   PostgreSQL — RLS, restricted app/worker roles, append-only audit,
   outbox, idempotency, financial monitoring
```

### 1.4 The four surfaces

| Surface | Route prefix | Audience | Intended priority |
|---|---|---|---|
| Public site | `/`, `/pricing`, `/how-it-works`, `/legal/*`, `/blog/*`, `/demo` | prospects | proposition, proof, a route into the product |
| Supplier workspace | `/app/*` | the trader | what is owed, what is late, what needs a decision |
| Buyer journey | `/buyer/*`, `/c/*`, `/pay/*`, `/receipt/*`, `/buyer-invitations/*` | the customer | who, what, how much, when, one next action |
| Owner console | `/admin/*` | the founder alone | attention queue, operating controls |

### 1.5 The five things that shape the audit

1. **Two-and-a-half competing stylesheets.** `app.css` (global + page-specific + an `!important` "product finishing layer"), `product-ui.css` (a second global override layer), and per-component `<style>` blocks. They contradict each other, and the conflicts are silent. §3.1.
2. **A component library nothing imports.** 25 of 48 components in `web/src/lib/components/` are dead. They are precisely the *domain* components — status, money input, dates, fee breakdown, schedule table, timeline. Routes hand-roll instead. §3.2.
3. **A generic list page doing the work of eight screens.** `WorkspacePage.svelte` guesses at field names across disputes, payments, obligations and trade lines, and falls back to printing a UUID. §3.3.
4. **A Go backend that is markedly better than the frontend it serves.** Idempotency, append-only audit, RLS, tenant context, fail-closed rate limiting are real. The gap between backend rigour and frontend rigour is the single largest quality inconsistency in the repository.
5. **The owner's own named complaints are still present in source** — the disclaimer paragraph, the session-check wall, the workspace-as-marketing headline. §7.

---

## Part 2 — The audit plan

### 2.1 Scope: every file

| Area | Files | Coverage |
|---|---|---|
| A. Frontend foundation | `app.css`, `product-ui.css`, `app.html`, `hooks.server.ts`, `service-worker.ts`, `svelte.config.js`, `vite.config.ts`, `tsconfig.json`, `playwright.config.ts` | 9 |
| B. Frontend shared code | `web/src/lib/*.ts`, `lib/api/*`, `lib/server/*`, `lib/blog/*` | 18 |
| C. Components | `web/src/lib/components/*.svelte` | 48 |
| D. Routes | `web/src/routes/**` | ~105 |
| E. Go entry points | `cmd/**` | 15 |
| F. Go domain | `internal/**` non-web | ~150 |
| G. Go HTTP layer | `internal/web/*.go` | 40 |
| H. Database | `db/migrations/*.sql`, `db/queries/*.sql`, `db/generated/*.go`, `db/seeds`, `infra/postgres/*.sql` | ~120 |
| I. API contract | `api/openapi.yaml`, `api/generated`, `redocly.yaml` | 3 |
| J. Infra & scripts | `infra/**`, `scripts/**`, `docker-compose.yml`, `Taskfile.yml`, `.github/**` | ~70 |
| K. Tests | `**/*_test.go`, `web/tests/**`, `tests/**` | ~120 |
| L. Docs & content | `docs/**`, `README.md`, `CHANGELOG.md`, `*.md` | ~95 |

### 2.2 Dimensions applied to each file

Every file is read against the dimensions that apply to it:

**D1 Correctness** — does it do what it claims? Boundaries, nulls, races, error paths, dead branches.
**D2 Money** — exact integers end to end; no float; no failed read rendered as zero; rounding stated and matched across Go/TS/SQL; overflow bounded.
**D3 Security & privacy** — authz at every boundary; tenant isolation; secrets; injection; no data in browser storage; no leaked internals in responses, logs or bundles.
**D4 Concurrency & idempotency** — one logical effect per user intent under retry, replay, restart and out-of-order delivery.
**D5 Design quality** — hierarchy, density, typography, colour discipline, one component per job, no template tells.
**D6 Low-literacy usability** — is the screen readable to a trader who reads slowly? One idea per screen; nothing needing interpretation; the amount always visible.
**D7 Accessibility** — WCAG 2.2 AA: names, roles, keyboard, focus, contrast, live regions, zoom, reflow.
**D8 Copy** — precise transactional meaning; adult register; no meta-commentary; no defensive walls; consistent terminology.
**D9 State** — loading / empty / ready / stale / unavailable distinguished; obsolete responses discarded; identity-scoped caches.
**D10 Consistency** — does this file agree with the rest of the system, or invent its own version?
**D11 Maintainability** — dead code, duplication, minified source, magic values, absent types.
**D12 Testability** — is the claim covered by a test that would fail if the behaviour regressed?

### 2.3 Order of work

1. Frontend foundation and shared code — because a defect here multiplies across every screen.
2. Components — establishes what the design system actually is.
3. Routes — the four surfaces, in the order a user meets them.
4. Go HTTP layer — the authorisation and error-contract boundary.
5. Go domain — money, state machines, providers, workers.
6. Database — schema, constraints, RLS, migration safety.
7. Infra, scripts, tests, docs.
8. Verification pass: re-check every finding against source, discard what does not survive.

### 2.4 Cross-cutting sweeps

Run across the whole tree rather than per file:

- **Copy sweep** — every user-visible string, against D6 and D8.
- **Money sweep** — every arithmetic operation on an amount, in every language.
- **Authorisation sweep** — every route against its handler's permission check.
- **Contract sweep** — `openapi.yaml` against the Go handlers and against the TypeScript the frontend actually calls.
- **Consumer sweep** — every runtime setting from console control to the code that reads it.
- **Dead-code sweep** — unimported components, unreferenced CSS, unreachable routes, unused config.
- **Build-output sweep** — what actually ships in `web/.vercel/output` and `web/build`, since that is what users receive.

### 2.5 What this audit does not establish

Stated plainly so nothing here is over-read:

- No test suite was executed and no server was started. Runtime behaviour is inferred from source and from compiled output. Findings needing execution are marked **[needs runtime proof]**.
- No contrast ratio was measured against a rendered pixel; colour findings are computed from the declared hex values.
- No screen-reader, real-device or assistive-technology testing was performed.
- No provider call, live or sandbox, was made.
- Performance findings are structural (what the code must do), not measured.

---

## Part 3 — Frontend: the design system

### 3.1 Three stylesheets own the same property, and the wrong one wins

`web/src/app.css` (426 lines) is tokens + page-specific rules + a retrofitted `!important` block. `web/src/product-ui.css` (56 lines) is a second global layer. Every component then adds its own `<style>`. Svelte 5 scopes component styles with `:where(.svelte-hash)` — **zero added specificity** — so a component rule like `.messages h1` compiles to `.messages.svelte-x h1:where(.svelte-x)` = (0,2,1), while `:root body .product-route h1` in product-ui.css = (0,2,2) and wins.

**K-001 · S2 · The workspace `h1` has no single owner.**
Three files each supply a different property of the same heading:

| Property | Winner | Value |
|---|---|---|
| `font-family` | `app.css:355` `!important` | `Georgia, serif` |
| `font-weight` | `app.css:356` `!important` | `500` |
| `letter-spacing` | `app.css:357` `!important` | `-.05em` |
| `font-size` | `product-ui.css:9` (higher specificity) | `clamp(1.75rem, 3.6vw, 2.45rem)` |
| `line-height` | `product-ui.css:9` | `1.2` |
| the component's own size | **discarded** | e.g. `clamp(3rem, 6vw, 5.4rem)` |

The result is a Georgia serif heading at 28–39px carrying `letter-spacing: -.05em` — negative tracking tuned for the 86px display size the component author intended and product-ui.css cancelled. Nobody designed the thing that renders. Affects at minimum `WorkspacePage`, `NotificationHistory`, `app/payments`, `app/reports`, `app/search`, `buyer/+page`, `admin/+page`, `admin/mono`, `app/customers/new`.

**K-002 · S3 · `product-ui.css`'s radius tokens are dead on arrival.**
`product-ui.css:3-4` defines `--radius-control: .4rem` and `--radius-panel: .65rem` and applies them at lines 14, 15, 17, 22. `app.css:383` and `app.css:362-381` set `border-radius: 0 !important` on the same selectors. `!important` beats any specificity, so **both tokens never take effect anywhere**. The same `!important` also silently kills `border-radius` written in `DisputeDetail.svelte` (`.summary article{border-radius:1rem}`), `app/customers/new` (`.7rem`), `app/customers/[id]` (`1rem`), `admin/settings` (`1rem`), `admin/diagnostics` (`1rem`), `admin/controls` (`.6rem`). Six route files style a corner radius that cannot render.

**K-003 · S3 · Tokens exist and are bypassed.**
`app.css:7-31` declares a token set, then the same file hardcodes `#17181b`, `#2738d6`, `#606862`, `#f58a6d`, `#e7e5df`, `#a9a8a3`, `#8f8e8a`, `#cfd4ff`, `#3e3f43`, `#aeb4ff` in the rules below it. Route files add off-palette near-misses: `#202d9e` and `#233398` (pricing) beside `--color-primary: #2738d6`; `#e85f3d`, `#ec6a47`, `#ff6848`, `#ff8b70`, `#f58a6d`, `#ffa184`, `#ffb8a7` — **seven coral values** against one `--color-accent: #ff5b3a`; `#153f2f` (admin/mono) and `#126542` and `#135d3f` beside one `--color-positive`. `admin/settings` abandons the palette entirely for `#ccc`, `#bbb`, `#ddd`, `#555`, `#222`, `#9c2525`, `#fff7da`.

**K-004 · S4 · Tokens named for values they no longer hold.**
`--radius-lg: 0`, `--shadow-sm: none` (`app.css:22-23`). Consumers read as design intent and produce nothing: `CommandPalette` `border-radius: var(--radius-lg)`, `+error.svelte` `box-shadow: var(--shadow-sm)`, `WorkspacePage` `.empty-state{border-radius:var(--radius-lg)}`.

**K-005 · S3 · `--font-sans` is referenced but never defined.**
`DocumentLayout.svelte:26` sets `article :global(h2){font: 650 1.4rem/1.35 var(--font-sans, Arial, sans-serif)}`. A repository-wide grep finds no definition. **Every `h2` in the terms, privacy and complaints documents renders in Arial** while the rest of the site is Inter.

**K-006 · S3 · Fossils of a reverted design.**
`transform: none` written explicitly on `.agreement-window`, `.floating-receipt`, `.phone`; `box-shadow: none` on `.button-dark`, `.button-coral`, `.primary-button`; `.product-glow{opacity:1}` on a solid rectangle called a glow; `.button-coral{background:#2738d6}` — a class named coral that paints blue. These are the residue of a previous visual direction partly undone in place, which is what produced the `!important` layer.

**K-007 · S3 · Three dialog systems, three backdrops.**
`CommandPalette` `rgb(16 45 42 / .55)` (dark green), `ProtectedActionDialog` `rgb(23 24 27 / .68)` + blur, `PortalNav` `rgb(23 24 27 / .48)`, `PaymentReview` `rgb(23 24 27 / .55)`, and `admin/platform-settings` a hand-rolled `div.modal-overlay` with no backdrop element at all. No dialog primitive exists.

**K-008 · S3 · Two elevation languages.**
`WorkspacePage` `.records article` and `app/payments` `.claim-list article` use `box-shadow: 6px 6px 0 #ded8cc`; the homepage `.story-canvas` uses `18px 18px 0 #2738d6`; `product-ui.css:17` sets `box-shadow: none` for `.card`. Hard-offset shadows and no shadow coexist on adjacent screens.

### 3.2 Half the component library is dead

**K-009 · S2 · 25 of 48 components in `web/src/lib/components/` are never imported.**
Verified by grepping every `components/<Name>.svelte` import path across `web/src`:

`AgreementSummary`, `AuditTimeline`, `BusinessVerificationCard`, `CollectionAttemptCard`, `ConfirmFinancialAction`, `CustomerIdentityCard`, `DateTime`, `DisputePanel`, `DocumentViewer`, `DueDate`, `EmptyState`, `FeeBreakdown`, `InlineError`, `MandateStatus`, `MoneyInput`, `OutstandingBalance`, `PaymentBreakdown`, `Percentage`, `ReferenceCode`, `RiskFact`, `ScheduleTable`, `StatusPill`, `StepUpAuthDialog`, `Timeline`, `TradeLineMeter`.

These are precisely the **domain** components — status, money entry, dates, references, fee and payment breakdowns, schedules, timelines, meters. Routes hand-roll each of them instead, which is the direct cause of every inconsistency in §3.4 and §4. `app.css` carries matching dead CSS (`.status-*`, `.risk-fact`, `.timeline`, `.breakdown`, `.meter`).

**K-010 · S3 · `StepUpAuthDialog.svelte` is a non-functional stub.** It renders `<dialog open>` (non-modal), an unbound `<input>`, and a Cancel button. There is no submit, no verification call, no state. It is a picture of an MFA prompt. Unused today; it must be deleted, not left where someone can wire it up.

**K-011 · S3 · `StatusPill.svelte` would mostly render grey anyway.**
`app.css:328-331` styles exactly nine status classes (`active, paid, verified, success, overdue, failed, rejected, disputed, review`). `product-language.ts` maps about fifty states. Every other state — `READY_TO_RELEASE`, `BUYER_REVIEWING`, `GOODS_RELEASED`, `PENDING`, `RECOGNIZED`, `SUBMITTED`, `UNKNOWN` — falls back to neutral grey. `WorkspacePage:59` sidesteps the component entirely and emits bare `class="status"`, so **every status in every generic list is grey.**

### 3.3 One generic component does the work of eight screens

`WorkspacePage.svelte` backs `/app/customers`, `/app/collections`, `/app/disputes`, `/app/overdue`, `/admin/audit`, `/buyer/disputes`, `/buyer/requests`, `/buyer/obligations`.

**K-012 · S2 · It guesses at field names and prints a UUID when it fails.**
`WorkspacePage.svelte:59` — title is `record.buyer_legal_name ?? record.legal_name ?? record.reason ?? record.reference ?? record.provider ?? record.id ?? 'Item'`; body is `record.goods_description ?? record.explanation ?? record.subject_type ?? record.description ?? 'Open this item to see more.'`.

**K-013 · S1 · The audit trail is unusable.**
`admin/audit/+page.svelte` passes `endpoint="/api/v1/ops/audit" collectionKey="events" showDetails={false}`. Audit events carry `actor_user_id`, `action`, `resource_type`, `resource_id`, `outcome`, `occurred_at` — **none** of which appear in either fallback chain. Every row therefore renders as:

- title: a raw UUID
- status pill: `Open` (from the `?? 'OPEN'` default)
- body: `Open this item to see more.`
- no link (`showDetails={false}`)

The page's own `description` — *"Trace actor, purpose, resource, outcome and request correlation"* — names five fields the component cannot display. This is the platform's compliance surface.

**K-014 · S2 · The page descriptions promise data the component cannot show.**
`/app/customers` says *"For each customer: what they owe you, and how well they have paid you before."* The rendered card shows a name, a grey pill, and `'Open this item to see more.'` — no balance, no history. Same contradiction on `/app/collections` and `/app/overdue`.

**K-015 · S3 · Search is `JSON.stringify` of the whole record.**
`WorkspacePage.svelte:8` — `records.filter(r => JSON.stringify(r).toLowerCase().includes(query.toLowerCase()))`. It matches internal IDs, enum names, timestamps and hidden fields, re-stringifies every record on every keystroke, and produces results the user cannot explain.

**K-016 · S3 · Client-side pagination over an unbounded fetch.** `pageSize = 9`, `filtered.slice(...)`. The whole collection is downloaded first. `/app/search` is worse: it loads **all** customers, credit requests and payments on mount and filters in memory.

**K-017 · S2 · Marketing bullets sit above the data on four workspace screens.**
The `tips` prop renders a three-column ✓ grid above the records:
`['We do not debit twice','A reported problem stops debit','We check slow payments']` · `['Extra days are respected','Bank problems are shown','Every action is saved']` · `['Only the problem amount stops','Photos and papers stay here','Both sides can see the answer']` · `['Customer details checked','Past payments with you','Your notes stay private']` · `['Purpose-aware access','Request correlation','Privacy-safe projections']`.

Two problems: it is marketing prose occupying the supplier's first screen, and *"We do not debit twice"* is an unqualified engineering guarantee stated as decoration.

**K-018 · S3 · The empty state is a full-bleed saturated blue block** (`.empty-state{padding:clamp(2rem,6vw,5rem);color:#fff;background:#2738d6}`) with a 3.5rem serif heading and `◎` at 3rem as an icon. "Nothing here yet" is given more visual weight than any real record.

**K-019 · S3 · Two org switchers, opposite treatments.** `WorkspacePage`'s `.toolbar` is a black `#17181b` bar; `app/overview`'s `.toolbar` is `var(--color-surface)`. Same element name, same job, adjacent screens.

### 3.4 Four ways to call the API

**K-020 · S2 · 124 raw `fetch(` call sites; the safe write path is used in six files.**

The repository contains a genuinely excellent mutation layer — `lib/api/mutation.ts` — which persists a random request identity plus a SHA-256 payload fingerprint in `sessionStorage`, refuses to reuse a key for a different payload, refuses to silently expire an unresolved request, and returns `not_sent | rejected | unknown`. It is used by exactly six files: `app/credit/quick`, `app/credit/new`, `app/credit/[id]`, `app/payments`, `app/overview`, `buyer/credit-requests/[requestID]`.

Everywhere else a mutation is `fetch(..., {'Idempotency-Key': idempotencyKey()})`, where `idempotencyKey()` (`lib/api/client.ts:67`) returns a **fresh `crypto.randomUUID()` on every call**. A retry after a timeout therefore arrives at the server as a *different* operation. Affected mutations include: buyer-invitation creation, team-role grant and revoke, admin financial-change proposals and decisions, business-policy proposals, protected commands (`suspend_user`, `retry_collection`, `cancel_collection`), dispute decisions, privacy and recovery reviews, correction decisions, support-case transitions, notification preferences, settlement and billing updates, report exports.

**K-021 · S1 · `admin/controls` applies a protected command with no re-entrancy guard.**
`admin/controls/+page.svelte:apply()` — the "Apply this change" button has no `disabled` binding and `apply()` has no `busy` check. Two clicks issue two requests with two different `Idempotency-Key` values. The command list includes `retry_collection` and `cancel_collection`. Two clicks on *Retry a bank debit* are two debit requests.

**K-022 · S3 · Four parallel client abstractions.**
`lib/api/client.ts` (`readJSON`, `csrfHeaders`, `idempotencyKey`), `lib/api/reliable.ts` (`checkedJSON`, `csrfHeader`, `randomKey`), `lib/admin-client.ts` (`adminGet`, `adminPost`), and bare `fetch`. `csrfHeaders()` and `csrfHeader()` are near-identical duplicates with different names and different return types.

**K-023 · S4 · The typed OpenAPI client is used once.**
`openapi-fetch` is a dependency; `web/src/lib/api/generated/schema.ts` + `schema.d.ts` total **18,567 lines**, regenerated by `predev`, `prebuild`, `pretest` and `prelint`. The exported `api` client appears in exactly one call: `buyer/history/+page.svelte:13`.

**K-024 · S3 · `admin-client.ts` cannot survive a non-JSON response.**
`adminGet` and `adminPost` call `await r.json()` unguarded. A proxy's HTML 502 page throws `SyntaxError`, which `admin/approvals`, `admin/attention`, `admin/inbox` and `admin/history` render to the operator as `error = String(e)` → `SyntaxError: Unexpected token '<'…`.

**K-025 · S3 · `MutationIntent` can lock a form permanently with no recovery path.**
`mutation.ts:31,49` — a malformed stored intent sets `blocked = true`, and every later submission on that URL scope throws *"An earlier request could not be verified. Check the record or contact support before submitting another."* Nothing in the UI clears it; the user must clear site data. Same for the 15-minute unresolved window (`mutation.ts:60`).

### 3.5 Money and time in the browser

**K-026 · S1 · Kobo divided as a JavaScript float in six places, while the exact formatter is imported in the same files.**

| File | Line | Expression |
|---|---|---|
| `lib/components/DisputeDetail.svelte` | 9 | `Intl…format((value ?? 0) / 100)` |
| `lib/components/DisputeDetail.svelte` | 10 | `remaining = String((dispute.remaining_disputed_kobo ?? 0) / 100)` |
| `routes/admin/disputes/[id]` | 6 | `money = (v) => Intl…format((v ?? 0) / 100)` |
| `routes/admin/disputes/[id]` | load | `remainingNaira = dispute.remaining_disputed_kobo / 100` |
| `routes/app/credit/[id]` | 58, 167, 113, 183 | `principal_kobo / 100`, `outstanding_kobo / 100`, `amount / 100`, `payment.amount_kobo / 100` |
| `routes/app/reports` | derived | `paidTotal`, `receivedRate`, `averagePayment`, `overdueShare` |
| `routes/admin/analytics` | `format` | `metric.value / 100` |

`lib/money.ts` provides `exactKobo` / `sumKobo` / `formatKobo` on `BigInt` for exactly this, and `admin/reconciliation` shows the correct pattern (`money(value:string)` with `BigInt`). The dispute pages are the worst case: `remainingNaira` is divided by 100 into a float, bound to a number input, then passed back through `parseNaira`, whose regex **rejects any value with more than two decimal places and returns `-1`** — which is then POSTed as the new disputed amount.

**K-027 · S1 · `?? 0` turns a failed financial read into zero money.**
The brief's rule — "a failed financial read must never become zero money" — is broken in:

- `routes/buyer/+page.svelte` — `if (requestsResponse.ok) requests = …`. When `/api/v1/buyer/credit-requests` fails, `requests` stays `[]`, and the buyer's home screen displays **"You owe ₦0.00"** with no error at all. The `/buyer/me` failure is handled; this one is not.
- `routes/buyer/+page.svelte` — `sumKobo(requests.map(i => i.obligation?.outstanding_kobo ?? 0))`: a sale whose obligation is missing contributes 0.
- `routes/app/reports` — `(summary?.voluntary_paid_kobo ?? 0) + (summary?.collected_paid_kobo ?? 0)`.
- `routes/app/customers/[id]` — `Number(history.on_time_percentage ?? 0).toFixed(0)` renders **"0% paid on time"** for a customer whose metric is simply absent. That misrepresents a named third party inside the supplier's own record.
- `routes/app/activity` — `if (!responses.some(r => r.ok)) error = …`. If **any one** of five endpoints succeeds, no error shows and the four failed sections render as confirmed-empty. A 500 on `audit-events` looks like a business with no activity.
- `routes/admin/recovery` and `routes/admin/privacy` — `items = response.ok ? … : []`, so a failed load of the account-recovery or privacy queue displays *"Nothing waiting for review."*

**K-028 · S2 · Financial dates and deadlines computed from the device clock.**
`records.ts` pins `Africa/Lagos` in `dateLabel`/`timeLabel`. These do not:

- `lib/datetime.ts:localDateTime` — builds the `datetime-local` value for **`collection_at`** (the earliest bank-debit time) from `date.getHours()`, i.e. the phone's timezone. Used by `app/credit/[id]:59`.
- `routes/buyer/+page.svelte` — `new Date(\`${due}T23:59:59\`)` and `new Date(\`${due}T12:00:00\`)` with no offset; the buyer's overdue count is decided by the phone.
- `routes/app/credit/[id]` — `new Date(item.due_at).toLocaleDateString('en-NG')`, `new Date(claim.paid_at).toLocaleString('en-NG')`, `new Date(attempt.requested_at).toLocaleString('en-NG')` — device zone, on the same page where `timeLabel()` (Lagos) is imported and used for `collection_at`. **The schedule table and the debit line can disagree on the same screen.**
- `routes/app/reports:paymentSummary()` — "today" is `new Date()` with `setHours(0,0,0,0)`, i.e. the device's day, and that figure is then shared externally through `ShareActions`.
- `routes/admin/team`, `routes/admin/controls` — `new Date(expires).toISOString()` for role expiry and risk-hold expiry, while `admin/inbox` and `admin/approvals` use `lagosISO()` for the same kind of field.
- `routes/admin/analytics` — `new Date().toISOString().slice(0,10)` as the default report end date (UTC), and `toLocaleString()` with **no locale and no timezone** anywhere on the page.
- `routes/admin/{users,money,cases,disputes,inbox}` — `toLocaleString('en-NG')` without `timeZone`.

**K-029 · S3 · `parseNaira` rejects what the audience will type.**
`money.ts:5` accepts only `1234`, `1,234`, `1234.56`. It returns `-1` for `₦5,000`, `5 000`, `5.000` (dot as thousands separator), or a trailing `.`. On `/pricing` the consequence is visible mid-typing: `<Money amountKobo={principal > 0 ? principal : null} />` flips all four rows of the fee table to **"Amount unavailable"** while the user is still entering digits.

**K-030 · S3 · `formatKobo` always prints kobo.** `money.ts:29-30` appends `.${kobo}` unconditionally, so every amount on every screen reads `₦120,000.00`. Nigerian retail practice drops the kobo; two extra characters on every figure is noise for the audience the product targets.

**K-031 · S3 · `Money.svelte` hides the exact amount in a `title` attribute.** `Money.svelte:8` — `<span title={full}>` with the abbreviated form (`₦1.2M`) as the visible text when `abbreviated` is set. `title` is unreachable on touch and unreliable for keyboard and screen-reader users. There is no `aria-label`.

### 3.6 State, errors and stacked alerts

**K-032 · S2 · Seven alert boxes for one outage.**
`app/overview/+page.svelte:95` renders `ResourceNotice` in a loop over five resources, plus one for the balance (line 91) and one for businesses (line 73). A single API outage produces seven stacked *"… unavailable"* boxes. `app/credit/[id]` carries eight independent error/notice variables (`error`, `paymentError`, `scheduleError`, `claimError`, `collectionError`, `eligibilityError`, `notice`, `paymentNotice`) with the same effect.

**K-033 · S2 · `AuthGate` is the session-diagnostic wall, restyled.**
`lib/components/AuthGate.svelte` blocks every private route behind a client-side `GET /api/v1/me`, showing a full-page *"Checking your account…"* and, on any non-401 failure, *"We cannot open your account. The account check did not finish. Check your connection and try again."* with a **Try again** button. That is the pattern the owner named. A server-rendered auth state is already available — `hooks.server.ts:28` reads the `kredit_session` cookie and `app/+page.server.ts` passes `hasSessionCookie` — and is not used here.

**K-034 · S2 · A server outage is reported as the user's connection or the user's mistake.**

| File | Text | Shown when |
|---|---|---|
| `lib/api/reliable.ts:34` | "We could not check {subject}. **Check your connection and try again.**" | any 5xx |
| `lib/components/AuthGate.svelte:41` | "**Check your connection** and try again." | any non-401 |
| `routes/buyer/+page.svelte` | "**Open the private link your seller sent you.** That link is how you get into this account." | any `/buyer/me` failure, including 500 |
| `routes/app/activity` | "**You do not have permission** to open these business records." | all five endpoints failing, for any reason |
| `routes/app/reports` | "**Sign in to see your reports.**" | any `/organizations` failure |
| `lib/components/DisputeDetail.svelte:12` | "**Somebody changed this problem** before your decision saved." | any non-OK on decide, including 403 and 500 |

**K-035 · S3 · `ShareActions` reports every failure as a cancellation.** `ShareActions.svelte:9` — `catch { message = 'Sharing was cancelled.' }`. A permission denial or a clipboard failure is announced as the user's own choice.

**K-036 · S3 · Success and failure share one polite live region.** `admin/team/+page.svelte` assigns both outcomes to `message` and renders `<p class="notice" role="status">`. An error is announced politely and styled as a success notice.

**K-037 · S3 · Four minimum lengths for the same "reason" field.** 4 characters (`admin/platform-settings` edit modal, `admin/cases/[id]` note), 8 characters (`ProtectedActionDialog`, `admin/approvals`, `admin/settings`, `admin/reconciliation`, `admin/team`). Two of them are single-line `<input>`s where the rest are `<textarea>`s.

### 3.7 Privacy in browser storage

**K-038 · S2 · Private customer notes survive sign-out.**
`app/customers/[id]/+page.svelte` writes `kredit:customer-note:{orgId}:{customerId}` to **localStorage** via `product-tools.ts:writeLocal`. `reliable.ts:clearPrivateBrowserData()` clears only keys equal to `kredit:saved-sale-items`, starting with `kredit:sale-draft:`, or matching `/^kredit\.(quick-sale\.|intent\.|bank-return\.|account\.)/`. **`kredit:customer-note:` matches none of them.** The page tells the user *"This note never leaves this phone. Not even Kredit can see it."* — true, and it also never leaves after they sign out, on a device shared by several traders.

**K-039 · S3 · Saved sale items are not scoped to a user.**
`product-tools.ts:29` writes `kredit:saved-sale-items` — a global key holding goods descriptions and amounts, with no user or organisation in the name and no expiry. It *is* cleared by `clearPrivateBrowserData()`, but only on an explicit sign-out or a 401; closing the browser leaves it for the next person. `sale-drafts.ts` in the same codebase does this correctly (`kredit.quick-sale.v3:{user}:{org}` with a 12-hour TTL and an opt-in checkbox), which shows the rule is understood and not applied here.

### 3.8 Accessibility

**K-040 · S2 · The focus indicator fails the contrast requirement.**
`app.css:19,89` — `--focus-ring: #ff5b3a`, applied as `outline: 3px solid var(--focus-ring)`. Against the page background `#fafaf8`, the computed contrast is **2.96:1**, below the 3:1 that WCAG 2.2 SC 1.4.11 requires for a focus indicator. It is the single most-used interactive affordance in the product.

**K-041 · S2 · The buyer's overdue strip fails text contrast.**
`buyer/+page.svelte` — `.due-strip.late{background:#ec6a47;color:#fff}` at `font-size:.9rem`. Computed contrast **3.12:1**, against a 4.5:1 requirement (14.4px is not "large text"). This is the notice telling a customer a payment day has passed.

**K-042 · S2 · 8–11px text throughout the homepage.**
`routes/+page.svelte` — `.chat small{font-size:.5rem}` (8px, contrast 3.25:1 — also failing), `.chat-date{.52rem}`, `.status-chip{.62rem}`, `.event small{.67rem}`, `.window-bottom{.67rem}`, `.sector-inner p{.68rem}`, `.assurances{.74rem}`. The hero mockup alone contains roughly twenty text elements at these sizes. For a low-literacy reader on a 360px phone this is not readable at any level.

**K-043 · S2 · The homepage hero mockup is read aloud as if it were real data.**
`routes/+page.svelte:37` — `<div class="hero-product" aria-label="Example Kredit credit sale, not a live account">`. `aria-label` on a `div` with no role is ignored. The subtree is not `aria-hidden`, so a screen reader announces "Adebayo Stores… Money left to pay ₦800,000.00… Deal accepted… Payment in progress" as the user's own account.

**K-044 · S2 · `role="progressbar"` announcing kobo.**
`routes/+page.svelte:42` — `aria-valuemin={0} aria-valuemax={120000000} aria-valuenow={40000000}` with no `aria-valuetext`. Announced as "40000000 of 120000000".

**K-045 · S2 · The story switcher is a tab set built from toggle buttons.**
`routes/+page.svelte:71` — three `<button aria-pressed>` swap a panel with no `role="tablist"/"tab"/"tabpanel"`, no `aria-controls`, no arrow-key navigation and no live region. Keyboard and screen-reader users get no indication that the content changed.

**K-046 · S2 · `CommandPalette` uses an invalid listbox.**
`CommandPalette.svelte:82-93` — `<ul role="listbox">` containing `<li>` wrappers around `<button role="option">`. `listbox` requires `option` children; `option` must not contain interactive elements. The input is not a `combobox` and there is no `aria-activedescendant`, so arrow-key selection is silent.

**K-047 · S2 · The platform-settings modal is a hand-rolled overlay.**
`admin/platform-settings/+page.svelte` — `<div class="modal-overlay" role="dialog" aria-modal="true">` with no focus trap, no initial focus, no Escape handler, no focus restoration, no `aria-labelledby` and no scroll lock. It is the dialog that edits secrets and transfers platform ownership, and it is the least accessible dialog in the product, while `<dialog>`-based modals exist elsewhere in the same codebase.

**K-048 · S3 · `aria-controls` points at an element that does not exist.**
`PortalNav.svelte:124,130` set `aria-controls={moreID}`, but the `<dialog id={moreID}>` is inside `{#if moreOpen}` (line 146) and absent while closed.

**K-049 · S3 · ARIA table semantics that break on mobile.**
`app/payments/+page.svelte:84` builds `role="table"/"row"/"columnheader"/"cell"` over CSS grid, then at ≤800px sets `.table-head{display:none}` and reorders cells with `grid-column`/`grid-row`. The result is a table whose column headers are removed from the accessibility tree and whose cell order no longer matches the header order.

**K-050 · S3 · `Skeleton` announces itself twice.** `Skeleton.svelte:5,9` — `role="status" aria-label="Loading"` plus `<span class="sr-only">Loading…</span>`.

**K-051 · S3 · `aria-label` on bare `<span>`s.** `StatusPill.svelte:2` (`aria-label={\`Status: ${label}\`}`) and `routes/+page.svelte:37`. `aria-label` is not valid on elements with no role and is ignored by several screen readers.

**K-052 · S3 · Private routes have no `<main>` landmark from the layout.** `routes/+layout.svelte:119` renders private children inside `<div id="main-content" tabindex="-1">`; the public branch does the same. Whether a `<main>` exists depends on each child page, and several (`app/credit/[id]` uses `<section class="page-shell">`) do not provide one.

**K-053 · S3 · `HomeProof` renders outside `<main>`.** `routes/+layout.svelte:124` places it after `{@render children()}`, so a quarter of the homepage's content sits outside the main landmark.

**K-054 · S3 · Desktop TOC links are below the stated touch target.** `DocumentLayout.svelte:26` — `nav a{padding:.55rem .7rem; font-size:.88rem}` ≈ 36px on desktop; the 44px minimum is applied only under `@media(max-width:800px)`.

**K-055 · S3 · The document TOC's active indicator will jump.** `DocumentLayout.svelte:8-10` — the `IntersectionObserver` callback assigns `active = entry.target.id` for every intersecting entry in the batch, so the last entry in the array wins rather than the topmost visible section.

**K-056 · S3 · Playwright covers one engine and blocks the service worker.** `web/playwright.config.ts:22` — a single `chromium` project; `serviceWorkers: 'block'`; `retries: 0`; `workers: 1`. Firefox and WebKit are never run, and the service worker is never exercised.

### 3.9 The Svelte template bug that breaks sign-in

**K-057 · S1 · `pattern="[0-9]{6}"` compiles to `pattern="[0-9]6"` and blocks the OTP form.**

In a Svelte template, `{…}` inside an attribute value is an expression. `pattern="[0-9]{6}"` is parsed as the literal `[0-9]` concatenated with the expression `6`.

Confirmed in the shipped bundles, not inferred:

```
web/.vercel/output/static/_app/immutable/nodes/37.CayzAqnU.js   R(h,"pattern","[0-9]6")   ← routes/app/+page.svelte  (sign-in)
web/.vercel/output/static/_app/immutable/nodes/51.CU9i9VE5.js   B(se,"pattern","[0-9]6")  ← routes/app/onboarding    (contact verification)
web/.vercel/output/static/_app/immutable/chunks/CuN__SYc.js     q(p,"pattern","[0-9]6")   ← lib/components/VerifyIdentity (MFA step-up)
```

The static template emits the input **without** `pattern`; Svelte sets it at hydration.

Consequences:

- **`routes/app/+page.svelte:71-76`** — the input has `required`, `minlength="6"`, `maxlength="6"` and now `pattern="[0-9]6"`. The submit control is `<button class="primary">` with no `type`, so it defaults to `type="submit"` inside `<form onsubmit=…>`. Native constraint validation runs before the submit event. `minlength="6"` and `^(?:[0-9]6)$` **cannot both be satisfied**. After hydration, no six-digit code can be submitted, and the browser shows its own "match the requested format" bubble instead of the app's error handling. **The primary sign-in journey is broken.**
- **`lib/components/VerifyIdentity.svelte:6`** — same shape inside `<form onsubmit=…>` with `required`. The MFA step-up used by `/admin/settings`, `/admin/approvals`, `/admin/inbox` and `/admin/platform-settings` cannot be submitted.
- `routes/app/onboarding` is not inside a `<form>`, so it is functionally unaffected — the attribute is merely wrong there.

**Why no test caught it.** `web/tests/audit-product-journeys.spec.ts:97` asserts only that the code textbox is *visible*. `web/tests/real-stack-financial.spec.ts:8,16` authenticates with `page.request.post('/api/v1/auth/otp/challenges')` and `…/verify` — it calls the API directly and never submits the form. There is no UI-level sign-in test in the suite.

### 3.10 Copy, register and generative-AI tells

**K-058 · S2 · The disclaimer the owner rejected is still on the public site, in three places.**

1. `lib/components/HomeProof.svelte:8` — card **04 of 4** on the homepage: *"Kredit does not choose your customer. Kredit does not lend money and cannot promise you will be paid. You decide who takes your goods on credit."* A defensive disclaimer given identical visual weight to three product benefits, in the homepage's main proof section.
2. `routes/+page.svelte:34` — `<span>We do not lend money</span>` in the `.assurances` row, rendered by `.assurances span::before{content:'✓'}` as a tick beside "No monthly fee" and "Free until a sale starts". A disclaimer dressed as a product benefit.

The brief says the paragraph "must not survive as public product copy" and specifically forbids shortening it and keeping the same negative statement everywhere.

**In fairness, the third instance is correct.** `lib/financial-copy.ts:2` — `creditBoundary = 'You provide the goods. Kredit does not lend money or guarantee repayment.'` — is rendered on `/pricing` inside `<details><summary>Does Kredit lend money?</summary>`, which is exactly the placement the brief asks for: a material role provision answered where someone asks the question, not plastered beneath every feature. Keep that one; remove the two above.

**K-059 · S2 · `admin/platform-settings` is written in a different voice from the rest of the product.**

- Title Case throughout — *Platform Settings & Launch Registry*, *All Settings*, *Launch & Visibility*, *Features & Switches*, *Integrations & Credentials*, *Security & Auth*, *Change Mode*, *Transfer Ownership*, *Rotate Secret*, *Edit Value*, *Apply Setting Change* — against sentence case everywhere else.
- The subhead: *"Comprehensive database-backed configuration for launch stages, feature gates, risk parameters, integration secrets, and operational policies. All mutations require step-up MFA, are strictly typed, and maintain an immutable versioned audit history."* Two comma-lists in two sentences, the page describing its own storage layer, and "mutations / strictly typed / immutable versioned audit history" offered to a Nigerian founder as product copy.
- The permanent banner: *"Single-Owner Autonomous Mode Active — Platform Owner is permitted to execute self-approvals on administrative and policy changes with explicit confirmation and mandatory audit justifications. **Maker-checker** is maintained for all non-owner roles."* and *"Strict **four-eyes** separation enforced across all administrative and policy **mutations**."* Two unexplained audit-industry idioms, styled as a warning, shown on every visit.
- Six `'Failed to …'` error strings, a register that appears nowhere else in the product ("We could not …").
- ASCII `...` where the rest of the app uses `…`.
- Three names for one page: `<h1>` *Platform Settings & Launch Registry*, `<title>` *Platform Settings & Launch Controls*, eyebrow *Super Admin / Platform Controls*.

**K-060 · S2 · The console tells the solo founder he needs a second person, then offers a self-approve button three lines below.**

- `admin/settings/+page.svelte` — *"Every change needs approval from a second platform administrator."*, then `Self-Approve (Solo Owner)`.
- `admin/approvals/+page.svelte` — *"Every correction needs a second person to approve it."*, then `Self-Approve (Solo Owner)`.
- `app/credit/[id]:183` — *"Changes of ₦10,000 or more need a second person to approve them first."* The server (`internal/operations/postgres.go:106`) does not offer a second-person path from that screen at all: it refuses with *"operation reaches the configured approval threshold"* and the correction must be re-proposed in the admin console. The threshold is also `min(₦10,000, policy.CorrectionThreshold)`, so the hardcoded figure in the UI is wrong whenever the policy is lower.

**K-061 · S2 · Roles and privacy requests are labelled with sentences.**
`lib/product-language.ts:14-18` — the role `SALES` renders as **"Add and manage sales"**, `FINANCE` as **"Manage payments"**, `COLLECTIONS` as **"Chase late payments"**, `ADMINISTRATOR` as **"Manage the account"**, `VIEWER` as **"Can look, cannot change"**. Privacy request types render in the requester's first person inside an admin table: `ACCESS` → **"Show me my information"**, `DELETION` → **"Delete my information"**. Two states render as full sentences used as status pills: `CONTESTED_ONLY` → **"Only the money in question is on hold"**, `FULL_BLOCK` → **"All bank debits are on hold"**.

**K-062 · S2 · Imprecise transactional language.**
`product-language.ts` — `FAILED` → **"Did not work"** (for a bank debit), `REJECTED` → **"Turned down"**, `BUYER_PAYMENT_CLAIM` → **"Customer says they paid"** (the very construction the brief names as bad), `UPHELD` → **"Customer was right"**, `PARTIALLY_UPHELD` → **"Customer was partly right"** — a formal dispute finding phrased as a playground verdict. `COOLING_OFF` → **"Short safety wait"**, an invented term.

**K-063 · S3 · Three different vocabularies for the same states.**
`product-language.ts` says `SUPPLIER_RECORDED_TRANSFER` → *"Bank transfer recorded"*; `app/payments/+page.svelte:28` says `supplier_recorded_transfer` → *"Bank transfer"*; `NotificationHistory.svelte:7-14` keeps a third map. `app/payments:29` maps unknown states to *"Status unavailable"* while `productLabel` maps them to a sentence-cased raw enum.

**K-064 · S2 · Raw enum values shown to users.**
The admin console's universal labelling strategy is `value.replaceAll('_',' ')` — `admin/users`, `admin/organizations`, `admin/money`, `admin/cases`, `admin/cases/[id]`, `admin/disputes`, `admin/disputes/[id]`, `admin/inbox`, `admin/history`, `admin/search`, `admin/attention`, `admin/+page`. `productLabel` is imported by none of them. `app/search/+page.svelte` renders `sale.state` with `text-transform: capitalize`, producing **"Buyer_reviewing"**.

**K-065 · S3 · Patronising register in shipped metadata and copy.**
`lib/seo.ts:66` — the `/glossary` meta description ends *"No big grammar."*, which is what Google will show. `WorkspacePage` empty states: *"Nobody is late. Well done."* and *"Good news. A late payment will show here."* `admin/settings` describes settings as *"Business settings"* while `admin/platform-settings` calls the same concept a *"Registry"*.

**K-066 · S3 · `Kredit Editorial Team` is invented.**
`routes/+layout.svelte:58` — article JSON-LD sets `author: { '@type': 'Organization', name: 'Kredit Editorial Team' }`. The company is one person.

**K-067 · S3 · Absolute guarantees in customer copy.**
`DocumentUploader.svelte:10` — *"We check every file for viruses before anybody can open it."* `security/+page.svelte:17` — *"A receipt you forward carries no full names, no bank details and no private notes."* `WorkspacePage` tips — *"We do not debit twice."* Each is an unqualified claim about system behaviour presented to customers as fact.

**K-068 · S3 · Unnecessary section numbering, four systems on one page.**
`routes/+page.svelte` carries `01 — WHY KREDIT` / `02 — HOW IT WORKS` / `03 — ONE CLEAR RECORD` / `04 — ON WHATSAPP` (section eyebrows), `01/02/03` in the story navigation, `01/02/03` in the process grid, and `01/02/03` in the WhatsApp list. `security/+page.svelte` adds `0{index+1}` per safeguard plus `01 / LINKS YOU SHARE` and `02 / WHEN A DEBIT IS SLOW`. `HomeProof` adds `01–04`. `admin/+page` adds `01–06`.

**K-069 · S3 · Both primary homepage CTAs promise an action and deliver a login form.**
`routes/+page.svelte:31,97` — *"Add your first sale ↗"* links to `/app`, the sign-in page.

**K-070 · S3 · Engineering vocabulary as page copy.**
`admin/diagnostics` lede: *"Provider latency signals, webhook lag, queue age, reconciliation, drift, dead letters, notifications, scanning, mandates, and settlements—without raw payloads or unredacted correlation identifiers."* `security/+page.svelte` offers *"Amounts are recorded in kobo"* as a security safeguard to a trader. `admin/audit` tips: *"Purpose-aware access", "Request correlation", "Privacy-safe projections"*.

**K-071 · S3 · Developer hedging surfaced to the user.**
`app/overview` — *"No overdue balance in this **checked summary**."* and *"Some records have not been checked yet."* `app/payments` — *"Payments in this **checked record**."* `app/credit/[id]` — *"No scheduled payments are present in this **checked record**."*

**K-072 · S3 · The error page shows an HTTP status and a mismatched explanation.**
`routes/+error.svelte:17` renders `Problem {page.status}` as the eyebrow, and line 8 explains *"An earlier request may still be processing. Check its status before trying again."* — correct after a failed write, wrong for the common case of a failed page load. The 96px `<h1>` and the pill-shaped button (`border-radius:999px`, the only one in the product) complete the mismatch.

**K-073 · S4 · The footer carries no company identity.**
`lib/components/SiteFooter.svelte` — four link columns and *"Know who owes you. Know what they paid. Know when."* No registered name, no RC number, no address, no contact address. The accepted company facts are available in `lib/server/legal-publication.ts` and are rendered only inside the legal documents.

**K-074 · S4 · Four sources of truth for the company's own details.**
`legal-publication.ts` holds `entityName`, `serviceAddress`, `legalEmail`. `legal/terms/+page.svelte:31` hardcodes `(RC 9834452)` inline while reading the name and address from config. `legal/privacy/+page.svelte:21` hardcodes `hello@kredit.com.ng` instead of `data.legal.privacyEmail`. `legal/complaints/+page.svelte:7` hardcodes the entire company name, RC number and address, and has no `+page.server.ts`, so it cannot be republished at all.

### 3.11 Public site and legal documents

**K-075 · S2 · The public site is built on display type the workspace then contradicts.**
Four serif headlines near 100px on the homepage (`h1` to 6.15rem, `.belief h2` to 6.6rem, `.section-heading h2` to 6rem, `.whatsapp-copy h2` to 5.4rem); `security/+page.svelte` `h1` to **7.5rem** (120px); `pricing` `h1` to 6.25rem with two 5.5rem percentages beside it; `+error.svelte` `h1` to 6rem. The same treatment written into workspace components is then overridden to 2.45rem by `product-ui.css`, so the product has an expressive public voice and a silently truncated version of it inside the account — rather than two deliberate treatments.

**K-076 · S3 · Decoration the brief names explicitly.**
`.product-glow` — a 31×34rem solid blue rectangle with two inset outlines behind the hero card. `.floating-receipt` — a card overlapping the hero mockup at `right:-1.5rem`. `.phone-scene::before` — a 31×34rem coral rectangle behind a phone mockup. `.story-canvas{box-shadow:18px 18px 0 #2738d6}`. Five full-bleed colour blocks on the homepage.

**K-077 · S3 · A negative-margin positioning hack in the hero.**
`routes/+page.svelte:111` — `.demo-label{margin-left:-8rem}` inside a `justify-content:space-between` row, changing to `-3rem` at ≤700px. Between those widths the "Example sale" label has no defined relationship to the brand mark it sits beside. **[needs runtime proof]**

**K-078 · S3 · The pricing slider is half-wired.**
`pricing/+page.svelte:35` — the range input's `value` is computed but not bound, so it does not follow typed changes; its `oninput` overwrites the typed field; the accompanying note says *"The typed amount is used for the calculation"*, which contradicts what the control does. Its bounds (₦1,000–₦10,000,000) do not match the text field's (up to `MAX_SAFE_INTEGER` kobo).

**K-079 · S3 · Reassigning `$derived` state on the pricing page.**
`pricing/+page.svelte:8` declares `rates` and `pricingError` with `$derived`, and `loadRates()` assigns to both. Svelte 5.55 permits overriding a derived, but the override is discarded the next time `data.rates` changes. **[needs runtime proof]**

**K-080 · S3 · Public pages get a shared CDN cache; pricing correctly opts out, nothing else does.**
`hooks.server.ts:43-45` sets `public, max-age=0, s-maxage=300, stale-while-revalidate=86400` on every non-private route without its own header. `pricing/+page.server.ts:6` sets `no-store`. Any other page that later renders published policy — a fee reference in a guide, the legal documents themselves — inherits a 24-hour stale-while-revalidate window.

**K-081 · S3 · Legal publication is a frozen TypeScript constant.**
`lib/server/legal-publication.ts` — `active: true`, `effectiveDate: '2026-09-07'`, `termsVersion`, `privacyVersion`, all `Object.freeze`d in source. Publishing a new version of the terms is a code change and a redeploy, not an owner action, while `internal/legalpublication/` and the platform-settings registry exist on the server side. Two candidate sources of truth for the published legal version.

**K-082 · S3 · `active: false` would publish the documents with the company details missing.**
`legal/privacy/+page.svelte:9` — `{data.legal.active ? data.legal.entityName : 'The company operating Kredit'}`, and the service address, contact email and postal address are each wrapped in `{#if data.legal.active}`. The document still renders; only the identity disappears. A placeholder controller name is one boolean away from being published.

**K-083 · S4 · The effective date equals the CAC incorporation date.** `legal-publication.ts:10` — `effectiveDate: '2026-09-07'`, the incorporation date in the brief, which the brief asks not to be used as a legal effective date.

**K-084 · S4 · Legal pages are `noindex` by default.** `lib/seo.ts:86` places `/legal/privacy` and `/legal/terms` in `nonIndexablePaths`; `+layout.svelte:25` re-includes them only when `legal.active`. Publication state is coupled to the robots meta tag.

**K-085 · S4 · Document section IDs are maintained by hand in three files.** `legal/terms` (14), `legal/privacy` (14) and `legal/complaints` (5) each duplicate their `<section id>` list into a `sections` array. They currently match — verified — but nothing enforces it, and the brief asked for the contents to be generated from the heading structure.

### 3.12 Build, config and dead weight

**K-086 · S3 · `hooks.server.ts` throws a SvelteKit `HttpError` at module load.**
`lib/server/legal-config.ts:14,20,23` uses `error(503, …)` from `@sveltejs/kit`, and `hooks.server.ts:5` calls `assertLaunchWebConfig()` at module scope. `error()` produces a value that is only meaningful when thrown inside `load` or `handle`; thrown during module initialisation it crashes the module with an opaque object rather than serving a 503. **[needs runtime proof]**

**K-087 · S3 · Buyers and admins are redirected to the supplier sign-in page.**
`hooks.server.ts:27-34` — any unauthenticated request to `/buyer/*` or `/admin/*` is redirected to `/app?next=…`, which is the seller sign-in screen, headed *"Start or sign in"* with the copy *"New here? Start with your phone number or email. Your business details come next."*

**K-088 · S3 · The upload body limit is keyed on a path suffix.** `hooks.server.ts:79` — `event.url.pathname.endsWith('/documents') ? 3 : 2` MiB. `POST …/documents/{id}/complete` and `…/documents/upload-slot` do not end in `/documents` and receive the 2 MiB limit.

**K-089 · S4 · `$features` alias points at a directory that does not exist.** `svelte.config.js:9` — `$features: 'src/lib/features'`.

**K-090 · S4 · Service-worker handler takes an unused parameter and calls `skipWaiting()` outside `waitUntil`.** `service-worker.ts:6-10`.

**K-091 · S4 · Two components are written in Svelte legacy mode.** `admin/diagnostics/+page.svelte` and `admin/analytics/+page.svelte` use plain `let` with no runes, while every other component uses `$state`/`$derived`.

**K-092 · S4 · `any` in a `strict: true` project.** `admin/platform-settings` (`editDraftValue: any`, six `catch (e: any)`), `admin/analytics` (`scorecard: any`, `metric: any`), `WorkspacePage` (`records: any[]`), `app/credit/[id]` (`view: any` and eleven more), `DisputeDetail`, `AdminAttention`, `admin/*` list pages.

**K-093 · S4 · Minified source.** `HomeProof.svelte`'s style block is a single 1,400-character line. `ProtectedActionDialog`, `admin-client.ts`, `AdminAttention`, `admin/diagnostics`, and most `admin/*` list pages compress their entire script into one or two lines. `admin/platform-settings` carries **501 lines of `<style>` in one route file** — more CSS than the entire shared `app.css` (426 lines).


## Part 4 — Backend (Go)

The Go layer is the strongest part of this repository. Before the findings, what is genuinely right, because it changes where the fixing effort belongs:

- **No `TODO`, `FIXME`, `XXX`, `HACK` or "not implemented" anywhere in the Go source.** Verified by sweep.
- **No `float64` touches money.** Every `float64` is a latency metric, an outbox jitter fraction, a percentile, or a statistic (`OnTimePercentage`, `AverageDaysLate`). `reports/store.go:131` even carries `Score *float64` with the comment *"always nil: factual history is not a score"*, matching the privacy notice's promise that there is no hidden score.
- **`ledger.Money` is `int64` kobo end to end**, with `FeeAtRate` splitting the multiplication (`(amount/10000)*bps + (amount%10000)*bps/10000`) so no valid int64 principal can overflow, and `ledger.CheckedAdd` guarding sums.
- **The fee rounding matches the frontend exactly.** Go floors; `fee-terms.ts` `amount * BigInt(bps) / 10000n` floors. They agree.
- **`payments/postgres.go` is exemplary**: `FOR UPDATE OF o, c`, an idempotency check *before and after* taking the lock, currency match, refusal of amounts above the authoritative outstanding balance, refusal of future-dated payments, and payment + allocation + balance + journal + fee + snapshot + outbox in one transaction.
- **`web/http_helpers.go:55 safeProblemDetail`** replaces any 5xx detail with a fixed string and scans every 4xx detail for `postgres`, `pgx`, `sql:`, `database`, `connection`, `provider`, `secret`, `password`, `token`, `stack`, `panic` and URLs, truncating on a rune boundary. Infrastructure detail cannot reach a client.
- **`web/auth_handlers.go`** rotates the session token on MFA step-up, issues recovery codes on first enrolment, enforces CSRF as double-submit *plus* `Sec-Fetch-Site`/`Origin`, sets `HttpOnly`/`Secure`/`SameSite=Lax`, and fails closed when `crypto/rand` fails rather than deriving a token from the clock.
- **`access/roles.go`** is a closed permission model whose every switch ends in `default: return false`.

Against that standard, the findings below matter more, not less.

### 4.1 The owner console controls nothing

**K-094 · S1 · The platform settings registry has no consumers.**

`internal/platformsettings` is imported by exactly two non-test files: `internal/web/platform_settings_handlers.go` (the CRUD for the settings screen) and `internal/web/runtime.go` (which constructs the store at line 173 and assigns it to the struct at line 666, and never reads it). Every read of `runtime.PlatformSettings` in the entire codebase is inside `platform_settings_handlers.go` — the settings page reading back its own rows.

Consumer counts, measured across `internal/` and `cmd/` excluding the registry definition and tests:

| Setting family | Consumers |
|---|---|
| `launch.mode`, `launch.banner_*`, `launch.waitlist_enabled` | **0** |
| `security.mfa_enforced`, `security.session_idle_minutes`, `security.max_login_attempts`, `security.ip_allowlist_enabled` | **0** |
| `fees.supplier_rate_bps`, `fees.late_fee_rate_bps`, `fees.grace_period_days` | **0** |
| `notifications.channels`, `notifications.pre_debit_reminder_days` | **0** |
| `integrations.mono.*`, `integrations.paystack.*`, `integrations.termii.*`, `integrations.resend.*` | **0** |
| `features.*` | 1 — `platform_settings_handlers.go:36`, which serves `/api/v1/platform/capabilities` |

Of 37 registry keys, the only ones with any effect are the seven `features.*` flags, and their sole effect is telling the browser what to render. The API enforces nothing from them. That is the failure mode the brief names directly: *"Hiding a button is not backend authorisation; front-end code is not a private security boundary."*

The required chain — **owner control → validated write → durable revision → effective policy → API/provider/worker action → customer interface** — is complete through "durable revision" and stops there. The counter-example is in the same file: `businesspolicy.NewStore(...)` *is* read at runtime (`operations/approvals.go:115`, `operations/postgres.go:101` read it inside the transaction), which proves the pattern is understood and simply was not applied to the platform registry.

**K-095 · S1 · The Mono client cannot be constructed in production.**

`internal/web/runtime.go:137`:
```go
if (cfg.MonoSweepEnabled || cfg.CollectionProvider == "mono-sweep") &&
   cfg.Environment != "production" &&
   strings.HasPrefix(cfg.MonoSecretKey, "test_sk_") {
    monoClient, _ = mono.New(...)
}
```
`internal/config/config.go:75`:
```go
if c.MonoSecretKey == "" || strings.HasPrefix(c.MonoSecretKey, "test_sk_") {
    return errors.New("mono Sweep production requires a live Mono secret key; sandbox keys are refused")
}
```
The two conditions are mutually exclusive. In production `monoClient` is always `nil`, so `runtime.go:155` (`mandateRuntime = mandates.NewUnavailableProvider("mono-sweep")`) stands and every mandate operation fails.

A third guard confirms the intent: `internal/providers/mono/webhook.go:40` rejects any event whose `live_mode` is true — *"live provider event rejected by sandbox adapter"*.

The Mono integration is, by three independent guards, **a sandbox-only adapter**. That is a defensible safety decision. The defect is that nothing says so: `integrations.mono.enabled` is offered as an owner switch, `/admin/mono` reports *"Technical readiness: Ready · No configuration blocker detected"* from configuration presence alone, and the homepage advertises WhatsApp reminders and payment links as shipped behaviour.

**K-096 · S1 · Production cannot start without live collections, live identity and ten approval references.**

`config.go:144-215` makes all of the following mandatory when `APP_ENV=production`: `FEATURE_APPROVED_RETENTION_POLICY`, `FEATURE_PRODUCTION_PILOT`, `FEATURE_REAL_IDENTITY` with a non-mock provider plus endpoint, token and webhook secret, `FEATURE_REAL_COLLECTIONS` with a non-mock provider plus endpoint, token and webhook secret, a non-empty `ADMIN_SURFACES`, and ten non-empty strings: `SECURITY_REVIEW_REFERENCE`, `DPIA_REFERENCE`, `LEGAL_APPROVAL_REFERENCE`, `PEN_TEST_REFERENCE`, `BACKUP_RESTORE_REFERENCE`, `PROVIDER_CERTIFICATION_REFERENCE`, `SUPPORT_TRAINING_REFERENCE`, `LAUNCH_APPROVAL_REFERENCE`, `PILOT_ALLOWED_PROVIDER_ACCOUNTS`, `PILOT_ALLOWED_INDUSTRIES`, plus seven positive pilot limits.

The brief requires the opposite: *"Missing Mono onboarding must not block a correctly configured public website or genuinely independent customer service"* and *"Production must not silently become development when optional keys are absent."*

As written there are two ways to launch, and both are wrong. Run `APP_ENV=production` and you must have a live bank collection provider and a certified identity provider before the marketing site can serve a request. Run `APP_ENV=staging` and **every** production check disappears at once — placeholder secrets, `http://` URLs, a localhost database, mock identity, mock collections. There is no third setting. The environment switch conflates *infrastructure hardening* with *capability activation*, which is precisely the refactor the brief asks for.

**K-097 · S2 · Three integrations are advertised with no adapter.**

`internal/providers/` contains one directory: `mono`. A sweep for `paystack`, `termii` and `resend` across all Go source finds them only in `platformsettings/registry.go` and in one test that rotates a Paystack secret. The registry nevertheless exposes `integrations.paystack.enabled`, `.secret_key`, `.public_key`, `.status`, and the same four keys each for Termii and Resend — twelve settings, including three secret-rotation surfaces, for vendors that have no code. This is exactly what the brief forbids: *"A form for Mono, Paystack, Termii, Resend … is not proof of a working integration. Do not add a new vendor merely because its name appears in a registry."*

**K-098 · S2 · The registry ships the launch banner the brief forbids.**

`registry.go:76-106` defines `launch.mode` with `pre_launch | private_launch | public_launch`, plus `launch.banner_enabled`, `launch.banner_text`, `launch.waitlist_enabled`. The brief: *"Do not render those internal readiness assessments as three customer-facing launch modes or banners. The customer-facing service is simply Kredit. It does not need a 'We are now live' badge."* There is also no waitlist feature anywhere in the codebase.

**K-099 · S2 · A late-penalty fee setting exists for a product that does not charge one.**

`registry.go` — `fees.late_fee_rate_bps`, *"Default late penalty fee rate in basis points"*, range 0–5000 (up to 50%). The product's actual fee model, in `ledger/fee_terms.go` and in the published disclosure, is a base fee on activation plus a collection fee on amounts collected. No penalty fee exists in the ledger, the schedule, or the customer disclosure. The brief: *"Do not invent or enable penalty fees not approved for this business model."*

**K-100 · S3 · "Verified" is an unreachable integration status.**

`registry.go:30` declares `IntegrationStatusVerified`, and `validateIntegrationStatus` deliberately omits it from the accepted set — correctly preventing an owner from typing themselves a verification. But nothing else ever writes it either. There is no server-owned verification record anywhere: no provider, environment, credential fingerprint, test type, timestamp, result or expiry, as the brief requires. The constant is a placeholder for a mechanism that does not exist.

**K-101 · S2 · An emergency pause never reaches a running process.**

`runtime.go:581` calls `collectionEngine.SetFeatureEnabled(collectionEnabled)` **once, at construction**. There is no refresh path, no invalidation, no re-read. `cmd/worker/main.go` and `internal/jobs/client.go` do not reference platform settings at all. The brief: *"Specify cross-instance propagation, cache invalidation and worker refresh; emergency pauses must not wait indefinitely on a stale process-local cache."* Today a pause takes effect on the next deploy.

### 4.2 Secret handling

**K-102 · S1 · Settings encryption has no key identity, no context binding, and no start-up validation.**

`internal/platformsettings/crypto.go`:

- Line 23 — the key is read with `os.Getenv("SETTINGS_ENCRYPTION_KEY")` **inside the constructor**. It is not a field of `config.Config`, so it is not covered by `Validate()`, not in the production required-secrets list, and not checked by `validateSecret`. **A production deployment starts successfully with settings encryption unusable** and only discovers it when the first secret write fails.
- Line 28 — `sha256.Sum256([]byte(key))` derives the AES key by a bare hash: no KDF, no salt.
- Line 51 — `gcm.Seal(nonce, nonce, []byte(plaintext), nil)`: **no additional authenticated data**. Nothing binds a ciphertext to the setting key it belongs to, so a ciphertext copied from one settings row into another decrypts cleanly. A mis-targeted update, a partial restore, or anyone with database write access can silently swap one provider credential for another.
- Nothing stores a key ID alongside the ciphertext, so **there is no rotation path**. Changing `SETTINGS_ENCRYPTION_KEY` makes every stored secret permanently undecryptable with no way to tell which key encrypted what.
- `len(key) < 32` counts bytes of the string, so a 32-character hex key carries 128 bits of entropy and is accepted.

The one part of KSV-02 that *is* fixed: the usable default key is gone. `NewEncryptor` returns a keyless `&Encryptor{}` and both `Encrypt` and `Decrypt` fail explicitly.

**K-103 · S2 · The secret fingerprint is an unsalted 64-bit hash of the plaintext.**

`crypto.go:86-92` — `SecretFingerprint` is `hex(sha256(plaintext)[:8])`. It is deterministic and unsalted, so the same credential in staging and production produces the same fingerprint, letting anyone with console access confirm they are shared; and 64 bits is short enough to attack offline for a low-entropy secret of known format. It is also rendered in the main settings table rather than a detail view. A keyed HMAC under the root key fixes both properties.

**K-104 · S3 · `MaskSecret` reveals the last four characters and does not know Mono's key format.**

`crypto.go:94-114` — every masked secret exposes its final four characters. For a webhook signing secret that is a real reduction in entropy. It special-cases the prefixes `sk_live_` and `sk_test_`, but `config.go:75` shows Mono's sandbox prefix is **`test_sk_`**, which matches neither branch and falls to the generic `te••••••••XXXX`.

**K-105 · S3 · Hardcoded fallback keys remain in the auth layer.**

`internal/auth/store.go:111-116` — `NewStoreWithKeys` substitutes `"development-only-change-me"` for an empty `tokenHashKey` or `otpHMACKey`. Production configuration refuses that value, so the path is unreachable there, but the literal is still the auth package's own default.

**K-106 · S3 · Default credentials in configuration.**

`config.go:119-133` — `DATABASE_URL` defaults to a URL with an embedded password; `OBJECT_STORAGE_ACCESS_KEY`/`SECRET_KEY` default to `minioadmin`; four cryptographic keys default to `development-only-change-me`. `Validate()` rejects all of them in production and **none of them in staging**, where placeholder secrets, `http://` endpoints and a localhost database are all accepted.

**K-107 · S3 · The Mono webhook is authenticated by a shared secret, not a signature.**

`internal/providers/mono/mono.go:288` — `VerifySecret` is a constant-time comparison against the configured secret. There is no HMAC over the body. Kredit's own connectors do this correctly: `collections/webhook_provider.go:67` and `identity/webhook.go:79` both compute `hmac.New(sha256.New, secret)` over the payload. Mono's published contract is a shared-secret header, so this matches the provider — but it means a captured request body plus the secret forges any event, and event-ID de-duplication is the only replay control. It should be documented as an accepted provider limitation rather than described as signature authentication.

### 4.3 Silent failure and swallowed errors

**K-108 · S2 · Every provider constructor discards its error.**

`internal/web/runtime.go`:

| Line | Code | Consequence of failure |
|---|---|---|
| 135 | `webhookJobs, _ = jobs.NewEnqueueClient(...)` | webhook jobs silently unavailable |
| 137 | `monoClient, _ = mono.New(...)` | Mono silently absent — e.g. an invalid redirect URL |
| 129 | `if provider, err := identity.NewWebhookProvider(...); err == nil` | identity silently stays "unavailable" |
| 150 | `if remote, err := mandates.NewWebhookProvider(...); err == nil` | silently falls back to the Postgres-only provider |

Nothing is logged and nothing is surfaced in readiness. The brief: *"Remove unsafe placeholder implementations, swallowed errors, silent fallbacks … and accidental mock adapters."*

**K-109 · S3 · An unchecked type assertion on client-influenced JSON.**

`internal/businesspolicy/policy.go:85` — `n := fields[f.Key].(float64)`, where `fields` comes from marshalling `Values` and re-unmarshalling into `map[string]any`. Any catalogued numeric key absent from the marshalled JSON yields `nil`, and `nil.(float64)` panics. The panic-recovery middleware converts it to a 500 and the idempotency middleware records the failure, so no state is corrupted — but a policy proposal can panic the handler. **[needs verification of the `Values` struct tags]**

**K-110 · S4 · 103 discarded errors in `credit_handlers.go` are benign but hide the real ones.**

Almost all are `orgID, _ := pathID(r, "organizationID")`. An empty `orgID` then fails `requireOrganizationAccess` and an empty `id` fails the `GetForSupplier` lookup, so the outcome is safe. The cost is that the file's genuine discards (`_, _ = s.runtime.EmitNotification(...)` at line 568, where a critical `PaymentRecorded` notification failure is dropped) are indistinguishable from the noise.

### 4.4 Error contracts and status codes

**K-111 · S3 · A malformed identifier is reported as rate limiting.**

`internal/web/auth_handlers.go:43-46` maps **every** `RequestOTP` error to `429 Too Many Requests`, including `"valid identifier and channel are required"`. The frontend maps 429 to *"Please wait before requesting another code. The previous code may still arrive."* A user who mistypes their email is told to wait.

**K-112 · S3 · Domain error strings become customer copy.**

`safeProblemDetail` strips infrastructure detail but passes domain text through. Strings that reach a browser today include *"otp challenge is locked"*, *"mfa method is not enrolled"*, *"idempotency key was reused for a different payment"*, *"operation reaches the configured approval threshold"*, *"cumulative corrections require a proposal in Admin Financial changes"*. Several admin screens render `body.detail` verbatim, so these are the owner's error messages.

**K-113 · S3 · `/meta` is unauthenticated and reports the environment.**

`internal/web/server.go:569` registers `/meta` and `/api/v1/meta` returning `environment`, `version`, `timezone`, `currency`, `money_unit` with no authentication.

### 4.5 Money and product-boundary findings

**K-114 · S2 · A correction of ₦10,000 or more is refused from the supplier screen with no path forward.**

`internal/operations/store.go:14` — `highValueThreshold = 1_000_000` kobo. `internal/operations/postgres.go:106` computes `threshold := min(highValueThreshold, policy.Values.CorrectionThreshold)` and refuses any unverified correction at or above it, and the same for cumulative corrections. The correction must instead be proposed in the admin console and approved (self-approved in solo-owner mode). That is a reasonable design — but:

- the supplier UI says *"Changes of ₦10,000 or more need a second person to approve them first"*, which is wrong on both counts;
- the figure is hardcoded in the frontend while the server takes the minimum of the constant and a configurable policy value, so the displayed threshold is wrong whenever policy is lower;
- the in-memory `operations.Store` has **no `verified` path at all** (`store.go:66`), so in any deployment using it, corrections at or above the threshold are permanently impossible.

**K-115 · S2 · Overpayment is refused outright.**

`internal/payments/postgres.go:127` — `if input.AmountKobo > snapshot.OutstandingKobo { return errors.New("payment exceeds authoritative outstanding amount") }`. A buyer who rounds up a transfer, or pays a small amount twice, leaves the supplier unable to record what actually arrived. The brief asks for overpayment *handling*; refusal is a decision, not a mechanism, and there is no credit-balance or refund path.

**K-116 · S3 · Nil fee terms silently mean 0.5% / 0.5%.**

`internal/ledger/fee_terms.go:18-23` — `(*FeeTerms)(nil).Rates()` returns `(50, 50)`, documented as *"the historical 50/50 basis-point contract"*. Because `Validate()` calls `Rates()`, **`(*FeeTerms)(nil).Validate()` returns nil** — nil fee terms validate successfully. The frontend takes the opposite position: `fee-terms.ts:3 validFeeTerms` returns false for null and the buyer screen refuses acceptance with *"Fee terms are unavailable."* Two layers disagree about whether missing money terms are acceptable.

**K-117 · S3 · Two different fee disclosures for the same terms.**

`ledger/fee_terms.go:63` produces *"0.50% supplier base service fee on activated principal; an additional 0.50% only on amounts Kredit successfully collects…"*. `web/src/lib/fee-terms.ts:12` produces *"The seller pays 0.5% when this sale becomes active…"*. Different precision and different wording for the term that goes into a binding agreement, depending on whether the reader is looking at the document or the screen.

**K-118 · S3 · No minimum collection amount.**

`internal/collections/engine.go:174-256` reduces the target by disputed and claimed amounts and returns `Eligible: len(reasons) == 0` with whatever remains. If holds reduce the target to a few kobo without zeroing it, a bank debit for that amount is eligible. Provider cost exceeds the recovery.

**K-119 · S3 · Provider capabilities are asserted, not discovered.**

`internal/collections/webhook_provider.go:37` returns every capability as `true` for any configured webhook provider, so `ValidatePolicy(snapshot.CollectionPolicy, capabilities)` in the eligibility engine can never fail for a real provider. The capability check only constrains the mock.

**K-120 · S3 · A collection connector may be configured over plain HTTP outside production.**

`webhook_provider.go:28` accepts `http` as well as `https`. `config.Validate` requires https for `COLLECTION_PROVIDER_ENDPOINT` in production only.

### 4.6 Policy and session defaults

**K-121 · S3 · Creating a sale requires step-up MFA.**

`access/roles.go:162-168` — `RequiresStepUp` includes `PermissionCreateCredit` and `PermissionReleaseGoods`. For a trader whose daily work is recording sales, a fresh authenticator code per sale is a heavy tax; combined with **K-057** (the step-up form cannot be submitted) it is currently a wall.

**K-122 · S3 · An Administrator cannot read financial data but a Viewer can.**

`access/roles.go:153-154` — `PermissionReadFinancial` is granted to `RoleFinance`, `RoleCollections` and `RoleViewer`, and not to `RoleAdministrator`. Whether deliberate or not, "Administrator" ranking below "Viewer" for financial reads will surprise the owner configuring staff access.

**K-123 · S3 · Session lifetimes are long for a shared-device market.**

`auth/store.go:37-39` — 30-day absolute lifetime, 14-day idle timeout, with `security.session_idle_minutes` present in the registry and unread (**K-094**).

**K-124 · S3 · Per-IP rate limits on a carrier-NAT market.**

`web/server.go:41,52` — `rateLimitPerMinute = 120` per client IP, and `sensitiveRateLimitPerWindow = 20` per 10 minutes for the whole `auth` route group, keyed on IP. Nigerian mobile networks put very large numbers of subscribers behind one public address; twenty OTP requests per ten minutes shared across everyone on a carrier NAT will lock out legitimate users. The shared counter is correctly fail-closed (`web/server.go:199-206`), which makes the blast radius larger, not smaller.

**K-125 · S3 · Any RFC1918 peer is a trusted proxy.**

`web/server.go:233-262 clientIP` trusts `CF-Connecting-IP`, `X-Real-IP` and `X-Forwarded-For` from any loopback, private or link-local peer. The SvelteKit proxy correctly strips those headers before forwarding, but the trust boundary is "any private-network peer" rather than a configured proxy address.

**K-126 · S3 · Enrolling MFA does not require step-up.**

`web/auth_handlers.go:111-132` — `enrollTOTP` requires only `requireAuth` + CSRF and returns the TOTP secret in the response. Anyone holding a live session on a shared device can enrol the second factor for that account. `BeginTOTPEnrollment` refuses when a verified method already exists, which bounds it to first enrolment.

---

## Part 5 — Database

The schema is the most mature layer in the repository.

- **107 tables, 174 row-level-security policies**, restricted `kredit_app` / `kredit_worker` / `kredit_migrator` / `kredit_backup` roles, `BYPASSRLS` only on the backup role.
- **`infra/postgres/roles.sql:29`** — `REVOKE UPDATE, DELETE ON app.audit_events FROM kredit_app, kredit_worker`. The audit log is append-only by grant, not by convention.
- **`roles.sql:33-36`** — `ALTER DEFAULT PRIVILEGES … REVOKE` so a new table is unreachable by either runtime role until a migration makes an explicit privilege decision.
- **Every stored money column is `bigint`.** No `numeric`, `real`, or `double precision` holds an amount, and 39 `CHECK` constraints guard kobo columns.
- **Migration 088** enforces append-only settings history with both a `BEFORE UPDATE OR DELETE` trigger and a grant revocation, and its `Down` deliberately preserves the boundary.
- **Migration 089** protects the last owner with a statement-level `pg_advisory_xact_lock` taken *before* row locks and shared across `platform_role_assignments` and `users`, then re-checks the invariant in an `AFTER STATEMENT` trigger — which is the correct shape for the concurrent-removal race. It also refuses to install against a database that already lacks an active, non-expiring owner.
- **Migration 090** makes `current_governance_mode()` use `SELECT … INTO STRICT`, so a missing governance row raises instead of defaulting to solo-owner.

Findings:

**K-127 · S3 · The reconciliation table stores money as `numeric`, and the console cannot read it.**
`db/migrations/056_financial_reconciliation.sql:38` — `expected numeric NOT NULL, actual numeric NOT NULL`. This is the only place a money value is stored outside `bigint`. `web/src/routes/admin/reconciliation/+page.svelte` parses it with `BigInt(value)`, which **throws on any value with a decimal point** and falls back to `'Unavailable'`. A fractional reconciliation difference is invisible in the console.

**K-128 · S4 · The advisory lock key is a bare magic number.**
`089_owner_lifecycle_guard.sql` — `pg_advisory_xact_lock(746219830045::bigint)` with no named constant and no note on how it was derived, so a future lock cannot be chosen safely.

**K-129 · S4 · The two runtime roles have identical table grants.**
`roles.sql:27-28` grants `SELECT, INSERT, UPDATE` on all tables to both `kredit_app` and `kredit_worker`. Their separation rests entirely on RLS policies; the grants do not distinguish them.

**K-130 · S3 · One-way `Down` migrations need a documented forward-repair path.**
Migrations 088, 089 and 090 all have `Down: SELECT 1` with a comment explaining that the integrity boundary is deliberately retained. That is right, but it means a code rollback past these points leaves constraints the older code does not expect. The brief asks explicitly for *"Separate schema rollback, forward repair and code rollback"* — the separation exists in the migrations and is not written down anywhere an operator would find it.

---

## Part 6 — Infrastructure, CI and tests

**K-131 · S1 · The legal-publication mechanism was replaced by a constant and its CI gate was left behind.**

- `.env.example:63-70` documents `LEGAL_DOCUMENTS_ACTIVE`, `LEGAL_ENTITY_NAME`, `LEGAL_SERVICE_ADDRESS`, `LEGAL_CONTACT_EMAIL`, `PRIVACY_CONTACT_EMAIL`, `LEGAL_EFFECTIVE_DATE`, `TERMS_VERSION`, `PRIVACY_VERSION`.
- **No source file reads any of them.** `web/src/lib/server/legal-config.ts:6` is `return legalPublication;` — the frozen object in `legal-publication.ts`.
- `.github/workflows/product-audit.yml` runs a step named *"Verify legal-document activation with fictional CI-only details"* which sets all eight variables to fictional values (`Kredit Launch Test Limited`, `1 Fictional Test Street`, `legal@example.test`, effective `2026-09-01`).
- The test it runs, `web/tests/content-seo.spec.ts:82-94`, asserts the **real** values: `KREDIT TECHNOLOGIES LIMITED` and `Effective 7 September 2026`.

The fictional variables are ignored; the assertions pass only because the values are hardcoded in source. There is no test that publication works — only a test that a constant is a constant. And `.env.example` describes a deployment contract that no longer exists.

**K-132 · S2 · Four of seven workflows are gated on branches that are not the working branch.**
`phase2-tenant-isolation.yml`, `phase3-financial-proof.yml`, `phase4-provider-verification.yml`, `phase5-production-assurance.yml` and `phase6-context-audit.yml` trigger on `push` to branches such as `phase2-tenant-isolation` and `phase6-hardening-polish`, plus path-filtered pull requests. They do not run on a push to `main`. The tenant-isolation gate, the financial-proof gate and the request-context audit are therefore not gates on the branch being released.

**K-133 · S2 · The browser suite runs one engine and blocks the service worker.**
`web/playwright.config.ts:22` — a single `chromium` project; `serviceWorkers: 'block'`; `retries: 0`; `workers: 1`. `.github/workflows/product-audit.yml` installs only Chromium. Firefox and WebKit — the brief's explicit requirement — are never executed, and the service worker's cache behaviour is never tested.

**K-134 · S2 · No UI-level sign-in test exists, which is why K-057 shipped.**
`web/tests/audit-product-journeys.spec.ts:97` asserts only that the six-digit code textbox is visible. `web/tests/real-stack-financial.spec.ts:8,16` authenticates by calling `page.request.post('/api/v1/auth/otp/challenges')` and `…/verify` directly, bypassing the form entirely. No test fills the field and submits it. The suite's shape — API-level requests standing in for the journey — is the reason a broken `pattern` attribute on the primary authentication form reached the build.

**K-135 · S3 · Unpinned container images in the local stack.**
`docker-compose.yml` pins `postgres:18` but uses `minio/minio:latest`, `minio/mc:latest` and `axllent/mailpit:latest`. The local environment a financial product is developed and verified against is not reproducible.

**K-136 · S4 · `$features` alias points at a directory that does not exist** (`svelte.config.js:9`), and `web/src/lib/api/generated/` — 18,567 lines — is regenerated on every `dev`, `build`, `test` and `lint` to serve one call site (**K-023**).

---

## Part 7 — The owner's named complaints, re-checked against source

| Complaint | Status in the current tree |
|---|---|
| "Complete pre-launch draft — legal approval pending" and similar | **Gone.** No forbidden-phrase string survives in `web/src`. |
| Pricing opens in an error state | **Fixed at the source.** `pricing/+page.server.ts` fetches `/api/v1/pricing` server-side with a 5-second deadline and `cache-control: no-store`, and renders a single restrained *"Pricing is temporarily unavailable."* when there is no valid publication. |
| Session diagnostics shown to visitors | **Fixed for public pages, moved not removed for private ones.** Public routes render without any session call. `AuthGate.svelte` still blocks every private route behind a client-side `/api/v1/me` with *"Checking your account…"* and a **Try again** button (**K-033**), while the server-side cookie state that would remove it already exists in `hooks.server.ts:28`. |
| The defensive disclaimer paragraph | **Still on the public site in three places** (**K-058**): `HomeProof.svelte` card 04, the homepage `.assurances` tick-list, and `financial-copy.ts:creditBoundary` on `/pricing`. |
| Poorly designed tables of contents | **Substantially fixed.** `DocumentLayout.svelte` gives a sticky bounded column, a mobile `<details>`, `scroll-margin-top`, and print rules. Remaining: the active-section indicator picks the last intersecting entry rather than the topmost (**K-055**), desktop links are ~36px (**K-054**), section IDs are duplicated by hand (**K-085**), and every `h2` renders in Arial because `--font-sans` is undefined (**K-005**). |
| Unnatural or patronising copy | **Partly.** Product copy has improved markedly. Still present: sentence-length role labels (**K-061**), *"Customer was right"* for a dispute finding (**K-062**), *"No big grammar."* in a public meta description (**K-065**), and `admin/platform-settings` written in a wholly different, machine-generated register (**K-059**). |
| Workspace occupied by marketing prose | **Still present.** `tips` bullets above the data on five workspace screens (**K-017**), sentence-as-heading page titles, and display headings sized for a landing page then silently truncated by a second stylesheet (**K-001**). |

### The ten KSV findings

| | Status |
|---|---|
| **KSV-01** disconnected provider controls | **Not fixed — K-094, K-095.** The registry has no consumers; Mono cannot be built in production. |
| **KSV-02** unsafe encryption fallback | **Partly — K-102.** The usable default key is gone. No key ID, no AAD, no rotation path, not validated at start-up. |
| **KSV-03** editable "verified" status | **Fixed, but hollow — K-100.** Owners cannot set it; nothing can. |
| **KSV-04** public rollout leakage / fail-open UI | **Fixed.** `/api/v1/platform/capabilities` returns only feature booleans, defaults to `false`, and keeps `disputes` unconditionally available. |
| **KSV-05** ineffective versioning | **Fixed.** `expected_version` is mandatory and `ErrVersionConflict` returns 409. |
| **KSV-06** mutable / unredacted settings history | **Fixed server-side (migration 088).** The client still prints a secret verbatim if the server ever sends one (**K-047** context). |
| **KSV-07** open-ended registry keys | **Fixed.** `ValidateKeyAndValue` rejects unknown keys, `null`, and out-of-range values. |
| **KSV-08** owner lifecycle gaps | **Largely fixed (migration 089).** Concurrency handled properly. |
| **KSV-09** incomplete console coverage | **Not fixed — K-094, K-097, K-098, K-099, K-013.** ~29 of 37 settings are inert, three vendors have no adapter, and the audit trail renders as UUID cards. |
| **KSV-10** release evidence | Out of audit scope; not executed. |

---

## Part 8 — What is genuinely world-class already

Naming this matters, because the fix plan should protect it rather than rewrite it.

1. **`internal/payments/postgres.go`** — locking order, double idempotency check around the lock, currency and outstanding-balance validation, one transaction for payment, allocation, balance, journal, fee, snapshot and outbox.
2. **`db/migrations/088`, `089`, `090`** — append-only history by trigger *and* grant; last-owner protection with a statement-level advisory lock that actually closes the concurrent-removal race; governance that fails closed. These are better than most production systems of this size.
3. **`infra/postgres/roles.sql`** — least privilege with `ALTER DEFAULT PRIVILEGES … REVOKE`, so new tables are unreachable until someone decides otherwise.
4. **`web/src/lib/api/mutation.ts`** — a genuinely correct client mutation layer: persisted request identity, payload fingerprint, refusal to reuse a key for different data, refusal to silently expire, and `not_sent | rejected | unknown` as first-class outcomes.
5. **`web/src/routes/buyer/credit-requests/[requestID]/+page.svelte`** — the best screen in the product. `mayAccept` gates acceptance on a verified 64-hex agreement hash, valid fee terms, non-empty party names and parseable dates; the Mono URL is allowlisted; and the copy is exact — *"A transfer report is not a payment confirmation"*, *"Returning from the provider is not confirmation."*
6. **`web/src/routes/app/overview/+page.svelte`** — `Resource<T>` states, `LatestRequest` generation guards, per-organisation scope checks on every result, and a real attention queue.
7. **`web/src/lib/api/reliable.ts`** — `LatestRequest`, `checkedJSON` with a deadline that survives body decoding, and decoders that refuse to render an unverified financial value.
8. **`web/src/routes/legal/terms` and `legal/privacy`** — substantive, adult, precise, with real regulator references, no "in plain words" panels, and no draft warnings.
9. **`internal/web/http_helpers.go:safeProblemDetail`** — a disciplined outbound redaction boundary.
10. **`internal/ledger`** — double-entry with named control accounts, overflow-safe fee arithmetic, and rounding that matches the frontend exactly.

---

## Part 9 — Suggested fix order

Sequenced so that each stage makes the next one cheaper.

**Stage 1 — things that are broken now (days)**
K-057 sign-in `pattern` (and the two other occurrences) · K-021 unguarded protected command · K-027 failed reads rendered as ₦0 · K-026 float kobo in the dispute and report paths · K-038 customer notes surviving sign-out · K-013 the audit trail · K-131 the legal-publication CI gate.

**Stage 2 — decide the launch model (days, mostly design)**
K-096 split `APP_ENV` into infrastructure hardening and capability activation, so the public site and non-payment services can run in production without a live bank provider · K-095 decide whether Mono is sandbox-only and make every surface say the same thing · K-102 move `SETTINGS_ENCRYPTION_KEY` into `Config`, add a key ID and AAD, define rotation.

**Stage 3 — make the console true (1–2 weeks)**
K-094 wire each surviving setting to a real consumer with a refresh path, and delete every key that has none · K-097 remove Paystack, Termii and Resend until adapters exist · K-098 remove the launch modes and banner · K-099 remove the late-penalty fee · K-101 give collections a runtime refresh so a pause takes effect · K-060 remove the "second person" copy and finish solo-owner governance · K-047 replace the hand-rolled modal.

**Stage 4 — one design system (2–3 weeks)**
K-001 to K-008 collapse `app.css`, `product-ui.css` and the `!important` layer into one token set with one owner per property · K-009 delete or adopt the 25 dead components, starting by adopting `StatusPill`, `Money`, `DateTime`/`DueDate`, `ReferenceCode`, `FeeBreakdown`, `ScheduleTable` · K-012 to K-019 replace `WorkspacePage` with typed list components per domain · K-020 route every mutation through `MutationIntent` · K-032 one error surface per page.

**Stage 5 — the audience (1–2 weeks)**
K-042 raise every font below 12px · K-040, K-041 fix the two contrast failures · K-043 to K-051 the accessibility set · K-061 to K-072 one copy pass with a single voice · K-030, K-031 decide how money is displayed and show the exact amount everywhere.

**Stage 6 — prove it (1 week)**
K-134 a UI-level sign-in test that would have caught K-057 · K-133 add Firefox and WebKit and stop blocking the service worker · K-132 make the tenant-isolation, financial-proof and context-audit gates run on `main` · a test per Stage 1 finding that fails on the old behaviour first.

---

## Appendix — coverage

| Area | Read | Notes |
|---|---|---|
| Frontend foundation (9) | all | incl. compiled output in `web/.vercel/output` and `web/build` |
| Frontend shared code (18) | all | |
| Components (48) | all | import graph verified per component |
| Routes (~105) | all | |
| Go entry points (15) | `cmd/api`, `cmd/worker`, `cmd/bootstrap-owner`, `cmd/migrate` inspected; rest by sweep | |
| Go domain (~150) | config, access, auth, ledger, operations, payments, collections, businesspolicy, platformsettings, providers/mono, db read closely; remainder by pattern sweep across all files | |
| Go HTTP layer (40) | `server.go`, `auth_handlers.go`, `http_helpers.go`, `platform_settings_handlers.go`, `runtime.go`, `credit_handlers.go` read closely; rest by sweep | |
| Database (~120) | roles, migrations 088–090 read in full; all 90 migrations swept for money types, constraints, RLS, policies | |
| Infra & CI (~70) | workflows, `docker-compose.yml`, `Taskfile.yml`, `roles.sql`, script inventory | Terraform and the ~50 shell/python scripts inventoried, not line-audited |
| Tests (~120) | Playwright suite inventoried and the authentication path read; Go tests not line-audited | |
| Docs (~95) | the governing brief read in full; `docs/` inventoried and deliberately not used as evidence | |

**Not established by this audit:** nothing was executed. No test ran, no server started, no provider was called, no contrast ratio was measured against a rendered pixel, and no assistive technology was used. Colour findings are computed from declared hex values; the two failures (2.96:1 and 3.12:1) are far enough from the threshold to be safe conclusions. Findings that would need execution are marked **[needs runtime proof]** at the point they are made.
