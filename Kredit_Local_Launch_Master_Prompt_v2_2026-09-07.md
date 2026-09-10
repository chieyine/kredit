# Kredit — Local launch implementation and full design/content overhaul

**Replacement brief · Version 2 · 7 September 2026**

This is the complete replacement for the earlier GitHub-focused launch prompt. Use this document by itself as the governing task. Do not append the older prompt's remote-delivery, domain-verification or mailbox-verification requirements. The latest owner instructions in this brief take precedence over contradictory historical reports.

**Working environment:** the current project on the owner's computer. **Delivery:** the completed local application, checked local release build and verifiable local changes. **Not part of this task:** GitHub/cloud operations, domain/mailbox verification, remote deployment or live financial activation.

## 1. Your assignment: finish the real local product, not another checklist

Work in the Kredit project that is open on my local computer. Be the implementation engineer and product designer responsible for finishing its launch experience across public pages, authentication, merchant and buyer journeys, Owner Super Admin, backend, database and workers. Change the actual application, run it locally, test it and visually inspect it. Do not stop at an audit, a plan, a mock-up, a settings table or a completion narrative.

My present dissatisfaction is specific: customer-facing draft/legal-approval warnings, repeated defensive explanations, unnatural or patronising copy, poorly designed tables of contents, pricing that opens in an error state, and session diagnostics shown to visitors. These are release defects. A prettier heading, a few removed sentences or another layer of CSS will not close them.

The result should feel like a mature Nigerian business product: distinctive, composed, useful, fast and trustworthy. It must be understandable without treating people as children. Marketing can have character; financial workspaces must be efficient. Every supported route needs the same care as the homepage.

I intend to launch immediately after the local work is complete. Build the normal customer experience as the published product, not as a preview of something unfinished. Do not add new public launch caveats to explain your own implementation limitations. Keep engineering notes and genuinely unresolved dependencies private, and give me an honest handover. Do not invent successful payments, licences, reviews, users, external approvals or tests to make the product look established.

Make concrete design decisions, implement them and show their results. Do not ask me to choose between generic visual directions before making progress. Do not argue about which AI product is better. Demonstrate the quality in the actual working application.

## 2. Accepted business information and scope boundaries

Use these supplied business facts and contact details directly:

| Field | Accepted value |
| --- | --- |
| Brand | Kredit |
| Registered company | KREDIT TECHNOLOGIES LIMITED |
| Company registration | RC 9834452 |
| Incorporation date | 7 September 2026 |
| Company type | Private company limited by shares, Nigeria |
| Company status | Incorporated; active in the supplied CAC documents |
| Website | https://kredit.com.ng |
| General company contact | hello@kredit.com.ng |
| Registered company address | House No. 348, Jamaina Road, Pompomari Bypass, Maiduguri, Borno State, Nigeria |

Use the registered company address where a company/service address is required. Do not substitute or expose the director's residential address, private phone, identity documents or full CAC uploads. Use hello@kredit.com.ng as the general contact and privacy-enquiry contact unless a different owner-supplied address is already configured. Do not invent separate privacy@, support@, legal@ addresses or name a Data Protection Officer without supplied information. Do not imply that using the general contact establishes a formal DPO appointment.

Domain registration, ownership, DNS, TLS certificates on the remote site, public reachability, SPF/DKIM/DMARC and mailbox provisioning/deliverability are OUT OF SCOPE for this task. Accept the supplied website and contact as the configured values. Do not verify them, send test mail to them, request access to them, or turn their untested external state into a warning or launch blocker. You may check that the code constructs links, messages and canonical URLs correctly; that is not a live domain or mailbox investigation.

Kredit is Nigeria-focused software for B2B trade-credit management, receivables administration and payment/collection integrations. Suppliers and buyers agree their credit sale. Preserve the actual business model and fund flows. CAC incorporation does not establish a banking licence, provider entitlement, a repayment guarantee or an external legal review. Do not manufacture those claims; do not repeat an entire defensive legal explanation across every marketing screen either.

I will initially operate the company alone. I require a genuine Owner Super Admin, separate from merchant ownership, for every supported ordinary operating decision. No second employee or fake approver account should be required in solo-owner mode. Customers see enabled services, not my private activation controls or your implementation checklist.

## 3. Local working tree is the source of truth

Start in the opened project directory. Inspect the current files, package manifests, working-tree status and local instructions. Do not assume the local project equals an old GitHub commit or download another copy to replace it. Existing uncommitted changes may be the latest real work. Preserve and inspect them.

Do not connect to GitHub, fetch branches, inspect remote PRs, configure branch protection, push, create PRs, merge remotely, query Vercel, or require a remote repository for this task. Local Git may be used to inspect history, save controlled checkpoints and produce a patch. Work remains local unless I later explicitly request a remote operation. Do not treat unavailable remote permissions as a blocker.

Read AGENTS.md and other project instructions, README, relevant architecture/runtime/configuration documents, local test scripts, and docs/product/public-launch-and-admin-controls.md if present. Read the supplied implementation verification report if available. Historical reports identify things to recheck; they do not prove what is in my current local working tree. If an old report is absent, continue from the requirements and actual source instead of asking me to recover it.

The earlier architecture was Svelte/SvelteKit, TypeScript, Go, PostgreSQL and workers. Verify the installed local versions and actual structure. Preserve functioning modules. Do not introduce a new framework, microservices, an unrelated starter app or a second administrator system simply because you prefer them.

Inspect the exact files that generate the complaints I supplied. Trace both the rendered component and its data dependencies. A string may come from a Markdown document, CMS record, server response, layout, email template, fallback constant, error translator or stale built asset—not only the visible page component.

## 4. Local execution, safety and durable progress

Verify at the start that you can read and edit this local project, run a terminal, start the application, use a browser and save evidence. Use existing package managers and lockfiles. Determine the required local API, worker and PostgreSQL services and start/configure isolated versions as needed. Do not spend the task trying unavailable cloud connectors.

Inspect environment-file NAMES and required configuration without dumping secret values. Before migrations, seeds, resets, load tests or destructive commands, establish that the target is an isolated disposable development/test database. A file on my laptop may still point to a production database. Do not infer safety from the working directory alone.

I authorise implementation, local commands, isolated local tests, ordinary local security checks and browser review. I do not authorise publishing the site, changing real customer data, rotating live credentials, contacting banks, debiting accounts, buying services or sending real messages merely because the task is to prepare for launch. Build the release and show exactly how to run it. Financial activation remains an explicit separate owner operation.

Use a local branch or checkpoint when local Git is available, but preserve the existing branch and unrelated work. Do not reset, clean, rewrite history, switch away from uncommitted work or stage all user files blindly. Save a clear change manifest and a patch that contains your changes, with the baseline identified. If there is no Git metadata, use a source manifest and safe patch/archive instead. Do not introduce encoded CI patch runners, cloud source-export workflows or credential workarounds.

When a command fails, inspect the error, fix its cause and use bounded retries. Do not repeatedly issue identical unavailable calls. If a tool limitation genuinely prevents a test, record precisely which test is unverified and preserve the real edits. Never say that work exists locally unless you can show its exact file path and diff. Never say a test passed unless you obtained its outcome.

Give short updates at meaningful implementation and visual-review milestones. Ask only for a genuinely missing fact that cannot be resolved locally and materially changes the work. Do not ask again whether I want good design, local execution or a working owner console. Do not promise unattended work or future delivery.

## 5. Establish a finite, complete launch inventory

Map all routes, pages, reusable components, API endpoints, data stores, background jobs, integrations, emails/messages, exports, configuration, permissions and deployment artifacts. Follow relationships between them; do not declare an area reviewed from a filename or a happy-path screenshot.

