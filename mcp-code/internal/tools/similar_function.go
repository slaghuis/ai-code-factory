package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/slaghuis/mcp-code/internal/search"
)

func RegisterSimilarFunction(s *server.MCPServer, d Deps) {
	tool := mcp.NewTool("find_similar_function",
		mcp.WithDescription(
			"Given a chunk of Go code (signature + body or just description), find "+
				"semantically similar functions already in the codebase. Use this before "+
				"writing a new helper to avoid duplication."),
		mcp.WithString("code", mcp.Required(),
			mcp.Description("Function signature, pseudo-code, or description.")),
		mcp.WithNumber("limit",
			mcp.Description("Max results (default 5).")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		code, err := req.RequireString("code")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		limit := uint64(req.GetFloat("limit", 5))
		if limit == 0 || limit > 15 {
			limit = 5
		}

		vec, err := d.Embedder.Embed(ctx, code)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("embed: %v", err)), nil
		}
		hits, err := d.CodeStore.Search(ctx, vec, limit, search.Filters{
			Lang: "go", Kind: "func",
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{
			"count": len(hits), "hits": hits,
		}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})
}