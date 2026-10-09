---

## ADR 0012 — Dagger for pipeline execution

```markdown
---
id: 0012
title: Dagger for pipeline execution
status: Accepted
date: 2024-11-24
deciders: [slaghuis]
tags: [build, infrastructure]
---

# ADR 0012: Dagger for pipeline execution

## Context

Pipeline stages (lint, test, scan, build, integration tests) need to:

- Run identically locally and in CI.
- Cache aggressively between runs.
- Compose cleanly in Go code (we want agents to invoke stages).
- Handle service dependencies (Postgres for integration tests) hermetically.
- Not require Docker Desktop.

## Decision

Use **Dagger** as the pipeline execution engine. Each stage is a Go function
that returns a `report.StageResult`. Stages compose into commands (`lint`,
`test`, `full`, `release`). Dagger provides:

- Content-addressed caching of `go mod download`, build cache, test cache.
- Service bindings for ephemeral Postgres instances during integration tests.
- Multi-platform builds via BuildKit under the hood.
- Portable execution: the same code runs in GitHub Actions with no changes.

Dagger engine runs in OrbStack (lighter than Docker Desktop on Apple
Silicon). Agents invoke pipelines through `pipeline-mcp`, which shells out
to the per-service binary.

## Consequences

### Positive

- Second runs on unchanged code take ~5 seconds thanks to caching.
- Integration tests run against real dependencies without polluting the host.
- Pipelines are typed Go code: agents can parse `StageResult` and reason
  about failures.
- Same binary runs locally and in CI; no "works on my machine" surprises.

### Negative

- Dagger engine itself must be running (OrbStack takes RAM).
- First-run cold cache is slow (~2 min pulling base images).
- Debugging failed stages requires reading Dagger's log output, which is
  verbose.
- Learning curve for Dagger's Go SDK.

## Alternatives considered

- **Taskfile / Make**: no sandboxing, no automatic caching, poor
  observability.
- **Earthly**: similar idea to Dagger but uses an Earthfile DSL instead of
  Go; worse composition from agents.
- **Pure Dockerfiles + shell**: everything we'd want to add would be a
  reinvention of Dagger.