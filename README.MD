 # AI Software Factory

 ## Architecture Overview
```
┌─────────────────────────────────────────────────────────────┐
│  You (Cursor IDE + Telegram on phone)                       │
└─────────────────────────────────────────────────────────────┘
           │                              │
           ▼                              ▼
┌────────────────────┐         ┌────────────────────────┐
│  Cursor (IDE)      │         │  Telegram Bot          │
│  - Interactive dev │         │  - Approvals           │
│  - Pair coding     │         │  - Notifications       │
└────────────────────┘         │  - Status queries      │
           │                   └────────────────────────┘
           ▼                              │
┌─────────────────────────────────────────▼───────────────────┐
│  OpenCode / Claude Code (autonomous agent runner)           │
│  - Long-running tasks                                       │
│  - Uses cloud models (Claude Sonnet 4.5, GPT-5, etc.)       │
│  - MCP servers for tools                                    │
└──────────────────────────────────────────────────────────────┘
           │                              │
           ▼                              ▼
┌────────────────────┐         ┌────────────────────────┐
│  Local LLM stack   │         │  Dev infra             │
│  (Ollama/LM Studio)│         │  (Docker, k3d, Go)     │
└────────────────────┘         └────────────────────────┘
```
 ## Software to Install
 ### Core IDE & Agents
 - **Cursor** — primary interactive IDE
 - **opencode** (`npm i -g opencode-ai` or Homebrew) — terminal-based autonomous agent, model-agnostic
 - **Claude Code** (optional, complements opencode) — Anthropic's CLI agent, great for long autonomous runs
 - **Aider** (optional) — git-aware pair programmer, cheap for small edits
 ### Local LLM Runtime (for cost optimization)
 - **Ollama** — easiest model runner, great Mac Metal support
 - **LM Studio** — GUI + OpenAI-compatible server, useful for testing
 - **llama.cpp** (optional, direct) — maximum control
 ### Go Toolchain
 - **Go (via `brew install go` or `mise`/`asdf` for version pinning)
 - **golangci-lint, gopls, delve, gotestsum, mockgen**
 - **air** — hot reload for Go services
 ### Containers & Orchestration
 - **OrbStack** (recommended over Docker Desktop on Apple Silicon — faster, lighter)
 - **k3d** or **kind** — local Kubernetes
 - **Tilt** or **Skaffold** — dev loop against k8s
 - **Terraform** / **OpenTofu + Pulumi** (if you prefer Go for IaC)
 ### CI/CD & Deployment Orchestration
 - **GitHub Actions** (cloud) or **Gitea Actions** (self-hosted)
 - **ArgoCD** or **Flux** — GitOps deployment
 - **Dagger** — pipelines-as-Go-code; excellent fit for your stack
 ### Observability (local dev)
 - **Grafana + Loki + Tempo + Prometheus** via docker-compose
 - **OpenTelemetry Collector**
 ### MCP Servers (for agents)
Install MCP servers so opencode/Claude Code can act:
 - `filesystem`, `git`, `github`, `postgres`, `fetch`, `playwright` (browser), `sequential-thinking`
 - Custom MCP server for your **Telegram bot** (see §5)
 ###Design/SDLC
 - **Excalidraw** (local or VSCode ext) — diagrams
 - **PlantUML** / **Mermaid** — AI-friendly diagrams-as-code
 - **ADR tools** (`adr-tools`) — architecture decision records
 ## What to Run Locally vs Cloud
 ###Mac Mini M4 24GB — realistic capacity
You can comfortably run one 7B–14B model at Q4/Q5 alongside your dev work, or a 30B at Q3 if you close other apps. Keep 8–10GB headroom for the OS and Cursor.
 ### Run locally (cost savers)
 | Workload | Model Suggestion |
 | -------- | ---------------- |
 | Code autocomplete (Cursor "tab" / Continue.dev) | Qwen2.5-Coder 7B Q5 |
 | Commit message / PR description generation | Qwen2.5-Coder 7B |
 | Log/trace summarization in CI | Llama 3.1 8B | 
 | Embeddings for RAG over your codebase | nomic-embed-text or bge-m3 |
 | Doc Q&A / README generation | Qwen2.5 14B Q4 | 
 | Lint/test output triage | Qwen2.5-Coder 7B |
 | Semantic code search | local embeddings + Qdrant (Docker) |
 | Transcription (voice notes to specs) | whisper.cpp (large-v3-turbo) |
 | Quick refactors / rename-across-files | Qwen2.5-Coder 14B |
 ### Run in the cloud (quality matters)
 | Workload | Recommended Model |
 | Architecture/design reasoning | Claude Sonnet 4.5 or GPT-5 |
 | Multi-file feature implementation | Claude Sonnet 4.5 (coding leader) |
 | Complex debugging / root cause | Claude Sonnet 4.5 / GPT-5 | 
 | Deployment plan generation | Claude Sonnet 4.5 |
 | Code review of agent output | GPT-5 (cross-check) |
