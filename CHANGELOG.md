# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
- Initial release of markdown-vector-mcp
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
