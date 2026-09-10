# Owner Super Admin and private service activation

Status: product requirements updated from the owner's instructions on 7 September 2026. This is an implementation contract, NOT a claim that this console or its server/database changes already exist.

Parent audit: PR #14 at `0a74125a58d5f6e451ca12c9089a926445287ed6`. Follow-on implementation: PR #15. This revision supersedes the earlier unconditional requirement for a second platform administrator and the suggestion to show optional-feature availability explanations to customers.

## Owner's operating model

Kredit is intended to operate with its founder as the only employee initially. One verified platform owner must be able to administer every supported product and operating setting without editing code, creating a second employee account or waiting for another internal approver. Hiring staff later must not require an access-system redesign.

Public production deployment, customer-service availability and live bank collection are independent decisions. The owner controls activation privately. Customers see the enabled product, not an internal rollout checklist, missing integration notices, disabled-feature badges or approval status.

Public deployment must not silently enable payments, simulated identity verification or simulated financial success. Customer-facing language describes the services actually provided; internal activation details are not public marketing content.

## 1. Platform owner and solo operation

Introduce an explicit platform-owner/Super Admin authority, separate from supplier organisation ownership and ordinary platform staff. The existing supplier `owner` role MUST NOT become a platform owner. Extend the current authentication and operations system; do not create an unrelated administrator login or use a client-side flag as ownership.

Bootstrap ownership through a secured, one-time deployment process tied to the intended verified account. Never grant ownership to the first public registrant or infer it from a client-submitted role, email domain or display name. Protect the last active owner against accidental deletion or demotion. Provide owner-controlled, audited ownership transfer and a tested recovery process using protected recovery material; do not require an employee to exist to recover the owner's account.

Support two explicit internal operating modes:

- `solo_owner`: the verified owner reviews and applies their own permitted configuration changes. Sensitive changes require fresh MFA, an impact summary, explicit confirmation and an audit reason. Record this as an owner-authorised decision, not as independent approval. Existing external provider/customer requirements still apply.
- `delegated_team`: the owner grants granular staff permissions and can require an independent internal approver for selected actions. Staff cannot grant themselves ownership or switch governance modes. The owner chooses the mode; do not automatically weaken governance when staff accounts are removed or suspended.

The owner's authenticated confirmation replaces an unnecessary second-employee step, not validation, customer consent, accounting rules or external provider capability. No duplicate/fake staff account is an acceptable workaround. Owner self-authorisation must be implemented in API permissions, domain validation, database constraints and row-level policies together, not just by exposing an approval button.

## 2. What customers see

For optional modules, the default off-state is hidden:

- Remove off-state modules from navigation, action buttons, onboarding, forms, search, pricing promises, contextual help and automated marketing.
- Do not display `coming soon`, `not enabled`, `pending approval`, `provider not configured`, pre-launch badges or greyed-out placeholders.
- Direct requests for a never-exposed/off optional module get a normal not-found response without leaking the flag key, vendor credentials, internal reason or rollout status. The corresponding new-action API remains protected on the server.
- Public SSR output, client route data, search metadata, sitemaps, emails, push messages and offline/service-worker caches must agree with the effective customer experience. Do not serialize the private settings catalogue or readiness diagnostics into public hydration data.
- Only render services the owner has enabled and the server can actually provide. Never show an apparently working collection action backed by a simulator or a no-op.

Turning a module off stops new use; it does not erase existing customer records, pending payment outcomes, disputes, consent withdrawals, receipts or applicable terms. Existing commitments need truthful states and a usable support/resolution path. Hiding rollout details is different from hiding what happened to a customer's money. Account access, privacy requests, complaints and sign-out must remain reachable when new business activity is paused.

Provider-readiness details, requested/effective flags, errors, deployment status and operational reasons are visible only to the owner and explicitly authorised staff. A saved requested setting is not shown as active until its dependencies are satisfied. Unsupported activation returns an actionable private explanation to the owner while the optional customer module stays absent.

## 3. One control centre, organised for one operator

Use a single authenticated Super Admin destination with a clear overview, searchable settings and grouped sections. It should work on a phone as well as a desktop. Avoid a wall of hundreds of unlabeled toggles.

Required sections and configuration coverage:

