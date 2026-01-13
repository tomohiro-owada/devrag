package indexer

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// RelationType represents the type of code relationship
type RelationType string

const (
	RelationTypeCalls    RelationType = "calls"
	RelationTypeImports  RelationType = "imports"
	RelationTypeInherits RelationType = "inherits"
)

// CodeRelation represents a relationship between code symbols
type CodeRelation struct {
	SourceSymbol string       // Symbol making the reference
	TargetName   string       // Name being referenced
	RelationType RelationType // Type of relationship
	SourceFile   string       // Source file path
	TargetFile   string       // Target file path (if resolved)
	SourceLine   int          // Line number of the reference
}

// RelationExtractor extracts code relationships using Tree-sitter
type RelationExtractor struct {
	parsers map[string]*sitter.Parser
	queries map[string]map[RelationType]*sitter.Query
}

// relationQueryConfig holds Tree-sitter query configurations for each relation type
type relationQueryConfig struct {
	lang        *sitter.Language
	callQuery   string
	importQuery string
	inheritQuery string
}

// NewRelationExtractor creates a new RelationExtractor
func NewRelationExtractor() (*RelationExtractor, error) {
	re := &RelationExtractor{
		parsers: make(map[string]*sitter.Parser),
		queries: make(map[string]map[RelationType]*sitter.Query),
	}

	configs := map[string]relationQueryConfig{
		"go":         getGoRelationConfig(),
		"python":     getPythonRelationConfig(),
		"typescript": getTypeScriptRelationConfig(),
		"javascript": getJavaScriptRelationConfig(),
		"php":        getPHPRelationConfig(),
		"rust":       getRustRelationConfig(),
		// Note: Vue files are parsed as HTML, relation extraction is limited
	}

	for name, cfg := range configs {
		parser := sitter.NewParser()
		parser.SetLanguage(cfg.lang)
		re.parsers[name] = parser
		re.queries[name] = make(map[RelationType]*sitter.Query)

		if cfg.callQuery != "" {
			q, err := sitter.NewQuery([]byte(cfg.callQuery), cfg.lang)
			if err == nil {
				re.queries[name][RelationTypeCalls] = q
			}
		}

		if cfg.importQuery != "" {
			q, err := sitter.NewQuery([]byte(cfg.importQuery), cfg.lang)
			if err == nil {
				re.queries[name][RelationTypeImports] = q
			}
		}

		if cfg.inheritQuery != "" {
			q, err := sitter.NewQuery([]byte(cfg.inheritQuery), cfg.lang)
			if err == nil {
				re.queries[name][RelationTypeInherits] = q
			}
		}
	}

	return re, nil
}

// Close releases resources
func (re *RelationExtractor) Close() {
	for _, parser := range re.parsers {
		parser.Close()
	}
	for _, queries := range re.queries {
		for _, query := range queries {
			query.Close()
		}
	}
}

// ExtractRelations extracts all relations from source code
func (re *RelationExtractor) ExtractRelations(content []byte, lang string, sourceFile string, sourceSymbol string, sourceLine int) ([]CodeRelation, error) {
	parser, ok := re.parsers[lang]
	if !ok {
		return nil, nil // Unsupported language, return empty
	}

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	var relations []CodeRelation

	queries := re.queries[lang]

	// Extract calls
	if q, ok := queries[RelationTypeCalls]; ok {
		calls := re.extractWithQuery(tree.RootNode(), content, q, RelationTypeCalls)
		for _, r := range calls {
			r.SourceFile = sourceFile
			r.SourceSymbol = sourceSymbol
			relations = append(relations, r)
		}
	}

	// Extract imports
	if q, ok := queries[RelationTypeImports]; ok {
		imports := re.extractWithQuery(tree.RootNode(), content, q, RelationTypeImports)
		for _, r := range imports {
			r.SourceFile = sourceFile
			r.SourceSymbol = sourceSymbol
			relations = append(relations, r)
		}
	}

	// Extract inherits
	if q, ok := queries[RelationTypeInherits]; ok {
		inherits := re.extractWithQuery(tree.RootNode(), content, q, RelationTypeInherits)
		for _, r := range inherits {
			r.SourceFile = sourceFile
			r.SourceSymbol = sourceSymbol
			relations = append(relations, r)
		}
	}

	return relations, nil
}

// extractWithQuery runs a query and extracts relations
func (re *RelationExtractor) extractWithQuery(root *sitter.Node, source []byte, query *sitter.Query, relType RelationType) []CodeRelation {
	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	cursor.Exec(query, root)

	seen := make(map[string]bool)
	var relations []CodeRelation

	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}

		for _, capture := range match.Captures {
			name := capture.Node.Content(source)
			line := int(capture.Node.StartPoint().Row) + 1

			// Clean up the name
			name = cleanRelationName(name, relType)
			if name == "" {
				continue
			}

			// Deduplicate
			key := name + ":" + string(relType)
			if seen[key] {
				continue
			}
			seen[key] = true

			relations = append(relations, CodeRelation{
				TargetName:   name,
				RelationType: relType,
				SourceLine:   line,
			})
		}
	}

	return relations
}