Create a traceability matrix with: capability, intended user, route/control, server/domain enforcement, database representation, worker/provider consumer, permissions, failure states, configuration dependencies, automated/manual tests, current status and evidence. Include every supported or publicly promised launch capability, not only the examples in this prompt.

Classify each item as: required for public website launch; required for an enabled customer service; required for live financial activation; or genuinely optional later expansion. Fix all in-scope defects. Do not quietly reclassify an agreed core feature as optional just because it is difficult. Conversely, do not postpone launch for speculative AI, native apps, extra countries or new providers I have not requested.

Maintain a separate issue list with severity, reproducibility, affected users, correction and regression proof. Unknown is not passed. “Not applicable” needs a reason. Tie every claimed completion to the actual candidate source revision.

Create a route-and-state inventory rather than a marketing-only checklist. Include logged-out, signed-in supplier, signed-in buyer, owner and delegated-staff experiences; desktop and mobile layouts; documents, tables of contents, dialogs, validation, empty states, navigation, emails and downloadable receipts. A screenshot of an empty dashboard is not proof that its populated and exception states are designed.

For this local task, external domain/mailbox checks and remote repository operations are explicitly excluded, not BLOCKED. Keep their exclusion out of customer-facing content and do not repeatedly raise them in the handover.

## 6. Recheck and resolve the ten known implementation findings

These findings were observed at the earlier baseline. Reproduce or disprove them on current source; do not assume they remain or have been fixed.

1. **KSV-01: disconnected provider controls.** Trace settings through `internal/web/runtime.go`, provider creation and workers. Saving `integrations.mono.*` must affect the real supported adapter, not merely a configuration table while `cfg.Mono*` remains authoritative elsewhere.
2. **KSV-02: unsafe encryption fallback.** Inspect `internal/platformsettings/crypto.go` and configuration. Remove a usable public/default development key from production-like secret operations; resolve the `TOKEN_SECRET` versus `TOKEN_HASH_KEY` mismatch without merely renaming one insecure fallback.
3. **KSV-03: editable “verified” status.** Generic configuration must not manufacture provider verification. Separate owner decisions from measured technical evidence and provider entitlements. Invalidate relevant evidence when credentials or environment change.
4. **KSV-04: public rollout leakage and fail-open UI.** Remove private launch/governance diagnostics from public capability responses. Optional new-use controls must not initialise enabled, appear during hydration or remain visible after failed capability reads. Preserve existing-record access.
5. **KSV-05: ineffective versioning.** Require and check the revision the owner reviewed. Reject stale changes atomically; support safe versioned rollback and atomic multi-setting changes where needed.
6. **KSV-06: mutable or unredacted settings history.** Enforce append-only runtime history and return neither plaintext secrets nor encrypted secret payloads in browser/history/export responses.
7. **KSV-07: open-ended registry keys.** Reject unknown, misspelled, untyped and incorrectly sensitive keys. A generic JSON store is not a supported settings registry.
8. **KSV-08: owner lifecycle gaps.** Verify the real bootstrap command, explicit identity verification, one-time behaviour, recovery, transfer, suspension/expiry and concurrent last-owner removal protection. Do not document flags or endpoints that do not exist.
9. **KSV-09: incomplete console coverage.** Connect every agreed operating control and correct category mismatches. Integrate existing working operational pages; implement missing supported content/legal/notification settings rather than adding inert tabs.
10. **KSV-10, adapted to local scope: release evidence.** Record the actual local source, build, migration and worker versions and the tests executed against them. Historical remote branch protection and deployment-target investigation are outside this task. Do not claim a local build proves a remote deployment or an enabled bank integration.

Resolve the original audit's financial-state, retry, timezone, organisation-switch, accessibility and copy defects too. Preserve its improvements. A green historical suite or an old completion report is not a waiver for a reproduced defect.

## 7. Art direction and component design: not cosmetic repair

Treat rendered visual quality as an independent acceptance gate. Do not call an interface premium because you added a gradient, a large headline, rounded cards or animation. Review page composition, information hierarchy, typography, spacing, alignment, density, colour, state treatment and interaction as one system.

Evaluate the existing warm-paper, ink, cobalt and restrained-coral identity. Keep recognisable assets that work, but do not preserve weak styling simply because it exists. Prefer clean warm-white/neutral surfaces for workspaces, strong readable ink, one disciplined primary accent and semantic colours reserved for meaning. Avoid yellow-looking wash over every screen, weak grey body text, oversized serif forms and blocks of coral used as decoration. Public editorial typography can be expressive; dashboard and financial text must remain exceptionally readable.

Consolidate tokens and components rather than appending another global premium.css layer. Establish a clear typography scale, tabular numeric treatment, spacing rhythm, layout grid, borders, radii, elevation, icon family, interactive states and responsive rules. Reuse one well-designed field, button, status, dialog and table system. Do not make every section a card or every note an alert.

Avoid the recognisable generic-template habits: repeated bento grids, floating money tiles, random trust badges, glass panels, gradients behind unrelated content, emoji labels, huge rounded empty-state boxes, decorative stock charts, repeated eyebrow/headline/paragraph/button blocks, unnecessary section numbering and a uniform three-card layout for every story. Personality should come from composition and precise product details, not ornament density.

Give the product four related but distinct treatments:

| Surface | Visual and informational priority |
| --- | --- |
| Public website | Clear proposition, memorable but restrained brand expression, useful product demonstration and direct routes into working services. |
| Supplier workspace | Amounts owed, upcoming/overdue work, customers, sales and required decisions, with compact readable navigation and tables. |
| Buyer journey | Who the seller is, what was agreed, the amount, due date, current status and one unmistakable next action. |
| Owner console | Attention queue and friendly operating controls, with technical identifiers and advanced diagnostics secondary. |

Improve actual layouts, not just their headings. The supplier's first screen should not be occupied by marketing prose. A buyer should not need to decode provider terms or locate a financial action beneath several reassurance boxes. The owner should not have to inspect a hundred raw keys to turn off SMS or change a published fee.

Design before writing new copy into a layout. Decide which content belongs on the page, which is redundant, which belongs in a detail view and which is mandatory at a decision point. Reducing clutter must not remove meaningful fee, consent, payment or dispute information.

Implement and inspect the real application in the local browser. Capture the initial state of representative screens, then compare the same route, viewport and state after changes. Review homepage, pricing, sign-in/MFA, terms, privacy, complaints, security, guides/TOCs, supplier overview, customers, sales, quick/advanced forms, buyer acceptance, payment handling, disputes, reports and owner controls. Do not substitute concept images or screenshots of a separate static mock-up.

The review must resolve clipping, accidental scrolling, inconsistent widths, poor line lengths, awkward gaps, noisy borders, competing primary actions, truncation of critical amounts, mismatched icons, duplicate step labels and controls hidden behind fixed headers, footers or the mobile keyboard. Test both empty and realistically populated pages. A numeric self-rating is not design evidence.

## 8. Smooth interaction, responsive behaviour and accessibility

Make mobile a first-class environment, not a compressed desktop layout. Test representative widths such as 320, 360, 390, 430, 768, 1024 and 1440 CSS pixels; include landscape, larger text, browser zoom, virtual keyboards and safe-area insets. Use an intentional table strategy rather than accidental whole-page horizontal scrolling.

Cover Chromium, Firefox and WebKit for critical journeys, and distinguish browser emulation from tests on real Android/iPhone hardware. Keep any physical-device or assistive-technology work not actually performed explicitly unverified. Do not claim conversations with merchants or observed user studies unless they happened.

