package indexer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// CodeParser parses source code files using Tree-sitter
type CodeParser struct {
	parsers map[string]*sitter.Parser
	queries map[string]*sitter.Query
}

// NewCodeParser creates a new CodeParser instance
func NewCodeParser() (*CodeParser, error) {
	cp := &CodeParser{
		parsers: make(map[string]*sitter.Parser),
		queries: make(map[string]*sitter.Query),
	}

	// Initialize parsers and queries for all supported languages
	for name, config := range SupportedLanguages() {
		parser := sitter.NewParser()
		parser.SetLanguage(config.Language)
		cp.parsers[name] = parser

		query, err := sitter.NewQuery([]byte(config.Query), config.Language)
		if err != nil {
			return nil, fmt.Errorf("failed to create query for %s: %w", name, err)
		}
		cp.queries[name] = query
	}

	return cp, nil
}

// Close releases resources used by the parser
func (cp *CodeParser) Close() {
	for _, parser := range cp.parsers {
		parser.Close()
	}
	for _, query := range cp.queries {
		query.Close()
	}
}

// ParseFile parses a source code file and returns code chunks
func (cp *CodeParser) ParseFile(filepath string) ([]CodeChunk, error) {
	// Read file content
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Detect language from extension
	lang, err := cp.detectLanguage(filepath)
	if err != nil {
		return nil, err
	}

	// Parse the file
	parser := cp.parsers[lang]
	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}
	defer tree.Close()

	// Extract symbols
	chunks, err := cp.extractSymbols(tree.RootNode(), content, lang)
	if err != nil {
		return nil, fmt.Errorf("failed to extract symbols: %w", err)
	}

	if len(chunks) == 0 {
		fmt.Fprintf(os.Stderr, "[WARN] No symbols extracted from %s\n", filepath)
	}

	return chunks, nil
}

// detectLanguage determines the programming language from file extension
func (cp *CodeParser) detectLanguage(filePath string) (string, error) {
	ext := filepath.Ext(filePath)
	config := GetLanguageByExtension(ext)
	if config == nil {
		return "", fmt.Errorf("unsupported file type: %s. Supported: [.go .py .ts .tsx .js .jsx]", ext)
	}
	return config.Name, nil
}

// extractSymbols extracts code symbols from AST using Tree-sitter queries
func (cp *CodeParser) extractSymbols(root *sitter.Node, source []byte, lang string) ([]CodeChunk, error) {
	query := cp.queries[lang]
	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	cursor.Exec(query, root)

	var chunks []CodeChunk
	seen := make(map[string]bool) // For deduplication based on byte position
	position := 0

	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}

		var symbolName string
		var symbolNode *sitter.Node
		var symbolType SymbolType

		// Process captures to find symbol name and node
		for _, capture := range match.Captures {
			captureName := query.CaptureNameForId(capture.Index)

			switch captureName {
			case "name":
				symbolName = capture.Node.Content(source)
			case "function":
				symbolNode = capture.Node
				symbolType = SymbolTypeFunction
			case "method":
				symbolNode = capture.Node
				symbolType = SymbolTypeMethod
			case "class":
				symbolNode = capture.Node
				symbolType = SymbolTypeClass
			case "struct":
				symbolNode = capture.Node
				symbolType = SymbolTypeStruct
			case "interface":
				symbolNode = capture.Node
				symbolType = SymbolTypeInterface
			}
		}

		if symbolNode == nil || symbolName == "" {
			continue
		}

		// Deduplicate by byte position
		key := fmt.Sprintf("%d-%d", symbolNode.StartByte(), symbolNode.EndByte())
		if seen[key] {
			continue
		}
		seen[key] = true

		// Extract content and metadata
		content := symbolNode.Content(source)
		startLine := int(symbolNode.StartPoint().Row) + 1 // 1-indexed
		endLine := int(symbolNode.EndPoint().Row) + 1

		// Extract signature
		signature := cp.extractSignature(content, lang)

		// Find parent symbol (for methods)
		parentSymbol := cp.findParentSymbol(symbolNode, source, lang)

		chunk := CodeChunk{
			Content:      content,
			Position:     position,
			SymbolName:   symbolName,
			SymbolType:   symbolType,
			Language:     lang,
			StartLine:    startLine,
			EndLine:      endLine,
			ParentSymbol: parentSymbol,
			Signature:    signature,
		}

		chunks = append(chunks, chunk)
		position++
	}

	return chunks, nil
}

