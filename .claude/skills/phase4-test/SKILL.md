---
name: phase4-test
description: Phase 4テストとビルド。ユニットテスト、統合テスト作成、クロスプラットフォームビルド、動作確認、ドキュメント整備。Phase 3完了後、最終リリース準備時に使用。
---

# Phase 4: テスト・ビルド

テストを作成し、クロスプラットフォームビルドを行い、リリース準備を整えます。

## 前提条件

Phase 1-3が完了していること：
- 基盤構築完了
- コア機能実装完了
- MCP統合完了

## タスク一覧

### 4.1 ユニットテスト作成

#### config_test.go

**ファイル**: `internal/config/config_test.go`

```go
package config

import (
    "os"
    "testing"
)

func TestLoadConfig_NoFile(t *testing.T) {
    // Remove config.json if exists
    os.Remove("config.json")
    defer os.Remove("config.json")

    cfg, err := Load()
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }

    if cfg.DocumentsDir != "./documents" {
        t.Errorf("Expected default documents_dir, got %s", cfg.DocumentsDir)
    }

    if cfg.ChunkSize != 500 {
        t.Errorf("Expected default chunk_size 500, got %d", cfg.ChunkSize)
    }
}

func TestLoadConfig_Valid(t *testing.T) {
    // Create test config
    testConfig := `{
        "documents_dir": "./test_docs",
        "db_path": "./test.db",
        "chunk_size": 300,
        "search_top_k": 10
    }`

    if err := os.WriteFile("test_config.json", []byte(testConfig), 0644); err != nil {
        t.Fatal(err)
    }
    defer os.Remove("test_config.json")

    // TODO: Modify Load() to accept config path
}

func TestValidate(t *testing.T) {
    cfg := DefaultConfig()

    if err := cfg.Validate(); err != nil {
        t.Errorf("Default config should be valid, got %v", err)
    }

    // Test invalid chunk_size
    cfg.ChunkSize = -1
    if err := cfg.Validate(); err == nil {
        t.Error("Expected validation error for negative chunk_size")
    }
}
```

#### markdown_test.go

**ファイル**: `internal/indexer/markdown_test.go`

```go
package indexer

import (
    "os"
    "testing"
)

func TestParseMarkdown_ShortFile(t *testing.T) {
    content := "# Test\n\nThis is a short file."
    tmpfile, err := os.CreateTemp("", "test*.md")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpfile.Name())

    if _, err := tmpfile.WriteString(content); err != nil {
        t.Fatal(err)
    }
    tmpfile.Close()

    chunks, err := ParseMarkdown(tmpfile.Name(), 500)
    if err != nil {
        t.Fatalf("ParseMarkdown failed: %v", err)
    }

    if len(chunks) != 1 {
        t.Errorf("Expected 1 chunk, got %d", len(chunks))
    }

    if chunks[0].Position != 0 {
        t.Errorf("Expected position 0, got %d", chunks[0].Position)
    }
}

func TestParseMarkdown_LongFile(t *testing.T) {
    // Generate long content
    var content string
    for i := 0; i < 100; i++ {
        content += "This is a test paragraph with some content. "
    }

    tmpfile, err := os.CreateTemp("", "test*.md")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpfile.Name())

    if _, err := tmpfile.WriteString(content); err != nil {
        t.Fatal(err)
    }
    tmpfile.Close()

    chunks, err := ParseMarkdown(tmpfile.Name(), 500)
    if err != nil {
        t.Fatalf("ParseMarkdown failed: %v", err)
    }

    if len(chunks) < 2 {
        t.Errorf("Expected multiple chunks, got %d", len(chunks))
    }
}

func TestSplitIntoChunks(t *testing.T) {
    content := "Paragraph 1\n\nParagraph 2\n\nParagraph 3"
    chunks := splitIntoChunks(content, 50)

    if len(chunks) == 0 {
        t.Error("Expected at least one chunk")
    }
}
```

#### embedder_test.go

**ファイル**: `internal/embedder/embedder_test.go`

```go
package embedder

import (
    "testing"
)

func TestDetectDevice(t *testing.T) {
    tests := []struct {
        name          string
        deviceConfig  string
        fallback      bool
        expectsPanic  bool
    }{
        {"Auto CPU", "auto", true, false},
        {"Force CPU", "cpu", true, false},
        {"GPU with fallback", "gpu", true, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            device := DetectDevice(tt.deviceConfig, tt.fallback)
            if device != CPU && device != GPU {
                t.Errorf("Invalid device returned: %v", device)
            }
        })
    }
}

func TestSimpleTokenizer(t *testing.T) {
    tokenizer := &SimpleTokenizer{vocabSize: 30000}

    text := "Hello world"
    tokens := tokenizer.Tokenize(text)

    if len(tokens) == 0 {
        t.Error("Expected tokens, got none")
    }
}
```

