package embedder

import "crypto/sha256"

// Embedder interface for text embedding
type Embedder interface {
	// Embed embeds a single text string into a vector
	Embed(text string) ([]float32, error)

	// EmbedBatch embeds multiple text strings into vectors
	EmbedBatch(texts []string) ([][]float32, error)

	// Close releases resources used by the embedder
	Close() error
}

// MockEmbedder is a simple embedder for testing purposes
// It generates deterministic embeddings based on text hash
type MockEmbedder struct{}

// Embed generates a simple mock embedding (384 dimensions)
func (m *MockEmbedder) Embed(text string) ([]float32, error) {
	// Generate a deterministic embedding based on text hash
	hash := sha256.Sum256([]byte(text))

	// Create 384-dimensional vector
	embedding := make([]float32, 384)
	for i := 0; i < 384; i++ {
		// Use hash bytes to generate pseudo-random values
		embedding[i] = float32(hash[i%32]) / 255.0
	}

	// Normalize the vector (simple L2 normalization)
	var norm float32
	for _, v := range embedding {
		norm += v * v
	}
	if norm > 0 {
		norm = 1.0 / float32(norm)
		for i := range embedding {
			embedding[i] *= norm
		}
	}

	return embedding, nil
}

// EmbedBatch embeds multiple texts
func (m *MockEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := m.Embed(text)
		if err != nil {
			return nil, err
		}
		results[i] = embedding
	}
	return results, nil
}

// Close does nothing for mock embedder
func (m *MockEmbedder) Close() error {
	return nil
}
