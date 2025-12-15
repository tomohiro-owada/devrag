# DevRag - Claude Code Project Guide

## Overview

DevRag is a lightweight RAG (Retrieval-Augmented Generation) MCP server for Claude Code. It enables semantic vector search over markdown documents, reducing token consumption by 40x compared to reading entire files.

## Quick Reference

| Item | Value |
|------|-------|
| Language | Go 1.23+ |
| Embedding Model | multilingual-e5-small (384 dimensions) |
| Vector DB | SQLite + sqlite-vec |
| Protocol | MCP (Model Context Protocol) via stdio |
| Current Version | v1.1.0 |

## Project Structure

```
devrag/
├── cmd/
│   └── main.go                 # Entry point, initialization sequence
├── internal/
│   ├── config/                 # Configuration management
│   │   ├── config.go           # Load, validate, save config
│   │   └── config_test.go
│   ├── embedder/               # Text-to-vector conversion
│   │   ├── embedder.go         # Interface + MockEmbedder
│   │   ├── onnx.go             # ONNX Runtime implementation
│   │   ├── device.go           # GPU/CPU detection
│   │   ├── download.go         # Model auto-download from HuggingFace
│   │   ├── tokenizer.go        # BERT tokenizer
│   │   └── embedder_test.go
│   ├── indexer/                # Document indexing
│   │   ├── indexer.go          # IndexFile, IndexDirectory
│   │   ├── markdown.go         # ParseMarkdown, chunk splitting
│   │   ├── sync.go             # Differential sync (add/update/delete)
│   │   └── markdown_test.go
│   ├── vectordb/               # Vector database operations
│   │   ├── db.go               # Insert, Delete, ListDocuments
│   │   ├── sqlite.go           # SQLite + vec0 initialization
│   │   ├── schema.go           # Table definitions
│   │   ├── search.go           # Cosine similarity search
│   │   └── db_test.go
│   ├── mcp/                    # MCP Protocol server
│   │   ├── server.go           # MCP server setup
│   │   └── tools.go            # 5 MCP tools implementation
│   └── frontmatter/            # Markdown frontmatter parsing
│       └── frontmatter.go
├── models/                     # ONNX model files (auto-downloaded)
├── test_data/                  # Test fixtures
├── build.sh                    # Build script
└── integration_test.go         # E2E tests
```

## Key Components

### 1. Configuration (`internal/config/`)

Loads `config.json` with defaults:

```json
{
  "documents_dir": "./documents",
  "db_path": "./vectors.db",
  "chunk_size": 500,
  "search_top_k": 5,
  "compute": { "device": "auto", "fallback_to_cpu": true },
  "model": { "name": "multilingual-e5-small", "dimensions": 384 }
}
```

### 2. Embedder (`internal/embedder/`)

- **ONNXEmbedder**: Uses ONNX Runtime for inference
- **MockEmbedder**: For testing without model files
- **Device Detection**: Auto-detects GPU (Metal on macOS, CUDA on Linux/Windows)
- **Auto-download**: Downloads model from HuggingFace on first run

### 3. Indexer (`internal/indexer/`)

- **ParseMarkdown**: Splits markdown into chunks respecting paragraph boundaries
- **Sync**: Detects new/updated/deleted files and updates database accordingly
- **IndexFile/IndexDirectory**: Full indexing pipeline

### 4. VectorDB (`internal/vectordb/`)

Schema:
```sql
documents(id, filename, indexed_at, modified_at)
chunks(id, document_id, position, content)
vec_chunks(embedding FLOAT[384])  -- sqlite-vec virtual table
```

Search uses `vec_distance_cosine()` for similarity ranking.

### 5. MCP Server (`internal/mcp/`)

7 tools available:
- `search` - Semantic search with query string and optional filters
- `index_markdown` - Index a single file
- `list_documents` - List all indexed documents
- `delete_document` - Remove document from index
- `reindex_document` - Re-index a document
- `add_frontmatter` - Add metadata to markdown files
- `update_frontmatter` - Update metadata in markdown files

#### Search Tool Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Natural language search query |
| `top_k` | number | No | Max results (default: 5) |
| `directory` | string | No | Filter to specific directory (e.g., "docs/api") |
| `file_pattern` | string | No | Glob pattern for filename (e.g., "api-*.md")

## Startup Sequence

1. Load config (`config.Load()`)
2. Validate config (`config.Validate()`)
3. Download model if needed (`embedder.DownloadModel()`)
4. Detect device (`embedder.DetectDevice()`)
5. Initialize database (`vectordb.Init()`)
6. Create documents directory if needed
7. Initialize embedder (ONNX or Mock)
8. Run differential sync (`indexer.Sync()`)
9. Start MCP server (`mcp.Start()`)

## Building

```bash
# Build for current platform
./build.sh

# Direct build
go build -o devrag cmd/main.go

# Run tests
go test ./...
```

## Testing

```bash
# Unit tests
go test ./internal/config -v
go test ./internal/indexer -v
go test ./internal/embedder -v
go test ./internal/vectordb -v

# Integration tests
go test . -v -run TestEndToEnd
```

## Common Development Tasks

### Adding a new MCP tool

1. Define tool schema in `internal/mcp/tools.go`
2. Implement handler function
3. Register in `registerTools()`

### Modifying chunk splitting logic

Edit `internal/indexer/markdown.go`:
- `splitIntoChunks()` - Main splitting logic
- `splitLargeParagraph()` - Handles oversized paragraphs

### Changing embedding model

1. Update model download URL in `internal/embedder/download.go`
2. Update dimensions in `config.go` defaults
3. Update tokenizer config path in `tokenizer.go`

## Important Notes

- **stdout is reserved for MCP protocol** - All logs go to stderr
- **CGO required** - sqlite-vec needs C bindings
- **First run needs internet** - Model auto-downloads (~450MB)
- **GPU detection is automatic** - Falls back to CPU if unavailable

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/mark3labs/mcp-go` | MCP protocol |
| `github.com/mattn/go-sqlite3` | SQLite driver |
| `github.com/asg017/sqlite-vec-go-bindings` | Vector search |
| `github.com/yalue/onnxruntime_go` | ONNX inference |
| `github.com/sugarme/tokenizer` | BERT tokenization |

## Performance Targets

| Metric | Target | Achieved |
|--------|--------|----------|
| Startup | < 3s | ~2.3s |
| Search | < 500ms | ~95ms |
| Indexing | > 100 chunks/s | ~200 chunks/s |
| Memory | < 500MB | ~200-400MB |
