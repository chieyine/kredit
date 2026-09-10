# Round two: the ten remaining browser failures

> Historical handover from an earlier revision. Its toolchain limitations,
> test outcomes and three-setting admin inventory are superseded by the
> ongoing [September 9 audit](docs/launch-audit-2026-09-08/FILE-BY-FILE.md).
> Supported provider connections now have admin controls; current changes
> still await consolidated verification.

Go is fully green — `go build`, `go vet`, and all 50 packages pass.

The browser run went much deeper this time. The first run's output was cut off
at test 134 of 161, so a third of the suite had never actually been reported.
149 passed, 10 failed, 3 skipped.

Four of the ten were real product defects. Two were regressions this fix pass
introduced in your copy. One was a bad test I wrote yesterday. The rest were
tests written against wording the product had deliberately moved past.

---

## The four real defects

### 1. "What you owe" linked to sales that do not exist

`/buyer/obligations/[id]` reads an **obligation** id — `GET
/api/v1/buyer/obligations/{obligationID}`. When I rewrote the list component I
had it build the link from the credit **request** id instead. Every row on a
buyer's "what you owe" screen pointed at a record the API would not find, so
tapping any of them landed on "We could not open this sale."

This is mine, from the `WorkspacePage` rewrite, and the test caught it exactly
as it should have. Fixed, with the reason written next to it so it does not come
back.

### 2. The privacy page asked in one language and answered in another

This one is also mine. When I rewrote `product-language.ts` I labelled privacy
request types from the operator's side of the desk — `PORTABILITY` became
"Portable copy". But `/app/settings/privacy` is the screen where a person asks
for **their own** data. So the form offered "Give me a copy I can download", and
the list directly beneath it reported back "Portable copy". Same request, two
vocabularies, one screen.

Both wordings are needed — an operator triaging other people's requests should
not read "Give me a copy I can download". What was wrong was letting them drift.
There is now a single `privacyRequestChoices` array that the person's page uses
for **both** its dropdown and its list, so they cannot disagree again.

**And that turned up a second defect next door.** The operator's console at
`/admin/privacy` was not using the operator wording either — it was rendering
the raw enum: `PORTABILITY`, `CONSENT_WITHDRAWAL`, `IN_REVIEW`. Raw constants,
shown to a person, on the screen where privacy requests get handled. It now uses
the labels that were sitting there unused.

### 3. Turning off customer limits left a blank screen

Same shape as the drawdown gate I fixed yesterday. `/app/trade-lines` now
fail-closes on `features.trade_lines` — correct, because the server default is
off and the old fail-open behaviour offered a form the API would refuse. But
with the feature off, the form simply vanished and nothing said why. There is
now a short block explaining it is switched off, noting that existing limits
still show, and offering the ordinary "Record a sale" route.

### 4. The legal documents' publication state was asserted in three places, one of which disagreed

The legal-publication work in this pass makes the terms and privacy notice
indexable once the published versions are approved and in effect. That is right
— people and regulators have to be able to find them.

`product-quality.spec.ts` still encoded the old rule ("legal pages are always
`noindex`"), and would also have failed on the sitemap assertion right after it.
Rather than flip the expected values, the test now asserts the thing that
actually matters: **the meta tag, `robots.txt` and `sitemap.xml` agree with each
other**, in whichever state the deployment is in. A page inviting crawlers that
`robots.txt` shuts out is the failure worth catching, and it is now caught in
either direction.

---

## Two copy regressions, reverted

Both were drifts toward generic SaaS wording on a product built for traders
whose English is working English.

**The footer.** "Trust and rules" had become "Company", and:

| Was | Had become |
|---|---|
| How we keep it safe | Security |
| How we use your information | Privacy notice |
| Rules for using Kredit | Terms of service |

The originals say what is behind the link. The replacements are the words a
lawyer uses. Reverted.

**The homepage proof section.** Item 04 had been "Kredit does not choose your
customer. — Kredit does not lend money and cannot promise you will be paid. You
decide who takes your goods on credit." It had been replaced with "The balance
is the same on both screens", which mostly repeats item 03.

That swap traded away the single most important thing a trader needs to know
about what Kredit is *not*, for a restatement of something already said. On a
page for people deciding whether to trust a credit product, the boundary belongs
above the fold, not in a FAQ. Restored.

---

## The test I got wrong yesterday

The pricing test I wrote asserted that `/pricing` shows "We could not verify the
current fees". It passed for me and failed for you — because your Go API was
running. Pricing renders on the server, so whether rates load depends on whether
the API is up, and Playwright cannot mock a request the browser never makes. A
test that only passes while a service is down is worse than no test.

It now asserts the invariant that holds either way, which is also the one that
matters on a page with a fee calculator on it: **the page shows rates it has
verified, or it says it could not verify them — never a number it has not
checked, and never silence.** When the outage path is the live one, it still
drives the browser-side "Try again" recovery and checks the exact 0.25% / 0.75%.

---

## Three stale assertions

- **Workspace error copy.** Was "We could not open this page"; is now "We could
  not check customers. Check your connection and try again." — it names what
  failed and the next move. Test updated to the better message.
- **Pagination.** A page holds 20 records; the test seeded 10, so there was no
  second page and no "Next" button. It seeds 25 now, and additionally asserts
  page two does *not* still show the first record.
- **Row markup.** A list of records is now a real list (`<ul><li>`) rather than a
  pile of `<article>` elements, so a screen reader announces "list, 20 items".
  Selectors updated.

---

## Verified

`svelte-check` 0 errors / 0 warnings · production build completes · all four
repository gates pass · no temporary files or links left in the tree.

Please re-run:

```
go test ./...
cd web && npx playwright test --reporter=line
```

The Go side is unchanged since your green run, so it should stay green; the
browser suite is where the work landed.
