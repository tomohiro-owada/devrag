# Long Document for Chunking Test

This is a long document designed to test the chunking functionality. It contains multiple paragraphs and sections to ensure proper splitting.

## Section 1: Introduction

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.

これは日本語の段落です。マークダウンベクトル検索システムは、日本語と英語の混在したテキストを適切に処理する必要があります。この段落は、日本語の文章が正しくチャンク化されることを確認するためのテストケースです。

## Section 2: Technical Details

The system implements vector search using SQLite with vec0 extension. It supports:

- Document parsing and chunking
- Text vectorization using ONNX Runtime
- Efficient similarity search
- Incremental indexing

### Subsection 2.1: Chunk Size

The default chunk size is 500 characters. This ensures that each chunk is meaningful while keeping the vector embeddings focused. The chunker respects paragraph boundaries and tries to split at sentence boundaries when a paragraph is too large.

### Subsection 2.2: Vector Embeddings

We use the multilingual-e5-small model which produces 384-dimensional vectors. This model is optimized for multilingual text and works well with both English and Japanese content.

## Section 3: Performance Considerations

Performance is critical for a good user experience. The system is designed to:

1. Index documents quickly (target: >100 chunks/sec)
2. Respond to search queries fast (target: <500ms for 1000 documents)
3. Use minimal memory (target: <500MB)

### Subsection 3.1: Optimization Techniques

Several optimization techniques are employed:

- Batch processing of embeddings
- Efficient SQLite queries with proper indexing
- Memory-mapped database files
- ONNX Runtime with hardware acceleration (Metal on macOS)

## Section 4: Code Example

Here's an example of how to use the parser:

```go
chunks, err := ParseMarkdown("document.md", 500)
if err != nil {
    log.Fatal(err)
}

for i, chunk := range chunks {
    fmt.Printf("Chunk %d: %s\n", i, chunk.Content)
}
```

This code demonstrates the basic usage of the ParseMarkdown function. It reads a markdown file and splits it into chunks of approximately 500 characters each.

## Section 5: Conclusion

This document contains enough text to generate multiple chunks. The chunking algorithm should properly handle:

- Multiple paragraphs
- Code blocks (which should not be split)
- Mixed language content (English and Japanese)
- Proper boundary detection

日本語での結論: このシステムは、マークダウンファイルを効率的にベクトル検索できるようにします。チャンク分割アルゴリズムは、段落の境界を尊重し、意味のある単位でテキストを分割します。

End of document.
