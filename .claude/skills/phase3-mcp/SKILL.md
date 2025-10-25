---
name: phase3-mcp
description: Phase 3 MCP統合。MCPサーバー実装、5つのツール（search、index_markdown、list_documents、delete_document、reindex_document）実装、メインループ統合。Phase 2完了後、MCPプロトコル統合時に使用。
---

# Phase 3: MCP統合・完成

MCPプロトコルを統合し、Claude Codeから使用可能にします。

## 前提条件

Phase 1とPhase 2が完了していること：
- 基盤構築完了
- マークダウンパーサー実装済み
- ベクトル化・検索機能実装済み
- 差分同期機能実装済み

## タスク一覧

### 3.1 MCP依存追加とセットアップ

**依存追加**:
```bash
go get github.com/mark3labs/mcp-go
```

**ファイル**: `internal/mcp/server.go`

```go
package mcp

import (
    "fmt"
    "os"

    "github.com/mark3labs/mcp-go/server"
    "github.com/towada/devrag/internal/config"
    "github.com/towada/devrag/internal/embedder"
    "github.com/towada/devrag/internal/indexer"
    "github.com/towada/devrag/internal/vectordb"
)

type MCPServer struct {
    server   *server.MCPServer
    indexer  *indexer.Indexer
    db       *vectordb.DB
    embedder embedder.Embedder
    config   *config.Config
}

// NewMCPServer creates a new MCP server
func NewMCPServer(idx *indexer.Indexer, db *vectordb.DB, emb embedder.Embedder, cfg *config.Config) *MCPServer {
    return &MCPServer{
        indexer:  idx,
        db:       db,
        embedder: emb,
        config:   cfg,
    }
}

// Start starts the MCP server
func (s *MCPServer) Start() error {
    fmt.Fprintf(os.Stderr, "[INFO] Starting MCP server...\n")

    // Create MCP server
    s.server = server.NewMCPServer(
        "devrag",
        "1.0.0",
    )

    // Register tools
    s.registerTools()

    // Start server (stdio)
    if err := s.server.Serve(); err != nil {
        return fmt.Errorf("MCP server error: %w", err)
    }

    return nil
}

// registerTools registers all MCP tools
func (s *MCPServer) registerTools() {
    s.registerSearchTool()
    s.registerIndexMarkdownTool()
    s.registerListDocumentsTool()
    s.registerDeleteDocumentTool()
    s.registerReindexDocumentTool()

    fmt.Fprintf(os.Stderr, "[INFO] Registered 5 MCP tools\n")
}
```

### 3.2 MCPツール実装

**ファイル**: `internal/mcp/tools.go`

