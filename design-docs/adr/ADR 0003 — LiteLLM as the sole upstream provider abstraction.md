---
id: 0003
title: LiteLLM as the sole upstream provider abstraction
status: Accepted
date: 2024-11-15
deciders: [slaghuis]
tags: [infrastructure, routing]
---

# ADR 0003: LiteLLM as the sole upstream provider abstraction

## Context

We need to call Anthropic, OpenAI, and local Ollama models from the cache
proxy and the escalator. Each provider has its own SDK, auth model, and
retry quirks. We also want a uniform OpenAI-compatible API so that any
agent can be pointed at our stack.

## Decision

Run **LiteLLM** in proxy mode as the single upstream from the cache proxy.
It exposes an OpenAI-compatible `/v1/chat/completions` endpoint, routes to
Anthropic / OpenAI / Ollama based on model name, handles provider-specific
retries, and enforces rate limits.

Cache proxy does **not** import any provider SDK directly. It only speaks
OpenAI chat completions to LiteLLM.

## Consequences

### Positive

- Adding a new provider (e.g. Gemini) is a one-line config change in
  LiteLLM, no cache-proxy changes needed.
- Agents configured with `OPENAI_BASE_URL=http://localhost:8080/v1` work
  transparently regardless of which provider ultimately serves the call.
- Provider outages are handled by LiteLLM's fallback rules, not by us.

### Negative

- Python runtime dependency on the host (via pipx).
- Extra network hop (cache proxy → LiteLLM → provider).
- Debugging requires looking at two services' logs.

## Alternatives considered

- **Direct provider SDKs in Go**: fewer moving parts, but each SDK is
  substantial and we'd re-implement features LiteLLM gives us for free
  (retries, cost tracking, format normalisation).
- **Writing our own thin router**: possible, but we'd end up re-building
  LiteLLM poorly.