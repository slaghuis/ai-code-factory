package sources

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"

	"github.com/slaghuis/metrics-exporter/internal/collector"
)

type CacheProxy struct {
	path string
	db   *sql.DB
}

func NewCacheProxy(path string) (*CacheProxy, error) {
	db, err := sql.Open("sqlite3", path+"?mode=ro&_journal=WAL")
	if err != nil {
		return nil, err
	}
	return &CacheProxy{path: path, db: db}, nil
}

func (c *CacheProxy) Name() string { return "cache_proxy" }

func (c *CacheProxy) Collect() error {
	rows, err := c.db.Query(`
		SELECT model, COALESCE(tag,'') AS tag, SUM(cost_usd), COUNT(*)
		FROM calls GROUP BY model, tag`)
	if err != nil {
		return err
	}
	defer rows.Close()
	collector.CostUSDGauge.Reset()
	for rows.Next() {
		var model, tag string
		var cost float64
		var count int64
		if err := rows.Scan(&model, &tag, &cost, &count); err != nil {
			continue
		}
		collector.CostUSDGauge.WithLabelValues(model, tag).Set(cost)
	}

	rows2, err := c.db.Query(`
		SELECT model, cached, escalated, COUNT(*)
		FROM calls GROUP BY model, cached, escalated`)
	if err != nil {
		return err
	}
	defer rows2.Close()
	collector.CallsGauge.Reset()
	for rows2.Next() {
		var model, cached string
		var escalated int
		var count int64
		if err := rows2.Scan(&model, &cached, &escalated, &count); err != nil {
			continue
		}
		esc := "false"
		if escalated == 1 {
			esc = "true"
		}
		collector.CallsGauge.WithLabelValues(model, cached, esc).Set(float64(count))
	}
	return nil
}