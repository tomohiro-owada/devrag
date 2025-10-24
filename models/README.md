# モデルファイル

このディレクトリには、ベクトル埋め込み用のONNXモデルが配置されます。

## 現在のモデル

- **モデル名**: intfloat/multilingual-e5-small
- **形式**: ONNX
- **サイズ**: 約448MB
- **次元**: 384
- **ライセンス**: MIT
- **特徴**: 多言語対応（日本語・英語など）

## ファイル一覧

```
models/
├── model.onnx              # ONNX形式のモデル本体 (448MB)
├── tokenizer.json          # トークナイザー設定 (16MB)
├── config.json             # モデル設定
├── tokenizer_config.json   # トークナイザー詳細設定
└── special_tokens_map.json # 特殊トークン定義
```

## モデルのダウンロード方法

### 自動ダウンロード（推奨）

プロジェクトルートから以下のコマンドを実行:

```bash
python3 scripts/download_model.py
```

### 必要な環境

- Python 3.8+
- pip パッケージ:
  - transformers
  - optimum[onnxruntime]

### 手動でのインストール

```bash
pip3 install transformers 'optimum[onnxruntime]'
python3 scripts/download_model.py
```

## モデル詳細

### アーキテクチャ

- ベース: BERT (BertModel)
- 隠れ層サイズ: 384
- アテンションヘッド: 12
- レイヤー数: 12
- 最大トークン数: 512
- 語彙サイズ: 250,037

### 用途

このモデルは以下の用途で使用されます:

1. マークダウンチャンクのベクトル埋め込み生成
2. 検索クエリのベクトル埋め込み生成
3. コサイン類似度による意味検索

### パフォーマンス

- CPU: 約100-200ms/チャンク
- GPU (Apple Silicon): 約10-30ms/チャンク

## 代替モデル（将来的な選択肢）

より小さいモデルが必要な場合:
- `sentence-transformers/all-MiniLM-L6-v2` (80MB, 384次元, 英語のみ)
- `sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2` (420MB, 384次元)

より精度が必要な場合:
- `intfloat/multilingual-e5-base` (1.1GB, 768次元)
- `intfloat/multilingual-e5-large` (2.2GB, 1024次元)

## トラブルシューティング

### ダウンロードに失敗する

```bash
# キャッシュをクリア
rm -rf ~/.cache/huggingface/
python3 scripts/download_model.py
```

### モデルサイズが大きすぎる

量子化版を使用する（未実装）、または代替の小さいモデルを検討してください。

## 参考リンク

- Hugging Face モデルページ: https://huggingface.co/intfloat/multilingual-e5-small
- E5 論文: https://arxiv.org/abs/2212.03533
- ONNX Runtime: https://onnxruntime.ai/
