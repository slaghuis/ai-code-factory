---
id: 0002
title: Separate embedding models for code and prose
status: Accepted
date: 2024-11-15
deciders: [slaghuis]
tags: [retrieval, models]
---

# ADR 0002: Separate embedding models for code and prose

## Context

Semantic search quality depends heavily on the embedding model. Code and
natural-language prose have different distributions, and no single model is
best at both.

We also need to run embeddings locally and continuously, so model size and
inference speed matter.

## Decision

Use **two different embedding models**, one per content type:

- **`nomic-embed-text`** (768-dim) for code chunks.
  Small (274MB), fast on Metal, tuned on technical content.
- **`bge-m3`** (1024-dim) for design docs, ADRs, runbooks, and the semantic
  prompt cache.
  Larger (1.2GB) but noticeably better at concept-level retrieval and
  instruction-style queries.

Do **not** mix dimensions in a single Qdrant collection. Each collection is
pinned to one model and one dimension at creation time.

## Consequences

### Positive

- Each content type gets the embedding quality it deserves.
- Prompt-cache lookups and ADR search benefit from bge-m3's instruction tuning.
- Code search stays fast because 768-dim vectors are smaller and quicker.

### Negative

- Two models loaded in Ollama when both are in active use.
- Switching models later requires re-embedding the full collection.
- Developers must remember which collection uses which dimension.

## Alternatives considered

- **One model (bge-m3) for everything**: simpler, but the code-search index
  would be larger and slightly slower, and nomic-embed-text is specifically
  optimised for technical content.
- **One model (nomic-embed-text) for everything**: faster across the board,
  but prose retrieval quality suffers noticeably on ADR queries.