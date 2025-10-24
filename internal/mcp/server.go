package mcp

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/towada/markdown-vector-mcp/internal/config"
	"github.com/towada/markdown-vector-mcp/internal/embedder"
	"github.com/towada/markdown-vector-mcp/internal/indexer"
	"github.com/towada/markdown-vector-mcp/internal/vectordb"
)

type MCPServer struct {
	server   *server.MCPServer
	indexer  *indexer.Indexer
	db       *vectordb.DB
	embedder embedder.Embedder
	config   *config.Config
}

// NewMCPServer creates a new MCP server
func NewMCPServer(idx *indexer.Indexer, db *vectordb.DB, emb embedder.Embedder, cfg *config.Config) *MCPServer {
	return &MCPServer{
		indexer:  idx,
		db:       db,
		embedder: emb,
		config:   cfg,
	}
}

// Start starts the MCP server
func (s *MCPServer) Start() error {
	fmt.Fprintf(os.Stderr, "[INFO] Starting MCP server...\n")

	// Create MCP server
	s.server = server.NewMCPServer(
		"markdown-vector-mcp",
		"1.0.0",
	)

	// Register tools
	s.registerTools()

	// Start server (stdio)
	if err := server.ServeStdio(s.server); err != nil {
		return fmt.Errorf("MCP server error: %w", err)
	}

	return nil
}

// registerTools registers all MCP tools
func (s *MCPServer) registerTools() {
	s.registerSearchTool()
	s.registerIndexMarkdownTool()
	s.registerListDocumentsTool()
	s.registerDeleteDocumentTool()
	s.registerReindexDocumentTool()

	fmt.Fprintf(os.Stderr, "[INFO] Registered 5 MCP tools\n")
}
