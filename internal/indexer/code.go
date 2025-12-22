package indexer

import (
	"context"
	"fmt"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// CodeParser parses source code files using Tree-sitter
type CodeParser struct {
	parser *sitter.Parser
}

// NewCodeParser creates a new CodeParser
func NewCodeParser() *CodeParser {
	return &CodeParser{
		parser: sitter.NewParser(),
	}
}

// ParseCodeFile parses a source code file and returns code chunks
func (cp *CodeParser) ParseCodeFile(filePath string) ([]CodeChunk, error) {
	langName, config, ok := DetectLanguage(filePath)
	if !ok {
		return nil, fmt.Errorf("unsupported language for file: %s", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	cp.parser.SetLanguage(config.Language)

	tree, err := cp.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}
	defer tree.Close()

	return cp.extractSymbols(tree.RootNode(), content, langName, config)
}

// ParseCodeContent parses source code content directly (useful for testing)
func (cp *CodeParser) ParseCodeContent(content []byte, langName string) ([]CodeChunk, error) {
	config, ok := SupportedLanguages[langName]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", langName)
	}

	cp.parser.SetLanguage(config.Language)

	tree, err := cp.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}
	defer tree.Close()

	return cp.extractSymbols(tree.RootNode(), content, langName, config)
}

func (cp *CodeParser) extractSymbols(root *sitter.Node, content []byte, langName string, config *LanguageConfig) ([]CodeChunk, error) {
	var chunks []CodeChunk
	position := 0
	seen := make(map[string]bool)

	for _, symbolQuery := range config.Queries {
		q, err := sitter.NewQuery([]byte(symbolQuery.Pattern), config.Language)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to compile query for %s: %v\n", symbolQuery.SymbolType, err)
			continue
		}

		cursor := sitter.NewQueryCursor()
		cursor.Exec(q, root)

		for {
			match, ok := cursor.NextMatch()
			if !ok {
				break
			}

			chunk := cp.createChunkFromMatch(match, q, content, langName, symbolQuery.SymbolType, position, seen)
			if chunk != nil {
				chunks = append(chunks, *chunk)
				position++
			}
		}

		cursor.Close()
		q.Close()
	}

	return chunks, nil
}

func (cp *CodeParser) createChunkFromMatch(match *sitter.QueryMatch, query *sitter.Query, content []byte, langName string, symbolType SymbolType, position int, seen map[string]bool) *CodeChunk {
	var symbolName string
	var nodeContent string
	var startLine, endLine uint32
	var startByte, endByte uint32

	for _, capture := range match.Captures {
		captureName := query.CaptureNameForId(capture.Index)

		switch captureName {
		case "name":
			symbolName = capture.Node.Content(content)
		case "function", "method", "class", "struct", "interface":
			nodeContent = capture.Node.Content(content)
			startLine = capture.Node.StartPoint().Row + 1
			endLine = capture.Node.EndPoint().Row + 1
			startByte = capture.Node.StartByte()
			endByte = capture.Node.EndByte()
		}
	}

	if nodeContent == "" {
		return nil
	}

	key := fmt.Sprintf("%d-%d", startByte, endByte)
	if seen[key] {
		return nil
	}
	seen[key] = true

	signature := extractSignature(nodeContent, symbolType, langName)

	return &CodeChunk{
		Content:    nodeContent,
		Position:   position,
		SymbolName: symbolName,
		SymbolType: symbolType,
		Language:   langName,
		StartLine:  int(startLine),
		EndLine:    int(endLine),
		Signature:  signature,
	}
}

func extractSignature(content string, symbolType SymbolType, langName string) string {
	if symbolType != SymbolTypeFunction && symbolType != SymbolTypeMethod {
		return ""
	}

	lines := strings.SplitN(content, "\n", 2)
	if len(lines) == 0 {
		return ""
	}

	firstLine := strings.TrimSpace(lines[0])

	if langName == "go" {
		return strings.TrimSpace(strings.TrimSuffix(firstLine, "{"))
	}

	if langName == "python" {
		if strings.HasPrefix(firstLine, "def ") || strings.HasPrefix(firstLine, "async def ") {
			return strings.TrimSpace(strings.TrimSuffix(firstLine, ":"))
		}
	}

	if langName == "typescript" || langName == "javascript" {
		if strings.HasPrefix(firstLine, "function ") {
			if idx := strings.Index(firstLine, "{"); idx > 0 {
				return strings.TrimSpace(firstLine[:idx])
			}
			return firstLine
		}
		if strings.Contains(firstLine, "=>") {
			if idx := strings.Index(firstLine, "=>"); idx > 0 {
				return strings.TrimSpace(firstLine[:idx+2])
			}
		}
		if idx := strings.Index(firstLine, "{"); idx > 0 {
			return strings.TrimSpace(firstLine[:idx])
		}
	}

	return firstLine
}
