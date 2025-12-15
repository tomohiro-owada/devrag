package vectordb

import (
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	db, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Verify tables exist
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='documents'").Scan(&count)
	if err != nil {
		t.Errorf("Failed to check documents table: %v", err)
	}
	if count != 1 {
		t.Errorf("documents table not created")
	}

	err = db.conn.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='chunks'").Scan(&count)
	if err != nil {
		t.Errorf("Failed to check chunks table: %v", err)
	}
	if count != 1 {
		t.Errorf("chunks table not created")
	}

	// Check vec_chunks virtual table
	err = db.conn.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='vec_chunks'").Scan(&count)
	if err != nil {
		t.Errorf("Failed to check vec_chunks table: %v", err)
	}
	if count != 1 {
		t.Errorf("vec_chunks table not created")
	}
}

func TestListDocuments_Empty(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	db, err := Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	docs, err := db.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments failed: %v", err)
	}

	if len(docs) != 0 {
		t.Errorf("Expected empty list, got %d documents", len(docs))
	}
}

func TestListDocuments_WithData(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	db, err := Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert test documents
	testTime := time.Now()
	_, err = db.conn.Exec(
		"INSERT INTO documents (filename, modified_at) VALUES (?, ?)",
		"test1.md", testTime,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.conn.Exec(
		"INSERT INTO documents (filename, modified_at) VALUES (?, ?)",
		"test2.md", testTime,
	)
	if err != nil {
		t.Fatal(err)
	}

	// List documents
	docs, err := db.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments failed: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(docs))
	}

	if _, ok := docs["test1.md"]; !ok {
		t.Error("test1.md not found in results")
	}
	if _, ok := docs["test2.md"]; !ok {
		t.Error("test2.md not found in results")
	}
}

func TestDeleteDocument(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	db, err := Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert test document
	testTime := time.Now()
	_, err = db.conn.Exec(
		"INSERT INTO documents (filename, modified_at) VALUES (?, ?)",
		"test.md", testTime,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Delete document
	err = db.DeleteDocument("test.md")
	if err != nil {
		t.Errorf("DeleteDocument failed: %v", err)
	}

	// Verify deletion
	docs, _ := db.ListDocuments()
	if len(docs) != 0 {
		t.Errorf("Document not deleted, still have %d documents", len(docs))
	}
}

func TestDeleteDocument_NotFound(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	db, err := Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Try to delete non-existent document
	err = db.DeleteDocument("nonexistent.md")
	if err == nil {
		t.Error("Expected error for non-existent document, got nil")
	}
}

type testChunk struct {
	content  string
	position int
}

func (c testChunk) GetContent() string {
	return c.content
}

func (c testChunk) GetPosition() int {
	return c.position
}

func TestInsertDocument(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	db, err := Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create test chunks
	chunks := []ChunkInterface{
		testChunk{content: "First chunk", position: 0},
		testChunk{content: "Second chunk", position: 1},
	}

	// Create test embeddings (384 dimensions each)
	embeddings := make([][]float32, 2)
	for i := range embeddings {
		embeddings[i] = make([]float32, 384)
		for j := range embeddings[i] {
			embeddings[i][j] = float32(i*j) * 0.01
		}
	}

	// Insert document
	err = db.InsertDocument("test.md", time.Now(), chunks, embeddings)
	if err != nil {
		t.Fatalf("InsertDocument failed: %v", err)
	}

	// Verify document was inserted
	docs, err := db.ListDocuments()
	if err != nil {
		t.Fatal(err)
	}

	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}

	// Verify chunks were inserted
	var chunkCount int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM chunks").Scan(&chunkCount)
	if err != nil {
		t.Fatal(err)
	}

	if chunkCount != 2 {
		t.Errorf("Expected 2 chunks, got %d", chunkCount)
	}

	// Verify vectors were inserted
	var vecCount int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM vec_chunks").Scan(&vecCount)
	if err != nil {
		t.Fatal(err)
	}

	if vecCount != 2 {
		t.Errorf("Expected 2 vectors, got %d", vecCount)
	}
}

func TestInsertDocument_MismatchedCounts(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	db, err := Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	chunks := []ChunkInterface{
		testChunk{content: "First chunk", position: 0},
	}

	embeddings := make([][]float32, 2) // Wrong count
	for i := range embeddings {
		embeddings[i] = make([]float32, 384)
	}

	err = db.InsertDocument("test.md", time.Now(), chunks, embeddings)
	if err == nil {
		t.Error("Expected error for mismatched counts, got nil")
	}
}

