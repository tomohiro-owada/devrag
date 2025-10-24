# markdown-vector-mcp 作業計画書

## 1. プロジェクト概要

**目的**: マークダウンファイルのベクトル検索を提供するMCPサーバーをワンバイナリーで実装

**開発期間**: 約2-3週間（1人）

**成果物**:
- クロスプラットフォーム対応バイナリー（Windows/macOS/Linux）
- 設定ファイルテンプレート
- README・ドキュメント

---

## 2. 開発フェーズ

### Phase 1: 基盤構築 (3-4日)
プロジェクトセットアップと基本インフラの構築

### Phase 2: コア機能実装 (5-7日)
ベクトル化・検索機能の実装

### Phase 3: MCP統合・完成 (3-4日)
MCPプロトコル実装と全体統合

### Phase 4: テスト・ビルド (2-3日)
動作確認とクロスプラットフォームビルド

---

## 3. Phase 1: 基盤構築

### 3.1 プロジェクトセットアップ

#### タスク 1.1: Goプロジェクト初期化
- [ ] `go mod init`
- [ ] 基本ディレクトリ構成作成
- [ ] `.gitignore` 作成
- [ ] README.md 骨子作成

**成果物**:
```
markdown-vector-mcp/
├── cmd/main.go
├── internal/
├── go.mod
├── .gitignore
└── README.md
```

**所要時間**: 1時間

---

#### タスク 1.2: 設定ファイルモジュール実装

**ファイル**: `internal/config/config.go`

**実装内容**:
- [ ] Config 構造体定義
- [ ] デフォルト値設定
- [ ] JSON読み込み処理
- [ ] エラーハンドリング（不正JSON対応）
- [ ] 初回起動時の雛形生成

**テストケース**:
- config.json が存在しない
- config.json が不正なJSON
- config.json が正常

**所要時間**: 3-4時間

---

#### タスク 1.3: SQLite + vec0 セットアップ

**ファイル**: `internal/vectordb/schema.go`, `internal/vectordb/sqlite.go`

**実装内容**:
- [ ] mattn/go-sqlite3 依存追加
- [ ] sqlite-vec 拡張ロード
- [ ] スキーマ定義（documents, chunks, vec_chunks）
- [ ] DB初期化処理
- [ ] マイグレーション処理

**技術調査**:
- [ ] sqlite-vec のGo統合方法
- [ ] CGo設定要否
- [ ] クロスコンパイル時の注意点

**所要時間**: 6-8時間

**依存**: なし

---

#### タスク 1.4: ONNX Runtime統合

**ファイル**: `internal/embedder/device.go`, `internal/embedder/onnx.go`

**実装内容**:
- [ ] onnxruntime_go 依存追加
- [ ] GPU/CPU検出ロジック実装
  - macOS: Metal/CoreML検出
  - Windows: CUDA検出
  - Linux: CUDA/ROCm検出
- [ ] ONNXランタイム初期化
- [ ] モデルファイル埋め込み準備

**技術調査**:
- [ ] ONNX Runtimeの各OS対応状況
- [ ] Execution Provider設定方法
- [ ] モデル埋め込み方法（go:embed）

**所要時間**: 8-10時間

**依存**: なし

---

#### タスク 1.5: モデル準備

**実装内容**:
- [ ] multilingual-e5-small モデルダウンロード
- [ ] ONNX形式に変換（必要に応じて）
- [ ] 量子化検討（サイズ削減）
- [ ] `models/model.onnx` 配置
- [ ] go:embed 設定

**技術調査**:
- [ ] Hugging FaceからのONNX Export方法
- [ ] 量子化ツール（optimum等）

**所要時間**: 4-6時間

**依存**: なし

---

### Phase 1 完了条件

- [ ] プロジェクトがビルド可能
- [ ] 設定ファイルが読み込める
- [ ] SQLiteが初期化できる
- [ ] ONNXモデルがロードできる
- [ ] GPU/CPU検出が動作する

---

## 4. Phase 2: コア機能実装

### 4.1 マークダウン処理

#### タスク 2.1: マークダウンパーサー実装

**ファイル**: `internal/indexer/markdown.go`

**実装内容**:
- [ ] マークダウンファイル読み込み
- [ ] チャンク分割ロジック実装
  - 文字数ベース（デフォルト500文字）
  - 段落境界を考慮
  - コードブロックを分割しない
