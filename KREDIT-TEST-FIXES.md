# What the first full test run found, and what I changed

> Historical handover from an earlier revision. Its toolchain limitations,
> test outcomes and three-setting admin inventory are superseded by the
> ongoing [September 9 audit](docs/launch-audit-2026-09-08/FILE-BY-FILE.md).
> Supported provider connections now have admin controls; current changes
> still await consolidated verification.

Your run was the first time this has been true: `go build` clean, `go vet` clean,
50 Go packages passing, `svelte-check` clean, production build clean, and the
**whole** 161-test browser suite executed. `docs/launch-readiness/TEST-RESULTS.md`
records that only 11 public browser tests had ever been run before — "all
signed-in browser suites … are not certified by these results." So most of these
failures are not regressions; they are the first honest look at tests nobody had
executed.

Eight failures total: two Go, six browser. All eight are fixed.

---

## Go — `internal/config` (2 failures, both fixed)

### One was my mistake

When I split production validation into "infrastructure always required" and
"capabilities required only where switched on", I moved
`FEATURE_APPROVED_RETENTION_POLICY` under `FEATURE_REAL_COLLECTIONS`. That is
wrong. A deployment holds people's records — names, phone numbers, what they owe
— from its first sale, whether or not it ever debits a bank account. How long
those are kept is not a collections decision.

Retention is back in the always-required half, with a message that says why. The
money-specific pack (DPIA, pen test, launch approval, the pilot exposure limits)
stays with collections, where it belongs and where its own error text already
said it belonged.

### The other was a brittle test

`TestProductionRequiresApprovedRetentionAndPilotEnablement` and
`TestMonoProductionRequiresCertificationAndLiveCredentials` both built a nearly
empty `Config` and asserted on whichever error came out first. That made them
tests of *the order of the checks*, not of the gates they were named after —
which is why moving one unrelated check to the top of the block failed two tests
that had nothing to do with it.

There is now a `productionBase()` helper: a production config whose
infrastructure is complete and whose capabilities are all off. Each gate test
starts there and switches on the one thing it is about, so a test named for a
gate fails for that gate. Three tests now read clearly:

- `TestProductionRunsWithoutAnyExternalProvider` — new, and the one that matters
  today: production is valid with no bank, no identity provider, no messaging
  provider. That is the launch change, now asserted rather than assumed.
- `TestProductionAlwaysRequiresApprovedRetention` — including that enabling it
  without its written reference still fails closed.
- `TestLiveCollectionsRequireBoundedPilotEnablement` — the pilot limits, reached
  through a config that actually gets there.

The Mono test's last assertion named retention only because that happened to be
the next check to fire; it now asserts what the test is actually about — that
satisfying Mono's own gates hands off to the collections readiness gates.

---

## Browser — 6 failures

### 1. `health.spec.ts` — homepage CTA (fixed)

The test asserted a link named "Add your first sale". The homepage now says
"Open your account" and "Try a sample sale". That copy change is right: the
homepage is read by someone with no account, and a button promising a sale that
actually lands you on a sign-in code screen is a small lie. The test asserts the
real actions now.

### 2. `business-settings.spec.ts` — pricing (fixed, plus a copy regression)

Pricing moved to server-side rendering in this pass, so the fee rates are in the
HTML for a reader with no JavaScript and for a crawler. That moved the request
out of the browser — and the test was mocking it there with `page.route`, which
can no longer see a request the page does not make.

**A copy regression came with that move and is now reverted.** The error had
become "Pricing is temporarily unavailable." The original said "We could not
verify the current fees. Try again before relying on a quote." On a page with a
fee calculator on it, telling the reader not to rely on a quote is the whole
point. The better sentence is back, in both the server and client paths.

The test now drives what a visitor actually meets: the server could not reach
pricing, the page says so instead of printing a number it cannot stand behind,
and "Try again" — which *does* run in the browser — recovers and shows 0.25% /
0.75%. It covers both paths where it used to cover one.

### 3. `admin-workflows.spec.ts` — approvals (fixed, plus a copy fix and new coverage)

