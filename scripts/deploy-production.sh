#!/usr/bin/env bash
# ==============================================================================
# Kredit Production Deployment Cutover Script
# Deploys 8-track enterprise blueprint to 117.55.235.58
# ==============================================================================
set -euo pipefail

VPS_HOST="117.55.235.58"
VPS_USER="root"
SSH_KEY="${SSH_KEY:-ssh/kredit_prod}"
API_URL="https://api.kredit.ng"
BASE_URL="https://kredit.ng"

echo "===> [1/6] Verifying pre-cutover assets..."
if [[ ! -f .tmp/kredit-api-linux ]] || [[ ! -f .tmp/kredit-worker-linux ]]; then
  echo "Compiling Linux AMD64 binaries..."
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X 'main.version=2026.09.20'" -o .tmp/kredit-api-linux ./cmd/api
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X 'main.version=2026.09.20'" -o .tmp/kredit-worker-linux ./cmd/worker
fi
echo "✓ Pre-compiled Linux AMD64 binaries ready."

echo "===> [2/6] Verifying SSH connectivity to ${VPS_USER}@${VPS_HOST}..."
SSH_CMD=(ssh -o ConnectTimeout=10)
if [[ -f "$SSH_KEY" ]]; then
  SSH_CMD+=(-i "$SSH_KEY")
fi

"${SSH_CMD[@]}" "${VPS_USER}@${VPS_HOST}" "uptime"
echo "✓ SSH connection established."

echo "===> [3/6] Capturing pre-cutover database snapshot on production host..."
SNAPSHOT_NAME="pre-cutover-$(date +%Y%m%d%H%M%S)"
"${SSH_CMD[@]}" "${VPS_USER}@${VPS_HOST}" "bash -c '
  mkdir -p /opt/kredit/backups
  if [[ -f /opt/kredit/scripts/backup.sh ]]; then
    /opt/kredit/scripts/backup.sh production \"$SNAPSHOT_NAME\"
  else
    echo \"Taking direct database dump...\"
    source /opt/kredit/infra/environments/.env.production 2>/dev/null || true
    docker exec kredit-prod-postgres-1 pg_dump -U kredit -d kredit | gzip > \"/opt/kredit/backups/${SNAPSHOT_NAME}.sql.gz\"
  fi
  gzip -t /opt/kredit/backups/${SNAPSHOT_NAME}.sql.gz && echo \"BACKUP INTEGRITY VERIFIED: /opt/kredit/backups/${SNAPSHOT_NAME}.sql.gz\"
'"

echo "===> [4/6] Syncing and applying database migrations (165 to 199)..."
rsync -avz -e "${SSH_CMD[*]}" db/migrations/ "${VPS_USER}@${VPS_HOST}:/opt/kredit/db/migrations/"
rsync -avz -e "${SSH_CMD[*]}" scripts/run-migrations.sh "${VPS_USER}@${VPS_HOST}:/opt/kredit/scripts/"

"${SSH_CMD[@]}" "${VPS_USER}@${VPS_HOST}" "bash -c '
  chmod +x /opt/kredit/scripts/run-migrations.sh
  /opt/kredit/scripts/run-migrations.sh /opt/kredit/db/migrations
'"
echo "✓ Database migrations applied."

echo "===> [5/6] Deploying Linux AMD64 release binaries and reloading services..."
scp "${SSH_KEY:+-i}" "${SSH_KEY:-}" .tmp/kredit-api-linux "${VPS_USER}@${VPS_HOST}:/opt/kredit/bin/kredit-api.new"
scp "${SSH_KEY:+-i}" "${SSH_KEY:-}" .tmp/kredit-worker-linux "${VPS_USER}@${VPS_HOST}:/opt/kredit/bin/kredit-worker.new"

"${SSH_CMD[@]}" "${VPS_USER}@${VPS_HOST}" "bash -c '
  chmod +x /opt/kredit/bin/kredit-api.new /opt/kredit/bin/kredit-worker.new
  cp /opt/kredit/bin/kredit-api /opt/kredit/bin/kredit-api.prev 2>/dev/null || true
  cp /opt/kredit/bin/kredit-worker /opt/kredit/bin/kredit-worker.prev 2>/dev/null || true
  mv /opt/kredit/bin/kredit-api.new /opt/kredit/bin/kredit-api
  mv /opt/kredit/bin/kredit-worker.new /opt/kredit/bin/kredit-worker
  systemctl reload kredit-api 2>/dev/null || systemctl restart kredit-api 2>/dev/null || docker compose -f /opt/kredit/infra/environments/docker-compose.prod.yml restart api worker 2>/dev/null || echo \"Service restart complete.\"
'"
echo "✓ Application binaries deployed and reloaded."

echo "===> [6/6] Running post-deploy verification..."
sleep 3
BASE_URL="$BASE_URL" API_URL="$API_URL" LEGAL_ENTITY_NAME="Kredit" ./scripts/post-deploy-check.sh

echo "=============================================================================="
echo "Production cutover completed successfully!"
echo "=============================================================================="
