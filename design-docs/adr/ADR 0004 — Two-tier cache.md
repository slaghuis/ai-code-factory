---
id: 0004
title: Two-tier cache - exact SQLite in front of semantic Qdrant
status: Accepted
date: 2024-11-18
deciders: [slaghuis]
tags: [caching, performance]
---

# ADR 0004: Two-tier cache - exact SQLite in front of semantic Qdrant

## Context

A naive agent produces thousands of near-identical prompts per day: repeated
test-fix loops, same clarifying questions, boilerplate generation. Caching
these dramatically reduces cost. Two caching strategies are relevant:

- **Exact match**: hash the normalized request; identical prompts return
  identical responses.
- **Semantic match**: embed the prompt; return responses to prompts within a
  cosine-similarity threshold.

Exact match is cheap (one SHA256 + one SQLite lookup, ~1ms) but misses
rephrasings. Semantic match is powerful but costs an embedding call (~30ms
local) and a Qdrant query (~5ms).

## Decision

Run **both caches, exact first, semantic second**:

1. On a request, compute the exact key from the normalized request. Look up
   in SQLite. If hit, serve immediately.
2. On miss, compute the embedding and query the `semantic_cache` collection
   in Qdrant with a cosine similarity threshold (default 0.95).
3. On semantic hit, **promote** the response to the exact cache so the next
   identical call is instant.
4. On full miss, call upstream, store the response in both caches.

Cache keys include the **model**, so a cached Claude response is never
served to a GPT-5 or Qwen request. See ADR 0005.

Semantic threshold defaults to 0.95. Lower values risk serving wrong-ish
answers; higher values negate the point.

## Consequences

### Positive

- Repeated test-fix loops become free after the first iteration.
- Rephrasings of the same question still hit the cache 15–30% of the time.
- Promotion from semantic to exact makes the second rephrase instant.

### Negative

- Two cache stores to monitor and age out.
- Threshold tuning is a judgement call; too loose and quality degrades.
- Tool-use flows and high-temperature requests bypass caching (see
  `cache.Cacheable`).

## Alternatives considered

- **Semantic cache only**: simpler, but costs an embedding on every call
  including the many that are literally identical.
- **Exact cache only**: cheap but misses the long tail of rephrasings; a
  significant chunk of achievable savings is left on the table.