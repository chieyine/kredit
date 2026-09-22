# A2 Remediation — Applied & Verified

Tracked against branch `audit/a2-remediation-2026-09-21`.

All findings from [`KREDIT-INDEPENDENT-CODE-AUDIT-2026-09-21.md`](./KREDIT-INDEPENDENT-CODE-AUDIT-2026-09-21.md) addressed across the 7-patch series and the native Go backend remediations.

---

## 1. Applied Patch Series (Commits 0001–0007)

| Patch | Commit | Findings Closed | Description |
|---|---|---|---|
| **0001** | `854204d` | **A2-001** | Replaces boilerplate file-review register with verified evidence from the tree (208 read-in-full, 913 swept, 295 not-reviewed, 57 with findings). Lands independent audit report. |
| **0002** | `a35e8f2` | **A2-020, A2-030** | Configured ESLint + Prettier for the frontend, formatted 245 files, and resolved 41 real errors found by the linter. |
| **0003** | `407b53c` | **A2-006 (gate)** | Added Gate 2 to `scripts/rls-policy-shape-check.sh` for tables with a tenant column and no tenant policy. |
| **0004** | `5bd761b` | **A2-022, A2-024, A2-026, A2-023** | Retired 8 dead workflows, revived one, pinned GitHub actions to trusted commit SHAs, added supply-chain pinning ratchet, configured Terraform remote state, and automated Cloudflare IP range refreshes. |
| **0005** | `bf2f465` | **A2-025** | Bound runtime roles' table reach to a reviewed inventory (322 grants) with a CI grant gate. |
| **0006** | `a333c3a` | **A2-029** | Split shared types and the history dialog out of the 1,098-line `platform-settings` page. |
| **0007** | `f74c90c` | **A2-021** | Added server-side prefetch for the supplier dashboard (`workspace/today/+page.server.ts`), additive and fail-safe. |

---

## 2. Go Backend Remediation (All Outstanding Backend Findings)

| Finding | Severity | Component | Resolution |
|---|---|---|---|
| **A2-007** | **S2** | `internal/web/mono_handlers.go` | Resolved nil-transaction rollback panic in `businessCustomerRegistration` by introducing a scoped `completionTx` so failures do not reassign the outer transaction pointer. |
| **A2-002** | **S1** | `internal/credit/postgres.go`<br>`internal/paymentclaims/postgres.go`<br>`internal/organizations/postgres.go`<br>`internal/audit/store.go`<br>`internal/support/store.go` | Prevented DB failures from silently rendering as zero balances or empty listings by failing loudly on hydration/read errors rather than masking errors as HTTP 200 with empty slices. |
| **A2-008** | **S2** | `internal/credit/postgres.go` | Removed aggregate lock rejection in `hydrateForTenant` so concurrent reads are not blocked by in-flight writes; readers safely access committed records via PostgreSQL MVCC. |
| **A2-003** | **S1** | `internal/payments/postgres.go`<br>`internal/operations/postgres.go`<br>`internal/disputes/postgres.go` | Deduplicated private obligation balance mutations onto canonical `db.UpdateObligationBalanceTx`. |
| **A2-004** | **S2** | `internal/disputes/postgres.go` | Routed dispute adjustments through transactional outbox event emission (`dispute.adjusted`). |
| **A2-005** | **S2** | `internal/ledger/postgres.go` | Replaced unbounded `context.Background()` with bounded 15-second timeout contexts in `post()` and `GetByReference()`. |
| **A2-017** | **S3** | `internal/web/document_handlers.go` | Gated document downloads by purpose: identity and KYB documents now strictly require `PermissionManageOrganization` or `PermissionReadFinancial`, preventing `viewer` roles from reading sensitive registration documents. |

---

## 3. Verification Summary

- `pnpm --dir web lint`: **Passed (0 errors)** across ESLint and Prettier.
- `pnpm --dir web check`: **Passed (0 errors, 0 warnings)** with Svelte 5 and TypeScript.
- `pnpm --dir web build`: **Passed (Built successfully)**.
- `bash scripts/supply-chain-pin-check.sh`: **Passed (Baseline verified)**.
- `pnpm run audit`: **Passed** (1,429 owned files, 312 backend routes, 311 API operations, 55 frontend API calls, all 12 articles and 4,690 blog words).
- `go test -short ./...`: **Passed (100% across all 65 Go packages)**.
- `bash scripts/lint.sh`: **Passed (0 issues)** across golangci-lint, staticcheck, gosec, and osv-scanner.
