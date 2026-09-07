# 6 September product and reliability audit implementation

Base reviewed: `5ce8a16b2d9b79f77db2f05a9a1bdc805095348c`.
Branch: `improvement/audit-2026-09-06`. Review: PR #14.

## Scope and release boundary

This change implements the repository-owned fixes below. It is not a certification,
not a production deployment, and not permission to enable live bank collections.
Actual provider approval (#5), independent security/legal/privacy/operational and
manual accessibility evidence (#7), and administrator-enforced release protections
remain external gates. No existing agreement is silently rewritten. No ledger,
authorization, tenant isolation, mandate, dispute or reconciliation guard is removed.

## Finding-by-finding implementation

| Finding | Implementation | Verification / remaining boundary |
|---|---|---|
| K-01 Hosted authorisation and false readiness | Separate exact agreement acceptance, bank-permission request, approved hosted URL and authoritative status refresh. Pending does not mean ready to release. | Unit state/URL tests; browser acceptance and callback-parameter tests. Actual provider certification remains #5. |
| K-02 Unavailable financial data shown as empty | Typed loading/ready/error resources and runtime decoding on overview, quick sale, buyer sale and payments. | Failed/malformed response tests; no-all-clear browser assertions. |
| K-03 Business switch races | Captured business scope, cancellation and generation guards; old figures clear during switching. | Delayed-response browser test and controller unit test. |
| K-04 Device-dependent timing | Authenticated server preview and opt-in server validation for new quick-sale drafts, fixed to Africa/Lagos. | Go tests cover Lagos, UTC, New York, Dubai, leap/year boundaries and invalid dates. Existing accepted terms unchanged. |
| K-05 Sign-in network recovery | Bounded requests and busy-state cleanup on all outcomes. | Browser aborted-request and resend/expiry scenarios. |
| K-06 Unconfirmed sign-out | Logout response checked, visible failure, private draft/intent data cleared only after confirmed revocation or expired session. | Browser failed-logout scenario; shared-device acceptance requires manual review. |
| K-07 Browser draft privacy | User/business namespaces, validated 12-hour lifetime, explicit opt-in (off by default), shared-device warning, legacy record removal and logout cleanup. | Account/business/expiry/tamper unit tests. |
| K-08 Attention ordering and totals | Entire queue counted before pagination; disputes and transfer checks precede ordinary drafts. | More-than-eight-items regression. |
| K-09 Money display | Exact-kobo helpers on overview and reports; no failed-value-to-zero conversion. | Money, summary validation and exact pricing tests. |
| K-10 Readability and contrast | Dark-text overdue notice, larger working controls, semantic states, consolidated working-screen typography. | Automated axe/reflow evidence; manual device/assistive-technology sign-off remains required. |
| K-11 Safe mutation retries | Persistent per-account operation identity and payload digest, no automatic mutation retry or unknown-operation expiry reset. Buyer actions, quick-sale creation, business creation and supplier transfer decisions and sale-detail money operations use it. | Lost response, navigation, changed payload, duplicate click, malformed response, storage and offline unit tests; browser transfer-decision retry. |
| K-12 First-use authentication | Start-or-sign-in flow, phone-first/email choice, Nigerian number normalisation, masked target, expiry and resend countdown. | Phone/redirect unit tests and OTP browser scenarios. |
| K-13 Truthful debit/dispute copy | Shared wording matches partial/full/no automatic block and in-flight debit caveats. | State/copy assertions; approved legal wording remains a human gate. |
| K-14 Consistent examples | One labelled sample sale, computed balances/progress and working demo links. No fake live activity. | Sample arithmetic and public-route browser checks. |
| K-15 Respectful language | Removed accusations and repayment guarantees from touched journeys; named parties and concrete next actions. | Source/content checks and editorial review; user comprehension still needs pilot interviews. |
| K-16 Coherent visual system | Removed premium overlay files; one working-screen layer, compact headings, exact stationary money, fewer decorative effects. Public identity retained. | Build, responsive screenshots, reduced-motion/axe tests. |
| K-17 Navigation | Home, Sales, Customers, Payments; new searchable sales list; all existing advanced destinations retained. Native focus-trapped account menu. | Menu and navigation browser checks. |
| K-18 Merge/deployment enforcement | Database-required CI guard, retained strict checks, independent UI regression job and evidence artifacts. | Effective branch/ruleset/reviewer/environment configuration still requires an administrator. A workflow file is not proof of protection. |
| K-19 External approvals | Existing live-money restrictions and #5/#7 gates preserved; no approvals fabricated. | OPEN: actual provider, independent security/legal/privacy, production restore/load/on-call and real-device accessibility evidence. |
| K-20 Exact pricing | Typed naira input, optional slider, partial/voluntary examples, integer-kobo fees, supported floor/cap disclosure and bounded scope of quote. | Exact/floor/cap unit tests and typed pricing browser test. |

## Verification commands

Use the pinned versions in `.go-version`, `.node-version` and `package.json`.

```sh
pnpm install --frozen-lockfile
pnpm --dir web check
pnpm --dir web build
pnpm --dir web exec playwright test
# Full regression, database, security and container verification:
bash scripts/ci.sh
```

The product audit workflow publishes synthetic screenshots and traces for review.
Fixtures are fictitious and are not evidence of actual customers or bank outcomes.
Always bind release evidence to the final commit and intended environment; results
from earlier branch revisions do not certify a later revision.

## Behavioural decisions

- A browser callback never grants debit permission. Only the server/provider state does.
- Local storage contains a random mutation identity and a hash, not the mutation body.
- An unresolved mutation older than the conservative retry window needs reconciliation,
  not a new identity. A different form payload cannot silently replace it.
- A payment screenshot or debit alert is not sufficient confirmation of receipt.
- An unavailable permission, balance, customer list or dispute list remains unavailable.
- Currency is integer kobo; dates are shown in Nigerian business time.
- Approved provider host: `authorise.mono.co` for the current Mono adapter. Changes
  require contract verification, code review and tests, not a permissive URL wildcard.

## Remaining hands-on review

Test on intended affordable Android phones, iPhone, shared devices, interrupted data,
TalkBack/VoiceOver, 400% zoom and high-contrast mode. Recruit actual suppliers and
buyers for comprehension tests; do not equate formal education with device skill.
Verify bank-authorisation return routing with the configured provider callback before
certification. Confirm the operational support channel and privacy/legal details.
