package main

import (
	"fmt"
	"os"
	"time"

	"github.com/tomohiro-owada/devrag/internal/config"
	"github.com/tomohiro-owada/devrag/internal/indexer"
	"github.com/tomohiro-owada/devrag/internal/vectordb"
)

type PlaceholderEmbedder struct {
	dimensions int
	callCount  int
}

func (e *PlaceholderEmbedder) Embed(text string) ([]float32, error) {
	e.callCount++
	vec := make([]float32, e.dimensions)
	for i := range vec {
		vec[i] = float32((len(text)+i)%100) / 100.0
	}
	return vec, nil
}

func (e *PlaceholderEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := e.Embed(text)
		if err != nil {
			return nil, err
		}
		results[i] = emb
	}
	return results, nil
}

func (e *PlaceholderEmbedder) Close() error {
	return nil
}

func main() {
	fmt.Printf("=== Indexing Performance Test ===\n\n")

	// Setup
	cfg, _ := config.Load()
	cfg.DBPath = "./benchmark.db"
	cfg.ChunkSize = 500

	db, err := vectordb.Init(cfg.DBPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	emb := &PlaceholderEmbedder{dimensions: 384}
	idx := indexer.NewIndexer(db, emb, cfg)

	// Test 1: Single small document
	fmt.Println("[Test 1] Single small document")
	content1 := generateContent(300) // ~300 chars
	testFile1 := "./bench_small.md"
	os.WriteFile(testFile1, []byte(content1), 0644)
	defer os.Remove(testFile1)

	start := time.Now()
	idx.IndexFile(testFile1)
	t1 := time.Since(start)
	fmt.Printf("  Time: %v\n\n", t1)

	// Test 2: Single medium document
	fmt.Println("[Test 2] Single medium document")
	content2 := generateContent(2000) // ~2KB
	testFile2 := "./bench_medium.md"
	os.WriteFile(testFile2, []byte(content2), 0644)
	defer os.Remove(testFile2)

	start = time.Now()
	idx.IndexFile(testFile2)
	t2 := time.Since(start)
	fmt.Printf("  Time: %v\n\n", t2)

	// Test 3: Single large document
	fmt.Println("[Test 3] Single large document")
	content3 := generateContent(10000) // ~10KB
	testFile3 := "./bench_large.md"
	os.WriteFile(testFile3, []byte(content3), 0644)
	defer os.Remove(testFile3)

	start = time.Now()
	idx.IndexFile(testFile3)
	t3 := time.Since(start)
	fmt.Printf("  Time: %v\n\n", t3)

	// Statistics
	docs, _ := db.ListDocuments()

	fmt.Println("=== Summary ===")
	fmt.Printf("Total documents indexed: %d\n", len(docs))
	fmt.Printf("Total embeddings generated: %d\n", emb.callCount)
	fmt.Printf("\nPerformance:\n")
	fmt.Printf("  Small doc:  %v (%.2f ms)\n", t1, float64(t1.Microseconds())/1000.0)
	fmt.Printf("  Medium doc: %v (%.2f ms)\n", t2, float64(t2.Microseconds())/1000.0)
	fmt.Printf("  Large doc:  %v (%.2f ms)\n", t3, float64(t3.Microseconds())/1000.0)

	// Calculate throughput
	totalTime := t1 + t2 + t3
	chunksPerSec := float64(emb.callCount) / totalTime.Seconds()
	fmt.Printf("\nThroughput: %.0f chunks/sec\n", chunksPerSec)
}

func generateContent(targetSize int) string {
	var content string
	content += "# Performance Test Document\n\n"

	for i := 1; len(content) < targetSize; i++ {
		content += fmt.Sprintf("## Section %d\n\n", i)
		content += "This is a test paragraph with some content to generate a document of appropriate size. "
		content += "It contains multiple sentences to make the chunking more realistic. "
		content += "The content is repetitive but that's acceptable for performance testing.\n\n"
	}

	return content
}
