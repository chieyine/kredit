# Launch audit repairs

Current completion and essential-check evidence are in [the implementation record](../platform-improvements/IMPLEMENTATION.md) and [the check record](../platform-improvements/ESSENTIAL-CHECKS.md). The audit-stage notes below describe the earlier snapshot.

The original audit remains unchanged. These are implementation statuses, not verification or a launch approval. No tests, builds, database migrations or live provider actions have run during repairs.

| Finding | Problem | Repair status |
|---|---|---|
| F001 | Sendly is not integrated (launch blocker) | implemented; verification pending |
| F002 | Production stack contradicts startup requirements (launch blocker) | implemented; verification pending |
| F003 | Caddy gateway configuration is invalid (launch blocker) | implemented; verification pending |
| F004 | PostgreSQL 18 volume is mounted at the old location (launch blocker) | implemented; verification pending |
| F005 | Generic connector interfaces are not implemented vendor integrations (launch dependency) | provider selection and contract required; integration incomplete |
| F006 | Deployment command does not load compose interpolation values (launch blocker) | implemented; verification pending |
| F007 | Successful bank debits cannot be posted to the production payment ledger (critical) | implemented; verification pending |
| F008 | All failed Mono debits are permanently non-retryable (high) | implemented; verification pending |
| F009 | Sub-NGN-200 balances are eligible for an unsupported Mono debit (high) | implemented; verification pending |
| F010 | A lost mandate-creation response has no durable recovery workflow (high) | implemented; verification pending |
| F011 | Public proxy chain groups sign-in throttling by Vercel egress address (high) | implemented; verification pending |
| F012 | Account recovery cannot reach its evidence step (launch blocker) | implemented; verification pending |
| F013 | Admin financial changes cannot resolve their target obligation (high) | implemented; verification pending |
| F014 | Ownership transfer cannot read the recipient account (high) | implemented; verification pending |
| F015 | Admin directories and money totals can hide real records (high) | implemented; verification pending |
| F016 | Granting an admin role can commit while the API reports failure (high) | implemented; verification pending |
| F017 | Financial review can miss discrepancies and close unresolved cases (high) | implemented; verification pending |
| F018 | Supplier settlement setup has no production verification path (launch blocker) | provider selection and contract required; integration incomplete |
| F019 | Billing setup records a choice without establishing billing (high) | unsafe readiness claim fixed; contracted billing integration still required |
| F020 | Buyer onboarding cannot resume pending identity verification (high) | implemented; verification pending |
| F021 | Collection eligibility loses tenant scope before reading the debt (launch blocker) | implemented; verification pending |
| F022 | Opening a sent sale breaks the next buyer action's saved version (launch blocker) | implemented; verification pending |
| F023 | Plain-text agreement labels kobo as naira (high) | implemented; verification pending |
| F024 | Customer limits and purchases lack the required database scope (launch blocker when enabled) | implemented; verification pending |
| F025 | The management scorecard reads hidden financial records as real zeroes (high) | implemented; verification pending |
| F026 | Current Mono dispute automation events are rejected (high) | implemented; verification pending |
| F027 | Business money settings can silently target the wrong business (high) | implemented; verification pending |
| F028 | Buyer invitation acceptance records consents the page never presents (high) | implemented; verification pending |
| F029 | The advertised payment page sends the customer in a circle (high) | implemented; verification pending |
| F030 | Dispute documents cannot be properly submitted and reviewed (high) | implemented; verification pending |
| F031 | Invoice attachments are lost or inaccessible in the sale workflow (high) | implemented; verification pending |
| F032 | Sending a draft can ignore the edits currently shown (medium) | implemented; verification pending |
| F033 | The report mislabels 31–60-day debts as 60+ days overdue (medium) | implemented; verification pending |
| F034 | The robots rule accidentally blocks the public contact page (medium) | implemented; verification pending |
| F035 | Published sharing images still advertise the old domain (medium) | implemented; verification pending |
| F036 | The recovery workflow cannot locate its own backup (high) | implemented; verification pending |
| F037 | Release certification calls a removed SQL generation workflow (high) | implemented; verification pending |
| F038 | Document uploads use an unsupported Cloudflare R2 header (high) | implemented; verification pending |
| F039 | Local SSH credentials can enter the API build context and cache (high) | implemented; verification pending |
| F040 | Existing tests bypass several broken production paths (high assurance gap) | deferred until repairs are complete; no tests run |
| F041 | Required privacy and retention approvals are not evidenced (launch dependency) | technical inventory updated; approval evidence required |
| F042 | Operating instructions refer to obsolete schema and legal versions (medium) | implemented; verification pending |

## Remaining outside-code inputs

The contracted identity/business/authority verification provider, SMS/WhatsApp provider, seller-settlement arrangement and fee-billing provider are still unspecified. Generic connector interfaces cannot establish these integrations. Billing preferences now remain pending, and the public balance link makes no checkout claim. These changes prevent false readiness; they do not supply the missing payment services.

Actual provider account permissions and certification, privacy lawful-basis/retention decisions, transfer assessments and launch approval references must come from their responsible owners. No evidence has been invented. The technical inventory includes the new recovery and document-reference fields.

## Verification still deferred

After the remaining repairs and provider decisions, use only focused checks of the production runtime wiring, restricted database roles, provider request/response contracts, frontend type/route integration and the affected money/security flows. Existing passing mock tests cannot prove these fixes. Tests and migrations have not run.

## Sharing image repair

The built-in image-editing tool replaced the domain in `web/static/og.png`; `web/static/og.jpg` is its JPEG counterpart. Final prompt: “Replace only the bottom-left domain text kredit.com.ng with exactly kredit.ng; preserve all other text, logo, colors, typography and layout.” Both website assets are stored in the repository at 1200 × 630.

## Follow-up source review

Provider audit events now use a system actor and return persistence errors to the durable webhook worker. Verification checks preserve pending accounts, fence concurrent updates, and select the business named on the sale. Account creation returns before calling the identity provider; the saved account offers an explicit status check. Both storage implementations retain pending verification. An unconfigured identity provider cannot create unusable provider-bound requests. Admin recovery checks provider references against the original subject and records the review evidence. Admin directory and money reads install request identity on their own database transaction; each privileged projection requires its actor to match that identity. Account screens identify expired verification evidence rather than relying only on cached profile status.

The new invoice download UI imports its response helpers, and dispute attachments remain restricted to their dispute participants. Customer-facing source no longer contains the old domain or a coming-soon/waitlist message. This is source review, not proof of deployed behavior.

Mono's [current Sweep guide](https://docs.mono.co/docs/payments/direct-debit/mono-sweep/integration-guide) was checked again on 12 September 2026: authorization uses the initiate endpoint and hosted `mono_url`; collections require `ready_to_debit`. The implementation retains those boundaries. Sendly evidence and provider-specific uncertainties are recorded in [PROVIDERS.md](PROVIDERS.md).
