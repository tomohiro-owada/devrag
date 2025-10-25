package mcp

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/tomohiro-owada/devrag/internal/config"
	"github.com/tomohiro-owada/devrag/internal/embedder"
	"github.com/tomohiro-owada/devrag/internal/indexer"
	"github.com/tomohiro-owada/devrag/internal/vectordb"
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
		"devrag",
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
	s.registerAddFrontmatterTool()
	s.registerUpdateFrontmatterTool()

	fmt.Fprintf(os.Stderr, "[INFO] Registered 7 MCP tools\n")
}
