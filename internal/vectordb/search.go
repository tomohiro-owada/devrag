package vectordb

import (
	"fmt"
)

// SearchResult represents a single search result
type SearchResult struct {
	DocumentName string
	ChunkContent string
	Similarity   float64
	Position     int
}

// Search performs vector similarity search using cosine distance
// Returns top-K most similar chunks to the query vector
func (db *DB) Search(queryVector []float32, topK int) ([]SearchResult, error) {
	if len(queryVector) == 0 {
		return nil, fmt.Errorf("query vector is empty")
	}
	if topK <= 0 {
		return nil, fmt.Errorf("topK must be positive, got %d", topK)
	}

	// Serialize query vector to format expected by sqlite-vec
	queryBlob := serializeVector(queryVector)

	// Execute similarity search query
	// vec_distance_cosine returns distance where smaller values = more similar
	// Distance range: 0 (identical) to 2 (opposite direction)
	query := `
		SELECT
			d.filename,
			c.content,
			c.position,
			vec_distance_cosine(v.embedding, ?) as distance
		FROM vec_chunks v
		JOIN chunks c ON v.rowid = c.id
		JOIN documents d ON c.document_id = d.id
		ORDER BY distance ASC
		LIMIT ?
	`

	rows, err := db.conn.Query(query, queryBlob, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	// Collect results
	results := make([]SearchResult, 0, topK)
	for rows.Next() {
		var result SearchResult
		var distance float64

		err := rows.Scan(&result.DocumentName, &result.ChunkContent, &result.Position, &distance)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result row: %w", err)
		}

		// Convert distance to similarity score
		// Cosine distance: 0 = same direction, 2 = opposite direction
		// Similarity: 1 - (distance/2) gives us a 0-1 range where 1 = identical
		result.Similarity = 1.0 - (distance / 2.0)

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating result rows: %w", err)
	}

	return results, nil
}
