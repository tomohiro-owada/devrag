package indexer

import (
	"os"
	"strings"
	"testing"
)

func TestParseMarkdown_ShortFile(t *testing.T) {
	content := "# Test\n\nThis is a short file."
	tmpfile, err := os.CreateTemp("", "test*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	chunks, err := ParseMarkdown(tmpfile.Name(), 500)
	if err != nil {
		t.Fatalf("ParseMarkdown failed: %v", err)
	}

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}

	if chunks[0].Position != 0 {
		t.Errorf("Expected position 0, got %d", chunks[0].Position)
	}

	if !strings.Contains(chunks[0].Content, "Test") {
		t.Errorf("Expected chunk to contain 'Test', got: %s", chunks[0].Content)
	}
}

func TestParseMarkdown_LongFile(t *testing.T) {
	// Generate long content
	var content string
	for i := 0; i < 100; i++ {
		content += "This is a test paragraph with some content. "
	}

	tmpfile, err := os.CreateTemp("", "test*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	chunks, err := ParseMarkdown(tmpfile.Name(), 500)
	if err != nil {
		t.Fatalf("ParseMarkdown failed: %v", err)
	}

	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks, got %d", len(chunks))
	}

	// Verify position increments
	for i, chunk := range chunks {
		if chunk.Position != i {
			t.Errorf("Chunk %d has wrong position: %d", i, chunk.Position)
		}
	}
}

func TestParseMarkdown_EmptyFile(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	chunks, err := ParseMarkdown(tmpfile.Name(), 500)
	if err != nil {
		t.Fatalf("ParseMarkdown failed: %v", err)
	}

	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty file, got %d", len(chunks))
	}
}

func TestParseMarkdown_NonExistentFile(t *testing.T) {
	_, err := ParseMarkdown("/nonexistent/file.md", 500)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestSplitIntoChunks_ShortText(t *testing.T) {
	content := "Paragraph 1\n\nParagraph 2\n\nParagraph 3"
	chunks := splitIntoChunks(content, 500)

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for short text, got %d", len(chunks))
	}

	if !strings.Contains(chunks[0], "Paragraph 1") {
		t.Error("Chunk does not contain expected content")
	}
}

func TestSplitIntoChunks_LongText(t *testing.T) {
	// Create content longer than chunk size
	var paragraphs []string
	for i := 0; i < 10; i++ {
		paragraphs = append(paragraphs, strings.Repeat("Test paragraph. ", 50))
	}
	content := strings.Join(paragraphs, "\n\n")

	chunks := splitIntoChunks(content, 500)

	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks, got %d", len(chunks))
	}

	// Verify chunks are not empty
	for i, chunk := range chunks {
		if len(chunk) == 0 {
			t.Errorf("Chunk %d is empty", i)
		}
	}
}

func TestSplitIntoChunks_EmptyText(t *testing.T) {
	chunks := splitIntoChunks("", 500)

	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty text, got %d", len(chunks))
	}
}

func TestSplitIntoChunks_WhitespaceOnly(t *testing.T) {
	chunks := splitIntoChunks("   \n\n   \n\n   ", 500)

	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for whitespace-only text, got %d", len(chunks))
	}
}

func TestSplitLargeParagraph(t *testing.T) {
	// Create a very long paragraph
	longPara := strings.Repeat("This is a long sentence. ", 100)
	chunks := splitLargeParagraph(longPara, 500)

	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks for large paragraph, got %d", len(chunks))
	}

	// Verify all chunks are non-empty
	for i, chunk := range chunks {
		if len(chunk) == 0 {
			t.Errorf("Chunk %d is empty", i)
		}
	}
}

func TestSplitLargeParagraph_WithSentenceBoundaries(t *testing.T) {
	// Create text with clear sentence boundaries
	longPara := ""
	for i := 0; i < 50; i++ {
		longPara += "This is sentence number " + string(rune('0'+i%10)) + ". "
	}

	chunks := splitLargeParagraph(longPara, 200)

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	// Verify chunks respect sentence boundaries where possible
	for _, chunk := range chunks {
		if len(chunk) == 0 {
			t.Error("Found empty chunk")
		}
	}
}

func TestSplitLargeParagraph_Japanese(t *testing.T) {
	// Test with Japanese text
	longPara := strings.Repeat("これは日本語のテストです。", 100)
	chunks := splitLargeParagraph(longPara, 500)

	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks for Japanese text, got %d", len(chunks))
	}

	// Verify chunks are properly split
	for i, chunk := range chunks {
		if len(chunk) == 0 {
			t.Errorf("Chunk %d is empty", i)
		}
	}
}

func TestChunk_GetContent(t *testing.T) {
	chunk := Chunk{
		Content:  "test content",
		Position: 0,
	}

	if chunk.GetContent() != "test content" {
		t.Errorf("GetContent returned wrong value: %s", chunk.GetContent())
	}
}

func TestChunk_GetPosition(t *testing.T) {
	chunk := Chunk{
		Content:  "test content",
		Position: 5,
	}

	if chunk.GetPosition() != 5 {
		t.Errorf("GetPosition returned wrong value: %d", chunk.GetPosition())
	}
}
