---
id: 0005
title: Cache keyed per model
status: Accepted
date: 2024-11-18
deciders: [slaghuis]
tags: [caching, correctness]
---

# ADR 0005: Cache keyed per model

## Context

Different models produce different responses for the same prompt. Claude is
verbose and structured; GPT-5 is concise; qwen-coder is terser still. Styles
differ. Tool-use conventions differ. Confidence calibration differs.

If the cache ignored the model, a request routed to Claude could be served a
response originally produced by qwen-coder, and vice versa. This introduces
confusing cross-contamination and defeats quality controls.

## Decision

The cache key for both exact and semantic storage **includes the resolved
target model**. The router runs *before* the cache lookup specifically to
determine the key.

A hit requires: same normalized prompt (exact) or similar embedding
(semantic) **AND** same model.

Cache stores the response under the model that **actually produced it**:

- If escalation kept the response local, the key is the local model.
- If escalation went to cloud, the key is the cloud model.

This way the cache stays honest: a cached response is always one that model
*did* produce for that prompt, not one that a different model produced.

## Consequences

### Positive

- No cross-model contamination.
- Switching routing rules does not corrupt the cache; it just reduces hits
  until the new target model's cache warms up.
- Local-pass responses are cached and served even when the request routes to
  the local model, maximising the compounding benefit of caching + local-first
  routing.

### Negative

- Hit rate per-model is lower than a model-agnostic cache would show.
- A/B testing two models on the same prompts requires two cache warm-ups.

## Alternatives considered

- **Model-agnostic cache**: higher hit rate but incorrect semantics; rejected
  as unsafe.
- **Family-level keying** (e.g. "any Claude variant" → same cache): some gain
  but Claude Sonnet and Claude Opus produce meaningfully different responses;
  not worth the risk.