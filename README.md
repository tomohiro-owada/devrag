# DevRag

**Free Local RAG for Claude Code - Save Tokens & Time**

[日本語版はこちら](#日本語版) | [Japanese Version](#日本語版)

DevRag is a lightweight RAG (Retrieval-Augmented Generation) system designed specifically for developers using Claude Code. Stop wasting tokens by reading entire documents - let vector search find exactly what you need.

## Why DevRag?

When using Claude Code, reading documents with the Read tool consumes massive amounts of tokens:

- ❌ **Wasting Context**: Reading entire docs every time (3,000+ tokens per file)
- ❌ **Poor Searchability**: Claude doesn't know which file contains what
- ❌ **Repetitive**: Same documents read multiple times across sessions

**With DevRag:**

- ✅ **40x Less Tokens**: Vector search retrieves only relevant chunks (~200 tokens)
- ✅ **15x Faster**: Search in 100ms vs 30 seconds of reading
- ✅ **Auto-Discovery**: Claude Code finds documents without knowing file names

## Features

- 🤖 **Simple RAG** - Retrieval-Augmented Generation for Claude Code
- 📝 **Markdown Support** - Auto-indexes .md files
- 🔍 **Semantic Search** - Natural language queries like "JWT authentication method"
- 🚀 **Single Binary** - No Python, models auto-download on first run
- 🖥️ **Cross-Platform** - macOS / Linux / Windows
- ⚡ **Fast** - Auto GPU/CPU detection, incremental sync
- 🌐 **Multilingual** - Supports 100+ languages including Japanese & English

## Quick Start

### 1. Download Binary

Get the appropriate binary from [Releases](https://github.com/tomohiro-owada/devrag/releases):

| Platform | Download |
|----------|----------|
| macOS (Apple Silicon) | [devrag-macos-apple-silicon.tar.gz](https://github.com/tomohiro-owada/devrag/releases/download/v1.3.0/devrag-macos-apple-silicon.tar.gz) |
| macOS (Intel) | [devrag-macos-intel.tar.gz](https://github.com/tomohiro-owada/devrag/releases/download/v1.3.0/devrag-macos-intel.tar.gz) |
| Linux (x64) | Coming soon |
| Linux (ARM64) | Coming soon |
| Windows (x64) | Coming soon |

**macOS/Linux:**
```bash
tar -xzf devrag-*.tar.gz
chmod +x devrag-*
sudo mv devrag-* /usr/local/bin/devrag
```

**Windows:**
- Extract the zip file
- Place in your preferred location (e.g., `C:\Program Files\devrag\`)

### 2. Configure Claude Code

Add to `~/.claude.json` or `.mcp.json`:

```json
{
  "mcpServers": {
    "devrag": {
      "type": "stdio",
      "command": "/usr/local/bin/devrag"
    }
  }
}
```

**Using a custom config file:**
```json
{
  "mcpServers": {
    "devrag": {
      "type": "stdio",
      "command": "/usr/local/bin/devrag",
      "args": ["--config", "/path/to/custom-config.json"]
    }
  }
}
```

### 3. Add Your Documents

```bash
mkdir documents
cp your-notes.md documents/
```

That's it! Documents are automatically indexed on startup.

### 4. Search with Claude Code

In Claude Code:
```
"Search for JWT authentication methods"
```

## Configuration

Create `config.json`:

```json
{
  "document_patterns": [
    "./documents",
    "./notes/**/*.md",
    "./projects/backend/**/*.md"
  ],
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

### Configuration Options

- `document_patterns`: Array of document paths and glob patterns
  - Supports directory paths: `"./documents"`
  - Supports glob patterns: `"./docs/**/*.md"` (recursive)
  - Multiple patterns: Index files from different locations
  - **Note**: Old `documents_dir` field is still supported (automatically migrated)
- `db_path`: Vector database file path
- `chunk_size`: Document chunk size in characters
- `search_top_k`: Number of search results to return
- `compute.device`: Compute device (`auto`, `cpu`, `gpu`)
- `compute.fallback_to_cpu`: Fallback to CPU if GPU unavailable
- `model.name`: Embedding model name
- `model.dimensions`: Vector dimensions

### Command-Line Options

- `--config <path>`: Specify a custom configuration file path (default: `config.json`)

**Example:**
```bash
devrag --config /path/to/custom-config.json
```

This is useful for:
- Running multiple instances with different configurations
- Testing different models or chunk sizes
- Maintaining separate dev/test/prod configurations

### Pattern Examples

```json
{
  "document_patterns": [
    "./documents",                    // All .md files in documents/
    "./notes/**/*.md",                // Recursive search in notes/
    "./projects/*/docs/*.md",         // docs/ in each project
    "/path/to/external/docs"          // Absolute path
  ]
}
```

## MCP Tools

DevRag provides the following tools via Model Context Protocol:

### search
Perform semantic vector search with optional filtering

**Parameters:**
- `query` (string, required): Search query in natural language
- `top_k` (number, optional): Maximum number of results (default: 5)
- `directory` (string, optional): Filter to specific directory (e.g., "docs/api")
- `file_pattern` (string, optional): Glob pattern for filename (e.g., "api-*.md", "*.md")

**Returns:**
Array of search results with filename, chunk content, and similarity score

**Examples:**
```
// Basic search
search(query: "JWT authentication")

// Search only in docs/api directory
search(query: "user endpoints", directory: "docs/api")

// Search only files matching pattern
search(query: "deployment", file_pattern: "guide-*.md")

// Combined filters
search(query: "authentication", directory: "docs/api", file_pattern: "auth*.md")
```

### index_markdown
Index a markdown file

**Parameters:**
- `filepath` (string): Path to the file to index

### list_documents
List all indexed documents

**Returns:**
Document list with filenames and timestamps

### delete_document
Remove a document from the index

**Parameters:**
- `filepath` (string): Path to the file to delete

### reindex_document
Re-index a document

**Parameters:**
- `filepath` (string): Path to the file to re-index

## Team Development

Perfect for teams with large documentation repositories:

1. **Manage docs in Git**: Normal Git workflow
2. **Each developer runs DevRag**: Local setup on each machine
3. **Search via Claude Code**: Everyone can search all docs
4. **Auto-sync**: `git pull` automatically updates the index

Configure for your project's docs directory:

```json
{
  "document_patterns": [
    "./docs",
    "./api-docs/**/*.md",
    "./wiki/**/*.md"
  ],
  "db_path": "./.devrag/vectors.db"
}
```

## Performance

Environment: MacBook Pro M2, 100 files (1MB total)

| Operation | Time | Tokens |
|-----------|------|--------|
| Startup | 2.3s | - |
| Indexing | 8.5s | - |
| Search (1 query) | 95ms | ~300 |
| Traditional Read | 25s | ~12,000 |

**260x faster search, 40x fewer tokens**

## Development

### Run Tests

```bash
# All tests
go test ./...

# Specific packages
go test ./internal/config -v
go test ./internal/indexer -v
go test ./internal/embedder -v
go test ./internal/vectordb -v

# Integration tests
go test . -v -run TestEndToEnd
```

### Build

```bash
# Using build script
./build.sh

# Direct build
go build -o devrag cmd/main.go

# Cross-platform release build
./scripts/build-release.sh
```

### Creating a Release

```bash
# Create version tag
git tag v1.0.1

# Push tag
git push origin v1.0.1
```

GitHub Actions automatically:
1. Builds for all platforms
2. Creates GitHub Release
3. Uploads binaries
4. Generates checksums

## Project Structure

```
devrag/
├── cmd/
│   └── main.go              # Entry point
├── internal/
│   ├── config/              # Configuration
│   ├── embedder/            # Vector embeddings
│   ├── indexer/             # Indexing logic
│   ├── mcp/                 # MCP server
│   └── vectordb/            # Vector database
├── models/                  # ONNX models
├── build.sh                 # Build script
└── integration_test.go      # Integration tests
```

## Troubleshooting

### Model Download Fails

**Cause**: Internet connection or Hugging Face server issues

**Solutions**:
1. Check internet connection
2. For proxy environments:
   ```bash
   export HTTP_PROXY=http://your-proxy:port
   export HTTPS_PROXY=http://your-proxy:port
   ```
3. Manual download (see `models/DOWNLOAD.md`)
4. Retry (incomplete files are auto-removed)

### GPU Not Detected

Explicitly set CPU in `config.json`:

```json
{
  "compute": {
    "device": "cpu",
    "fallback_to_cpu": true
  }
}
```

### Won't Start

- Ensure Go 1.21+ is installed (for building)
- Check CGO is enabled: `go env CGO_ENABLED`
- Verify dependencies are installed
- Internet required for first run (model download)

### Unexpected Search Results

- Adjust `chunk_size` (default: 500)
- Rebuild index (delete vectors.db and restart)

### High Memory Usage

- GPU mode loads model into VRAM
- Switch to CPU mode for lower memory usage

## Requirements

- Go 1.21+ (for building from source)
- CGO enabled (for sqlite-vec)
- macOS, Linux, or Windows

## License

MIT License

## Credits

- Embedding Model: [intfloat/multilingual-e5-small](https://huggingface.co/intfloat/multilingual-e5-small)
- Vector Database: [sqlite-vec](https://github.com/asg017/sqlite-vec)
- MCP Protocol: [Model Context Protocol](https://modelcontextprotocol.io/)
- ONNX Runtime: [onnxruntime-go](https://github.com/yalue/onnxruntime_go)

## Contributing

Issues and Pull Requests are welcome!

## Contributors

Special thanks to all contributors who helped improve DevRag:

- **[@badri](https://github.com/badri)** - Multiple document paths with glob patterns ([#2](https://github.com/tomohiro-owada/devrag/pull/2)), `--config` CLI flag ([#3](https://github.com/tomohiro-owada/devrag/pull/3))
- **[@io41](https://github.com/io41)** - Project cleanup and documentation improvements ([#4](https://github.com/tomohiro-owada/devrag/pull/4))

Your contributions make DevRag better for everyone!

## Author

[towada](https://github.com/tomohiro-owada)

---

# 日本語版

**Claude Code用の無料ローカルRAG - トークン＆時間を節約**

DevRagは、Claude Codeを使う開発者のための軽量RAG（Retrieval-Augmented Generation）システムです。ドキュメント全体を読み込んでトークンを無駄にするのをやめて、ベクトル検索で必要な情報だけを取得しましょう。

## なぜDevRagが必要か？

Claude Codeでドキュメントを読み込むと、大量のトークンを消費します：

- ❌ **コンテキストの浪費**: 毎回ドキュメント全体を読み込み（1ファイル3,000トークン以上）
- ❌ **検索性の欠如**: Claudeはどのファイルに何が書いてあるか知らない
- ❌ **繰り返し**: セッションをまたいで同じドキュメントを何度も読む

**DevRagを使えば:**

- ✅ **トークン消費1/40**: ベクトル検索で必要な部分だけ取得（約200トークン）
- ✅ **15倍高速**: 検索100ms vs 読み込み30秒
- ✅ **自動発見**: ファイル名を知らなくてもClaude Codeが見つける

## 特徴

- 🤖 **簡易RAG** - Claude Code用の検索拡張生成
- 📝 **マークダウン対応** - .mdファイルを自動インデックス化
- 🔍 **意味検索** - 「JWTの認証方法」のような自然言語クエリ
- 🚀 **ワンバイナリー** - Python不要、モデルは初回起動時に自動ダウンロード
- 🖥️ **クロスプラットフォーム** - macOS / Linux / Windows
- ⚡ **高速** - GPU/CPU自動検出、差分同期
- 🌐 **多言語** - 日本語・英語を含む100以上の言語対応

## クイックスタート

### 1. バイナリダウンロード

[Releases](https://github.com/tomohiro-owada/devrag/releases)から環境に合ったファイルをダウンロード：

| プラットフォーム | ダウンロード |
|----------|----------|
| macOS (Apple Silicon) | [devrag-macos-apple-silicon.tar.gz](https://github.com/tomohiro-owada/devrag/releases/download/v1.3.0/devrag-macos-apple-silicon.tar.gz) |
| macOS (Intel) | [devrag-macos-intel.tar.gz](https://github.com/tomohiro-owada/devrag/releases/download/v1.3.0/devrag-macos-intel.tar.gz) |
| Linux (x64) | Coming soon |
| Linux (ARM64) | Coming soon |
| Windows (x64) | Coming soon |

**macOS/Linux:**
```bash
tar -xzf devrag-*.tar.gz
chmod +x devrag-*
sudo mv devrag-* /usr/local/bin/devrag
```

**Windows:**
- zipファイルを解凍
- 任意の場所に配置（例: `C:\Program Files\devrag\`）

### 2. Claude Code設定

`~/.claude.json` または `.mcp.json` に追加：

```json
{
  "mcpServers": {
    "devrag": {
      "type": "stdio",
      "command": "/usr/local/bin/devrag"
    }
  }
}
```

**カスタム設定ファイルを使用する場合:**
```json
{
  "mcpServers": {
    "devrag": {
      "type": "stdio",
      "command": "/usr/local/bin/devrag",
      "args": ["--config", "/path/to/custom-config.json"]
    }
  }
}
```

### 3. ドキュメントを配置

```bash
mkdir documents
cp your-notes.md documents/
```

これで完了！起動時に自動的にインデックス化されます。

### 4. Claude Codeで検索

Claude Codeで：
```
「JWTの認証方法について検索して」
```

## 設定

`config.json`を作成：

```json
{
  "document_patterns": [
    "./documents",
    "./notes/**/*.md",
    "./projects/backend/**/*.md"
  ],
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

### 設定項目

- `document_patterns`: ドキュメントのパスとglobパターンの配列
  - ディレクトリパス対応: `"./documents"`
  - globパターン対応: `"./docs/**/*.md"` (再帰的)
  - 複数パターン: 異なる場所からファイルをインデックス化
  - **注意**: 旧形式の`documents_dir`もサポート（自動的に移行）
- `db_path`: ベクトルデータベースのパス
- `chunk_size`: ドキュメントのチャンクサイズ（文字数）
- `search_top_k`: 検索結果の返却件数
- `compute.device`: 計算デバイス（`auto`, `cpu`, `gpu`）
- `compute.fallback_to_cpu`: GPU利用不可時にCPUにフォールバック
- `model.name`: 埋め込みモデル名
- `model.dimensions`: ベクトル次元数

### コマンドラインオプション

- `--config <path>`: カスタム設定ファイルのパスを指定（デフォルト: `config.json`）

**使用例:**
```bash
devrag --config /path/to/custom-config.json
```

これは以下の用途で便利です：
- 異なる設定で複数のインスタンスを実行
- 異なるモデルやチャンクサイズをテスト
- 開発/テスト/本番環境の設定を分離

### パターン例

```json
{
  "document_patterns": [
    "./documents",                    // documents/内の全.mdファイル
    "./notes/**/*.md",                // notes/内を再帰的に検索
    "./projects/*/docs/*.md",         // 各プロジェクトのdocs/
    "/path/to/external/docs"          // 絶対パス
  ]
}
```

## MCPツール

Model Context Protocolを通じて以下のツールを提供：

### search
フィルター機能付き意味ベクトル検索を実行

**パラメータ:**
- `query` (string, 必須): 自然言語の検索クエリ
- `top_k` (number, 任意): 最大結果数（デフォルト: 5）
- `directory` (string, 任意): 特定ディレクトリに絞り込み（例: "docs/api"）
- `file_pattern` (string, 任意): ファイル名のglobパターン（例: "api-*.md", "*.md"）

**戻り値:**
ファイル名、チャンク内容、類似度スコアを含む検索結果の配列

**使用例:**
```
// 基本検索
search(query: "JWT認証")

// docs/apiディレクトリ内のみ検索
search(query: "ユーザーエンドポイント", directory: "docs/api")

// パターンに一致するファイルのみ検索
search(query: "デプロイ", file_pattern: "guide-*.md")

// フィルターの組み合わせ
search(query: "認証", directory: "docs/api", file_pattern: "auth*.md")
```

### index_markdown
マークダウンファイルをインデックス化

**パラメータ:**
- `filepath` (string): インデックス化するファイルのパス

### list_documents
インデックス化されたドキュメントの一覧を取得

**戻り値:**
ファイル名とタイムスタンプを含むドキュメントリスト

### delete_document
ドキュメントをインデックスから削除

**パラメータ:**
- `filepath` (string): 削除するファイルのパス

### reindex_document
ドキュメントを再インデックス化

**パラメータ:**
- `filepath` (string): 再インデックス化するファイルのパス

## チーム開発

大量のドキュメントがあるチームに最適：

1. **Gitでドキュメント管理**: 通常のGitワークフロー
2. **各開発者がDevRagを起動**: 各マシンでローカルセットアップ
3. **Claude Codeで検索**: 全員が全ドキュメントを検索可能
4. **自動同期**: `git pull`でインデックスを自動更新

プロジェクトのdocsディレクトリ用に設定：

```json
{
  "document_patterns": [
    "./docs",
    "./api-docs/**/*.md",
    "./wiki/**/*.md"
  ],
  "db_path": "./.devrag/vectors.db"
}
```

## パフォーマンス

環境: MacBook Pro M2, 100ファイル (合計1MB)

| 操作 | 時間 | トークン |
|------|------|----------|
| 起動 | 2.3秒 | - |
| インデックス化 | 8.5秒 | - |
| 検索 (1クエリ) | 95ms | ~300 |
| 従来のRead | 25秒 | ~12,000 |

**検索は260倍速、トークンは40分の1**

## 開発

### テスト実行

```bash
# すべてのテスト
go test ./...

# 特定のパッケージ
go test ./internal/config -v
go test ./internal/indexer -v
go test ./internal/embedder -v
go test ./internal/vectordb -v

# 統合テスト
go test . -v -run TestEndToEnd
```

### ビルド

```bash
# ビルドスクリプト使用
./build.sh

# 直接ビルド
go build -o devrag cmd/main.go

# クロスプラットフォームリリースビルド
./scripts/build-release.sh
```

### リリース作成

```bash
# バージョンタグを作成
git tag v1.0.1

# タグをプッシュ
git push origin v1.0.1
```

GitHub Actionsが自動的に：
1. 全プラットフォーム向けにビルド
2. GitHub Releaseを作成
3. バイナリをアップロード
4. チェックサムを生成

## プロジェクト構造

```
devrag/
├── cmd/
│   └── main.go              # エントリーポイント
├── internal/
│   ├── config/              # 設定管理
│   ├── embedder/            # ベクトル埋め込み
│   ├── indexer/             # インデックス処理
│   ├── mcp/                 # MCPサーバー
│   └── vectordb/            # ベクトルDB
├── models/                  # ONNXモデル
├── build.sh                 # ビルドスクリプト
└── integration_test.go      # 統合テスト
```

## トラブルシューティング

### モデルのダウンロードに失敗

**原因**: インターネット接続またはHugging Faceサーバーの問題

**解決方法**:
1. インターネット接続を確認
2. プロキシ環境の場合：
   ```bash
   export HTTP_PROXY=http://your-proxy:port
   export HTTPS_PROXY=http://your-proxy:port
   ```
3. 手動ダウンロード（`models/DOWNLOAD.md`参照）
4. 再試行（不完全なファイルは自動削除）

### GPUが検出されない

`config.json`で明示的にCPUを指定：

```json
{
  "compute": {
    "device": "cpu",
    "fallback_to_cpu": true
  }
}
```

### 起動しない

- Go 1.21+がインストールされているか確認（ソースからビルドする場合）
- CGOが有効か確認: `go env CGO_ENABLED`
- 依存関係がインストールされているか確認
- 初回起動時はインターネット接続が必要（モデルダウンロード）

### 検索結果が期待と異なる

- `chunk_size`を調整（デフォルト: 500）
- インデックスを再構築（vectors.dbを削除して再起動）

### メモリ使用量が多い

- GPUモードではモデルがVRAMにロード
- CPUモードに切り替えるとメモリ使用量が減少

## 必要要件

- Go 1.21+（ソースからビルドする場合）
- CGO有効（sqlite-vecのため）
- macOS, Linux, または Windows

## ライセンス

MIT License

## クレジット

- 埋め込みモデル: [intfloat/multilingual-e5-small](https://huggingface.co/intfloat/multilingual-e5-small)
- ベクトルデータベース: [sqlite-vec](https://github.com/asg017/sqlite-vec)
- MCPプロトコル: [Model Context Protocol](https://modelcontextprotocol.io/)
- ONNX Runtime: [onnxruntime-go](https://github.com/yalue/onnxruntime_go)

## コントリビューション

IssuesとPull Requestsを歓迎します！

## コントリビューター

DevRagの改善に貢献してくださった皆様に感謝します：

- **[@badri](https://github.com/badri)** - 複数ドキュメントパスとglobパターン対応 ([#2](https://github.com/tomohiro-owada/devrag/pull/2))、`--config` CLIフラグ ([#3](https://github.com/tomohiro-owada/devrag/pull/3))
- **[@io41](https://github.com/io41)** - プロジェクトのクリーンアップとドキュメント改善 ([#4](https://github.com/tomohiro-owada/devrag/pull/4))

皆様の貢献がDevRagをより良くしています！

## 作者

[towada](https://github.com/tomohiro-owada)
