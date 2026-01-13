package indexer

import "fmt"

// SymbolType represents the type of code symbol
type SymbolType string

const (
	SymbolTypeFunction  SymbolType = "function"
	SymbolTypeMethod    SymbolType = "method"
	SymbolTypeClass     SymbolType = "class"
	SymbolTypeStruct    SymbolType = "struct"
	SymbolTypeInterface SymbolType = "interface"
)

// CodeChunk represents a parsed code symbol with metadata
type CodeChunk struct {
	Content      string     // Code content
	Position     int        // Position within document (0-indexed)
	SymbolName   string     // Symbol name (function name, class name, etc.)
	SymbolType   SymbolType // Type of symbol
	Language     string     // Programming language
	StartLine    int        // Start line number (1-indexed)
	EndLine      int        // End line number (1-indexed)
	ParentSymbol string     // Parent symbol (e.g., class name for methods)
	Signature    string     // Function/method signature
}

// GetContent implements ChunkInterface
func (c CodeChunk) GetContent() string {
	return c.Content
}

// GetPosition implements ChunkInterface
func (c CodeChunk) GetPosition() int {
	return c.Position
}

// GetEmbeddingText returns formatted text for embedding
// Format: "{language} {symbol_name}: {content}"
func (c CodeChunk) GetEmbeddingText() string {
	return fmt.Sprintf("%s %s: %s", c.Language, c.SymbolName, c.Content)
}

// GetSymbolName returns the symbol name
func (c CodeChunk) GetSymbolName() string {
	return c.SymbolName
}

// GetSymbolType returns the symbol type as string
func (c CodeChunk) GetSymbolType() string {
	return string(c.SymbolType)
}

// GetLanguage returns the programming language
func (c CodeChunk) GetLanguage() string {
	return c.Language
}

// GetStartLine returns the start line number
func (c CodeChunk) GetStartLine() int {
	return c.StartLine
}

// GetEndLine returns the end line number
func (c CodeChunk) GetEndLine() int {
	return c.EndLine
}

// GetParentSymbol returns the parent symbol name
func (c CodeChunk) GetParentSymbol() string {
	return c.ParentSymbol
}

// GetSignature returns the function/method signature
func (c CodeChunk) GetSignature() string {
	return c.Signature
}

// CodeMetadata holds metadata for a code chunk stored in database
type CodeMetadata struct {
	ID           int64
	ChunkID      int64
	SymbolName   string
	SymbolType   SymbolType
	Language     string
	StartLine    int
	EndLine      int
	ParentSymbol string
	Signature    string
}
