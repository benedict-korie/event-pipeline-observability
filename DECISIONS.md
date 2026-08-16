# Decisions

Short record of the non-obvious choices in this repo and why.

## Hand-rolled Prometheus metrics instead of client_golang

The metrics surface here is small - three counters, a gauge, one histogram.
That doesn't justify pulling in client_golang and its transitive tree for a
project this size. Rolling a small registry by hand keeps the binary small
and the metrics logic readable in one file (`internal/metrics/metrics.go`),
which matters more here than having the full client library's feature set.

Trade-off: this only works for a single-instance service scraped directly.
No multi-process aggregation, no push gateway support. Fine for this
project; wouldn't reach for this pattern on anything that needs to scale
out horizontally.

## Histogram bucket boundaries

`latencyBuckets = [0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5]` (seconds).

The processor simulates most jobs finishing in 20-80ms, with ~5% hitting a
slow path of 500ms-2s. Buckets are denser in the 10ms-250ms range to get
good p50/p95 resolution where most of the traffic actually lands, then
spread out further for the slow tail. No point having fine-grained buckets
past 2.5s for a service where a 5s job is already a clear outlier.

## Alerting: failure rate threshold

Alert fires when `events_failed_total` rate / `events_received_total` rate
stays above 5% for 5 minutes.

The processor simulates a ~4% baseline failure rate by design, so 5% sits
just above normal noise - won't fire on expected background failures, but
catches a real regression. The 5-minute sustained window filters out a
single bad batch; only a sustained shift triggers it.

Bug hit while building this: Grafana's threshold expression (`type:
threshold`) can't take a raw range query directly - it needs an explicit
`reduce` step in between to collapse the range query down to a single
value first. Skipping that step doesn't error clearly; Grafana reports
`no variable specified to reference for refId C`, which reads like a
config typo rather than a missing pipeline stage. Took a couple of
restarts to trace back to the real cause. Alert pipeline is now:
Prometheus query (A) -> reduce last value (B) -> threshold >0.05 (C).

## SLO and burn-rate alerting

SLO: 95% -> revised down to 90% success rate over a rolling 30-day window.

Started at 95% because it's the common default, but the math did not hold
up against this service's actual behaviour. Baseline failure rate is ~4%
(events_failed_total / events_received_total under normal load), so a 95%
SLO (5% allowed error budget) means baseline alone burns ~80% of the budget
continuously - burn_rate = 0.04 / 0.05 = 0.8. That is not a real target,
it is decoration: the SLO would sit permanently near breach even when
nothing is wrong.

Considered narrowing events_failed_total to exclude non-fault failures
(e.g. bad input correctly rejected), counting only genuine processing
faults against the SLO. Checked the metrics code: events_failed_total is
one undifferentiated counter, and the processor has a single failure path
(ErrProcessingFailed) - no validation path to split it from. Splitting
the metric now would add a label with only one real value behind it, so
skipped it.

Landed on 90% (10% error budget): baseline burn_rate = 0.04/0.10 = 0.4,
real margin above normal noise. All failures currently count against the
SLO because the system does not model more than one failure type yet -
that is a known limitation, not a workaround. Revisit the metric split
and SLO target if a real validation path gets added later.
