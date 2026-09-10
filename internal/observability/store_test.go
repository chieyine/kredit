package observability

import (
	"strings"
	"testing"
	"time"
)

func TestStoreRecordsBoundedDurationSummaries(t *testing.T) {
	store := NewStore()
	store.Inc("http_requests_total")
	store.ObserveDuration("http.request.duration", 10*time.Millisecond)
	store.ObserveDuration("http.request.duration", 30*time.Millisecond)
	snapshot := store.Snapshot()
	if snapshot.Counters["http_requests_total"] != 1 {
		t.Fatalf("unexpected counter snapshot: %#v", snapshot.Counters)
	}
	duration := snapshot.Durations["http.request.duration"]
	if duration.Count != 2 || duration.MaxMilliseconds != 30 || duration.P95Milliseconds != 10 {
		t.Fatalf("unexpected duration summary: %#v", duration)
	}
	if !strings.Contains(store.Prometheus(), "kredit_http_requests_total 1") {
		t.Fatal("prometheus output did not contain the counter")
	}
}

func TestStoreRejectsUnboundedMetricNames(t *testing.T) {
	store := NewStore()
	store.Inc("request/path/user-123")
	store.ObserveDuration("request/path/user-123", time.Second)
	if len(store.Snapshot().Counters) != 0 || len(store.Snapshot().Durations) != 0 {
		t.Fatal("unbounded metric name was accepted")
	}
}

func TestDurationCountersRemainCumulativeWhenSamplesRotate(t *testing.T) {
	store := NewStore()
	store.ObserveDuration("worker", 10*time.Second)
	for range 4096 {
		store.ObserveDuration("worker", time.Millisecond)
	}
	value := store.Snapshot().Durations["worker"]
	if value.Count != 4097 || value.SumMilliseconds != 14096 || value.MaxMilliseconds != 10000 {
		t.Fatalf("sample rotation changed cumulative totals: %+v", value)
	}
	if value.P95Milliseconds != 1 || len(store.durations["worker"]) > 2048 {
		t.Fatal("recent percentile samples are not bounded")
	}
	if !strings.Contains(store.Prometheus(), "kredit_worker_milliseconds_count 4097") {
		t.Fatal("Prometheus count must remain cumulative")
	}
}
