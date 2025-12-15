package vectordb

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SearchResult represents a single search result
type SearchResult struct {
	DocumentName string  `json:"document"`
	ChunkContent string  `json:"chunk"`
	Similarity   float64 `json:"similarity"`
	Position     int     `json:"position"`
}

// SearchFilter defines filtering options for search
type SearchFilter struct {
	// Directory filters results to documents under a specific directory path
	// Example: "docs/api" will match "docs/api/auth.md", "docs/api/users.md"
	Directory string

	// FilePattern filters results using glob pattern matching on filename
	// Example: "*.md" matches all markdown files
	// Example: "api-*.md" matches "api-auth.md", "api-users.md"
	FilePattern string
}

// Search performs vector similarity search using cosine distance
// Returns top-K most similar chunks to the query vector
func (db *DB) Search(queryVector []float32, topK int) ([]SearchResult, error) {
	return db.SearchWithFilter(queryVector, topK, nil)
}

// SearchWithFilter performs vector similarity search with optional filtering
// Filter is applied via SQL WHERE clause for efficiency
func (db *DB) SearchWithFilter(queryVector []float32, topK int, filter *SearchFilter) ([]SearchResult, error) {
	if len(queryVector) == 0 {
		return nil, fmt.Errorf("query vector is empty")
	}
	if topK <= 0 {
		return nil, fmt.Errorf("topK must be positive, got %d", topK)
	}

	// Serialize query vector to format expected by sqlite-vec
	queryBlob := serializeVector(queryVector)

	// Build query with optional filters
	// vec_distance_cosine returns distance where smaller values = more similar
	// Distance range: 0 (identical) to 2 (opposite direction)
	var queryBuilder strings.Builder
	args := []interface{}{queryBlob}

	queryBuilder.WriteString(`
		SELECT
			d.filename,
			c.content,
			c.position,
			vec_distance_cosine(v.embedding, ?) as distance
		FROM vec_chunks v
		JOIN chunks c ON v.rowid = c.id
		JOIN documents d ON c.document_id = d.id
	`)

	// Add WHERE clause if filters are specified
	whereClauses := []string{}
	if filter != nil {
		if filter.Directory != "" {
			// Normalize directory path and add trailing separator for prefix matching
			dir := strings.TrimSuffix(filter.Directory, "/")
			dir = strings.TrimSuffix(dir, string(filepath.Separator))
			// Match files directly in directory or in subdirectories
			whereClauses = append(whereClauses, "(d.filename LIKE ? OR d.filename LIKE ?)")
			args = append(args, dir+"/%", dir+string(filepath.Separator)+"%")
		}
		if filter.FilePattern != "" {
			// Convert glob pattern to SQL LIKE pattern
			// Pattern matches against the filename (basename) part, not the full path
			// Use %/pattern OR pattern to match both with and without directory prefix
			likePattern := globToLike(filter.FilePattern)
			whereClauses = append(whereClauses, "(d.filename LIKE ? OR d.filename LIKE ?)")
			args = append(args, "%/"+likePattern, likePattern)
		}
	}

	if len(whereClauses) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(whereClauses, " AND "))
	}

	queryBuilder.WriteString(" ORDER BY distance ASC LIMIT ?")
	args = append(args, topK)

	query := queryBuilder.String()

	rows, err := db.conn.Query(query, args...)
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

// globToLike converts a glob pattern to a SQL LIKE pattern
// Supports: * (any characters), ? (single character)
func globToLike(pattern string) string {
	// Escape SQL LIKE special characters first
	result := strings.ReplaceAll(pattern, "%", "\\%")
	result = strings.ReplaceAll(result, "_", "\\_")

	// Convert glob wildcards to LIKE wildcards
	result = strings.ReplaceAll(result, "*", "%")
	result = strings.ReplaceAll(result, "?", "_")

	return result
}
