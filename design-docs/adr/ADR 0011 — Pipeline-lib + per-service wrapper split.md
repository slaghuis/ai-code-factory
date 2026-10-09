---
id: 0011
title: Pipeline-lib plus per-service wrapper split
status: Accepted
date: 2024-11-24
deciders: [slaghuis]
tags: [architecture, build]
---

# ADR 0011: Pipeline-lib plus per-service wrapper split

## Context

The pipeline (lint, test, scan, build, deploy) is largely identical across
services, but each service has its own image name, main package, deploy
target, and occasionally a custom stage (proto codegen, migrations).

Options:

- **One monolithic pipeline binary** with hard-coded stages, service-selected
  via flags.
- **Fully bespoke pipeline per service**: duplicate code everywhere.
- **Shared library + thin per-service wrapper.**

## Decision

Split into two parts:

- **`pipeline-lib`** at `~/code/ai-factory/pipeline-lib/`: a Go library
  (no binary) containing all reusable stage code, config schema, Dagger
  orchestration, Telegram client, and the CLI entry point (`cli.Run`).
- **Per-service wrapper** at `<service>/pipeline/main.go`: a ~10-line Go
  program that imports `pipeline-lib` and calls `cli.Run(os.Args[1:])`.
  Each service has its own `pipeline.yaml` for configuration.

Custom per-service behaviour is supported through hook registration:

```go
cli.RegisterPreHook("build", func(ctx, dag, cfg, src) error { ... })