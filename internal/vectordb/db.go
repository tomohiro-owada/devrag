package vectordb

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unsafe"
)

// ListDocuments returns a map of filename -> modified_at for all indexed documents
func (db *DB) ListDocuments() (map[string]time.Time, error) {
	rows, err := db.conn.Query("SELECT filename, modified_at FROM documents")
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	docs := make(map[string]time.Time)
	for rows.Next() {
		var filename string
		var modifiedAt time.Time
		if err := rows.Scan(&filename, &modifiedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		docs[filename] = modifiedAt
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return docs, nil
}

// DeleteDocument deletes a document and its chunks from the database
func (db *DB) DeleteDocument(filename string) error {
	// Get document ID first
	var docID int64
	err := db.conn.QueryRow("SELECT id FROM documents WHERE filename = ?", filename).Scan(&docID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("document not found: %s", filename)
		}
		return fmt.Errorf("failed to query document: %w", err)
	}

	// Delete document (chunks will be cascade deleted due to FOREIGN KEY)
	result, err := db.conn.Exec("DELETE FROM documents WHERE id = ?", docID)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("document not deleted: %s", filename)
	}

	return nil
}

// ChunkInterface defines the interface for chunk-like objects
type ChunkInterface interface {
	GetContent() string
	GetPosition() int
}

