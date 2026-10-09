---
id: 0008
title: SSE transport for shared MCP servers; stdio for single-tenant
status: Accepted
date: 2024-11-11
deciders: [slaghuis]
tags: [integration, agents]
---

# ADR 0008: SSE transport for shared MCP servers; stdio for single-tenant

## Context

MCP supports two transports: **stdio** (agent spawns the server as a
subprocess, communicates over stdin/stdout) and **SSE** (agent connects to a
long-running HTTP server over Server-Sent Events).

Stdio is simpler and the historical default. SSE requires a long-lived
process and port.

Our capabilities differ in whether they need shared state:

- `mcp-code`: read-only, stateless. Could use either.
- `design-mcp`: read-only, stateless. Could use either.
- `telegram-mcp`: holds pending-approval sessions in memory. **Must be shared
  across all agents**: if each agent spawned its own copy, approvals would
  go to the wrong bot instance.
- `pipeline-mcp`: holds no cross-call state, but does cache the service
  registry and spawn subprocesses. Shared is simpler.

## Decision

- **Shared MCP servers run SSE on their own port**:
  - `telegram-mcp` on `:8765`
  - `pipeline-mcp` on `:8766`
  - `design-mcp` on `:8767`
  - `mcp-code` on `:8768` (if upgraded to SSE later; currently stdio is fine)
- **Agent-local or single-use MCP servers run stdio**:
  - `mcp-code` currently uses stdio because it's cheap to spawn and has no
    cross-call state.

Each SSE server also mounts a Prometheus `/metrics` endpoint and, where
applicable, additional HTTP endpoints for non-MCP callers (e.g.
`telegram-mcp`'s `/pipeline/approve` for Dagger).

## Consequences

### Positive

- Telegram sessions survive agent restarts because the bot process is
  independent.
- One Grafana scrape target per shared service.
- Dagger pipelines can call `telegram-mcp` HTTP endpoints without speaking
  MCP.

### Negative

- Agents must be configured with URLs, not commands.
- SSE servers run as background daemons (launchd on macOS); more operational
  surface.
- stdio servers can't easily be observed via Prometheus; if we add live
  metrics for mcp-code later we'll need to migrate it to SSE.

## Alternatives considered

- **All stdio**: simpler to deploy but breaks the shared-state requirement
  for `telegram-mcp`.
- **All SSE**: uniform but adds background processes for things that don't
  need them.