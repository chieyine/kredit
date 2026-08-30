# Monitoring deployment

Deploy the OpenTelemetry Collector on a private network, configure Prometheus
to scrape `/api/v1/ops/metrics/prometheus` with the protected operations
identity, load `prometheus-rules.yaml`, and route `critical` alerts to the
financial-operations on-call path. Alert routing must be exercised in staging;
the repository cannot supply deployment-owned receiver addresses or tokens.
