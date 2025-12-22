package indexer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewCodeParser(t *testing.T) {
	parser := NewCodeParser()
	if parser == nil {
		t.Fatal("Expected non-nil parser")
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantLang string
		wantOK   bool
	}{
		{"Go file", "main.go", "go", true},
		{"Python file", "script.py", "python", true},
		{"TypeScript file", "app.ts", "typescript", true},
		{"TSX file", "component.tsx", "typescript", true},
		{"JavaScript file", "app.js", "javascript", true},
		{"JSX file", "component.jsx", "javascript", true},
		{"Markdown file", "README.md", "", false},
		{"Text file", "notes.txt", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lang, config, ok := DetectLanguage(tt.filePath)
			if ok != tt.wantOK {
				t.Errorf("DetectLanguage(%q) ok = %v, want %v", tt.filePath, ok, tt.wantOK)
			}
			if lang != tt.wantLang {
				t.Errorf("DetectLanguage(%q) lang = %q, want %q", tt.filePath, lang, tt.wantLang)
			}
			if tt.wantOK && config == nil {
				t.Errorf("DetectLanguage(%q) config is nil", tt.filePath)
			}
		})
	}
}

func TestIsCodeFile(t *testing.T) {
	tests := []struct {
		filePath string
		want     bool
	}{
		{"main.go", true},
		{"script.py", true},
		{"app.ts", true},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			got := IsCodeFile(tt.filePath)
			if got != tt.want {
				t.Errorf("IsCodeFile(%q) = %v, want %v", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestParseCodeContent_Go(t *testing.T) {
	parser := NewCodeParser()

	goCode := []byte(`package main

func Hello(name string) string {
	return "Hello, " + name
}

func main() {
	println(Hello("World"))
}

type User struct {
	Name string
}

func (u *User) Greet() string {
	return "Hi, I'm " + u.Name
}

type Greeter interface {
	Greet() string
}
`)

	chunks, err := parser.ParseCodeContent(goCode, "go")
	if err != nil {
		t.Fatalf("ParseCodeContent failed: %v", err)
	}

	symbolTypes := make(map[SymbolType]int)
	for _, chunk := range chunks {
		symbolTypes[chunk.SymbolType]++
	}

	if symbolTypes[SymbolTypeFunction] != 2 {
		t.Errorf("Expected 2 functions, got %d", symbolTypes[SymbolTypeFunction])
	}
	if symbolTypes[SymbolTypeMethod] != 1 {
		t.Errorf("Expected 1 method, got %d", symbolTypes[SymbolTypeMethod])
	}
	if symbolTypes[SymbolTypeStruct] != 1 {
		t.Errorf("Expected 1 struct, got %d", symbolTypes[SymbolTypeStruct])
	}
	if symbolTypes[SymbolTypeInterface] != 1 {
		t.Errorf("Expected 1 interface, got %d", symbolTypes[SymbolTypeInterface])
	}
}

func TestParseCodeContent_Python(t *testing.T) {
	parser := NewCodeParser()

	pythonCode := []byte(`
def greet(name):
    return f"Hello, {name}!"

class User:
    def __init__(self, name):
        self.name = name
`)

	chunks, err := parser.ParseCodeContent(pythonCode, "python")
	if err != nil {
		t.Fatalf("ParseCodeContent failed: %v", err)
	}

	symbolTypes := make(map[SymbolType]int)
	for _, chunk := range chunks {
		symbolTypes[chunk.SymbolType]++
	}

	if symbolTypes[SymbolTypeFunction] < 1 {
		t.Errorf("Expected at least 1 function, got %d", symbolTypes[SymbolTypeFunction])
	}
	if symbolTypes[SymbolTypeClass] != 1 {
		t.Errorf("Expected 1 class, got %d", symbolTypes[SymbolTypeClass])
	}
}

func TestParseCodeContent_TypeScript(t *testing.T) {
	parser := NewCodeParser()

	tsCode := []byte(`
interface User {
    id: number;
    name: string;
}

class UserService {
    getUser(id: number): User | undefined {
        return undefined;
    }
}

function formatUser(user: User): string {
    return user.name;
}
`)

	chunks, err := parser.ParseCodeContent(tsCode, "typescript")
	if err != nil {
		t.Fatalf("ParseCodeContent failed: %v", err)
	}

	symbolTypes := make(map[SymbolType]int)
	for _, chunk := range chunks {
		symbolTypes[chunk.SymbolType]++
	}

	if symbolTypes[SymbolTypeInterface] != 1 {
		t.Errorf("Expected 1 interface, got %d", symbolTypes[SymbolTypeInterface])
	}
	if symbolTypes[SymbolTypeClass] != 1 {
		t.Errorf("Expected 1 class, got %d", symbolTypes[SymbolTypeClass])
	}
	if symbolTypes[SymbolTypeFunction] < 1 {
		t.Errorf("Expected at least 1 function, got %d", symbolTypes[SymbolTypeFunction])
	}
}

func TestParseCodeFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")

	goCode := `package test

func Add(a, b int) int {
	return a + b
}

type Calculator struct{}
`
	if err := os.WriteFile(tmpFile, []byte(goCode), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	parser := NewCodeParser()
	chunks, err := parser.ParseCodeFile(tmpFile)
	if err != nil {
		t.Fatalf("ParseCodeFile failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	for _, chunk := range chunks {
		if chunk.Language != "go" {
			t.Errorf("Expected language 'go', got %q", chunk.Language)
		}
		if chunk.StartLine <= 0 {
			t.Errorf("Expected positive start line, got %d", chunk.StartLine)
		}
	}
}

func TestParseCodeFile_UnsupportedLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(tmpFile, []byte("text"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	parser := NewCodeParser()
	_, err := parser.ParseCodeFile(tmpFile)
	if err == nil {
		t.Error("Expected error for unsupported language")
	}
}

func TestCodeChunk_Interface(t *testing.T) {
	chunk := CodeChunk{
		Content:    "func test() {}",
		Position:   0,
		SymbolName: "test",
		SymbolType: SymbolTypeFunction,
		Language:   "go",
		StartLine:  1,
		EndLine:    1,
	}

	if chunk.GetContent() != "func test() {}" {
		t.Errorf("GetContent() = %q", chunk.GetContent())
	}
	if chunk.GetPosition() != 0 {
		t.Errorf("GetPosition() = %d", chunk.GetPosition())
	}
	if chunk.GetSymbolType() != "function" {
		t.Errorf("GetSymbolType() = %q", chunk.GetSymbolType())
	}
}

func TestGetSupportedExtensions(t *testing.T) {
	extensions := GetSupportedExtensions()
	if len(extensions) == 0 {
		t.Error("Expected at least one supported extension")
	}
}