| Section | Owner-managed functions |
| --- | --- |
| Overview and attention | Current effective services, private integration health, failed/uncertain operations, support requests, reconciliation exceptions and scheduled work in one prioritised queue. |
| Product controls | Public-site maintenance, registration, invitations, sales, invoice upload, instalments, manual payment reporting/confirmation, receipts, bank linking/authorisation, bank collection, automatic collection and retries, reports and supported optional modules. |
| Integrations | Supported Mono, identity, email, SMS, WhatsApp, storage and scanning provider connections; write-only credential entry/rotation, callbacks, diagnostics and safe connection checks. Only expose providers with implemented adapters. |
| Pricing and business rules | Supported fees, notice/grace settings, limits, reminder timing, retry policy, exposure rules and effective dates. Preserve the terms already attached to accepted sales. |
| Customers and businesses | Search, profiles, verification/review status, account suspension/restoration, organisation-level restrictions, recorded exceptions and attributable support interventions. No silent customer impersonation. |
| Transactions and operations | Payment/sale history, reported transfers, disputes, collection exceptions, jobs, reconciliation, authorised financial corrections and exports. Corrections append an attributable record; they do not rewrite history or label an unreceived payment as bank-confirmed. |
| Website and content | Supported homepage sections, text, images, navigation, FAQs, guides, contact details, SEO/social metadata and bounded brand tokens. Draft, preview, publish and restore revisions. No arbitrary JavaScript/SQL/shell execution from the content editor. |
| Notifications and automation | Supported channel switches, reviewed message templates, reminder/operational schedules, quiet hours, retry limits, private test sends and owner alerts. Customer consent and transactional obligations remain applicable. |
| Legal and business details | Actual operator details, service/support/legal/privacy contacts, versioned documents, effective dates, publication and acceptance history. Drafts stay private; applicable published documents remain accessible. |
| Access and security | Owner MFA/recovery and active sessions, granular employee roles when needed, operational mode, audit history, key rotation status and recovery/backup status. |

Search should accept plain terms such as `SMS`, `fees`, `logo`, `bank collection` and `customer limits`. Show the control's purpose, current effective value, scope, whether it applies to new or existing records and whether it takes effect now, on schedule, or after an explicitly identified restart.

Use safe previews for public-content and operational changes, a before/after review, immediate apply or scheduled activation where supported, and a settings rollback. Rolling back configuration must never reverse a payment, delete an audit event or overwrite terms already accepted by a customer. Private owner preview must not create real provider actions or pollute customer records.

## 4. Configuration architecture and coverage contract

Use a typed settings registry as the shared contract for server validation and Admin form generation. Each setting has a key, category, type, default, validation, permission, scope, dependency rules, sensitivity, effective-time behavior and audit/redaction policy. Do not implement raw arbitrary key/value editing as a substitute for supported settings.

Maintain a coverage inventory for every setting currently in environment files or hardcoded policy, marking it as `runtime-admin`, `deployment-secret/bootstrap`, `implemented-provider-only`, or `code-change`. Every user-facing operating setting needs a linked UI control, persistence path, enforcement point and test. Do not declare coverage complete while provider settings still say 'go through deployment instead' or the database unconditionally rejects the solo owner's own approval.

Persist supported runtime settings in a server-owned versioned database representation. Use expected-version checks to prevent lost updates, atomic audit writes, strict validation and permissions on every read/write. Clients cannot set their own capabilities. Return only the minimal effective customer-facing capabilities, never the private catalogue, secrets or internal dependency reasons.

Check effective settings in the API and all workers immediately before the relevant operation. Apply a confirmed update across instances and invalidate affected caches. Report propagation/restart state privately rather than falsely claiming instant activation. Queued work rechecks policy; reconciliation and callbacks for submitted work continue when new collection is off. New money actions fail closed when effective configuration cannot be established.

A feature must be connected end to end: owner control -> validated durable setting -> effective server policy -> API/worker behavior -> public/customer rendering. A standalone settings screen, a frontend-only toggle or a disconnected configuration table is not an implementation.

## 5. Provider and secret management

Provider settings are edited in Super Admin where an adapter supports them. Store secret values through the designated encrypted secret mechanism. Never return saved secrets, include them in exports/audit logs, place them in browser storage or expose them through customer endpoints. Show configured state, masked identifiers and safe diagnostics. Validate destination URLs against the relevant provider/egress policy; an owner URL field must not become unrestricted server-side network access.

Sandbox tools remain isolated from production identities, data and workers. Switching mode must not lose submitted operations, callback handling or reconciliation. Diagnostic tests must identify whether they contact a real provider, and must not charge customers implicitly.

Owner-controlled activation does not manufacture a provider entitlement or a customer's mandate. Keep observable technical state separate from owner-recorded decisions. Missing Mono configuration prevents Mono-dependent actions, not the general public website from starting.

## 6. Security and infrastructure boundaries

