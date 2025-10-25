# devrag 設計書

## 1. プロジェクト概要

### 1.1 目的
マークダウンファイルをベクトル検索可能にするMCPサーバー。
Claude CodeやChatGPTから自然言語でマークダウンを検索できるようにする。

### 1.2 主要機能
- マークダウンファイルの自動インデックス化
- ベクトル検索による意味的な検索
- ファイルの差分検出と自動同期
- ワンバイナリー配布（スクリプト不要）
- クロスプラットフォーム対応（Windows/macOS/Linux）
- GPU/CPU自動検出

---

## 2. アーキテクチャ

### 2.1 全体構成

```
┌─────────────────┐
│ Claude Code /   │
│ ChatGPT         │
└────────┬────────┘
         │ MCP Protocol (stdio)
         │
┌────────▼────────────────────────────┐
│   devrag (Go Binary)   │
│  ┌──────────────────────────────┐   │
│  │ MCP Server                   │   │
│  │ - search                     │   │
│  │ - index_markdown             │   │
│  │ - list_documents             │   │
│  │ - delete_document            │   │
│  │ - reindex_document           │   │
│  └──────────┬───────────────────┘   │
│             │                        │
│  ┌──────────▼───────────────────┐   │
│  │ Embedder (ONNX Runtime)      │   │
│  │ - Text → Vector[384]         │   │
│  │ - GPU/CPU Auto Detection     │   │
│  └──────────┬───────────────────┘   │
│             │                        │
│  ┌──────────▼───────────────────┐   │
│  │ Vector DB (SQLite + vec0)    │   │
│  │ - Document storage           │   │
│  │ - Vector similarity search   │   │
│  └──────────────────────────────┘   │
└─────────────────────────────────────┘
```

### 2.2 起動シーケンス

```
1. バイナリ起動
   ↓
2. config.json 読み込み (なければデフォルト値)
   ↓
3. GPU/CPU 検出・選択
   ↓
4. ONNX モデル初期化
   ↓
5. SQLite + vec0 初期化
   ↓
6. ./documents/ ディレクトリスキャン
   ↓
7. ファイル差分チェック（新規/更新/削除）
   ↓
8. 差分があれば自動同期
   ↓
9. MCP サーバー起動 (stdio)
```

---

## 3. ディレクトリ構成

### 3.1 配置構成

```
devrag              # 実行バイナリ
├── config.json                  # 設定ファイル（オプション）
├── documents/                   # マークダウン配置ディレクトリ
│   ├── note1.md
│   ├── project.md
│   └── api.md
└── vectors.db                   # SQLiteファイル（自動生成）
```

### 3.2 プロジェクト構成

```
devrag/
├── cmd/
│   └── main.go                  # エントリーポイント
├── internal/
│   ├── config/
│   │   └── config.go            # 設定読み込み
│   ├── mcp/
│   │   ├── server.go            # MCPサーバー実装
│   │   └── tools.go             # ツール定義
│   ├── embedder/
│   │   ├── embedder.go          # 埋め込みインターフェース
│   │   ├── onnx.go              # ONNX Runtime実装
│   │   └── device.go            # GPU/CPU検出
│   ├── vectordb/
│   │   ├── db.go                # DB操作インターフェース
│   │   ├── sqlite.go            # SQLite + vec0実装
│   │   └── schema.go            # スキーマ定義
│   └── indexer/
│       ├── indexer.go           # インデックス化処理
│       ├── markdown.go          # マークダウン解析
│       └── sync.go              # 差分同期
├── models/
│   └── model.onnx               # 埋め込みモデル（バイナリに埋め込み）
├── go.mod
├── go.sum
├── DESIGN.md                    # 本ファイル
├── PLAN.md                      # 作業計画書
└── README.md
```

---

## 4. 設定ファイル仕様

### 4.1 config.json

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

### 4.2 パラメータ説明

| パラメータ | 型 | デフォルト | 説明 |
|-----------|-----|-----------|------|
| `documents_dir` | string | `"./documents"` | マークダウン配置ディレクトリ |
| `db_path` | string | `"./vectors.db"` | SQLiteファイルパス |
| `chunk_size` | int | `500` | テキスト分割サイズ（文字数） |
| `search_top_k` | int | `5` | 検索結果の最大件数 |
| `compute.device` | string | `"auto"` | `"auto"`, `"gpu"`, `"cpu"` |
| `compute.fallback_to_cpu` | bool | `true` | GPU使用不可時にCPUフォールバック |
| `model.name` | string | `"multilingual-e5-small"` | モデル名（情報表示用） |
| `model.dimensions` | int | `384` | ベクトル次元数 |

