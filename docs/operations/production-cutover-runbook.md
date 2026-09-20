# Kredit Production Cutover Runbook & Verification Protocol

This runbook specifies the step-by-step cutover sequence for deploying the 8 architectural blueprint tracks (migrations 165 through 199, versioned ERP reconciliation, item-level order lifecycle, dual-approval branch boundaries, and enterprise reporting) to the production infrastructure (`kredit.ng` and API host `117.55.235.58`).

> [!IMPORTANT]
> **Strict Pre-Execution Rule**: Do NOT run this cutover or execute remote changes on the production VPS until explicitly commanded by the operator. All steps below must be executed in the exact sequence specified.

---

## 1. Architectural Scope & Invariants

The cutover encompasses:
- **Track 1**: Canonical organization consolidation (`195_canonical_organization_consolidation.sql`).
- **Track 2**: Delegated drawdowns, specialist claims, and spend ceilings (`196_delegated_drawdown_purchasing.sql`).
- **Track 3**: Branch-scoped enforcement and dual credit approval for drawdowns (`197_branch_drawdown_approvals.sql`).
- **Track 4**: Item-level order lifecycles, partial shipments, delivery receipts, and credit notes (`198_item_level_lifecycles.sql`, `internal/orders`).
- **Track 5**: Dual-control opening balance imports and terms staging (`199_opening_balance_import_batches.sql`, `internal/buyers/terms_import.go`).
- **Track 6**: Versioned ERP contract ingestion (V1/V2) and ledger reconciler (`internal/erp`).
- **Track 7**: Multi-branch exposure analytics, portfolio health scoring, and signed enterprise reporting (`internal/reports/enterprise.go`, `web/src/routes/workspace/reports`).
- **Track 8**: Production cutover execution, verification script, and rollback protocols.

### Architectural Invariants
1. **Zero Phantom Debt**: Staged terms, import batches, and draft purchase orders create no debt obligations, payment mandates, or ledger entries until verified acceptance or drawdown release.
2. **Double-Entry Ledger Immutability**: No ledger entry or journal item is ever modified or deleted (`append-only`).
3. **Integer Kobo Precision**: All financial arithmetic uses 64-bit integer kobo (`bigint`) with zero floating-point math.
4. **Tenant Isolation**: PostgreSQL `FORCE ROW LEVEL SECURITY` remains active and verified across all new tables.

---

## 2. Phase 1: Pre-Cutover Verification Checklist

Before initiating cutover, the operator must execute the following readiness checks:

### 2.1 Database State & Backup Verification
Run on the production database host:
```bash
# 1. Capture full pre-cutover database snapshot
./scripts/backup.sh production pre-cutover-$(date +%Y%m%d%H%M%S)

# 2. Verify snapshot archive integrity
gzip -t /opt/kredit/backups/pre-cutover-*.sql.gz && echo "BACKUP INTEGRITY VERIFIED"

# 3. Check current schema version
psql "$DATABASE_DIRECT_URL" -c "SELECT version, applied_at FROM schema_migrations ORDER BY version DESC LIMIT 5;"
```

### 2.2 Provider Adapter Health Checks
Verify external provider responsiveness:
```bash
# Verify Mono Open Banking Direct Debit connectivity
curl -sf -H "mono-sec-key: $MONO_SECRET_KEY" https://api.withmono.com/v1/health || echo "WARNING: Mono check failed"

# Verify Termii SMS Provider connectivity
curl -sf "https://api.ng.termii.com/api/check/balance?api_key=$TERMII_API_KEY" || echo "WARNING: Termii check failed"

# Verify QoreID KYC/KYB connectivity
curl -sf -H "Authorization: Bearer $QOREID_TOKEN" https://api.qoreid.com/health || echo "WARNING: QoreID check failed"
```

---

## 3. Phase 2: Database Migration Execution

Migrations must be applied in strict ascending order from 165 to 199.

