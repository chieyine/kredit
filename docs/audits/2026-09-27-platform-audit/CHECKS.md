# Code-audit and implementation checks — 27 September 2026

The audit recommendations, one-person-per-business rule and subsequent pre-launch compatibility cleanup are implemented locally. No agents were used. No tests were added or executed; obsolete feature tests were removed and retained fixtures had references to removed interfaces cleaned up. The checks below inspect source, compilation, static diagnostics and build output; they do not establish browser usability or transactional correctness.

## Implementation checks

| Check | Result | Scope and limits |
|---|---|---|
| Go compilation | Passed after cleanup | `go build ./cmd/...`, with a writable temporary build cache. The final deletion of obsolete approval-error handling was compiled again. |
| Go static lint | Passed after cleanup, 0 issues | `golangci-lint run --tests=false --disable=unused --timeout=12m`, with a temporary cache and two execution threads. The subsequent deletion of obsolete approval-error handling received formatting and compilation. |
| Svelte/TypeScript diagnostics | Passed, 0 errors and 0 warnings | Full web workspace after removing the old screens and compatibility code. The final deletion of obsolete approval-error handling received targeted lint and production compilation. |
| Frontend ESLint | Passed | Full production source directory, with warnings treated as failures. The final mutation-error cleanup also passed targeted lint. |
| Source formatting | Passed | Changed Go files formatted with `goimports`; changed frontend files and shared JavaScript scripts formatted with the repository's configuration. |
| Frontend production build | Passed after final cleanup | Client/server compilation, prerendering and the configured Vercel adapter completed, including the last removal of obsolete approval-error handling. No deployment was performed. |
| Repository file integrity | Passed | 1,434 owned files passed the file/encoding/JSON/link/brand checker. This is a structural check, not behavior verification; historical audit links to deleted files now explicitly identify those removals. |
| Legal publication consistency | Passed | Checked-in publication metadata and backend document versions agree. Pre-launch legal defaults and fallback pages now describe one business owner. No external publication was performed. |
| API operation alignment | Passed | 76 statically resolved frontend calls, 288 backend routes and 287 API operations under the shared inventory's rules. Dynamic calls and runtime responses are outside this check. |
| Route-to-screen inventory | Passed | 285 routes classified, including 39 with no separate screen and an explicit reason. Path-based classification is weaker than operation-level verification. |
| Content audit | Passed | The combined repository audit completed: 12 articles, 4,690 blog words, route discovery and required legal sections checked. |
| Script/config syntax | Passed | Five changed shell scripts parsed, one changed JSON file decoded, four JavaScript modules syntax-checked, and the final OpenAPI YAML parsed with 255 paths and no unresolved local schema references. No full OpenAPI semantic validator was run. |
| Terraform | Passed | Terraform 1.11.0 formatting, initialization without a backend using the read-only lockfile, and configuration validation. The Kubernetes 2.38.0 provider lock covers Linux/macOS on amd64/arm64. No plan, apply or cluster access was performed. |
| Mechanical Go file moves | Passed at the decomposition stage | AST comparison preserved 187 declarations across the original credit store, persistence and HTTP-handler files and their extracted files. Subsequent intentional deletion of obsolete approval handling is separate from that earlier result. |
| Removed-schema references | Passed static inspection | Latest SQL function definitions were compared with the tables removed by migration 213; no remaining function referenced a removed table. This is not SQL execution or a full database dependency proof. |
| Patch whitespace | Passed | `git diff --check`. |

`unused` analysis was excluded alongside test loading because production declarations used only by excluded tests otherwise produce misleading unused reports. This is not the default lint configuration with tests loaded.

## Audit coverage snapshot

The [file register](FILES.csv) preserves the completed audit's **pre-implementation** inventory and hashes; it is not a fresh hash manifest for the modified code.

- 1,465 unique baseline paths were inventoried.
- 870 production source/configuration paths have cumulative complete-file review evidence: 197 current full-file inspections and 673 exact matches to the earlier complete-file review's `reviewed_sha256` values.
- All 28 new/changed production paths at audit completion, relative to that earlier review, received full inspection. Of the 673 reused reviews, 40 also received focused inspection and 633 static screening; hash matching does not independently re-prove the earlier review's quality.
- 595 other paths remained inventory-only. This does not claim full review of test behavior, third-party dependencies or every historical document.
- Implementation introduced and changed additional files, which received source inspection and the checks above. They are recorded in the working-tree patch and [implementation record](IMPLEMENTATION.md), not relabelled as original audit evidence.

The [report](REPORT.md) preserves the original findings and their corrections. [IMPLEMENTATION.md](IMPLEMENTATION.md) maps F01–F16 and M01–M08 to the final changes and retained boundaries.

## Environment and unexecuted checks

| Tool | Local version | Declared version |
|---|---|---|
| Node.js | 24.11.0 | >=24.20.0 and <25 |
| pnpm | 11.17.0 | 11.17.0 |
| Go | 1.27.1 | 1.26.8 |

Exact deployment-toolchain compatibility is unverified. Application dependencies are unchanged; the infrastructure provider lockfile was added using Terraform's signed provider checksums.

No unit, integration, end-to-end, browser, accessibility-device, clean-volume startup or external-provider behavior checks were performed. No database was inspected or changed, and migration 213 was not executed. No deployment, provider-service changes or financial writes occurred. No dependency vulnerability scan or fresh exhaustive security assessment is claimed.

Migration 213 is forward-only, removes unused staff schema without archiving it, and requires ownership preflight and a backup as documented in the [cutover guide](../../operations/production-cutover-runbook.md). Build/static success is not evidence that migration or rollout has succeeded.
