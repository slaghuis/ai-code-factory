 # MCP Server for Qdrant-Backed Code Search

A Go-based MCP server that exposes search_code, search_docs, and related tools to opencode, Claude Code, Cursor, and any other MCP-compatible agent. Communicates via stdio (the standard MCP transport) so agents can spawn it as a subprocess.

 ### What Just Shifted
With this server registered in your agents:
 - Agent prompt: "Add rate limiting to the login endpoint"
 - Instead of reading 15 files to orient itself, it calls search_code("rate limiting middleware"), search_code("login endpoint handler"), maybe search_docs("auth design decisions").
 - Returns ~5KB of highly relevant snippets instead of 50KB of raw file contents.
 - Cloud model gets a sharp, pre-focused prompt — faster, cheaper, often better output.

Typical savings in realistic use: 40–60% input token reduction on codebase-aware tasks.

 ## Design Decisions
 | Decision  | Choice | Why |
 | --------- | ------ | --- | 
 | Transport | stdio (JSON-RPC 2.0) | Universal MCP support; no network setup |
 | MCP SDK | github.com/mark3labs/mcp-go | Mature, idiomatic, actively maintained |
 | Tools exposed | search_code, search_docs, find_similar_function, get_file_context, list_symbols | Covers 90% of agent retrieval needs |
 | Query embedding | Same Ollama model as indexer | Dimension & semantics must match |
 | Result shape | Compact JSON with score, symbol, path, snippet | Minimize tokens going back to the agent |
 | Snippet strategy | Signature + doc + truncated body | Agent fetches full body only if needed |
 | Filtering | By repo, package, kind, path prefix | Lets agents scope searches |
 | Caching | In-memory LRU for query embeddings | Repeated queries are free |

 ## Testing Standalone with the MCP Inspector
Before plugging into agents, verify the server works:
```
npx @modelcontextprotocol/inspector ~/.local/bin/mcp-code -config ./config.yaml
```

This opens a web UI where you can:
 - See the tool list.
 - Call search_code with a query like "retry with exponential backoff".
 - Inspect the JSON response the agent would receive.

 ## Nudging Agents to Use It
Even perfect tools get ignored if the agent doesn't know when to use them. Add this to the agent's system prompt (opencode AGENTS.md, Claude Code CLAUDE.md, Cursor rules):

```
## Code Search Tool Usage

Before answering any question about the codebase or proposing code changes:
1. Call `search_code` with a natural-language query describing what you need.
2. If search returns relevant files, call `get_file_context` for full picture.
3. Before writing new utility functions, call `find_similar_function`.
4. Before architectural decisions, call `search_docs`.

Do NOT read large files blindly when you can search semantically.
Do NOT escalate to deep reasoning until you've checked existing solutions.
```
This is the key to actually realizing the cost savings — the tool must be the path of least resistance.

## Build and install
```
cd mcp-server
go build -o ~/.local/bin/mcp-code ./cmd/mcp-code
chmod +x ~/.local/bin/mcp-code

# Quick sanity check (will wait on stdin — press Ctrl+D)
echo '' | ~/.local/bin/mcp-code -config ./config.yaml 2>&1 | head
```

 ## Wiring Into Agents
 ### opencode
Edit `~/.config/opencode/opencode.json`:
```
{
  "mcp": {
    "code-search": {
      "type": "local",
      "command": ["/Users/you/.local/bin/mcp-code", "-config", "/Users/you/mcp-server/config.yaml"],
      "enabled": true
    }
  }
}
```
 ### Claude Code
```
claude mcp add code-search \
  /Users/you/.local/bin/mcp-code -- -config /Users/you/mcp-server/config.yaml
```
Or edit `~/.claude/mcp.json`:
```
{
  "mcpServers": {
    "code-search": {
      "command": "/Users/you/.local/bin/mcp-code",
      "args": ["-config", "/Users/you/mcp-server/config.yaml"]
    }
  }
}
```
 ### Cursor
In `~/.cursor/mcp.json` (or project `.cursor/mcp.json`):
```
{
  "mcpServers": {
    "code-search": {
      "command": "/Users/you/.local/bin/mcp-code",
      "args": ["-config", "/Users/you/mcp-server/config.yaml"]
    }
  }
}
```
Then enable it in Cursor Settings -> MCP

 ## Testing Standalone with the MCP Inspector
Before plugging into agents, verify the server works:
```
npx @modelcontextprotocol/inspector ~/.local/bin/mcp-code -config ./config.yaml
```
This opens a web UI where you can:
 - See the tool list.
 - Call search_code with a query like "retry with exponential backoff".
 - Inspect the JSON response the agent would receive.


 ## Nudging Agents to Use It
Even perfect tools get ignored if the agent doesn't know when to use them. Add this to the agent's system prompt (opencode `AGENTS.md`, Claude Code `CLAUDE.md`, Cursor rules):

```
## Code Search Tool Usage

Before answering any question about the codebase or proposing code changes:
1. Call `search_code` with a natural-language query describing what you need.
2. If search returns relevant files, call `get_file_context` for full picture.
3. Before writing new utility functions, call `find_similar_function`.
4. Before architectural decisions, call `search_docs`.

Do NOT read large files blindly when you can search semantically.
Do NOT escalate to deep reasoning until you've checked existing solutions.
```
This is the key to actually realizing the cost savings — the tool must be the path of least resistance.

 ## Operational Notes
 - **Startup is ~instant** — Qdrant & Ollama connections are lazy.
 - **Each tool call** = 1 Ollama embed (~30ms) + 1 Qdrant query (~5ms) = ~40ms total. With cache hits, ~5ms.
 - **No stdout logging ever** — anything on stdout breaks the JSON-RPC framing. Only log.SetOutput(os.Stderr) is used.
 - **One binary, multiple agents** — each agent spawns its own subprocess, they don't share state (except Qdrant). That's fine.
 - **Debugging** — set MCP_DEBUG=1 to tee stderr to a file: ~/.local/bin/mcp-code ... 2>>~/mcp-code.log.
 - **Hot-reload** — restart agents to pick up server rebuilds. Not a daemon.

