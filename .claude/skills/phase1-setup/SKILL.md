---
name: phase1-setup
description: Phase 1基盤構築を実装。Goプロジェクト初期化、設定ファイルモジュール、SQLite+vec0セットアップ、ONNX Runtime統合、埋め込みモデル準備を行う。Phase 1開始時やプロジェクトセットアップ時に使用。
---

# Phase 1: 基盤構築

devragプロジェクトの基礎インフラを構築します。

## タスク一覧

### 1.1 Goプロジェクト初期化

**実行コマンド**:
```bash
go mod init github.com/towada/devrag
```

**ディレクトリ構成作成**:
```
devrag/
├── cmd/
│   └── main.go
├── internal/
│   ├── config/
│   ├── mcp/
│   ├── embedder/
│   ├── vectordb/
│   └── indexer/
├── models/
├── go.mod
└── .gitignore
```

**.gitignore**:
```
# Binaries
devrag
*.exe

# Build output
bin/
dist/

# Runtime files
config.json
vectors.db
documents/

# Go
*.so
*.dylib

# Test
*.test
*.out

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db
```

### 1.2 設定ファイルモジュール実装

**ファイル**: `internal/config/config.go`

```go
package config

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
)

type Config struct {
    DocumentsDir string `json:"documents_dir"`
    DBPath       string `json:"db_path"`
    ChunkSize    int    `json:"chunk_size"`
    SearchTopK   int    `json:"search_top_k"`
    Compute      struct {
        Device         string `json:"device"`
        FallbackToCPU  bool   `json:"fallback_to_cpu"`
    } `json:"compute"`
    Model        struct {
        Name       string `json:"name"`
        Dimensions int    `json:"dimensions"`
    } `json:"model"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
    cfg := &Config{
        DocumentsDir: "./documents",
        DBPath:       "./vectors.db",
        ChunkSize:    500,
        SearchTopK:   5,
    }
    cfg.Compute.Device = "auto"
    cfg.Compute.FallbackToCPU = true
    cfg.Model.Name = "multilingual-e5-small"
    cfg.Model.Dimensions = 384
    return cfg
}

// Load reads config from file or returns default
func Load() (*Config, error) {
    const configFile = "config.json"

    // Check if config.json exists
    if _, err := os.Stat(configFile); os.IsNotExist(err) {
        fmt.Fprintf(os.Stderr, "[INFO] config.json not found, using defaults\n")
        cfg := DefaultConfig()

        // Generate template
        if err := cfg.Save(configFile); err != nil {
            fmt.Fprintf(os.Stderr, "[WARN] Failed to generate config template: %v\n", err)
        } else {
            fmt.Fprintf(os.Stderr, "[INFO] Generated config template: %s\n", configFile)
        }

        return cfg, nil
    }

    // Read existing config
    data, err := os.ReadFile(configFile)
    if err != nil {
        return nil, fmt.Errorf("failed to read config: %w", err)
    }

    cfg := DefaultConfig()
    if err := json.Unmarshal(data, cfg); err != nil {
        fmt.Fprintf(os.Stderr, "[WARN] Invalid JSON in config.json: %v\n", err)
        fmt.Fprintf(os.Stderr, "[WARN] Using default configuration\n")
        return cfg, nil
    }

    fmt.Fprintf(os.Stderr, "[INFO] Loaded configuration from %s\n", configFile)
    return cfg, nil
}

// Save writes config to file
func (c *Config) Save(path string) error {
    data, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }

    if err := os.WriteFile(path, data, 0644); err != nil {
        return fmt.Errorf("failed to write config: %w", err)
    }

    return nil
}

