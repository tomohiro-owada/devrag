#!/usr/bin/env python3
"""
モデルダウンロード・ONNX変換スクリプト

multilingual-e5-small モデルをHugging FaceからダウンロードしてONNX形式に変換します。
"""

import os
import sys
from pathlib import Path
from optimum.onnxruntime import ORTModelForFeatureExtraction
from transformers import AutoTokenizer

def main():
    # プロジェクトルートの models/ ディレクトリにダウンロード
    script_dir = Path(__file__).parent
    project_root = script_dir.parent
    models_dir = project_root / "models"

    print(f"モデル保存先: {models_dir}")

    # Hugging Faceのモデル名
    model_name = "intfloat/multilingual-e5-small"

    print(f"\n{model_name} をダウンロード中...")
    print("※ 初回実行時は時間がかかります（約120MB）\n")

    try:
        # ONNX形式でモデルをダウンロード・変換
        print("モデルをONNX形式に変換中...")
        model = ORTModelForFeatureExtraction.from_pretrained(
            model_name,
            export=True
        )

        # トークナイザーもダウンロード
        print("トークナイザーをダウンロード中...")
        tokenizer = AutoTokenizer.from_pretrained(model_name)

        # models/ ディレクトリに保存
        print(f"\n{models_dir} に保存中...")
        model.save_pretrained(str(models_dir))
        tokenizer.save_pretrained(str(models_dir))

        print("\n✓ ダウンロード完了！")
        print(f"\n配置されたファイル:")
        for file in sorted(models_dir.iterdir()):
            size_mb = file.stat().st_size / (1024 * 1024)
            print(f"  - {file.name} ({size_mb:.2f} MB)")

        return 0

    except Exception as e:
        print(f"\n✗ エラーが発生しました: {e}", file=sys.stderr)
        return 1

if __name__ == "__main__":
    sys.exit(main())