#### db_test.go

**ファイル**: `internal/vectordb/db_test.go`

```go
package vectordb

import (
    "os"
    "testing"
    "time"
)

func TestInit(t *testing.T) {
    dbPath := "test.db"
    defer os.Remove(dbPath)

    db, err := Init(dbPath)
    if err != nil {
        t.Fatalf("Init failed: %v", err)
    }
    defer db.Close()

    // Verify tables exist
    var count int
    err = db.conn.QueryRow("SELECT COUNT(*) FROM documents").Scan(&count)
    if err != nil {
        t.Errorf("documents table not created: %v", err)
    }
}

func TestListDocuments(t *testing.T) {
    dbPath := "test.db"
    defer os.Remove(dbPath)

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

func TestDeleteDocument(t *testing.T) {
    dbPath := "test.db"
    defer os.Remove(dbPath)

    db, err := Init(dbPath)
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    // Insert test document
    _, err = db.conn.Exec(
        "INSERT INTO documents (filename, modified_at) VALUES (?, ?)",
        "test.md", time.Now(),
    )
    if err != nil {
        t.Fatal(err)
    }

    // Delete
    err = db.DeleteDocument("test.md")
    if err != nil {
        t.Errorf("DeleteDocument failed: %v", err)
    }

    // Verify deletion
    docs, _ := db.ListDocuments()
    if len(docs) != 0 {
        t.Errorf("Document not deleted")
    }
}
```

### 4.2 統合テスト作成

**ファイル**: `integration_test.go`

```go
package main

import (
    "os"
    "testing"

    "github.com/towada/markdown-vector-mcp/internal/config"
    "github.com/towada/markdown-vector-mcp/internal/embedder"
    "github.com/towada/markdown-vector-mcp/internal/indexer"
    "github.com/towada/markdown-vector-mcp/internal/vectordb"
)

func TestEndToEnd_FirstRun(t *testing.T) {
    // Setup
    testDir := "test_documents"
    dbPath := "test_vectors.db"
    os.MkdirAll(testDir, 0755)
    defer os.RemoveAll(testDir)
    defer os.Remove(dbPath)

    // Create test markdown
    testFile := testDir + "/test.md"
    content := "# Test Document\n\nThis is a test."
    if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
        t.Fatal(err)
    }

    // Initialize components
    cfg := config.DefaultConfig()
    cfg.DocumentsDir = testDir
    cfg.DBPath = dbPath

    db, err := vectordb.Init(dbPath)
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    // Mock embedder for testing
    // emb := &mockEmbedder{}
    // idx := indexer.NewIndexer(db, emb, cfg)

    // Test indexing
    // err = idx.IndexFile(testFile)
    // if err != nil {
    //     t.Errorf("Indexing failed: %v", err)
    // }

    // Verify in database
    docs, err := db.ListDocuments()
    if err != nil {
        t.Fatal(err)
    }

    // TODO: Verify document was indexed
    _ = docs
}
```

### 4.3 クロスプラットフォームビルド

**ファイル**: `build.sh`

```bash
#!/bin/bash

set -e

echo "Building markdown-vector-mcp..."

# Output directory
mkdir -p bin

# Build flags
LDFLAGS="-s -w"
TAGS="netgo"

# macOS (Apple Silicon)
echo "Building for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -tags "$TAGS" -ldflags="$LDFLAGS" \
  -o bin/markdown-vector-mcp-darwin-arm64 cmd/main.go

# macOS (Intel)
echo "Building for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -tags "$TAGS" -ldflags="$LDFLAGS" \
  -o bin/markdown-vector-mcp-darwin-amd64 cmd/main.go

# Windows
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -tags "$TAGS" -ldflags="$LDFLAGS" \
  -o bin/markdown-vector-mcp-windows-amd64.exe cmd/main.go

# Linux
echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -tags "$TAGS" -ldflags="$LDFLAGS" \
  -o bin/markdown-vector-mcp-linux-amd64 cmd/main.go

echo "Build complete!"
ls -lh bin/
```

**ファイル**: `build.bat`

```bat
@echo off
echo Building markdown-vector-mcp...

mkdir bin 2>nul

set LDFLAGS=-s -w
set TAGS=netgo

echo Building for Windows (amd64)...
set GOOS=windows
set GOARCH=amd64
go build -tags %TAGS% -ldflags="%LDFLAGS%" -o bin\markdown-vector-mcp-windows-amd64.exe cmd\main.go

echo Build complete!
dir bin\
```

**実行権限付与**:
```bash
chmod +x build.sh
```

### 4.4 動作確認チェックリスト

#### 基本動作
- [ ] バイナリが実行可能
- [ ] config.json が生成される
- [ ] documents/ ディレクトリが作成される
- [ ] vectors.db が作成される
- [ ] エラーなく起動する

