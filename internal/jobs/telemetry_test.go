package jobs

import (
	"context"
	"testing"

	"kredit/internal/observability"

	"github.com/riverqueue/river/rivertype"
)

func TestTelemetryMiddlewareRecordsJobOutcome(t *testing.T) {
	metrics := observability.NewStore()
	middleware := &TelemetryMiddleware{Tracer: observability.NewNoopTracer(), Metrics: metrics}
	err := middleware.Work(context.Background(), &rivertype.JobRow{Queue: QueueProvider, Kind: KindProviderWebhook}, func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if metrics.Snapshot().Counters["river_jobs_total"] != 1 || metrics.Snapshot().Durations["river_job_duration"].Count != 1 {
		t.Fatalf("job telemetry was not recorded: %#v", metrics.Snapshot())
	}
}
