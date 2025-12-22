package indexer

import (
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// SymbolQuery defines a Tree-sitter query for extracting symbols
type SymbolQuery struct {
	Pattern    string     // Tree-sitter query pattern
	SymbolType SymbolType // Type of symbol this query extracts
}

// LanguageConfig holds configuration for a programming language
type LanguageConfig struct {
	Name       string           // Language name (go, python, typescript, javascript)
	Language   *sitter.Language // Tree-sitter language
	Extensions []string         // File extensions for this language
	Queries    []SymbolQuery    // Queries for extracting symbols
}

// SupportedLanguages maps language names to their configurations
var SupportedLanguages = map[string]*LanguageConfig{
	"go":         goConfig(),
	"python":     pythonConfig(),
	"typescript": typescriptConfig(),
	"javascript": javascriptConfig(),
}

func goConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:       "go",
		Language:   golang.GetLanguage(),
		Extensions: []string{".go"},
		Queries: []SymbolQuery{
			{Pattern: `(function_declaration name: (identifier) @name) @function`, SymbolType: SymbolTypeFunction},
			{Pattern: `(method_declaration name: (field_identifier) @name) @method`, SymbolType: SymbolTypeMethod},
			{Pattern: `(type_declaration (type_spec name: (type_identifier) @name type: (struct_type))) @struct`, SymbolType: SymbolTypeStruct},
			{Pattern: `(type_declaration (type_spec name: (type_identifier) @name type: (interface_type))) @interface`, SymbolType: SymbolTypeInterface},
		},
	}
}

func pythonConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:       "python",
		Language:   python.GetLanguage(),
		Extensions: []string{".py"},
		Queries: []SymbolQuery{
			{Pattern: `(function_definition name: (identifier) @name) @function`, SymbolType: SymbolTypeFunction},
			{Pattern: `(class_definition name: (identifier) @name) @class`, SymbolType: SymbolTypeClass},
		},
	}
}

func typescriptConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:       "typescript",
		Language:   typescript.GetLanguage(),
		Extensions: []string{".ts", ".tsx"},
		Queries: []SymbolQuery{
			{Pattern: `(function_declaration name: (identifier) @name) @function`, SymbolType: SymbolTypeFunction},
			{Pattern: `(lexical_declaration (variable_declarator name: (identifier) @name value: (arrow_function))) @function`, SymbolType: SymbolTypeFunction},
			{Pattern: `(class_declaration name: (type_identifier) @name) @class`, SymbolType: SymbolTypeClass},
			{Pattern: `(interface_declaration name: (type_identifier) @name) @interface`, SymbolType: SymbolTypeInterface},
			{Pattern: `(method_definition name: (property_identifier) @name) @method`, SymbolType: SymbolTypeMethod},
		},
	}
}

func javascriptConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:       "javascript",
		Language:   javascript.GetLanguage(),
		Extensions: []string{".js", ".jsx"},
		Queries: []SymbolQuery{
			{Pattern: `(function_declaration name: (identifier) @name) @function`, SymbolType: SymbolTypeFunction},
			{Pattern: `(lexical_declaration (variable_declarator name: (identifier) @name value: (arrow_function))) @function`, SymbolType: SymbolTypeFunction},
			{Pattern: `(class_declaration name: (identifier) @name) @class`, SymbolType: SymbolTypeClass},
			{Pattern: `(method_definition name: (property_identifier) @name) @method`, SymbolType: SymbolTypeMethod},
		},
	}
}

// DetectLanguage detects the programming language from a file path
func DetectLanguage(filePath string) (string, *LanguageConfig, bool) {
	ext := strings.ToLower(filepath.Ext(filePath))
	for langName, config := range SupportedLanguages {
		for _, supportedExt := range config.Extensions {
			if ext == supportedExt {
				return langName, config, true
			}
		}
	}
	return "", nil, false
}

// GetSupportedExtensions returns all supported file extensions
func GetSupportedExtensions() []string {
	var extensions []string
	for _, config := range SupportedLanguages {
		extensions = append(extensions, config.Extensions...)
	}
	return extensions
}

// IsCodeFile checks if a file path has a supported code extension
func IsCodeFile(filePath string) bool {
	_, _, ok := DetectLanguage(filePath)
	return ok
}