Target WCAG 2.2 AA. Test semantic structure, names/labels, keyboard operation, visible unobscured focus, modal focus containment and restoration, error association, status announcements, contrast, zoom/reflow, reduced motion and accessible authentication. Automated checks alone do not establish conformance. Prefer approximately 44–48 CSS-pixel interactive targets in the product; this is a usability design target, not a claim that WCAG AA universally mandates that size.

Use restrained motion to explain changes. Prefer short feedback and transitions, avoid repeated layout/repaint work, honour reduced motion and pause hidden/offscreen effects. Never animate financial totals from zero or show celebratory settlement before authoritative confirmation. Do not block inputs until decorative animations finish.

Document measurable performance budgets. Use Core Web Vitals targets of LCP at most 2.5 seconds, INP at most 200 milliseconds and CLS at most 0.1 at the 75th percentile, split by device class where field data exists. Before meaningful production traffic, report repeatable lab and interaction tests as provisional evidence, not invented field percentiles. Record tools, network/CPU profile, sample count, cold/warm state and representative pages. Lighthouse alone does not prove real-user INP.

Optimise critical assets, images, font loading, route chunks, cache rules, main-thread work and unnecessary polling. Set justified bundle/API budgets against the real launch environment. No convenience feature should introduce uncontrolled third-party scripts, trackers or autoplay media into the critical path.

Keep interactions visually stable: no navigation flash from anonymous to authenticated to anonymous, no first-render disabled-feature flash, no controls jumping when amounts load, no skeleton that looks like a fake payment, and no repeated progress banners for background refreshes. Use appropriately sized placeholders only when loading is necessary; public static content must not depend on an account request.

Choose transitions deliberately, generally brief enough not to slow repeated tasks. Keep button focus and scroll position predictable. Preserve direct links and browser Back/Forward. Respect touch, pointer and keyboard use. A smooth experience is primarily one without confusion, unnecessary waiting or lost work.

## 9. Editorial overhaul: adult, precise, confident language

Rewrite the entire customer-facing vocabulary, including server error strings, not only the examples I pasted. Use natural business English suitable for Nigeria. Respect people with varied literacy and smartphone experience without baby talk, forced slang, rhetorical reassurance or scolding. Simple does not mean talking down to a merchant.

Do not repeatedly explain that the wording is simple. Remove meta introductions such as "Before you read all this", "In plain words", "a helping hand", "Don't worry", and "Here's the simple version" unless there is a concrete, exceptional reason for the phrase. Do not create a short-summary warning box before every legal section.

Use direct labels: Sales, Customers, Payments, Outstanding, Due date, Terms of service, Privacy notice, Contact, Save changes. Do not automatically replace familiar product terminology with long conversational labels. Buttons should state their action. Headings should state the topic. Empty states should help start the task without blaming the user.

The paragraph "Kredit writes your sale down and helps you follow the payment. Kredit does not lend you money, does not own the goods, does not choose your customer and cannot promise that anybody will pay you" must not survive as public product copy. It is clumsy and disproportionately defensive. Do not just shorten its first sentence and retain the same negative paragraph everywhere.

Illustrative direction for ordinary product description: "Manage credit sales and repayments in one place." Supporting text can say: "Record sales, track outstanding balances and keep payment records organised." Match the actual enabled product; these are editorial examples, not permission to advertise an unfinished capability.

Place material role/relationship provisions in a concise, professionally written service-role section of the terms and at relevant agreement points. Preserve substantive responsibilities and repayment-risk disclosures where required; do not plaster a multi-sentence disclaimer beneath every feature or use polished marketing to imply that Kredit lends, holds customer deposits or guarantees repayment.

Use exact transactional meanings: "Transfer reported" differs from "Payment confirmed"; "Sale accepted" differs from "Bank authorisation completed". Prefer a named counterparty and amount to vague "somebody says they paid you" language. Do not invent an assurance such as "Your request is saved" unless saving is confirmed.

Use consistent editorial decisions for sentence case, Nigerian/British spelling where appropriate, date formats, naira formatting, labels, button verbs and terminology. Keep invoices, receipts, email templates and admin-generated messages aligned. Avoid excessive exclamation marks, artificial friendliness, moralising and vague fintech buzzwords.

Maintain a private copy inventory with old text, context, problem, replacement and any preserved material meaning. Implement replacements in the real source. The inventory alone is not delivery.

## 10. Frontend state, forms and weak-network reliability

For each data section distinguish loading, confirmed empty, ready, stale and unavailable. A failed financial read must never become zero money, an empty dispute queue or “everything is fine.” Show last-known data only with appropriate freshness context and do not permit new financial actions based on obsolete eligibility.

Scope request state, caches and drafts to the correct identity, organisation and resource. Abort obsolete reads or ignore late responses. Test rapid organisation switching and navigating between two sales while requests are delayed. Do not reuse another customer's state during hydration, logout or login.

Make drafts genuinely opt-in, expiring and minimal on shared devices. Namespaces must include the necessary user/business scope. Do not store bank credentials, OTPs, unnecessary customer identity or invoice contents in browser storage. Clear/restrict private caches on logout and account changes. Preserve no sensitive stale service-worker pages across users.

Every important mutation needs a recoverable state: not sent, submitting, confirmed, rejected or outcome unknown. Keep the operation identity and original submitted values across retries, navigation and lost responses. Do not assume a timeout means the server did nothing. Persist only the minimal safe reconciliation reference needed to resolve an uncertain operation.

Retain inputs after recoverable errors, release loading states in all code paths, validate signed upload outcomes, and make refresh/back/duplicate tabs/expired sessions safe. Distinguish signing out locally from verified server-session revocation if connectivity fails. Do not announce remote logout succeeded when it did not.

Derive customer UI visibility from effective server capabilities before exposing optional actions. Test SSR, hydration, delayed/error reads, deep links and cached navigation. Hiding a button is not backend authorisation; front-end code is not a private security boundary.

### A. Pricing must not open with an internal verification failure

Investigate "We could not verify the current fees. Try again before relying on a quote." Determine why the healthy pricing route reaches that state: API base URL/proxy, server routing, missing published configuration, configuration lookup permissions, response contract, accidental authentication dependency, hydration or another concrete cause. Capture the local failure and fix the actual cause. Do not guess that it is a bank problem and do not merely remove the error component.

Published pricing must not depend on a visitor being signed in, a bank provider being configured or a session lookup succeeding. Establish one authoritative, versioned published fee schedule from the actual business-policy source. Render its current public projection server-side or through an equally dependable publication architecture; let the calculator use the same exact amounts, bounds and effective version. An existing approved schedule can remain published during an optional integration outage. Do not invent a new fee, treat a missing schedule as free, or show a previously superseded price as current.

If caching a published schedule is appropriate, define validity and invalidation; publishing a new price must not leave active calculators and new quotes on inconsistent versions. A personalised quote or binding transaction still requires the relevant authoritative server checks. Explain genuine price changes at the quote/commit boundary, not through a permanent public error banner.

Normal anonymous load must show a finished pricing page with the real configured terms. In an intentionally injected complete failure where no valid publication exists, use one restrained "Pricing is temporarily unavailable" state for the affected area and an appropriate contact path. Never turn an actual failure into fabricated successful pricing. This exceptional recovery state is not the default product design.

### B. Public navigation must not display routine session diagnostics

Investigate "We could not check your current session. You can try the check again or sign in below" and "Check session again." Find their originating request and conditions. No cookie/ordinary guest state is not a service error. An expired session, a rejected credential, a transient network failure and an unavailable auth backend are not the same state.