// Validate checks config values
func (c *Config) Validate() error {
    if c.ChunkSize <= 0 {
        return fmt.Errorf("chunk_size must be positive")
    }
    if c.SearchTopK <= 0 {
        return fmt.Errorf("search_top_k must be positive")
    }
    if c.Model.Dimensions <= 0 {
        return fmt.Errorf("model.dimensions must be positive")
    }
    return nil
}
```

### 1.3 SQLite + vec0 セットアップ

**依存追加**:
```bash
go get github.com/mattn/go-sqlite3
```

**ファイル**: `internal/vectordb/schema.go`

```go
package vectordb

const schemaSQL = `
CREATE TABLE IF NOT EXISTS documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename TEXT NOT NULL UNIQUE,
    indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_filename ON documents(filename);

CREATE TABLE IF NOT EXISTS chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    document_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    content TEXT NOT NULL,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_document_id ON chunks(document_id);
`

// Note: vec0 virtual table will be created separately
// CREATE VIRTUAL TABLE vec_chunks USING vec0(
//     chunk_id INTEGER PRIMARY KEY,
//     embedding FLOAT[384]
// );
```

**ファイル**: `internal/vectordb/sqlite.go`

```go
package vectordb

import (
    "database/sql"
    "fmt"
    "os"

    _ "github.com/mattn/go-sqlite3"
)

type DB struct {
    conn *sql.DB
}

// Init initializes the SQLite database
func Init(dbPath string) (*DB, error) {
    fmt.Fprintf(os.Stderr, "[INFO] Initializing database: %s\n", dbPath)

    conn, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // Enable foreign keys
    if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
        conn.Close()
        return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
    }

    // Create tables
    if _, err := conn.Exec(schemaSQL); err != nil {
        conn.Close()
        return nil, fmt.Errorf("failed to create schema: %w", err)
    }

    fmt.Fprintf(os.Stderr, "[INFO] Database initialized successfully\n")

    return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
    return db.conn.Close()
}
```

**技術調査項目**:
- sqlite-vec拡張のロード方法（CGo経由）
- vec0仮想テーブルの作成方法
- クロスコンパイル時のCGo設定

### 1.4 ONNX Runtime統合

**依存追加**:
```bash
go get github.com/yalue/onnxruntime_go
```

**ファイル**: `internal/embedder/device.go`

```go
package embedder

import (
    "fmt"
    "os"
    "runtime"
)

type Device int

const (
    CPU Device = iota
    GPU
)

func (d Device) String() string {
    if d == GPU {
        return "GPU"
    }
    return "CPU"
}

// DetectDevice detects the best available device
func DetectDevice(deviceConfig string, fallbackToCPU bool) Device {
    switch deviceConfig {
    case "auto":
        if hasGPU() {
            fmt.Fprintf(os.Stderr, "[INFO] GPU detected, using GPU\n")
            return GPU
        }
        fmt.Fprintf(os.Stderr, "[INFO] No GPU detected, using CPU\n")
        return CPU

    case "gpu":
        if hasGPU() {
            fmt.Fprintf(os.Stderr, "[INFO] Using GPU as requested\n")
            return GPU
        }
        if fallbackToCPU {
            fmt.Fprintf(os.Stderr, "[WARN] GPU requested but unavailable, falling back to CPU\n")
            return CPU
        }
        fmt.Fprintf(os.Stderr, "[ERROR] GPU required but unavailable\n")
        os.Exit(1)

    case "cpu":
        fmt.Fprintf(os.Stderr, "[INFO] Using CPU as requested\n")
        return CPU

    default:
        fmt.Fprintf(os.Stderr, "[WARN] Unknown device '%s', using CPU\n", deviceConfig)
        return CPU
    }

    return CPU
}

// hasGPU checks if GPU is available
func hasGPU() bool {
    // Platform-specific GPU detection
    switch runtime.GOOS {
    case "darwin":
        // macOS: Check for Metal support (M1/M2)
        return runtime.GOARCH == "arm64"
    case "windows", "linux":
        // Check for NVIDIA GPU (simplified)
        // In production, use nvidia-smi or CUDA runtime check
        return false // Default to CPU for now
    }
    return false
}
```

**ファイル**: `internal/embedder/embedder.go`

```go
package embedder