**Routing tip**: use **LiteLLM** (`pip install litellm`) as a local proxy with an OpenAI-compatible endpoint. Define rules so cheap tasks hit Ollama and heavy tasks hit Anthropic/OpenAI. Also enforces budget caps and logs cost per task.
 ## Recommended Autonomous Loop
 1. **You** write a task spec in a markdown file (tasks/TASK-123.md) with acceptance criteria.
 2. **opencode** (or Claude Code) picks it up, creates a feature branch, plans, implements, runs tests.
 3. **Dagger pipeline** runs locally: lint, test, build container, deploy to local k3d, run integration tests.
 4. **Telegram bot** notifies you at checkpoints and asks approval for: merging to main, deploying to staging, destructive operations.
 5. On approval, agent pushes PR; **GitHub Actions** deploys via **ArgoCD**.
 6. Agent summarizes with local LLM and reports via Telegram.
 ## Telegram Integration
 Build a tiny Go **MCP server** exposing tools:
 - `telegram.notify(message)`
 - `telegram.request_approval(context, options[]) → waits for reply`
 - `telegram.ask_question(prompt) → string`
The agent calls these as tools. Approval requests block until you tap a button.
## Cost Optimization Levers
 1. **LiteLLM proxy** with per-model budgets and a daily cap.
 2. **Prompt caching** — Anthropic prompt caching cuts repeated-context costs 70–90%; make sure opencode has it enabled.
 3. **Local embeddings + RAG** — never pay cloud for embedding your repo.
 4. **Local "cheap first" router** — try Qwen2.5-Coder locally; escalate only if confidence/tests fail.
 5. **Batch non-urgent jobs** overnight.
 6. **Cache test runs** via Dagger; agents waste a lot of money re-running CI.
 7. **Context hygiene** — use `.cursorignore` / `.opencodeignore` aggressively; large contexts dominate cost.
 ## Suggested Install Order
```
# 1. Base
brew install go orbstack mise git gh jq ripgrep fd
brew install --cask cursor

# 2. Local LLM
brew install ollama
ollama pull qwen2.5-coder:7b
ollama pull qwen2.5-coder:14b-instruct-q4_K_M
ollama pull nomic-embed-text

# 3. Agents
npm i -g opencode-ai
npm i -g @anthropic-ai/claude-code   # optional

# 4. Proxy & orchestration
pipx install litellm
brew install dagger/tap/dagger
brew install k3d tilt argocd

# 5. Vector store + observability (via Docker)
# docker-compose with qdrant, grafana, loki, tempo, prometheus, otel-collector

# 6. Go tools
go install golang.org/x/tools/gopls@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install gotest.tools/gotestsum@latest
go install github.com/air-verse/air@latest
```
## Realistic Gotchas on 24GB
 - Running **Qwen2.5-Coder 14B + Cursor + Docker + a Go service** at once is tight. Prefer the 7B for always-on autocomplete and switch to 14B on demand.
 - **OrbStack beats Docker Desktop** by a wide margin on RAM.
 - Keep **k3d** nodes to 1–2 with low resource limits for local dev.
 - If you want 30B+ coding models locally, you'll need to upgrade RAM — otherwise just route those to cloud.

## Minimal First Milestone
Before building everything, prove the loop end-to-end:
 1. Cursor + opencode working with Claude Sonnet 4.5 on a trivial Go service.
 2. LiteLLM proxy in front, logging cost. (See below)
 3. Telegram bot that can send a notification from a shell script.
 4. Dagger pipeline that builds + tests one Go service.
 5. Then wire Telegram as an MCP tool and let an agent request approval.

## Why Qdrant Belongs in the Core Architecture
Every cloud call has two cost drivers: input tokens (your context) and output tokens (the reply). Vector search attacks both:
 | Problem | Without Vector DB | With Qdrant |
 | ------- | ----------------- | ----------- | 
 | Agent needs to "understand the codebase" | Stuffs 50k tokens of files into prompt | Retrieves top-10 relevant chunks (~3k tokens) |
 | Repeated similar questions | Pays full cost each time | Semantic cache hit → $0 |
 | "Have we solved this before?" | Agent re-derives from scratch | Returns prior solution from memory |
 | ADR / design doc lookup | Reads whole doc tree | Retrieves 2–3 relevant ADRs |
 | Error triage | Sends full stack trace + logs to cloud | Matches against prior incidents first |