The public homepage, pricing, guides, security and legal pages must render normally for guests without a session-check banner. Do not make reading public content dependent on /me returning success. Use a coherent server-side/initial auth state where practical; suppress redundant client checks and render a stable signed-out navigation when appropriate. Avoid a login-link flicker while repeatedly rechecking an already known state.

Protected pages must still enforce authorisation server-side. An expired session should lead to sign-in and a safe validated return path. A genuine backend outage must not be disguised as invalid credentials or grant access. Where a failed request affects sign-in itself, show a concise error within that form, such as "Sign-in is temporarily unavailable. Please try again." Do not replace permission checks with assumed authentication to remove a banner.

### C. Error design must follow the user's action and its consequence

Classify expected state, background refresh, recoverable read failure, expired session, rejected input and uncertain financial write. No page should show several stacked technical alerts for one dependency failure. Use field errors for fields, section errors for sections, a single accessible confirmation for successful work and a blocking dialog only when the decision genuinely requires it.

Keep technical provider names, stack traces, internal approval status, database errors and configuration keys in private diagnostics. Give customers a clear state and useful next action without a lecture. Preserve genuine uncertainty for money; a pending or failed payment is not forbidden copy. A concise pending status is better than either a wall of explanation or a false success.

## 11. Complete all core supplier and buyer journeys

Inventory and implement the actual supported journeys end to end: registration and verification; organisation setup and membership; customer invitations; customer/business verification; quick sales; invoice-based sales; instalments; review and agreement acceptance; bank authorisation where enabled; goods release/receipt; manual payment reporting; supplier confirmation; allocations; disputes; statements, receipts and reports; account recovery; and support/privacy requests.

For trade lines, drawdowns, extensions, early-settlement offers or similar agreed modules, either finish the explicitly agreed launch capability or document its legitimately optional hidden state. Do not silently discard core work or leave controls that do nothing.

Define state machines, authorised actors, allowed transitions, prerequisites, audit events and explanatory copy. Decide explicitly which flows can operate without a bank mandate. A non-Mono bookkeeping service must have a coherent supported flow; do not simply delete the mandate prerequisite from a contract built around automatic debit.

Cover partial delivery, incorrect quantities, damaged goods, returns, partial and third-party transfers, duplicate proof uploads, disputed portions, overpayment handling, cancellation before/after irreversible steps, expiring invitations and restricted/suspended accounts. Be precise about who can amend an accepted sale and what renewed consent is required.

A customer starting from a shared link should understand who sent it and what the action does, without first understanding Kredit's internal terminology. Links must remain secure, scoped, expiring where appropriate and recoverable without exposing another party's data.

## 12. Financial correctness and accounting

Use exact integer kobo or an equally exact, explicitly defined decimal representation throughout. Validate JavaScript safe-integer boundaries, Go integer overflow, JSON serialization, database ranges, rounding, minimums, maximums, negative values, NaN/infinity and scientific-notation input. Identifiers and references are strings. Never use binary floating point as the source of truth for money.

Document the accounting model and system of record. Where ledger postings apply, enforce balanced journal entries, currency consistency and atomic updates across payment recognition, allocation, balances, fees, ledger, audit and outbox. Distinguish receivables from settlement money. Do not invent a wallet or an unrelated ledger balance to make numbers reconcile.

Guard against duplicate recognition using stable, scoped operation identities and provider-event uniqueness. A manual payment racing a bank collection, two workers handling one job, a webhook racing a poll, or replay after a process crash must produce one correct financial outcome. Do not promise distributed “exactly once” delivery; implement idempotent effects under retries and at-least-once delivery.

Unknown provider outcomes must remain unresolved and reconcilable, not be guessed as success/failure or immediately retried against another account. Distinguish current outstanding amounts, reserved/in-flight collection, reported transfers, cleared payments and reversals. Any owner override affecting eligibility or money must be bounded, explained, attributable and tested.

Store the agreed fee schedule, tax treatment where applicable, due date, grace rules, notices and agreement version with the transaction. Owner pricing changes apply according to an explicit effective-date policy, normally to future commitments; they must not silently rewrite accepted obligations. Do not invent or enable penalty fees not approved for this business model.

Use Africa/Lagos for agreed business scheduling. Store machine timestamps consistently; preserve date-only business meaning. Let the server calculate canonical collection times and return them for review. Test device timezones, daylight-saving locations, midnight, month-end, leap years and clock drift. Never require the customer's phone to determine financial deadlines correctly.

Provide reconciliation by obligation, payment, provider reference and settlement where applicable. Corrections, write-offs, waivers and reversals append controlled records. A settings rollback cannot reverse money or erase audit evidence. Produce tests for mathematical invariants and adversarial event order, not just successful response codes.

## 13. Backend, API and worker engineering

Review domain boundaries, validation, authorisation, service interfaces, error contracts and OpenAPI accuracy. Remove unsafe placeholder implementations, swallowed errors, silent fallbacks, request-context loss, uncontrolled goroutines and accidental mock adapters. Keep architecture proportionate to the existing product and solo-founder operations.

Enforce identity, organisation membership and platform permissions on every route, export, object fetch and worker invocation. Propagate tenant context correctly across transactions and jobs. Use parameterised queries, bounded pagination, body limits, deadlines, context cancellation and predictable timeouts. Redact internal exceptions from customer responses while preserving private diagnostic references.

Use durable outbox/inbox patterns where appropriate. Test scheduling, deduplication, retry/backoff, dead letters, poison events, cancellation, visibility timeouts, graceful shutdown and restart. Jobs should not disappear silently or retry forever at provider expense.

Recheck effective feature state, financial eligibility and consent at the appropriate operation boundary, including just before submitting a new debit. Define what happens if a pause races a submission: do not claim to undo a request already sent to a provider. Record and reconcile that outcome. Pausing new business must not disable valid callbacks, reversals or required dispute processing.

Rate-limit abuse-sensitive endpoints, notifications and expensive provider operations. Separate internal liveness, dependency readiness and customer service availability. A public website must not fail health checks solely because an optional provider is intentionally off.

## 14. PostgreSQL and data readiness

Review every schema and migration involved in the release. Verify foreign keys, uniqueness, money constraints, role grants, row-level security, indexes, transaction isolation, locking order, pool sizes and query plans. Test under the actual restricted application and worker roles, not only the migration owner or a superuser.

Add forward migrations for deployed-schema corrections. Do not edit an already-applied migration and claim existing databases are fixed. Test a clean installation and an upgrade from the verified prior version; account for existing data, lock duration, backfill and constraint validation. Separate schema rollback, forward repair and code rollback; destructive rollback is not automatically safe.

Verify concurrency and deadlock handling around money, settings revisions and owner lifecycle. Ensure lazy connection pools and proxy modes do not leak tenant state between requests. Include timeouts, cancellation and leaked-transaction tests.

Keep production separate from test/demo seed data. Make production seeding refusal explicit. Validate import/export consistency, streaming/pagination, CSV formula safety and private document references. Query optimisation must use representative, clearly synthetic data; report the workload rather than pretending to know future traffic.

Define backup scope for database, object storage, encryption material, applicable configuration and restore dependencies. Verify the available backup tooling and a restoration into an isolated local environment; document configured retention and failure handling without claiming a remote production backup ran. Record measured recovery time and maximum data-loss exposure against agreed objectives. “Backups enabled” is not a restore test. Restoring data must not automatically replay historical customer charges or revive revoked access.

## 15. Security and privacy engineering

Use a documented threat model and an OWASP ASVS-based verification scope with the exact standard version recorded. Include account takeover, cross-business access, malicious supplier/buyer actions, insider misuse, webhook forgery, duplicate collection, owner lockout, secret exposure, unsafe imports and supply-chain compromise. Preserve existing required scanners; scope new active tests to systems owned or explicitly authorised for this task.

