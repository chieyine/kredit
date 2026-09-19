# Review limits and secondary observations

The project-file reading pass is complete. This is a static audit, not a guarantee of correctness or a launch certification. Confirmed, actionable application findings are separated into FINDINGS.md. The following observations are retained so that uncertainty is not silently converted into approval.

## Scope and evidence

- All 1,140 tracked files and seven local project files are accounted for. The six local environment/SSH files received metadata-only inspection; their values and credential bodies were not opened. The extra tracked-looking draft `Claude outputs/ci.yml` was read as an untracked project file.
- Every application source file and test source file was read. No tests, builds, linters, database commands, provider calls or application/browser runs were performed. Existing test/build logs were inspected only as historical artifacts.
- Historical registers and logs received structured artifact inspection. Their previous review claims do not substitute for this audit's independent current-source reads. The archived changes.patch received provenance and complete file/hunk inventory inspection, not another line-by-line review of obsolete code.
- The 54,144 dependency, generated/temporary and Git-metadata paths are recorded in inventory.json but were not individually source-audited. Local databases and third-party implementation bodies are not covered. Generated project-owned source tracked in the repository was included in the project review.
- Live provider contracts, supported-bank lists, account entitlements, deployment settings, DNS, dependency advisories and legal approvals were not independently refreshed. Documentation that mentions them was read as local project content. No live financial or legal conclusion follows from that reading.
- Recorded SHA256 values bind the review to the inspected bytes. A final comparison found no reviewed-file changes. Git status showed only this new audit directory. New audit artifacts are deliverables, not recursively counted source.

## Unconfirmed hypotheses retained for future investigation

These are not part of the 22 confirmed findings and should not be treated as defects without additional evidence.

| Area | Remaining uncertainty |
| --- | --- |
| Session-scoped idempotency | HTTP scopes include session/authentication state. Domain-specific deduplication must be established before claiming that MFA rotation causes duplicate financial effects. |
| Development payment compensation | Fresh payment identities interact with fixed compensation keys. This is a possible memory-adapter retry discrepancy; it is not evidence of a production PostgreSQL payment failure. |
| Health probes | API liveness and provider/database readiness differ. Whether a deployment routes traffic before readiness depends on its actual routing configuration. |
| Provider and notification concurrency | Remote lookup delays, provider callback ordering and delivery leases warrant runtime evidence; no universal race or provider failure is claimed from a suspicious-looking path alone. |
| Owner/role lifecycle | Some handlers authorize before a later write. Existing lifecycle locks, database invariants and transaction rechecks must be considered before asserting exploitable stale authority. |
| Fee UI business switches | Old fee rows or delayed responses may remain during a selection change. Backend identifier/scope validation limits consequences; an incorrect financial write has not been demonstrated. |
| Mandate renewal navigation | Accepted sales and the requests-list filter may make renewal guidance inconvenient. The complete alternate-navigation impact has not been established as a separate finding. |
| Document/media size and remote URLs | Direct-upload length binding, provider-returned media URLs and truncation have relevant caller/provider boundaries. No unauthorized upload, token disclosure or exploitable fetch is claimed here. |

## Test-source assurance gaps — no tests executed

These are specific source inconsistencies recorded in the file ledger, not reported test-run failures:

- `internal/buyers/audit_postgres_test.go`: acceptance fixtures omit consent/legal-version fields required by the current input validation, so the intended identity-reuse assertion may never be reached.
- `internal/platformops/directory_postgres_test.go`: direct directory calls omit the operator context required by the current store.
- `internal/platformsettings/store_test.go`: a custom connector fixture omits the explicit connector adapter while email defaults to the native Sendly adapter.
- `internal/usercontrol/store_test.go`: the recovery delivery-failure scenario uses the seven-character reason `verified`, below the current minimum; validation can reject before the delivery branch.
- `web/tests/audit-fixes.spec.ts`: invitation-preview fixtures omit required legal/identity-notice fields; acceptance also needs the presented consent action.
- `web/tests/audit-full-form.spec.ts`: successful upload fixture supplies a hash without the document ID required by the form before sale creation.
- Some financial-browser tests check an obsolete empty-state phrase, only HTTP 503 logout failure, or opening a receipt dispute without resolving it. Those assertions do not cover F002, F019 or F021.

## Documentation and local metadata

- The compliance inventory has 1,424 field records; all legal bases and retention annotations remain pending approval. Some classifications appear heuristic (for example job counts as identity and an OTP HMAC as commercial). This catalog is not evidence that every stored value is correctly classified or encrypted.
- The secondary OpenAPI document is a stale full specification, despite its README describing it as an index. The main specification also omits important request/response shapes. Migration numbers and provider statements in several runbooks lag the current implementation.
- Historical completion reports sometimes conflict with their own earlier checkpoints. Their date, scope, runtime and later repair records must remain explicit. Do not promote old audit findings into current findings without tracing today's source.
- The environment file has mode 0644; public SSH material and known_hosts have mode 0666. Private keys are 0600. Git and Docker exclusions cover these paths. Permissions merit tightening on a shared workstation, but no credential contents, historical leak or exposure to another user was established.
