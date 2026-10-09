---
id: 0015
title: Local-first retrieval before cloud reasoning
status: Accepted
date: 2024-11-20
deciders: [slaghuis]
tags: [cost, quality, retrieval]
---

# ADR 0015: Local-first retrieval before cloud reasoning

## Context

Agents can send arbitrary context to cloud models. Without constraints, a
single "fix the login bug" request can balloon to 50k tokens of raw file
contents, costing many cents and often producing worse results because the
model is drowning in noise.

Three retrieval MCP servers (`mcp-code`, `design-mcp`, implied future
`incidents-mcp`) exist to provide **small, highly-relevant context** before
the cloud call happens.

## Decision

Establish a mandatory ordering in agent prompts (`AGENTS.md`, `CLAUDE.md`,
Cursor rules):

1. **Retrieve first**. Before reading files or asking the cloud, call:
   - `search_code` for implementation questions.
   - `find_decision` or `search_adrs` for architectural questions.
   - `search_docs` for broad doc questions.
2. **Attribute**. When summarising design decisions, call `cite_source`.
3. **Then reason**. Only after retrieved context is in hand, pose the full
   question to the cloud model (via the cache proxy, which may escalate or
   serve from cache).

The retrieval tools are deliberately cheap and local: embedding happens in
Ollama, search happens in Qdrant, no cloud call is made.

## Consequences

### Positive

- Typical input-token reduction of 50–70% versus naive "paste the files".
- Cloud model responses improve because the context is pre-focused.
- Combines multiplicatively with caching (ADR 0004) and local-first routing
  (ADR 0006). Together: 70–85% total cloud cost reduction in practice.
- Agents can honestly cite their sources via `cite_source`, improving
  trust.

### Negative

- Agents must be nudged (via system prompts) to actually use the tools;
  undisciplined agents will skip retrieval.
- Retrieval quality depends on index freshness; stale `code_chunks` means
  bad retrieval. The indexer runs continuously but downtime matters.
- Agents sometimes retrieve too narrowly and miss context the cloud model
  would have inferred from raw files; tune retrieval `limit` upward
  conservatively.

## Alternatives considered

- **Pass full files always**: simplest, most expensive, and quality suffers
  on large files.
- **Agents decide freely**: data from the first week shows they
  under-retrieve without explicit nudges in the system prompt.