Test sign-in/OTP enumeration, expiry and rate limits; session fixation/rotation; secure cookie settings; CSRF and origin handling; MFA/step-up; recovery; object-level permissions; SQL/command injection; XSS; SSRF; CORS; upload abuse; path traversal; signed URLs; security headers; dependency vulnerabilities; and accidental secret exposure in source, logs, client bundles or artifacts.

Provider keys, webhook secrets and recovery material must be write-only in the console after entry. Use a deployment-managed, validated root key or approved secret manager. No public fallback key, implicit production development mode, shared default admin credential or credential-bearing browser storage is acceptable. Include key IDs, environment separation, authenticated encryption context and a deliberate rotation/migration process. Missing keys must fail the relevant secret operation safely without leaking data or corrupting existing encrypted values.

Collect only the data needed for each declared purpose. Protect BVN, bank details, identity documents and private links. Do not collect contact lists or unrelated personal information for collections. Do not send debt details to unauthorised third parties, use harassment-based reminders or manipulate the customer into consent.

Implement appropriate consent and privacy-request workflows, retention/deletion behaviour and audit retention consistently with verified obligations. Do not promise instant deletion of records that must lawfully be retained. Keep diagnostic analytics and session recordings free of sensitive form contents; disable unnecessary tracking until its lawful use and disclosures are established.

## 16. Owner Super Admin: genuine solo operation

Create or complete a distinct platform-owner authority. Public registration, organisation ownership, editable profile fields, email domains and browser flags must not grant it. Unknown roles and permissions should fail closed. Review all existing platform access paths, not just the settings screen.

Secure initial ownership with an explicit one-time deployment/bootstrap process bound to the intended verified identity. Test the actual command or workflow. Do not require the founder to create a fake employee. Provide safe, documented recovery using protected, revocable recovery material and strong identity checks; do not build a support bypass that undermines MFA.

Protect the last effective owner across role revocation/deletion/demotion, account suspension/deletion, expiry and concurrent operations. Ownership transfer must be atomic and leave the rightful owner able to recover access, with appropriate session and permission refresh. Test failure partway through the process.

Support explicit `solo_owner` and `delegated_team` governance. In solo mode, I can confirm my own permitted operating/policy changes with fresh MFA, an impact preview, explicit confirmation where required and an audit reason. Do not describe this as independent approval. In delegated mode, selected high-risk changes require the configured separation of duties. Losing a staff account or failing to read governance must not silently relax the rules.

Integrate founder work into one usable workspace: prioritised attention queue, customer/business lookup, sales/payments/disputes, reconciliations, jobs, provider diagnostics, supported settings, content, notifications, documents, staff access and recovery. Reuse working modules rather than duplicating administrative systems. A searchable technical key/value table is not, by itself, the finished founder experience.

Give the owner console a professional interface, not a collection of developer utilities. Use labels such as "Bank collection", "Email messages", "Pricing", "Website content", "Legal documents" and "Team access" with the current effective state and relevant action. Raw setting keys, JSON payloads and cryptographic fingerprints belong in optional detail views. Generated forms must respect actual types and dependencies; a single free-text input for every value is not acceptable.

Use the existing operational screens as part of one coherent console. Fix their navigation, tables, filters, detail panels and error states as carefully as public screens. Sensitive confirmations should show the exact effect and a short reason field, not intimidating legal prose or repeated warnings about being pre-launch. Keep owner MFA and financial safeguards intact.

## 17. Runtime settings: complete connections, not decorative switches

Maintain a closed, typed registry containing category, friendly name, explanation, type/bounds/enums, scope, default, permission, sensitivity, dependencies, effective-time behaviour, audit policy and actual consumers. Reject unknown keys and inappropriate null/type values. Generate consistent controls and documentation from that contract where practical.

For each supported setting prove this chain:

**Owner control → authorised validated write → durable revision → effective policy → API/provider/worker action → correct customer interface.**

Version edits against what the owner reviewed. Make related updates atomic. Implement before/after review, safe cancellation, error recovery, effective-state feedback and rollback as a new revision. Record the actor and reason atomically with the setting. Never expose encrypted credential payloads as history. Enforce append-only history using restricted database roles and protections, not merely disabled buttons.

Distinguish requested, pending, effective, paused and blocked-by-dependency states privately. Do not label a saved switch effective unless its supported consumers observe it. Specify cross-instance propagation, cache invalidation and worker refresh; emergency pauses must not wait indefinitely on a stale process-local cache. Identify genuinely restart-dependent bootstrap settings honestly.

Cover supported service switches, invitations/registration, sales and invoices, instalments, manual payment workflows, bank authorisations, collection/auto-collection/retries, reports, pricing, limits, notices, reminders, integrations, public content, legal publication and owner alerts. Inventory every current environment/hardcoded operating value as runtime-admin, deployment-secret/bootstrap, unsupported-adapter or genuine code-change. Resolve duplicate sources of truth.

Do not offer switches that turn off core tenant isolation, consent, accurate accounting, idempotency, webhook authentication, owner security or audit integrity. Do not shut off customers' necessary complaint/privacy access because a dispute feature toggle was designed too broadly.

## 18. Providers, bank permission and payment activation

Use the implemented provider contracts and official technical documentation when a contract needs clarification. This does not authorise checking the owner's domain/mailbox or making paid/live provider calls. Inventory which adapters actually exist. A form for Mono, Paystack, Termii, Resend, identity, WhatsApp, storage or scanning is not proof of a working integration. Do not add a new vendor merely because its name appears in a registry.

Supported integration credentials, mode and allowed callback settings must feed actual adapter construction and safely refresh API/worker consumers. Validate destinations against an egress policy; the admin console must not become arbitrary server-side HTTP access. Preserve handling of transactions created under an earlier provider/credential version during rotation or switching.

Make verification server-owned. Record provider, environment, credential fingerprint, test type, timestamp, result and expiry or invalidation conditions. Owner-entered approval notes are separately attributed. A successful connection check is not full product certification, bank entitlement or customer consent. Credential changes invalidate the appropriate prior evidence.

For Mono, verify the intended products: BVN-linked account discovery, authorised mandates, readiness, variable/fixed collection behaviour, approved fallback accounts and Partial Sweep where actually enabled. Account discovery does not itself give debit permission. Never simulate approval or use a global owner toggle as a substitute for a customer mandate.

Implement and test hosted-authorisation redirect/return, customer abandonment, pending readiness, signature-authenticated webhooks, duplicate/out-of-order delivery, polling where supported, reversals, disputed/partial outcomes, notification obligations and reconciliation. Do not declare the sale ready for goods release merely because agreement acceptance returned HTTP 200.

Keep sandbox and live credentials, accounts, events, data, webhooks and workers isolated. No development mock may produce a paid or identity-verified production record. Keep live-dependent controls private/off until real requirements are satisfied. Prepare the onboarding configuration and callback URL construction using the accepted website/base configuration. Test local handlers and permitted isolated adapters; do not claim remote callback reachability or provider approval.

Diagnostics must not charge customers or incur provider fees without explicit approval. Record the difference between a local contract test, provider sandbox test, non-financial live connectivity test and authorised live collection.

## 19. Public launch independent of Mono

Maintain three distinct PRIVATE readiness assessments: **public website**, **enabled customer services**, and **live bank collection**. Each has its own evidence and prerequisites. Missing Mono onboarding must not block a correctly configured public website or genuinely independent customer service.

