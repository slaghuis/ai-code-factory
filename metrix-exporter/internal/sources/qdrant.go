package sources

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/slaghuis/metrics-exporter/internal/collector"
)

type Qdrant struct {
	url         string
	collections []string
	http        *http.Client
}

func NewQdrant(url string, collections []string) *Qdrant {
	return &Qdrant{
		url: url, collections: collections,
		http: &http.Client{Timeout: 5 * time.Second},
	}
}

func (q *Qdrant) Name() string { return "qdrant" }

type qdrantResp struct {
	Result struct {
		PointsCount         int64 `json:"points_count"`
		IndexedVectorsCount int64 `json:"indexed_vectors_count"`
	} `json:"result"`
}

func (q *Qdrant) Collect() error {
	for _, col := range q.collections {
		resp, err := q.http.Get(fmt.Sprintf("%s/collections/%s", q.url, col))
		if err != nil {
			return err
		}
		var r qdrantResp
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()
		collector.QdrantPoints.WithLabelValues(col).Set(float64(r.Result.PointsCount))
		if r.Result.PointsCount > 0 {
			collector.QdrantIndexedPct.WithLabelValues(col).Set(
				float64(r.Result.IndexedVectorsCount) / float64(r.Result.PointsCount))
		}
	}
	return nil
}