- [ ] メタデータ抽出（ファイル名、更新日時等）

**テストケース**:
- 短いファイル（<500文字）
- 長いファイル（>5000文字）
- コードブロック含む
- 日本語・英語混在

**所要時間**: 6-8時間

**依存**: なし

---

#### タスク 2.2: ベクトル化処理実装

**ファイル**: `internal/embedder/embedder.go`

**実装内容**:
- [ ] Embedder インターフェース定義
- [ ] ONNX推論実行
- [ ] トークナイザー実装（必要に応じて）
- [ ] バッチ処理対応
- [ ] エラーハンドリング

**技術調査**:
- [ ] multilingual-e5 の入力形式
- [ ] トークナイザー処理（サブワード分割等）
- [ ] 正規化処理要否

**所要時間**: 8-10時間

**依存**: タスク 1.4, 1.5

---

### 4.2 インデックス化機能

#### タスク 2.3: インデックス化処理実装

**ファイル**: `internal/indexer/indexer.go`

**実装内容**:
- [ ] ファイル → チャンク → ベクトル → DB のパイプライン
- [ ] トランザクション処理
- [ ] 進捗表示（stderr）
- [ ] エラーリカバリー

**処理フロー**:
```
1. マークダウン読み込み
2. チャンク分割
3. 各チャンクをベクトル化
4. DBに挿入（documents, chunks, vec_chunks）
```

**所要時間**: 6-8時間

**依存**: タスク 2.1, 2.2

---

#### タスク 2.4: 差分同期機能実装

**ファイル**: `internal/indexer/sync.go`

**実装内容**:
- [ ] ディレクトリスキャン（*.md）
- [ ] ファイル mtime とDB比較
- [ ] 新規ファイル検出 → インデックス化
- [ ] 更新ファイル検出 → 再インデックス化
- [ ] 削除ファイル検出 → DB削除
- [ ] 起動時自動実行

**テストケース**:
- 空ディレクトリ（初回起動）
- ファイル追加
- ファイル更新
- ファイル削除

**所要時間**: 6-8時間

**依存**: タスク 2.3

---

### 4.3 検索機能

#### タスク 2.5: ベクトル検索実装

**ファイル**: `internal/vectordb/db.go`

**実装内容**:
- [ ] クエリテキスト → ベクトル化
- [ ] sqlite-vec でコサイン類似度検索
- [ ] Top-K結果取得
- [ ] 結果のソート・整形

**SQL例**:
```sql
SELECT
    c.id, c.content, d.filename,
    vec_distance_cosine(v.embedding, ?) as similarity
FROM vec_chunks v
JOIN chunks c ON v.chunk_id = c.id
JOIN documents d ON c.document_id = d.id
ORDER BY similarity DESC
LIMIT ?
```

**所要時間**: 6-8時間

**依存**: タスク 2.2

---

### Phase 2 完了条件

- [ ] マークダウンファイルがインデックス化できる
- [ ] 自然言語クエリで検索できる
- [ ] 差分同期が正しく動作する
- [ ] 日本語検索が機能する

---

## 5. Phase 3: MCP統合・完成

### 5.1 MCPサーバー実装

#### タスク 3.1: MCP依存追加とセットアップ

**実装内容**:
- [ ] mcp-go 依存追加
- [ ] MCPサーバー初期化処理
- [ ] stdio 通信設定

**技術調査**:
- [ ] mcp-go の使用方法
- [ ] stdio プロトコルの仕様確認

**所要時間**: 3-4時間

**依存**: なし

---

#### タスク 3.2: MCPツール実装 - search

**ファイル**: `internal/mcp/tools.go`

**実装内容**:
- [ ] Tool定義（JSON Schema）
- [ ] ハンドラー実装
- [ ] 入力バリデーション
- [ ] エラーレスポンス

**所要時間**: 3-4時間

**依存**: タスク 2.5, 3.1

---

#### タスク 3.3: MCPツール実装 - index_markdown

**実装内容**:
- [ ] Tool定義
- [ ] ファイルパス検証
- [ ] インデックス化呼び出し
- [ ] 成功/失敗レスポンス

**所要時間**: 2-3時間

**依存**: タスク 2.3, 3.1

---

#### タスク 3.4: MCPツール実装 - list_documents