// cleanRelationName normalizes the relation target name
func cleanRelationName(name string, relType RelationType) string {
	// Remove quotes from import paths
	name = strings.Trim(name, `"'`)
	name = strings.TrimSpace(name)

	// Skip empty or built-in names
	if name == "" {
		return ""
	}

	// For calls, skip common built-ins
	if relType == RelationTypeCalls {
		builtins := map[string]bool{
			"len": true, "make": true, "append": true, "delete": true,
			"print": true, "println": true, "panic": true, "recover": true,
			"range": true, "return": true, "break": true, "continue": true,
		}
		if builtins[name] {
			return ""
		}
	}

	return name
}

// Go relation queries
func getGoRelationConfig() relationQueryConfig {
	langConfig := GoLanguageConfig()
	return relationQueryConfig{
		lang: langConfig.Language,
		callQuery: `
			(call_expression
				function: (identifier) @call)
			(call_expression
				function: (selector_expression
					field: (field_identifier) @call))
		`,
		importQuery: `
			(import_spec
				path: (interpreted_string_literal) @import)
		`,
		inheritQuery: "", // Go doesn't have inheritance
	}
}

// Python relation queries
func getPythonRelationConfig() relationQueryConfig {
	langConfig := PythonLanguageConfig()
	return relationQueryConfig{
		lang: langConfig.Language,
		callQuery: `
			(call
				function: (identifier) @call)
			(call
				function: (attribute
					attribute: (identifier) @call))
		`,
		importQuery: `
			(import_statement
				name: (dotted_name) @import)
			(import_from_statement
				module_name: (dotted_name) @import)
		`,
		inheritQuery: `
			(class_definition
				superclasses: (argument_list
					(identifier) @inherit))
		`,
	}
}

// TypeScript relation queries
func getTypeScriptRelationConfig() relationQueryConfig {
	langConfig := TypeScriptLanguageConfig()
	return relationQueryConfig{
		lang: langConfig.Language,
		callQuery: `
			(call_expression
				function: (identifier) @call)
			(call_expression
				function: (member_expression
					property: (property_identifier) @call))
		`,
		importQuery: `
			(import_statement
				source: (string) @import)
		`,
		inheritQuery: `
			(class_declaration
				(class_heritage
					(extends_clause
						value: (identifier) @inherit)))
			(class_declaration
				(class_heritage
					(implements_clause
						(type_identifier) @inherit)))
		`,
	}
}

// JavaScript relation queries (same as TypeScript minus type-specific features)
func getJavaScriptRelationConfig() relationQueryConfig {
	langConfig := JavaScriptLanguageConfig()
	return relationQueryConfig{
		lang: langConfig.Language,
		callQuery: `
			(call_expression
				function: (identifier) @call)
			(call_expression
				function: (member_expression
					property: (property_identifier) @call))
		`,
		importQuery: `
			(import_statement
				source: (string) @import)
		`,
		inheritQuery: `
			(class_declaration
				(class_heritage
					(extends_clause
						(identifier) @inherit)))
		`,
	}
}

// PHP relation queries
func getPHPRelationConfig() relationQueryConfig {
	langConfig := PHPLanguageConfig()
	return relationQueryConfig{
		lang: langConfig.Language,
		callQuery: `
			(function_call_expression
				function: (name) @call)
			(function_call_expression
				function: (qualified_name) @call)
			(member_call_expression
				name: (name) @call)
			(scoped_call_expression
				name: (name) @call)
		`,
		importQuery: `
			(namespace_use_declaration
				(namespace_use_clause
					(qualified_name) @import))
			(namespace_use_declaration
				(namespace_use_clause
					(name) @import))
		`,
		inheritQuery: `
			(class_declaration
				(base_clause
					(name) @inherit))
			(class_declaration
				(class_interface_clause
					(name) @inherit))
		`,
	}
}

// Rust relation queries
func getRustRelationConfig() relationQueryConfig {
	langConfig := RustLanguageConfig()
	return relationQueryConfig{
		lang: langConfig.Language,
		callQuery: `
			(call_expression
				function: (identifier) @call)
			(call_expression
				function: (field_expression
					field: (field_identifier) @call))
			(call_expression
				function: (scoped_identifier
					name: (identifier) @call))
		`,
		importQuery: `
			(use_declaration
				argument: (scoped_identifier) @import)
			(use_declaration
				argument: (identifier) @import)
			(use_declaration
				argument: (use_wildcard) @import)
		`,
		inheritQuery: "", // Rust uses traits, not traditional inheritance
	}
}
