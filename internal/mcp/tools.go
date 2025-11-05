package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tomohiro-owada/devrag/internal/frontmatter"
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
	if err := validatePath(filePath, s.config.GetBaseDirectories()); err != nil {
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

	// Delete file (filename is the full path)
	if err := os.Remove(filename); err != nil {
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

	// Reindex (filename is the full path)
	if err := s.indexer.IndexFile(filename); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to reindex: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"success": true,
		"message": "Document reindexed successfully",
	})
}

// Tool 6: add_frontmatter
func (s *MCPServer) registerAddFrontmatterTool() {
	tool := mcp.NewTool(
		"add_frontmatter",
		mcp.WithDescription("マークダウンファイルにメタデータ（frontmatter）を追加"),
		mcp.WithString("filepath",
			mcp.Required(),
			mcp.Description("マークダウンファイルのパス"),
		),
		mcp.WithString("domain",
			mcp.Description("領域: frontend | backend | mobile | infrastructure | other"),
		),
		mcp.WithString("docType",
			mcp.Description("文書種別: spec | design | api | guide | note | other"),
		),
		mcp.WithString("language",
			mcp.Description("言語: go | typescript | python | rust | java | kotlin | swift | other"),
		),
		mcp.WithString("tags",
			mcp.Description("タグ（カンマ区切り）: authentication, database, caching"),
		),
		mcp.WithString("project",
			mcp.Description("プロジェクト名（任意）"),
		),
	)

	s.server.AddTool(tool, s.handleAddFrontmatter)
}

func (s *MCPServer) handleAddFrontmatter(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filePath := request.GetString("filepath", "")
	if filePath == "" {
		return mcp.NewToolResultError("filepath is required"), nil
	}

	// Validate path
	if err := validatePath(filePath, s.config.GetBaseDirectories()); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid path: %v", err)), nil
	}

	// Build metadata
	metadata := &frontmatter.Metadata{
		Domain:   request.GetString("domain", ""),
		DocType:  request.GetString("docType", ""),
		Language: request.GetString("language", ""),
		Project:  request.GetString("project", ""),
	}

	// Parse tags
	tagsStr := request.GetString("tags", "")
	if tagsStr != "" {
		tags := []string{}
		for _, tag := range strings.Split(tagsStr, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
		metadata.Tags = tags
	}

	// Add frontmatter
	if err := frontmatter.AddFrontmatter(filePath, metadata); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add frontmatter: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"success": true,
		"message": "Frontmatter added successfully",
	})
}

// Tool 7: update_frontmatter
func (s *MCPServer) registerUpdateFrontmatterTool() {
	tool := mcp.NewTool(
		"update_frontmatter",
		mcp.WithDescription("マークダウンファイルのメタデータ（frontmatter）を更新"),
		mcp.WithString("filepath",
			mcp.Required(),
			mcp.Description("マークダウンファイルのパス"),
		),
		mcp.WithString("domain",
			mcp.Description("領域: frontend | backend | mobile | infrastructure | other"),
		),
		mcp.WithString("docType",
			mcp.Description("文書種別: spec | design | api | guide | note | other"),
		),
		mcp.WithString("language",
			mcp.Description("言語: go | typescript | python | rust | java | kotlin | swift | other"),
		),
		mcp.WithString("tags",
			mcp.Description("タグ（カンマ区切り）: authentication, database, caching"),
		),
		mcp.WithString("project",
			mcp.Description("プロジェクト名（任意）"),
		),
	)

	s.server.AddTool(tool, s.handleUpdateFrontmatter)
}

func (s *MCPServer) handleUpdateFrontmatter(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filePath := request.GetString("filepath", "")
	if filePath == "" {
		return mcp.NewToolResultError("filepath is required"), nil
	}

	// Validate path
	if err := validatePath(filePath, s.config.GetBaseDirectories()); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid path: %v", err)), nil
	}

	// Build metadata
	metadata := &frontmatter.Metadata{
		Domain:   request.GetString("domain", ""),
		DocType:  request.GetString("docType", ""),
		Language: request.GetString("language", ""),
		Project:  request.GetString("project", ""),
	}

	// Parse tags
	tagsStr := request.GetString("tags", "")
	if tagsStr != "" {
		tags := []string{}
		for _, tag := range strings.Split(tagsStr, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
		metadata.Tags = tags
	}

	// Update frontmatter
	if err := frontmatter.UpdateFrontmatter(filePath, metadata); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update frontmatter: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"success": true,
		"message": "Frontmatter updated successfully",
	})
}

// validatePath prevents path traversal attacks
// It checks if the file is within any of the configured base directories
func validatePath(filePath string, baseDirs []string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}

	// Check if path is within any of the base directories
	for _, baseDir := range baseDirs {
		absBase, err := filepath.Abs(baseDir)
		if err != nil {
			continue
		}

		relPath, err := filepath.Rel(absBase, absPath)
		if err != nil {
			continue
		}

		// Check if path escapes base directory
		if len(relPath) > 0 && relPath[0] != '.' {
			// Path is within this base directory
			return nil
		}
	}

	return fmt.Errorf("path not within any configured document directory: %s", filePath)
}