**実装内容**:
- [ ] Tool定義
- [ ] DB クエリ実装
- [ ] レスポンス整形

**所要時間**: 2時間

**依存**: タスク 3.1

---

#### タスク 3.5: MCPツール実装 - delete_document

**実装内容**:
- [ ] Tool定義
- [ ] DB削除処理
- [ ] ファイル削除処理
- [ ] トランザクション処理

**所要時間**: 2-3時間

**依存**: タスク 3.1

---

#### タスク 3.6: MCPツール実装 - reindex_document

**実装内容**:
- [ ] Tool定義
- [ ] 削除 + 再インデックス化
- [ ] エラーハンドリング

**所要時間**: 2時間

**依存**: タスク 3.3, 3.5

---

### 5.2 メインループ実装

#### タスク 3.7: main.go 統合

**ファイル**: `cmd/main.go`

**実装内容**:
```go
func main() {
    // 1. 設定読み込み
    config := config.Load()

    // 2. デバイス検出
    device := embedder.DetectDevice(config)

    // 3. コンポーネント初期化
    db := vectordb.Init(config)
    emb := embedder.Init(device, config)

    // 4. 差分同期
    indexer.Sync(config, db, emb)

    // 5. MCPサーバー起動
    mcp.Start(config, db, emb)
}
```

**所要時間**: 4-6時間

**依存**: すべての前タスク

---

### Phase 3 完了条件

- [ ] MCPサーバーが起動する
- [ ] 5つのツールすべてが動作する
- [ ] Claude Codeから呼び出せる
- [ ] エラーが適切にハンドリングされる

---

## 6. Phase 4: テスト・ビルド

### 6.1 テスト実装

#### タスク 4.1: ユニットテスト作成

**対象モジュール**:
- [ ] `config_test.go`
- [ ] `markdown_test.go`
- [ ] `embedder_test.go`
- [ ] `db_test.go`
- [ ] `sync_test.go`

**所要時間**: 8-10時間

---

#### タスク 4.2: 統合テスト作成

**テストシナリオ**:
- [ ] 初回起動 → インデックス化 → 検索
- [ ] ファイル更新 → 再起動 → 差分同期確認
- [ ] ファイル削除 → 再起動 → DB削除確認

**所要時間**: 4-6時間

---

### 6.2 クロスプラットフォームビルド

#### タスク 4.3: ビルドスクリプト作成

**ファイル**: `build.sh`, `build.bat`

**実装内容**:
```bash
#!/bin/bash

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -tags netgo -ldflags="-s -w" \
  -o bin/markdown-vector-mcp-darwin-arm64 cmd/main.go

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -tags netgo -ldflags="-s -w" \
  -o bin/markdown-vector-mcp-darwin-amd64 cmd/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -tags netgo -ldflags="-s -w" \
  -o bin/markdown-vector-mcp-windows-amd64.exe cmd/main.go

# Linux
GOOS=linux GOARCH=amd64 go build -tags netgo -ldflags="-s -w" \
  -o bin/markdown-vector-mcp-linux-amd64 cmd/main.go
```

**所要時間**: 2-3時間

**技術調査**:
- [ ] CGo クロスコンパイルの問題対処
- [ ] 各OS向けONNX Runtimeバイナリ準備

---

#### タスク 4.4: 動作確認（各OS）

**確認項目**:
- [ ] macOS (M1/M2): GPU検出、検索動作
- [ ] macOS (Intel): CPU動作
- [ ] Windows: CUDA検出（オプション）、CPU動作
- [ ] Linux: CPU動作

**所要時間**: 4-6時間

---

### 6.3 ドキュメント整備

#### タスク 4.5: README.md 完成

**内容**:
- [ ] インストール方法
- [ ] Claude Code設定例
- [ ] 使用例
- [ ] トラブルシューティング
- [ ] ライセンス情報

**所要時間**: 3-4時間

---

#### タスク 4.6: リリースノート作成

**ファイル**: `CHANGELOG.md`

**所要時間**: 1時間

---

### Phase 4 完了条件

- [ ] すべてのテストがパス
- [ ] 3つのOS向けバイナリが生成される
- [ ] 各OSで動作確認完了
- [ ] ドキュメントが完備

---

## 7. 技術調査項目（事前確認推奨）

### 優先度: 高

