# Phase 6 — hardening and product polish

Status: **repository-owned engineering work complete; external/manual launch gates remain pending**.

This phase follows the merged engineering work from Phases 2–5. It does not convert simulator evidence into provider certification and does not replace independent security, legal/privacy, accessibility, recovery, deployment or human launch approval.

## Completed repository-owned work

1. **CI/API governance** — CI installs pinned Redocly CLI `2.51.2` and requires a real OpenAPI lint pass. The structural fallback is local-only. Trivy is pinned to `v0.74.0`. `scripts/phase6-governance-test.sh` makes those decisions regression-tested.
2. **Request cancellation** — browser cancellation is combined with the 30-second proxy timeout. Synchronous HTTP handlers no longer manufacture background contexts for buyer invitation acceptance or onboarding notifications. Payment reads prefer `GetContext`/`ReadContext`, and analytics persistence now exposes request-aware context methods while retaining compatibility wrappers for non-request callers. `scripts/phase6-context-audit.py` is part of CI and intentionally excludes process-startup and worker/job contexts.
3. **CSP/browser boundary** — SvelteKit owns CSP generation in `auto` mode so framework scripts use generated nonce/hash evidence. `script-src 'unsafe-inline'` is not permitted. Existing same-origin proxy header stripping, secure cookies, CSRF/Origin/Sec-Fetch controls and private-route cache rules remain intact.
4. **Upload quarantine boundary** — the existing document pipeline remains fail-closed: bounded upload slots, allowed content type and size checks, private storage, object verification on finalize, scan state separate from upload completion, and only the scanner path can promote a document to clean. Actual malware-scanner effectiveness remains an environment/provider acceptance item.
5. **Account/admin safeguards** — existing recent-MFA enforcement, recovery cooling-off, role/permission checks and specialist admin separation remain the supported boundaries. No Phase 6 change weakens them.
6. **Analytics privacy** — subject identifiers remain SHA-256 pseudonymized and analytics metadata rejects direct identifiers/sensitive fields including phone, email, BVN, NIN, bank account and provider token. Metadata count/value limits remain enforced. HTTP analytics writes now carry request cancellation to PostgreSQL.
7. **SEO evidence** — fixed public pages no longer publish fabricated sitemap `lastmod` values. Blog topic dates derive from publishable article metadata. Private routes remain outside public discovery/indexing behavior.
8. **Release/status governance** — Phase 5's candidate/environment/configuration-bound release-evidence gate remains authoritative. Phase 6 distinguishes implemented/tested engineering work from external certification, deployed-environment proof and human approval rather than using a successful build/deployment as launch authorization.

## Existing controls deliberately preserved

Phase 6 does not rewrite the ledger, payment transaction, collection reconciliation, mandate/provider boundary, tenant isolation model, authentication foundation or SvelteKit application architecture. Those areas are exercised by the earlier phase suites and normal CI.

## External/manual work that remains open

These are not repository defects that can be truthfully closed by code alone:

- Actual Mono Sweep/Partial Sweep sandbox certification and provider confirmation: issue #5.
- Independent penetration/API authorization assessment, legal/mandate/DPIA approval, intended-platform backup/PITR and key recovery, representative staging load/concurrency, deployed alert/incident exercises, manual VoiceOver/TalkBack/400% zoom/high-contrast/real-device accessibility, support readiness and independent human launch approval: issue #7.
- Branch/deployment protection administration: the connected repository integration does not have permission to modify branch-protection settings. This must be enabled by a repository administrator; it is not treated as an engineering pass.

## Acceptance rule

A repository-owned Phase 6 item is complete only when its deterministic check is present and green on the exact reviewed commit. External or manual items remain pending until genuine evidence exists. No production collection flag, legal-page activation, external debit, paid assessment or production deployment is authorized by this document.
