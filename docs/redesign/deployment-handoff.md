# Deployment handoff

The user is handling production deployment and live provider/alert checks. Local verification is not evidence that the deployed services have been updated.

## Apply the coordinated release

1. Preserve the personal account and take a restorable database backup. This implementation does not require a production reset.
2. Apply all migrations through **199** using the repository's `cmd/migrate` and the deployment migration connection. Apply the updated `infra/postgres/roles.sql` using the role administrator. Do not use the migration role as the application login.
3. Release the backend and workers together with the frontend. New import, approval and network-operation screens require PostgreSQL; unavailable persistence produces an explicit failure.
4. Build the frontend with the normal Vercel adapter. Avoid sharing generated build files with an actively running development server.
5. Update configured links and published content to `/workspace`, `/personal/purchases`, `/account`, `/signin` and `/start`. Removed `/app` and `/buyer` page routes have no compatibility redirects. Preserve provider references and reconcile provider callback/return configuration with the actual route handlers.
6. Verify sign-in and personal-account access, each business's purchases and sales, customer invitations, import recovery, independent approval/reviewer limits, branch assignments and access boundaries, owner-managed purchasing grants, verified staff acceptance/receipt, revoked staff access and consumer purchases on the deployed version.
7. Run authorized provider sandbox and alert-receiver checks using deployment-managed credentials. Validate backup recovery and job health before onboarding real credit exposure.

## Feature boundaries

Reviewer ceilings apply to single-sale credit offers. Branch/account-manager assignments organize work; owner-managed branch scopes now restrict staff to assigned customers for single-sale workflows and financial history. Company-wide consumer, shared-document and specialist mutation workflows remain restricted to company-wide staff. Closing a branch suspends branch-scoped customer access. Credit approval is not customer acceptance, debt creation or guaranteed collection. Imported contact rosters create no opening balance or mandate. Consumer financial contracts remain separate from business credit.

Purchasing grants bind to a specific membership and authority version. Restoring or reinviting staff requires an owner to renew their purchasing grant. Staff enrollment creates pending verification subjects, not automatic approval. Apply backend, frontend and role grants together.

All 8 blueprint redesign architecture tracks have been implemented, tested, and validated locally (see `implementation-status.md` and `docs/operations/production-cutover-runbook.md`). This handoff provides the exact cutover steps when the user authorizes production deployment.

