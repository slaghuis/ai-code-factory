package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"gopkg.in/yaml.v3"

	"github.com/slaghuis/mcp-code/internal/embedder"
	"github.com/slaghuis/mcp-code/internal/search"
	"github.com/slaghuis/mcp-code/internal/tools"
)

type Config struct {
	Qdrant struct {
		Host           string `yaml:"host"`
		Port           int    `yaml:"port"`
		CodeCollection string `yaml:"code_collection"`
		DocsCollection string `yaml:"docs_collection"`
	} `yaml:"qdrant"`

	Ollama struct {
		BaseURL string `yaml:"base_url"`
		Model   string `yaml:"model"`
	} `yaml:"ollama"`

	CacheSize int `yaml:"cache_size"`
}

func main() {
	cfgPath := flag.String("config", "config.yaml", "config path")
	flag.Parse()

	// IMPORTANT: all logging MUST go to stderr. stdout is reserved for MCP JSON-RPC.
	log.SetOutput(os.Stderr)

	b, err := os.ReadFile(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		log.Fatalf("yaml: %v", err)
	}
	if cfg.CacheSize == 0 {
		cfg.CacheSize = 512
	}

	emb := embedder.NewOllama(cfg.Ollama.BaseURL, cfg.Ollama.Model)
	codeStore, err := search.NewStore(cfg.Qdrant.Host, cfg.Qdrant.Port, cfg.Qdrant.CodeCollection)
	if err != nil {
		log.Fatalf("qdrant code: %v", err)
	}

	var docsStore *search.Store
	if cfg.Qdrant.DocsCollection != "" {
		docsStore, err = search.NewStore(cfg.Qdrant.Host, cfg.Qdrant.Port, cfg.Qdrant.DocsCollection)
		if err != nil {
			log.Printf("warn: docs collection unavailable: %v", err)
		}
	}

	deps := tools.Deps{
		Embedder:   emb,
		CodeStore:  codeStore,
		DocsStore:  docsStore,
		Cache:      search.NewEmbedCache(cfg.CacheSize),
		EmbedModel: cfg.Ollama.Model,
	}

	s := server.NewMCPServer(
		"code-search",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithInstructions(systemInstructions),
	)

	tools.RegisterSearchCode(s, deps)
	tools.RegisterSearchDocs(s, deps)
	tools.RegisterSimilarFunction(s, deps)
	tools.RegisterGetFileContext(s, deps)
	tools.RegisterListSymbols(s, deps)

	log.Printf("mcp-code starting on stdio (code=%s docs=%s)",
		cfg.Qdrant.CodeCollection, cfg.Qdrant.DocsCollection)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintln(os.Stderr, "serve:", err)
		os.Exit(1)
	}
}

const systemInstructions = `
This MCP server provides semantic search over an indexed Go codebase and documentation.

GUIDELINES FOR AGENTS:
1. Before reading large files or asking cloud models about the codebase, call 'search_code'
   with a natural-language description. It is MUCH cheaper than reading files blindly.
2. Before proposing architectural changes, call 'search_docs' to check ADRs and design docs.
3. Before writing a new helper function, call 'find_similar_function' to avoid duplication.
4. Use 'get_file_context' after search_code identifies a relevant file.
5. Use 'list_symbols' for structural queries like 'show me all types in package X'.
6. Prefer narrow filters (repo, package, kind) to get higher-quality results.
`