# ADRs for the AI Factory
A pragmatic set. Not one ADR per micro-decision — one ADR per consequential choice that a future contributor (including future-you) would otherwise waste hours reverse-engineering.
Fifteen in total, grouped into infrastructure, data, routing, and process. Each follows the format your design-indexer already understands.
 ## The List
 | ID | Title | Status |
 | -- | ----- | ------ | 
 | 0001 | Use Qdrant as the single vector database | Accepted |
 | 0002 | Separate embedding models for code and prose | Accepted | 
 | 0003 | LiteLLM as the sole upstream provider abstraction | Accepted | 
 | 0004 | Two-tier cache: exact (SQLite) in front of semantic (Qdrant) | Accepted |
 | 0005 | Cache keyed per model | Accepted  | 
 | 0006 | Confidence-gated local-first routing with signal-based scoring | Accepted |
 | 0007 | MCP servers as the agent integration contract | Accepted |
 | 0008 | SSE transport for shared MCP servers; stdio for single-tenant | Accepted |
 | 0009 | Telegram as the human-in-the-loop channel | Accepted | 
 | 0010 | Session rehydration via SQLite for approval durability | Accepted | 
 | 0011 | Pipeline-lib + per-service wrapper split | Accepted | 
 | 0012 | Dagger for pipeline execution | Accepted  | 
 | 0013 | Per-service pipeline registry (services.yaml) | Accepted  | 
 | 0014 | Prometheus + Loki + Grafana for observability | Accepted | 
 | 0015 | Local-first retrieval before cloud reasoning | Accepted | 

Each one below is written as it would appear in ~/code/design-docs/adr/. Copy them in as individual files and design-indexer picks them up on the next run.

## How to Add These to Your Factory
```
# Create the central ADR folder if you haven't
mkdir -p ~/code/design-docs/adr

# Write each ADR file:
cat > ~/code/design-docs/adr/0001-qdrant-vector-db.md <<'EOF'
<paste ADR 0001 content>
EOF
# ... and so on for each

# Re-run the design indexer
~/.local/bin/design-indexer -config ~/code/ai-factory/design-indexer/config.yaml
# INFO index complete docs=15 chunks=~90 skipped=0
```
Then verify an agent can find them:
```
npx @modelcontextprotocol/inspector --transport sse http://localhost:8767/sse

# In the inspector:
#   list_adrs(status="Accepted")  → returns all 15
#   find_decision(topic="should I use pgvector?")  → returns ADR 0001
#   find_decision(topic="can an agent run without approvals?")  → ADR 0009
#   find_decision(topic="why two caches?")  → ADR 0004
```
Now the system knows its own design. Any agent proposing a change — "should we switch to pgvector", "let's drop the exact cache", "could we route by prompt length only" — gets intercepted by `find_decision` and presented with the authoritative answer, including the alternatives you already considered.
That's the loop closing: the factory documents itself, the agents consult the documentation, and future-you doesn't have to argue the same decisions twice.

 ## What's Not Here (and Why)
Some decisions I deliberately didn't write ADRs for:
 - **"Use Go for the services"**: not interesting enough; obvious from context.
 - **"Use OrbStack over Docker Desktop"**: operational preference, not architecture.
 - **"Use launchd for daemons"**: platform convention, not a decision worth documenting.
 - **"Use mark3labs/mcp-go"**: library selection, no meaningful alternatives at the time.
 - **"Rate limit at 30 msg/sec for Telegram"**: hard limit from Telegram, not our decision.
Rule of thumb I used: if a reasonable engineer six months from now might ask "why on earth did they do it that way?" and the answer isn't trivially recoverable from the code, it deserves an ADR. Otherwise, let the code speak.