```go
package mcp

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/mark3labs/mcp-go/server"
)

// Tool 1: search
func (s *MCPServer) registerSearchTool() {
    tool := server.Tool{
        Name:        "search",
        Description: "自然言語クエリでマークダウンをベクトル検索",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "query": map[string]interface{}{
                    "type":        "string",
                    "description": "検索クエリ（自然言語）",
                },
                "top_k": map[string]interface{}{
                    "type":        "integer",
                    "description": "検索結果の最大件数",
                    "default":     5,
                },
            },
            "required": []string{"query"},
        },
    }

    s.server.AddTool(tool, s.handleSearch)
}

func (s *MCPServer) handleSearch(args map[string]interface{}) (interface{}, error) {
    query, ok := args["query"].(string)
    if !ok {
        return nil, fmt.Errorf("query must be a string")
    }

    topK := s.config.SearchTopK
    if k, ok := args["top_k"].(float64); ok {
        topK = int(k)
    }

    fmt.Fprintf(os.Stderr, "[INFO] Search query: %s (top_k=%d)\n", query, topK)

    // Vectorize query
    queryVector, err := s.embedder.Embed(query)
    if err != nil {
        return nil, fmt.Errorf("failed to vectorize query: %w", err)
    }

    // Search
    results, err := s.db.Search(queryVector, topK)
    if err != nil {
        return nil, fmt.Errorf("search failed: %w", err)
    }

    // Format results
    response := map[string]interface{}{
        "results": results,
    }

    fmt.Fprintf(os.Stderr, "[INFO] Found %d results\n", len(results))
    return response, nil
}

// Tool 2: index_markdown
func (s *MCPServer) registerIndexMarkdownTool() {
    tool := server.Tool{
        Name:        "index_markdown",
        Description: "指定したマークダウンファイルをインデックス化",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "filepath": map[string]interface{}{
                    "type":        "string",
                    "description": "マークダウンファイルのパス",
                },
            },
            "required": []string{"filepath"},
        },
    }

    s.server.AddTool(tool, s.handleIndexMarkdown)
}

func (s *MCPServer) handleIndexMarkdown(args map[string]interface{}) (interface{}, error) {
    filePath, ok := args["filepath"].(string)
    if !ok {
        return nil, fmt.Errorf("filepath must be a string")
    }

    // Validate path (prevent path traversal)
    if err := validatePath(filePath, s.config.DocumentsDir); err != nil {
        return nil, err
    }

    // Index file
    if err := s.indexer.IndexFile(filePath); err != nil {
        return nil, fmt.Errorf("indexing failed: %w", err)
    }

    return map[string]interface{}{
        "success": true,
        "message": "File indexed successfully",
    }, nil
}

// Tool 3: list_documents
func (s *MCPServer) registerListDocumentsTool() {
    tool := server.Tool{
        Name:        "list_documents",
        Description: "インデックス済みドキュメント一覧を取得",
        InputSchema: map[string]interface{}{
            "type":       "object",
            "properties": map[string]interface{}{},
        },
    }

    s.server.AddTool(tool, s.handleListDocuments)
}

func (s *MCPServer) handleListDocuments(args map[string]interface{}) (interface{}, error) {
    docs, err := s.db.ListDocuments()
    if err != nil {
        return nil, fmt.Errorf("failed to list documents: %w", err)
    }

    // Format response
    documents := []map[string]interface{}{}
    for filename, modTime := range docs {
        documents = append(documents, map[string]interface{}{
            "filename":    filename,
            "modified_at": modTime.Format("2006-01-02T15:04:05Z"),
        })
    }

    return map[string]interface{}{
        "documents": documents,
    }, nil
}

// Tool 4: delete_document
func (s *MCPServer) registerDeleteDocumentTool() {
    tool := server.Tool{
        Name:        "delete_document",
        Description: "ドキュメントをDBとファイルシステムの両方から削除",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "filename": map[string]interface{}{
                    "type":        "string",
                    "description": "削除するファイル名",
                },
            },
            "required": []string{"filename"},
        },
    }

    s.server.AddTool(tool, s.handleDeleteDocument)
}

func (s *MCPServer) handleDeleteDocument(args map[string]interface{}) (interface{}, error) {
    filename, ok := args["filename"].(string)
    if !ok {
        return nil, fmt.Errorf("filename must be a string")
    }

    // Delete from database
    if err := s.db.DeleteDocument(filename); err != nil {
        return nil, fmt.Errorf("failed to delete from database: %w", err)
    }

    // Delete file
    filePath := filepath.Join(s.config.DocumentsDir, filename)
    if err := os.Remove(filePath); err != nil {
        fmt.Fprintf(os.Stderr, "[WARN] Failed to delete file: %v\n", err)
    }

    return map[string]interface{}{
        "success": true,
        "message": "Document deleted successfully",
    }, nil
}

// Tool 5: reindex_document
func (s *MCPServer) registerReindexDocumentTool() {
    tool := server.Tool{
        Name:        "reindex_document",
        Description: "ドキュメントを削除して再インデックス化",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "filename": map[string]interface{}{
                    "type":        "string",
                    "description": "再インデックス化するファイル名",
                },
            },
            "required": []string{"filename"},
        },
    }

    s.server.AddTool(tool, s.handleReindexDocument)
}

func (s *MCPServer) handleReindexDocument(args map[string]interface{}) (interface{}, error) {
    filename, ok := args["filename"].(string)
    if !ok {
        return nil, fmt.Errorf("filename must be a string")
    }

    // Delete from database
    if err := s.db.DeleteDocument(filename); err != nil {
        return nil, fmt.Errorf("failed to delete document: %w", err)
    }

    // Reindex
    filePath := filepath.Join(s.config.DocumentsDir, filename)
    if err := s.indexer.IndexFile(filePath); err != nil {
        return nil, fmt.Errorf("failed to reindex: %w", err)
    }

    return map[string]interface{}{
        "success": true,
        "message": "Document reindexed successfully",
    }, nil
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
```

