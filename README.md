# event-pipeline-observability

A small Go service that simulates event/job processing and exposes hand-rolled Prometheus metrics — built to explore observability and infrastructure trade-offs in a small system, not to demonstrate a production event pipeline.

## Why hand-rolled metrics

`client_golang` wasn't used on purpose. The metrics surface here is small — a few counters, one gauge, one histogram — and rolling it by hand keeps the binary small and the exposition logic easy to reason about in a single file. See `DECISIONS.md` for the full reasoning once that's added.

## What it does

- Accepts a job via `POST /events`
- "Processes" it with randomised latency (most jobs finish in tens to low-hundreds of milliseconds, with a small slow tail) and a small simulated failure rate
- Tracks outcomes and latency in a Prometheus-compatible `/metrics` endpoint
- Exposes `GET /healthz` for liveness checks

## Running locally

    go build -o server .
    ./server

The server listens on `:8080` by default (override with the `PORT` env var).

Send a test event:

    curl -X POST http://localhost:8080/events -d '{"id":"job-1","payload":"test"}'

Check the metrics:

    curl http://localhost:8080/metrics

## Project structure

    main.go                    - HTTP server and routing
    internal/processor/        - simulated job processing logic
    internal/metrics/          - hand-rolled Prometheus metrics registry

## Metrics exposed

| Metric | Type | Description |
|---|---|---|
| `events_received_total` | counter | Total events accepted |
| `events_succeeded_total` | counter | Events processed successfully |
| `events_failed_total` | counter | Events that errored |
| `events_in_flight` | gauge | Events currently processing |
| `event_processing_duration_seconds` | histogram | Time to process an event |
