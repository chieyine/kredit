# Kredit launch audit — 12 September 2026

Current completion and essential-check evidence are in [the implementation record](../platform-improvements/IMPLEMENTATION.md) and [the check record](../platform-improvements/ESSENTIAL-CHECKS.md). The audit-stage notes below describe the earlier snapshot.

**Decision: not ready to launch.** The source audit is complete. The repairs and runtime verification are not complete.

All **940 tracked files** were inspected, including **117 migrations**, backend and frontend source, admin pages, tests, configuration, infrastructure, documentation and images. No agents, tests, builds, migrations, provider transactions or deployments ran. Application source was not changed. Only this audit folder was created; `.gitignore` changed externally during the audit and was reread.

## Deliverables

- [42 findings, impacts and repair directions](FINDINGS.md).
- [Every file and its review result](FILE-BY-FILE.md).
- [Current Mono, Sendly and other provider research](PROVIDERS.md).
- [File register with hashes, methods and notes](file-review-index.json), [CSV](file-review.csv), and [structured findings](findings.json).
- [Scope and exclusions](scope.json).

“Inspected” means source/evidence review, not a passing test. Historical logs and previous audit claims are tied to earlier snapshots. Structured evidence was parsed completely and inspected through its semantic records, diagnostics and provenance; it was not rerun. Third-party dependency source, secrets, local databases, production configuration and remote account settings were not audited.

## What prevents launch

| Area | Result | Findings |
| --- | --- | --- |
| Deployment | Production database URLs conflict with TLS validation; Caddy syntax and PostgreSQL 18 volume location are wrong; environment instructions are incomplete | F002–F006, F011 |
| Account access | Sendly is not integrated; recovery cannot reach its evidence step | F001, F012 |
| Supplier setup | Production settlement verification is missing; saved billing details do not establish billing | F018–F019 |
| Buyer sale | Opening a sent sale changes an unsaved version and blocks the next action; invitation consent evidence does not match what was shown | F020, F022, F028 |
| Collections | Production reads lose tenant context; external successes cannot be recognized locally; minimum and retry handling need correction | F007–F010, F021, F026 |
| Admin | Scope failures hide records, prevent ownership/financial actions and can falsely report role-grant failure or financial resolution | F013–F017, F025 |
| Customer limits | Durable operations and expiry work lack required database scope | F024 |
| Documents | R2 request mismatch; invoice retrieval and dispute attachment workflows are incomplete | F030–F031, F038 |
| Customer-facing accuracy | Payment-page loop, wrong-business settings, wrong agreement amount, unsaved draft sending and ageing labels | F023, F027, F029, F032–F033 |
| Launch evidence | Backup/certification scripts are inconsistent; several tests bypass production wiring; privacy approvals are not attached | F036–F037, F040–F042 |
| Credential boundary | Extensionless SSH files can enter local Docker build contexts despite Git exclusion | F039 |

These are not all independent failures. For example, F021 stops collection before submission; F007 remains a second failure after eligibility is repaired. Fixing one does not clear the other. Several admin/report issues share the same tenant-context cause, but need distinct authorized read paths.

The code contains useful protections worth preserving: exact kobo arithmetic, immutable financial records, scoped database policies, stable request identities, provider-response reconciliation, bounded requests and document quarantine. Do not remove those protections to make a blocked path appear to work.

## Launch domain and wording

The intended public domain is **kredit.ng**, with **api.kredit.ng** for the backend where the deployment requires it. Current public URL configuration largely reflects this. The remaining confirmed customer-visible domain problem is the old `kredit.com.ng` text embedded in both sharing images (F035). The public-receipt test also expects the old email domain (F040). `/contact` is inadvertently blocked by the private `/c` robots prefix (F034).

Regenerate the sharing images, align callback/return URLs and mail sender configuration, and inspect any database-published content overrides during deployment. The source audit cannot establish what is currently saved in the production content settings or DNS. Existing accepted legal records must retain their original version and hash.

The admin analytics page still says “pilot”; use clear launch wording for the intended audience after the operational scope is agreed. Keep real safety limits even if their labels change. Sample sales should remain visibly labelled examples. Do not turn unresolved approvals or missing integrations into claims that the platform is live and working.

## Repair order

1. Correct credential/build exclusions and make the production configuration internally consistent, preserving restricted database roles.
2. Repair tenant context through actual runtime interfaces, then the sale-version transition and admin read/mutation paths.
3. Implement the contracted Sendly, identity, settlement and billing workflows; close Mono mandate recovery, minimum, retry and callback gaps.
4. Finish account recovery, document access, payment navigation, business selection, consent and amount/copy fixes.
5. Align runbooks, deployment evidence and the small set of meaningful checks below.

## Essential verification after repairs — not executed

Run checks only against the repaired candidate and isolated data. Reuse existing tests where their setup reflects production; do not run broad suites merely to collect a pass count.

| Necessary check | What it must prove |
| --- | --- |
| One build/configuration pass | Changed Go packages and frontend compile; Compose/Caddy accept the final configuration; startup uses TLS and the correct migration/role boundary. |
| Focused database regressions | Actual runtime wiring with restricted API/worker roles: sent sale → first buyer action → activation → debit recognition → duplicate/restart → reversal; two tenants remain isolated. Cover the affected admin, recovery, reporting and limit paths without context-injecting fixture substitutes. |
| Provider contract checks | Small sandbox cases for mandate creation/readiness, success, partial/final failure, unknown result and dispute/reversal callbacks; Sendly recipient acceptance/suppression and authenticated delivery; one R2 upload/scan/download. No live money by default. |
| Focused browser journeys | Sign-in/recovery, supplier setup, correct second-business settings, sale save/send, invoice/dispute evidence and real payment navigation; confirm critical actions and amounts on desktop and a phone viewport. |
| One isolated backup restore | Restore the exact produced archive and compare necessary financial/source-of-truth records, using the corrected archive path. |

A final launch decision additionally needs real provider enablement, approved privacy/retention records, verified sender/DNS and deployment evidence. Those cannot be proved from local source or invented approval-reference strings.