### 4.3 設定読み込み挙動

1. `config.json` が存在しない → デフォルト値で起動 & 雛形生成
2. `config.json` が不正 → 警告表示 & デフォルト値で起動
3. `config.json` が正常 → 設定値で起動

---

## 5. MCPツール仕様

### 5.1 search

```json
{
  "name": "search",
  "description": "自然言語クエリでマークダウンをベクトル検索",
  "inputSchema": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "検索クエリ（自然言語）"
      },
      "top_k": {
        "type": "integer",
        "description": "検索結果の最大件数（デフォルト: 5）",
        "default": 5
      }
    },
    "required": ["query"]
  }
}
```

**戻り値例：**
```json
{
  "results": [
    {
      "document": "project.md",
      "chunk": "JWTトークンを使った認証の実装方法...",
      "similarity": 0.89,
      "position": 123
    }
  ]
}
```

### 5.2 index_markdown

```json
{
  "name": "index_markdown",
  "description": "指定したマークダウンファイルをインデックス化",
  "inputSchema": {
    "type": "object",
    "properties": {
      "filepath": {
        "type": "string",
        "description": "マークダウンファイルのパス"
      },
      "chunk_size": {
        "type": "integer",
        "description": "チャンクサイズ（文字数）",
        "default": 500
      }
    },
    "required": ["filepath"]
  }
}
```

### 5.3 list_documents

```json
{
  "name": "list_documents",
  "description": "インデックス済みドキュメント一覧を取得",
  "inputSchema": {
    "type": "object",
    "properties": {}
  }
}
```

**戻り値例：**
```json
{
  "documents": [
    {
      "filename": "project.md",
      "chunks": 45,
      "indexed_at": "2025-10-24T15:30:00Z",
      "modified_at": "2025-10-24T12:00:00Z"
    }
  ]
}
```

### 5.4 delete_document

```json
{
  "name": "delete_document",
  "description": "ドキュメントをDBとファイルシステムの両方から削除",
  "inputSchema": {
    "type": "object",
    "properties": {
      "filename": {
        "type": "string",
        "description": "削除するファイル名"
      }
    },
    "required": ["filename"]
  }
}
```

### 5.5 reindex_document

```json
{
  "name": "reindex_document",
  "description": "ドキュメントを削除して再インデックス化",
  "inputSchema": {
    "type": "object",
    "properties": {
      "filename": {
        "type": "string",
        "description": "再インデックス化するファイル名"
      }
    },
    "required": ["filename"]
  }
}
```

---

## 6. データベーススキーマ

### 6.1 documents テーブル

```sql
CREATE TABLE documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename TEXT NOT NULL UNIQUE,
    indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME NOT NULL
);

CREATE INDEX idx_filename ON documents(filename);
```

### 6.2 chunks テーブル

```sql
CREATE TABLE chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    document_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    content TEXT NOT NULL,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE INDEX idx_document_id ON chunks(document_id);
```

### 6.3 vec_chunks 仮想テーブル (sqlite-vec)

```sql
CREATE VIRTUAL TABLE vec_chunks USING vec0(
    chunk_id INTEGER PRIMARY KEY,
    embedding FLOAT[384]
);
```

---

## 7. 技術スタック

### 7.1 言語・フレームワーク

| 技術 | バージョン | 用途 |
|-----|-----------|------|
| Go | 1.21+ | メイン言語 |
| mcp-go | latest | MCP Protocol実装 |
| mattn/go-sqlite3 | latest | SQLiteドライバ |
| sqlite-vec | latest | ベクトル検索拡張 |
| onnxruntime_go | latest | ONNX推論 |

### 7.2 埋め込みモデル

**推奨：intfloat/multilingual-e5-small**
- サイズ: ~120MB (量子化版: ~30MB)
- 次元: 384
- 特徴: 多言語対応、CPU最適化
- ライセンス: MIT

**代替案：**
- pkshatech/GLuCoSE-base-ja (日本語特化、768次元)
- paraphrase-multilingual-MiniLM-L12-v2 (バランス型)

### 7.3 ビルドツール

```bash
# シングルバイナリー化
go build -tags netgo -ldflags="-s -w" -o devrag

# クロスコンパイル
GOOS=windows GOARCH=amd64 go build
GOOS=darwin GOARCH=arm64 go build
GOOS=linux GOARCH=amd64 go build
```

