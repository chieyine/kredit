#!/usr/bin/env python3
from pathlib import Path

path = Path('IMPLEMENTATION_STATUS.md')
text = path.read_text(encoding='utf-8')
marker = '## Repository code audit (3 September 2026)\n'
if marker not in text:
    raise SystemExit('status section marker not found')
_, tail = text.split(marker, 1)
head = '''# Implementation Status

Last updated: 6 September 2026

## Current engineering state (6 September 2026)

Repository-owned engineering work through **Phase 6** is implemented on the reviewed branch. Database migrations run through **086**. Phases 2–5 established tenant isolation, financial-core/real-stack proof, provider-adapter verification tooling and production-assurance controls; Phase 6 closes the remaining repository hardening around fail-closed API linting/tool pinning, request cancellation, script CSP, truthful sitemap evidence, analytics request-context/privacy boundaries and status governance.

**Request-scoped cancellation:** synchronous HTTP handlers no longer manufacture background contexts in the audited web paths. Browser disconnect cancellation is propagated through the web proxy, payment reads prefer the existing context-aware PostgreSQL methods, buyer invitation acceptance and onboarding notifications use `r.Context()`, and analytics persistence exposes request-aware context methods. Process-startup code, workers and asynchronous jobs intentionally retain independent lifecycle contexts rather than inheriting browser cancellation. `scripts/phase6-context-audit.py` protects this boundary in CI.

**Security/privacy hardening preserved:** session/OTP/TOTP protections, CSRF plus Origin/Sec-Fetch validation, forwarded-header stripping, recent-MFA gates, recovery cooling-off, tenant RLS, immutable financial records, document quarantine/scanner promotion and analytics metadata minimization remain fail-closed. SvelteKit now owns CSP generation so application scripts do not require `script-src 'unsafe-inline'`.

**Release status is not the same as production approval.** Actual Mono sandbox certification remains open in issue #5. Independent penetration/API authorization review, legal/mandate/DPIA approval, intended-platform backup/PITR/key recovery, representative staging load, deployed alert/incident exercises, manual accessibility/device evidence, support readiness and independent human launch approval remain open in issue #7. A successful CI or deployment status does not substitute for those approvals. Repository branch/deployment protection also requires an administrator because the connected integration cannot modify branch-protection settings.

See `docs/testing/phase6-hardening.md` for the Phase 6 completion boundary and `docs/runbooks/phase5-production-assurance.md` for external evidence requirements.

'''
path.write_text(head + marker + tail, encoding='utf-8')
