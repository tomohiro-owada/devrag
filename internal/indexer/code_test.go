package indexer

import (
	"path/filepath"
	"testing"
)

func TestNewCodeParser(t *testing.T) {
	parser, err := NewCodeParser()
	if err != nil {
		t.Fatalf("Failed to create code parser: %v", err)
	}
	defer parser.Close()

	// Check that parsers were created for all supported languages
	expectedLangs := []string{"go", "python", "typescript", "javascript", "php", "rust", "vue"}
	for _, lang := range expectedLangs {
		if _, ok := parser.parsers[lang]; !ok {
			t.Errorf("Parser not found for language: %s", lang)
		}
		if _, ok := parser.queries[lang]; !ok {
			t.Errorf("Query not found for language: %s", lang)
		}
	}
}

func TestParseGoFile(t *testing.T) {
	parser, err := NewCodeParser()
	if err != nil {
		t.Fatalf("Failed to create code parser: %v", err)
	}
	defer parser.Close()

	chunks, err := parser.ParseFile(filepath.Join("..", "..", "test_data", "code", "sample.go"))
	if err != nil {
		t.Fatalf("Failed to parse Go file: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("No chunks extracted from Go file")
	}

	// Check expected symbols
	expectedSymbols := map[string]SymbolType{
		"User":           SymbolTypeStruct,
		"UserService":    SymbolTypeStruct,
		"NewUserService": SymbolTypeFunction,
		"GetUser":        SymbolTypeMethod,
		"CreateUser":     SymbolTypeMethod,
		"Greeter":        SymbolTypeInterface,
		"Hello":          SymbolTypeFunction,
	}

	foundSymbols := make(map[string]bool)
	for _, chunk := range chunks {
		foundSymbols[chunk.SymbolName] = true
		if expectedType, ok := expectedSymbols[chunk.SymbolName]; ok {
			if chunk.SymbolType != expectedType {
				t.Errorf("Symbol %s: expected type %s, got %s", chunk.SymbolName, expectedType, chunk.SymbolType)
			}
		}
		// Verify language
		if chunk.Language != "go" {
			t.Errorf("Expected language 'go', got '%s'", chunk.Language)
		}
		// Verify line numbers are set
		if chunk.StartLine == 0 || chunk.EndLine == 0 {
			t.Errorf("Line numbers not set for symbol %s", chunk.SymbolName)
		}
	}

	for name := range expectedSymbols {
		if !foundSymbols[name] {
			t.Errorf("Expected symbol not found: %s", name)
		}
	}
}

func TestParsePythonFile(t *testing.T) {
	parser, err := NewCodeParser()
	if err != nil {
		t.Fatalf("Failed to create code parser: %v", err)
	}
	defer parser.Close()

	chunks, err := parser.ParseFile(filepath.Join("..", "..", "test_data", "code", "sample.py"))
	if err != nil {
		t.Fatalf("Failed to parse Python file: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("No chunks extracted from Python file")
	}

	// Check expected symbols
	expectedSymbols := map[string]SymbolType{
		"User":          SymbolTypeClass,
		"UserService":   SymbolTypeClass,
		"greet":         SymbolTypeFunction,
		"calculate_sum": SymbolTypeFunction,
	}

	foundSymbols := make(map[string]bool)
	for _, chunk := range chunks {
		foundSymbols[chunk.SymbolName] = true
		if chunk.Language != "python" {
			t.Errorf("Expected language 'python', got '%s'", chunk.Language)
		}
	}

	for name := range expectedSymbols {
		if !foundSymbols[name] {
			t.Errorf("Expected symbol not found: %s", name)
		}
	}
}

func TestParseTypeScriptFile(t *testing.T) {
	parser, err := NewCodeParser()
	if err != nil {
		t.Fatalf("Failed to create code parser: %v", err)
	}
	defer parser.Close()

	chunks, err := parser.ParseFile(filepath.Join("..", "..", "test_data", "code", "sample.ts"))
	if err != nil {
		t.Fatalf("Failed to parse TypeScript file: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("No chunks extracted from TypeScript file")
	}

	// Check for expected symbols
	expectedSymbols := []string{"User", "UserRepository", "UserService", "greet", "calculateSum"}

	foundSymbols := make(map[string]bool)
	for _, chunk := range chunks {
		foundSymbols[chunk.SymbolName] = true
		if chunk.Language != "typescript" {
			t.Errorf("Expected language 'typescript', got '%s'", chunk.Language)
		}
	}

	for _, name := range expectedSymbols {
		if !foundSymbols[name] {
			t.Errorf("Expected symbol not found: %s", name)
		}
	}
}

func TestParsePHPFile(t *testing.T) {
	parser, err := NewCodeParser()
	if err != nil {
		t.Fatalf("Failed to create code parser: %v", err)
	}
	defer parser.Close()

	chunks, err := parser.ParseFile(filepath.Join("..", "..", "test_data", "code", "sample.php"))
	if err != nil {
		t.Fatalf("Failed to parse PHP file: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("No chunks extracted from PHP file")
	}

	// Check expected symbols
	expectedSymbols := []string{"UserService", "CacheService", "Cacheable", "calculateDiscount"}

	foundSymbols := make(map[string]bool)
	for _, chunk := range chunks {
		foundSymbols[chunk.SymbolName] = true
		if chunk.Language != "php" {
			t.Errorf("Expected language 'php', got '%s'", chunk.Language)
		}
	}

	for _, name := range expectedSymbols {
		if !foundSymbols[name] {
			t.Errorf("Expected symbol not found: %s", name)
		}
	}
}

func TestParseRustFile(t *testing.T) {
	parser, err := NewCodeParser()
	if err != nil {
		t.Fatalf("Failed to create code parser: %v", err)
	}
	defer parser.Close()

	chunks, err := parser.ParseFile(filepath.Join("..", "..", "test_data", "code", "sample.rs"))
	if err != nil {
		t.Fatalf("Failed to parse Rust file: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("No chunks extracted from Rust file")
	}

	// Check expected symbols
	expectedSymbols := []string{"Config", "User", "UserRepository", "InMemoryUserRepository", "UserService"}

	foundSymbols := make(map[string]bool)
	for _, chunk := range chunks {
		foundSymbols[chunk.SymbolName] = true
		if chunk.Language != "rust" {
			t.Errorf("Expected language 'rust', got '%s'", chunk.Language)
		}
	}

	for _, name := range expectedSymbols {
		if !foundSymbols[name] {
			t.Errorf("Expected symbol not found: %s", name)
		}
	}
}

func TestExtractSignature(t *testing.T) {
	parser, err := NewCodeParser()
	if err != nil {
		t.Fatalf("Failed to create code parser: %v", err)
	}
	defer parser.Close()

	tests := []struct {
		content  string
		lang     string
		expected string
	}{
		{
			content:  "func Hello(name string) string {\n\treturn \"Hello, \" + name\n}",
			lang:     "go",
			expected: "func Hello(name string) string",
		},
		{
			content:  "def greet(name: str) -> str:\n    return f\"Hello, {name}!\"",
			lang:     "python",
			expected: "def greet(name: str) -> str",
		},
		{
			content:  "function greet(name: string): string {\n  return `Hello, ${name}!`;\n}",
			lang:     "typescript",
			expected: "function greet(name: string): string",
		},
		{
			content:  "const add = (a: number, b: number): number => {\n  return a + b;\n}",
			lang:     "typescript",
			expected: "const add = (a: number, b: number): number =>",
		},
	}

	for _, tc := range tests {
		sig := parser.extractSignature(tc.content, tc.lang)
		if sig != tc.expected {
			t.Errorf("extractSignature(%s):\nexpected: %s\ngot: %s", tc.lang, tc.expected, sig)
		}
	}
}

func TestGetLanguageByExtension(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
		isNil    bool
	}{
		{".go", "go", false},
		{".py", "python", false},
		{".ts", "typescript", false},
		{".tsx", "typescript", false},
		{".js", "javascript", false},
		{".jsx", "javascript", false},
		{".php", "php", false},
		{".rs", "rust", false},
		{".vue", "vue", false},
		{".md", "", true},
		{".txt", "", true},
	}

	for _, tc := range tests {
		config := GetLanguageByExtension(tc.ext)
		if tc.isNil {
			if config != nil {
				t.Errorf("GetLanguageByExtension(%s): expected nil, got %v", tc.ext, config)
			}
		} else {
			if config == nil {
				t.Errorf("GetLanguageByExtension(%s): expected config, got nil", tc.ext)
			} else if config.Name != tc.expected {
				t.Errorf("GetLanguageByExtension(%s): expected %s, got %s", tc.ext, tc.expected, config.Name)
			}
		}
	}
}

func TestIsSupportedExtension(t *testing.T) {
	supported := []string{".go", ".py", ".ts", ".tsx", ".js", ".jsx", ".php", ".rs", ".vue"}
	unsupported := []string{".md", ".txt", ".yaml", ".json", ".c", ".cpp"}

	for _, ext := range supported {
		if !IsSupportedExtension(ext) {
			t.Errorf("Expected %s to be supported", ext)
		}
	}

	for _, ext := range unsupported {
		if IsSupportedExtension(ext) {
			t.Errorf("Expected %s to be unsupported", ext)
		}
	}
}

func TestCodeChunkInterface(t *testing.T) {
	chunk := CodeChunk{
		Content:      "func Test() {}",
		Position:     0,
		SymbolName:   "Test",
		SymbolType:   SymbolTypeFunction,
		Language:     "go",
		StartLine:    1,
		EndLine:      1,
		ParentSymbol: "",
		Signature:    "func Test()",
	}

	// Test ChunkInterface methods
	if chunk.GetContent() != "func Test() {}" {
		t.Error("GetContent() failed")
	}
	if chunk.GetPosition() != 0 {
		t.Error("GetPosition() failed")
	}

	// Test CodeChunkInterface methods
	if chunk.GetSymbolName() != "Test" {
		t.Error("GetSymbolName() failed")
	}
	if chunk.GetSymbolType() != "function" {
		t.Error("GetSymbolType() failed")
	}
	if chunk.GetLanguage() != "go" {
		t.Error("GetLanguage() failed")
	}
	if chunk.GetStartLine() != 1 {
		t.Error("GetStartLine() failed")
	}
	if chunk.GetEndLine() != 1 {
		t.Error("GetEndLine() failed")
	}
	if chunk.GetSignature() != "func Test()" {
		t.Error("GetSignature() failed")
	}

	// Test GetEmbeddingText
	expected := "go Test: func Test() {}"
	if chunk.GetEmbeddingText() != expected {
		t.Errorf("GetEmbeddingText(): expected %s, got %s", expected, chunk.GetEmbeddingText())
	}
}