Refactor startup validation into secure infrastructure requirements and capability-specific dependencies. Production must not silently become development when optional keys are absent. Ensure the website, sign-in and intended non-payment services can start and operate without an active Mono client, if those services do not themselves depend on it.

Do not stop after enabling a brochure site and call the whole platform complete. Prepare the actual owner-operable customer services and finish the code for intended integrations. Keep the specific external activation item visible in the private readiness register rather than turning it into an excuse to leave engineering unfinished.

Off-state features must be absent from customer navigation, forms, onboarding, pricing promises, search, metadata, sitemap and relevant notifications. Return appropriate ordinary not-found/authorisation behaviour for unexposed optional routes without leaking internal reasons. Existing agreements, pending outcomes, payment history, support and privacy remain reachable.

Do not promise invisible services on the homepage, fabricate a temporary bank outage to conceal missing setup, or announce a transaction succeeded when it did not. Hiding internal preparation is compatible with truthful service behaviour.

Do not render those internal readiness assessments as three customer-facing launch modes or banners. The customer-facing service is simply Kredit. It does not need a "We are now live" badge to look finished. A disabled optional feature should be absent; an enabled feature must work. A customer with an existing record retains a truthful way to view and resolve it.

## 20. Complete website redesign, legal pages and tables of contents

Review all public and private content surfaces: landing page, pricing/calculators, product explanations, help, guides, security, contact, terms, privacy, complaints, onboarding, login, customer workspaces, email/notification templates, exports, receipts, metadata, sitemap, error/offline pages and installed-app/offline assets where implemented. Shared layouts and fallback text count. Do not stop after a global text search or after editing the security page.

### A. Remove development-stage copy at its source

None of the following may appear in a normal published public/customer experience:

- "Complete pre-launch draft — legal approval pending" or close variants.
- "Before we go live" and a customer-facing checklist of tests, provider approvals or recovery practice.
- "The final registered company name ... must be added before launch."
- "These terms are not active" because the application shipped an unconfigured document.
- "The final notice will add ... before activation."
- "Before you read all this" followed by meta instructions about summaries and legal text.
- Statements that incorporation is pending, details will be completed later, or the product is a pilot/beta/coming-soon service.

Fix the data and publication model that creates those notices. Fill factual company/contact fields with the accepted information, complete the actual documents and publish the configured versions through the owner workflow. Do not hide a draft warning with CSS while leaving it in HTML, accessibility output, metadata, a downloaded document or a public API response. Do not present a genuinely incomplete contract as effective merely by deleting its label.

Internal draft states, development tests, private review notes and historical documents may retain their truthful terminology where they are not published customer content. Do not remove automated tests or rewrite immutable history to satisfy a keyword grep. Scope forbidden-copy assertions to the actual published surfaces and supported locales. Do not expose internal evidence folders in static assets.

Use stable document metadata. For new publication, set a deliberate effective/publication date from the actual owner publication action or local release configuration, and keep a document version and content hash where used for acceptance. Do not automatically treat the CAC incorporation date as a legal effective date, change the date on every render, backdate approvals or rewrite accepted terms. Keep external-review status private and do not claim external endorsement.

### B. Professional document layout—not a stack of caveat boxes

Create a shared legal/long-form layout. Use a concise title, a short optional description when it helps, one restrained effective-date/version line and the document itself. Use a comfortable reading width (typically about 60–75 characters, adjusted for actual type), legible body text, considered paragraph spacing and a consistent H1/H2/H3 hierarchy.

Remove repeated coloured "in plain words" panels. Summaries are optional editorial aids, not mandatory decoration before every clause. Prefer headings that name the topic. Move routine facts into a tidy metadata/contact block rather than long provisional explanations. Do not put every section into a rounded card or all legal copy into an accordion that prevents useful reading, printing and searching.

Keep terms, privacy and complaints distinct and coherently linked. Security is a factual account-protection page, not an internal launch-readiness report. Present accurate security features without unverified certification badges or absolute guarantees. Avoid marketing typography so large that document titles monopolise small screens.

### C. Design every table of contents intentionally

Inventory tables of contents in legal pages, guides, articles, help and any other long-form surfaces. Do not fix one and leave the others as raw Markdown bullet dumps.

On sufficiently wide screens, use a bounded secondary column for "On this page" with clear hierarchy, restrained text size, real section links and a subtle active-section indicator. A typical 220–260-pixel navigation column can be a starting point, not a forced width that crushes the article. Do not give every link its own card or heavy border. Long labels must wrap cleanly without clipping or losing hierarchy.

Keep sticky navigation below the global header and within its content region. It must not cover a footer or become an inaccessible tall panel. Ensure all links remain keyboard reachable. Do not allow automatic scrolling in the TOC to fight the user's reading position.

On small screens, use a compact accessible "On this page" disclosure rather than a permanent desktop sidebar. Preserve a sensible initially collapsed/expanded state based on length and usability. Support keyboard activation, a correctly associated expanded state, predictable focus and readable touch targets. Closing it must not lose the article position.

Generate section labels, numbering and anchor IDs from the actual heading structure. Support stable unique IDs, nested levels, direct URLs, browser Back/Forward and heading targets unobscured by sticky headers. Use appropriate semantic navigation and aria-current where a current link is represented. Do not duplicate section numbers in the heading and generated marker. A full-page print view should have sensible margins and no sticky overlays; hide or simplify redundant interactive navigation in print.

Test every link in every generated TOC. Include duplicate heading names, long section titles, nested headings, 200% text zoom, hash navigation and back navigation. Visually inspect both legal documents and a representative long guide. Screenshot evidence must show the article and its TOC together on desktop and the mobile disclosure in both states.

### D. Keep real complaint routes; remove placeholders

For the cited privacy paragraph, remove the promise that a final notice will later add contact/address/DPO details. Use the accepted contact and company address now. Do not automatically remove a substantive complaint-escalation route because the surrounding sentence was unfinished. Do not invent a named DPO or a different mailbox.

Illustrative direction: "For privacy enquiries, contact hello@kredit.com.ng. You may also raise a complaint with the Nigeria Data Protection Commission." Preserve the appropriate substance and use the document's established accurate regulator reference. This is an editorial example based on the supplied text, not a claim of independent legal approval.

### E. Publishing, navigation and metadata

Use bounded owner-managed fields for supported homepage sections, FAQs, guides, pricing descriptions, company contacts and document publication. Provide preview, validation, publish and restore-as-new-version; schedule only where implemented. Sanitize content and uploads. No arbitrary executable JavaScript, SQL or shell from a content editor.

Use the accepted domain for canonical links, Open Graph and structured data. Keep private workspaces, invitations, financial records, owner previews and evidence out of indexing and public data feeds. No DNS or mailbox validation is required. Do not invent reviews, transactions or operating history in structured data.

Ensure every call to action points to a useful enabled journey. Remove marketing promises for disabled optional modules at the same time as their buttons. Examples remain labelled "Example" or "Sample" in a clearly isolated demonstration—not disguised as real activity and not accompanied by a site-wide beta banner.

## 21. Messages and support: implementation without external mailbox checks

Complete the implemented communication paths for OTP, invitations, agreements, payment reports, confirmed payments, reminders, pending outcomes, disputes, recovery and owner alerts. Test composition, link generation, dispatch contracts, deduplication, expiry, resend, throttling, delivery-state processing and failure handling using synthetic identities, a local mail sink or permitted test doubles. Do not send to real customers or the supplied company mailbox.

Accept the website/contact values as supplied. Do NOT verify sending-domain authentication, SPF/DKIM/DMARC, mailbox provisioning, remote inbox receipt or delivery to hello@kredit.com.ng. Do not block completion on those excluded tasks. Keep local adapter verification distinct from externally proven delivery; report only what was tested.

