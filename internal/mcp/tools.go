package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
)

// Tool 1: search
func (s *MCPServer) registerSearchTool() {
	tool := mcp.NewTool(
		"search",
		mcp.WithDescription("自然言語クエリでマークダウンをベクトル検索"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("検索クエリ（自然言語）"),
		),
		mcp.WithNumber("top_k",
			mcp.Description("検索結果の最大件数"),
		),
	)

	s.server.AddTool(tool, s.handleSearch)
}

func (s *MCPServer) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := request.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	topK := request.GetInt("top_k", s.config.SearchTopK)

	fmt.Fprintf(os.Stderr, "[INFO] Search query: %s (top_k=%d)\n", query, topK)

	// Vectorize query
	queryVector, err := s.embedder.Embed(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to vectorize query: %v", err)), nil
	}

	// Search
	results, err := s.db.Search(queryVector, topK)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	fmt.Fprintf(os.Stderr, "[INFO] Found %d results\n", len(results))

	// Format results as JSON
	return mcp.NewToolResultJSON(map[string]interface{}{
		"results": results,
	})
}

// Tool 2: index_markdown
func (s *MCPServer) registerIndexMarkdownTool() {
	tool := mcp.NewTool(
		"index_markdown",
		mcp.WithDescription("指定したマークダウンファイルをインデックス化"),
		mcp.WithString("filepath",
			mcp.Required(),
			mcp.Description("マークダウンファイルのパス"),
		),
	)

	s.server.AddTool(tool, s.handleIndexMarkdown)
}

func (s *MCPServer) handleIndexMarkdown(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filePath := request.GetString("filepath", "")
	if filePath == "" {
		return mcp.NewToolResultError("filepath is required"), nil
	}

	// Validate path (prevent path traversal)
	if err := validatePath(filePath, s.config.DocumentsDir); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid path: %v", err)), nil
	}

	// Index file
	if err := s.indexer.IndexFile(filePath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("indexing failed: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"success": true,
		"message": "File indexed successfully",
	})
}

// Tool 3: list_documents
func (s *MCPServer) registerListDocumentsTool() {
	tool := mcp.NewTool(
		"list_documents",
		mcp.WithDescription("インデックス済みドキュメント一覧を取得"),
	)

	s.server.AddTool(tool, s.handleListDocuments)
}

func (s *MCPServer) handleListDocuments(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	docs, err := s.db.ListDocuments()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list documents: %v", err)), nil
	}

	// Format response
	documents := []map[string]interface{}{}
	for filename, modTime := range docs {
		documents = append(documents, map[string]interface{}{
			"filename":    filename,
			"modified_at": modTime.Format("2006-01-02T15:04:05Z"),
		})
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"documents": documents,
	})
}

// Tool 4: delete_document
func (s *MCPServer) registerDeleteDocumentTool() {
	tool := mcp.NewTool(
		"delete_document",
		mcp.WithDescription("ドキュメントをDBとファイルシステムの両方から削除"),
		mcp.WithString("filename",
			mcp.Required(),
			mcp.Description("削除するファイル名"),
		),
	)

	s.server.AddTool(tool, s.handleDeleteDocument)
}

func (s *MCPServer) handleDeleteDocument(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filename := request.GetString("filename", "")
	if filename == "" {
		return mcp.NewToolResultError("filename is required"), nil
	}

	// Delete from database
	if err := s.db.DeleteDocument(filename); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete from database: %v", err)), nil
	}

	// Delete file
	filePath := filepath.Join(s.config.DocumentsDir, filename)
	if err := os.Remove(filePath); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] Failed to delete file: %v\n", err)
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"success": true,
		"message": "Document deleted successfully",
	})
}

// Tool 5: reindex_document
func (s *MCPServer) registerReindexDocumentTool() {
	tool := mcp.NewTool(
		"reindex_document",
		mcp.WithDescription("ドキュメントを削除して再インデックス化"),
		mcp.WithString("filename",
			mcp.Required(),
			mcp.Description("再インデックス化するファイル名"),
		),
	)

	s.server.AddTool(tool, s.handleReindexDocument)
}

func (s *MCPServer) handleReindexDocument(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filename := request.GetString("filename", "")
	if filename == "" {
		return mcp.NewToolResultError("filename is required"), nil
	}

	// Delete from database
	if err := s.db.DeleteDocument(filename); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete document: %v", err)), nil
	}

	// Reindex
	filePath := filepath.Join(s.config.DocumentsDir, filename)
	if err := s.indexer.IndexFile(filePath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to reindex: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"success": true,
		"message": "Document reindexed successfully",
	})
}

// validatePath prevents path traversal attacks
func validatePath(filePath, baseDir string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return err
	}

	relPath, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return err
	}

	// Check if path escapes base directory
	if len(relPath) > 0 && relPath[0] == '.' {
		return fmt.Errorf("path traversal detected: %s", filePath)
	}

	return nil
}
