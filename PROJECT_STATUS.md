# devrag - プロジェクト現状

**最終更新**: 2025-10-24
**現在のフェーズ**: Phase 3 完了 → Phase 4 準備完了

---

## プロジェクト概要

Claude Code用のMCPサーバー。マークダウンファイルのベクトル検索を提供。

### 主要機能
- マークダウンの自動インデックス化
- セマンティック検索（ベクトル検索）
- 差分同期（新規・更新・削除の自動検出）
- GPU/CPU自動検出
- 5つのMCPツール

---

## フェーズ別完了状況

### ✅ Phase 1: 基盤構築
- [x] プロジェクト初期化
- [x] 設定管理（config.json）
- [x] データベース実装（SQLite + sqlite-vec）
- [x] デバイス検出（GPU/CPU）

### ✅ Phase 2: コア機能実装
- [x] マークダウンパーサー
- [x] ONNXベクトル化エンジン
- [x] ベクトル検索
- [x] 差分同期機能

### ✅ Phase 3: MCP統合
- [x] MCPサーバー実装
- [x] 5つのツール実装
- [x] メインループ統合

### ⏳ Phase 4: テストとビルド（次のステップ）
- [ ] ユニットテスト
- [ ] 統合テスト
- [ ] ビルドスクリプト
- [ ] ドキュメント整備

---

## MCPツール一覧

| ツール名 | 説明 | 主要パラメータ |
|---------|------|--------------|
| `search` | ベクトル検索 | query, top_k |
| `index_markdown` | ファイルインデックス化 | filepath |
| `list_documents` | ドキュメント一覧 | なし |
| `delete_document` | ドキュメント削除 | filename |
| `reindex_document` | 再インデックス化 | filename |

---

## ディレクトリ構成

```
devrag/
├── cmd/
│   └── main.go              # MCPサーバーエントリーポイント
├── internal/
│   ├── config/             # 設定管理
│   ├── embedder/           # ベクトル化エンジン
│   ├── indexer/            # インデックス作成・同期
│   ├── mcp/                # MCPサーバー・ツール
│   └── vectordb/           # ベクトルデータベース
├── documents/              # インデックス対象ディレクトリ
├── models/                 # ONNXモデル（別途ダウンロード）
├── config.json             # 設定ファイル
└── vectors.db              # ベクトルデータベース
```

---

## ビルドと実行

### ビルド
```bash
go build -o devrag cmd/main.go
```

### 実行
```bash
./devrag
```

### 設定ファイル（config.json）
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

---

## Claude Code統合

### 設定方法

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

### 使用例

Claude Codeで以下のような自然言語でリクエスト：
- "マークダウンドキュメントからXXXについて検索して"
- "ドキュメント一覧を表示して"
- "test.mdをインデックス化して"

---

## 技術スタック

- **言語**: Go 1.23+
- **ベクトルDB**: SQLite + sqlite-vec
- **ベクトル化**: ONNX Runtime
- **モデル**: multilingual-e5-small (384次元)
- **プロトコル**: MCP (Model Context Protocol)
- **通信**: stdio (JSON-RPC 2.0)

---

## パフォーマンス

- **GPU検出**: 自動（Metal/CUDA対応）
- **バッチ処理**: 対応
- **メモリ効率**: チャンク単位の処理
- **差分同期**: ファイルmtime比較

---

## 現在の制限事項

1. **モデルファイル**: 別途ダウンロード必要
   - 場所: `models/multilingual-e5-small/model.onnx`
   - 無い場合はMockモードで動作（開発用）

2. **対応ファイル**: マークダウン（.md）のみ

3. **言語**: 日本語・英語対応（multilingual-e5-smallによる）

---

## 次のステップ

### 優先度: 高
1. Phase 4のテスト実装
2. 実モデルファイルの配置
3. Claude Codeでの実地テスト

### 優先度: 中
1. ドキュメント整備
2. エラーハンドリングの強化
3. パフォーマンス最適化

### 優先度: 低
1. 他のファイル形式対応（PDF、テキストなど）
2. カスタムモデル対応
3. Web UI追加

---

## トラブルシューティング

### ビルドエラー
```bash
# 依存関係の更新
go mod tidy
go mod download
```

### 実行時エラー
```bash
# デバッグモード（ログ確認）
./devrag 2>&1 | tee debug.log

# データベースのリセット
rm vectors.db
```

### GPU認識しない
```bash
# config.jsonでCPU強制
"device": "cpu"
```

---

## リンク

- **設計書**: `DESIGN.md`
- **実装計画**: `PLAN.md`
- **Phase 2完了報告**: `PHASE_2_3_REPORT.md`
- **Phase 3完了報告**: `PHASE_3_COMPLETION_REPORT.md`
- **スキルファイル**: `.claude/skills/phase3-mcp/SKILL.md`

---

## 貢献者

- Phase 1-3 実装: 2025-10-24

## ライセンス

（未定）
