# Observability contract

The API exposes privacy-safe operational telemetry through authenticated
operator endpoints:

- `GET /api/v1/ops/metrics` returns counters and bounded duration summaries.
- `GET /api/v1/ops/metrics/prometheus` returns aggregate Prometheus text for a
  trusted internal scraper.
- `X-Request-ID` and `X-Trace-ID` are returned on every response. A valid W3C
  `traceparent` trace ID is propagated; otherwise a fresh ID is generated.

In staging and production, API and River workers export spans over OTLP/HTTP
using `OTEL_EXPORTER_OTLP_ENDPOINT`. Development uses a no-op exporter so a
missing local collector cannot block requests. Only bounded operation names,
queue names, status and duration attributes are attached to spans.

Metrics deliberately contain no request paths, user IDs, organization IDs,
provider payloads or restricted values. Metric names are bounded and reject
dynamic identifiers. Request logs include status, duration, request ID and
trace ID, with the repository-wide logging sanitizer applied.

## Pilot objectives

| Signal | Initial target | Page when |
| --- | --- | --- |
| API availability | 99.9% monthly | sustained 5xx or readiness failure |
| normal read p95 | <500 ms | 15-minute burn-rate breach |
| normal write p95 | <900 ms | 15-minute burn-rate breach |
| signed webhook persisted | <5 s p95 | backlog or latency breach |
| financial webhook applied/queued | <60 s p95 | queue stalled or reconciliation mismatch |
| critical notification submitted | <2 min p95 | delivery queue stalled |
| ledger imbalance | zero | page immediately |
| duplicate debit caused by Kredit | zero | page immediately and open incident |

The metrics endpoint is not a substitute for an external collector. Production
must scrape it over a private network, retain time series according to the
approved retention policy, and route alerts to the incident owner.