// InsertDocument inserts or updates a document and its chunks
func (db *DB) InsertDocument(filename string, modifiedAt time.Time, chunks []ChunkInterface, embeddings [][]float32) error {
	if len(chunks) != len(embeddings) {
		return fmt.Errorf("chunks count (%d) does not match embeddings count (%d)", len(chunks), len(embeddings))
	}

	// Begin transaction
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Will be no-op if tx.Commit() succeeds

	// Insert or replace document
	result, err := tx.Exec(
		"INSERT OR REPLACE INTO documents (filename, modified_at, indexed_at) VALUES (?, ?, CURRENT_TIMESTAMP)",
		filename, modifiedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	docID, err := result.LastInsertId()
	if err != nil {
		// If INSERT OR REPLACE updated an existing row, we need to get the document ID
		err = tx.QueryRow("SELECT id FROM documents WHERE filename = ?", filename).Scan(&docID)
		if err != nil {
			return fmt.Errorf("failed to get document ID: %w", err)
		}
	}

	// Delete existing chunks for this document (if any)
	// This is necessary when re-indexing
	_, err = tx.Exec("DELETE FROM chunks WHERE document_id = ?", docID)
	if err != nil {
		return fmt.Errorf("failed to delete old chunks: %w", err)
	}

	// Insert chunks and their vectors
	for i, chunk := range chunks {
		// Insert chunk
		result, err := tx.Exec(
			"INSERT INTO chunks (document_id, position, content) VALUES (?, ?, ?)",
			docID, chunk.GetPosition(), chunk.GetContent(),
		)
		if err != nil {
			return fmt.Errorf("failed to insert chunk %d: %w", i, err)
		}

		chunkID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get chunk ID for chunk %d: %w", i, err)
		}

		// Insert embedding into vec_chunks virtual table
		// vec0 expects vectors as a blob of float32 values
		embedding := embeddings[i]
		if len(embedding) == 0 {
			return fmt.Errorf("empty embedding for chunk %d", i)
		}

		// Convert []float32 to a format suitable for vec0
		// vec0 expects a serialized format - we'll insert directly as a blob
		vectorBlob := serializeVector(embedding)

		// vec0 table uses ROWID which we need to match with chunk_id
		_, err = tx.Exec(
			"INSERT INTO vec_chunks (rowid, embedding) VALUES (?, ?)",
			chunkID, vectorBlob,
		)
		if err != nil {
			return fmt.Errorf("failed to insert vector for chunk %d: %w", i, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// serializeVector converts a float32 slice to a byte slice for storage
func serializeVector(vec []float32) []byte {
	// Convert float32 slice to byte slice
	// Each float32 is 4 bytes
	result := make([]byte, len(vec)*4)
	for i, v := range vec {
		// Use IEEE 754 binary representation
		bits := *(*uint32)(unsafe.Pointer(&v))
		result[i*4] = byte(bits)
		result[i*4+1] = byte(bits >> 8)
		result[i*4+2] = byte(bits >> 16)
		result[i*4+3] = byte(bits >> 24)
	}
	return result
}

// CodeChunkInterface extends ChunkInterface with code-specific metadata
type CodeChunkInterface interface {
	ChunkInterface
	GetSymbolName() string
	GetSymbolType() string
	GetLanguage() string
	GetStartLine() int
	GetEndLine() int
	GetParentSymbol() string
	GetSignature() string
	GetEmbeddingText() string
}

// InsertCodeDocument inserts or updates a code document with metadata
func (db *DB) InsertCodeDocument(filename string, modifiedAt time.Time, chunks []CodeChunkInterface, embeddings [][]float32) error {
	if len(chunks) != len(embeddings) {
		return fmt.Errorf("chunks count (%d) does not match embeddings count (%d)", len(chunks), len(embeddings))
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert or replace document
	result, err := tx.Exec(
		"INSERT OR REPLACE INTO documents (filename, modified_at, indexed_at) VALUES (?, ?, CURRENT_TIMESTAMP)",
		filename, modifiedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	docID, err := result.LastInsertId()
	if err != nil {
		err = tx.QueryRow("SELECT id FROM documents WHERE filename = ?", filename).Scan(&docID)
		if err != nil {
			return fmt.Errorf("failed to get document ID: %w", err)
		}
	}

	// Delete existing chunks for this document
	_, err = tx.Exec("DELETE FROM chunks WHERE document_id = ?", docID)
	if err != nil {
		return fmt.Errorf("failed to delete old chunks: %w", err)
	}

	// Insert chunks, vectors, and metadata
	for i, chunk := range chunks {
		// Insert chunk
		result, err := tx.Exec(
			"INSERT INTO chunks (document_id, position, content) VALUES (?, ?, ?)",
			docID, chunk.GetPosition(), chunk.GetContent(),
		)
		if err != nil {
			return fmt.Errorf("failed to insert chunk %d: %w", i, err)
		}

		chunkID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get chunk ID for chunk %d: %w", i, err)
		}

		// Insert embedding
		embedding := embeddings[i]
		if len(embedding) == 0 {
			return fmt.Errorf("empty embedding for chunk %d", i)
		}

		vectorBlob := serializeVector(embedding)
		_, err = tx.Exec(
			"INSERT INTO vec_chunks (rowid, embedding) VALUES (?, ?)",
			chunkID, vectorBlob,
		)
		if err != nil {
			return fmt.Errorf("failed to insert vector for chunk %d: %w", i, err)
		}

		// Insert code metadata
		_, err = tx.Exec(`
			INSERT INTO code_metadata (chunk_id, symbol_name, symbol_type, language, start_line, end_line, parent_symbol, signature)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			chunkID,
			chunk.GetSymbolName(),
			chunk.GetSymbolType(),
			chunk.GetLanguage(),
			chunk.GetStartLine(),
			chunk.GetEndLine(),
			chunk.GetParentSymbol(),
			chunk.GetSignature(),
		)
		if err != nil {
			return fmt.Errorf("failed to insert code metadata for chunk %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CodeMetadata holds metadata for a code chunk
type CodeMetadata struct {
	ID           int64
	ChunkID      int64
	SymbolName   string
	SymbolType   string
	Language     string
	StartLine    int
	EndLine      int
	ParentSymbol string
	Signature    string
}

// GetCodeMetadata retrieves code metadata for a chunk
func (db *DB) GetCodeMetadata(chunkID int64) (*CodeMetadata, error) {
	var meta CodeMetadata
	err := db.conn.QueryRow(`
		SELECT id, chunk_id, symbol_name, symbol_type, language, start_line, end_line, parent_symbol, signature
		FROM code_metadata WHERE chunk_id = ?`, chunkID).Scan(
		&meta.ID, &meta.ChunkID, &meta.SymbolName, &meta.SymbolType, &meta.Language,
		&meta.StartLine, &meta.EndLine, &meta.ParentSymbol, &meta.Signature,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No metadata for this chunk (it's a markdown chunk)
		}
		return nil, fmt.Errorf("failed to get code metadata: %w", err)
	}
	return &meta, nil
}

// CodeRelation represents a relationship between code symbols
type CodeRelation struct {
	ID            int64
	SourceChunkID int64
	TargetChunkID *int64
	RelationType  string
	TargetName    string
	TargetFile    string
	Confidence    float64
	// Additional fields for search results
	SourceName string
	SourceFile string
}

// InsertRelations inserts code relations into the database
func (db *DB) InsertRelations(relations []CodeRelation) error {
	if len(relations) == 0 {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, rel := range relations {
		_, err := tx.Exec(`
			INSERT INTO code_relations (source_chunk_id, target_chunk_id, relation_type, target_name, target_file, confidence)
			VALUES (?, ?, ?, ?, ?, ?)`,
			rel.SourceChunkID, rel.TargetChunkID, rel.RelationType, rel.TargetName, rel.TargetFile, rel.Confidence,
		)
		if err != nil {
			return fmt.Errorf("failed to insert relation: %w", err)
		}
	}

	return tx.Commit()
}

// GetChunkIDBySymbol returns the chunk ID for a symbol in a given file
func (db *DB) GetChunkIDBySymbol(filename string, symbolName string) (int64, error) {
	var chunkID int64
	err := db.conn.QueryRow(`
		SELECT cm.chunk_id
		FROM code_metadata cm
		JOIN chunks c ON cm.chunk_id = c.id
		JOIN documents d ON c.document_id = d.id
		WHERE d.filename = ? AND cm.symbol_name = ?
		LIMIT 1`,
		filename, symbolName,
	).Scan(&chunkID)
	if err != nil {
		return 0, err
	}
	return chunkID, nil
}

// GetRelationsFrom returns relations where the given chunk is the source
func (db *DB) GetRelationsFrom(chunkID int64, relType string) ([]CodeRelation, error) {
	query := "SELECT id, source_chunk_id, target_chunk_id, relation_type, target_name, target_file, confidence FROM code_relations WHERE source_chunk_id = ?"
	args := []interface{}{chunkID}

	if relType != "" {
		query += " AND relation_type = ?"
		args = append(args, relType)
	}

	return db.queryRelations(query, args...)
}

// GetRelationsTo returns relations where the given chunk is the target
func (db *DB) GetRelationsTo(chunkID int64, relType string) ([]CodeRelation, error) {
	query := "SELECT id, source_chunk_id, target_chunk_id, relation_type, target_name, target_file, confidence FROM code_relations WHERE target_chunk_id = ?"
	args := []interface{}{chunkID}

	if relType != "" {
		query += " AND relation_type = ?"
		args = append(args, relType)
	}

	return db.queryRelations(query, args...)
}

// FindSymbolRelations finds all relations for a symbol by name
func (db *DB) FindSymbolRelations(symbolName string, direction string, relType string) ([]CodeRelation, error) {
	var query string
	var args []interface{}

	switch direction {
	case "incoming":
		// Find relations where this symbol is the target (who calls this symbol)
		query = `
			SELECT cr.id, cr.source_chunk_id, cr.target_chunk_id, cr.relation_type, cr.target_name, cr.target_file, cr.confidence,
			       cm.symbol_name as source_name, d.filename as source_file
			FROM code_relations cr
			JOIN code_metadata cm ON cr.source_chunk_id = cm.chunk_id
			JOIN chunks c ON cm.chunk_id = c.id
			JOIN documents d ON c.document_id = d.id
			WHERE cr.target_name = ?`
		args = []interface{}{symbolName}
	case "outgoing":
		// Find relations where this symbol is the source (what does this symbol call)
		query = `
			SELECT cr.id, cr.source_chunk_id, cr.target_chunk_id, cr.relation_type, cr.target_name, cr.target_file, cr.confidence,
			       cm.symbol_name as source_name, d.filename as source_file
			FROM code_relations cr
			JOIN code_metadata cm ON cr.source_chunk_id = cm.chunk_id
			JOIN chunks c ON cm.chunk_id = c.id
			JOIN documents d ON c.document_id = d.id
			WHERE cm.symbol_name = ?`
		args = []interface{}{symbolName}
	default: // "both"
		// Find both incoming and outgoing relations
		query = `
			SELECT cr.id, cr.source_chunk_id, cr.target_chunk_id, cr.relation_type, cr.target_name, cr.target_file, cr.confidence,
			       cm.symbol_name as source_name, d.filename as source_file
			FROM code_relations cr
			JOIN code_metadata cm ON cr.source_chunk_id = cm.chunk_id
			JOIN chunks c ON cm.chunk_id = c.id
			JOIN documents d ON c.document_id = d.id
			WHERE cr.target_name = ? OR cm.symbol_name = ?`
		args = []interface{}{symbolName, symbolName}
	}

	if relType != "" {
		query += " AND cr.relation_type = ?"
		args = append(args, relType)
	}

	return db.queryRelations(query, args...)
}

func (db *DB) queryRelations(query string, args ...interface{}) ([]CodeRelation, error) {
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query relations: %w", err)
	}
	defer rows.Close()

	var relations []CodeRelation
	for rows.Next() {
		var rel CodeRelation
		var targetChunkID sql.NullInt64
		err := rows.Scan(&rel.ID, &rel.SourceChunkID, &targetChunkID, &rel.RelationType, &rel.TargetName, &rel.TargetFile, &rel.Confidence, &rel.SourceName, &rel.SourceFile)
		if err != nil {
			return nil, fmt.Errorf("failed to scan relation: %w", err)
		}
		if targetChunkID.Valid {
			rel.TargetChunkID = &targetChunkID.Int64
		}
		relations = append(relations, rel)
	}

	return relations, rows.Err()
}

// WordMapping represents a word translation mapping
type WordMapping struct {
	ID             int64
	SourceWord     string
	TargetWord     string
	SourceLang     string
	Confidence     float64
	SourceDocument string
}

// InsertWordMapping inserts a word mapping into the dictionary
func (db *DB) InsertWordMapping(mapping WordMapping) error {
	_, err := db.conn.Exec(`
		INSERT OR IGNORE INTO word_mapping (source_word, target_word, source_lang, confidence, source_document)
		VALUES (?, ?, ?, ?, ?)`,
		mapping.SourceWord, mapping.TargetWord, mapping.SourceLang, mapping.Confidence, mapping.SourceDocument,
	)
	return err
}

// InsertWordMappings inserts multiple word mappings
func (db *DB) InsertWordMappings(mappings []WordMapping) error {
	if len(mappings) == 0 {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, m := range mappings {
		_, err := tx.Exec(`
			INSERT OR IGNORE INTO word_mapping (source_word, target_word, source_lang, confidence, source_document)
			VALUES (?, ?, ?, ?, ?)`,
			m.SourceWord, m.TargetWord, m.SourceLang, m.Confidence, m.SourceDocument,
		)
		if err != nil {
			return fmt.Errorf("failed to insert word mapping: %w", err)
		}
	}

	return tx.Commit()
}

// LookupWordMappings looks up target words for a source word
func (db *DB) LookupWordMappings(sourceWord string, sourceLang string) ([]string, error) {
	query := "SELECT target_word FROM word_mapping WHERE source_word = ?"
	args := []interface{}{sourceWord}

	if sourceLang != "" {
		query += " AND source_lang = ?"
		args = append(args, sourceLang)
	}

	query += " ORDER BY confidence DESC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup word mapping: %w", err)
	}
	defer rows.Close()

	var targets []string
	for rows.Next() {
		var target string
		if err := rows.Scan(&target); err != nil {
			return nil, fmt.Errorf("failed to scan target word: %w", err)
		}
		targets = append(targets, target)
	}

	return targets, rows.Err()
}

// GetWordMappingCount returns the number of word mappings
func (db *DB) GetWordMappingCount() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM word_mapping").Scan(&count)
	return count, err
}

// SearchSymbolsByKeywords searches code_metadata for symbols matching keywords
func (db *DB) SearchSymbolsByKeywords(keywords []string, limit int) ([]SearchResult, error) {
	if len(keywords) == 0 {
		return nil, nil
	}

	// Build query with LIKE conditions for each keyword
	query := `
		SELECT
			d.filename,
			c.content,
			c.position,
			c.id as chunk_id,
			0.0 as distance,
			cm.symbol_name,
			cm.symbol_type,
			cm.language,
			cm.start_line,
			cm.end_line,
			cm.parent_symbol,
			cm.signature
		FROM code_metadata cm
		JOIN chunks c ON cm.chunk_id = c.id
		JOIN documents d ON c.document_id = d.id
		WHERE `

	var conditions []string
	var args []interface{}
	for _, kw := range keywords {
		conditions = append(conditions, "LOWER(cm.symbol_name) LIKE ?")
		args = append(args, "%"+strings.ToLower(kw)+"%")
	}
	query += "(" + strings.Join(conditions, " OR ") + ")"
	query += " LIMIT ?"
	args = append(args, limit)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search symbols: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var distance float64
		var symbolName, symbolType, language, parentSymbol, signature sql.NullString
		var startLine, endLine sql.NullInt64

		err := rows.Scan(
			&r.DocumentName, &r.ChunkContent, &r.Position, &r.ChunkID, &distance,
			&symbolName, &symbolType, &language, &startLine, &endLine, &parentSymbol, &signature,
		)
		r.Similarity = 1.0 - distance // Convert distance to similarity
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		if symbolName.Valid {
			r.Metadata = &CodeMetadataResult{
				SymbolName:   symbolName.String,
				SymbolType:   symbolType.String,
				Language:     language.String,
				StartLine:    int(startLine.Int64),
				EndLine:      int(endLine.Int64),
				ParentSymbol: parentSymbol.String,
				Signature:    signature.String,
			}
		}

		results = append(results, r)
	}

	return results, rows.Err()
}
