# Mixed Content Document

## Code Block Test

The following code block should not be split:

```python
def calculate_embedding(text):
    # This is a long code block that should remain together
    # Even if it exceeds the chunk size
    tokens = tokenizer.encode(text)
    embeddings = model(tokens)
    return embeddings.numpy()

    # More lines to make it longer
    for i in range(100):
        print(f"Processing item {i}")
        result = process(i)
        if result:
            save(result)
```

## Japanese Text

日本語のテキストです。これは、日本語の文章が適切に処理されることを確認するためのテストです。日本語には句読点として「。」や「！」や「?」が使用されます。

長い日本語の段落をテストします。この段落は500文字を超えるように意図的に長く書かれています。マークダウンパーサーは、この長い段落を適切な境界で分割する必要があります。日本語の文の境界は「。」で判断されます。英語と違って、日本語には明確な単語の区切りがないため、文の境界での分割が重要です。この実装では、句読点を検出して適切に分割します。さらに、コードブロックが含まれている場合は、それを分割しないように注意する必要があります。ベクトル検索の精度を高めるためには、意味のある単位でテキストを分割することが重要です。ここで追加のテキストを入れます。これにより、段落が確実に500文字を超えるようにします。日本語のテキスト処理では、文字数のカウント方法が重要になります。バイト数ではなく、ルーン数（Unicodeコードポイント数）を使用する必要があります。これにより、日本語のような多バイト文字でも正しく処理できます。この実装では、utf8.RuneCountInString関数を使用しています。さらにテキストを追加して、確実に500文字を超えるようにします。Goのstringsパッケージには、文字列を操作するための便利な関数が多数用意されています。段落の分割アルゴリズムは、これらの関数を活用して実装されています。

## Mixed Language Paragraph

This paragraph contains both English and Japanese. 日本語と英語が混在しています。The chunking algorithm should handle this properly. 適切に処理されるべきです。

## Short Paragraphs

First paragraph.

Second paragraph.

Third paragraph.

Fourth paragraph.

These short paragraphs should be combined into a single chunk if they fit within the size limit.
