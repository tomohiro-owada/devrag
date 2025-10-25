package main

import (
	"fmt"
	"os"

	"github.com/tomohiro-owada/devrag/internal/indexer"
)

func main() {
	testFiles := []string{
		"test_data/test_short.md",
		"test_data/test_long.md",
		"test_data/test_mixed.md",
	}

	for _, filepath := range testFiles {
		fmt.Printf("\n=== Testing: %s ===\n", filepath)

		chunks, err := indexer.ParseMarkdown(filepath, 500)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", filepath, err)
			continue
		}

		fmt.Printf("Total chunks: %d\n\n", len(chunks))

		for i, chunk := range chunks {
			runeCount := len([]rune(chunk.Content))
			fmt.Printf("--- Chunk %d (Position: %d, Bytes: %d, Runes: %d) ---\n", i, chunk.Position, len(chunk.Content), runeCount)
			// Show first 200 chars of content
			runes := []rune(chunk.Content)
			preview := string(runes)
			if len(runes) > 200 {
				preview = string(runes[:200]) + "..."
			}
			fmt.Printf("%s\n\n", preview)
		}
	}
}
