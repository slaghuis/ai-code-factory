package collector

import (
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/slaghuis/metrics-exporter/internal/sources"
)

type Collector struct {
	log      *slog.Logger
	interval time.Duration
	sources  []sources.Source
}

func New(interval time.Duration, log *slog.Logger, s ...sources.Source) *Collector {
	return &Collector{log: log, interval: interval, sources: s}
}

func (c *Collector) Run(stop <-chan struct{}) {
	t := time.NewTicker(c.interval)
	defer t.Stop()
	c.scrape() // immediate
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			c.scrape()
		}
	}
}

func (c *Collector) scrape() {
	var wg sync.WaitGroup
	for _, s := range c.sources {
		wg.Add(1)
		go func(src sources.Source) {
			defer wg.Done()
			if err := src.Collect(); err != nil {
				c.log.Warn("source failed", "name", src.Name(), "err", err)
			}
		}(s)
	}
	wg.Wait()
}

// Metrics declared once; sources update them.
var (
	CostUSDTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_factory_cost_usd_total",
			Help: "Historical cost from the SQLite ledger.",
		},
		[]string{"model", "tag"},
	)

	CallsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_factory_calls_total",
			Help: "Historical call count by cache outcome.",
		},
		[]string{"model", "cache", "escalated"},
	)

	TelegramPending = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ai_factory_telegram_pending",
		Help: "Pending Telegram sessions (sqlite view).",
	})

	TelegramOrphaned = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ai_factory_telegram_orphaned",
		Help: "Rehydrated-but-not-yet-answered sessions.",
	})

	PipelineReports = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_factory_pipeline_reports_total",
			Help: "Pipeline reports scanned on disk.",
		},
		[]string{"service", "command", "status"},
	)

	QdrantPoints = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ai_factory_qdrant_points",
			Help: "Points per Qdrant collection.",
		},
		[]string{"collection"},
	)

	QdrantIndexedPct = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ai_factory_qdrant_indexed_fraction",
			Help: "Fraction of points indexed vs total.",
		},
		[]string{"collection"},
	)
)

func init() {
	prometheus.MustRegister(
		CostUSDTotal, CallsTotal,
		TelegramPending, TelegramOrphaned,
		PipelineReports, QdrantPoints, QdrantIndexedPct,
	)
}