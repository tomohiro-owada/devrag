# Model Download Instructions

このディレクトリにONNXモデルファイルをダウンロードする必要があります。

## 自動ダウンロード（推奨）

```bash
# プロジェクトルートから実行
python3 scripts/download_model.py
```

## 必要なパッケージ

```bash
pip install transformers optimum[onnxruntime]
```

## ダウンロードされるファイル

- `model.onnx` (約450MB) - multilingual-e5-smallのONNXモデル
- `tokenizer.json` (約16MB) - トークナイザー設定
- `config.json` - モデル設定
- `special_tokens_map.json` - 特殊トークン定義
- `tokenizer_config.json` - トークナイザー詳細設定

## 手動ダウンロード

Hugging Faceから直接ダウンロードする場合：

```bash
# Hugging Face CLIを使用
huggingface-cli download intfloat/multilingual-e5-small --local-dir models/
```

または、Webブラウザから：
https://huggingface.co/intfloat/multilingual-e5-small

## モデル情報

- **モデル名**: intfloat/multilingual-e5-small
- **サイズ**: 約450MB（ONNX形式）
- **次元**: 384
- **言語**: 多言語対応（日本語・英語含む100以上の言語）
- **ライセンス**: MIT

## トラブルシューティング

### Pythonパッケージのインストールエラー

```bash
# 仮想環境を使用
python3 -m venv venv
source venv/bin/activate  # macOS/Linux
# または
venv\Scripts\activate  # Windows

pip install transformers optimum[onnxruntime]
python3 scripts/download_model.py
```

### ダウンロードが遅い

- Hugging Faceのミラーサイトを使用
- または、VPNを使用

### ストレージ容量不足

モデルファイルは約500MB必要です。十分な空き容量を確保してください。
