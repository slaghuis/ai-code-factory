package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/slaghuis/mcp-code/internal/embedder"
	"github.com/slaghuis/mcp-code/internal/search"
)

type Deps struct {
	Embedder   *embedder.Ollama
	CodeStore  *search.Store
	DocsStore  *search.Store
	Cache      *search.EmbedCache
	EmbedModel string
}

func RegisterSearchCode(s *server.MCPServer, d Deps) {
	tool := mcp.NewTool("search_code",
		mcp.WithDescription(
			"Semantic search over the indexed Go codebase. "+
				"Use this BEFORE asking cloud models to reason about code, so you can "+
				"pass only relevant snippets instead of whole files. Returns symbol, "+
				"path, signature, godoc, and a truncated body."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Natural-language description of what you're looking for, "+
				"e.g. 'HTTP middleware that validates JWT' or 'function that retries with backoff'."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max results (default 8, cap 25)."),
		),
		mcp.WithString("repo",
			mcp.Description("Restrict to a specific repo name."),
		),
		mcp.WithString("package",
			mcp.Description("Restrict to a Go package name, e.g. 'auth'."),
		),
		mcp.WithString("kind",
			mcp.Description("Restrict by symbol kind: 'func', 'method', or 'type'."),
		),
		mcp.WithString("path_prefix",
			mcp.Description("Restrict to paths matching a substring, e.g. 'internal/auth'."),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		limit := uint64(req.GetFloat("limit", 8))
		if limit == 0 || limit > 25 {
			limit = 8
		}

		vec, cached := d.Cache.Get(d.EmbedModel, query)
		if !cached {
			v, err := d.Embedder.Embed(ctx, query)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("embed: %v", err)), nil
			}
			d.Cache.Put(d.EmbedModel, query, v)
			vec = v
		}

		hits, err := d.CodeStore.Search(ctx, vec, limit, search.Filters{
			Repo:       req.GetString("repo", ""),
			Package:    req.GetString("package", ""),
			Kind:       req.GetString("kind", ""),
			PathPrefix: req.GetString("path_prefix", ""),
			Lang:       "go",
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		payload, _ := json.MarshalIndent(map[string]any{
			"query":  query,
			"count":  len(hits),
			"cached": cached,
			"hits":   hits,
		}, "", "  ")
		return mcp.NewToolResultText(string(payload)), nil
	})
}