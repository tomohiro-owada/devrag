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
