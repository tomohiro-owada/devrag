package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tomohiro-owada/devrag/internal/frontmatter"
	"github.com/tomohiro-owada/devrag/internal/indexer"
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

	// Check if query contains Japanese and lookup dictionary
	var keywordResults []vectordb.SearchResult
	var translatedKeywords []string

	var dictionaryAutoBuilt bool
	if indexer.IsJapanese(query) {
		// Extract Japanese words and look up translations
		jaWords := extractJapaneseWords(query)
		for _, word := range jaWords {
			translations, err := s.db.LookupWordMappings(word, "ja")
			if err == nil && len(translations) > 0 {
				translatedKeywords = append(translatedKeywords, translations...)
				fmt.Fprintf(os.Stderr, "[INFO] Dictionary: %s -> %v\n", word, translations)
			}
		}

		// Auto-build dictionary if no translations found for Japanese words
		if len(translatedKeywords) == 0 && len(jaWords) > 0 {
			dictCount, _ := s.db.GetWordMappingCount()
			fmt.Fprintf(os.Stderr, "[INFO] No translations found, auto-building dictionary (current size: %d)\n", dictCount)

			// Build dictionary from indexed documents
			extracted := s.autoBuildDictionary("ja")
			if extracted > 0 {
				dictionaryAutoBuilt = true
				fmt.Fprintf(os.Stderr, "[INFO] Auto-built dictionary: %d new mappings\n", extracted)

				// Retry lookup after building dictionary
				for _, word := range jaWords {
					translations, err := s.db.LookupWordMappings(word, "ja")
					if err == nil && len(translations) > 0 {
						translatedKeywords = append(translatedKeywords, translations...)
						fmt.Fprintf(os.Stderr, "[INFO] Dictionary (after auto-build): %s -> %v\n", word, translations)
					}
				}
			}
		}

		// If we have translated keywords, do symbol search
		if len(translatedKeywords) > 0 {
			keywordResults, _ = s.db.SearchSymbolsByKeywords(translatedKeywords, topK)
			fmt.Fprintf(os.Stderr, "[INFO] Keyword search found %d results\n", len(keywordResults))
		}
	}

	// Vectorize query for semantic search
	queryVector, err := s.embedder.Embed(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to vectorize query: %v", err)), nil
	}

	// Search with optional filter
	semanticResults, err := s.db.SearchWithFilter(queryVector, topK, filter)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	// Merge results: keyword results first (more precise), then semantic
	results := mergeSearchResults(keywordResults, semanticResults, topK)

	fmt.Fprintf(os.Stderr, "[INFO] Found %d results (keyword: %d, semantic: %d)\n",
		len(results), len(keywordResults), len(semanticResults))

	// Build response
	response := map[string]interface{}{
		"results": results,
	}

	// Add dictionary info if translations were used
	if len(translatedKeywords) > 0 {
		response["dictionary_used"] = true
		response["translated_keywords"] = translatedKeywords
	}
	if dictionaryAutoBuilt {
		response["dictionary_auto_built"] = true
	}

	// Check for updates (24h cache, only notifies once)
	if updateInfo := updater.GetUpdateInfo(version.Version, ""); updateInfo != nil {
		response["update_available"] = updateInfo
	}

	return mcp.NewToolResultJSON(response)
}

// extractJapaneseWords extracts Japanese words from a query string
func extractJapaneseWords(query string) []string {
	var words []string
	var currentWord strings.Builder

	for _, r := range query {
		if indexer.IsJapanese(string(r)) {
			currentWord.WriteRune(r)
		} else {
			if currentWord.Len() > 0 {
				word := currentWord.String()
				if len(word) >= 2 { // Only words with 2+ characters
					words = append(words, word)
				}
				currentWord.Reset()
			}
		}
	}

	if currentWord.Len() > 0 {
		word := currentWord.String()
		if len(word) >= 2 {
			words = append(words, word)
		}
	}

	return words
}

