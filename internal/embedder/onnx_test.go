package embedder

import (
	"testing"
	"time"
)

func TestCreateSessionWithTimeout_Timeout(t *testing.T) {
	// Save and restore the original timeout
	orig := CoreMLTimeout
	CoreMLTimeout = 100 * time.Millisecond
	defer func() { CoreMLTimeout = orig }()

	// Use a non-existent model path to trigger an error (not a hang),
	// but we can at least verify the timeout mechanism works.
	// A real hang test would require mocking ONNX Runtime internals.
	_, err := createSessionWithTimeout(
		"/nonexistent/model.onnx",
		[]string{"input_ids"},
		[]string{"output"},
		nil, // nil options will cause ONNX to fail fast
		true,
	)

	// We expect either an error from ONNX or a timeout — either way, not a hang
	if err == nil {
		t.Error("Expected error for non-existent model path")
	}
}

func TestCoreMLTimeout_Default(t *testing.T) {
	if CoreMLTimeout != 60*time.Second {
		t.Errorf("Default CoreMLTimeout should be 60s, got %v", CoreMLTimeout)
	}
}

func TestMockEmbedder_UnicodeHeavyText(t *testing.T) {
	// Issue #18: Verify that embedding works with Unicode-heavy content
	// MockEmbedder doesn't use the tokenizer, but this tests the interface contract
	emb := &MockEmbedder{}

	tests := []struct {
		name string
		text string
	}{
		{
			"Box drawing table",
			"┌──────────┐\n│  Header  │\n└──────────┘",
		},
		{
			"Japanese with box drawing",
			"## アーキテクチャ\n┌─────────┬─────────┐\n│ サービス │ ポート  │\n└─────────┴─────────┘",
		},
		{
			"Emoji + Japanese dense",
			"🎉 完了！📊 レポート生成 ✅ テスト通過 🚀 デプロイ完了",
		},
		{
			"Block elements progress bar",
			"進捗: ▓▓▓▓▓▓░░░░ 60%",
		},
		{
			"Braille spinner",
			"⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏ Loading...",
		},
		{
			"Mixed complex content",
			"# Infrastructure\n\n```\n┌─────┐     ┌─────┐\n│ API │────▶│ DB  │\n└─────┘     └─────┘\n```\n\n■ ステータス: 正常\n□ メンテナンス: なし",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedding, err := emb.Embed(tt.text)
			if err != nil {
				t.Fatalf("Embed failed: %v", err)
			}
			if len(embedding) != 384 {
				t.Errorf("Expected 384 dimensions, got %d", len(embedding))
			}
		})
	}
}

func TestMockEmbedder_EmbedBatch_UnicodeHeavy(t *testing.T) {
	emb := &MockEmbedder{}

	texts := []string{
		"┌──────┐ Normal text",
		"日本語テキスト with ■ shapes",
		"▓▓▓░░░ progress",
		"Regular ASCII text",
	}

	embeddings, err := emb.EmbedBatch(texts)
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	for i, emb := range embeddings {
		if len(emb) != 384 {
			t.Errorf("Embedding %d has %d dimensions, want 384", i, len(emb))
		}
	}
}

func TestMeanPooling(t *testing.T) {
	// Simple test: 2 tokens, 3-dimensional hidden states
	hiddenStates := []float32{
		1.0, 2.0, 3.0, // token 0
		4.0, 5.0, 6.0, // token 1
	}
	attentionMask := []int64{1, 1}
	seqLength := 2
	hiddenSize := 3

	result := meanPooling(hiddenStates, attentionMask, seqLength, hiddenSize)

	// Expected: mean of [1,2,3] and [4,5,6] = [2.5, 3.5, 4.5]
	expected := []float32{2.5, 3.5, 4.5}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("meanPooling[%d] = %f, want %f", i, v, expected[i])
		}
	}
}

func TestMeanPooling_WithPadding(t *testing.T) {
	// 3 tokens but only first 2 are real (attention_mask = [1, 1, 0])
	hiddenStates := []float32{
		1.0, 2.0, // token 0
		3.0, 4.0, // token 1
		0.0, 0.0, // token 2 (padding)
	}
	attentionMask := []int64{1, 1, 0}
	seqLength := 3
	hiddenSize := 2

	result := meanPooling(hiddenStates, attentionMask, seqLength, hiddenSize)

	// Expected: mean of [1,2] and [3,4] only = [2.0, 3.0]
	expected := []float32{2.0, 3.0}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("meanPooling[%d] = %f, want %f", i, v, expected[i])
		}
	}
}

func TestNormalize(t *testing.T) {
	vec := []float32{3.0, 4.0} // L2 norm = 5.0
	result := normalize(vec)

	// Expected: [3/5, 4/5] = [0.6, 0.8]
	if result[0] < 0.59 || result[0] > 0.61 {
		t.Errorf("normalize[0] = %f, want ~0.6", result[0])
	}
	if result[1] < 0.79 || result[1] > 0.81 {
		t.Errorf("normalize[1] = %f, want ~0.8", result[1])
	}
}

func TestNormalize_ZeroVector(t *testing.T) {
	vec := []float32{0.0, 0.0, 0.0}
	result := normalize(vec)

	for i, v := range result {
		if v != 0.0 {
			t.Errorf("normalize[%d] = %f, want 0.0 for zero vector", i, v)
		}
	}
}
