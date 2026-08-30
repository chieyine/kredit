# End-to-end acceptance suite

The browser implementation lives in `web/tests/product-flows.spec.ts` so it
runs with the SvelteKit Playwright configuration. It covers the customer-facing
critical path; provider timeout, duplicate webhook, partial collection,
financial invariants, restart, and PostgreSQL concurrency are covered by the
Go integration/contract suites where deterministic process control is
possible. `scripts/test-e2e.sh` is the single entry point.
