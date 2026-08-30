package observability

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Snapshot struct {
	GeneratedAt time.Time           `json:"generated_at"`
	Counters    map[string]uint64   `json:"counters"`
	Durations   map[string]Duration `json:"durations"`
}

type Duration struct {
	Count           uint64  `json:"count"`
	SumMilliseconds float64 `json:"sum_milliseconds"`
	MaxMilliseconds float64 `json:"max_milliseconds"`
	P95Milliseconds float64 `json:"p95_milliseconds"`
}

type Store struct {
	mu        sync.RWMutex
	counters  map[string]uint64
	durations map[string][]float64
	now       func() time.Time
}

func NewStore() *Store {
	return &Store{counters: map[string]uint64{}, durations: map[string][]float64{}, now: func() time.Time { return time.Now().UTC() }}
}
func (s *Store) Inc(name string) {
	name = metricName(name)
	if name == "" {
		return
	}
	s.mu.Lock()
	s.counters[name]++
	s.mu.Unlock()
}

// ObserveDuration stores a bounded sample for a named operation. Names are
// caller-controlled only through internal code and are normalised to prevent
// unbounded metric cardinality.
func (s *Store) ObserveDuration(name string, duration time.Duration) {
	name = metricName(name)
	if name == "" {
		return
	}
	milliseconds := float64(duration) / float64(time.Millisecond)
	if milliseconds < 0 {
		milliseconds = 0
	}
	s.mu.Lock()
	samples := s.durations[name]
	if len(samples) >= 2048 {
		copy(samples[:1024], samples[len(samples)-1024:])
		samples = samples[:1024]
	}
	s.durations[name] = append(samples, milliseconds)
	s.mu.Unlock()
}

func metricName(name string) string {
	name = strings.TrimSpace(name)
	if len(name) > 80 {
		return ""
	}
	for index, character := range name {
		if index == 0 && ((character < 'a' || character > 'z') && (character < 'A' || character > 'Z')) {
			return ""
		}
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '_' && character != '.' {
			return ""
		}
	}
	return name
}

func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	values := make(map[string]uint64, len(s.counters))
	for key, value := range s.counters {
		values[key] = value
	}
	durations := make(map[string]Duration, len(s.durations))
	for key, samples := range s.durations {
		if len(samples) == 0 {
			continue
		}
		ordered := append([]float64(nil), samples...)
		sort.Float64s(ordered)
		var sum float64
		for _, sample := range ordered {
			sum += sample
		}
		index := int(float64(len(ordered)-1) * 0.95)
		durations[key] = Duration{Count: uint64(len(ordered)), SumMilliseconds: sum, MaxMilliseconds: ordered[len(ordered)-1], P95Milliseconds: ordered[index]}
	}
	return Snapshot{GeneratedAt: s.now(), Counters: values, Durations: durations}
}

// Prometheus returns a stable, privacy-safe text exposition for internal
// scraping. It intentionally exports aggregate names only, never request
// paths, users, organizations or identifiers.
func (s *Store) Prometheus() string {
	snapshot := s.Snapshot()
	var builder strings.Builder
	keys := make([]string, 0, len(snapshot.Counters))
	for key := range snapshot.Counters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		name := "kredit_" + strings.ReplaceAll(key, ".", "_")
		builder.WriteString("# TYPE " + name + " counter\n")
		builder.WriteString(fmt.Sprintf("%s %d\n", name, snapshot.Counters[key]))
	}
	durationKeys := make([]string, 0, len(snapshot.Durations))
	for key := range snapshot.Durations {
		durationKeys = append(durationKeys, key)
	}
	sort.Strings(durationKeys)
	for _, key := range durationKeys {
		name := "kredit_" + strings.ReplaceAll(key, ".", "_")
		value := snapshot.Durations[key]
		builder.WriteString("# TYPE " + name + "_milliseconds summary\n")
		builder.WriteString(fmt.Sprintf("%s_milliseconds_count %d\n", name, value.Count))
		builder.WriteString(fmt.Sprintf("%s_milliseconds_sum %g\n", name, value.SumMilliseconds))
		builder.WriteString(fmt.Sprintf("%s_milliseconds{quantile=\"0.95\"} %g\n", name, value.P95Milliseconds))
	}
	return builder.String()
}
