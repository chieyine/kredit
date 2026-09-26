# Production backend cutover

This procedure updates the existing `kredit-prod` Compose API, worker and migrator. The frontend has its own deployment. It requires an operator-authorized production maintenance window; source review does not authorize executing it.

## Before cutover

Commit the reviewed release. The deploy script refuses a dirty checkout and archives exactly `HEAD`, verifies its checksum after transfer, and retains that archive. Confirm the production host has Docker Compose with `--wait` support, Bash, `flock`, SHA-256 utilities, and an existing healthy stack. Configuration and certificates stay under `/opt/kredit/infra/environments`; no credentials are copied from the local checkout.

Set `LEGAL_ENTITY_NAME` to the approved operator name. Optional settings are `VPS_HOST`, `VPS_USER`, `SSH_KEY`, `API_URL` and `BASE_URL`. Then invoke `scripts/deploy-production.sh` from the repository when the operator has authorized cutover. A missing optional key file uses the SSH agent/configuration; authentication remains noninteractive.

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