---

## 8. クロスプラットフォーム対応

### 8.1 パス処理

```go
import "path/filepath"

// OSに依存しないパス処理
documentsDir := filepath.FromSlash(config.DocumentsDir)

// ホームディレクトリ展開
if strings.HasPrefix(documentsDir, "~/") {
    home, _ := os.UserHomeDir()
    documentsDir = filepath.Join(home, documentsDir[2:])
}
```

### 8.2 改行コード

```go
// 自動変換（bufio.Scanner使用）
scanner := bufio.NewScanner(file)
for scanner.Scan() {
    line := scanner.Text() // 改行コードは自動除去
}
```

### 8.3 ファイル更新検出

```go
// os.Stat で mtime 取得（全OS対応）
info, err := os.Stat(filepath)
modTime := info.ModTime()
```

---

## 9. GPU/CPU自動検出

### 9.1 検出ロジック

```go
type Device int

const (
    CPU Device = iota
    GPU
)

func detectDevice(config Config) Device {
    switch config.Compute.Device {
    case "auto":
        if hasGPU() {
            log.Println("[INFO] GPU detected, using GPU")
            return GPU
        }
        log.Println("[INFO] No GPU, using CPU")
        return CPU

    case "gpu":
        if hasGPU() {
            return GPU
        }
        if config.Compute.FallbackToCPU {
            log.Println("[WARN] GPU requested but unavailable, fallback to CPU")
            return CPU
        }
        log.Fatal("[ERROR] GPU required but unavailable")

    case "cpu":
        return CPU
    }
}
```

### 9.2 プラットフォーム別GPU検出

| OS | GPU | 検出方法 |
|----|-----|---------|
| macOS | Metal | `syscall.Sysctl("hw.optional.arm64")` or ONNX provider check |
| Windows | CUDA | `nvidia-smi` 実行 or CUDA Runtime check |
| Linux | CUDA/ROCm | `nvidia-smi` / `rocm-smi` or device file check |

### 9.3 ONNX Runtime設定

```go
func initONNX(device Device) (*ort.Session, error) {
    options := ort.NewSessionOptions()

    switch device {
    case GPU:
        // macOS: CoreML or Metal
        // Windows/Linux: CUDA
        if runtime.GOOS == "darwin" {
            options.AppendExecutionProvider("CoreML", nil)
        } else {
            options.AppendExecutionProvider("CUDA", nil)
        }
    case CPU:
        // CPUプロバイダ（デフォルト）
    }

    return ort.NewSession(modelData, options)
}
```

---

## 10. セキュリティ考慮事項

### 10.1 パストラバーサル対策

```go
func validatePath(filepath string, baseDir string) error {
    absPath, err := filepath.Abs(filepath)
    if err != nil {
        return err
    }

    absBase, err := filepath.Abs(baseDir)
    if err != nil {
        return err
    }

    if !strings.HasPrefix(absPath, absBase) {
        return fmt.Errorf("path traversal detected")
    }

    return nil
}
```

### 10.2 SQLインジェクション対策

```go
// プリペアドステートメント必須
stmt, err := db.Prepare("SELECT * FROM documents WHERE filename = ?")
defer stmt.Close()
```

---

## 11. エラーハンドリング

### 11.1 ログ出力先

- **stdout**: MCPプロトコル通信専用
- **stderr**: すべてのログ出力

### 11.2 エラーレベル

```
[INFO]  : 通常動作ログ
[WARN]  : 警告（処理継続）
[ERROR] : エラー（処理中断）
[FATAL] : 致命的エラー（プロセス終了）
```

---

## 12. パフォーマンス目標

| 項目 | 目標値 |
|-----|-------|
| 起動時間 | < 3秒 |
| 検索レスポンス | < 500ms (1000件/DB) |
| インデックス速度 | > 100 chunks/sec |
| メモリ使用量 | < 500MB (モデル込み) |
| バイナリサイズ | < 200MB (モデル込み) |

---

## 13. 将来の拡張性

### 13.1 Phase 2候補機能

- メタデータフィルタリング（タグ、日付範囲など）
- ハイブリッド検索（ベクトル + FTS5キーワード検索）
- ファイル監視（fsnotify）による自動リアルタイム同期
- Web UI（オプション）

### 13.2 Phase 3候補機能

- 複数ディレクトリ監視
- Notion/Obsidian連携
- カスタムモデル対応
- クラウドストレージ連携（S3, GCS）
