---
name: phase2-core
description: Phase 2コア機能実装。マークダウンパーサー、ベクトル化処理、インデックス化、差分同期、ベクトル検索を実装。Phase 1完了後、検索機能の実装時に使用。
---

# Phase 2: コア機能実装

markdown-vector-mcpのベクトル検索コア機能を実装します。

## 前提条件

Phase 1が完了していること：
- Goプロジェクトが初期化されている
- 設定モジュールが実装されている
- SQLite + vec0がセットアップされている
- ONNX Runtimeが統合されている

## タスク一覧

### 2.1 マークダウンパーサー実装

**ファイル**: `internal/indexer/markdown.go`

```go
package indexer

import (
    "bufio"
    "fmt"
    "os"
    "strings"
)

type Chunk struct {
    Content  string
    Position int
}

// ParseMarkdown parses a markdown file and splits into chunks
func ParseMarkdown(filepath string, chunkSize int) ([]Chunk, error) {
    file, err := os.Open(filepath)
    if err != nil {
        return nil, fmt.Errorf("failed to open file: %w", err)
    }
    defer file.Close()

    // Read entire file
    var content strings.Builder
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        content.WriteString(scanner.Text())
        content.WriteString("\n")
    }

    if err := scanner.Err(); err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }

    // Split into chunks
    chunks := splitIntoChunks(content.String(), chunkSize)

    // Create Chunk structs
    result := make([]Chunk, len(chunks))
    for i, c := range chunks {
        result[i] = Chunk{
            Content:  c,
            Position: i,
        }
    }

    return result, nil
}

// splitIntoChunks splits text into chunks of approximately chunkSize characters
func splitIntoChunks(content string, chunkSize int) []string {
    if len(content) <= chunkSize {
        return []string{content}
    }

    var chunks []string
    var currentChunk strings.Builder

    // Split by paragraphs (double newline)
    paragraphs := strings.Split(content, "\n\n")

    for _, para := range paragraphs {
        para = strings.TrimSpace(para)
        if para == "" {
            continue
        }

        // If adding this paragraph exceeds chunk size, start new chunk
        if currentChunk.Len() > 0 && currentChunk.Len()+len(para) > chunkSize {
            chunks = append(chunks, currentChunk.String())
            currentChunk.Reset()
        }

        // If single paragraph is too large, split it
        if len(para) > chunkSize {
            // Split by sentences or fixed size
            subChunks := splitLargeParagraph(para, chunkSize)
            for _, sub := range subChunks {
                if currentChunk.Len() > 0 {
                    chunks = append(chunks, currentChunk.String())
                    currentChunk.Reset()
                }
                chunks = append(chunks, sub)
            }
        } else {
            if currentChunk.Len() > 0 {
                currentChunk.WriteString("\n\n")
            }
            currentChunk.WriteString(para)
        }
    }

    // Add remaining chunk
    if currentChunk.Len() > 0 {
        chunks = append(chunks, currentChunk.String())
    }

    return chunks
}

// splitLargeParagraph splits a large paragraph into smaller chunks
func splitLargeParagraph(para string, chunkSize int) []string {
    var chunks []string
    for len(para) > chunkSize {
        // Try to split at sentence boundary
        cutPoint := chunkSize
        for i := chunkSize; i > chunkSize/2; i-- {
            if para[i] == '.' || para[i] == '!' || para[i] == '?' || para[i] == '\n' {
                cutPoint = i + 1
                break
            }
        }
        chunks = append(chunks, strings.TrimSpace(para[:cutPoint]))
        para = strings.TrimSpace(para[cutPoint:])
    }
    if len(para) > 0 {
        chunks = append(chunks, para)
    }
    return chunks
}
```

**テストケース**:
- 短いファイル（<500文字）
- 長いファイル（>5000文字）
- コードブロック含む
- 日本語・英語混在

### 2.2 ベクトル化処理実装

**ファイル**: `internal/embedder/tokenizer.go`

```go
package embedder

import (
    "strings"
    "unicode"
)

// SimpleTokenizer provides basic tokenization
// Note: For production, use proper BPE tokenizer
type SimpleTokenizer struct {
    vocabSize int
}

// Tokenize converts text to token IDs (simplified)
func (t *SimpleTokenizer) Tokenize(text string) []int32 {
    // Simplified tokenization
    // In production, use proper BPE tokenizer from sentencepiece or similar

    text = strings.ToLower(text)
    words := strings.FieldsFunc(text, func(r rune) bool {
        return unicode.IsSpace(r) || unicode.IsPunct(r)
    })

    // Convert words to IDs (simplified)
    tokens := make([]int32, len(words))
    for i, word := range words {
        // Simple hash-based ID assignment
        tokens[i] = int32(hashString(word) % t.vocabSize)
    }

    return tokens
}

func hashString(s string) int {
    h := 0
    for _, c := range s {
        h = h*31 + int(c)
    }
    if h < 0 {
        h = -h
    }
    return h
}
```

