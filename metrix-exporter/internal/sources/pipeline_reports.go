// Reads the JSON files pipeline-mcp drops in ~/.local/share/ai-factory/reports/.

package sources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/slaghuis/metrics-exporter/internal/collector"
)

type PipelineReports struct {
	dir    string
	seen   map[string]bool
}

func NewPipelineReports(dir string) *PipelineReports {
	return &PipelineReports{dir: dir, seen: map[string]bool{}}
}

func (p *PipelineReports) Name() string { return "pipeline_reports" }

type report struct {
	Pipeline string `json:"pipeline"`
	Service  string `json:"service"`
	Success  bool   `json:"success"`
}

func (p *PipelineReports) Collect() error {
	entries, err := os.ReadDir(p.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if p.seen[e.Name()] {
			continue
		}
		b, err := os.ReadFile(filepath.Join(p.dir, e.Name()))
		if err != nil {
			continue
		}
		var r report
		if err := json.Unmarshal(b, &r); err != nil {
			continue
		}
		status := "success"
		if !r.Success {
			status = "failure"
		}
		collector.PipelineReports.WithLabelValues(r.Service, r.Pipeline, status).Inc()
		p.seen[e.Name()] = true
	}
	// Trim seen set to recent files (prevent unbounded growth)
	cutoff := time.Now().Add(-30 * 24 * time.Hour).Unix()
	for name := range p.seen {
		full := filepath.Join(p.dir, name)
		info, err := os.Stat(full)
		if err != nil || info.ModTime().Unix() < cutoff {
			delete(p.seen, name)
		}
	}
	return nil
}