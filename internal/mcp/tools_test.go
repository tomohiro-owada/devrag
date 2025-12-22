package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tomohiro-owada/devrag/internal/config"
	"github.com/tomohiro-owada/devrag/internal/embedder"
	"github.com/tomohiro-owada/devrag/internal/indexer"
	"github.com/tomohiro-owada/devrag/internal/vectordb"
)

// testHelper creates a fully initialized test server
func testHelper(t *testing.T) (*MCPServer, string) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	docsDir := filepath.Join(tmpDir, "docs")

	// Create docs directory
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("Failed to create docs dir: %v", err)
	}

	// Initialize database
	db, err := vectordb.Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create config
	cfg := config.DefaultConfig()
	cfg.DocumentPatterns = []string{docsDir}
	cfg.DBPath = dbPath

	// Create mock embedder
	emb := &embedder.MockEmbedder{}

	// Create indexer
	idx := indexer.NewIndexer(db, emb, cfg)

	// Create MCP server (without actually starting it)
	server := NewMCPServer(idx, db, emb, cfg)

	return server, tmpDir
}

// createTestRequest creates a CallToolRequest with given arguments
func createTestRequest(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

// createTestMarkdown creates a test markdown file
func createTestMarkdown(t *testing.T, baseDir, relativePath, content string) string {
	t.Helper()

	fullPath := filepath.Join(baseDir, relativePath)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", fullPath, err)
	}

	return fullPath
}

// ====================
// validatePath tests
// ====================

