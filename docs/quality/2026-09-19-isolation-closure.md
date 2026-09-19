# Database isolation and security closure — 19 September 2026

## Changes completed locally

- **Ordinary database owner seeding:** migration 157 allows the exact activity
  trigger owner to append its audit record while FORCE RLS remains enabled.
  It grants no runtime bypass and no audit update/delete capability. Repeated
  seeding succeeds using an ordinary, non-superuser migration owner.
- **PostgreSQL 18 bootstrap setup:** development provisioning recognizes the
  permanent bootstrap role (OID 10), which PostgreSQL cannot demote. Separate
  application and worker logins remain non-superuser and NOBYPASSRLS. Ordinary
  owner roles are still demoted when requested. Fresh migrations and role setup
  were exercised in a separate PostgreSQL 18 cluster.
- **All 14 pending tables:** migrations 158–163 remove blanket runtime policies.
  Three related child tables are hardened too. Stores now set transaction-local
  identity; worker discovery has bounded capabilities; role checks protect
  reviewer access. Migration 164 preserves narrowly authorized operations
  counts, exact dispute reference search and a pending send's recovery scope.
- **Sign-in origin weakness:** the focused security review found that a
  cross-origin form could submit an attacker-owned OTP and install that
  account's session in a victim's browser. Local reproduction returned HTTP 200
  and two cookies for cross-site, same-site and Origin-only foreign requests.
  The fix rejects them before consuming the OTP. A subsequent legitimate
  same-origin request succeeds using that same code. This was login CSRF,
  **not evidence of access to an existing victim account**.

The security workbench's generated [source review](2026-09-19-security-review/report.md)
records the finding **before remediation**. The current fix and regression test
are in `internal/web/auth_handlers.go` and `internal/web/login_origin_test.go`.
The scan was deliberately focused and partial, used no agents and performed no
production exploitation. Measured scan token usage was unavailable.

## Verification evidence

Local logs are under `.tmp/closure-20260919/` (ignored verification artifacts).

| Check | Result / evidence |
|---|---|
| Fresh migrations and bootstrap role provisioning | Passed; `bootstrap-migrate-final.log`, `bootstrap-roles.log`, `bootstrap-role-proof.log`, `final-migrate.log` |
| Ordinary-owner repeat seed and restricted runtime roles | Passed; `ordinary-seed-final.log`, `ordinary-role-proof.log` |
| Populated rows across 14 tables and 3 child tables | Passed for both runtime roles; unrelated/missing identity reads and updates denied; `go-integration-complete.log` |
| Pool identity clearing and transaction release | Passed, including cancellation and one-connection reuse; `internal/db/scoped_database_test.go` |
| Restricted customer and worker repositories | Passed; `restricted-repositories2.log`, `restricted-usercontrol-final.log`, `platformops-final.log` |
| Policy-shape backlog | Zero; `policy-shape.log` |
| Login-origin reproduction / fix | Failed before fix as expected; passed after fix; `login-origin-before.log`, `login-origin-after.log` |
| Prometheus rules and rule tests | All 17 rules valid; rule tests passed with checksum-verified Prometheus 3.14.0 |
| Actual local alert delivery | Firing and resolved events received through Prometheus → Alertmanager 0.34.1 → synthetic loopback receiver using the repository rule; `alert-delivery-final.log`, `alert-delivery-events.json` |

The full integration suite passed with exit code 0 using the restricted API and
worker login connections (`go-integration-passed.log`). Race checks for the six
changed core packages also passed with exit code 0 (`race-final.log`).

Earlier runs exposed missing fixture connection variables, a missing seed, a
synthetic identity mismatch and a duplicate test import. Those runs were not
counted as successful. Production provider requests and messages were not sent.

## What this says about production security

`https://kredit.ng` redirects to `https://www.kredit.ng`. The homepage and public
API readiness endpoint returned HTTP 200. The API response included no-store,
HSTS, frame restrictions and content-type protection. These are useful checks,
but they do not establish that the VPS, database or deployed application is
secure or that it contains this workspace's fixes.

The user confirmed Vercel frontend, Cloudflare components and a VPS backend and
requested proportionate verification. We did not inspect VPS SSH/firewall rules,
public database exposure, production secrets, provider sandbox accounts or the
real alert receiver. Local simulated delivery is not proof of production alert
delivery. The connected hosting account did not expose the Kredit project.

The focused review found a real sign-in weakness and fixed it locally. It would
therefore be misleading to label the current live deployment unhackable or
"worldclass" from normal operation alone. The highest-priority next deployment
step is to ship the coordinated database/API/worker changes and confirm the VPS
uses restricted runtime credentials with database ports inaccessible publicly.

## Rollout

These changes have **not been deployed** by this task. Apply migrations through
164 and the matching `infra/postgres/roles.sql`, then deploy matching API and
worker builds. Drain old workers during the coordinated rollout: old unscoped
repositories are incompatible with the stricter policies. See
[Phase 2 completion](../operations/rls-phase2-completion.md). Preserve financial
and audit history; use forward fixes rather than restoring broad policies.
