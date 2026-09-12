#!/usr/bin/env bash
# ==============================================================================
# Kredit Cloudcone VPS Bootstrap Script (Ubuntu 24.04 LTS)
# Run as root: sudo bash setup-vps.sh
# ==============================================================================
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root (use: sudo bash setup-vps.sh)" 1>&2
   exit 1
fi

echo "===> [1/6] Updating system packages..."
apt-get update -y
DEBIAN_FRONTEND=noninteractive apt-get upgrade -y
apt-get install -y curl git ufw ca-certificates openssl jq

echo "===> [2/6] Installing Docker CE and Docker Compose plugin..."
install -m 0755 -d /etc/apt/keyrings
if [[ ! -f /etc/apt/keyrings/docker.asc ]]; then
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
    chmod a+r /etc/apt/keyrings/docker.asc
fi

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}") stable" | \
  tee /etc/apt/sources.list.d/docker.list > /dev/null

apt-get update -y
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

systemctl enable docker
systemctl start docker

echo "===> [3/6] Configuring UFW Firewall..."
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp comment 'SSH'
ufw allow 80/tcp comment 'HTTP'
ufw allow 443/tcp comment 'HTTPS'
ufw --force enable

echo "===> [4/6] Setting up /opt/kredit deployment directory..."
mkdir -p /opt/kredit/infra/environments/certs
mkdir -p /opt/kredit/infra/environments/db-certs

echo "===> [5/6] Generating systemd service for Kredit stack..."
cat << 'EOF' > /etc/systemd/system/kredit.service
[Unit]
Description=Kredit Production Container Stack
Requires=docker.service
After=docker.service network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/kredit/infra/environments
ExecStart=/usr/bin/docker compose --env-file .env.production -f docker-compose.prod.yml up -d
ExecStop=/usr/bin/docker compose --env-file .env.production -f docker-compose.prod.yml stop
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable kredit.service

echo "===> [6/6] VPS setup complete!"
echo "Next steps:"
echo "1. Place your Cloudflare Origin Certificate at:"
echo "   /opt/kredit/infra/environments/certs/origin.pem"
echo "   /opt/kredit/infra/environments/certs/origin.key"
echo "2. Copy env.production.example to /opt/kredit/infra/environments/.env.production"
echo "3. Copy env.runtime.example to .env.runtime and complete the runtime settings."
echo "4. Install PostgreSQL TLS certificates as described in the production guide."
echo "5. Run: cd /opt/kredit/infra/environments && docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build"
