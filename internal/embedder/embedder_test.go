package embedder

import (
	"testing"
)

func TestMockEmbedder_Embed(t *testing.T) {
	embedder := &MockEmbedder{}

	text := "Hello world"
	embedding, err := embedder.Embed(text)
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	// Verify embedding dimensions
	if len(embedding) != 384 {
		t.Errorf("Expected 384 dimensions, got %d", len(embedding))
	}

	// Verify embedding is not all zeros
	allZeros := true
	for _, v := range embedding {
		if v != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		t.Error("Embedding is all zeros")
	}

	// Verify deterministic: same input should give same output
	embedding2, err := embedder.Embed(text)
	if err != nil {
		t.Fatalf("Second embed failed: %v", err)
	}

	for i := range embedding {
		if embedding[i] != embedding2[i] {
			t.Errorf("Embedding not deterministic at index %d: %f vs %f", i, embedding[i], embedding2[i])
			break
		}
	}
}

func TestMockEmbedder_EmbedDifferentTexts(t *testing.T) {
	embedder := &MockEmbedder{}

	text1 := "Hello world"
	text2 := "Different text"

	embedding1, err := embedder.Embed(text1)
	if err != nil {
		t.Fatalf("Embed text1 failed: %v", err)
	}

	embedding2, err := embedder.Embed(text2)
	if err != nil {
		t.Fatalf("Embed text2 failed: %v", err)
	}

	// Different texts should produce different embeddings
	same := true
	for i := range embedding1 {
		if embedding1[i] != embedding2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("Different texts produced identical embeddings")
	}
}

func TestMockEmbedder_EmbedBatch(t *testing.T) {
	embedder := &MockEmbedder{}

	texts := []string{
		"First text",
		"Second text",
		"Third text",
	}

	embeddings, err := embedder.EmbedBatch(texts)
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}

	// Verify number of embeddings
	if len(embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	// Verify each embedding has correct dimensions
	for i, emb := range embeddings {
		if len(emb) != 384 {
			t.Errorf("Embedding %d has wrong dimensions: %d", i, len(emb))
		}
	}

	// Verify embeddings are different from each other
	for i := 0; i < len(embeddings)-1; i++ {
		same := true
		for j := range embeddings[i] {
			if embeddings[i][j] != embeddings[i+1][j] {
				same = false
				break
			}
		}
		if same {
			t.Errorf("Embeddings %d and %d are identical", i, i+1)
		}
	}
}

func TestMockEmbedder_EmbedBatchEmpty(t *testing.T) {
	embedder := &MockEmbedder{}

	embeddings, err := embedder.EmbedBatch([]string{})
	if err != nil {
		t.Fatalf("EmbedBatch with empty slice failed: %v", err)
	}

	if len(embeddings) != 0 {
		t.Errorf("Expected 0 embeddings, got %d", len(embeddings))
	}
}

func TestMockEmbedder_Close(t *testing.T) {
	embedder := &MockEmbedder{}

	err := embedder.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestMockEmbedder_EmbedEmptyString(t *testing.T) {
	embedder := &MockEmbedder{}

	embedding, err := embedder.Embed("")
	if err != nil {
		t.Fatalf("Embed empty string failed: %v", err)
	}

	if len(embedding) != 384 {
		t.Errorf("Expected 384 dimensions, got %d", len(embedding))
	}
}

func TestDetectDevice(t *testing.T) {
	tests := []struct {
		name          string
		deviceConfig  string
		fallback      bool
		expectsPanic  bool
	}{
		{"Auto CPU", "auto", true, false},
		{"Force CPU", "cpu", true, false},
		{"GPU with fallback", "gpu", true, false},
		{"Auto no fallback", "auto", false, false},
		{"CPU no fallback", "cpu", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := DetectDevice(tt.deviceConfig, tt.fallback)
			if device != CPU && device != GPU {
				t.Errorf("Invalid device returned: %v", device)
			}
		})
	}
}

func TestDetectDevice_Unknown(t *testing.T) {
	// Unknown device config should default to CPU
	device := DetectDevice("unknown", true)
	if device != CPU {
		t.Errorf("Expected CPU for unknown device config, got %v", device)
	}
}

func TestDeviceString(t *testing.T) {
	tests := []struct {
		device Device
		want   string
	}{
		{CPU, "CPU"},
		{GPU, "GPU"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.device.String()
			if got != tt.want {
				t.Errorf("Device.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
