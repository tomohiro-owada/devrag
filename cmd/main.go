package main

import (
	"fmt"
	"os"

	"github.com/towada/markdown-vector-mcp/internal/config"
	"github.com/towada/markdown-vector-mcp/internal/embedder"
	"github.com/towada/markdown-vector-mcp/internal/indexer"
	"github.com/towada/markdown-vector-mcp/internal/mcp"
	"github.com/towada/markdown-vector-mcp/internal/vectordb"
)

func main() {
	fmt.Fprintf(os.Stderr, "[INFO] markdown-vector-mcp starting...\n")

	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "[INFO] Configuration loaded successfully\n")
	fmt.Fprintf(os.Stderr, "[INFO] Documents directory: %s\n", cfg.DocumentsDir)
	fmt.Fprintf(os.Stderr, "[INFO] Database path: %s\n", cfg.DBPath)
	fmt.Fprintf(os.Stderr, "[INFO] Model: %s (dimensions: %d)\n", cfg.Model.Name, cfg.Model.Dimensions)
	fmt.Fprintf(os.Stderr, "[INFO] Device: %s\n", cfg.Compute.Device)

	// 2. Detect device
	device := embedder.DetectDevice(cfg.Compute.Device, cfg.Compute.FallbackToCPU)
	fmt.Fprintf(os.Stderr, "[INFO] Using device: %s\n", device)

	// 3. Initialize components

	// Ensure documents directory exists
	if err := os.MkdirAll(cfg.DocumentsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] Failed to create documents directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize database
	db, err := vectordb.Init(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize embedder
	// Note: Model file is required for production use
	// For testing purposes, we'll use mock embedder if model is not available
	var emb embedder.Embedder
	modelPath := "models/multilingual-e5-small/model.onnx"
	if _, err := os.Stat(modelPath); err == nil {
		emb, err = embedder.NewONNXEmbedder(modelPath, device)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[FATAL] Failed to initialize embedder: %v\n", err)
			os.Exit(1)
		}
		defer emb.Close()
		fmt.Fprintf(os.Stderr, "[INFO] Loaded ONNX model from %s\n", modelPath)
	} else {
		fmt.Fprintf(os.Stderr, "[WARN] Model not found at %s, using mock embedder\n", modelPath)
		emb = &embedder.MockEmbedder{}
	}

	// Initialize indexer
	idx := indexer.NewIndexer(db, emb, cfg)

	// 4. Sync documents
	fmt.Fprintf(os.Stderr, "[INFO] Syncing documents...\n")
	syncResult, err := idx.Sync()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] Sync error: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "[INFO] Sync complete: +%d, ~%d, -%d\n",
			len(syncResult.Added),
			len(syncResult.Updated),
			len(syncResult.Deleted))
	}

	// 5. Start MCP server
	fmt.Fprintf(os.Stderr, "[INFO] Starting MCP server...\n")
	server := mcp.NewMCPServer(idx, db, emb, cfg)
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] MCP server error: %v\n", err)
		os.Exit(1)
	}
}