// mergeSearchResults merges keyword and semantic results, removing duplicates
func mergeSearchResults(keyword, semantic []vectordb.SearchResult, limit int) []vectordb.SearchResult {
	seen := make(map[string]bool)
	var results []vectordb.SearchResult

	// Add keyword results first (higher priority)
	for _, r := range keyword {
		key := fmt.Sprintf("%s:%d", r.DocumentName, r.Position)
		if !seen[key] {
			seen[key] = true
			results = append(results, r)
		}
	}

	// Add semantic results
	for _, r := range semantic {
		key := fmt.Sprintf("%s:%d", r.DocumentName, r.Position)
		if !seen[key] {
			seen[key] = true
			results = append(results, r)
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}

	return results
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

// Tool 8: index_code
func (s *MCPServer) registerIndexCodeTool() {
	tool := mcp.NewTool(
		"index_code",
		mcp.WithDescription("ソースコードファイルをAST解析してインデックス化。関数・クラス・メソッドなどのシンボル単位でチャンキングし、セマンティック検索を可能にする。対応言語: Go, Python, TypeScript, JavaScript"),
		mcp.WithString("filepath",
			mcp.Description("単一ファイルをインデックス化する場合のパス"),
		),
		mcp.WithString("directory",
			mcp.Description("ディレクトリ配下のソースコードを再帰的にインデックス化"),
		),
		mcp.WithString("filepaths",
			mcp.Description("複数ファイルをインデックス化（カンマ区切り）"),
		),
		mcp.WithBoolean("force",
			mcp.Description("変更有無に関わらず強制的に再インデックス化（デフォルト: false）"),
		),
	)

	s.server.AddTool(tool, s.handleIndexCode)
}

func (s *MCPServer) handleIndexCode(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filePath := request.GetString("filepath", "")
	directory := request.GetString("directory", "")
	filepaths := request.GetString("filepaths", "")
	force := request.GetBool("force", false)

	// At least one parameter is required
	if filePath == "" && directory == "" && filepaths == "" {
		return mcp.NewToolResultError("filepath, directory, or filepaths is required"), nil
	}

	// Initialize code parser if not already done
	if err := s.indexer.InitCodeParser(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to initialize code parser: %v", err)), nil
	}

	// Handle single file
	if filePath != "" {
		fmt.Fprintf(os.Stderr, "[INFO] Indexing code file: %s\n", filePath)
		if err := s.indexer.IndexCodeFile(filePath); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to index file: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{
			"success": true,
			"message": "Code file indexed successfully",
			"file":    filePath,
		})
	}

	// Handle multiple files
	if filepaths != "" {
		files := strings.Split(filepaths, ",")
		results := []map[string]interface{}{}
		successCount := 0
		errorCount := 0

		for _, f := range files {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}

			fmt.Fprintf(os.Stderr, "[INFO] Indexing code file: %s\n", f)
			if err := s.indexer.IndexCodeFile(f); err != nil {
				results = append(results, map[string]interface{}{
					"file":    f,
					"success": false,
					"error":   err.Error(),
				})
				errorCount++
			} else {
				results = append(results, map[string]interface{}{
					"file":    f,
					"success": true,
				})
				successCount++
			}
		}

		return mcp.NewToolResultJSON(map[string]interface{}{
			"success":       errorCount == 0,
			"message":       fmt.Sprintf("Indexed %d files successfully, %d errors", successCount, errorCount),
			"results":       results,
			"success_count": successCount,
			"error_count":   errorCount,
		})
	}

	// Handle directory
	if directory != "" {
		fmt.Fprintf(os.Stderr, "[INFO] Indexing code directory: %s (force=%v)\n", directory, force)
		result, err := s.indexer.IndexCodeDirectory(directory, force)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to index directory: %v", err)), nil
		}

		return mcp.NewToolResultJSON(map[string]interface{}{
			"success":       true,
			"message":       "Directory indexing completed",
			"directory":     directory,
			"files_indexed": result.Indexed,
			"files_added":   result.Added,
			"files_updated": result.Updated,
			"files_skipped": result.Skipped,
			"files_failed":  result.Failed,
		})
	}

	return mcp.NewToolResultError("unexpected state"), nil
}

// Tool 9: search_relations
func (s *MCPServer) registerSearchRelationsTool() {
	tool := mcp.NewTool(
		"search_relations",
		mcp.WithDescription("コードシンボル間の関係を検索。呼び出し元・呼び出し先、インポート、継承関係を探索可能"),
		mcp.WithString("symbol",
			mcp.Required(),
			mcp.Description("検索するシンボル名（関数名、クラス名など）"),
		),
		mcp.WithString("relation_type",
			mcp.Description("関係タイプで絞り込み: calls | imports | inherits（省略時は全て）"),
		),
		mcp.WithString("direction",
			mcp.Description("探索方向: outgoing（呼び出し先）| incoming（呼び出し元）| both（省略時はboth）"),
		),
	)

	s.server.AddTool(tool, s.handleSearchRelations)
}

