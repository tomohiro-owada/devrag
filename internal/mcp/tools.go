package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tomohiro-owada/devrag/internal/frontmatter"
	"github.com/tomohiro-owada/devrag/internal/updater"
	"github.com/tomohiro-owada/devrag/internal/vectordb"
	"github.com/tomohiro-owada/devrag/internal/version"
)

// Tool 1: search
func (s *MCPServer) registerSearchTool() {
	tool := mcp.NewTool(
		"search",
		mcp.WithDescription("自然言語クエリでマークダウンをベクトル検索。フィルター条件でディレクトリやファイル名パターンを指定可能。レスポンスに update_available フィールドがある場合は、新しいバージョンが利用可能なことをユーザーに必ず伝えてください"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("検索クエリ（自然言語）"),
		),
		mcp.WithNumber("top_k",
			mcp.Description("検索結果の最大件数（デフォルト: 5）"),
		),
		mcp.WithString("directory",
			mcp.Description("検索対象ディレクトリを限定（例: 'docs/api' でdocs/api配下のみ検索）"),
		),
		mcp.WithString("file_pattern",
			mcp.Description("ファイル名パターンで絞り込み（glob形式。例: 'api-*.md', '*.md'）"),
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
	directory := request.GetString("directory", "")
	filePattern := request.GetString("file_pattern", "")

	// Build filter if any filter parameters are specified
	var filter *vectordb.SearchFilter
	if directory != "" || filePattern != "" {
		filter = &vectordb.SearchFilter{
			Directory:   directory,
			FilePattern: filePattern,
		}
		fmt.Fprintf(os.Stderr, "[INFO] Search request (top_k=%d, filtered=true)\n", topK)
	} else {
		fmt.Fprintf(os.Stderr, "[INFO] Search request (top_k=%d)\n", topK)
	}

	// Vectorize query
	queryVector, err := s.embedder.Embed(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to vectorize query: %v", err)), nil
	}

	// Search with optional filter
	results, err := s.db.SearchWithFilter(queryVector, topK, filter)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	fmt.Fprintf(os.Stderr, "[INFO] Found %d results\n", len(results))

	// Build response
	response := map[string]interface{}{
		"results": results,
	}

	// Check for updates (24h cache, only notifies once)
	if updateInfo := updater.GetUpdateInfo(version.Version, ""); updateInfo != nil {
		response["update_available"] = updateInfo
	}

	return mcp.NewToolResultJSON(response)
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
		mcp.WithDescription("ドキュメントをインデックスから削除。delete_file=trueの場合は物理ファイルも削除"),
		mcp.WithString("filename",
			mcp.Required(),
			mcp.Description("削除するファイル名"),
		),
		mcp.WithBoolean("delete_file",
			mcp.Description("物理ファイルも削除するかどうか（デフォルト: false）"),
		),
	)

	s.server.AddTool(tool, s.handleDeleteDocument)
}

func (s *MCPServer) handleDeleteDocument(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filename := request.GetString("filename", "")
	if filename == "" {
		return mcp.NewToolResultError("filename is required"), nil
	}

	deleteFile := request.GetBool("delete_file", false)

	// Validate path to prevent path traversal
	if err := validatePath(filename, s.config.GetBaseDirectories()); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid path: %v", err)), nil
	}

	// Delete from database
	if err := s.db.DeleteDocument(filename); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete from database: %v", err)), nil
	}

	// Delete physical file only if explicitly requested
	fileDeleted := false
	if deleteFile {
		if err := os.Remove(filename); err != nil {
			// Log warning but don't fail since DB deletion succeeded
			fmt.Fprintf(os.Stderr, "[WARN] Failed to delete file: %v\n", err)
		} else {
			fileDeleted = true
		}
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"success":      true,
		"message":      "Document deleted successfully",
		"file_deleted": fileDeleted,
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
