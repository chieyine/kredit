# Monitoring deployment

Deploy the OpenTelemetry Collector on a private network, configure Prometheus
to scrape `/api/v1/ops/metrics/prometheus` with the protected operations
identity, load `prometheus-rules.yaml`, and route `critical` alerts to the
financial-operations on-call path. Alert routing must be exercised in staging;
the repository cannot supply deployment-owned receiver addresses or tokens.

Migration 156 and the matching API collector add open/unassigned financial-review counts, oldest-open review age, and open/oldest delivery-issue gauges. Apply the migration before the new API. Update alert rules after the API rollout; the original metric function is unchanged and the new API combines it with a separate recovery projection. Revert required-metric alert rules when rolling back the API.

Initial escalation thresholds are fifteen minutes without a financial-review owner, one hour for the oldest financial review, and one day for an open delivery issue. These gauges contain no customer references. Follow `docs/runbooks/recovery-operations.md` for ownership and recovery. Verify receiver routing separately; these files do not configure a paging destination.
