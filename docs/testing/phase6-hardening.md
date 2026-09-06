# Phase 6 — remaining hardening and product polish

Status: **started**. This phase follows the merged engineering work from Phases 2–5. It does not replace the external provider, security, legal, accessibility, recovery or launch approvals tracked separately.

## First implemented slice

- CI installs a pinned Redocly CLI and requires a real OpenAPI lint pass. The structural fallback remains available only for non-CI local smoke checks.
- Trivy is pinned instead of silently tracking `latest`.
- Public fixed pages no longer claim a fabricated sitemap `lastmod`; topic timestamps derive from the newest publishable article in that category.
- `scripts/phase6-governance-test.sh` protects those controls from regression and is part of `scripts/ci.sh`.

## Remaining Phase 6 work

The remaining repository-owned hardening work is intentionally separate from the financial-core architecture already proved in earlier phases:

1. Finish request-scoped `context.Context` propagation through synchronous HTTP service/repository paths and add cancellation tests. Worker jobs keep independent job contexts rather than inheriting browser cancellation.
2. Continue CSP hardening toward nonce/hash-based script execution where SvelteKit constraints permit it, without weakening CSRF, Origin/Sec-Fetch, proxy-header stripping or secure-cookie controls.
3. Review account-recovery abuse controls, key-rotation procedures, recent-MFA coverage and admin privilege separation; add targeted regression tests where missing.
4. Harden upload validation/quarantine/storage authorization boundaries.
5. Finish transaction-certainty UX, manual accessibility evidence and conservative pilot/kill-switch behavior without redesigning the product architecture.
6. Keep SEO private-route/noindex behavior fail-closed and ensure dates/metadata represent real evidence.
7. Bring implementation/status documentation in sync with actual migrations and distinguish implemented, tested, externally certified and deployed states.
8. Continue CI/release reproducibility: pinned tools, signed/reviewable releases where available, and branch/deployment protection when repository-administration access is available.
9. Review analytics privacy and KPI definitions so financial/customer identifiers are not exported into product analytics.

## Acceptance rule

A Phase 6 code item is complete only when the repository test that proves it is present and green on the exact reviewed commit. External or manual items remain explicitly pending until their real evidence exists. Production enablement is not part of this document.
