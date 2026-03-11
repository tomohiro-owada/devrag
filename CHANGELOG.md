# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.0] - 2026-03-11

### Added
- **CLI subcommand interface** - All MCP tools now available as CLI commands
  - `devrag search`, `index`, `index-code`, `list`, `delete`, `reindex`, `search-relations`, `build-dictionary`, `add-frontmatter`, `update-frontmatter`
  - `devrag serve` or no subcommand starts MCP server (backwards compatible)
- **Agent-first CLI design** following AI CLI design guidelines
  - JSON output by default on all commands (`--output text` for human-readable)
  - Stable response envelope: `{status, data, error}` on every command
  - Structured errors with `{code, message, field, hint}`
  - `devrag schema` returns full CLI specification as machine-readable JSON
  - `--fields` flag to limit output fields (token efficiency)
  - `--dry-run` on destructive operations (`delete`, `reindex`)
  - `--params` JSON payload input for complex operations (frontmatter)
  - Input hardening: path traversal rejection, control character validation
  - Flag syntax documented in schema for AI agent consumption

## [1.2.2] - 2026-03-11

### Fixed
- **FATAL panic in tokenizer** - Added `recover()` to catch panics from upstream `sugarme/tokenizer` v0.3.0 bug (#15)
  - Consecutive whitespace and certain text patterns caused `slice bounds out of range` panic in Metaspace pretokenizer
  - Affected ONNX embedder only (mock embedder was not affected)
- **Batch embedding resilience** - `EmbedBatch` no longer fails the entire batch when individual chunks cause tokenizer errors
  - Failed chunks now use zero-vector fallback instead of aborting the whole indexing process
  - Warnings logged to stderr for visibility

## [1.4.0] - 2026-01-13

### Added
- **Code indexing with AST parsing** - Tree-sitter based source code analysis
  - Supported languages: Go, Python, TypeScript, JavaScript, PHP, Rust, Vue
  - Extracts functions, classes, methods as searchable chunks
  - New MCP tool: `index_code` for indexing source code files/directories
- **Code relation extraction** - Knowledge graph for code navigation
  - Extracts call relationships (function calls)
  - Extracts import relationships (dependencies)
  - Extracts inheritance relationships (classes)
  - New MCP tool: `search_relations` to query code relationships
- **Dictionary feature** - Japanese→English word mapping for multilingual search
  - Auto-extracts word mappings from `日本語 (English)` patterns in documents
  - CamelCase splitting for better matching
  - New MCP tool: `build_dictionary` to build/update dictionary
  - Auto-builds dictionary when Japanese search has no translations

### Fixed
- **Relation search query** - Fixed NULL target_chunk_id issue causing 0 results
- **Code relation extraction** - Fixed to use full file content instead of chunk content

### Changed
- Total MCP tools increased from 7 to 10
- Improved search to use dictionary translations for Japanese queries

### Technical Details
- Added `code_metadata` table for source code symbol information
- Added `code_relations` table for relationship graph
- Added `word_mapping` table for multilingual dictionary

## [1.3.0] - 2025-12-22

### Added
- **Update checker** - Notifies when new version is available
  - Shows update notification in search results
  - 24-hour cache to avoid excessive API calls

### Changed
- **Optional file deletion** - `delete_document` tool now only removes from DB by default
  - Physical file deletion is optional via parameter

## [1.2.0] - 2025-12-15

### Added
- **Filtered search** - New parameters for the `search` tool
  - `directory`: Filter results to specific directory (e.g., "docs/api")
  - `file_pattern`: Filter by filename glob pattern (e.g., "api-*.md")
  - `top_k`: Control maximum number of results
- **Multiple document paths** - Support for multiple document directories with glob patterns
  - Configure via `document_patterns` array in config.json
  - Supports recursive patterns like `./docs/**/*.md`
  - Backward compatible with old `documents_dir` field
- **Custom config file** - `--config` CLI flag to specify configuration file path
- **CLAUDE.md** - Comprehensive project guide for Claude Code
- **Contributors section** - Added to README to acknowledge community contributions

### Fixed
- **Security: Info leak** - Removed user input (query, directory, file_pattern) from stderr logs

### Changed
- Improved README documentation with filtered search examples
- Updated MCP tool descriptions

### Contributors
- [@badri](https://github.com/badri) - Multiple document paths (#2), --config CLI flag (#3)
- [@io41](https://github.com/io41) - Project cleanup (#4)

## [1.1.0] - 2024-10-25

### Changed
- **Project renamed to DevRag** - Rebranded from markdown-vector-mcp to devrag
- Updated module name to `github.com/tomohiro-owada/devrag`
- All binary names changed to `devrag-*`
- MCP server configuration name changed to `devrag`

### Added
- Frontmatter support for markdown files
- Local MCP configuration file (.mcp.json)

## [1.0.2] - 2024-10-24

### Changed
- **User-friendly binary names** - More intuitive file names for releases
  - `macos-apple-silicon` instead of `darwin-arm64` (for M1/M2/M3 Macs)
  - `macos-intel` instead of `darwin-amd64` (for Intel Macs)
  - `linux-x64` instead of `linux-amd64`
  - `windows-x64` instead of `windows-amd64`

## [1.0.1] - 2024-10-24

### Added
- **Automatic model download on first run** - No Python dependencies required!
- Models are automatically downloaded from Hugging Face on first startup
- Progress display during model download
- Eliminates the need for manual model setup

### Changed
- Simplified installation process - just build and run
- Updated documentation to reflect automatic download feature

## [1.0.0] - 2024-10-24

### Added
- Initial release of DevRag
- Vector search for markdown files using multilingual-e5-small model
- MCP Protocol support for integration with Claude Desktop
- Cross-platform builds for macOS (arm64/amd64)
- GPU/CPU auto-detection with fallback support
- Automatic file synchronization on startup
- Five MCP tools:
  - `search` - Natural language semantic search
  - `index_markdown` - Index markdown files
  - `list_documents` - List all indexed documents
  - `delete_document` - Remove document from index
  - `reindex_document` - Re-index a document
- SQLite-based vector database using sqlite-vec
- Configurable chunk size and search parameters
- Comprehensive test suite (unit tests and integration tests)
- Build script for easy compilation
- Detailed documentation in README

### Technical Details
- Written in Go 1.21+
- Uses ONNX Runtime for model inference
- Supports Japanese and English text
- Vector dimension: 384
- Default chunk size: 500 characters
- Default search results: top 5

### Supported Platforms
- macOS (arm64) - Apple Silicon
- macOS (amd64) - Intel
- Linux (amd64) - with appropriate cross-compilation setup
- Windows (amd64) - with appropriate cross-compilation setup

### Dependencies
- github.com/asg017/sqlite-vec-go-bindings - SQLite vector extension
- github.com/yalue/onnxruntime_go - ONNX Runtime bindings
- Standard Go libraries

### Performance
- Startup time: ~2-3 seconds (including model loading)
- Indexing speed: ~100-200 chunks/second
- Search response: <100ms for 1000 documents
- Memory usage: ~200-400MB
- Binary size: ~7-8MB

### Known Limitations
- Cross-compilation with CGO requires platform-specific toolchains
- GPU support currently limited to macOS Apple Silicon (Metal)
- First run requires internet connection to download model files (~450MB)

## [Unreleased]

### Planned Features
- Additional language model support
- Batch indexing improvements
- Enhanced error messages
- Configuration validation tool
- Performance monitoring and metrics

---

For more information, see the [README](README.md).
