---
id: 0007
title: MCP servers as the agent integration contract
status: Accepted
date: 2024-11-10
deciders: [slaghuis]
tags: [integration, agents]
---

# ADR 0007: MCP servers as the agent integration contract

## Context

The factory exposes capabilities to agents: code search, ADR lookup,
pipeline execution, Telegram prompts. We need a protocol that:

- Multiple agents (Cursor, opencode, Claude Code) can speak natively.
- Supports typed tool definitions with argument schemas.
- Handles tool invocation, discovery, and system-level instructions.
- Avoids per-agent custom integrations.

## Decision

All agent-facing capabilities are exposed as **MCP (Model Context Protocol)
servers**. One MCP server per bounded capability:

- `mcp-code` — code search
- `design-mcp` — ADR and design doc search
- `pipeline-mcp` — pipeline orchestration
- `telegram-mcp` — human-in-the-loop

Each server registers its tools via `mark3labs/mcp-go`. Agents discover tools
by connecting to each server's endpoint.

System-level nudges (when to call which tool) are passed via each MCP
server's `WithInstructions` block, so the behaviour is consistent across
agents.

## Consequences

### Positive

- Supported out-of-the-box by all three major agents.
- Adding a new agent to the setup requires only pointing it at the existing
  MCP endpoints.
- Tool definitions are self-describing; agents introspect them.
- Clear boundaries: each MCP server has one job.

### Negative

- MCP is young; the protocol has evolved during the build.
- Debugging MCP flows requires the inspector tool (`mcp-inspector`).
- No native streaming of tool results (not currently a problem for our
  use cases).

## Alternatives considered

- **Custom HTTP APIs per capability**: more flexibility but each agent would
  need per-API wiring; defeats the "any agent works" goal.
- **OpenAI function-calling format directly**: tied to one vendor's
  conventions.