The test clicked "Send for a second person to approve". That copy is gone,
because you are the only admin: the button now reads "Record this proposal" for
a solo owner and "Send for approval" otherwise.

**A real problem turned up next to it.** When the governance endpoint cannot be
read, `governanceMode` falls back to `'unavailable'` — and the page was then
telling the owner "Every correction needs a second administrator to approve it."
That is a statement of fact that may be false, on a screen about changing
someone's balance. It now says the approval rule could not be read and will be
applied when the decision is made.

The old test kept its meaning (it is about the two-administrator regime, so it
now mocks that mode explicitly instead of relying on a fallback). And I added
the test that was missing: **the solo-owner self-approval path had no coverage
at all**, despite being the one you will use every day. It is the only route by
which a balance changes without a second pair of eyes, so the new test pins down
that it cannot happen quietly — the approve button stays disabled until a written
reason is entered, and the reason reaches the server.

### 4. `product-flows.spec.ts` — draft cancellation (fixed)

The test clicked "Delete this sale"; the button says "Cancel this draft". The
product is right — a draft is cancelled, not deleted, and the record persists.
Test updated.

### 5. `product-flows.spec.ts` — drawdowns (fixed, plus a silent empty screen)

Selling from a customer limit used to be shown by default and hidden only if the
platform explicitly said `drawdowns: false`. That is fail-open against a feature
whose server-side default is **off**, so the form was offered for something the
API would refuse. It now shows only when the capability is on — correct, and the
cause of the test timeout, since the test never enabled it.

**But turning it off made the form vanish with no explanation**, which reads as a
broken page. There is now a short block in its place saying the feature is
switched off for this account, that the balance below is still current, and
offering the ordinary "Record a sale" route.

### 6. `audit-product-journeys.spec.ts` — accessibility on `/` (fixed)

axe found `aria-hidden-focus`, serious, on `.hero-product-art`.

The homepage hero contains a mock of a sale — a fake customer, a fake balance, a
fake timeline. It is correctly marked `aria-hidden="true"`, with an `sr-only`
line above it saying "The picture below is a made-up example of a sale, not a
real account." That part is good work.

But inside that hidden block sat a real link: `<a href="/demo">Try this example
→</a>`. So a keyboard user could Tab into it while a screen-reader user could
not perceive it at all — focus lands on something that, as far as the
accessibility tree is concerned, does not exist. That is WCAG 4.1.2, and it is
the kind of defect you only find by running the tests.

The link is removed and the block is now genuinely decorative. Nothing is lost:
"Try a sample sale" is a real, accessible button in the same hero, two elements
away, and the picture was offering the identical destination a second time. The
now-unused `.window-bottom a` rule went with it.

**I then swept all 129 Svelte files** for the same defect class — nesting-aware,
not a plain grep — looking for any focusable element inside any `aria-hidden`
container. This was the only one.

#### One thing I changed on a wrong guess

Before I had your output, I predicted the failure was contrast and changed the
header's "Try it free ↗" arrow from `#f58a6d` to `#ffb59e`. That was not the
failure: the arrow is `aria-hidden`, so axe excludes it from contrast checks.

I have kept the change, but you should know it is a judgment call and not a bug
fix. The fact behind it stands on its own — `#f58a6d` on the button's `#2738d6`
is 3.33:1, and axe skipping decorative glyphs is a limit of the tool rather than
evidence the arrow is easy to see. `#ffb59e` is 4.7:1 and the same warm accent.
If you would rather keep the original, it is one line in `web/src/app.css`.

#### Why the other axe test did not catch it

`product-quality.spec.ts` runs axe across 11 public routes but only at **390px**.
This test runs **390 and 1440**. Anything that only appears on a desktop layout
is invisible to the wider sweep. Worth widening later; I have not done it today,
because adding a broad new gate I cannot execute here could hand you a failing
suite on launch day.

---

## Verified after these changes

`svelte-check` 0 errors / 0 warnings · production build completes · all four
repository gates pass. The Go changes are again **parse- and format-checked
only** — please re-run `go test ./internal/config/...` (and ideally `go test
./...`) before you deploy.
