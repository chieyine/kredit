# Production Deployment Guide: kredit.ng

This guide walks through deploying the complete Kredit production stack:
- **Frontend**: Vercel
- **Backend & Database**: Cloudcone VPS (Ubuntu 24.04)
- **Edge, Security & Storage**: Cloudflare (DNS, WAF, SSL, R2 Storage)
- **Domain Strategy**: `kredit.ng` as primary, with permanent 301 redirect from `kredit.com.ng`

---

## Part 1: Cloudflare Setup

### 1.1 DNS Records for `kredit.ng`

In Cloudflare Dashboard for domain **`kredit.ng`** -> **DNS** -> **Records**:

| Type | Name | Content / Target | Proxy status |
|---|---|---|---|
| **CNAME** | `@` (or root) | `cname.vercel-dns.com` | DNS only (Grey Cloud) or Proxied* |
| **CNAME** | `www` | `cname.vercel-dns.com` | DNS only (Grey Cloud) or Proxied* |
| **A** | `api` | `<Your Cloudcone VPS IP>` | **Proxied (Orange Cloud)** |

*\*Note for Vercel: Vercel can manage certificates directly when DNS is Grey Cloud, or you can keep Orange Cloud with Cloudflare SSL set to Full/Strict.*

### 1.2 Redirect `kredit.com.ng` to `kredit.ng`

In Cloudflare Dashboard for domain **`kredit.com.ng`** -> **Rules** -> **Redirect Rules** -> **Create Rule**:
- **Rule name**: `Redirect to kredit.ng`
- **When incoming requests match**: `Hostname equals kredit.com.ng` OR `Hostname equals www.kredit.com.ng`
- **Then**:
  - **Type**: Dynamic
  - **Expression**: `concat("https://kredit.ng", http.request.uri.path)`
  - **Status code**: `301 Moved Permanently`
  - **Preserve query string**: Enabled

### 1.3 Cloudflare Origin CA Certificate (End-to-End Encryption)

To ensure strict encryption between Cloudflare and your Cloudcone VPS:
1. In Cloudflare -> **SSL/TLS** -> **Origin Server** -> Click **Create Certificate**.
2. Keep defaults (RSA 2048, hostnames `*.kredit.ng, kredit.ng`, validity: 15 years).
3. Click **Create**.
4. Copy the **Origin Certificate** and save it as `origin.pem`.
5. Copy the **Private Key** and save it as `origin.key`.
6. Set **SSL/TLS encryption mode** to **Full (Strict)**.

### 1.4 Cloudflare R2 Object Storage

1. In Cloudflare Dashboard -> **R2** -> Click **Create bucket**.
2. **Bucket Name**: `kredit-production`.
3. Location: Automatic.
4. Click **Create Bucket**.
5. Click **Manage R2 API Tokens** (top right) -> **Create API Token**:
   - Token Name: `kredit-backend`
   - Permissions: **Object Read & Write**
   - Apply to bucket: `kredit-production`
6. Click **Create API Token**.
7. Note the following three values:
   - **Endpoint URL**: `https://<ACCOUNT_ID>.r2.cloudflarestorage.com`
   - **Access Key ID**
   - **Secret Access Key**

---

## Part 2: Cloudcone VPS Setup (Ubuntu 24.04)

### 2.1 Clone Repository and Bootstrap VPS

SSH into your Cloudcone VPS as root:
```bash
ssh root@<YOUR_VPS_IP>
```

Run the automated setup script:
```bash
git clone https://github.com/chieyine/kredit.git /opt/kredit
cd /opt/kredit
bash scripts/setup-vps.sh
```
This updates packages, installs Docker CE + Docker Compose plugin, configures the UFW firewall (allowing only ports 22, 80, 443), and creates the systemd auto-restart service.

### 2.2 Install Cloudflare Origin Certificates

Create the certs directory and paste the certificates generated in Step 1.3:
```bash
mkdir -p /opt/kredit/infra/environments/certs

# Paste your Origin Certificate:
nano /opt/kredit/infra/environments/certs/origin.pem

# Paste your Private Key:
nano /opt/kredit/infra/environments/certs/origin.key

# Secure the key permissions:
chmod 600 /opt/kredit/infra/environments/certs/origin.key
chmod 644 /opt/kredit/infra/environments/certs/origin.pem
```

### 2.3 Configure Production Environment (`.env.production`)

Copy the template:
```bash
cd /opt/kredit/infra/environments
cp env.production.example .env.production
nano .env.production
```

Generate secure 32-byte hexadecimal random keys using `openssl rand -hex 32` for:
- `POSTGRES_ROOT_PASSWORD`
- `KREDIT_APP_DB_PASSWORD`
- `KREDIT_WORKER_DB_PASSWORD`
- `KREDIT_BACKUP_DB_PASSWORD`
- `SESSION_SIGNING_KEY`
- `FIELD_ENCRYPTION_KEY`
- `OTP_HMAC_KEY`
- `TOKEN_HASH_KEY`
- `SETTINGS_ENCRYPTION_KEY`

