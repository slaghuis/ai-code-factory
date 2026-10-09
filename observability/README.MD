# Docker Compose for Grafana + Prometheus + Loki


 ## Launch Everything
 ### Start observability stack
```
cd ~/code/ai-factory/observability
docker compose up -d
# → Grafana at http://localhost:3000 (admin/admin)
```
 ### Confirm scrape targets
Prometheus → http://localhost:9090/targets — all five jobs should be UP.
If any show DOWN:
 - Service not running yet.
 - Service didn't register /metrics (check that mux.Handle("/metrics", ...) is present).
 - macOS firewall blocking Docker-to-host traffic. Fix with host.docker.internal:host-gateway extra_hosts entry (already in compose).
 ### Open the dashboard
http://localhost:3000 → "AI Factory" folder → "AI Factory" dashboard.
If panels are empty:
 - Need traffic to produce metrics. Make a cache-proxy call, run a pipeline, etc.
 - `metrics-exporter` scrapes every 30s; wait one cycle before expecting SQLite-derived panels.


 ## Alerting (Optional, Lightweight)
Grafana's built-in alert rules work for the dashboard; a few that pay for themselves:

```
Cost today > $20                            → notify
Pending approvals > 5 for > 10 min          → notify
Pipeline pass rate (24h) < 70%              → notify
Local-pass rate (24h) < 10%                 → notify (local model broken)
Qdrant points fell by > 20% from prev hour  → notify (indexer broken)
```

Grafana → Alerting → Contact points → add a webhook to your telegram-mcp:

```
URL: http://host.docker.internal:8765/pipeline/notify
```
With a template message like:
```
🚨 {{ .ReceiverName }}: {{ .Title }}
{{ .Message }}
```
Now the dashboard talks to the same bot that handles your approvals. The loop closes.


 ### The Complete Stack
Final layout. Nine services, one dashboard, three shared data stores.
```
Your phone 📱
                                    │
             ┌──────────────────────┴──────────────────────┐
             │                                             │
             ▼                                             ▼
    Telegram approvals                              Grafana alerts
             │                                             │
             ▼                                             ▼
┌────────────────────────────────────────────────────────────────┐
│  Agents (Cursor, opencode, Claude Code)                        │
│                                                                │
│  Retrieval tools         Decision tools        Action tools    │
│    mcp-code                 design-mcp           pipeline-mcp  │
│    search_code              find_decision        pipeline_run  │
│    search_docs              search_adrs          list_services │
│    get_file_context         get_adr                            │
│                             cite_source                        │
│                                                                │
│                   All chat calls flow through                  │
│                                                                │
│                       cache-proxy :8080                        │
│   static router → cache → escalation → LiteLLM → upstream     │
└────────────────────────────────────────────────────────────────┘
             │                      │                     │
             ▼                      ▼                     ▼
      Local Ollama           Anthropic             OpenAI
    (qwen-coder, bge-m3)   (claude-sonnet-4.5)   (gpt-5)

           ┌─────────────────────────────────────────┐
           │ Shared data                             │
           │  Qdrant   (code_chunks, design_docs,    │
           │            semantic_cache)              │
           │  SQLite   (cache ledger, telegram       │
           │            sessions)                    │
           │  Disk     (pipeline reports)            │
           └─────────────────────────────────────────┘

           ┌─────────────────────────────────────────┐
           │ Observability                           │
           │   metrics-exporter → Prometheus         │
           │   each service /metrics → Prometheus    │
           │   service logs → Promtail → Loki        │
           │   Grafana dashboard: AI Factory         │
           └─────────────────────────────────────────┘

           ┌─────────────────────────────────────────┐
           │ Automation                              │
           │   indexer → code_chunks                 │
           │   design-indexer → design_docs          │
           │   pipeline-lib → Dagger stages          │
           │   per-service wrappers → ./bin/pipeline │
           └─────────────────────────────────────────┘
```

 ## Closing Thoughts on the Build
What you have now:
 - **Three retrieval layers** cutting input tokens 50–70%.
 - **Two caching layers** (exact + semantic) cutting repeat calls to zero.
 - **Confidence-gated local-first routing** cutting another 20–40% on top.
 - **Human-in-the-loop** on your phone for anything irreversible.
 - **Full pipeline automation** that agents can drive end-to-end.
 - **Session rehydration** that keeps approvals intact across restarts.
 - **One dashboard** that quantifies all of the above.
The economic model: expect 70–85% total cloud cost reduction vs naive agent usage, with quality controlled by the escalation signals and design decisions enforced by the ADR MCP.
The autonomy model: a well-prompted agent can run a task from spec → PR → staging deploy with 1–3 approval taps on your phone. Overnight runs are safe because every destructive action gates on you.
The maintenance model: each service is a small Go binary with a focused responsibility. New services plug in via the services registry. New dashboards extend the one JSON file. No heavyweight orchestrator; just well-fitting pieces.
That's the full factory. Build it in the order we built it in, let each piece bake for a day or two before moving to the next, and tune the thresholds against your Grafana numbers. In a month you'll trust it with real work.

 ## Notes
 - Assumes services write logs to `\~/.local/share/ai-factory/logs/<service>.log`.
 - For launchd-managed services, you can set `StandardOutPath` / `StandardErrorPath` in their plists to those paths and promtail picks everything up automatically.