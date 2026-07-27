// Package processor contains the (simulated) work a job goes through.
//
// In a real service this is where you'd call out to a database, another
// API, or a queue. Here it's stubbed with randomised latency and a small
// failure rate so the metrics and alerting layered on top have something
// realistic to react to.
package processor

import (
	"errors"
	"math/rand"
	"time"
)

var ErrProcessingFailed = errors.New("job failed during processing")

type Job struct {
	ID      string `json:"id"`
	Payload string `json:"payload"`
}

// Process simulates doing work on a job. Most jobs finish quickly; a
// small fraction take much longer (the "slow tail" the histogram buckets
// are shaped around) and a small fraction fail outright.
func Process(job Job) (time.Duration, error) {
	start := time.Now()

	base := time.Duration(20+rand.Intn(60)) * time.Millisecond
	if rand.Intn(20) == 0 { // ~5% of jobs hit a slow path
		base += time.Duration(500+rand.Intn(1500)) * time.Millisecond
	}
	time.Sleep(base)

	elapsed := time.Since(start)

	if rand.Intn(25) == 0 { // ~4% failure rate
		return elapsed, ErrProcessingFailed
	}
	return elapsed, nil
}