**ファイル**: `internal/embedder/onnx.go` (update)

```go
// Embed embeds a single text (implementation)
func (e *ONNXEmbedder) Embed(text string) ([]float32, error) {
    // TODO: Proper tokenization
    // For now, return placeholder

    // Tokenize
    tokenizer := &SimpleTokenizer{vocabSize: 30000}
    tokens := tokenizer.Tokenize(text)

    // Run inference
    // TODO: Implement ONNX inference

    // Return placeholder embedding
    embedding := make([]float32, 384)
    for i := range embedding {
        embedding[i] = 0.1
    }

    return embedding, nil
}

// EmbedBatch embeds multiple texts
func (e *ONNXEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
    results := make([][]float32, len(texts))
    for i, text := range texts {
        emb, err := e.Embed(text)
        if err != nil {
            return nil, err
        }
        results[i] = emb
    }
    return results, nil
}
```

### 2.3 インデックス化処理実装

**ファイル**: `internal/indexer/indexer.go`

```go
package indexer

import (
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/towada/markdown-vector-mcp/internal/config"
    "github.com/towada/markdown-vector-mcp/internal/embedder"
    "github.com/towada/markdown-vector-mcp/internal/vectordb"
)

type Indexer struct {
    db       *vectordb.DB
    embedder embedder.Embedder
    config   *config.Config
}

// NewIndexer creates a new indexer
func NewIndexer(db *vectordb.DB, emb embedder.Embedder, cfg *config.Config) *Indexer {
    return &Indexer{
        db:       db,
        embedder: emb,
        config:   cfg,
    }
}

// IndexFile indexes a single markdown file
func (idx *Indexer) IndexFile(filepath string) error {
    fmt.Fprintf(os.Stderr, "[INFO] Indexing file: %s\n", filepath)

    // Get file info
    info, err := os.Stat(filepath)
    if err != nil {
        return fmt.Errorf("failed to stat file: %w", err)
    }

    // Parse markdown
    chunks, err := ParseMarkdown(filepath, idx.config.ChunkSize)
    if err != nil {
        return fmt.Errorf("failed to parse markdown: %w", err)
    }

    fmt.Fprintf(os.Stderr, "[INFO] Parsed %d chunks\n", len(chunks))

    // Vectorize chunks
    texts := make([]string, len(chunks))
    for i, chunk := range chunks {
        texts[i] = chunk.Content
    }

    vectors, err := idx.embedder.EmbedBatch(texts)
    if err != nil {
        return fmt.Errorf("failed to vectorize: %w", err)
    }

    // Store in database
    filename := filepath
    if err := idx.db.InsertDocument(filename, info.ModTime(), chunks, vectors); err != nil {
        return fmt.Errorf("failed to store in database: %w", err)
    }

    fmt.Fprintf(os.Stderr, "[INFO] Successfully indexed %s (%d chunks)\n", filename, len(chunks))
    return nil
}

// IndexDirectory indexes all markdown files in a directory
func (idx *Indexer) IndexDirectory(dir string) error {
    return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if !info.IsDir() && filepath.Ext(path) == ".md" {
            if err := idx.IndexFile(path); err != nil {
                fmt.Fprintf(os.Stderr, "[WARN] Failed to index %s: %v\n", path, err)
            }
        }

        return nil
    })
}
```

**ファイル**: `internal/vectordb/sqlite.go` (update)

```go
// InsertDocument inserts a document with its chunks and vectors
func (db *DB) InsertDocument(filename string, modTime time.Time, chunks []indexer.Chunk, vectors [][]float32) error {
    tx, err := db.conn.Begin()
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Insert document
    result, err := tx.Exec(
        "INSERT OR REPLACE INTO documents (filename, modified_at) VALUES (?, ?)",
        filename, modTime,
    )
    if err != nil {
        return fmt.Errorf("failed to insert document: %w", err)
    }

    docID, err := result.LastInsertId()
    if err != nil {
        return fmt.Errorf("failed to get document ID: %w", err)
    }

    // Insert chunks
    for i, chunk := range chunks {
        result, err := tx.Exec(
            "INSERT INTO chunks (document_id, position, content) VALUES (?, ?, ?)",
            docID, chunk.Position, chunk.Content,
        )
        if err != nil {
            return fmt.Errorf("failed to insert chunk: %w", err)
        }

        chunkID, err := result.LastInsertId()
        if err != nil {
            return fmt.Errorf("failed to get chunk ID: %w", err)
        }

        // Insert vector
        // TODO: Insert into vec_chunks virtual table
        _ = chunkID
        _ = vectors[i]
    }

    return tx.Commit()
}
```

### 2.4 差分同期機能実装

**ファイル**: `internal/indexer/sync.go`

