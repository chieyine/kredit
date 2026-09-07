# Public launch and administrator-managed availability

Status: approved product direction from the owner on 7 September 2026; implementation specification, not a claim that the controls below are already implemented.

Base: PR #14 at `0a74125a58d5f6e451ca12c9089a926445287ed6`.

## Product decision

Kredit must be publishable on its production domain before Mono onboarding or payment activation. Public website availability, account/service availability and live bank collection are separate decisions. Provider-specific requirements must apply to provider-dependent capabilities, not prevent a non-payment public deployment. A public deployment must not silently enable payments, simulated customer verification or simulated financial success.

The public product must use normal, present-tense service language. Internal preparation, certification checklists and engineering evidence belong in Admin or private operational records, not customer-facing banners. Do not replace an unverified claim with a claim that certification, launch, payment, legal review or security testing has happened.

## Admin structure

Extend the existing operations system rather than creating a second administrator login.

### Availability

Provide independently configurable controls for public-site maintenance, new account registration, buyer invitations, new sales, manual payment recording, new bank authorisations, new bank collections, automatic collection and automatic retries. Pausing new activity must not prevent existing customers from accessing their records, resolving complaints, signing out or exercising privacy rights. Public-site maintenance must not lock administrators out.

Each control must show the requested value, effective value and any actual dependency preventing activation. A green requested switch is not evidence that a capability is effective. Public launch must not require collection readiness.

### Mono integration

Provide provider selection, disabled/sandbox/live configuration, write-only credential entry and rotation, redirect/callback configuration, connection diagnostics and separate Sweep/Partial Sweep settings. Secrets are encrypted through the designated secret store; the UI receives only configured/missing state, masked identifiers and non-sensitive diagnostic results. Never return secret values after saving, place them in browser storage or record them in audit payloads.

A sandbox mode used by operators must remain isolated from production customers, records and workers. Do not route real customer transactions into a simulator. A mode switch must not erase in-flight requests or prevent processing, reconciliation or reversal of already-submitted operations.

Provider approval records can be captured and reviewed in Admin, with reviewer, evidence reference and timestamp. A manually entered label is not provider confirmation, a working credential, a ready customer mandate or transaction success. Separately show observable technical state and recorded human decisions. No toggle may manufacture an approved mandate, a paid balance or a provider entitlement.

### Legal and business information

Manage the actual operator name, service address, support/legal/privacy contacts, document versions, effective dates and publication state through restricted Admin workflows. Separate general website/account terms from terms and disclosures needed for bank collection. Mono onboarding must not block publication of accurate general service information.

Keep unpublished legal drafts out of public navigation and responses. Publish the actual applicable documents once their factual fields and publication decision are recorded. Do not invent company-registration details, claim approval without evidence or remove an inactive-document banner while continuing to present the same draft as an effective agreement. Account/data-processing and payment actions must use the applicable published terms, not a marketing lifecycle flag.

### Operating policy and safeguards

Reuse the versioned business-policy system for fees, collection notices, customer/business/exposure limits, retry limits, emergency pauses and approval history. Show internal operating limits as limits, not as public pilot messaging. Existing accepted offers retain their recorded terms.

Authentication, administrator permissions, MFA for sensitive changes, tenant isolation, CSRF protection, webhook authentication, exact-kobo accounting, idempotency, consent, reconciliation and immutable history remain enforced. They are not optional launch blockers and must not have disable-safety switches.

DNS, the initial database connection, the secret-encryption root, server TLS and initial administrator bootstrap must exist before Admin can operate. Keep that small bootstrap boundary explicit. Subsequent supported application configuration is managed through Admin; settings requiring a restart must show that truth rather than pretending to take effect immediately.

## Server and database requirements

Persist settings in a server-owned, versioned database representation with strict validation and least-privilege access. Record actor, reason, redacted before/after values, version and timestamp. Use current-version checks and preserve the existing independent-approval requirement for high-impact financial changes. Separate read, propose, approve and secret-rotation permissions as needed.

Apply effective capabilities on the server and in every worker immediately before the relevant operation. Hiding a button is not enforcement. Settings errors must fail closed for new financial actions without turning the public website into an error page. Preserve reconciliation, callback handling and necessary customer/support access when new collection is paused. Test scheduled work, retries, concurrent saves and configuration rollback.

Do not replace required bootstrap credentials with defaults or fake evidence strings to pass production validation. Refactor startup validation into secure infrastructure requirements and capability-specific requirements. A deployment with no Mono configuration must start in its explicitly selected non-payment operating state.

## Public content rules

Remove public-facing `pre-launch`, `before we go live`, `coming soon`, `pilot`, `beta`, `prototype`, testing-only banners and descriptions of missing internal approvals where they describe the service lifecycle. Sweep the landing page, navigation/footer, security page, legal pages, pricing/FAQ, onboarding, account empty states, emails, notifications, SEO, social metadata and cached/offline assets.

Use ordinary service labels such as `Create an account`, `Sign in`, `Your sales`, `Payment history` and `Help and complaints` only where the corresponding journey is available. Hide an unavailable optional payment method, or explain its availability at the point of use; do not invite the user into a broken flow or falsely attribute setup work to a temporary bank outage.

Keep examples labelled as examples. Keep zero balances and genuine empty states. Keep operational states such as pending, failed, reported and confirmed where they describe an actual transaction. Do not publish invented customers, fabricated testimonials, invented volume or simulated transactions as live activity. Automated tests and internal audit evidence are retained.

## Acceptance matrix

1. Production public routes render with Mono keys, provider certification and collection features absent. Non-payment launch does not expose development adapters, mocks or secrets.
2. The intended public pages contain no lifecycle/pre-launch banners or internal readiness checklist. Examples remain clearly examples. Current security/layout protections remain enabled.
3. Authorised administrators can persist supported availability and configuration changes and see requested versus effective state. Unauthorised users, stale revisions and invalid inputs are refused.
4. Public launch, registration, sales and manual recordkeeping can be controlled independently from bank authorisation and collection. Tests identify any remaining prerequisite for each capability.
5. No key or approval reference alone can produce customer-authorised bank access or enable an unavailable provider feature. No live/sandbox data mixing is permitted.
6. Existing payment eligibility, notice, idempotency, tenant, accounting, webhook and concurrency tests remain enabled. Pausing new collection does not stop reconciliation of submitted work.
7. Legal publication uses actual configured information and versioned acceptance. Unpublished drafts are not presented to visitors as effective documents.
8. Final evidence is bound to the exact candidate commit. Public-domain deployment and live collection activation are separately recorded; neither is inferred from copy changes or a green workflow.

## Current implementation progress

The first code change removes the security page's public pre-launch section and future-tense external-review claims, replaces unsupported absolute promises and preserves the page layout. A source regression is supplied for that change. The broader backend/database/Admin refactor and full public-content sweep above are not complete in this initial increment. Existing production validation is unchanged by the initial copy commit.
