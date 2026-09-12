# Deploy Kredit on kredit.ng

This deployment uses Vercel for the SvelteKit frontend and the VPS Compose stack for the API, worker and PostgreSQL. Cloudflare protects `api.kredit.ng` and holds private documents in R2. These are deployment instructions, not evidence that production is already configured or approved.

## Domains and certificates

Add `kredit.ng` to the Vercel project with root directory `web`. Add `www.kredit.ng` as a redirect to the apex domain. Use the exact A/CNAME records shown in that project's Domains settings; do not copy a generic Vercel target. Keep website records DNS-only while Vercel provisions its certificate. Keep existing mail and domain-verification records. [Vercel domain configuration](https://vercel.com/docs/domains/working-with-domains/add-a-domain).

Create a proxied A record for `api.kredit.ng` pointing at the VPS. Configure Cloudflare Full (Strict) and an Origin CA certificate covering `api.kredit.ng`. Install its certificate and key as `infra/environments/certs/origin.pem` and `origin.key`, with the key readable only by the Caddy container's operator. Origin CA certificates protect the Cloudflare-to-origin connection; they are not general browser certificates. [Cloudflare Origin CA](https://developers.cloudflare.com/ssl/origin-configuration/origin-ca/).

If Kredit controls the old `kredit.com.ng` zone, use a permanent redirect from both its apex and `www` to `https://kredit.ng`, preserving the path and query string. The old domain is only a redirect, never the application origin or a provider callback URL.

## Prepare the VPS

Review and run `scripts/setup-vps.sh` on the intended Ubuntu host after placing the approved release in `/opt/kredit`. The script installs Docker, configures ports 22/80/443, and creates the systemd service. It does not provide database certificates, approvals or provider credentials. PostgreSQL and the API have no published host ports.

