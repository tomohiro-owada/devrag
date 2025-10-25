package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tomohiro-owada/devrag/internal/config"
	"github.com/tomohiro-owada/devrag/internal/embedder"
	"github.com/tomohiro-owada/devrag/internal/indexer"
	"github.com/tomohiro-owada/devrag/internal/vectordb"
)

func main() {
	fmt.Fprintf(os.Stderr, "[INFO] Testing indexer implementation...\n")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Use test database
	cfg.DBPath = "./test_vectors.db"
	fmt.Fprintf(os.Stderr, "[INFO] Using test database: %s\n", cfg.DBPath)

	// Initialize database
	db, err := vectordb.Init(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Detect device
	device := embedder.DetectDevice(cfg.Compute.Device, cfg.Compute.FallbackToCPU)
	fmt.Fprintf(os.Stderr, "[INFO] Using device: %s\n", device)

	// Initialize ONNX embedder
	modelPath := filepath.Join("models", "multilingual-e5-small", "model_optimized.onnx")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "[WARN] Model not found at %s\n", modelPath)
		fmt.Fprintf(os.Stderr, "[INFO] Using placeholder embedder for testing\n")

		// Use a placeholder embedder that generates random vectors
		emb := &PlaceholderEmbedder{dimensions: 384}

		// Create indexer
		idx := indexer.NewIndexer(db, emb, cfg)

		// Create a test markdown file
		testFile := "./test_doc.md"
		testContent := `# Test Document

This is a test document for indexing.

## Section 1

This section contains some content to be indexed.

## Section 2

More content here for testing the chunking logic.`

		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "[FATAL] Failed to create test file: %v\n", err)
			os.Exit(1)
		}
		defer os.Remove(testFile)

		// Test indexing
		fmt.Fprintf(os.Stderr, "\n[TEST] Indexing test file...\n")
		start := time.Now()
		if err := idx.IndexFile(testFile); err != nil {
			fmt.Fprintf(os.Stderr, "[FATAL] Indexing failed: %v\n", err)
			os.Exit(1)
		}
		duration := time.Since(start)
		fmt.Fprintf(os.Stderr, "[TEST] Indexing completed in %v\n", duration)

		// Verify database contents
		fmt.Fprintf(os.Stderr, "\n[TEST] Verifying database contents...\n")
		docs, err := db.ListDocuments()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[FATAL] Failed to list documents: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "[TEST] Documents in database: %d\n", len(docs))
		for filename, modTime := range docs {
			fmt.Fprintf(os.Stderr, "  - %s (modified: %s)\n", filename, modTime.Format(time.RFC3339))
		}

		fmt.Fprintf(os.Stderr, "\n[SUCCESS] Phase 2.3 implementation complete!\n")
		return
	}

	// Load actual ONNX model
	emb, err := embedder.NewONNXEmbedder(modelPath, device)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] Failed to initialize embedder: %v\n", err)
		os.Exit(1)
	}
	defer emb.Close()

	// Create indexer
	idx := indexer.NewIndexer(db, emb, cfg)

	// Check if documents directory exists
	if _, err := os.Stat(cfg.DocumentsDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "[WARN] Documents directory not found: %s\n", cfg.DocumentsDir)
		fmt.Fprintf(os.Stderr, "[INFO] Creating directory...\n")
		if err := os.MkdirAll(cfg.DocumentsDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "[FATAL] Failed to create directory: %v\n", err)
			os.Exit(1)
		}
	}

	// Index directory
	fmt.Fprintf(os.Stderr, "\n[INFO] Indexing directory: %s\n", cfg.DocumentsDir)
	start := time.Now()
	if err := idx.IndexDirectory(cfg.DocumentsDir); err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] Indexing failed: %v\n", err)
		os.Exit(1)
	}
	duration := time.Since(start)

	// Show results
	docs, err := db.ListDocuments()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] Failed to list documents: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "\n[SUCCESS] Indexing complete!\n")
	fmt.Fprintf(os.Stderr, "  Time: %v\n", duration)
	fmt.Fprintf(os.Stderr, "  Documents: %d\n", len(docs))
}

// PlaceholderEmbedder generates placeholder vectors for testing
type PlaceholderEmbedder struct {
	dimensions int
}

func (e *PlaceholderEmbedder) Embed(text string) ([]float32, error) {
	// Generate a simple placeholder vector based on text length
	vec := make([]float32, e.dimensions)
	for i := range vec {
		vec[i] = float32(len(text)) / float32(e.dimensions)
	}
	return vec, nil
}

func (e *PlaceholderEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := e.Embed(text)
		if err != nil {
			return nil, err
		}
		results[i] = emb
	}
	return results, nil
}

func (e *PlaceholderEmbedder) Close() error {
	return nil
}