The owner has broad operating authority but cannot turn off tenant isolation, accurate kobo accounting, request authentication, CSRF protection, webhook verification, payment idempotency, consent or audit integrity. Sensitive owner changes require fresh authentication. Those measures protect the founder's control; they do not require an additional employee.

DNS, server TLS, initial database connectivity, the root secret-encryption key, hosting credentials and the initial owner bootstrap must exist before the console can work. Keep this small deployment boundary explicit. Normal supported product operations after setup should not require editing code or redeploying. A new feature, new provider adapter, database redesign or application bug fix remains software work; do not promise a switch can implement nonexistent behavior.

Maintain core security, payment and isolation tests. Replacing customer-facing development language is not permission to remove tests or present an unreviewed security/legal assertion as verified.

## 7. Public content and document publication

Sweep landing pages, navigation/footer, pricing/FAQ, security, legal content, onboarding, empty states, emails, notification templates, social/SEO metadata and cached/offline assets. Remove lifecycle/preparation messages about the product where they are not part of the intended live service experience. Remove promises and calls to action for hidden modules at the same time as disabling their controls.

Use normal service language for enabled journeys. Keep examples labelled as examples, zero balances accurate and transaction states truthful. Do not invent customers, reviews, historical volumes, certification, bank success or operational uptime to make the service appear established.

Manage actual legal/business facts and publication in Super Admin. Keep unpublished legal drafts private rather than stripping their draft label and presenting them as effective documents. General website/account terms and bank-collection terms have separate applicable scopes; Mono onboarding is not required to publish accurate general information. Existing applicable document versions and customer acceptance remain traceable.

## 8. Acceptance tests required for completion

1. Exactly the intended owner can access owner-only settings; normal customers, supplier owners and ordinary staff cannot acquire ownership through signup, role edits, URLs or crafted requests.
2. In solo-owner mode, the verified owner can review and apply permitted settings without another account. Attempts without fresh authentication, with invalid inputs or with stale revisions fail without partial change.
3. Delegated staff cannot use the solo-owner path. Optional team approval is enforced when selected, and staff deletion never silently relaxes it. Owner transfer/recovery and last-owner safeguards are tested.
4. Supported settings persist through restart, are consistently enforced across API/workers and have attributable redacted history. Rollback produces a new version rather than deleting history.
5. Disabled optional modules are absent from customer navigation, onboarding, marketing promises, metadata and notifications; private reasons and secrets never appear in public responses. A direct new-action request is refused server-side.
6. Activating a configured module makes the real supported journey visible without a code change. An unready provider stays hidden to customers and gives the owner a specific private diagnostic, never false activation.
7. Owner-only preview and sandbox operations are isolated from customers and live financial effects. No simulator can produce a paid or verified production record.
8. Pausing new activity preserves existing records, applicable disclosures, outstanding transaction handling, privacy/support access and reconciliation. Scheduled jobs and in-flight work are explicitly covered.
9. Public production routes work without Mono configuration in the selected non-payment state while required infrastructure security remains enforced.
10. Accepted sale terms, exact balances, mandate consent, signatures, tenant boundaries and idempotency survive changes to settings and mode. Emergency pause is immediate for new actions, not an instruction to abandon submitted work.
11. Every in-scope operational setting has a discoverable Super Admin control. Search, mobile layout, keyboard use, change previews, effective dates, error recovery and safe secret rotation are tested.
12. Final completion evidence identifies the exact source revision, environment and tested UI/API/database behavior. Public deployment and collection activation are separately recorded, not inferred from copy or a green workflow.

## Historical implementation status at the initial PR #15 inspection

PR #15 initially changed the public security page and added two regression tests plus this specification. The inspected Admin still requires a second platform administrator for business-policy changes and still excludes provider/credential/live-money settings from its settings screen. `internal/access/roles.go` separates supplier ownership from platform staff but does not yet define the dedicated platform-owner model required above.

This document update resolves the product requirements; it does not implement the owner role, solo approval, runtime settings registry, database migrations, secret-management UI, hidden-feature enforcement or the full public-content sweep. Do not claim the complete Super Admin works until those paths and the acceptance tests above have been implemented and verified. Neither this documentation change nor the earlier security-copy change merges, deploys or activates any financial service.

## September 9 audit status

Owner roles, solo-owner decisions and supported runtime settings now have implementation paths. The initial inspection above is historical. The direct audit remains in progress; use `docs/launch-audit-2026-09-08/file-by-file-audit.json` for unresolved findings. Website content and legal publication metadata are still source-managed and do not yet satisfy the full content-management requirement in this document. No complete-admin or launch sign-off is claimed.
