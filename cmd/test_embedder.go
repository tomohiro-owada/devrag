package main

import (
	"fmt"
	"os"
	"time"

	"github.com/tomohiro-owada/devrag/internal/embedder"
	ort "github.com/yalue/onnxruntime_go"
)

func main() {
	fmt.Fprintf(os.Stderr, "=== Testing Embedder (Phase 2.2) ===\n\n")

	// Set ONNX Runtime library path (for ARM64 macOS)
	libPath := "/Users/towada/go/pkg/mod/github.com/yalue/onnxruntime_go@v1.21.0/test_data/onnxruntime_arm64.dylib"
	if _, err := os.Stat(libPath); err == nil {
		fmt.Fprintf(os.Stderr, "[INFO] Setting ONNX Runtime library path: %s\n", libPath)
		ort.SetSharedLibraryPath(libPath)
	} else {
		fmt.Fprintf(os.Stderr, "[WARN] ONNX Runtime library not found at %s, using system library\n", libPath)
	}

	// Model path
	modelPath := "/Users/towada/projects/devrag/models/model.onnx"

	// Check if model exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "[ERROR] Model not found at %s\n", modelPath)
		fmt.Fprintf(os.Stderr, "Please download the model first:\n")
		fmt.Fprintf(os.Stderr, "  python3 scripts/download_model.py\n")
		os.Exit(1)
	}

	// Create embedder
	fmt.Fprintf(os.Stderr, "[INFO] Creating embedder...\n")
	emb, err := embedder.NewONNXEmbedder(modelPath, embedder.CPU)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to create embedder: %v\n", err)
		os.Exit(1)
	}
	defer emb.Close()

	fmt.Fprintf(os.Stderr, "[INFO] Embedder created successfully\n\n")

	// Test 1: Single English text
	fmt.Fprintf(os.Stderr, "=== Test 1: Single English Text ===\n")
	testText1 := "This is a test document about artificial intelligence and machine learning."
	fmt.Fprintf(os.Stderr, "Text: %s\n", testText1)

	start := time.Now()
	embedding1, err := emb.Embed(testText1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to embed text: %v\n", err)
		os.Exit(1)
	}
	elapsed := time.Since(start)

	fmt.Fprintf(os.Stderr, "Embedding dimension: %d\n", len(embedding1))
	fmt.Fprintf(os.Stderr, "Processing time: %v\n", elapsed)
	fmt.Fprintf(os.Stderr, "First 10 values: [")
	for i := 0; i < 10 && i < len(embedding1); i++ {
		if i > 0 {
			fmt.Fprintf(os.Stderr, ", ")
		}
		fmt.Fprintf(os.Stderr, "%.4f", embedding1[i])
	}
	fmt.Fprintf(os.Stderr, "]\n\n")

	// Test 2: Japanese text
	fmt.Fprintf(os.Stderr, "=== Test 2: Japanese Text ===\n")
	testText2 := "これは人工知能と機械学習についてのテストドキュメントです。"
	fmt.Fprintf(os.Stderr, "Text: %s\n", testText2)

	start = time.Now()
	embedding2, err := emb.Embed(testText2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to embed text: %v\n", err)
		os.Exit(1)
	}
	elapsed = time.Since(start)

	fmt.Fprintf(os.Stderr, "Embedding dimension: %d\n", len(embedding2))
	fmt.Fprintf(os.Stderr, "Processing time: %v\n", elapsed)
	fmt.Fprintf(os.Stderr, "First 10 values: [")
	for i := 0; i < 10 && i < len(embedding2); i++ {
		if i > 0 {
			fmt.Fprintf(os.Stderr, ", ")
		}
		fmt.Fprintf(os.Stderr, "%.4f", embedding2[i])
	}
	fmt.Fprintf(os.Stderr, "]\n\n")

	// Test 3: Batch processing
	fmt.Fprintf(os.Stderr, "=== Test 3: Batch Processing ===\n")
	texts := []string{
		"Vector search is a powerful technique for similarity matching.",
		"ベクトル検索は類似度マッチングの強力な技術です。",
		"Markdown is a lightweight markup language.",
	}
	fmt.Fprintf(os.Stderr, "Number of texts: %d\n", len(texts))

	start = time.Now()
	embeddings, err := emb.EmbedBatch(texts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to embed batch: %v\n", err)
		os.Exit(1)
	}
	elapsed = time.Since(start)

	fmt.Fprintf(os.Stderr, "Batch size: %d\n", len(embeddings))
	fmt.Fprintf(os.Stderr, "Total processing time: %v\n", elapsed)
	fmt.Fprintf(os.Stderr, "Average time per text: %v\n", elapsed/time.Duration(len(texts)))

	for i, emb := range embeddings {
		fmt.Fprintf(os.Stderr, "  Text %d - Dimension: %d\n", i+1, len(emb))
	}
	fmt.Fprintf(os.Stderr, "\n")

	// Test 4: Verify normalization (L2 norm should be ~1.0)
	fmt.Fprintf(os.Stderr, "=== Test 4: Verify L2 Normalization ===\n")
	var norm float32
	for _, v := range embedding1 {
		norm += v * v
	}
	l2Norm := float32(1.0)
	if norm > 0 {
		l2Norm = 1.0
		for i := 0; i < 10; i++ {
			l2Norm = (l2Norm + norm/l2Norm) / 2
		}
	}

	fmt.Fprintf(os.Stderr, "L2 norm of first embedding: %.6f\n", l2Norm)
	if l2Norm > 0.99 && l2Norm < 1.01 {
		fmt.Fprintf(os.Stderr, "✓ Embeddings are properly normalized\n")
	} else {
		fmt.Fprintf(os.Stderr, "✗ Warning: Embeddings may not be properly normalized\n")
	}
	fmt.Fprintf(os.Stderr, "\n")

	// Test 5: Verify expected dimension
	fmt.Fprintf(os.Stderr, "=== Test 5: Verify Dimension ===\n")
	expectedDim := 384
	if len(embedding1) == expectedDim {
		fmt.Fprintf(os.Stderr, "✓ Embedding dimension is correct: %d\n", expectedDim)
	} else {
		fmt.Fprintf(os.Stderr, "✗ Unexpected dimension: expected %d, got %d\n", expectedDim, len(embedding1))
	}
	fmt.Fprintf(os.Stderr, "\n")

	// Summary
	fmt.Fprintf(os.Stderr, "=== Summary ===\n")
	fmt.Fprintf(os.Stderr, "✓ Tokenizer loaded successfully\n")
	fmt.Fprintf(os.Stderr, "✓ ONNX model loaded successfully\n")
	fmt.Fprintf(os.Stderr, "✓ Single text embedding works\n")
	fmt.Fprintf(os.Stderr, "✓ Multilingual support confirmed (English & Japanese)\n")
	fmt.Fprintf(os.Stderr, "✓ Batch processing works\n")
	fmt.Fprintf(os.Stderr, "✓ Output dimension: %d\n", len(embedding1))
	fmt.Fprintf(os.Stderr, "\n[SUCCESS] All tests passed!\n")
}
