# Security Review: Kredit.com

## Scope

Focused review of common browser/API attack paths

- Scan mode: scoped_path
- Target kind: git_worktree
- Target ID: target_sha256_cf07476788ff019c5c010229ffa2d115b1ab5b30c1c8a6b64747beb42023c5e2
- Revision: ea7a5a88460857035362604f7dc86cef84bd9501
- Snapshot digest: codex-security-snapshot/v1:sha256:bce23e910b5cbe710ec101cc68d0cd650d0f87e32d117a3cb1ba0ef892c4f3de
- Inventory strategy: scoped_path
- Included paths: internal/web
- Excluded paths: none
- Runtime or test status: No production attacks performed
- Artifacts reviewed: internal/web/auth_handlers.go, internal/web/http_helpers.go, internal/web/document_handlers.go, internal/web/notification_receipt_handlers.go, web/src/hooks.server.ts

Limitations and exclusions:
- Partial review of internal/web under the user's request for proportionate checks; selected supporting control functions also inspected.
- No independent agent or live exploitation. Production VPS and deployed revision remain unverified.
- Excluded internal/web/\*: Remaining internal/web handlers outside the focused review; no whole-repository penetration-test claim.
- Excluded production configuration: Live provider transactions, credentials, VPS configuration and intrusive production testing.

### Scan Summary

| Field | Value |
| --- | --- |
| Scan outcome | completed |
| Reportable findings | 1 |
| Severity mix | medium: 1 |
| Confidence mix | high: 1 |
| Coverage | partial |
| Validation mode | source review |

Canonical artifacts: `scan-manifest.json`, `findings.json`, and `coverage.json`. This report is a deterministic projection of those files.

## Threat Model

Kredit serves browser financial workflows through a SvelteKit API proxy into Go handlers. User-reported deployment uses Vercel, Cloudflare and a VPS; live configuration and deployed revision are unverified.

### Assets

- Authenticated browser sessions
- Customer documents and personal financial records
- Tenant-separated financial actions

### Trust Boundaries

- Browser-controlled JSON and Origin enter OTP verification before cookie issuance (internal/web/auth_handlers.go:80-99; internal/web/http_helpers.go:19-30).
- Authenticated writes normally require origin validation and double-submit CSRF (internal/web/auth_handlers.go:228-242).
- Uploads require organization access, CSRF, size limits and clean scan before download (internal/web/document_handlers.go:25-121).
- Provider receipt handlers authenticate evidence before accepting signals (internal/web/notification_receipt_handlers.go:15-134).
- SvelteKit forwards /api requests directly, including body and Origin, without invoking resolve for API routes (web/src/hooks.server.ts:27-114).

### Attacker Capabilities

- An anonymous remote party can submit HTTP requests and host a webpage.
- An attacker can legitimately obtain an unused OTP for their own account; cannot read another user's OTP or session.

### Security Objectives

- Cross-origin navigation must not install an attacker-controlled session.
- Tenant records require authenticated, authorized identity.
- Provider callbacks must not directly fabricate financial outcomes.

### Assumptions

- Focused, proportionate review requested; coverage is partial within internal/web.
- No agents used, so independent baseline unavailable.
- VPS firewall, production database roles, secrets, deployment revision and provider configuration not inspected.

## Findings

| Finding | Severity | Confidence | Detailed write-up |
| --- | --- | --- | --- |
| [Cross-site OTP verification can install an attacker-owned browser session](#finding-1) | medium | high | inline below |

### Confidence Scale

| Label | Meaning |
| --- | --- |
| high | Direct evidence supports the finding with no material unresolved blocker. |
| medium | Evidence supports a plausible issue, but material runtime or reachability proof remains. |
| low | Evidence is incomplete and the item is retained only for explicit follow-up. |

<a id="finding-1"></a>

### [1] Cross-site OTP verification can install an attacker-owned browser session

| Field | Value |
| --- | --- |
| Severity | medium |
| Confidence | high |
| Confidence rationale | The public handler consumes OTP and sets cookies without checking Origin, and the proxy forwards form bodies directly. |
| Category | cross-site-request-forgery |
| CWE | CWE-352 |
| Affected lines | internal/web/auth_handlers.go:79-92, internal/web/http_helpers.go:19-30, web/src/hooks.server.ts:99-112 |

#### Summary

OTP verification accepts a valid attacker-owned challenge from a cross-origin form and sets first-party session cookies. The victim may unknowingly enter data into the attacker's account. This does not expose the victim's pre-existing account or bypass OTP possession.

#### Root Cause

The origin control is applied by requireCSRF on authenticated mutations, but omitted from unauthenticated OTP verification that establishes authentication.

#### Validation

Traced anonymous OTP verification through decoder, OTP service call and Set-Cookie; checked middleware and SvelteKit proxy for an equivalent origin guard and found none.

Validation method: Parent source review

- **Status:** source_validated

Counterevidence and remaining uncertainty:
- A valid OTP is still required; an attacker can only supply their own challenge.
- HttpOnly, Secure and SameSite=Lax protect existing cookies but do not prevent setting a new session during a top-level navigation.

Limitations:
- No production attack performed; deployed ingress may have additional protections.
- Source-backed finding; local regression reproduction will run during remediation.

#### Dataflow

Attacker-owned OTP plus form POST reaches verifyOTP, passes JSON decoding, creates attacker-account session and returns Set-Cookie in the victim browser.

#### Reachability

Public /api/v1/auth/otp/verify route; no existing session or idempotency header is required. API proxy forwards the request before framework resolve.

Assumptions:
- Victim visits attacker-controlled content before the OTP expires.
- No separately configured ingress origin policy blocks the request.

#### Severity

**Medium** — A remotely inducible browser action can confuse account ownership, but requires a live attacker-owned OTP and victim interaction.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

Reject cross-origin browser requests before consuming OTP or issuing a session. Preserve legitimate same-origin sign-in and verify rejected requests leave the OTP usable.

Tests:
- Cross-site and same-site form submissions with valid synthetic OTPs must return 403 with no cookies.
- The same unconsumed challenge must subsequently succeed from the configured origin.

## Reviewed Surfaces

| Surface | Risk Area | Outcome | Notes |
| --- | --- | --- | --- |
| OTP sign-in, cookie issuance and CSRF | not recorded | Reported | Fully reviewed auth_handlers.go and http_helpers.go; traced API proxy in web/src/hooks.server.ts. |
| Organization documents and invoice downloads | not recorded | No issue found | Fully reviewed internal/web/document_handlers.go; guards, bounded uploads and scan-gated downloads present. |
| Notification receipt authentication | not recorded | No issue found | Fully reviewed internal/web/notification_receipt_handlers.go; receipt signals do not independently establish delivery. |
| Shared rate limiting, proxy IP and platform permission guards | not recorded | No issue found | Selected control functions reviewed in server.go:755-959 and platform_operations_handlers.go:27-70; these files were not reviewed in full. |

## Open Questions And Follow Up

- Does the actual deployed revision match this workspace?
- Does ingress impose a separate origin/content-type policy?
