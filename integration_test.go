package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tomohiro-owada/devrag/internal/config"
	"github.com/tomohiro-owada/devrag/internal/embedder"
	"github.com/tomohiro-owada/devrag/internal/indexer"
	"github.com/tomohiro-owada/devrag/internal/vectordb"
)

func TestEndToEnd_FirstRun(t *testing.T) {
	// Setup temporary directories
	tmpDir := t.TempDir()
	testDir := tmpDir + "/test_documents"
	dbPath := tmpDir + "/test_vectors.db"

	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create test markdown file
	testFile := testDir + "/test.md"
	content := "# Test Document\n\nThis is a test document for integration testing.\n\nIt has multiple paragraphs."
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize components
	cfg := config.DefaultConfig()
	cfg.DocumentsDir = testDir
	cfg.DBPath = dbPath
	cfg.ChunkSize = 100

	db, err := vectordb.Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Use mock embedder for testing
	emb := &embedder.MockEmbedder{}
	idx := indexer.NewIndexer(db, emb, cfg)

	// Test indexing
	err = idx.IndexFile(testFile)
	if err != nil {
		t.Errorf("Indexing failed: %v", err)
	}

	// Verify in database
	docs, err := db.ListDocuments()
	if err != nil {
		t.Fatal(err)
	}

	if len(docs) != 1 {
		t.Errorf("Expected 1 document in DB, got %d", len(docs))
	}

	// Check if any document matches (could be stored with full path)
	found := false
	for filename := range docs {
		if filepath.Base(filename) == "test.md" {
			found = true
			break
		}
	}
	if !found {
		t.Error("test.md not found in database")
	}
}

func TestEndToEnd_Reindex(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	testDir := tmpDir + "/test_documents"
	dbPath := tmpDir + "/test_vectors.db"

	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	testFile := testDir + "/test.md"
	originalContent := "# Original\n\nOriginal content."
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize
	cfg := config.DefaultConfig()
	cfg.DocumentsDir = testDir
	cfg.DBPath = dbPath

	db, err := vectordb.Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	emb := &embedder.MockEmbedder{}
	idx := indexer.NewIndexer(db, emb, cfg)

	// First index
	err = idx.IndexFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	// Sleep to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	// Modify file
	updatedContent := "# Updated\n\nThis content has been updated with more information."
	if err := os.WriteFile(testFile, []byte(updatedContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Re-index
	err = idx.IndexFile(testFile)
	if err != nil {
		t.Errorf("Re-indexing failed: %v", err)
	}

	// Verify still only 1 document
	docs, err := db.ListDocuments()
	if err != nil {
		t.Fatal(err)
	}

	if len(docs) != 1 {
		t.Errorf("Expected 1 document after re-index, got %d", len(docs))
	}
}

func TestEndToEnd_MultipleFiles(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	testDir := tmpDir + "/test_documents"
	dbPath := tmpDir + "/test_vectors.db"

	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create multiple test files
	files := map[string]string{
		"doc1.md": "# Document 1\n\nFirst document content.",
		"doc2.md": "# Document 2\n\nSecond document content.",
		"doc3.md": "# Document 3\n\nThird document content.",
	}

	for filename, content := range files {
		filepath := testDir + "/" + filename
		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Initialize
	cfg := config.DefaultConfig()
	cfg.DocumentsDir = testDir
	cfg.DBPath = dbPath

	db, err := vectordb.Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	emb := &embedder.MockEmbedder{}
	idx := indexer.NewIndexer(db, emb, cfg)

	// Index all files
	for filename := range files {
		filepath := testDir + "/" + filename
		err = idx.IndexFile(filepath)
		if err != nil {
			t.Errorf("Failed to index %s: %v", filename, err)
		}
	}

	// Verify all documents are in database
	docs, err := db.ListDocuments()
	if err != nil {
		t.Fatal(err)
	}

	if len(docs) != 3 {
		t.Errorf("Expected 3 documents, got %d", len(docs))
	}

	for filename := range files {
		found := false
		for docPath := range docs {
			if filepath.Base(docPath) == filename {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Document %s not found in database", filename)
		}
	}
}

func TestEndToEnd_DeleteDocument(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	testDir := tmpDir + "/test_documents"
	dbPath := tmpDir + "/test_vectors.db"

	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	testFile := testDir + "/test.md"
	content := "# Test\n\nTest content."
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize and index
	cfg := config.DefaultConfig()
	cfg.DocumentsDir = testDir
	cfg.DBPath = dbPath

	db, err := vectordb.Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	emb := &embedder.MockEmbedder{}
	idx := indexer.NewIndexer(db, emb, cfg)

	err = idx.IndexFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	// Verify indexed
	docs, err := db.ListDocuments()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(docs))
	}

	// Delete document (need to use full path as stored in DB)
	var docPath string
	for path := range docs {
		if filepath.Base(path) == "test.md" {
			docPath = path
			break
		}
	}
	if docPath == "" {
		t.Fatal("Could not find document path for deletion")
	}

	err = db.DeleteDocument(docPath)
	if err != nil {
		t.Errorf("DeleteDocument failed: %v", err)
	}

	// Verify deleted
	docs, err = db.ListDocuments()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents after deletion, got %d", len(docs))
	}
}

func TestEndToEnd_Sync(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	testDir := tmpDir + "/test_documents"
	dbPath := tmpDir + "/test_vectors.db"

	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create initial file
	file1 := testDir + "/doc1.md"
	content1 := "# Doc 1\n\nContent 1."
	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize
	cfg := config.DefaultConfig()
	cfg.DocumentsDir = testDir
	cfg.DBPath = dbPath

	db, err := vectordb.Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	emb := &embedder.MockEmbedder{}
	idx := indexer.NewIndexer(db, emb, cfg)

	// First sync
	_, err = idx.Sync()
	if err != nil {
		t.Fatal(err)
	}

	// Verify
	docs, err := db.ListDocuments()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Errorf("Expected 1 document after first sync, got %d", len(docs))
	}

	// Add another file
	time.Sleep(10 * time.Millisecond)
	file2 := testDir + "/doc2.md"
	content2 := "# Doc 2\n\nContent 2."
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatal(err)
	}

	// Second sync
	_, err = idx.Sync()
	if err != nil {
		t.Fatal(err)
	}

	// Verify both files indexed
	docs, err = db.ListDocuments()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents after second sync, got %d", len(docs))
	}
}

func TestEndToEnd_ConfigValidation(t *testing.T) {
	// Test invalid configuration
	cfg := config.DefaultConfig()
	cfg.ChunkSize = -1

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for negative chunk_size")
	}
}

func TestEndToEnd_EmptyDirectory(t *testing.T) {
	// Setup empty directory
	tmpDir := t.TempDir()
	testDir := tmpDir + "/test_documents"
	dbPath := tmpDir + "/test_vectors.db"

	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize
	cfg := config.DefaultConfig()
	cfg.DocumentsDir = testDir
	cfg.DBPath = dbPath

	db, err := vectordb.Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	emb := &embedder.MockEmbedder{}
	idx := indexer.NewIndexer(db, emb, cfg)

	// Sync empty directory should not error
	_, err = idx.Sync()
	if err != nil {
		t.Errorf("Sync on empty directory failed: %v", err)
	}

	// Verify no documents
	docs, err := db.ListDocuments()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents, got %d", len(docs))
	}
}