The Compose file uses PostgreSQL 18 with the volume mounted at `/var/lib/postgresql`. For an existing database, identify its actual PostgreSQL version and volume contents, take a verified backup and follow a PostgreSQL upgrade/restore procedure. Changing a volume mount or image tag is not a database upgrade. Never delete the production volume to make startup pass. [Official PostgreSQL image](https://hub.docker.com/_/postgres).

## PostgreSQL TLS

Provide a dedicated database server certificate with `DNS:postgres` in its subject alternative names, signed by a trusted private CA. Install these files in `infra/environments/db-certs/`:

| File | Purpose |
|---|---|
| `server.crt` | Server certificate and any intermediate chain |
| `server.key` | Matching server private key |
| `root.crt` | Public CA certificate used by clients |

Keep the CA private key outside the deployment directory and containers. Make `server.key` owned by the PostgreSQL process user with mode `0600`, or root-owned with mode `0640` and a group that PostgreSQL can read. Determine the UID/GID from the exact selected image; do not assume the host's `postgres` user matches. Public certificates can use mode `0644`. [PostgreSQL TLS requirements](https://www.postgresql.org/docs/current/ssl-tcp.html).

The Compose stack enables server TLS and rejects non-TLS TCP database connections. API, worker, migration and role setup connections use `sslmode=verify-full` with the mounted CA. Only PostgreSQL receives the server key. Do not switch these connections to `sslmode=disable` to work around a certificate error.

## Separate deployment and runtime secrets

From `/opt/kredit/infra/environments`, create two private files:

```sh
cp env.production.example .env.production
cp env.runtime.example .env.runtime
chmod 600 .env.production .env.runtime
```

`.env.production` contains only the four deployment database passwords. Use independent, random hexadecimal values; this avoids URL-escaping mistakes in Compose's database URLs. `.env.runtime` contains application keys and provider settings. Generate each application key independently, including `FRONTEND_PROXY_SIGNING_KEY`. Preserve the encryption key IDs and established encryption keys across releases.

The runtime must not receive the migration-owner connection string or password. Compose injects restricted application and worker connections. `DATABASE_DIRECT_URL` is a maintenance setting, not an API startup requirement. Do not paste secrets into support notes or commit either environment file or certificate directory.

Complete the runtime template from the root `.env.example` and `internal/config/config.go`. Configure a reachable OTLP collector, private R2 storage and the contracted identity provider. Configure a document scanner before permitting document use; pending files cannot be downloaded or submitted as evidence.

For R2 use its account endpoint, `auto` region and credentials limited to the private bucket. The adapter omits the unsupported S3 encryption header for R2; R2 still encrypts stored objects. Keep the bucket private and use short-lived signed downloads. [R2 S3 compatibility](https://developers.cloudflare.com/r2/api/s3/api/).

## Sendly email

Use `NOTIFICATION_EMAIL_ENDPOINT=https://api.sendlyai.com/v1/messages`, the live Sendly API key, a verified sender at `kredit.ng` in `NOTIFICATION_EMAIL_FROM`, and the assigned `SENDLY_WEBHOOK_SECRET`. Configure the same values in platform settings if that connector is managed there. Publish the DNS records Sendly provides for the sender domain; do not invent SPF/DKIM values.

Point Sendly's email receipt webhook at `https://api.kredit.ng/api/v1/webhooks/notifications/email` (confirm the route in `internal/web/server.go` before enabling it). The receiver authenticates the signed envelope. The worker retrieves message status from Sendly and records delivered status only for the saved message and recipient. An API acceptance is not delivery evidence. SMS and WhatsApp require their own contracted adapters; Sendly email does not establish either integration. [Sendly documentation](https://developer.sendlyai.com/).

## Provider and launch gates

Use the production Mono account and its approved Sweep permissions. Set the redirect and webhook URLs to the implemented `kredit.ng`/`api.kredit.ng` routes. Follow [the Mono runbook](../runbooks/mono-sweep.md). Never use a development mock as production identity or collection evidence.

Keep live collection, direct supplier settlement and other separately gated capabilities disabled until the actual provider implementation, account permission and approval references are present. Set production-pilot and retention approval gates only after the real approvals exist. Placeholder references such as `internal-pilot-launch` are not approvals. A template deliberately cannot start an approved production service by itself.

## Frontend settings

Set these private server variables in Vercel Production:

| Variable | Value |
|---|---|
| `APP_ENV` | `production` |
| `ORIGIN` | `https://kredit.ng` |
| `API_INTERNAL_URL` | `https://api.kredit.ng` |
| `FRONTEND_PROXY_SIGNING_KEY` | The same independently generated key as the API |

The frontend signs the original client address forwarded to the API. This key must never use a public environment prefix or enter browser code. Application links use `https://kredit.ng`.

Initial legal publication metadata lives in `web/src/lib/server/legal-publication.ts` and `internal/legalpublication/versions.go`. Initial terms and privacy versions are `supplier-terms-v2-2026-09-07` and `privacy-v2-2026-09-07`. Published updates use the website-content workflow and preserve earlier versions. Old `LEGAL_*`, `TERMS_VERSION` and `PRIVACY_VERSION` environment examples do not change these publications. Review the actual text, entity details, contact and effective date; do not treat a configured page as legal approval.

## Start the approved release

Run from `/opt/kredit/infra/environments` only after the repair review and essential checks are complete:

```sh
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

Compose runs all migrations, then applies runtime-role grants before starting the API and worker. The systemd service uses the same environment-file argument. Every Compose command in this directory must include it. Do not use `docker compose config` in shared logs: rendered configuration can include secrets.

Inspect startup without exposing environment values:

```sh
docker compose --env-file .env.production -f docker-compose.prod.yml ps
docker compose --env-file .env.production -f docker-compose.prod.yml logs --tail=100 migrate role-init api worker
```

The migration and role containers must exit successfully; API and worker must become healthy. Check the public health route through both `api.kredit.ng` and the frontend `/api` proxy. There is no host `127.0.0.1:8080` listener in this stack. Verify domain redirects and a narrowly scoped sign-in/provider flow only when the final verification phase is authorized. Health alone does not establish launch readiness.

Keep the previous image digests and release record. Database recovery uses a verified backup and forward repairs; do not automatically run financial migrations backwards. Follow [the go-live runbook](../release/go-live-runbook.md) for approval evidence and cutover ownership.
