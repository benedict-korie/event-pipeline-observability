// Package metrics implements a small Prometheus-compatible metrics registry.
//
// We're not pulling in client_golang here on purpose - the surface area we
// need (a handful of counters and one histogram) doesn't justify the extra
// dependency and its transitive tree. Rolling our own keeps the binary small
// and the metrics logic easy to reason about in one file. See DECISIONS.md
// for the full reasoning.
package metrics

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// histogram buckets in seconds, tuned for a service where most jobs finish
// in tens to low-hundreds of milliseconds but a slow tail matters.
var latencyBuckets = []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}

type Registry struct {
	mu sync.Mutex

	eventsReceived  int
	eventsSucceeded int
	eventsFailed    int
	inFlight        int

	bucketCounts map[float64]int
	sumSeconds   float64
	obsCount     int
}

func New() *Registry {
	buckets := make(map[float64]int, len(latencyBuckets))
	for _, b := range latencyBuckets {
		buckets[b] = 0
	}
	return &Registry{bucketCounts: buckets}
}

func (r *Registry) EventReceived() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.eventsReceived++
	r.inFlight++
}

func (r *Registry) EventFinished(success bool, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.inFlight--
	if success {
		r.eventsSucceeded++
	} else {
		r.eventsFailed++
	}

	seconds := duration.Seconds()
	r.sumSeconds += seconds
	r.obsCount++
	for _, b := range latencyBuckets {
		if seconds <= b {
			r.bucketCounts[b]++
		}
	}
}

// Render writes the current metric values in Prometheus text exposition
// format. Good enough for a single-instance service scraped directly;
// nothing here handles multi-process aggregation.
func (r *Registry) Render() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var sb strings.Builder

	sb.WriteString("# HELP events_received_total Total events accepted by the processor.\n")
	sb.WriteString("# TYPE events_received_total counter\n")
	fmt.Fprintf(&sb, "events_received_total %d\n", r.eventsReceived)

	sb.WriteString("# HELP events_succeeded_total Events processed successfully.\n")
	sb.WriteString("# TYPE events_succeeded_total counter\n")
	fmt.Fprintf(&sb, "events_succeeded_total %d\n", r.eventsSucceeded)

	sb.WriteString("# HELP events_failed_total Events that errored during processing.\n")
	sb.WriteString("# TYPE events_failed_total counter\n")
	fmt.Fprintf(&sb, "events_failed_total %d\n", r.eventsFailed)

	sb.WriteString("# HELP events_in_flight Events currently being processed.\n")
	sb.WriteString("# TYPE events_in_flight gauge\n")
	fmt.Fprintf(&sb, "events_in_flight %d\n", r.inFlight)

	sb.WriteString("# HELP event_processing_duration_seconds How long each event takes to process.\n")
	sb.WriteString("# TYPE event_processing_duration_seconds histogram\n")
	for _, b := range latencyBuckets {
		fmt.Fprintf(&sb, "event_processing_duration_seconds_bucket{le=\"%g\"} %d\n", b, r.bucketCounts[b])
	}
	fmt.Fprintf(&sb, "event_processing_duration_seconds_bucket{le=\"+Inf\"} %d\n", r.obsCount)
	fmt.Fprintf(&sb, "event_processing_duration_seconds_sum %f\n", r.sumSeconds)
	fmt.Fprintf(&sb, "event_processing_duration_seconds_count %d\n", r.obsCount)

	return sb.String()
}