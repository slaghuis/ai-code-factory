---
id: 0001
title: Use Qdrant as the single vector database
status: Accepted
date: 2024-11-15
deciders: [slaghuis]
tags: [infrastructure, retrieval]
---

# ADR 0001: Use Qdrant as the single vector database

## Context

The AI factory needs vector storage for multiple purposes: code chunks for
semantic code search, design docs for ADR retrieval, cached prompt-response
pairs for the semantic cache, and future expansion (incidents, PR history).
Running this on a Mac Mini M4 with 24GB RAM alongside Ollama, Docker, and
regular dev tools means every piece of infrastructure must be lean.

Candidates considered:

- **Qdrant** — Rust, single binary, native payload filtering, mature client.
- **Weaviate** — richer feature set but heavier memory footprint.
- **Chroma** — easiest to run but weaker at scale and production operations.
- **pgvector** — one DB for everything, but slower for pure vector workloads.
- **LanceDB** — embedded, no server; promising but younger ecosystem.
- **Milvus** — overkill for a single-machine setup.

## Decision

Adopt **Qdrant** as the sole vector database. Run it in a Docker container
with a mounted volume, exposing gRPC on port 6334 and HTTP on 6333.

All services that need vector operations connect to this one instance.
Logical isolation is achieved through **separate collections**, not separate
servers:

- `code_chunks` (768-dim, nomic-embed-text)
- `design_docs` (1024-dim, bge-m3)
- `semantic_cache` (1024-dim, bge-m3)

Future collections (`incidents`, `pr_history`, etc.) use the same instance.

## Consequences

### Positive

- One service to monitor, back up, upgrade.
- Low idle footprint (~200MB RAM).
- gRPC client is fast and type-safe from Go.
- Native payload indexing makes filtered queries (by repo, kind, status) O(1).
- Content-addressed point IDs enable idempotent upserts across all indexers.

### Negative

- Single point of failure for every retrieval feature.
- Backup strategy must cover all collections at once.
- A corrupted collection requires a full re-index of its source (code or docs).

## Alternatives considered

- **pgvector**: appealing as a single-store option, but we already use SQLite
  for ledgers and session state; adding Postgres for vectors introduces another
  heavyweight service.
- **LanceDB** (embedded): would eliminate the Docker dependency but each
  service would ship its own copy of the index, defeating the "shared retrieval"
  model.