func TestValidatePath(t *testing.T) {
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs")
	os.MkdirAll(docsDir, 0755)

	absDocsDir, _ := filepath.Abs(docsDir)
	baseDirs := []string{absDocsDir}

	tests := []struct {
		name      string
		path      string
		wantError bool
	}{
		{
			name:      "valid path within base directory",
			path:      filepath.Join(docsDir, "test.md"),
			wantError: false,
		},
		{
			name:      "valid path in subdirectory",
			path:      filepath.Join(docsDir, "sub", "test.md"),
			wantError: false,
		},
		{
			name:      "path outside base directory",
			path:      filepath.Join(tmpDir, "outside.md"),
			wantError: true,
		},
		{
			name:      "path traversal attempt",
			path:      filepath.Join(docsDir, "..", "outside.md"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path, baseDirs)
			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// ====================
// search tool tests
// ====================

func TestHandleSearch_MissingQuery(t *testing.T) {
	server, _ := testHelper(t)
	ctx := context.Background()

	// Test with empty query
	request := createTestRequest(map[string]interface{}{})
	result, err := server.handleSearch(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check it's an error result
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if !result.IsError {
		t.Error("Expected error result for missing query")
	}
}

func TestHandleSearch_EmptyQuery(t *testing.T) {
	server, _ := testHelper(t)
	ctx := context.Background()

	request := createTestRequest(map[string]interface{}{
		"query": "",
	})
	result, err := server.handleSearch(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for empty query")
	}
}

func TestHandleSearch_Success(t *testing.T) {
	server, tmpDir := testHelper(t)
	ctx := context.Background()

	// Create and index a test document
	docsDir := filepath.Join(tmpDir, "docs")
	filePath := createTestMarkdown(t, tmpDir, "docs/test.md", "# Test\n\nThis is test content for searching.")

	// Index the file
	if err := server.indexer.IndexFile(filePath); err != nil {
		t.Fatalf("Failed to index file: %v", err)
	}

	// Search for content
	request := createTestRequest(map[string]interface{}{
		"query": "test content",
	})
	result, err := server.handleSearch(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success, got error: %v", result)
	}

	_ = docsDir // suppress unused variable warning
}

func TestHandleSearch_WithTopK(t *testing.T) {
	server, tmpDir := testHelper(t)
	ctx := context.Background()

	// Create multiple test documents
	for i := 0; i < 5; i++ {
		filePath := createTestMarkdown(t, tmpDir, filepath.Join("docs", "test"+string(rune('0'+i))+".md"), "# Test\n\nContent number "+string(rune('0'+i)))
		if err := server.indexer.IndexFile(filePath); err != nil {
			t.Fatalf("Failed to index file: %v", err)
		}
	}

	// Search with top_k = 2
	request := createTestRequest(map[string]interface{}{
		"query": "test content",
		"top_k": 2,
	})
	result, err := server.handleSearch(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success, got error: %v", result)
	}
}

func TestHandleSearch_WithFilters(t *testing.T) {
	server, tmpDir := testHelper(t)
	ctx := context.Background()

	// Create documents in different directories
	createTestMarkdown(t, tmpDir, "docs/api/auth.md", "# Auth API\n\nAuthentication content")
	createTestMarkdown(t, tmpDir, "docs/api/users.md", "# Users API\n\nUsers content")
	createTestMarkdown(t, tmpDir, "docs/guides/setup.md", "# Setup Guide\n\nSetup content")

	// Index all files
	if err := server.indexer.IndexFile(filepath.Join(tmpDir, "docs/api/auth.md")); err != nil {
		t.Fatalf("Failed to index: %v", err)
	}
	if err := server.indexer.IndexFile(filepath.Join(tmpDir, "docs/api/users.md")); err != nil {
		t.Fatalf("Failed to index: %v", err)
	}
	if err := server.indexer.IndexFile(filepath.Join(tmpDir, "docs/guides/setup.md")); err != nil {
		t.Fatalf("Failed to index: %v", err)
	}

	// Search with directory filter
	request := createTestRequest(map[string]interface{}{
		"query":     "content",
		"directory": "docs/api",
	})
	result, err := server.handleSearch(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success, got error")
	}
}

// ====================
// index_markdown tool tests
// ====================

func TestHandleIndexMarkdown_MissingFilepath(t *testing.T) {
	server, _ := testHelper(t)
	ctx := context.Background()

	request := createTestRequest(map[string]interface{}{})
	result, err := server.handleIndexMarkdown(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for missing filepath")
	}
}

func TestHandleIndexMarkdown_InvalidPath(t *testing.T) {
	server, _ := testHelper(t)
	ctx := context.Background()

	// Try to index a file outside the configured directories
	request := createTestRequest(map[string]interface{}{
		"filepath": "/tmp/outside.md",
	})
	result, err := server.handleIndexMarkdown(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for invalid path")
	}
}

func TestHandleIndexMarkdown_Success(t *testing.T) {
	server, tmpDir := testHelper(t)
	ctx := context.Background()

	// Create a test file
	filePath := createTestMarkdown(t, tmpDir, "docs/newfile.md", "# New File\n\nThis is new content.")

	request := createTestRequest(map[string]interface{}{
		"filepath": filePath,
	})
	result, err := server.handleIndexMarkdown(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success, got error: %v", result)
	}

	// Verify document is in database
	docs, err := server.db.ListDocuments()
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}
}

func TestHandleIndexMarkdown_FileNotFound(t *testing.T) {
	server, tmpDir := testHelper(t)
	ctx := context.Background()

	// Try to index non-existent file (but in valid directory)
	filePath := filepath.Join(tmpDir, "docs", "nonexistent.md")

	request := createTestRequest(map[string]interface{}{
		"filepath": filePath,
	})
	result, err := server.handleIndexMarkdown(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for non-existent file")
	}
}

// ====================
// list_documents tool tests
// ====================

func TestHandleListDocuments_Empty(t *testing.T) {
	server, _ := testHelper(t)
	ctx := context.Background()

	request := createTestRequest(map[string]interface{}{})
	result, err := server.handleListDocuments(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.IsError {
		t.Error("Expected success result")
	}
}

func TestHandleListDocuments_WithDocuments(t *testing.T) {
	server, tmpDir := testHelper(t)
	ctx := context.Background()

	// Create and index test documents
	file1 := createTestMarkdown(t, tmpDir, "docs/file1.md", "# File 1\n\nContent 1")
	file2 := createTestMarkdown(t, tmpDir, "docs/file2.md", "# File 2\n\nContent 2")

	server.indexer.IndexFile(file1)
	server.indexer.IndexFile(file2)

	request := createTestRequest(map[string]interface{}{})
	result, err := server.handleListDocuments(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.IsError {
		t.Error("Expected success result")
	}

	// Verify documents are returned
	docs, _ := server.db.ListDocuments()
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(docs))
	}
}

// ====================
// delete_document tool tests
// ====================

func TestHandleDeleteDocument_MissingFilename(t *testing.T) {
	server, _ := testHelper(t)
	ctx := context.Background()

	request := createTestRequest(map[string]interface{}{})
	result, err := server.handleDeleteDocument(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for missing filename")
	}
}

func TestHandleDeleteDocument_NotFound(t *testing.T) {
	server, _ := testHelper(t)
	ctx := context.Background()

	request := createTestRequest(map[string]interface{}{
		"filename": "nonexistent.md",
	})
	result, err := server.handleDeleteDocument(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for non-existent document")
	}
}

func TestHandleDeleteDocument_Success(t *testing.T) {
	server, tmpDir := testHelper(t)
	ctx := context.Background()

	// Create and index a document
	filePath := createTestMarkdown(t, tmpDir, "docs/todelete.md", "# To Delete\n\nThis will be deleted.")
	if err := server.indexer.IndexFile(filePath); err != nil {
		t.Fatalf("Failed to index: %v", err)
	}

	// Verify document exists
	docs, _ := server.db.ListDocuments()
	if len(docs) != 1 {
		t.Fatalf("Expected 1 document before delete, got %d", len(docs))
	}

	// Delete the document
	request := createTestRequest(map[string]interface{}{
		"filename": filePath,
	})
	result, err := server.handleDeleteDocument(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success, got error: %v", result)
	}

	// Verify document is deleted from database
	docs, _ = server.db.ListDocuments()
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents after delete, got %d", len(docs))
	}

	// Verify file is deleted from filesystem
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("Expected file to be deleted from filesystem")
	}
}

// ====================
// reindex_document tool tests
// ====================

func TestHandleReindexDocument_MissingFilename(t *testing.T) {
	server, _ := testHelper(t)
	ctx := context.Background()

	request := createTestRequest(map[string]interface{}{})
	result, err := server.handleReindexDocument(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for missing filename")
	}
}

func TestHandleReindexDocument_NotInDB(t *testing.T) {
	server, _ := testHelper(t)
	ctx := context.Background()

	request := createTestRequest(map[string]interface{}{
		"filename": "nonexistent.md",
	})
	result, err := server.handleReindexDocument(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for non-existent document")
	}
}

func TestHandleReindexDocument_Success(t *testing.T) {
	server, tmpDir := testHelper(t)
	ctx := context.Background()

	// Create and index a document
	filePath := createTestMarkdown(t, tmpDir, "docs/toreindex.md", "# Original Content\n\nThis is the original.")
	if err := server.indexer.IndexFile(filePath); err != nil {
		t.Fatalf("Failed to index: %v", err)
	}

	// Modify the file
	if err := os.WriteFile(filePath, []byte("# Updated Content\n\nThis is updated content."), 0644); err != nil {
		t.Fatalf("Failed to update file: %v", err)
	}

	// Reindex
	request := createTestRequest(map[string]interface{}{
		"filename": filePath,
	})
	result, err := server.handleReindexDocument(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success, got error: %v", result)
	}

	// Verify document still exists
	docs, _ := server.db.ListDocuments()
	if len(docs) != 1 {
		t.Errorf("Expected 1 document after reindex, got %d", len(docs))
	}
}

// ====================
// add_frontmatter tool tests
// ====================

func TestHandleAddFrontmatter_MissingFilepath(t *testing.T) {
	server, _ := testHelper(t)
	ctx := context.Background()

	request := createTestRequest(map[string]interface{}{})
	result, err := server.handleAddFrontmatter(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for missing filepath")
	}
}

func TestHandleAddFrontmatter_InvalidPath(t *testing.T) {
	server, _ := testHelper(t)
	ctx := context.Background()

	request := createTestRequest(map[string]interface{}{
		"filepath": "/tmp/outside.md",
	})
	result, err := server.handleAddFrontmatter(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for invalid path")
	}
}

func TestHandleAddFrontmatter_Success(t *testing.T) {
	server, tmpDir := testHelper(t)
	ctx := context.Background()

	// Create a file without frontmatter
	filePath := createTestMarkdown(t, tmpDir, "docs/nofm.md", "# No Frontmatter\n\nJust content.")

	request := createTestRequest(map[string]interface{}{
		"filepath": filePath,
		"domain":   "backend",
		"docType":  "guide",
		"language": "go",
		"tags":     "testing, mcp",
		"project":  "devrag",
	})
	result, err := server.handleAddFrontmatter(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success, got error: %v", result)
	}

	// Verify frontmatter was added
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)
	if len(contentStr) < 3 || contentStr[:3] != "---" {
		t.Error("Expected frontmatter to be added")
	}
}