Realistic savings on a mature setup: 40–70% reduction in cloud tokens.
 ## Updated Architecture (with Qdrant as First-Class Citizen)
```
┌────────────────────────────────────────────────────────────┐
│  Agent (opencode / Claude Code)                            │
└────────────────────────────────────────────────────────────┘
            │
            ▼
┌────────────────────────────────────────────────────────────┐
│  LiteLLM Proxy  (routing, budget, logging)                 │
└────────────────────────────────────────────────────────────┘
            │
            ▼
┌────────────────────────────────────────────────────────────┐
│  Retrieval Gateway  (local Go service)                     │
│  1. Semantic cache lookup  →  if hit, return               │
│  2. RAG: query Qdrant for relevant context                 │
│  3. Local LLM attempt (Qwen2.5-Coder) with retrieved ctx   │
│  4. If low confidence → escalate to cloud with lean ctx    │
│  5. Store response + embedding back in Qdrant              │
└────────────────────────────────────────────────────────────┘
            │                 │                   │
            ▼                 ▼                   ▼
      ┌──────────┐      ┌──────────┐      ┌──────────────┐
      │ Qdrant   │      │ Ollama   │      │ Cloud (Claude│
      │ (vectors)│      │ (local)  │      │ /GPT)        │
      └──────────┘      └──────────┘      └──────────────┘
```

 ## What to Store in Qdrant (Collections)
Design multiple collections rather than one giant blob:
 | Collection | Content | Embedding Model | Refresh |
 | ---------- | ------- | --------------- | ------- |
 | code_chunks | Functions, methods, structs from your Go repos | nomic-embed-text or bge-m3 | On git push (hook) |
 | docs | ADRs, READMEs, design docs, runbooks | bge-m3 | On commit |
 | semantic_cache | prompt → response pairs | bge-small (fast) | TTL 7–30 days |
 | incidents | Past bugs, stack traces, resolutions | bge-m3 | Append-only |
 | pr_history | Merged PR diffs + descriptions | bge-m3 | On merge |
 | test_failures | Failing test outputs + fixes | bge-m3 | On CI run |
 | go_stdlib_patterns | Idiomatic snippets you've curated | nomic-embed-text | Manual |
 | tasks | Prior task specs + outcomes | bge-m3 | On task close |
Separate collections = faster search, better filtering, independent TTLs.

## The Semantic Cache Pattern (highest ROI)
This is the single biggest cost saver. Pseudocode:
```
func HandleAgentQuery(prompt string, ctx Context) Response {
    embedding := embed(prompt)                     // local, free
    hit := qdrant.Search("semantic_cache", embedding, 
                         threshold=0.93, limit=1)
    if hit != nil && !hit.Stale() {
        metrics.CacheHit()
        return hit.Response                         // $0, ~50ms
    }
    
    relevantCode := qdrant.Search("code_chunks", embedding, limit=10)
    relevantDocs := qdrant.Search("docs",        embedding, limit=3)
    
    leanPrompt := build(prompt, relevantCode, relevantDocs)
    
    // Try local first
    resp, confidence := ollama.Generate("qwen2.5-coder:14b", leanPrompt)
    if confidence < 0.75 || testsFailed(resp) {
        resp = cloud.Call("claude-sonnet-4.5", leanPrompt)  // escalate
    }
    
    qdrant.Upsert("semantic_cache", embedding, resp)
    return resp
}
```
Key thresholds (tune to taste):
 - **0.95+** similarity → almost certain cache hit
 - **0.88–0.95** → use cached as reference context, not final answer
 - **< 0.88** → treat as new query

 ## Why Qdrant Specifically (vs alternatives)
 | Option | Pros | Cons |
 | ------ | ---- | ---- |
 | Qdrant | Written in Rust (fast on M4), native payload filtering, gRPC, mature, single binary | Needs its own process |
 | Weaviate | Richer features | Heavier, more RAM |
 | Chroma | Simplest | Slower at scale, less production-grade |
 | pgvector | One DB for all | Slower for pure vector workloads |
 | LanceDB | Embedded, no server | Younger ecosystem |
 | Milvus | Scale-out | Overkill for a Mac Mini |
Qdrant is the sweet spot for a single-machine AI factory: ~200MB RAM idle, handles millions of vectors on your hardware, OpenAPI + gRPC, works with every agent framework.