The code should represent queued, provider-accepted, delivered, failed and unknown outcomes accurately where the provider exposes them. A provider HTTP success is not automatically customer receipt. Process callbacks idempotently, avoid duplicate reminders and honour relevant preferences, obligations and timezones. No customer-facing messages should carry developer launch notices or raw provider configuration failures.

Review message design, not only templates as text: subject lines, preheaders, heading hierarchy, readable amounts, mobile widths, safe links and clear actions. Match the product's voice. Avoid repeated prose such as "we help you follow the money" or promises that support is staffed 24/7 when I am operating alone.

Integrate a traceable support/complaints queue with case reference, status, owner attention and an appropriate next step. Use the supplied contact consistently. Private test-send controls may be implemented with limits and explicit destination review, but must not be invoked against a real mailbox as part of this task.

## 22. Observability, operations and cost control

Provide structured redacted logs, correlated request/job/provider references, useful metrics and alerts. Monitor unresolved payments, duplicate-event attempts, queue age, failed notifications, stale integration verification, reconciliation differences, backup failures and critical journey errors, not merely CPU and HTTP health checks.

Give me a practical daily/weekly operating checklist and a founder dashboard that links problems to their real records and next actions. Include safe pause/resume, bounded retry and recovery tools with preview, permission, audit and idempotency. Do not make an “admin retry” button that blindly charges a customer again.

Set documented service objectives and resource budgets appropriate to the actual launch environment. Load-test a justified, disclosed synthetic workload and concurrency profile. Examine API tail latency, slow queries, queue processing, memory, connections and provider timeouts. Do not invent production-scale performance from a tiny fixture.

Document the configured hosting/database/storage/email/SMS/provider cost drivers and implemented limits or alerts. Use available contract/configuration information; do not invent prices, purchase services or turn new vendor research into a dependency for completing the local product. Do not add paid dependencies without permission. Prefer maintainable infrastructure that a sole founder can understand and operate. Automation must be observable and recoverable, not invisible complexity.

Prepare runbooks for provider outage, stuck debit, wrong configuration, suspected account compromise, owner lockout, leaked credential, email failure, bad deployment, database failure and backup restoration. Separate reversible operational controls from irreversible external actions.

## 23. Company and legal publication: final product presentation, honest substance

Use the supplied CAC facts consistently. Current public company descriptions must not say incorporation is pending. The registered company name, RC number, address and contact are known inputs, not placeholder tasks. Do not expose personal director records or invent additional regulatory identifiers.

Reconcile the documents with actual product behaviour: supplier/buyer roles, fees, payment permissions, disputes, retention, data handling and support. Keep material fee/risk/consent information where it belongs. Professional copy is not permission to remove a relevant right or misrepresent a financial relationship. Do not add blanket legal disclaimers across unrelated marketing screens as a substitute for carefully written terms.

Prepare complete publication-ready terms, privacy and complaints pages with known factual fields and the proper version/effective-date mechanics. Use the owner-controlled publication model. Keep editable unpublished drafts private. A public page should never describe its own missing future company details or contain your internal approval checklist.

Do not claim that generated documents were externally reviewed, that CAC incorporation grants a financial licence, or that the owner has bank/provider permissions not actually obtained. If a specific substantive legal decision cannot be resolved from supplied material and existing policy, identify it precisely in the private handover, implement the independent work and do not cover the customer site with an "approval pending" notice. Do not manufacture a legal decision to eliminate a blocker.

If an implementation decision genuinely depends on an applicable legal/provider requirement, consult the relevant official source and record the issue privately. Do not expand the task into a general legal research project, and do not reintroduce domain/mailbox verification. Separately identify actual external decisions; do not label every page unfinished because an optional provider is not active.

## 24. Testing: challenge the product, not just the happy path

Run the repository's real lint, type, build, unit, integration, race/concurrency, financial, security, migration, browser and container checks. Inspect their entry points, environment conditions, skips and assertions. Add missing tests for actual requirements. Do not fabricate commands or change the product to satisfy an obsolete test that encodes the wrong behaviour.

For reproduced defects, add a test that fails on the old behaviour before applying the correction when practicable. Freeze clocks and use deterministic fixtures where appropriate. Do not remove assertions, loosen invariants, silently skip expensive suites or increase retries merely to obtain green results. Record flaky results and their correction.

Separate component/API-mocked UI tests, real Go/PostgreSQL journeys under restricted roles, permitted provider sandbox contracts, local release smoke tests, accessibility checks and manual user/device verification. A passing UI test that intercepts every API cannot establish database or provider wiring.

Exercise at least this failure/acceptance matrix, expanding it for all discovered core paths:

| Scenario | Required result |
| --- | --- |
| Anonymous user, merchant owner or ordinary staff opens owner APIs | No platform-owner access or private settings disclosure. |
| Sole founder changes a permitted setting | Fresh authentication, reviewed revision, clear effect and attributable persistence; no fake second account. |
| Two tabs submit different changes against the same revision | One wins; the other gets a recoverable conflict with no partial write. Identical idempotent replays do not create a second change. |
| API/worker restart or another instance reads configuration | Durable, consistent effective behaviour. |
| Missing approved encryption root | Relevant secret operation fails safely; no fallback key. |
| Unknown key, wrong type or attempted manual “verified” status | Rejected without unsafe storage or fake verification. |
| Credential rotation | Safe masking, invalidated old evidence, correct adapter consumption and continued reconciliation. |
| Disabled feature under SSR, hydration, fetch failure or deep link | No newly exposed action or private explanation; existing records still accessible. |
| Public launch without Mono | Approved independent pages/services work; no simulator or hidden production downgrade. |
| Buyer accepts but bank permission is pending | No unjustified ready-to-release or ready-to-debit state. |
| Double tap, timeout after commit, refresh and retry | One logical mutation, recoverable outcome and correct original reference. |
| Manual payment races collection; duplicate/out-of-order webhook arrives | No double recognition or unintended over-collection; reconciliation remains possible. |
| Feature pause races queued/submitted collection | No new prohibited submission after the effective boundary; submitted work still tracked. |
| Partial/disputed payment, reversal, overpayment | Correct allowed state, amounts, allocations and attributable records. |
| Device timezone or month-end changes | Agreed Nigerian business date/time and calculation remain correct. |
| Slow organisation/resource A response arrives after switching to B | No wrongly attributed data. |
| Failed financial reads | Unavailable is not zero, empty or all-clear. |
| Logout, shared device, new user and old service-worker cache | No cross-user private data or draft leakage. |
| Owner expiry/suspension/removal/transfer and concurrent attempts | No accidental loss of the last effective owner; safe documented recovery. |
| Restricted database role attempts history mutation or tenant crossover | Denied with intact records. |
| Old database upgrade and isolated backup restoration | Valid schema/data, reconciled financial state and measured recovery. |
| Small phone, open keyboard, zoom and keyboard navigation | Readable tasks, reachable actions, no obstruction or critical accessibility defect. |
| Expired invitation, uploaded malicious file or forged webhook | Safe rejection without data disclosure or partial financial effect. |
| Email/provider outage and retry | Bounded, observable recovery without duplicate user actions or charges. |

Use synthetic identities and data. Never run destructive fixtures, seeds, load tests or financial experiments against production by accident. Pin the environment and destination before every high-impact test.

Add explicit regressions for the supplied complaints, beyond existing happy-path tests:

