# Model Files

このディレクトリには`multilingual-e5-small`モデルファイルが配置されます。

## 自動ダウンロード（推奨）

**初回起動時に自動的にダウンロードされます！**

```bash
# ビルド後、初回起動時に自動ダウンロード
./devrag
```

プログラムは起動時にモデルファイルの存在を確認し、存在しない場合は自動的にHugging Faceからダウンロードします。

## ダウンロードされるファイル

- `model.onnx` (約450MB) - multilingual-e5-smallのONNXモデル
- `tokenizer.json` (約16MB) - トークナイザー設定
- `config.json` - モデル設定
- `special_tokens_map.json` - 特殊トークン定義
- `tokenizer_config.json` - トークナイザー詳細設定

## モデル情報

- **モデル名**: intfloat/multilingual-e5-small
- **サイズ**: 約450MB（ONNX形式）
- **次元**: 384
- **言語**: 多言語対応（日本語・英語含む100以上の言語）
- **ライセンス**: MIT
- **ソース**: https://huggingface.co/intfloat/multilingual-e5-small

## 手動ダウンロード（オプション）

事前にダウンロードしたい場合は、以下の方法があります：

### 方法1: curlを使用

```bash
cd models
curl -L -o model.onnx "https://huggingface.co/intfloat/multilingual-e5-small/resolve/main/onnx/model.onnx"
curl -L -o tokenizer.json "https://huggingface.co/intfloat/multilingual-e5-small/resolve/main/tokenizer.json"
curl -L -o config.json "https://huggingface.co/intfloat/multilingual-e5-small/resolve/main/config.json"
curl -L -o special_tokens_map.json "https://huggingface.co/intfloat/multilingual-e5-small/resolve/main/special_tokens_map.json"
curl -L -o tokenizer_config.json "https://huggingface.co/intfloat/multilingual-e5-small/resolve/main/tokenizer_config.json"
```

### 方法2: Hugging Face CLIを使用

```bash
pip install huggingface-hub
huggingface-cli download intfloat/multilingual-e5-small --local-dir models/
```

### 方法3: Pythonスクリプト（レガシー）

```bash
python3 scripts/download_model.py
```

## トラブルシューティング

### ダウンロードが遅い

- ネットワーク接続を確認してください
- Hugging Faceのサーバーが混雑している可能性があります
- 手動ダウンロードを試してください

### ストレージ容量不足

モデルファイルは約500MB必要です。十分な空き容量を確保してください。

### プロキシ環境

プロキシ環境でダウンロードできない場合：

```bash
export HTTP_PROXY=http://your-proxy:port
export HTTPS_PROXY=http://your-proxy:port
./devrag
```

または、手動でダウンロードしてこのディレクトリに配置してください。
