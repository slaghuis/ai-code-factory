---
id: 0006
title: Confidence-gated local-first routing with signal-based scoring
status: Accepted
date: 2024-11-22
deciders: [slaghuis]
tags: [routing, cost, quality]
---

# ADR 0006: Confidence-gated local-first routing with signal-based scoring

## Context

Local models (qwen-coder on Ollama) can handle a substantial fraction of
Go-dev tasks for free, but they fail silently on complex refactors and
nuanced reasoning. Static routing by prompt size or task tag catches the easy
cases but either over-escalates (wasting money) or under-escalates (shipping
poor answers).

We want: cheap tasks go local, hard tasks go cloud, and the decision is made
*after* seeing the local model's output rather than guessing beforehand.

## Decision

On a cache miss, call the configured local model first. **Score the response
with cheap local signals**:

- Refusal markers (regex)
- Length ratio vs expected tokens
- Structural coherence (balanced braces, closed code fences, no mid-sentence
  truncation)
- Go source compile check via `go/parser` when code blocks are present
- Prompt-answer similarity via embeddings (optional, weighted lower)

Weighted average; if above the per-task-tag threshold, serve the local
response. Below threshold, call the cloud model and serve that.

Thresholds and signal enablement are configured per task tag, with sane
defaults:

- `refactor` / `debug`: strict (threshold 0.80–0.85, Go compile check on)
- `boilerplate` / `explain`: loose (threshold 0.40–0.50, no compile check)

Agents can override routing via the `x-escalation` header:
`auto` (default), `local-only`, `cloud-only`, `escalate-always`.

## Consequences

### Positive

- Realistic additional 20–40% cloud cost reduction on top of caching.
- Quality maintained because bad local responses are caught and re-done
  before reaching the agent.
- Grafana surfaces the local-pass rate per task tag, so thresholds can be
  tuned from data.
- Compile check is the strongest single signal and catches most "confident
  nonsense" from local models.

### Negative

- Latency floor is slightly higher: 2–5s added for the local attempt even
  when it ultimately fails and escalates.
- Scoring heuristics are domain-specific (Go). Expanding to other languages
  needs more signals.
- Streaming requests bypass escalation; they take the legacy direct-forward
  path. Not currently a problem because most agent SDK operations are
  non-streaming.

## Alternatives considered

- **Second local model to score**: the "LLM judge" pattern. Rejected as
  adding too much latency for little gain over cheap signals.
- **Static routing only** (keep ADR 0003's LiteLLM rules as the only
  mechanism): simpler but either over-pays or under-delivers; data from the
  first week of operation confirmed this.
- **Always try cloud first, fall back on cloud failure**: inverts the
  economics; defeats the purpose.