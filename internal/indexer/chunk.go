package indexer

// SymbolType represents the type of code symbol
type SymbolType string

const (
	SymbolTypeFunction  SymbolType = "function"
	SymbolTypeMethod    SymbolType = "method"
	SymbolTypeClass     SymbolType = "class"
	SymbolTypeStruct    SymbolType = "struct"
	SymbolTypeInterface SymbolType = "interface"
)

// CodeChunk represents a code chunk with metadata
type CodeChunk struct {
	Content      string     // The actual code content
	Position     int        // Position in document (0-indexed)
	SymbolName   string     // Function/class/method name
	SymbolType   SymbolType // Type of symbol (function, method, class, etc.)
	Language     string     // Programming language (go, python, typescript, javascript)
	StartLine    int        // Start line number (1-indexed)
	EndLine      int        // End line number (1-indexed)
	ParentSymbol string     // Parent class/struct name for methods
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

// GetSymbolName returns the symbol name
func (c CodeChunk) GetSymbolName() string {
	return c.SymbolName
}

// GetSymbolType returns the symbol type as string (for CodeChunkInterface)
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
