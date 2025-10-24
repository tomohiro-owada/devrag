# Large Test Document

This is a comprehensive test document to verify chunking and indexing functionality.

## Introduction

The markdown-vector-mcp project is a Model Context Protocol (MCP) server that enables semantic search over markdown documentation. It uses ONNX Runtime for generating embeddings and SQLite with the vec0 extension for efficient vector storage and similarity search.

## Architecture Overview

The system consists of several key components:

### Parser Module

The parser module is responsible for reading markdown files and splitting them into manageable chunks. It uses a paragraph-based chunking strategy that respects document structure while maintaining a target chunk size. The chunking algorithm handles both English and Japanese text properly using rune-based counting.

### Embedder Module

The embedder module uses ONNX Runtime to generate vector embeddings from text chunks. It supports the multilingual-e5-small model which produces 384-dimensional vectors. The module includes proper tokenization and mean pooling to create high-quality embeddings.

### Vector Database

The vector database uses SQLite with the vec0 extension for storing and querying vectors. It maintains relationships between documents, chunks, and their embeddings using a normalized schema with proper foreign key constraints.

### Indexer Module

The indexer module orchestrates the entire indexing process. It reads markdown files, chunks them using the parser, generates embeddings using the embedder, and stores everything in the database using transactions for atomicity.

## Implementation Details

Each component is carefully designed to handle edge cases and provide robust error handling. The system uses stderr for logging and implements proper resource management with defer statements for cleanup.

## Performance Considerations

The indexing process aims for high throughput while maintaining reasonable memory usage. Batch processing is used where appropriate to minimize overhead.

## Conclusion

This document serves as both documentation and a test case for the chunking and indexing functionality. It contains enough content to be split into multiple chunks while demonstrating the system's capabilities.
