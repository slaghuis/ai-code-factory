package main

import (
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v3"

	"github.com/slaghuis/metrics-exporter/internal/collector"
	"github.com/slaghuis/metrics-exporter/internal/sources"
)

type Config struct {
	Listen                 string `yaml:"listen"`
	ScrapeIntervalSeconds  int    `yaml:"scrape_interval_seconds"`
	Sources struct {
		CacheProxy struct {
			SQLitePath string `yaml:"sqlite_path"`
		} `yaml:"cache_proxy"`
		Telegram struct {
			SQLitePath string `yaml:"sqlite_path"`
		} `yaml:"telegram"`
		Pipeline struct {
			ReportsDir string `yaml:"reports_dir"`
		} `yaml:"pipeline"`
		Qdrant struct {
			URL         string   `yaml:"url"`
			Collections []string `yaml:"collections"`
		} `yaml:"qdrant"`
	} `yaml:"sources"`
}

func main() {
	cfgPath := flag.String("config", "config.yaml", "")
	flag.Parse()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	b, err := os.ReadFile(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		log.Fatalf("yaml: %v", err)
	}
	if cfg.Listen == "" {
		cfg.Listen = ":9101"
	}
	if cfg.ScrapeIntervalSeconds == 0 {
		cfg.ScrapeIntervalSeconds = 30
	}

	var srcs []sources.Source
	if p := expand(cfg.Sources.CacheProxy.SQLitePath); p != "" {
		if s, err := sources.NewCacheProxy(p); err == nil {
			srcs = append(srcs, s)
		} else {
			logger.Warn("cache_proxy source", "err", err)
		}
	}
	if p := expand(cfg.Sources.Telegram.SQLitePath); p != "" {
		if s, err := sources.NewTelegram(p); err == nil {
			srcs = append(srcs, s)
		} else {
			logger.Warn("telegram source", "err", err)
		}
	}
	if d := expand(cfg.Sources.Pipeline.ReportsDir); d != "" {
		srcs = append(srcs, sources.NewPipelineReports(d))
	}
	if cfg.Sources.Qdrant.URL != "" && len(cfg.Sources.Qdrant.Collections) > 0 {
		srcs = append(srcs, sources.NewQdrant(cfg.Sources.Qdrant.URL, cfg.Sources.Qdrant.Collections))
	}

	stop := make(chan struct{})
	go collector.New(time.Duration(cfg.ScrapeIntervalSeconds)*time.Second, logger, srcs...).Run(stop)

	// Signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigs; close(stop); os.Exit(0) }()

	http.Handle("/metrics", promhttp.Handler())
	logger.Info("metrics-exporter starting",
		"listen", cfg.Listen, "sources", len(srcs))
	if err := http.ListenAndServe(cfg.Listen, nil); err != nil {
		log.Fatal(err)
	}
}

func expand(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}