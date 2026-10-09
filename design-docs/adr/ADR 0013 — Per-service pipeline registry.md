---
id: 0013
title: Per-service pipeline registry (services.yaml)
status: Accepted
date: 2024-11-24
deciders: [slaghuis]
tags: [architecture, discoverability]
---

# ADR 0013: Per-service pipeline registry

## Context

The `pipeline-mcp` server exposes tools like `pipeline_run(service, command)`.
Agents need to discover which services exist and where their pipeline
binaries live.

Options:

- Agents pass full absolute paths. Fragile; agent needs to know the layout.
- Hard-code service list in `pipeline-mcp`'s code. Every new service needs
  a rebuild.
- A registry file.

## Decision

A YAML file at `~/.config/ai-factory/services.yaml` lists every known service
with its path, pipeline binary location, default environment, and config:

```yaml
services:
  - name: myservice
    path: ~/code/services/myservice
    pipeline_binary: ./bin/pipeline
    config: ./pipeline.yaml
    default_env: staging