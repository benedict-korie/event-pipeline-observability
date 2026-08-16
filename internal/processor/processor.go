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
	"os"
	"strconv"
	"time"
)

var ErrProcessingFailed = errors.New("job failed during processing")

type Job struct {
	ID      string `json:"id"`
	Payload string `json:"payload"`
}

// failureRate is the probability (0.0-1.0) that a job fails. Defaults to
// the baseline ~4% used throughout this project's alerting and SLO
// reasoning. Overridable via FAILURE_RATE_OVERRIDE so burn-rate alerts
// can be triggered on demand for verification, without a permanent
// chaos-engineering setup this project doesn't need. See DECISIONS.md.
var failureRate = 0.04

func init() {
	if v := os.Getenv("FAILURE_RATE_OVERRIDE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 && f <= 1 {
			failureRate = f
		}
	}
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
	if rand.Float64() < failureRate {
		return elapsed, ErrProcessingFailed
	}
	return elapsed, nil
}
