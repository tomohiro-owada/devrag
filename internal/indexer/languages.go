package indexer

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/html"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// LanguageConfig holds Tree-sitter configuration for a language
type LanguageConfig struct {
	Name       string
	Language   *sitter.Language
	Extensions []string
	Query      string
}

// SupportedLanguages returns all supported language configurations
func SupportedLanguages() map[string]*LanguageConfig {
	return map[string]*LanguageConfig{
		"go":         GoLanguageConfig(),
		"python":     PythonLanguageConfig(),
		"typescript": TypeScriptLanguageConfig(),
		"javascript": JavaScriptLanguageConfig(),
		"php":        PHPLanguageConfig(),
		"rust":       RustLanguageConfig(),
		"vue":        VueLanguageConfig(),
	}
}

// GetLanguageByExtension returns the language config for a file extension
func GetLanguageByExtension(ext string) *LanguageConfig {
	extensionMap := map[string]string{
		".go":   "go",
		".py":   "python",
		".ts":   "typescript",
		".tsx":  "typescript",
		".js":   "javascript",
		".jsx":  "javascript",
		".php":  "php",
		".rs":   "rust",
		".vue":  "vue",
	}

	langName, ok := extensionMap[ext]
	if !ok {
		return nil
	}

	langs := SupportedLanguages()
	return langs[langName]
}

// IsSupportedExtension checks if the file extension is supported
func IsSupportedExtension(ext string) bool {
	return GetLanguageByExtension(ext) != nil
}

// GoLanguageConfig returns Tree-sitter configuration for Go
func GoLanguageConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:       "go",
		Language:   golang.GetLanguage(),
		Extensions: []string{".go"},
		Query: `
(function_declaration
  name: (identifier) @name) @function

(method_declaration
  name: (field_identifier) @name) @method

(type_declaration
  (type_spec
    name: (type_identifier) @name
    type: (struct_type))) @struct

(type_declaration
  (type_spec
    name: (type_identifier) @name
    type: (interface_type))) @interface
`,
	}
}

// PythonLanguageConfig returns Tree-sitter configuration for Python
func PythonLanguageConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:       "python",
		Language:   python.GetLanguage(),
		Extensions: []string{".py"},
		Query: `
(function_definition
  name: (identifier) @name) @function

(class_definition
  name: (identifier) @name) @class
`,
	}
}

// TypeScriptLanguageConfig returns Tree-sitter configuration for TypeScript
func TypeScriptLanguageConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:       "typescript",
		Language:   typescript.GetLanguage(),
		Extensions: []string{".ts", ".tsx"},
		Query: `
(function_declaration
  name: (identifier) @name) @function

(lexical_declaration
  (variable_declarator
    name: (identifier) @name
    value: (arrow_function))) @function

(class_declaration
  name: (type_identifier) @name) @class

(interface_declaration
  name: (type_identifier) @name) @interface

(method_definition
  name: (property_identifier) @name) @method
`,
	}
}

// JavaScriptLanguageConfig returns Tree-sitter configuration for JavaScript
func JavaScriptLanguageConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:       "javascript",
		Language:   javascript.GetLanguage(),
		Extensions: []string{".js", ".jsx"},
		Query: `
(function_declaration
  name: (identifier) @name) @function

(lexical_declaration
  (variable_declarator
    name: (identifier) @name
    value: (arrow_function))) @function

(class_declaration
  name: (identifier) @name) @class

(method_definition
  name: (property_identifier) @name) @method
`,
	}
}

// PHPLanguageConfig returns Tree-sitter configuration for PHP
func PHPLanguageConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:       "php",
		Language:   php.GetLanguage(),
		Extensions: []string{".php"},
		Query: `
(function_definition
  name: (name) @name) @function

(method_declaration
  name: (name) @name) @method

(class_declaration
  name: (name) @name) @class

(interface_declaration
  name: (name) @name) @interface

(trait_declaration
  name: (name) @name) @class
`,
	}
}

// RustLanguageConfig returns Tree-sitter configuration for Rust
func RustLanguageConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:       "rust",
		Language:   rust.GetLanguage(),
		Extensions: []string{".rs"},
		Query: `
(function_item
  name: (identifier) @name) @function

(impl_item
  trait: (type_identifier)? @trait
  type: (type_identifier) @name) @struct

(struct_item
  name: (type_identifier) @name) @struct

(enum_item
  name: (type_identifier) @name) @struct

(trait_item
  name: (type_identifier) @name) @interface

(mod_item
  name: (identifier) @name) @function
`,
	}
}

// VueLanguageConfig returns Tree-sitter configuration for Vue SFC
// Note: Vue files are complex SFCs. We use HTML parser and extract script content.
// For better results, consider preprocessing Vue files to extract TypeScript/JavaScript.
func VueLanguageConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:       "vue",
		Language:   html.GetLanguage(),
		Extensions: []string{".vue"},
		Query: `
(script_element
  (raw_text) @script)
`,
	}
}