// extractSignature extracts the function/method signature from content
func (cp *CodeParser) extractSignature(content string, lang string) string {
	switch lang {
	case "go":
		idx := strings.Index(content, "{")
		if idx == -1 {
			lines := strings.SplitN(content, "\n", 2)
			return strings.TrimSpace(lines[0])
		}
		sig := strings.TrimSpace(content[:idx])
		sig = strings.ReplaceAll(sig, "\n", " ")
		return strings.Join(strings.Fields(sig), " ")

	case "python":
		// For Python, find the last colon in the first line (before body)
		lines := strings.SplitN(content, "\n", 2)
		firstLine := strings.TrimSpace(lines[0])
		// Python signatures end with ":"
		if strings.HasSuffix(firstLine, ":") {
			return firstLine[:len(firstLine)-1]
		}
		// Handle multiline signatures - find the closing paren then the colon
		idx := strings.Index(content, "):")
		if idx != -1 {
			sig := strings.TrimSpace(content[:idx+1])
			sig = strings.ReplaceAll(sig, "\n", " ")
			return strings.Join(strings.Fields(sig), " ")
		}
		return firstLine

	case "typescript", "javascript":
		// For arrow functions, use "=>"
		if strings.Contains(content, "=>") {
			idx := strings.Index(content, "=>")
			if idx != -1 {
				sig := strings.TrimSpace(content[:idx+2])
				sig = strings.ReplaceAll(sig, "\n", " ")
				return strings.Join(strings.Fields(sig), " ")
			}
		}
		idx := strings.Index(content, "{")
		if idx == -1 {
			lines := strings.SplitN(content, "\n", 2)
			return strings.TrimSpace(lines[0])
		}
		sig := strings.TrimSpace(content[:idx])
		sig = strings.ReplaceAll(sig, "\n", " ")
		return strings.Join(strings.Fields(sig), " ")

	default:
		idx := strings.Index(content, "{")
		if idx == -1 {
			lines := strings.SplitN(content, "\n", 2)
			return strings.TrimSpace(lines[0])
		}
		sig := strings.TrimSpace(content[:idx])
		sig = strings.ReplaceAll(sig, "\n", " ")
		return strings.Join(strings.Fields(sig), " ")
	}
}

// findParentSymbol finds the parent class/struct name for methods
func (cp *CodeParser) findParentSymbol(node *sitter.Node, source []byte, lang string) string {
	parent := node.Parent()
	for parent != nil {
		nodeType := parent.Type()

		switch lang {
		case "go":
			// For Go methods, the receiver type is in the method declaration itself
			if nodeType == "method_declaration" {
				// Find the receiver type
				for i := 0; i < int(parent.ChildCount()); i++ {
					child := parent.Child(i)
					if child.Type() == "parameter_list" {
						// First parameter list is the receiver
						content := child.Content(source)
						// Extract type name from receiver (e.g., "(u *User)" -> "User")
						content = strings.Trim(content, "()")
						parts := strings.Fields(content)
						if len(parts) >= 2 {
							typeName := parts[len(parts)-1]
							return strings.TrimPrefix(typeName, "*")
						}
					}
				}
			}
		case "python", "typescript", "javascript":
			if nodeType == "class_definition" || nodeType == "class_declaration" {
				// Find the class name
				for i := 0; i < int(parent.ChildCount()); i++ {
					child := parent.Child(i)
					if child.Type() == "identifier" || child.Type() == "type_identifier" {
						return child.Content(source)
					}
				}
			}
		}

		parent = parent.Parent()
	}

	return ""
}

// ParseCode parses source code content directly (without file)
func (cp *CodeParser) ParseCode(content []byte, lang string) ([]CodeChunk, error) {
	parser, ok := cp.parsers[lang]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse code: %w", err)
	}
	defer tree.Close()

	return cp.extractSymbols(tree.RootNode(), content, lang)
}
