# Production backend cutover

This procedure updates the existing `kredit-prod` Compose API, worker and migrator. The frontend has its own deployment. It requires an operator-authorized production maintenance window; source review does not authorize executing it.

## Before cutover

Commit the reviewed release. The deploy script refuses a dirty checkout and archives exactly `HEAD`, verifies its checksum after transfer, and retains that archive. Confirm the production host has Docker Compose with `--wait` support, Bash, `flock`, SHA-256 utilities, and an existing healthy stack. Configuration and certificates stay under `/opt/kredit/infra/environments`; no credentials are copied from the local checkout.

Set `LEGAL_ENTITY_NAME` to the approved operator name. Optional settings are `VPS_HOST`, `VPS_USER`, `SSH_KEY`, `API_URL` and `BASE_URL`. Then invoke `scripts/deploy-production.sh` from the repository when the operator has authorized cutover. A missing optional key file uses the SSH agent/configuration; authentication remains noninteractive.

`LEGAL_ENTITY_NAME` here is the expected name checked by the post-deploy script, not a runtime override for the legal pages. Their initial metadata comes from `web/src/lib/server/legal-publication.ts`; release certification validates that publication against the backend versions and still requires separate legal-approval evidence.

## Migration 213: one person per business

The API and worker require migration 213. This is pre-launch cleanup for the one-owner product, not a customer-account conversion. It drops unused staff invitation, delegation, branch, internal credit-approval, proposed-terms import and history tables, along with the obsolete shared process-status table. It removes staff-only columns and constraints. Customer invitations, business ownership and financial records remain.

Use the stopped-writer and backup sequence below for any existing installation. Before applying it, confirm the following ownership preflight returns no rows:

```sql
SELECT id, organization_id FROM app.memberships WHERE role <> 'owner' OR status = 'invited';

SELECT organization_id, count(*) AS owners
FROM app.memberships
GROUP BY organization_id HAVING count(*) > 1;

SELECT o.id
FROM app.organizations o
WHERE o.status NOT IN ('closed', 'suspended')
  AND NOT EXISTS (
    SELECT 1 FROM app.memberships m
    WHERE m.organization_id = o.id AND m.role = 'owner' AND m.status = 'active'
  );
```

The migration rejects inconsistent ownership rather than selecting an owner or silently deleting accounts. Staff feature data in the dropped tables is deliberately not archived. The owner reviews and applies credit notes; financial transaction guards remain. No migration or account reset was performed during this source cleanup. The migration is forward-only, and old binaries that depend on the deleted tables must not run after it.

## Maintenance tooling

For standalone Caddy, `scripts/refresh-cloudflare-ranges.sh` creates a validated `.candidate` file using the deployment's environment, certificate mounts and Caddy image. Inspect it, then run the same script with `--apply` to validate again, back up and activate. The script uses the release override when present and exits unsuccessfully after attempting restoration if activation fails. It does not use the disabled Caddy admin API. Keep the previous configuration until the proxy is confirmed healthy.

Terraform uses the checked-in provider lockfile. Initialize with `-lockfile=readonly`; refresh provider selections deliberately with Terraform's `providers lock` command for `linux_amd64`, `linux_arm64`, `darwin_amd64` and `darwin_arm64`, then review the resulting checksums/version change. Static validation does not contact or certify the deployed cluster.

## What the script does

1. Takes a deployment lock and stages the archive in a private, versioned directory under `/opt/kredit/releases`.
2. Retains prior API/worker image IDs and rollback tags, then builds both images from the staged source under unique release tags. A generated `docker-compose.release.yml` pins the API, worker and migrator to those tags; failed staging does not retag the running release. The migrator is in the same API image. Image labels and process metadata identify the release and full Git revision.
3. Stops API and worker writers. The PostgreSQL 18 backup container assumes the dedicated `kredit_backup` role and writes a custom-format archive to `/opt/kredit/backups`. It checks the producer exit status, reads the archive contents without restoring them, and records a SHA-256 sidecar and the actual archive path.
4. Runs all pending migrations with the new migrator, then reapplies role/login provisioning. This includes forward-only integrity fixes; do not select a hard-coded migration range.
5. Starts both application services, waits for container health, and checks their actual image IDs. It refreshes the existing optional Caddy service, switches `/opt/kredit/current` and installs a systemd drop-in so reboot uses that release.
6. Checks the public home/legal/robots/sitemap responses and configured API health endpoint. This check does not exercise authenticated financial flows or deploy the frontend.

Once writers are stopped, a remote failure leaves them stopped for recovery and returns a failure. A failed public check after remote completion also returns a failure; it does not automatically reverse an otherwise running release.

## Recovery

Each release retains `source.tar.gz`, its checksum/revision, and `rollback/` containing the previous source directory, prior API/worker image IDs and backup manifest. Images also retain `kredit-api:rollback-<release>` and `kredit-worker:rollback-<release>` tags. Keep these artifacts until the maintenance window and retention requirements are resolved.

For a release created by this script, run administrative Compose commands from `/opt/kredit/current/infra/environments` with `--env-file .env.production -f docker-compose.prod.yml -f docker-compose.release.yml`. The generated override is required to select that release's images. Do not build or retag these release images in place.

Before restarting an older image, verify that it is compatible with the now-applied schema and integrity triggers. Migrations 201 and 207 onward require particular care: restoring unsafe behavior or starting an incompatible writer is not a generic rollback. Choose a reviewed forward correction when compatibility cannot be established. The script never runs migration `down`, deletes a production database, or automatically restores a backup.

A readable archive and matching checksum do not establish successful restoration or an achieved recovery time. Follow [the backup restoration runbook](../runbooks/backup-restore.md) using an isolated target with providers and workers disabled; do not restore over the serving primary. Preserve configuration/encryption keys separately under the existing secret-management process.