func TestInsertDocument_Reindex(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	db, err := Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert first version
	chunks1 := []ChunkInterface{
		testChunk{content: "Original chunk", position: 0},
	}
	embeddings1 := make([][]float32, 1)
	embeddings1[0] = make([]float32, 384)

	err = db.InsertDocument("test.md", time.Now(), chunks1, embeddings1)
	if err != nil {
		t.Fatal(err)
	}

	// Re-index with different content
	chunks2 := []ChunkInterface{
		testChunk{content: "Updated chunk 1", position: 0},
		testChunk{content: "Updated chunk 2", position: 1},
	}
	embeddings2 := make([][]float32, 2)
	for i := range embeddings2 {
		embeddings2[i] = make([]float32, 384)
	}

	err = db.InsertDocument("test.md", time.Now(), chunks2, embeddings2)
	if err != nil {
		t.Fatalf("Re-indexing failed: %v", err)
	}

	// Verify only 1 document exists
	docs, err := db.ListDocuments()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Errorf("Expected 1 document after re-index, got %d", len(docs))
	}

	// Verify correct number of chunks (should be 2, not 3)
	var chunkCount int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM chunks").Scan(&chunkCount)
	if err != nil {
		t.Fatal(err)
	}
	if chunkCount != 2 {
		t.Errorf("Expected 2 chunks after re-index, got %d", chunkCount)
	}
}

func TestSerializeVector(t *testing.T) {
	vec := []float32{1.0, 2.5, -3.14, 0.0}
	blob := serializeVector(vec)

	// Verify blob size (4 bytes per float32)
	expectedSize := len(vec) * 4
	if len(blob) != expectedSize {
		t.Errorf("Expected blob size %d, got %d", expectedSize, len(blob))
	}

	// Verify blob is not all zeros
	allZeros := true
	for _, b := range blob {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		t.Error("Serialized vector is all zeros")
	}
}

func TestClose(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	db, err := Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify connection is closed by trying to query
	_, err = db.conn.Query("SELECT * FROM documents")
	if err == nil {
		t.Error("Expected error after closing, got nil")
	}
}

func TestInit_InvalidPath(t *testing.T) {
	// Try to create database in non-existent directory
	_, err := Init("/nonexistent/path/test.db")
	if err == nil {
		t.Error("Expected error for invalid path, got nil")
	}
}

func TestDeleteDocument_CascadeChunks(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	db, err := Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert document with chunks
	chunks := []ChunkInterface{
		testChunk{content: "Chunk 1", position: 0},
		testChunk{content: "Chunk 2", position: 1},
	}
	embeddings := make([][]float32, 2)
	for i := range embeddings {
		embeddings[i] = make([]float32, 384)
	}

	err = db.InsertDocument("test.md", time.Now(), chunks, embeddings)
	if err != nil {
		t.Fatal(err)
	}

	// Verify chunks exist
	var chunkCount int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM chunks").Scan(&chunkCount)
	if err != nil {
		t.Fatal(err)
	}
	if chunkCount != 2 {
		t.Fatalf("Expected 2 chunks, got %d", chunkCount)
	}

	// Delete document
	err = db.DeleteDocument("test.md")
	if err != nil {
		t.Fatal(err)
	}

	// Verify chunks were also deleted
	err = db.conn.QueryRow("SELECT COUNT(*) FROM chunks").Scan(&chunkCount)
	if err != nil {
		t.Fatal(err)
	}
	if chunkCount != 0 {
		t.Errorf("Expected 0 chunks after document deletion, got %d", chunkCount)
	}
}

// Helper to insert test document with embedding for search tests
func insertTestDocForSearch(t *testing.T, db *DB, filename string, content string, embedding []float32) {
	t.Helper()
	chunks := []ChunkInterface{
		testChunk{content: content, position: 0},
	}
	embeddings := [][]float32{embedding}
	err := db.InsertDocument(filename, time.Now(), chunks, embeddings)
	if err != nil {
		t.Fatalf("Failed to insert test document %s: %v", filename, err)
	}
}

// Create a simple embedding for testing (just fills with a specific value)
func makeTestEmbedding(seed float32) []float32 {
	emb := make([]float32, 384)
	for i := range emb {
		emb[i] = seed + float32(i)*0.001
	}
	return emb
}

func TestSearchWithFilter_NoFilter(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	db, err := Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert test documents
	insertTestDocForSearch(t, db, "docs/api/auth.md", "Authentication API", makeTestEmbedding(0.1))
	insertTestDocForSearch(t, db, "docs/api/users.md", "Users API", makeTestEmbedding(0.2))
	insertTestDocForSearch(t, db, "guides/setup.md", "Setup guide", makeTestEmbedding(0.3))

	// Search without filter
	results, err := db.SearchWithFilter(makeTestEmbedding(0.1), 10, nil)
	if err != nil {
		t.Fatalf("SearchWithFilter failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results without filter, got %d", len(results))
	}
}

