package search

import (
	"context"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
)

type Hit struct {
	Score     float32           `json:"score"`
	Repo      string            `json:"repo"`
	Path      string            `json:"path"`
	Package   string            `json:"package,omitempty"`
	Symbol    string            `json:"symbol"`
	Kind      string            `json:"kind"`
	Signature string            `json:"signature,omitempty"`
	Doc       string            `json:"doc,omitempty"`
	Snippet   string            `json:"snippet,omitempty"`
	StartLine int64             `json:"start_line"`
	EndLine   int64             `json:"end_line"`
}

type Filters struct {
	Repo       string
	Package    string
	Kind       string   // "func" | "method" | "type"
	PathPrefix string
	Lang       string   // "go" by default
}

type Store struct {
	client     *qdrant.Client
	collection string
}

func NewStore(host string, port int, collection string) (*Store, error) {
	c, err := qdrant.NewClient(&qdrant.Config{Host: host, Port: port})
	if err != nil {
		return nil, err
	}
	return &Store{client: c, collection: collection}, nil
}

func (s *Store) Search(ctx context.Context, vec []float32, limit uint64, f Filters) ([]Hit, error) {
	flt := buildFilter(f)

	resp, err := s.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: s.collection,
		Query:          qdrant.NewQuery(vec...),
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
		Filter:         flt,
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant query: %w", err)
	}

	out := make([]Hit, 0, len(resp))
	for _, p := range resp {
		h := Hit{Score: p.Score}
		pl := p.Payload
		h.Repo = str(pl, "repo")
		h.Path = str(pl, "path")
		h.Package = str(pl, "package")
		h.Symbol = str(pl, "symbol")
		h.Kind = str(pl, "kind")
		h.Signature = str(pl, "signature")
		h.Doc = str(pl, "doc")
		body := str(pl, "body")
		h.Snippet = truncate(body, 1200)
		h.StartLine = integer(pl, "start_line")
		h.EndLine = integer(pl, "end_line")
		out = append(out, h)
	}
	return out, nil
}

func buildFilter(f Filters) *qdrant.Filter {
	var must []*qdrant.Condition
	if f.Repo != "" {
		must = append(must, qdrant.NewMatch("repo", f.Repo))
	}
	if f.Package != "" {
		must = append(must, qdrant.NewMatch("package", f.Package))
	}
	if f.Kind != "" {
		must = append(must, qdrant.NewMatch("kind", f.Kind))
	}
	if f.Lang != "" {
		must = append(must, qdrant.NewMatch("lang", f.Lang))
	}
	if f.PathPrefix != "" {
		must = append(must, qdrant.NewMatchText("path", f.PathPrefix))
	}
	if len(must) == 0 {
		return nil
	}
	return &qdrant.Filter{Must: must}
}

// ScrollByPath returns every chunk for one file, ordered by start_line.
func (s *Store) ScrollByPath(ctx context.Context, repo, path string) ([]Hit, error) {
	limit := uint32(256)
	resp, err := s.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: s.collection,
		Filter: &qdrant.Filter{
			Must: []*qdrant.Condition{
				qdrant.NewMatch("repo", repo),
				qdrant.NewMatch("path", path),
			},
		},
		Limit:       &limit,
		WithPayload: qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Hit, 0, len(resp))
	for _, p := range resp {
		pl := p.Payload
		out = append(out, Hit{
			Repo: str(pl, "repo"), Path: str(pl, "path"),
			Package: str(pl, "package"), Symbol: str(pl, "symbol"),
			Kind: str(pl, "kind"), Signature: str(pl, "signature"),
			Doc: str(pl, "doc"), Snippet: truncate(str(pl, "body"), 2000),
			StartLine: integer(pl, "start_line"), EndLine: integer(pl, "end_line"),
		})
	}
	return out, nil
}

// ListSymbols returns compact symbol list (no body) matching filters.
func (s *Store) ListSymbols(ctx context.Context, f Filters, limit uint32) ([]Hit, error) {
	flt := buildFilter(f)
	resp, err := s.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: s.collection,
		Filter:         flt,
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Hit, 0, len(resp))
	for _, p := range resp {
		pl := p.Payload
		out = append(out, Hit{
			Repo: str(pl, "repo"), Path: str(pl, "path"),
			Package: str(pl, "package"), Symbol: str(pl, "symbol"),
			Kind: str(pl, "kind"), Signature: str(pl, "signature"),
			StartLine: integer(pl, "start_line"), EndLine: integer(pl, "end_line"),
		})
	}
	return out, nil
}

func str(m map[string]*qdrant.Value, k string) string {
	if v, ok := m[k]; ok {
		return v.GetStringValue()
	}
	return ""
}

func integer(m map[string]*qdrant.Value, k string) int64 {
	if v, ok := m[k]; ok {
		return v.GetIntegerValue()
	}
	return 0
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n// ... truncated"
}