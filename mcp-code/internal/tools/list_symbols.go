package tools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/slaghuis/mcp-code/internal/search"
)

func RegisterListSymbols(s *server.MCPServer, d Deps) {
	tool := mcp.NewTool("list_symbols",
		mcp.WithDescription(
			"List symbols by filter without a semantic query. Useful for "+
				"'show me all methods on type X' or 'what types live in package Y'."),
		mcp.WithString("repo", mcp.Description("Repository name.")),
		mcp.WithString("package", mcp.Description("Package name.")),
		mcp.WithString("kind", mcp.Description("'func', 'method', or 'type'.")),
		mcp.WithString("path_prefix", mcp.Description("Path substring filter.")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 50, cap 200).")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := uint32(req.GetFloat("limit", 50))
		if limit == 0 || limit > 200 {
			limit = 50
		}
		hits, err := d.CodeStore.ListSymbols(ctx, search.Filters{
			Repo:       req.GetString("repo", ""),
			Package:    req.GetString("package", ""),
			Kind:       req.GetString("kind", ""),
			PathPrefix: req.GetString("path_prefix", ""),
			Lang:       "go",
		}, limit)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{
			"count": len(hits), "symbols": hits,
		}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})
}