| Concrete case | Required local evidence |
| --- | --- |
| Anonymous visit to home, pricing, terms, privacy, security and guides | Normal successful rendered page; no pre-launch text or session diagnostic banner. |
| Auth backend deliberately unavailable while loading a public page | Public content still works; protected data/access remains protected. |
| Missing/expired session on a protected action | Correct sign-in/recovery path without false authentication or a public diagnostic wall. |
| Pricing loaded without a user session and without Mono | Actual published rates and calculator remain correct. |
| Publication of a new pricing revision | Public projection, calculator and newly issued quotes use the right version; accepted sales retain theirs. |
| Genuine pricing source outage with no valid published version | No fabricated fee/free-service assumption; restrained scoped exception state. |
| Legal publication | Known company details, stable date/version and appropriate contact; unpublished drafts remain private. |
| Every table of contents | Working anchors, correct active item/hierarchy, keyboard use, mobile disclosure, zoom and print. |
| Delayed/failed capabilities | No flash of disabled new-use controls; historical records still accessible. |
| Production-like local build after content edits | No old warning hidden in server HTML, client render, templates, metadata or cached built output. |
| Console control changed through real API/database | Saved revision actually changes its supported runtime/worker consumer, not only the rendered toggle. |

Run a forbidden-copy sweep against local rendered routes and relevant output, using the reported phrases plus contextual variants. Inspect manually as well: crude regexes miss similar language and can wrongly flag legitimate transaction or draft-editor states. Do not blacklist meaningful "pending payment" or "failed" transaction statuses. Preserve tests and private historical records.

## 25. Local release build and reproducible runtime

Produce release artifacts from the exact final local source. Run the actual production build and, where supported, a production-mode local preview of the web app, API and workers against isolated PostgreSQL and storage. Verify local startup configuration, API routing, migration runner, owner bootstrap, session handling, static assets and health checks. A development-only mock screen is not a release preview.

Keep production validation for mandatory infrastructure and credentials. Missing optional Mono configuration must not force the application into development or break independent public/services pages. A missing secret-encryption root must fail the relevant unsafe secret operation; it must not select a known default key. Document actual required environment variable names and example values without copying live secrets.

Do not push, open PRs, merge remotely, inspect cloud deployments, configure remote branch protection, change DNS, verify mailboxes or publish production. Local Git is sufficient for checkpoints. Remote access is not a prerequisite. The output is the finished local application, a reproducible local verification record and an actionable release handover.

Test a clean local database installation and an upgrade from the relevant existing schema. Do not test destructive changes on the owner's real data. Prepare forward-repair/rollback instructions with their actual limitations, an isolated restore exercise and secure owner recovery. Fix stale compiled assets and development service-worker caching that can keep showing old content after the code is updated.

Before final handover, review the combined local diff, rerun required checks against the final working tree and confirm every artifact path. If there are uncommitted release edits, identify them and compute a source manifest/digest so test evidence maps to the real tree rather than only an older commit. Do not delete legitimate project CI files just because cloud execution is outside scope; do not depend on them running remotely for local completion.

## 26. Delivery: real local files, screenshots, tests and operating instructions

Keep concise useful documentation under an appropriate private project directory such as docs/launch-readiness/. Avoid generating so many reports that they become a substitute for implementation. Deliver the route/state inventory, settings-consumer matrix, changed-file list, relevant decisions, exact test results, owner operating instructions and local release runbook.

Use precise internal statuses such as IMPLEMENTED_UNVERIFIED, VERIFIED_LOCAL, EXTERNAL_NOT_EXERCISED and NOT_APPLICABLE_WITH_REASON. Public pages must not render that register. Explicitly excluded domain/mailbox/remote tasks are outside scope, not failed tests or reasons to block the finished local product.

Tie evidence to the final local commit plus working-tree diff or source manifest. Record each executed command, environment, exit status and genuine test totals. Distinguish pure unit tests, API-mocked browser tests, real Go/PostgreSQL journeys, simulated provider tests and actual provider calls. If no external provider call ran, say so. A mocked test does not prove external approval or real delivery.

Provide before/after screenshots of the actual application with route, viewport and state. Include a representative interaction recording where useful. Show the revised homepage, pricing, sign-in, terms/privacy with TOC, supplier dashboard, sale/payment journey and owner settings. Do not create AI-generated screenshots and present them as browser evidence.

The final response must state:

1. What changed in the actual local files, including design, copy and backend/root-cause fixes.
2. Which specific owner complaints are fixed, with paths and rendered evidence.
3. Test results, exact environments and any remaining failures/unverified paths.
4. Local source/checkpoint/patch locations and proof that they exist.
5. How to start the finished application and access/recover the Owner Super Admin using the actual implemented commands—not invented flags or credentials.
6. How each supported configuration change becomes effective and which small setup items necessarily remain outside the screen.
7. The state of public-page readiness, enabled customer journeys and bank activation, separately and privately.
8. Any precise remaining owner action, without a blanket caveat replacing unfinished work.

Do not claim that anything was pushed, merged, remotely deployed or externally verified as a result of this local task. Do not say "all complete" when customer routes, tables, forms or configuration consumers remain untested or defective.

## 27. Execution order and a concrete finish line

Work through this in coherent increments, preserving working code and keeping the scope finite:

**First, open and run the local product.** Record the baseline and the exact source of the supplied copy and error states. Confirm the relevant public, signed-in and owner routes are available for inspection. Do not begin with remote tools or pages of abstract planning.

**Second, fix the shared visual and editorial foundation.** Establish the public/workspace/document layout system, improve tables of contents and remove unfinished-language mechanisms. Build representative real pages using the components. Inspect them early so weak design does not spread across the product.

**Third, correct pricing and authentication at the source.** Prove that a normal anonymous visit renders published prices and that guest navigation does not depend on a successful current-session request. Add the exceptional failure tests rather than hide errors indiscriminately.

**Fourth, complete the functional and owner-control work.** Close the known security/settings gaps and trace every supported operating control through persistence, runtime consumers, workers and customer visibility. Finish missing core journeys rather than selectively disabling everything to achieve a clean screenshot.

**Fifth, complete the page/state sweep.** Refine every discovered public, supplier, buyer and owner page; forms, dialogs, lists, long documents, tables, errors, notifications and responsive states. Do not stop after the homepage or a few textual replacements.

**Finally, verify the actual local release.** Run the complete applicable checks, examine the production-like local build, capture evidence, review the final changes and preserve an actionable local handover. Use the final source, not results from an earlier revision.

Definition of done:

- No public/customer preparation banners, placeholders, promised future company details, internal approval states or routine session diagnostics.
- Complete professional published company/legal content using the supplied information, with material meaning and appropriate rights preserved.
- Consistently designed, usable tables of contents and long-form documents on desktop and mobile.
- Pricing and sign-in behave correctly in ordinary and explicitly induced failure conditions.
- Distinctive, cohesive and responsive visual design across public, logged-in and owner screens—not just a new theme.
- Every enabled launch journey works end to end and every supported owner control has a real consumer.
- Financial amounts, tenant boundaries, privacy, consent, idempotency, credentials and history remain correct.
- Database setup/upgrade and local recovery tooling are tested in isolation; required checks pass on the final source.
- Local edits, screenshots and test evidence can be located and reproduced.

Do not manufacture a clean result by hiding core features, weakening tests, pretending a provider is ready, deleting material financial/legal information, or replacing the entire site with static mock-ups. Do not expand the finish line with speculative modules, vendor changes, native apps or cloud investigations.

If a tool constraint stops completion, save the actual patch, evidence and a precise resume file. State exactly what is unfinished and continue all independent work possible. Never report unverifiable "local progress".

**Begin in the open local Kredit project. Implement, run, test, inspect and refine the real product. I want an excellent finished local release—not more public warnings about why it is not finished.**
