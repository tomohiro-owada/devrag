package vectordb

const schemaSQL = `
CREATE TABLE IF NOT EXISTS documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename TEXT NOT NULL UNIQUE,
    indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_filename ON documents(filename);

CREATE TABLE IF NOT EXISTS chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    document_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    content TEXT NOT NULL,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_document_id ON chunks(document_id);

CREATE VIRTUAL TABLE IF NOT EXISTS vec_chunks USING vec0(
    embedding FLOAT[384]
);

CREATE TABLE IF NOT EXISTS code_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chunk_id INTEGER NOT NULL UNIQUE,
    symbol_name TEXT,
    symbol_type TEXT NOT NULL,
    language TEXT NOT NULL,
    start_line INTEGER,
    end_line INTEGER,
    parent_symbol TEXT,
    signature TEXT,
    FOREIGN KEY (chunk_id) REFERENCES chunks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_code_symbol ON code_metadata(symbol_name);
CREATE INDEX IF NOT EXISTS idx_code_language ON code_metadata(language);
CREATE INDEX IF NOT EXISTS idx_code_type ON code_metadata(symbol_type);

CREATE TABLE IF NOT EXISTS code_relations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_chunk_id INTEGER NOT NULL,
    target_chunk_id INTEGER,
    relation_type TEXT NOT NULL,
    target_name TEXT NOT NULL,
    target_file TEXT,
    confidence REAL DEFAULT 1.0,
    FOREIGN KEY (source_chunk_id) REFERENCES chunks(id) ON DELETE CASCADE,
    FOREIGN KEY (target_chunk_id) REFERENCES chunks(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_rel_source ON code_relations(source_chunk_id);
CREATE INDEX IF NOT EXISTS idx_rel_target ON code_relations(target_chunk_id);
CREATE INDEX IF NOT EXISTS idx_rel_type ON code_relations(relation_type);
CREATE INDEX IF NOT EXISTS idx_rel_name ON code_relations(target_name);

CREATE TABLE IF NOT EXISTS word_mapping (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_word TEXT NOT NULL,
    target_word TEXT NOT NULL,
    source_lang TEXT DEFAULT 'ja',
    confidence REAL DEFAULT 1.0,
    source_document TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_word, target_word, source_lang)
);

CREATE INDEX IF NOT EXISTS idx_word_source ON word_mapping(source_word);
CREATE INDEX IF NOT EXISTS idx_word_target ON word_mapping(target_word);
CREATE INDEX IF NOT EXISTS idx_word_lang ON word_mapping(source_lang);
`
