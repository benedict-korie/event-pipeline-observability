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
