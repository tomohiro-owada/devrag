# markdown-vector-mcp

マークダウンファイルをベクトル検索可能にするMCPサーバー

## Features

- 🔍 自然言語による意味的検索
- 🖥️ クロスプラットフォーム対応（macOS/Linux/Windows）
- ⚡ GPU/CPU自動検出
- 🔄 ファイル差分自動同期
- 🌐 多言語対応（multilingual-e5-smallモデル使用）

## Installation

### Download Binary

[Releases](https://github.com/towada/markdown-vector-mcp/releases)ページから
お使いのOSに合ったバイナリをダウンロードしてください：

| Platform | Description | File |
|----------|-------------|------|
| macOS | Apple Silicon (M1/M2/M3) | `markdown-vector-mcp-macos-apple-silicon.tar.gz` |
| macOS | Intel | `markdown-vector-mcp-macos-intel.tar.gz` |
| Linux | x86_64 / x64 | `markdown-vector-mcp-linux-x64.tar.gz` |
| Linux | ARM64 | `markdown-vector-mcp-linux-arm64.tar.gz` |
| Windows | x64 | `markdown-vector-mcp-windows-x64.zip` |

ダウンロード後、解凍してバイナリを適切な場所に配置してください。

**macOS/Linuxの場合:**
```bash
tar -xzf markdown-vector-mcp-*.tar.gz
chmod +x markdown-vector-mcp-*
sudo mv markdown-vector-mcp-* /usr/local/bin/markdown-vector-mcp
```

**Windowsの場合:**
- zipファイルを解凍
- 任意のフォルダに配置（例: `C:\Program Files\markdown-vector-mcp\`）

## Quick Start

### 1. 初回起動

```bash
./markdown-vector-mcp
```

### 2. マークダウンファイルを配置

```bash
cp your-notes.md documents/
```

起動時に自動的にインデックス化されます。

### 3. Claude Code設定

`~/.claude.json or .mcp.json (macOS):
```json
{
  "mcpServers": {
    "markdown-vector": {
      "type": "stdio",
      "command": "/absolute/path/to/markdown-vector-mcp"
    }
  }
}
```

### 4. 検索

Claude Code等で：
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

### 設定項目

- `documents_dir`: マークダウンファイルを配置するディレクトリ
- `db_path`: ベクトルデータベースのパス
- `chunk_size`: ドキュメントを分割するチャンクサイズ（文字数）
- `search_top_k`: 検索結果の返却件数
- `compute.device`: 計算デバイス（`auto`, `cpu`, `gpu`）
- `compute.fallback_to_cpu`: GPU利用不可時にCPUにフォールバック
- `model.name`: 埋め込みモデル名
- `model.dimensions`: ベクトル次元数

## MCP Tools

### search
自然言語によるベクトル検索

**パラメータ:**
- `query` (string): 検索クエリ

**戻り値:**
検索結果の配列（各結果にファイル名、チャンク内容、スコアを含む）

### index_markdown
マークダウンファイルをインデックス化

**パラメータ:**
- `filepath` (string): インデックス化するファイルのパス

### list_documents
インデックス化されたドキュメントの一覧を取得

**戻り値:**
ドキュメント一覧（ファイル名と更新日時）

### delete_document
ドキュメントをインデックスから削除

**パラメータ:**
- `filepath` (string): 削除するファイルのパス

### reindex_document
ドキュメントを再インデックス化

**パラメータ:**
- `filepath` (string): 再インデックス化するファイルのパス

## Development

### Run Tests

```bash
# すべてのテストを実行
go test ./...

# 特定のパッケージのテスト
go test ./internal/config -v
go test ./internal/indexer -v
go test ./internal/embedder -v
go test ./internal/vectordb -v

# 統合テスト
go test . -v -run TestEndToEnd
```

### Build

```bash
# ビルドスクリプトを使用
./build.sh

# または直接ビルド
go build -o markdown-vector-mcp cmd/main.go

# リリース用ビルド（クロスプラットフォーム）
./scripts/build-release.sh
```

### Creating a Release

リリースを作成するには、バージョンタグをプッシュします：

```bash
# バージョンタグを作成（例: v1.0.1）
git tag v1.0.1

# タグをプッシュ
git push origin v1.0.1
```

GitHub Actionsが自動的に：
1. 全プラットフォーム向けにビルド
2. GitHub Releasesページを作成
3. バイナリをアップロード
4. チェックサムファイルを生成

リリースが完成したら[Releases](https://github.com/towada/markdown-vector-mcp/releases)ページで確認できます。

### Project Structure

```
markdown-vector-mcp/
├── cmd/
│   └── main.go              # エントリーポイント
├── internal/
│   ├── config/              # 設定管理
│   ├── embedder/            # ベクトル埋め込み
│   ├── indexer/             # インデックス処理
│   ├── mcp/                 # MCPサーバー実装
│   └── vectordb/            # ベクトルDB操作
├── models/                  # ONNXモデル
├── build.sh                 # ビルドスクリプト
└── integration_test.go      # 統合テスト
```

## Troubleshooting

### モデルのダウンロードに失敗する

**原因**: インターネット接続の問題、Hugging Faceサーバーの問題

**解決方法**:
1. インターネット接続を確認
2. プロキシ環境の場合、環境変数を設定：
   ```bash
   export HTTP_PROXY=http://your-proxy:port
   export HTTPS_PROXY=http://your-proxy:port
   ```
3. 手動でダウンロード（`models/DOWNLOAD.md`参照）
4. ダウンロードを再試行（不完全なファイルは削除されます）

### GPU検出されない

`config.json`で`"device": "cpu"`を明示的に指定してください。

```json
{
  "compute": {
    "device": "cpu",
    "fallback_to_cpu": true
  }
}
```

### 起動しない

- Go 1.21以上がインストールされているか確認
- CGOが有効になっているか確認（`go env CGO_ENABLED`で確認）
- 依存ライブラリが正しくインストールされているか確認
- 初回起動時はモデルダウンロードのためインターネット接続が必要

### 検索結果が期待と異なる

- `chunk_size`を調整してみる（デフォルト: 500）
- インデックスを再構築する（ファイルを削除して再起動）

### メモリ使用量が多い

- GPU使用時はモデルがVRAMにロードされます
- CPU使用に切り替えると省メモリになります

## Performance

- 起動時間: 約2-3秒（モデルロード含む）
- インデックス速度: 約100-200 chunks/秒
- 検索レスポンス: 100ms以下（1000ドキュメント）
- メモリ使用量: 約200-400MB
- バイナリサイズ: 約7-8MB

## Requirements

- Go 1.21以上（ビルド時）
- CGO enabled（sqlite-vecのため）
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

## Author

towada