| 項目 | 内容 | 所要時間 |
|-----|------|---------|
| sqlite-vec統合 | Go言語での使用方法、CGo要否 | 2-3時間 |
| ONNX Runtime Go | 各OS対応状況、Execution Provider設定 | 2-3時間 |
| mcp-go | 基本的な使い方、stdio通信 | 2時間 |
| モデル変換 | Hugging Face → ONNX変換手順 | 2時間 |

### 優先度: 中

| 項目 | 内容 | 所要時間 |
|-----|------|---------|
| GPU検出 | 各OS別GPU検出方法 | 3-4時間 |
| クロスコンパイル | CGo使用時の対応方法 | 2-3時間 |
| トークナイザー | multilingual-e5の前処理 | 2時間 |

---

## 8. リスクと対策

### リスク 1: CGo によるクロスコンパイル困難

**影響度**: 高
**対策**:
- pure-Go SQLiteドライバ検討（modernc.org/sqlite）
- ONNX Runtimeの静的リンク検討
- Docker使用したビルド環境整備

---

### リスク 2: ONNX Runtime の各OS対応

**影響度**: 中
**対策**:
- 各OS向けの事前検証
- CPUのみ対応から開始（GPU対応は後回し可）
- 代替ライブラリ検討（tract-onnx など）

---

### リスク 3: モデルサイズによるバイナリ肥大化

**影響度**: 低
**対策**:
- モデル量子化（INT8など）
- 外部ファイル配置オプション追加
- より小さいモデルの検討

---

## 9. マイルストーン

| マイルストーン | 完了日（目安） | 成果物 |
|--------------|-------------|-------|
| M1: Phase 1完了 | Day 4 | 基盤動作確認 |
| M2: Phase 2完了 | Day 11 | 検索機能動作 |
| M3: Phase 3完了 | Day 15 | MCP統合完了 |
| M4: Phase 4完了 | Day 18 | リリース準備完了 |
| **最終リリース** | **Day 21** | **全OS向けバイナリ配布** |

---

## 10. 作業開始時のチェックリスト

### 環境準備

- [ ] Go 1.21+ インストール確認
- [ ] Git セットアップ
- [ ] 各種依存ツール確認
  - [ ] Python（モデル変換用）
  - [ ] ONNX Runtime
  - [ ] SQLite

### 技術調査完了

- [ ] sqlite-vec の動作確認
- [ ] ONNX Runtime Go の動作確認
- [ ] mcp-go のサンプル動作確認
- [ ] モデル変換手順確認

### プロジェクトセットアップ

- [ ] リポジトリ作成
- [ ] ディレクトリ構成作成
- [ ] go.mod 初期化
- [ ] DESIGN.md, PLAN.md 配置

---

## 11. 次のステップ

1. **技術調査**: 優先度高項目を実施（1-2日）
2. **Phase 1 開始**: プロジェクトセットアップから着手
3. **週次レビュー**: 進捗確認と計画調整

---

## 付録A: 依存ライブラリ一覧

```go
module github.com/username/markdown-vector-mcp

go 1.21

require (
    github.com/mark3labs/mcp-go v0.x.x
    github.com/mattn/go-sqlite3 v1.14.x
    github.com/yalue/onnxruntime_go v1.x.x
    // or modernc.org/sqlite v1.x.x (pure-Go)
)
```

---

## 付録B: ディレクトリ構成（最終形）

```
markdown-vector-mcp/
├── cmd/
│   └── main.go
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   ├── mcp/
│   │   ├── server.go
│   │   ├── tools.go
│   │   └── server_test.go
│   ├── embedder/
│   │   ├── embedder.go
│   │   ├── onnx.go
│   │   ├── device.go
│   │   └── embedder_test.go
│   ├── vectordb/
│   │   ├── db.go
│   │   ├── sqlite.go
│   │   ├── schema.go
│   │   └── db_test.go
│   └── indexer/
│       ├── indexer.go
│       ├── markdown.go
│       ├── sync.go
│       └── indexer_test.go
├── models/
│   └── model.onnx
├── bin/                          # ビルド出力
├── testdata/                     # テスト用データ
├── go.mod
├── go.sum
├── build.sh
├── .gitignore
├── DESIGN.md
├── PLAN.md
├── README.md
├── CHANGELOG.md
└── LICENSE
```
