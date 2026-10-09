package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/slaghuis/mcp-code/internal/search"
)

func RegisterSearchDocs(s *server.MCPServer, d Deps) {
	if d.DocsStore == nil {
		return
	}
	tool := mcp.NewTool("search_docs",
		mcp.WithDescription(
			"Semantic search over ADRs, READMEs, runbooks, and design documents. "+
				"Use this before proposing architectural changes to check prior decisions."),
		mcp.WithString("query", mcp.Required(),
			mcp.Description("Natural-language question or topic.")),
		mcp.WithNumber("limit",
			mcp.Description("Max results (default 5, cap 15).")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		limit := uint64(req.GetFloat("limit", 5))
		if limit == 0 || limit > 15 {
			limit = 5
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

		hits, err := d.DocsStore.Search(ctx, vec, limit, search.Filters{})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		payload, _ := json.MarshalIndent(map[string]any{
			"query": query, "count": len(hits), "hits": hits,
		}, "", "  ")
		return mcp.NewToolResultText(string(payload)), nil
	})
}