```go
package indexer

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
)

type SyncResult struct {
    Added    []string
    Updated  []string
    Deleted  []string
}

// Sync synchronizes the documents directory with the database
func (idx *Indexer) Sync() (*SyncResult, error) {
    fmt.Fprintf(os.Stderr, "[INFO] Starting sync...\n")

    result := &SyncResult{
        Added:   []string{},
        Updated: []string{},
        Deleted: []string{},
    }

    // Get files from database
    dbFiles, err := idx.db.ListDocuments()
    if err != nil {
        return nil, fmt.Errorf("failed to list database files: %w", err)
    }

    dbFileMap := make(map[string]time.Time)
    for filename, modTime := range dbFiles {
        dbFileMap[filename] = modTime
    }

    // Scan filesystem
    fsFiles := make(map[string]time.Time)
    err = filepath.Walk(idx.config.DocumentsDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if !info.IsDir() && filepath.Ext(path) == ".md" {
            fsFiles[path] = info.ModTime()
        }

        return nil
    })
    if err != nil {
        return nil, fmt.Errorf("failed to scan filesystem: %w", err)
    }

    // Detect changes
    for fsPath, fsMtime := range fsFiles {
        if dbMtime, exists := dbFileMap[fsPath]; !exists {
            // New file
            result.Added = append(result.Added, fsPath)
            if err := idx.IndexFile(fsPath); err != nil {
                fmt.Fprintf(os.Stderr, "[ERROR] Failed to index new file %s: %v\n", fsPath, err)
            }
        } else if !fsMtime.Equal(dbMtime) {
            // Updated file
            result.Updated = append(result.Updated, fsPath)
            if err := idx.db.DeleteDocument(fsPath); err != nil {
                fmt.Fprintf(os.Stderr, "[ERROR] Failed to delete old version of %s: %v\n", fsPath, err)
                continue
            }
            if err := idx.IndexFile(fsPath); err != nil {
                fmt.Fprintf(os.Stderr, "[ERROR] Failed to reindex %s: %v\n", fsPath, err)
            }
        }
    }

    // Detect deletions
    for dbPath := range dbFileMap {
        if _, exists := fsFiles[dbPath]; !exists {
            result.Deleted = append(result.Deleted, dbPath)
            if err := idx.db.DeleteDocument(dbPath); err != nil {
                fmt.Fprintf(os.Stderr, "[ERROR] Failed to delete %s from database: %v\n", dbPath, err)
            }
        }
    }

    fmt.Fprintf(os.Stderr, "[INFO] Sync complete: +%d, ~%d, -%d\n",
        len(result.Added), len(result.Updated), len(result.Deleted))

    return result, nil
}
```

### 2.5 ベクトル検索実装

**ファイル**: `internal/vectordb/search.go`

```go
package vectordb

import (
    "fmt"
)

type SearchResult struct {
    DocumentName string
    ChunkContent string
    Similarity   float64
    Position     int
}

// Search performs vector similarity search
func (db *DB) Search(queryVector []float32, topK int) ([]SearchResult, error) {
    // TODO: Implement vec0 similarity search
    // SQL query:
    // SELECT
    //     d.filename,
    //     c.content,
    //     c.position,
    //     vec_distance_cosine(v.embedding, ?) as similarity
    // FROM vec_chunks v
    // JOIN chunks c ON v.chunk_id = c.id
    // JOIN documents d ON c.document_id = d.id
    // ORDER BY similarity ASC
    // LIMIT ?

    // Placeholder implementation
    results := []SearchResult{}
    return results, nil
}

// ListDocuments returns all indexed documents
func (db *DB) ListDocuments() (map[string]time.Time, error) {
    rows, err := db.conn.Query("SELECT filename, modified_at FROM documents")
    if err != nil {
        return nil, fmt.Errorf("failed to query documents: %w", err)
    }
    defer rows.Close()

    docs := make(map[string]time.Time)
    for rows.Next() {
        var filename string
        var modTime time.Time
        if err := rows.Scan(&filename, &modTime); err != nil {
            return nil, fmt.Errorf("failed to scan row: %w", err)
        }
        docs[filename] = modTime
    }

    return docs, rows.Err()
}

// DeleteDocument deletes a document and its chunks
func (db *DB) DeleteDocument(filename string) error {
    _, err := db.conn.Exec("DELETE FROM documents WHERE filename = ?", filename)
    return err
}
```

## Phase 2 完了条件

- [ ] マークダウンファイルがパースできる
- [ ] チャンク分割が正しく動作する
- [ ] テキストがベクトル化できる
- [ ] インデックス化処理が完了する
- [ ] 差分同期が動作する
- [ ] 検索クエリが実行できる（vec0統合後）

## パフォーマンス目標

- インデックス速度: > 100 chunks/sec
- 検索レスポンス: < 500ms（1000件/DB）
- メモリ使用量: < 500MB

## 次のステップ

Phase 2完了後は **phase3-mcp** スキルを使用してMCPプロトコルを統合します。
