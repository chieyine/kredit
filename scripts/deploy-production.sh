#!/usr/bin/env bash
# Deploy one committed backend release through the production Compose stack.
# Forward-only migrations are never automatically rolled back after failure.
set -euo pipefail
umask 077

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"
if [[ -n "$(git status --porcelain)" ]]; then
  printf 'Commit or remove local changes before deploying an identifiable release.\n' >&2
  exit 1
fi
: "${LEGAL_ENTITY_NAME:?Set the approved public operator name}"
VPS_HOST="${VPS_HOST:-117.55.235.58}"
VPS_USER="${VPS_USER:-root}"
API_URL="${API_URL:-https://api.kredit.ng}"
BASE_URL="${BASE_URL:-https://kredit.ng}"
SSH_KEY="${SSH_KEY:-ssh/kredit_prod}"
[[ "$VPS_HOST" =~ ^[A-Za-z0-9.-]+$ && "$VPS_USER" =~ ^[a-z_][a-z0-9_-]*$ ]] || exit 1
revision="$(git rev-parse HEAD)"
release="$(date -u +%Y%m%dT%H%M%SZ)-${revision:0:12}"
work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT
git archive --format=tar.gz --output="$work_dir/source.tar.gz" HEAD
if command -v sha256sum >/dev/null 2>&1; then
  archive_hash="$(sha256sum "$work_dir/source.tar.gz" | cut -d ' ' -f 1)"
else
  archive_hash="$(shasum -a 256 "$work_dir/source.tar.gz" | cut -d ' ' -f 1)"
fi
ssh_args=(-o BatchMode=yes -o ConnectTimeout=10)
if [[ -f "$SSH_KEY" ]]; then ssh_args+=(-i "$SSH_KEY"); fi
remote="${VPS_USER}@${VPS_HOST}"
upload="$(ssh "${ssh_args[@]}" "$remote" 'umask 077; mkdir -p /opt/kredit/releases; mktemp -d /opt/kredit/releases/upload-XXXXXXXX')"
[[ "$upload" =~ ^/opt/kredit/releases/upload-[A-Za-z0-9]+$ ]] || exit 1
scp "${ssh_args[@]}" "$work_dir/source.tar.gz" "$remote:$upload/source.tar.gz"
printf -v remote_command 'bash -s -- %q %q %q %q' "$upload" "$release" "$revision" "$archive_hash"
ssh "${ssh_args[@]}" "$remote" "$remote_command" <<'REMOTE'
set -euo pipefail
umask 077
upload=$1 release=$2 revision=$3 archive_hash=$4
exec 9>/opt/kredit/deploy.lock
flock -n 9 || { printf 'Another deployment is active.\n' >&2; exit 1; }
printf '%s  %s\n' "$archive_hash" "$upload/source.tar.gz" | sha256sum --check --status
release_dir="/opt/kredit/releases/$release"
mkdir "$release_dir"
tar -xzf "$upload/source.tar.gz" -C "$release_dir"
mv "$upload/source.tar.gz" "$release_dir/source.tar.gz"
rmdir "$upload"

config_dir=/opt/kredit/infra/environments
for name in .env.production .env.runtime certs db-certs; do
  [[ -e "$config_dir/$name" ]] || { printf 'Missing production configuration: %s\n' "$name" >&2; exit 1; }
  # Git archives must never contain deployment secrets or certificate material.
  [[ ! -e "$release_dir/infra/environments/$name" ]] || { printf 'Release contains private deployment configuration.\n' >&2; exit 1; }
  ln -s "$config_dir/$name" "$release_dir/infra/environments/$name"
done
cd "$release_dir/infra/environments"
export APP_BUILD_VERSION="$release" APP_BUILD_REVISION="$revision" KREDIT_BACKUP_DIR=/opt/kredit/backups
printf 'APP_VERSION=%s\nAPP_REVISION=%s\n' "$release" "$revision" > env.release
# Each staged release owns its image tags. A failed build must never replace
# tags used by the still-running release or its reboot configuration.
cat > docker-compose.release.yml <<IMAGES
services:
  api:
    image: kredit-api:$release
  migrate:
    image: kredit-api:$release
  worker:
    image: kredit-worker:$release