func TestHandleAddFrontmatter_AlreadyExists(t *testing.T) {
	server, tmpDir := testHelper(t)
	ctx := context.Background()

	// Create a file with existing frontmatter
	content := "---\ndomain: frontend\n---\n\n# With Frontmatter\n\nContent."
	filePath := createTestMarkdown(t, tmpDir, "docs/withfm.md", content)

	request := createTestRequest(map[string]interface{}{
		"filepath": filePath,
		"domain":   "backend",
	})
	result, err := server.handleAddFrontmatter(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result when frontmatter already exists")
	}
}

// ====================
// update_frontmatter tool tests
// ====================

func TestHandleUpdateFrontmatter_MissingFilepath(t *testing.T) {
	server, _ := testHelper(t)
	ctx := context.Background()

	request := createTestRequest(map[string]interface{}{})
	result, err := server.handleUpdateFrontmatter(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for missing filepath")
	}
}

func TestHandleUpdateFrontmatter_InvalidPath(t *testing.T) {
	server, _ := testHelper(t)
	ctx := context.Background()

	request := createTestRequest(map[string]interface{}{
		"filepath": "/tmp/outside.md",
	})
	result, err := server.handleUpdateFrontmatter(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for invalid path")
	}
}

func TestHandleUpdateFrontmatter_Success(t *testing.T) {
	server, tmpDir := testHelper(t)
	ctx := context.Background()

	// Create a file with existing frontmatter
	content := "---\ndomain: frontend\n---\n\n# With Frontmatter\n\nContent."
	filePath := createTestMarkdown(t, tmpDir, "docs/updatefm.md", content)

	request := createTestRequest(map[string]interface{}{
		"filepath": filePath,
		"domain":   "backend",
		"language": "go",
	})
	result, err := server.handleUpdateFrontmatter(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success, got error: %v", result)
	}

	// Verify frontmatter was updated
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(fileContent)
	if !containsString(contentStr, "domain: backend") {
		t.Error("Expected domain to be updated to 'backend'")
	}
	if !containsString(contentStr, "language: go") {
		t.Error("Expected language to be added")
	}
}

func TestHandleUpdateFrontmatter_NoExisting(t *testing.T) {
	server, tmpDir := testHelper(t)
	ctx := context.Background()

	// Create a file without frontmatter - UpdateFrontmatter should add it
	filePath := createTestMarkdown(t, tmpDir, "docs/nofm2.md", "# No Frontmatter\n\nJust content.")

	request := createTestRequest(map[string]interface{}{
		"filepath": filePath,
		"domain":   "backend",
	})
	result, err := server.handleUpdateFrontmatter(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success (should add frontmatter), got error: %v", result)
	}

	// Verify frontmatter was added
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if !containsString(string(content), "---") {
		t.Error("Expected frontmatter to be added")
	}
}

// ====================
// Helper functions
// ====================

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Suppress unused import warning
var _ = time.Now
