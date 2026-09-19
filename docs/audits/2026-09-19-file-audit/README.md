# Kredit source audit

**Project-file review complete: 22 findings, including five high-priority findings.** No agents or tests were used. No application code was changed.

The audit accounts for **1,147 project files**: 1,070 full text reads, 24 individual image inspections, 47 structured artifact reviews and six metadata-only environment/SSH reviews. This includes all 1,140 tracked files. Dependency/cache/Git paths were inventoried separately, not individually code-reviewed.

Start with [the findings](FINDINGS.md), then [the file-by-file register](FILE-BY-FILE.md). Each finding explains the trigger, effect, evidence and repair direction. The register records every project file, including those without a separate finding.

| Finding | Priority | Problem |
| --- | --- | --- |
| [F001](FINDINGS.md#f001--p1--identical-journal-retries-fail-because-posting-order-is-random) | P1 | Identical journal retries fail because posting order is random |
| [F002](FINDINGS.md#f002--p1--storage-failures-can-produce-a-false-successful-sign-out) | P1 | Storage failures can produce a false successful sign-out |
| [F009](FINDINGS.md#f009--p1--every-pilot-scorecard-fails-on-an-unimplemented-metric) | P1 | Every pilot scorecard fails on an unimplemented metric |
| [F012](FINDINGS.md#f012--p1--terraform-production-web-deployment-omits-a-required-signing-secret) | P1 | Terraform production web deployment omits a required signing secret |
| [F017](FINDINGS.md#f017--p1--default-monthly-instalment-terms-can-become-impossible-to-activate-after-release) | P1 | Default monthly instalment terms can become impossible to activate after release |
| [F003](FINDINGS.md#f003--p2--backup-retention-deletes-unrelated-old-files) | P2 | Backup retention deletes unrelated old files |
| [F004](FINDINGS.md#f004--p2--placeholder-r2-configuration-reports-successful-offsite-backup) | P2 | Placeholder R2 configuration reports successful offsite backup |
| [F005](FINDINGS.md#f005--p2--interrupted-bank-authorization-sends-customers-to-a-missing-support-page) | P2 | Interrupted bank authorization sends customers to a missing support page |
| [F006](FINDINGS.md#f006--p2--authorized-operators-receive-an-incomplete-bank-recovery-queue) | P2 | Authorized operators receive an incomplete bank-recovery queue |
| [F007](FINDINGS.md#f007--p2--consumer-sale-creators-cannot-reopen-the-sales-they-create) | P2 | Consumer sale creators cannot reopen the sales they create |
| [F008](FINDINGS.md#f008--p2--consumer-retry-helper-discards-the-key-for-an-in-progress-write) | P2 | Consumer retry helper discards the key for an in-progress write |
| [F010](FINDINGS.md#f010--p2--buyer-silence-guardrail-excludes-current-automatic-activations) | P2 | Buyer-silence guardrail excludes current automatic activations |
| [F011](FINDINGS.md#f011--p2--optional-report-view-events-always-fail-with-nil-metadata) | P2 | Optional report-view events always fail with nil metadata |
| [F013](FINDINGS.md#f013--p2--whitespace-bypasses-seller-feedback-membership-checks) | P2 | Whitespace bypasses seller feedback membership checks |
| [F014](FINDINGS.md#f014--p2--supplier-sale-listings-expose-buyer-bank-authorization-links) | P2 | Supplier sale listings expose buyer bank authorization links |
| [F015](FINDINGS.md#f015--p2--restored-credit-mandates-cannot-be-accepted-by-their-provider-reference) | P2 | Restored credit mandates cannot be accepted by their provider reference |
| [F016](FINDINGS.md#f016--p2--a-provider-lookup-failure-prevents-recording-local-cancellation) | P2 | A provider lookup failure prevents recording local cancellation |
| [F018](FINDINGS.md#f018--p2--credit-line-expiry-blocks-receipt-of-goods-already-dispatched) | P2 | Credit-line expiry blocks receipt of goods already dispatched |
| [F019](FINDINGS.md#f019--p2--reporting-a-drawdown-delivery-problem-has-no-resolution-path) | P2 | Reporting a drawdown delivery problem has no resolution path |
| [F020](FINDINGS.md#f020--p2--a-declined-or-cancelled-offer-makes-the-buyer-balance-permanently-unconfirmed) | P2 | A declined or cancelled offer makes the buyer balance permanently unconfirmed |
| [F021](FINDINGS.md#f021--p2--manual-payment-records-replace-the-actual-payment-date-with-a-ui-timestamp) | P2 | Manual payment records replace the actual payment date with a UI timestamp |
| [F022](FINDINGS.md#f022--p2--saved-normal-payment-terms-are-ignored-by-both-sale-creation-forms) | P2 | Saved normal payment terms are ignored by both sale creation forms |

P1 means high priority; P2 means a material defect to schedule for correction. Priorities reflect the described conditions, not proof of production occurrence.

The review was tied to checkout `ea7a5a88460857035362604f7dc86cef84bd9501`. Final hash reconciliation found no changes to reviewed files. Only this audit directory was added.

[Review limits and secondary observations](REVIEW-LIMITS.md) distinguish unconfirmed hypotheses, test-source inconsistencies and documentation gaps from the confirmed findings. No runtime, provider, deployment, security or legal certification is claimed.

Machine-readable records: [per-file notes and hashes](reviews.json), [complete physical inventory](inventory.json), and [source declaration navigation index](code-index.json). The approximate index locates 6,294 declarations across 890 code files; it is a navigation aid, not an automated correctness check.
