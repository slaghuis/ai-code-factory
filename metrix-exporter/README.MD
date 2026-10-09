 # SQLite Metrics Exporter
For data that lives in SQLite — the cache-proxy ledger (historical, not just live), the pipeline reports on disk, telegram sessions — a tiny exporter reads periodically and publishes gauges. Why run this alongside the per-service live metrics? Because:
 - Live metrics only count what's happened since the process started. Restart = lose series continuity.
 - Some facts are naturally stored (resolved approvals, historical cost); not emitted as events.
 - Reports persist on disk even after the pipeline-mcp forgets them.

 ## Build & Run
```
cd ~/code/ai-factory/metrics-exporter
go mod tidy
go build -o ~/.local/bin/metrics-exporter ./cmd/metrics-exporter
~/.local/bin/metrics-exporter -config ~/.config/ai-factory/metrics-exporter.yaml
```

 ##  launchd for Metrics Exporter
`~/Library/LaunchAgents/com.slaghuis.metrics-exporter.plist`:

```
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.slaghuis.metrics-exporter</string>
  <key>ProgramArguments</key>
  <array>
    <string>/Users/slaghuis/.local/bin/metrics-exporter</string>
    <string>-config</string>
    <string>/Users/slaghuis/.config/ai-factory/metrics-exporter.yaml</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>/Users/slaghuis/.local/share/ai-factory/logs/metrics-exporter.log</string>
  <key>StandardErrorPath</key><string>/Users/slaghuis/.local/share/ai-factory/logs/metrics-exporter.err</string>
</dict>
</plist>
```
```
launchctl load ~/Library/LaunchAgents/com.slaghuis.metrics-exporter.plist
```

 ## Note
Note: since the per-service exporters already emit live counters, the SQLite-reader metrics use different metric names (ai_factory_*) so Prometheus can store both — one is "what happened in the last N minutes", the other is "the full ledger". Dashboards use whichever is more appropriate per panel.
Because CostUSDTotal and CallsTotal need to be set to absolute totals (not deltas), use a trick: swap prometheus.NewCounterVec for custom gauges that reset each scrape. Simpler version — use gauges:
```
CostUSDGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ai_factory_cost_usd",
		Help: "Total USD spent (full history).",
	},
	[]string{"model", "tag"},
)

CallsGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ai_factory_calls",
		Help: "Total calls (full history).",
	},
	[]string{"model", "cache", "escalated"},
)
```
Each scrape, the source clears + sets the gauges. (Prometheus counter semantics require monotonic increase; absolute truth from SQLite doesn't fit that model well.)
