# Audit completion — 7 September 2026

Scope: PR #14, continuing the 6 September product audit. Implementation revisions: `7b290c2d2a118292c8d472802380fc696398e9b4` and `3030b3b4cb63b620e19b18819d31811f8075cd43`. This is engineering work on the review branch, not permission to merge, deploy, collect money or declare provider/legal certification.

## Corrections in this pass

- Made the notification quiet-hours regression deterministic through the existing injected clock. Added exact start/end, Lagos midnight, daytime-window, year-rollover and disabled-window boundary cases.
- Made quick and advanced sale drafts genuinely opt-in. Only a previously opted-in, unexpired account/business-scoped draft can restore consent. Legacy default-on v2 records are discarded, not grandfathered as consent. Storage explicitly allowlists goods, amount, date, scope and expiry; no customer identity or invoice is retained.
- Protected sale-detail financial mutations as well as the quick-sale and central payments journeys. Retried writes retain their operation identity; manual payments also retain their original paid-at timestamp across navigation. Failed writes leave inputs intact and recoverable. A reported-transfer decision requires the same bank-check confirmation dialog on both entry points.
- Distinguished unavailable payment/schedule/claim/collection reads from verified empty records in sale detail. Unavailable collection eligibility cannot enable a debit. Responses from an obsolete route cannot overwrite the current sale view or clear a newly opened sale's payment fields.
- Removed unsupported success or delivery guarantees from error recovery, onboarding, notification preferences and low-data settings.
- Protected guide filters against edits before hydration. Preserved no-match recovery and category/search interactions.
- Made feedback resilient to network and storage failure, and left its success confirmation visible until dismissed. Feedback state is account/business scoped.
- Aligned the interactive demonstration with the shared sample sale, exact-kobo partial payment, 24-hour grace and seven explicitly separate steps, including sample bank permission before goods release. Demonstration activity is labelled illustrative, not live.
- Updated obsolete selectors and copy expectations without deleting the existing behavioural checks. Added regression coverage for draft consent/revocation/isolation, legacy migration, feedback recovery, guide filtering, sale-detail payment replay across reload, failed financial reads and transfer bank checks.
- Run CI browser checks against the built production preview, with service workers blocked for deterministic synthetic routing, no automatic retries, and exclusive tests forbidden in CI. The full suite remains enabled.
- Added an isolated legal-document activation regression to Product audit. It uses explicitly fictional CI-only company/contact details, separate result directories and a separately launched preview. No live legal settings or approval records are changed.

## Evidence policy

Local checks on the main implementation: Svelte/TypeScript check passed with zero errors and zero warnings; production build passed; 31 selected pure reliability, privacy and money tests passed. The final corrections also passed local type checking with zero errors and zero warnings. Local browser navigation is blocked by the execution environment, so these do not constitute a local browser pass. Browser outcomes must come from the source-bound GitHub Product audit and CI workflows.

The original `7027e12` full frontend report contained 116 passing, 25 failing and four conditional tests. On `fc0ce499`, the expanded suite reached 147 passing and three failing; those remaining failures were precise-selector/wording mismatches and are corrected in the final implementation. These are historical reports, not the completion result.

The authoritative completion result is the latest PR-head workflow run, with `candidate-commit.txt`, `source-commit.txt`, JSON/HTML test report, screenshots and traces in its artifact. Never substitute an earlier passing revision for current-source verification. The PR conversation records the final observed result.

The ordinary frontend suite conditionally skips three real-stack tests and one active-legal test. These are not silently counted as passes. Phase 3's real-stack-smoke job executes the three financial browser journeys against the real Go API, worker/database roles and isolated PostgreSQL. Product audit separately executes the active-legal test against its fictional configuration. Count each test once and distinguish these environments from the ordinary API-mocked production-preview regressions. None establishes bank-provider certification or actual legal approval.

## Cleanup and unchanged gates

Temporary source snapshot/transfer files and the one-use write-enabled transport workflows were removed by the implementation commits. They are not part of the final source tree. Ordinary verification workflows remain read-only.

K-18 remains an effective administrator-enforced protection requirement, not a claim based on a workflow file. K-19 remains actual Mono sandbox certification (#5), independent legal/privacy/security/operational approval (#7), and intended-device/assistive-technology/recovery verification. No production settings, bank credentials or live-money feature flags were changed. Main remains outside this implementation branch.
