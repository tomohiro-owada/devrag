package indexer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/towada/markdown-vector-mcp/internal/config"
	"github.com/towada/markdown-vector-mcp/internal/embedder"
	"github.com/towada/markdown-vector-mcp/internal/vectordb"
)

type Indexer struct {
	db       *vectordb.DB
	embedder embedder.Embedder
	config   *config.Config
}

// NewIndexer creates a new indexer
func NewIndexer(db *vectordb.DB, emb embedder.Embedder, cfg *config.Config) *Indexer {
	return &Indexer{
		db:       db,
		embedder: emb,
		config:   cfg,
	}
}

// IndexFile indexes a single markdown file
func (idx *Indexer) IndexFile(filePath string) error {
	fmt.Fprintf(os.Stderr, "[INFO] Indexing file: %s\n", filePath)

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Parse markdown
	chunks, err := ParseMarkdown(filePath, idx.config.ChunkSize)
	if err != nil {
		return fmt.Errorf("failed to parse markdown: %w", err)
	}

	if len(chunks) == 0 {
		fmt.Fprintf(os.Stderr, "[WARN] No chunks extracted from %s (file may be empty)\n", filePath)
		return nil
	}

	fmt.Fprintf(os.Stderr, "[INFO] Parsed %d chunks\n", len(chunks))

	// Vectorize chunks
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}

	vectors, err := idx.embedder.EmbedBatch(texts)
	if err != nil {
		return fmt.Errorf("failed to vectorize: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[INFO] Generated %d embeddings\n", len(vectors))

	// Convert chunks to ChunkInterface slice
	chunkInterfaces := make([]vectordb.ChunkInterface, len(chunks))
	for i, chunk := range chunks {
		chunkInterfaces[i] = chunk
	}

	// Store in database
	if err := idx.db.InsertDocument(filePath, info.ModTime(), chunkInterfaces, vectors); err != nil {
		return fmt.Errorf("failed to store in database: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[INFO] Successfully indexed %s (%d chunks)\n", filePath, len(chunks))
	return nil
}

// IndexDirectory indexes all markdown files in a directory
func (idx *Indexer) IndexDirectory(dir string) error {
	fmt.Fprintf(os.Stderr, "[INFO] Indexing directory: %s\n", dir)

	fileCount := 0

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Error accessing %s: %v\n", path, err)
			return nil // Continue walking despite errors
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process markdown files
		if filepath.Ext(path) != ".md" {
			return nil
		}

		// Index the file
		if err := idx.IndexFile(path); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to index %s: %v\n", path, err)
			return nil // Continue with other files
		}

		fileCount++
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[INFO] Indexing complete: %d files processed\n", fileCount)
	return nil
}