## Installation & setup
```
# Run Qdrant via OrbStack
docker run -d --name qdrant \
  -p 6333:6333 -p 6334:6334 \
  -v ~/qdrant_storage:/qdrant/storage \
  qdrant/qdrant:latest

# Pull embedding models
ollama pull nomic-embed-text         # 274MB, fast, 768-dim
ollama pull bge-m3                   # 1.2GB, multilingual, 1024-dim

# Go client
go get github.com/qdrant/go-client
```

Recommended model split:
 - bge-small-en-v1.5 (via Ollama or fastembed) for the semantic cache — needs to be lightning fast
 - bge-m3 for code and docs — higher quality, worth the ms
 - Don't mix dimensions in one collection

 ## Indexing Pipeline (keep it fresh)
Create a small Go daemon or Dagger task that:
 1. **Watches git** (post-commit hook or fsnotify).
 2. **Chunks Go code semantically** — split by function/method using go/parser, not by lines. Each chunk includes: file path, function name, signature, body, package doc, surrounding comments.
 3. **Embeds** via local Ollama endpoint.
 4. **Upserts to Qdrant** with payload: {repo, path, symbol, kind, lang, mtime}.
 5. **Deletes stale vectors** when files change (use file hash as part of point ID).
For docs: chunk by markdown headings (H2/H3), keep heading path in payload for better retrieval.

 ## MCP Server Exposing Qdrant to Agents
Give agents tools, not raw queries:
```
tools:
  - search_code(query, lang?, repo?)         → chunks
  - search_docs(query, type?)                → docs
  - recall_similar_task(description)         → prior tasks + outcomes
  - find_prior_incident(error_signature)     → resolutions
  - cache_lookup(prompt)                     → cached response or null
  - remember(key, value, collection)         → upsert
```
Then in your opencode / Claude Code config, instruct the agent:
 > "Before answering any non-trivial question, call search_code and search_docs. Before escalating to cloud models, call cache_lookup. Store useful findings via remember."
This is the magic — the agent learns to query local first because its tools make it the path of least resistance.

 ## Updated Local vs Cloud Table
 | Workload | Where | Notes |
 | -------- | ----- | ----- |
 | Embeddings (all) | Local (Ollama) | Never pay cloud for this |
 | Vector search | Local (Qdrant) | Microseconds |
 | Semantic cache lookup | Local | Pre-cloud gate |
 | Code chunk retrieval | Local | Feeds leaner prompts to cloud |
 | Simple edits with good context | Local (Qwen2.5-Coder) | Confidence-gated |
 | Complex reasoning | Cloud | But with retrieved context, not whole repo |
 | Response caching | Local (Qdrant) | Reduces repeat spend |

## Observability — Prove the Savings
Add metrics so you can see what Qdrant saves you:
 - `cache_hits_total` / `cache_misses_total`
 - `cloud_tokens_saved_estimate` (hits × avg_tokens_per_response × price)
 - `local_llm_success_rate` (how often local was good enough)
 - `escalation_rate` (local → cloud)
 - `retrieval_latency_ms`
Grafana dashboard with a "cost avoided today" tile. Very satisfying to watch.

Then build (in order of ROI):
 1. **Indexer** for your main Go repo → `code_chunks` collection.
 2. **MCP server** exposing `search_code` + `search_docs`.
 3. **Semantic cache** wrapper around LiteLLM.
 4. **Incidents / PR history** collections (requires some backfill).
 5. **Confidence-based local → cloud escalation**.

  # Minimum Viable Setup to Start Saving Money This Week
 1. Qdrant in Docker.
 2. A 150-line Go indexer that chunks your repo and upserts.
 3. An MCP server with just search_code and search_docs.
 4. Point opencode at it and tell it in the system prompt to use these tools first.
 5. Even without the semantic cache or local LLM escalation, this alone typically cuts cloud context size by 60%+.


 ## Related Repos
- [GO Indexer for Qdrant](https://github.com/slaghuis/indexer)
- [MCP Server for Qdrant-Backed Code](https://github.com/slaghuis/mcp-server)
- [LiteLLM Middleware with Semantic Cache & Smart Routing](https://github.com/slaghuis/cache-proxy)
- [Telegram MCP Server](https://github.com/slaghuis/telegram-mcp)
- [GO Indexer for Qrant](https://github.com/slaghuis/indexer)
- [Pipeline Lib](https://github.com/slaghuis/pipeline-lib)
- [Pipeline MCP](https://github.com/slaghuis/pipeline-mcp)
- [Design Indexer](https://github.com/slaghuis/design-indexer)
- [Design MCP](https://github.com/slaghuis/design-mcp)




