---
id: 0014
title: Prometheus plus Loki plus Grafana for observability
status: Accepted
date: 2024-12-02
deciders: [slaghuis]
tags: [observability, infrastructure]
---

# ADR 0014: Prometheus + Loki + Grafana for observability

## Context

The AI factory has nine services, two SQLite ledgers, and a Qdrant instance.
We want:

- A single dashboard showing cost, cache hit rate, local-pass rate, pipeline
  health, and open approvals.
- Alerts for cost overruns, stuck approvals, and pipeline failures.
- Both live metrics (what's happening right now) and historical (what the
  SQLite ledger says we spent in total).
- Logs correlated with metrics.
- A stack we can run locally in Docker with modest resource use.

## Decision

Standard open-source stack, all in Docker Compose:

- **Prometheus** (`:9090`): scrapes `/metrics` from every service on 15s.
- **Loki** (`:3100`): receives logs via Promtail from each service's log file.
- **Grafana** (`:3000`): one dashboard, provisioned via JSON file.

Each service exposes `/metrics` on its existing HTTP port. SQLite-backed
historical data is exposed via a dedicated **`metrics-exporter`** daemon on
`:9101` that reads the ledger SQLite files and the pipeline reports
directory, publishing gauges with absolute totals.

Metric names use two prefixes:

- `cache_proxy_*`, `telegram_mcp_*`, `pipeline_mcp_*`, `design_mcp_*`:
  live per-service counters and histograms (reset on restart).
- `ai_factory_*`: historical gauges published by `metrics-exporter`,
  reflecting absolute truth from SQLite.

Alerting goes back to the user via `telegram-mcp`'s `/pipeline/notify`
endpoint, closing the loop.

## Consequences

### Positive

- One dashboard, four rows (Economics, Quality, Pipeline, Humans).
- Fits in <1GB RAM for the whole observability stack.
- Reproducible via `docker compose up -d`; dashboard checked into git.
- Alerts route back to the Telegram bot you already use for approvals.

### Negative

- Yet another stack of services running on the Mac.
- Metric name conventions must be followed carefully; mismatched names mean
  blank panels.
- Live counters reset on service restart; the `ai_factory_*` gauges exist
  specifically to compensate but add their own complexity.
- Dashboard JSON drift (edited in UI but not saved to file) is a recurring
  trap.

## Alternatives considered

- **OpenTelemetry Collector** with a cloud backend (Honeycomb, Datadog):
  works, but recurring cost and overkill for one laptop.
- **InfluxDB + Chronograf**: equivalent, less community momentum than
  Prom+Grafana.
- **Logs only**: fast to set up but can't answer "what's the local-pass
  rate per task tag over the last week?".