IMAGES
compose=(docker compose --env-file .env.production -f docker-compose.prod.yml -f docker-compose.release.yml)
rollback_dir="$release_dir/rollback"
mkdir "$rollback_dir"
previous_dir=/opt/kredit
if [[ -L /opt/kredit/current ]]; then previous_dir="$(readlink -f /opt/kredit/current)"; fi
printf '%s\n' "$previous_dir" > "$rollback_dir/previous-directory"
for service in api worker; do
  container="$("${compose[@]}" ps -q "$service")"
  [[ -n "$container" ]] || { printf 'This upgrade requires a running %s service.\n' "$service" >&2; exit 1; }
  previous_image="$(docker inspect --format '{{.Image}}' "$container")"
  docker image tag "$previous_image" "kredit-$service:rollback-$release"
  printf '%s\n' "$previous_image" > "$rollback_dir/$service.image"
done
cp docker-compose.prod.yml "$rollback_dir/new-compose.yml"
printf '%s\n' "$revision" > "$release_dir/revision"
printf '%s\n' "$archive_hash" > "$release_dir/source.sha256"
# Always build from this exact archive. Both targets share the build stage,
# which also produces the migrator embedded in the API image.
"${compose[@]}" build --pull api worker
api_image="$(docker image inspect "kredit-api:$release" --format '{{.Id}}')"
worker_image="$(docker image inspect "kredit-worker:$release" --format '{{.Id}}')"
for image in "$api_image" "$worker_image"; do
  [[ "$(docker image inspect "$image" --format '{{index .Config.Labels "org.opencontainers.image.revision"}}')" == "$revision" ]] || exit 1
done

# Stop application writers before the snapshot and forward-only migrations.
# If anything fails after this boundary, leave writers stopped for recovery.
cutover=0
on_exit() {
  result=$?
  if (( result != 0 )); then
    if (( cutover )); then "${compose[@]}" stop api worker || true; fi
    printf 'Deployment failed. Recovery artifacts: %s. No schema rollback was attempted.\n' "$rollback_dir" >&2
  fi
}
trap on_exit EXIT
cutover=1
"${compose[@]}" stop api worker
"${compose[@]}" run --rm --no-deps backup > "$rollback_dir/backup.manifest"
grep -Eq '^backup=/backups/kredit-[A-Za-z0-9-]+/backup.dump$' "$rollback_dir/backup.manifest"
grep -Eq '^checksum=/backups/kredit-[A-Za-z0-9-]+/backup.dump.sha256$' "$rollback_dir/backup.manifest"
"${compose[@]}" run --rm --no-deps migrate
"${compose[@]}" run --rm --no-deps role-init
"${compose[@]}" up -d --no-deps --no-build --wait --wait-timeout 180 api worker
for service in api worker; do
  container="$("${compose[@]}" ps -q "$service")"
  expected="$api_image"
  if [[ "$service" == worker ]]; then expected="$worker_image"; fi
  [[ -n "$container" && "$(docker inspect --format '{{.Image}}' "$container")" == "$expected" ]] || exit 1
  [[ "$(docker inspect --format '{{.State.Health.Status}}' "$container")" == healthy ]] || exit 1
done
# Refresh the optional ingress only when this installation already runs it.
if [[ -n "$("${compose[@]}" ps -q caddy)" ]]; then
  "${compose[@]}" --profile standalone up -d --no-deps --no-build --wait --wait-timeout 60 caddy
fi
# Reboot uses the same release and retained configuration. Prior source and
# image IDs remain available for a reviewed, schema-compatible rollback.
ln -s "$release_dir" "/opt/kredit/current-$release"
mv -Tf "/opt/kredit/current-$release" /opt/kredit/current
mkdir -p /etc/systemd/system/kredit.service.d
cat > /etc/systemd/system/kredit.service.d/release.conf <<'UNIT'
[Service]
WorkingDirectory=/opt/kredit/current/infra/environments
ExecStart=
ExecStart=/usr/bin/docker compose --env-file .env.production -f docker-compose.prod.yml -f docker-compose.release.yml up -d --no-build
ExecStop=
ExecStop=/usr/bin/docker compose --env-file .env.production -f docker-compose.prod.yml -f docker-compose.release.yml stop
UNIT
systemctl daemon-reload
cutover=0
printf 'Verified API and worker release %s (%s). Recovery artifacts: %s\n' "$release" "$revision" "$rollback_dir"
REMOTE
BASE_URL="$BASE_URL" API_URL="$API_URL" LEGAL_ENTITY_NAME="$LEGAL_ENTITY_NAME" ./scripts/post-deploy-check.sh
printf 'Backend release %s deployed and checked.\n' "$release"