### 3.3 メインループ統合

**ファイル**: `cmd/main.go`

```go
package main

import (
    "fmt"
    "os"

    "github.com/towada/devrag/internal/config"
    "github.com/towada/devrag/internal/embedder"
    "github.com/towada/devrag/internal/indexer"
    "github.com/towada/devrag/internal/mcp"
    "github.com/towada/devrag/internal/vectordb"
)

func main() {
    // 1. Load configuration
    cfg, err := config.Load()
    if err != nil {
        fmt.Fprintf(os.Stderr, "[FATAL] Failed to load config: %v\n", err)
        os.Exit(1)
    }

    if err := cfg.Validate(); err != nil {
        fmt.Fprintf(os.Stderr, "[FATAL] Invalid config: %v\n", err)
        os.Exit(1)
    }

    // 2. Detect device
    device := embedder.DetectDevice(cfg.Compute.Device, cfg.Compute.FallbackToCPU)
    fmt.Fprintf(os.Stderr, "[INFO] Using device: %s\n", device)

    // 3. Initialize components
    db, err := vectordb.Init(cfg.DBPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[FATAL] Failed to initialize database: %v\n", err)
        os.Exit(1)
    }
    defer db.Close()

    // TODO: Use actual model file
    emb, err := embedder.NewONNXEmbedder("models/model.onnx", device)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[FATAL] Failed to initialize embedder: %v\n", err)
        os.Exit(1)
    }
    defer emb.Close()

    idx := indexer.NewIndexer(db, emb, cfg)

    // 4. Sync documents
    fmt.Fprintf(os.Stderr, "[INFO] Syncing documents...\n")
    syncResult, err := idx.Sync()
    if err != nil {
        fmt.Fprintf(os.Stderr, "[WARN] Sync error: %v\n", err)
    } else {
        fmt.Fprintf(os.Stderr, "[INFO] Sync complete: +%d, ~%d, -%d\n",
            len(syncResult.Added),
            len(syncResult.Updated),
            len(syncResult.Deleted))
    }

    // 5. Start MCP server
    fmt.Fprintf(os.Stderr, "[INFO] Starting MCP server...\n")
    server := mcp.NewMCPServer(idx, db, emb, cfg)
    if err := server.Start(); err != nil {
        fmt.Fprintf(os.Stderr, "[FATAL] MCP server error: %v\n", err)
        os.Exit(1)
    }
}
```

## Phase 3 完了条件

- [ ] MCPサーバーが起動する
- [ ] 5つのツールすべてが登録されている
- [ ] 各ツールが正しく動作する
- [ ] Claude Codeから呼び出せる
- [ ] エラーが適切にハンドリングされる
- [ ] ログが適切に出力される（stderr）

## 動作確認方法

### 1. バイナリをビルド

```bash
go build -o devrag cmd/main.go
```

### 2. Claude Code設定

`~/.config/claude-code/config.json`:
```json
{
  "mcpServers": {
    "devrag": {
      "command": "/path/to/devrag"
    }
  }
}
```

### 3. Claude Codeから動作確認

```
Claude Codeで「マークダウンドキュメントを検索して」と入力
```

## 注意事項

### stdio通信
- `os.Stdout`: MCPプロトコル専用
- `os.Stderr`: すべてのログ出力

### エラーハンドリング
- すべてのツールで適切なエラーレスポンス
- ユーザーフレンドリーなエラーメッセージ

### セキュリティ
- パストラバーサル対策（validatePath関数）
- SQLインジェクション対策（プリペアドステートメント）
- 入力検証

### リソース管理
- 適切なdefer処理
- データベース接続のクローズ
- ONNX Runtimeのクリーンアップ

## 次のステップ

Phase 3完了後は **phase4-test** スキルを使用してテストとビルドを行います。
