package indexer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tomohiro-owada/devrag/internal/config"
	"github.com/tomohiro-owada/devrag/internal/embedder"
	"github.com/tomohiro-owada/devrag/internal/vectordb"
)

type Indexer struct {
	db                *vectordb.DB
	embedder          embedder.Embedder
	config            *config.Config
	codeParser        *CodeParser
	relationExtractor *RelationExtractor
}

// NewIndexer creates a new indexer
func NewIndexer(db *vectordb.DB, emb embedder.Embedder, cfg *config.Config) *Indexer {
	return &Indexer{
		db:       db,
		embedder: emb,
		config:   cfg,
	}
}

// InitCodeParser initializes the code parser for source code indexing
func (idx *Indexer) InitCodeParser() error {
	if idx.codeParser != nil {
		return nil // Already initialized
	}
	parser, err := NewCodeParser()
	if err != nil {
		return fmt.Errorf("failed to create code parser: %w", err)
	}
	idx.codeParser = parser

	// Initialize relation extractor
	re, err := NewRelationExtractor()
	if err != nil {
		return fmt.Errorf("failed to create relation extractor: %w", err)
	}
	idx.relationExtractor = re

	return nil
}

// CloseCodeParser releases resources used by the code parser
func (idx *Indexer) CloseCodeParser() {
	if idx.codeParser != nil {
		idx.codeParser.Close()
		idx.codeParser = nil
	}
	if idx.relationExtractor != nil {
		idx.relationExtractor.Close()
		idx.relationExtractor = nil
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

// IndexCodeFile indexes a single source code file
func (idx *Indexer) IndexCodeFile(filePath string) error {
	fmt.Fprintf(os.Stderr, "[INFO] Indexing code file: %s\n", filePath)

	// Ensure code parser is initialized
	if err := idx.InitCodeParser(); err != nil {
		return err
	}

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Parse source code
	chunks, err := idx.codeParser.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse code: %w", err)
	}

	if len(chunks) == 0 {
		fmt.Fprintf(os.Stderr, "[WARN] No symbols extracted from %s\n", filePath)
		return nil
	}

	fmt.Fprintf(os.Stderr, "[INFO] Extracted %d symbols\n", len(chunks))

	// Generate embeddings using formatted text
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.GetEmbeddingText()
	}

	vectors, err := idx.embedder.EmbedBatch(texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[INFO] Generated %d embeddings\n", len(vectors))

	// Convert to CodeChunkInterface slice
	codeChunks := make([]vectordb.CodeChunkInterface, len(chunks))
	for i := range chunks {
		codeChunks[i] = chunks[i]
	}

	// Store in database
	if err := idx.db.InsertCodeDocument(filePath, info.ModTime(), codeChunks, vectors); err != nil {
		return fmt.Errorf("failed to store in database: %w", err)
	}

	// Extract and store relations from the full file content
	if idx.relationExtractor != nil {
		content, err := os.ReadFile(filePath)
		if err == nil {
			lang := chunks[0].Language
			// Extract relations from the full file content to capture imports at file level
			relations, err := idx.relationExtractor.ExtractRelations(
				content,
				lang,
				filePath,
				"", // No specific symbol - extract all relations from file
				0,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[WARN] Failed to extract relations for %s: %v\n", filePath, err)
			} else if len(relations) > 0 {
				// Convert to vectordb.CodeRelation
				dbRelations := make([]vectordb.CodeRelation, len(relations))
				for i, r := range relations {
					dbRelations[i] = vectordb.CodeRelation{
						RelationType: string(r.RelationType),
						TargetName:   r.TargetName,
						TargetFile:   r.TargetFile,
						Confidence:   1.0,
					}
				}

				// Associate relations with the first symbol in the file
				// (typically the main class or primary function)
				chunkID, err := idx.db.GetChunkIDBySymbol(filePath, chunks[0].SymbolName)
				if err == nil && chunkID > 0 {
					for i := range dbRelations {
						dbRelations[i].SourceChunkID = chunkID
					}
					if err := idx.db.InsertRelations(dbRelations); err != nil {
						fmt.Fprintf(os.Stderr, "[WARN] Failed to insert relations: %v\n", err)
					} else {
						fmt.Fprintf(os.Stderr, "[INFO] Extracted %d relations from %s\n", len(relations), filePath)
					}
				}
			}
		}
	}

	fmt.Fprintf(os.Stderr, "[INFO] Successfully indexed %s (%d symbols)\n", filePath, len(chunks))
	return nil
}

// CodeSyncResult holds the result of a code directory sync operation
type CodeSyncResult struct {
	Indexed int
	Skipped int
	Failed  int
	Added   int
	Updated int
}

// IndexCodeDirectory indexes all supported code files in a directory with differential sync
func (idx *Indexer) IndexCodeDirectory(dir string, force bool) (*CodeSyncResult, error) {
	fmt.Fprintf(os.Stderr, "[INFO] Indexing code directory: %s (force=%v)\n", dir, force)

	// Ensure code parser is initialized
	if err := idx.InitCodeParser(); err != nil {
		return nil, err
	}

	// Get existing documents from DB
	existingDocs, err := idx.db.ListDocuments()
	if err != nil {
		return nil, fmt.Errorf("failed to list existing documents: %w", err)
	}

	result := &CodeSyncResult{}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Error accessing %s: %v\n", path, err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Check if it's a supported code file
		ext := filepath.Ext(path)
		if !IsSupportedExtension(ext) {
			return nil
		}

		// Check if file needs indexing
		modTime, exists := existingDocs[path]
		if exists && !force {
			// Check if file has been modified
			if info.ModTime().Truncate(1e9).Equal(modTime.Truncate(1e9)) {
				result.Skipped++
				return nil
			}
			result.Updated++
		} else if !exists {
			result.Added++
		}

		// Index the file
		if err := idx.IndexCodeFile(path); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to index %s: %v\n", path, err)
			result.Failed++
			return nil
		}

		result.Indexed++
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[INFO] Code indexing complete: %d indexed, %d skipped, %d failed\n",
		result.Indexed, result.Skipped, result.Failed)
	return result, nil
}