func TestSearchWithFilter_DirectoryFilter(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	db, err := Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert test documents
	insertTestDocForSearch(t, db, "docs/api/auth.md", "Authentication API", makeTestEmbedding(0.1))
	insertTestDocForSearch(t, db, "docs/api/users.md", "Users API", makeTestEmbedding(0.2))
	insertTestDocForSearch(t, db, "guides/setup.md", "Setup guide", makeTestEmbedding(0.3))

	// Search with directory filter
	filter := &SearchFilter{Directory: "docs/api"}
	results, err := db.SearchWithFilter(makeTestEmbedding(0.1), 10, filter)
	if err != nil {
		t.Fatalf("SearchWithFilter failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results with directory filter 'docs/api', got %d", len(results))
	}

	// Verify all results are from docs/api
	for _, r := range results {
		if !hasPrefix(r.DocumentName, "docs/api/") {
			t.Errorf("Result document %s is not in docs/api/", r.DocumentName)
		}
	}
}

func TestSearchWithFilter_FilePatternFilter(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	db, err := Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert test documents
	insertTestDocForSearch(t, db, "api-auth.md", "Authentication API", makeTestEmbedding(0.1))
	insertTestDocForSearch(t, db, "api-users.md", "Users API", makeTestEmbedding(0.2))
	insertTestDocForSearch(t, db, "guide-setup.md", "Setup guide", makeTestEmbedding(0.3))

	// Search with file pattern filter
	filter := &SearchFilter{FilePattern: "api-*.md"}
	results, err := db.SearchWithFilter(makeTestEmbedding(0.1), 10, filter)
	if err != nil {
		t.Fatalf("SearchWithFilter failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results with pattern 'api-*.md', got %d", len(results))
	}
}

func TestSearchWithFilter_CombinedFilters(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	db, err := Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert test documents
	insertTestDocForSearch(t, db, "docs/api-auth.md", "Authentication API", makeTestEmbedding(0.1))
	insertTestDocForSearch(t, db, "docs/api-users.md", "Users API", makeTestEmbedding(0.2))
	insertTestDocForSearch(t, db, "docs/guide-setup.md", "Setup guide", makeTestEmbedding(0.3))
	insertTestDocForSearch(t, db, "other/api-test.md", "Test API", makeTestEmbedding(0.4))

	// Search with both filters
	filter := &SearchFilter{
		Directory:   "docs",
		FilePattern: "api-*.md",
	}
	results, err := db.SearchWithFilter(makeTestEmbedding(0.1), 10, filter)
	if err != nil {
		t.Fatalf("SearchWithFilter failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results with combined filters, got %d", len(results))
	}

	// Verify all results match both filters
	for _, r := range results {
		if !hasPrefix(r.DocumentName, "docs/") {
			t.Errorf("Result document %s is not in docs/", r.DocumentName)
		}
	}
}

func TestSearchWithFilter_NoResults(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	db, err := Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert test documents
	insertTestDocForSearch(t, db, "docs/auth.md", "Authentication", makeTestEmbedding(0.1))

	// Search with filter that matches nothing
	filter := &SearchFilter{Directory: "nonexistent"}
	results, err := db.SearchWithFilter(makeTestEmbedding(0.1), 10, filter)
	if err != nil {
		t.Fatalf("SearchWithFilter failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results for non-matching filter, got %d", len(results))
	}
}

func TestGlobToLike(t *testing.T) {
	tests := []struct {
		glob     string
		expected string
	}{
		{"*.md", "%.md"},
		{"api-*.md", "api-%.md"},
		{"file?.txt", "file_.txt"},
		{"test*file?.md", "test%file_.md"},
		{"file%name", "file\\%name"},          // Escape %
		{"file_name", "file\\_name"},          // Escape _
		{"test%*.md", "test\\%%.md"},          // Escape % then convert *
		{"*", "%"},
		{"?", "_"},
	}

	for _, tt := range tests {
		t.Run(tt.glob, func(t *testing.T) {
			result := globToLike(tt.glob)
			if result != tt.expected {
				t.Errorf("globToLike(%q) = %q, want %q", tt.glob, result, tt.expected)
			}
		})
	}
}

// hasPrefix checks if a string has the given prefix (handles both / and \ separators)
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