### 3.1 Migration Command
```bash
# On the VPS / production orchestration node:
source /opt/kredit/infra/environments/.env.production
./scripts/run-migrations.sh /opt/kredit/db/migrations
```

### 3.2 Schema & RLS Verification Query
Execute the following verification query to ensure all new blueprint tables have RLS enforced:
```sql
SELECT 
    relname AS table_name,
    relrowsecurity AS rls_enabled,
    relforcerowsecurity AS rls_forced
FROM pg_class
JOIN pg_namespace ON pg_namespace.oid = pg_class.relnamespace
WHERE nspname = 'app'
  AND relname IN (
    'order_line_items',
    'order_shipments',
    'order_shipment_items',
    'order_delivery_receipts',
    'order_credit_notes',
    'tradeline_drawdown_approvals',
    'partner_terms_import_batches',
    'partner_terms_import_rows'
  )
ORDER BY relname;
```
*Expected Result: `rls_enabled = true` and `rls_forced = true` on all rows.*

---

## 4. Phase 3: Binary & Application Cutover

### 4.1 Linux AMD64 Binary Compilation
```bash
# Build production release binaries with optimizations
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X 'main.version=2026.09.20'" -o .tmp/kredit-api-linux ./cmd/api
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X 'main.version=2026.09.20'" -o .tmp/kredit-worker-linux ./cmd/worker
```

### 4.2 Zero-Downtime Atomic Service Swap
```bash
# 1. Transfer binaries to release directory
cp .tmp/kredit-api-linux /opt/kredit/bin/kredit-api.new
cp .tmp/kredit-worker-linux /opt/kredit/bin/kredit-worker.new
chmod +x /opt/kredit/bin/kredit-api.new /opt/kredit/bin/kredit-worker.new

# 2. Atomic rename
mv /opt/kredit/bin/kredit-api.new /opt/kredit/bin/kredit-api
mv /opt/kredit/bin/kredit-worker.new /opt/kredit/bin/kredit-worker

# 3. Reload systemd services (graceful worker drain and API reload)
sudo systemctl reload kredit-api || sudo systemctl restart kredit-api
sudo systemctl restart kredit-worker
```

### 4.3 Frontend Web Deployment
```bash
# Build and deploy SvelteKit frontend
cd web && pnpm build
# Sync assets to web server / Vercel deployment
```

---

## 5. Phase 4: Post-Cutover Verification Protocol

Execute the automated verification script:
```bash
./scripts/post-deploy-check.sh https://api.kredit.ng
```

The script performs:
1. Public health check (`GET /api/v1/healthz` -> HTTP 200 `status: ok`).
2. Readiness probe check (`GET /api/v1/organizations/{id}/readiness`).
3. Enterprise risk endpoint check (`GET /api/v1/organizations/{id}/reports/enterprise`).
4. Dual-approval route registration (`GET /api/v1/organizations/{id}/credit-approvals`).
5. Database connection pool metrics and active migration version.

---

## 6. Phase 5: Rollback Procedures

If any critical failure occurs (unrecoverable database deadlock, schema violation, provider failure > 5 minutes):

### 6.1 Service Binary Rollback
```bash
# Revert to previous binary snapshot
mv /opt/kredit/bin/kredit-api.prev /opt/kredit/bin/kredit-api
mv /opt/kredit/bin/kredit-worker.prev /opt/kredit/bin/kredit-worker
sudo systemctl restart kredit-api kredit-worker
```

### 6.2 Database Rollback
Migrations 195–199 are additive (new tables and non-destructive column additions). They do not break backwards compatibility with older API versions. If complete database rollback is necessary:
```bash
# Restore verified pre-cutover backup
dropdb -h localhost -U postgres kredit_prod
createdb -h localhost -U postgres kredit_prod
gunzip -c /opt/kredit/backups/pre-cutover-*.sql.gz | psql -h localhost -U postgres kredit_prod
```
