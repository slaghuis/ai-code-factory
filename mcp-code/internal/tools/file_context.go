package tools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func RegisterGetFileContext(s *server.MCPServer, d Deps) {
	tool := mcp.NewTool("get_file_context",
		mcp.WithDescription(
			"Return all indexed symbols for a given file, in order. Use this "+
				"after search_code identifies a relevant file but you need the full picture."),
		mcp.WithString("repo", mcp.Required(),
			mcp.Description("Repository name (as configured in the indexer).")),
		mcp.WithString("path", mcp.Required(),
			mcp.Description("Relative file path, e.g. 'internal/auth/jwt.go'.")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		repo, err := req.RequireString("repo")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		path, err := req.RequireString("path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		hits, err := d.CodeStore.ScrollByPath(ctx, repo, path)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{
			"repo": repo, "path": path, "symbols": hits,
		}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})
}