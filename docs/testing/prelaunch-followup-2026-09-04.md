# Prelaunch editorial and engineering follow-up — 4 September 2026

The first audit's 76 focused files included important financial, permission and shared-interface code. They were not the whole product. The remaining inventory entries were not equivalent to manual reviews, and passing checks did not establish that the writing was good. This follow-up does not claim an exhaustive review of every file or a launch sign-off. No agents were used.

## Editorial finding and changes

The old 100-article library expanded topic seeds into almost identical eleven-section articles. Topic-specific sentences appended to repeated paragraphs defeated the exact-paragraph duplicate check. Its checks required 1,500 words, ten sections and multiple sources per article, which rewarded padding. Publication dates were generated backwards from an arbitrary date. Those dates did not document publication, and the blanket research claims were unsupported.

The current library contains twelve separately written guides (4,690 words in total), with distinct worked examples, practical questions and relevant cross-links. Unwritten topics remain in `docs/content/topic-backlog.md`, outside public routing. No publication date is assigned before publication. The modification date records this editing pass. Reading times follow the actual length. The article template no longer shows a word-count badge or claims every guide used official research. Further-reading links appear only where a guide provides them.

The content audit now detects repeated substantial sentences, including repetition hidden by appended topic text. It checks content presence, reading metadata, dates, discoverable routes and links without imposing an article-count or word-count quota. These checks help detect defects; they do not prove editorial quality or human authorship.

The FAQ incorrectly described a payment-triggered base fee. It now agrees with the implemented activation-triggered fee and points to current pricing instead of hardcoding rates. Public copy now distinguishes conditional bank collection from a repayment guarantee, explains permitted staff access, avoids implying Kredit holds customer money and describes the demo as a guided example. The security page retains the outstanding prelaunch review requirement.

## Presentation

The homepage hero now includes its padding in its height calculation, reducing excess desktop whitespace. The guide index uses the actual guide count, correct singular labels and the Kredit initial. Related-guide cards identify the destination's category. Publication/research badges were removed rather than decorating thin content with unsupported authority.

## Backend changes

Report reads and exports accept request context throughout the HTTP-to-snapshot path. Credit-list reads do the same, including mandate and provider-event callers. The database read timeout now derives from the caller's context. Cancelled reports stop before accessing the financial source. This is a change to the current interfaces; old signatures are not preserved for compatibility.

This narrows, but does not close, the earlier cancellation finding: other repository reads and writes still use background contexts. No tenant-isolation migration is claimed in this follow-up.

## Verification

- Full Go package tests passed (`go-all-final.log`). Database-dependent cases were separately exercised as described below.
- The PostgreSQL durability and cancellation regressions passed against the isolated seeded audit database (`postgres-context-verified.log`). The synthetic cluster was stopped afterward.
- Unsuppressed Go lint: zero issues (`lint.log`).
- Svelte diagnostics: zero errors and warnings (`check-final.log`). Final source also built successfully with the Node adapter (`build-verified.log`).
- Content audit passed for twelve guides. A regression exercise copying a paragraph and appending topic-specific prose was rejected as intended (`repetition-regression.log`).
- All twelve guides loaded at 1440px and 390px without horizontal overflow. Eight screenshots were saved; the homepage, guide index and article previews were visually inspected (`visual.log`, `*-desktop.png`, `*-mobile.png`).
- Targeted production-browser suite: 14 passed, one configuration-specific legal-activation skip (`browser.log`). This includes the public accessibility sweep, responsive navigation, guide discovery, fee presentation and indexing boundaries.

Evidence is under `.tmp/prelaunch/`. An initial full Go test attempt exposed two test callers needing the new context argument and a sandbox listener restriction; those were corrected before the passing rerun. The initial database restart used the default occupied port, failed without affecting that existing server, and was corrected to the dedicated audit port before the passing database tests.

## Work that remains open

- Broad runtime database policies still weaken tenant isolation for several financial tables. The current repositories need consistently scoped transactions and dedicated cross-tenant regressions; removing policies alone would break their lookup/write paths. This is a current architecture problem, not a reason to preserve legacy behavior.
- Cancellation propagation remains incomplete outside the updated report and credit-list paths.
- Most private-interface browser tests use mocked APIs. Real provider, authenticated end-to-end, operational recovery and external launch reviews are not certified by those tests.
- The original file register remains a historical snapshot. This follow-up does not relabel mechanically checked files as manually reviewed or declare every screen perfect.