type Embedder interface {
    Embed(text string) ([]float32, error)
    EmbedBatch(texts []string) ([][]float32, error)
    Close() error
}
```

**ファイル**: `internal/embedder/onnx.go`

```go
package embedder

import (
    "fmt"
    "os"

    ort "github.com/yalue/onnxruntime_go"
)

type ONNXEmbedder struct {
    session *ort.AdvancedSession
    device  Device
}

// NewONNXEmbedder creates a new ONNX embedder
func NewONNXEmbedder(modelPath string, device Device) (*ONNXEmbedder, error) {
    fmt.Fprintf(os.Stderr, "[INFO] Initializing ONNX Runtime (%s)...\n", device)

    // Initialize ONNX Runtime
    if err := ort.InitializeEnvironment(); err != nil {
        return nil, fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
    }

    // Create session options
    options, err := ort.NewSessionOptions()
    if err != nil {
        return nil, fmt.Errorf("failed to create session options: %w", err)
    }
    defer options.Destroy()

    // Set execution provider based on device
    // Note: This is simplified. In production, configure properly based on platform

    // Load model
    session, err := ort.NewAdvancedSession(modelPath, []string{"input_ids"}, []string{"embeddings"}, options)
    if err != nil {
        return nil, fmt.Errorf("failed to load model: %w", err)
    }

    fmt.Fprintf(os.Stderr, "[INFO] ONNX model loaded successfully\n")

    return &ONNXEmbedder{
        session: session,
        device:  device,
    }, nil
}

// Embed embeds a single text
func (e *ONNXEmbedder) Embed(text string) ([]float32, error) {
    // TODO: Implement tokenization and inference
    return nil, fmt.Errorf("not implemented yet")
}

// EmbedBatch embeds multiple texts
func (e *ONNXEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
    // TODO: Implement batch processing
    return nil, fmt.Errorf("not implemented yet")
}

// Close closes the embedder
func (e *ONNXEmbedder) Close() error {
    if e.session != nil {
        e.session.Destroy()
    }
    return nil
}
```

### 1.5 埋め込みモデル準備

**タスク**:
1. multilingual-e5-smallモデルのダウンロード
2. ONNX形式への変換（必要に応じて）
3. models/model.onnxへの配置

**Hugging Faceからのダウンロード**:
```python
# download_model.py
from transformers import AutoModel, AutoTokenizer
from optimum.onnxruntime import ORTModelForFeatureExtraction

model_name = "intfloat/multilingual-e5-small"

# Download and convert to ONNX
model = ORTModelForFeatureExtraction.from_pretrained(
    model_name,
    export=True
)

# Save
model.save_pretrained("models/")
```

**go:embed設定** (future):
```go
package embedder

import _ "embed"

//go:embed model.onnx
var modelData []byte
```

## Phase 1 完了条件

- [ ] `go build cmd/main.go` が成功する
- [ ] config.jsonが生成され、読み込める
- [ ] SQLiteデータベースが初期化できる
- [ ] ONNXモデルがロードできる（modelファイル配置後）
- [ ] GPU/CPU検出が動作する

## 注意事項

### ログ出力
- すべてのログは`os.Stderr`に出力
- `os.Stdout`はMCPプロトコル専用

### エラーレベル
- `[INFO]`: 通常動作
- `[WARN]`: 警告（処理継続）
- `[ERROR]`: エラー（処理中断）
- `[FATAL]`: 致命的エラー（プロセス終了）

### パス処理
- `filepath`パッケージを使用してOS非依存に実装
- ホームディレクトリ展開対応

### セキュリティ
- パストラバーサル対策を実装
- プリペアドステートメント使用

## 次のステップ

Phase 1完了後は **phase2-core** スキルを使用してコア機能を実装します。