func (s *MCPServer) handleSearchRelations(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	symbol := request.GetString("symbol", "")
	if symbol == "" {
		return mcp.NewToolResultError("symbol is required"), nil
	}

	relationType := request.GetString("relation_type", "")
	direction := request.GetString("direction", "both")

	fmt.Fprintf(os.Stderr, "[INFO] Searching relations for symbol: %s (type=%s, direction=%s)\n", symbol, relationType, direction)

	// Query the database
	relations, err := s.db.FindSymbolRelations(symbol, direction, relationType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to search relations: %v", err)), nil
	}

	// Format results
	results := []map[string]interface{}{}
	for _, r := range relations {
		result := map[string]interface{}{
			"relation_type": r.RelationType,
			"target_name":   r.TargetName,
			"target_file":   r.TargetFile,
			"source_name":   r.SourceName,
			"source_file":   r.SourceFile,
			"confidence":    r.Confidence,
		}
		results = append(results, result)
	}

	fmt.Fprintf(os.Stderr, "[INFO] Found %d relations\n", len(results))

	return mcp.NewToolResultJSON(map[string]interface{}{
		"symbol":    symbol,
		"direction": direction,
		"relations": results,
		"count":     len(results),
	})
}

// Tool 10: build_dictionary
func (s *MCPServer) registerBuildDictionaryTool() {
	tool := mcp.NewTool(
		"build_dictionary",
		mcp.WithDescription("ドキュメントから多言語ワードマッピングを抽出して辞書を構築。日本語→英語の対応を自動学習"),
		mcp.WithString("source_lang",
			mcp.Description("抽出元言語（デフォルト: ja）"),
		),
		mcp.WithString("document",
			mcp.Description("対象ドキュメントパス（省略時は全文書から抽出）"),
		),
	)

	s.server.AddTool(tool, s.handleBuildDictionary)
}

func (s *MCPServer) handleBuildDictionary(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceLang := request.GetString("source_lang", "ja")
	document := request.GetString("document", "")

	fmt.Fprintf(os.Stderr, "[INFO] Building dictionary (lang=%s, doc=%s)\n", sourceLang, document)

	extractor := indexer.NewDictionaryExtractor()
	var allMappings []vectordb.WordMapping

	if document != "" {
		// Extract from specific document
		content, err := os.ReadFile(document)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to read document: %v", err)), nil
		}
		mappings := extractor.ExtractFromContent(string(content), document, sourceLang)
		allMappings = append(allMappings, mappings...)
	} else {
		// Extract from all indexed documents
		docs, err := s.db.ListDocuments()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list documents: %v", err)), nil
		}

		for docPath := range docs {
			content, err := os.ReadFile(docPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[WARN] Failed to read %s: %v\n", docPath, err)
				continue
			}

			// Detect if document has mixed language content
			lang := indexer.DetectLanguage(string(content))
			if lang == "mixed" || lang == sourceLang {
				mappings := extractor.ExtractFromContent(string(content), docPath, sourceLang)
				allMappings = append(allMappings, mappings...)
			}
		}
	}

	// Insert mappings into database
	if len(allMappings) > 0 {
		if err := s.db.InsertWordMappings(allMappings); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to insert mappings: %v", err)), nil
		}
	}

	// Get total count
	totalCount, _ := s.db.GetWordMappingCount()

	fmt.Fprintf(os.Stderr, "[INFO] Extracted %d mappings, total dictionary size: %d\n", len(allMappings), totalCount)

	// Return sample mappings
	sampleMappings := allMappings
	if len(sampleMappings) > 10 {
		sampleMappings = sampleMappings[:10]
	}

	samples := []map[string]interface{}{}
	for _, m := range sampleMappings {
		samples = append(samples, map[string]interface{}{
			"source": m.SourceWord,
			"target": m.TargetWord,
			"confidence": m.Confidence,
		})
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"success":          true,
		"extracted_count":  len(allMappings),
		"total_dictionary": totalCount,
		"sample_mappings":  samples,
	})
}

// autoBuildDictionary builds dictionary from indexed documents (internal helper for search)
func (s *MCPServer) autoBuildDictionary(sourceLang string) int {
	extractor := indexer.NewDictionaryExtractor()
	var allMappings []vectordb.WordMapping

	// Extract from all indexed documents
	docs, err := s.db.ListDocuments()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] Failed to list documents for auto-build: %v\n", err)
		return 0
	}

	for docPath := range docs {
		content, err := os.ReadFile(docPath)
		if err != nil {
			continue
		}

		// Detect if document has mixed language content
		lang := indexer.DetectLanguage(string(content))
		if lang == "mixed" || lang == sourceLang {
			mappings := extractor.ExtractFromContent(string(content), docPath, sourceLang)
			allMappings = append(allMappings, mappings...)
		}
	}

	// Insert mappings into database
	if len(allMappings) > 0 {
		if err := s.db.InsertWordMappings(allMappings); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to insert auto-built mappings: %v\n", err)
			return 0
		}
	}

	return len(allMappings)
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
