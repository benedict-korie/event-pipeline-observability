package metrics

import (
	"sync"
	"testing"
	"time"
)

// A duration exactly on a bucket boundary should count in that bucket -
// Render() uses `seconds <= b`, so this is the edge worth checking.
func TestEventFinished_BucketBoundary(t *testing.T) {
	r := New()
	r.EventReceived()
	r.EventFinished(true, 50*time.Millisecond) // exactly the 0.05 bucket

	if r.bucketCounts[0.05] != 1 {
		t.Errorf("expected bucket 0.05 to have count 1, got %d", r.bucketCounts[0.05])
	}
	if r.bucketCounts[0.025] != 0 {
		t.Errorf("expected bucket 0.025 to have count 0, got %d", r.bucketCounts[0.025])
	}
}

// Registry is shared across goroutines in the real service (concurrent
// requests). Run with -race to catch anything the mutex isn't actually
// protecting.
func TestRegistry_ConcurrentWrites(t *testing.T) {
	r := New()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.EventReceived()
			r.EventFinished(true, 10*time.Millisecond)
		}()
	}
	wg.Wait()

	if r.eventsReceived != 100 {
		t.Errorf("expected eventsReceived 100, got %d", r.eventsReceived)
	}
	if r.eventsSucceeded != 100 {
		t.Errorf("expected eventsSucceeded 100, got %d", r.eventsSucceeded)
	}
	if r.inFlight != 0 {
		t.Errorf("expected inFlight 0 after all finished, got %d", r.inFlight)
	}
}