Fill in your Cloudflare R2 credentials from Step 1.4:
```env
OBJECT_STORAGE_ENDPOINT=https://<CLOUDFLARE_ACCOUNT_ID>.r2.cloudflarestorage.com
OBJECT_STORAGE_BUCKET=kredit-production
OBJECT_STORAGE_REGION=auto
OBJECT_STORAGE_ACCESS_KEY=<YOUR_R2_ACCESS_KEY_ID>
OBJECT_STORAGE_SECRET_KEY=<YOUR_R2_SECRET_ACCESS_KEY>
```

Keep pilot/live-testing mode enabled so you can test end-to-end without live banking APIs:
```env
FEATURE_PRODUCTION_PILOT=true
FEATURE_APPROVED_RETENTION_POLICY=true
PILOT_APPROVAL_REFERENCE=internal-pilot-launch
RETENTION_APPROVAL_REFERENCE=internal-pilot-launch
COLLECTION_PROVIDER=mock-collection
IDENTITY_PROVIDER=mock-identity
```

### 2.4 Start the VPS Stack

Build and start the container stack:
```bash
cd /opt/kredit/infra/environments
docker compose -f docker-compose.prod.yml up -d --build
```

### 2.5 Verify Backend Status

1. Check running containers:
   ```bash
   docker compose -f docker-compose.prod.yml ps
   ```
   You should see `postgres`, `api`, `worker`, and `caddy` in healthy/running state. `migrate` and `role-init` will show exited with status 0 (completed).

2. Test local healthcheck:
   ```bash
   curl -i http://127.0.0.1:8080/api/v1/healthz
   ```

3. Test public gateway via Cloudflare:
   ```bash
   curl -i https://api.kredit.ng/api/v1/healthz
   ```
   Should return: `HTTP/2 200` with `{"service":"kredit-api","status":"ok","version":"..."}`.

---

## Part 3: Vercel Frontend Deployment

### 3.1 Import Project into Vercel

1. Log in to [Vercel](https://vercel.com).
2. Click **Add New** -> **Project**.
3. Import your GitHub repository: `chieyine/kredit`.
4. Configure Project Settings:
   - **Framework Preset**: `SvelteKit`
   - **Root Directory**: Click "Edit" and choose `web`.

### 3.2 Add Environment Variables in Vercel

Under **Environment Variables**, add the following for **Production**:

| Variable Name | Value | Description |
|---|---|---|
| `ORIGIN` | `https://kredit.ng` | Enforces authorized SSR host |
| `PUBLIC_BASE_URL` | `https://kredit.ng` | Public application URL |
| `APP_BASE_URL` | `https://kredit.ng` | Application base URL |
| `API_INTERNAL_URL` | `https://api.kredit.ng` | Cloudcone VPS API endpoint proxied by SvelteKit `/api/*` |
| `APP_ENV` | `production` | Production environment flag |
| `LEGAL_DOCUMENTS_ACTIVE` | `true` | Activates legal page indexing |
| `LEGAL_ENTITY_NAME` | `Kredit Technologies Limited` | Registered company name |
| `LEGAL_SERVICE_ADDRESS` | `Lagos, Nigeria` | Registered address |
| `LEGAL_CONTACT_EMAIL` | `support@kredit.ng` | Support address |
| `PRIVACY_CONTACT_EMAIL` | `privacy@kredit.ng` | Privacy inquiries address |
| `LEGAL_EFFECTIVE_DATE` | `2026-09-11` | Effective legal publication date |
| `TERMS_VERSION` | `supplier-terms-v1` | Published terms version |
| `PRIVACY_VERSION` | `privacy-v1` | Published privacy version |

### 3.3 Deploy and Add Custom Domain

1. Click **Deploy**.
2. Once the build completes, go to **Settings** -> **Domains**.
3. Add `kredit.ng` and `www.kredit.ng`.
4. Ensure DNS is verified in Cloudflare.

---

## Part 4: Final End-to-End Verification

1. **Root Website**: Open `https://kredit.ng` in a browser.
2. **Domain Redirect**: Open `https://kredit.com.ng` -> verify it automatically redirects (301) to `https://kredit.ng`.
3. **API Proxy**: Open `https://kredit.ng/api/v1/healthz` -> verify it proxies to the VPS backend and returns `200 OK`.
4. **Interactive Demo**: Visit `https://kredit.ng/demo` and test the sample credit sale workflow.
5. **Worker Logs**: On the VPS, run `docker compose -f docker-compose.prod.yml logs -f worker` to verify background jobs execute cleanly.