#### 機能確認
- [ ] マークダウンファイルがインデックス化される
- [ ] 差分同期が動作する
- [ ] 検索が実行できる
- [ ] MCPツールがすべて動作する

#### プラットフォーム別
- [ ] macOS (arm64): GPU検出、基本機能
- [ ] macOS (amd64): CPU動作、基本機能
- [ ] Windows: CPU動作、パス処理
- [ ] Linux: CPU動作、基本機能

### 4.5 README.md 更新

**ファイル**: `README.md`

```markdown
# markdown-vector-mcp

マークダウンファイルをベクトル検索可能にするMCPサーバー

## Features

- 🔍 自然言語による意味的検索
- 📦 ワンバイナリー配布
- 🖥️ クロスプラットフォーム対応（Windows/macOS/Linux）
- ⚡ GPU/CPU自動検出
- 🔄 ファイル差分自動同期

## Installation

### Download Binary

[Releases](https://github.com/towada/markdown-vector-mcp/releases)から
お使いのOSに合ったバイナリをダウンロード。

### Build from Source

```bash
git clone https://github.com/towada/markdown-vector-mcp.git
cd markdown-vector-mcp
go build -o markdown-vector-mcp cmd/main.go
```

## Quick Start

### 1. 初回起動

```bash
./markdown-vector-mcp
```

自動生成されるファイル：
- `config.json` - 設定ファイル
- `documents/` - マークダウン配置ディレクトリ
- `vectors.db` - SQLiteデータベース

### 2. マークダウンファイルを配置

```bash
cp your-notes.md documents/
```

### 3. Claude Code設定

`~/.config/claude-code/config.json`:
```json
{
  "mcpServers": {
    "markdown-vector": {
      "command": "/path/to/markdown-vector-mcp"
    }
  }
}
```

### 4. 検索

Claude Codeで：
```
「JWTの認証方法について検索して」
```

## Configuration

`config.json`:
```json
{
  "documents_dir": "./documents",
  "db_path": "./vectors.db",
  "chunk_size": 500,
  "search_top_k": 5,
  "compute": {
    "device": "auto",
    "fallback_to_cpu": true
  },
  "model": {
    "name": "multilingual-e5-small",
    "dimensions": 384
  }
}
```

## MCP Tools

- `search` - 自然言語検索
- `index_markdown` - ファイルをインデックス化
- `list_documents` - ドキュメント一覧
- `delete_document` - ドキュメント削除
- `reindex_document` - 再インデックス化

## Development

```bash
# Run tests
go test ./...

# Build
./build.sh

# Run with debug
./markdown-vector-mcp --debug
```

## Troubleshooting

### GPU検出されない
`config.json`で`"device": "cpu"`を指定。

### 起動しない
- Goバージョン確認（1.21+必要）
- 依存ライブラリ確認

## License

MIT License

## Credits

- Model: [intfloat/multilingual-e5-small](https://huggingface.co/intfloat/multilingual-e5-small)
- Vector DB: [sqlite-vec](https://github.com/asg017/sqlite-vec)
```

### 4.6 CHANGELOG.md

**ファイル**: `CHANGELOG.md`

```markdown
# Changelog

All notable changes to this project will be documented in this file.

## [1.0.0] - 2025-10-24

### Added
- Initial release
- Vector search for markdown files
- MCP Protocol support
- Cross-platform builds (macOS/Windows/Linux)
- GPU/CPU auto-detection
- File sync on startup
- 5 MCP tools: search, index_markdown, list_documents, delete_document, reindex_document

### Supported Platforms
- macOS (arm64/amd64)
- Windows (amd64)
- Linux (amd64)
```

## Phase 4 完了条件

- [ ] すべてのユニットテストがパス
- [ ] 統合テストがパス
- [ ] ビルドスクリプトが動作
- [ ] 3つのOS向けバイナリが生成
- [ ] 各OSで動作確認完了
- [ ] README.mdが完備
- [ ] CHANGELOG.mdが作成
- [ ] リリース準備完了

## パフォーマンス確認

- [ ] 起動時間 < 3秒
- [ ] 検索レスポンス < 500ms (1000件/DB)
- [ ] インデックス速度 > 100 chunks/sec
- [ ] メモリ使用量 < 500MB
- [ ] バイナリサイズ < 200MB

## 次のステップ

Phase 4完了後は：
1. GitHubにリリースタグを作成
2. バイナリをアップロード
3. チームメンバーに配布
4. フィードバック収集

## リリースコマンド

```bash
# Tag release
git tag v1.0.0
git push origin v1.0.0

# Create GitHub release
gh release create v1.0.0 \
  bin/markdown-vector-mcp-* \
  --title "v1.0.0" \
  --notes-file CHANGELOG.md
```
