package indexer

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

type Chunk struct {
	Content  string
	Position int
}

// GetContent returns the content of the chunk
func (c Chunk) GetContent() string {
	return c.Content
}

// GetPosition returns the position of the chunk
func (c Chunk) GetPosition() int {
	return c.Position
}

// ParseMarkdown parses a markdown file and splits into chunks
func ParseMarkdown(filepath string, chunkSize int) ([]Chunk, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read entire file
	var content strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Split into chunks
	chunks := splitIntoChunks(content.String(), chunkSize)

	// Create Chunk structs
	result := make([]Chunk, len(chunks))
	for i, c := range chunks {
		result[i] = Chunk{
			Content:  c,
			Position: i,
		}
	}

	return result, nil
}

// splitIntoChunks splits text into chunks of approximately chunkSize characters
func splitIntoChunks(content string, chunkSize int) []string {
	// Use rune count for character-based chunking
	if utf8.RuneCountInString(content) <= chunkSize {
		// If content is small enough, return as single chunk
		trimmed := strings.TrimSpace(content)
		if trimmed == "" {
			return []string{}
		}
		return []string{trimmed}
	}

	var chunks []string
	var currentChunk strings.Builder

	// Split by paragraphs (double newline)
	paragraphs := strings.Split(content, "\n\n")

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		currentLen := utf8.RuneCountInString(currentChunk.String())
		paraLen := utf8.RuneCountInString(para)

		// If adding this paragraph exceeds chunk size, start new chunk
		if currentLen > 0 && currentLen+paraLen+2 > chunkSize { // +2 for "\n\n"
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentLen = 0
		}

		// If single paragraph is too large, split it
		if paraLen > chunkSize {
			// Flush current chunk first
			if currentChunk.Len() > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
			}

			// Split by sentences or fixed size
			subChunks := splitLargeParagraph(para, chunkSize)
			chunks = append(chunks, subChunks...)
		} else {
			if currentChunk.Len() > 0 {
				currentChunk.WriteString("\n\n")
			}
			currentChunk.WriteString(para)
		}
	}

	// Add remaining chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// splitLargeParagraph splits a large paragraph into smaller chunks
func splitLargeParagraph(para string, chunkSize int) []string {
	var chunks []string

	// Convert to runes to handle multi-byte characters properly
	runes := []rune(para)

	for len(runes) > chunkSize {
		// Try to split at sentence boundary
		cutPoint := chunkSize

		// Search backwards from chunkSize to chunkSize/2 for a sentence boundary
		for i := chunkSize; i > chunkSize/2 && i < len(runes); i-- {
			r := runes[i]
			// Check for sentence boundaries (English and Japanese)
			if r == '.' || r == '!' || r == '?' || r == '\n' || r == '。' {
				cutPoint = i + 1
				break
			}
		}

		// Ensure cutPoint doesn't exceed runes length
		if cutPoint > len(runes) {
			cutPoint = len(runes)
		}

		// Convert runes back to string and add to chunks
		chunks = append(chunks, strings.TrimSpace(string(runes[:cutPoint])))
		runes = []rune(strings.TrimSpace(string(runes[cutPoint:])))
	}

	if len(runes) > 0 {
		chunks = append(chunks, string(runes))
	}

	return chunks
}
