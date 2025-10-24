package vectordb

import (
	"database/sql"
	"fmt"
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
