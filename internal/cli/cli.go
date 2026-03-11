package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/tomohiro-owada/devrag/internal/config"
	"github.com/tomohiro-owada/devrag/internal/embedder"
	"github.com/tomohiro-owada/devrag/internal/frontmatter"
	"github.com/tomohiro-owada/devrag/internal/indexer"
	"github.com/tomohiro-owada/devrag/internal/vectordb"
	"github.com/tomohiro-owada/devrag/internal/version"
)

// CLI holds shared dependencies for CLI commands
type CLI struct {
	db       *vectordb.DB
	embedder embedder.Embedder
	indexer  *indexer.Indexer
	config   *config.Config
}

// Response is the standardized CLI response envelope
type Response struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo provides structured error details
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

// New creates a new CLI instance
func New(db *vectordb.DB, emb embedder.Embedder, idx *indexer.Indexer, cfg *config.Config) *CLI {
	return &CLI{db: db, embedder: emb, indexer: idx, config: cfg}
}

// Run dispatches the subcommand
func (c *CLI) Run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "search":
		return c.runSearch(cmdArgs)
	case "index":
		return c.runIndex(cmdArgs)
	case "index-code":
		return c.runIndexCode(cmdArgs)
	case "list":
		return c.runList(cmdArgs)
	case "delete":
		return c.runDelete(cmdArgs)
	case "reindex":
		return c.runReindex(cmdArgs)
	case "search-relations":
		return c.runSearchRelations(cmdArgs)
	case "build-dictionary":
		return c.runBuildDictionary(cmdArgs)
	case "add-frontmatter":
		return c.runAddFrontmatter(cmdArgs)
	case "update-frontmatter":
		return c.runUpdateFrontmatter(cmdArgs)
	case "schema":
		return c.runSchema(cmdArgs)
	case "help":
		printUsage()
		return nil
	default:
		return outputError("unknown_command",
			fmt.Sprintf("unknown command: %s", cmd), "",
			"Run 'devrag schema' for available commands")
	}
}

// --- search ---

func (c *CLI) runSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	topK := fs.Int("top-k", 5, "max number of results")
	directory := fs.String("directory", "", "filter by directory")
	filePattern := fs.String("file-pattern", "", "filter by filename glob pattern")
	fields := fs.String("fields", "", "comma-separated fields to include (e.g. content,score,filename)")
	outputFmt := fs.String("output", "json", "output format: json | text")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return outputError("missing_argument", "query is required", "query",
			"devrag search <query> [--top-k N]")
	}

	query := strings.Join(fs.Args(), " ")
	if err := validateText(query, "query"); err != nil {
		return err
	}

	queryVector, err := c.embedder.Embed(query)
	if err != nil {
		return outputError("embed_failed", fmt.Sprintf("failed to vectorize query: %v", err), "", "")
	}

	var filter *vectordb.SearchFilter
	if *directory != "" || *filePattern != "" {
		filter = &vectordb.SearchFilter{
			Directory:   *directory,
			FilePattern: *filePattern,
		}
	}

	results, err := c.db.SearchWithFilter(queryVector, *topK, filter)
	if err != nil {
		return outputError("search_failed", fmt.Sprintf("search failed: %v", err), "", "")
	}

	data := map[string]interface{}{
		"query":   query,
		"count":   len(results),
		"results": filterFields(resultsToMaps(results), *fields),
	}

	if *outputFmt == "text" {
		return c.printSearchText(results)
	}
	return outputSuccess(data)
}

func (c *CLI) printSearchText(results []vectordb.SearchResult) error {
	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}
	for i, r := range results {
		fmt.Printf("--- [%d] %s (score: %.4f, position: %d) ---\n", i+1, r.DocumentName, r.Similarity, r.Position)
		fmt.Println(r.ChunkContent)
		fmt.Println()
	}
	return nil
}

// --- index ---

func (c *CLI) runIndex(args []string) error {
	fs := flag.NewFlagSet("index", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return outputError("missing_argument", "filepath is required", "filepath",
			"devrag index <filepath>")
	}

	filePath := fs.Arg(0)
	if err := validatePath(filePath); err != nil {
		return err
	}

	if err := c.indexer.IndexFile(filePath); err != nil {
		return outputError("index_failed", fmt.Sprintf("indexing failed: %v", err), "", "")
	}

	return outputSuccess(map[string]interface{}{
		"message": "file indexed successfully",
		"file":    filePath,
	})
}

// --- index-code ---

func (c *CLI) runIndexCode(args []string) error {
	fs := flag.NewFlagSet("index-code", flag.ContinueOnError)
	directory := fs.String("directory", "", "directory to index recursively")
	filepaths := fs.String("filepaths", "", "comma-separated list of file paths")
	force := fs.Bool("force", false, "force re-index regardless of modification time")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := c.indexer.InitCodeParser(); err != nil {
		return outputError("init_failed", fmt.Sprintf("failed to initialize code parser: %v", err), "", "")
	}

	// Single file from positional arg
	if fs.NArg() > 0 {
		filePath := fs.Arg(0)
		if err := validatePath(filePath); err != nil {
			return err
		}
		if err := c.indexer.IndexCodeFile(filePath); err != nil {
			return outputError("index_failed", fmt.Sprintf("indexing failed: %v", err), "", "")
		}
		return outputSuccess(map[string]interface{}{
			"message": "code file indexed successfully",
			"file":    filePath,
		})
	}

	// Multiple files
	if *filepaths != "" {
		files := strings.Split(*filepaths, ",")
		results := []map[string]interface{}{}
		successCount, errorCount := 0, 0

		for _, f := range files {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			if err := c.indexer.IndexCodeFile(f); err != nil {
				results = append(results, map[string]interface{}{"file": f, "success": false, "error": err.Error()})
				errorCount++
			} else {
				results = append(results, map[string]interface{}{"file": f, "success": true})
				successCount++
			}
		}
		return outputSuccess(map[string]interface{}{
			"message":       fmt.Sprintf("indexed %d files, %d errors", successCount, errorCount),
			"results":       results,
			"success_count": successCount,
			"error_count":   errorCount,
		})
	}

	// Directory
	if *directory != "" {
		if err := validatePath(*directory); err != nil {
			return err
		}
		result, err := c.indexer.IndexCodeDirectory(*directory, *force)
		if err != nil {
			return outputError("index_failed", fmt.Sprintf("indexing failed: %v", err), "", "")
		}
		return outputSuccess(map[string]interface{}{
			"message":       "directory indexing completed",
			"directory":     *directory,
			"files_indexed": result.Indexed,
			"files_added":   result.Added,
			"files_updated": result.Updated,
			"files_skipped": result.Skipped,
			"files_failed":  result.Failed,
		})
	}

	return outputError("missing_argument",
		"filepath, --directory, or --filepaths is required", "",
		"devrag index-code <filepath> | --directory DIR | --filepaths FILE1,FILE2")
}

// --- list ---

func (c *CLI) runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fields := fs.String("fields", "", "comma-separated fields to include (e.g. filename,modified_at)")
	outputFmt := fs.String("output", "json", "output format: json | text")
	if err := fs.Parse(args); err != nil {
		return err
	}

	docs, err := c.db.ListDocuments()
	if err != nil {
		return outputError("list_failed", fmt.Sprintf("failed to list documents: %v", err), "", "")
	}

	documents := []map[string]interface{}{}
	for filename, modTime := range docs {
		documents = append(documents, map[string]interface{}{
			"filename":    filename,
			"modified_at": modTime.Format("2006-01-02T15:04:05Z"),
		})
	}

	if *outputFmt == "text" {
		for _, d := range documents {
			fmt.Printf("%s\t%s\n", d["modified_at"], d["filename"])
		}
		fmt.Printf("\nTotal: %d documents\n", len(documents))
		return nil
	}

	return outputSuccess(map[string]interface{}{
		"count":     len(documents),
		"documents": filterFields(documents, *fields),
	})
}

// --- delete ---

func (c *CLI) runDelete(args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	removeFile := fs.Bool("remove-file", false, "also delete the physical file")
	dryRun := fs.Bool("dry-run", false, "show what would be deleted without executing")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return outputError("missing_argument", "filename is required", "filename",
			"devrag delete <filename> [--dry-run] [--remove-file]")
	}

	filename := fs.Arg(0)
	if err := validatePath(filename); err != nil {
		return err
	}

	if *dryRun {
		return outputSuccess(map[string]interface{}{
			"dry_run":         true,
			"would_delete_db": filename,
			"would_delete_file": *removeFile,
		})
	}

	if err := c.db.DeleteDocument(filename); err != nil {
		return outputError("delete_failed", fmt.Sprintf("failed to delete from database: %v", err), "", "")
	}

	fileDeleted := false
	if *removeFile {
		if err := os.Remove(filename); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to delete file: %v\n", err)
		} else {
			fileDeleted = true
		}
	}

	return outputSuccess(map[string]interface{}{
		"message":      "document deleted from index",
		"filename":     filename,
		"file_deleted": fileDeleted,
	})
}

// --- reindex ---

func (c *CLI) runReindex(args []string) error {
	fs := flag.NewFlagSet("reindex", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "show what would be reindexed without executing")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return outputError("missing_argument", "filename is required", "filename",
			"devrag reindex <filename> [--dry-run]")
	}

	filename := fs.Arg(0)
	if err := validatePath(filename); err != nil {
		return err
	}

	if *dryRun {
		return outputSuccess(map[string]interface{}{
			"dry_run":        true,
			"would_reindex":  filename,
		})
	}

	if err := c.db.DeleteDocument(filename); err != nil {
		return outputError("delete_failed", fmt.Sprintf("failed to delete document: %v", err), "", "")
	}
	if err := c.indexer.IndexFile(filename); err != nil {
		return outputError("index_failed", fmt.Sprintf("failed to reindex: %v", err), "", "")
	}

	return outputSuccess(map[string]interface{}{
		"message":  "document reindexed successfully",
		"filename": filename,
	})
}

// --- search-relations ---

func (c *CLI) runSearchRelations(args []string) error {
	fs := flag.NewFlagSet("search-relations", flag.ContinueOnError)
	relationType := fs.String("type", "", "relation type: calls | imports | inherits")
	direction := fs.String("direction", "both", "direction: outgoing | incoming | both")
	outputFmt := fs.String("output", "json", "output format: json | text")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return outputError("missing_argument", "symbol is required", "symbol",
			"devrag search-relations <symbol> [--type TYPE] [--direction DIR]")
	}

	symbol := fs.Arg(0)
	if err := validateText(symbol, "symbol"); err != nil {
		return err
	}

	relations, err := c.db.FindSymbolRelations(symbol, *direction, *relationType)
	if err != nil {
		return outputError("search_failed", fmt.Sprintf("search failed: %v", err), "", "")
	}

	if *outputFmt == "text" {
		if len(relations) == 0 {
			fmt.Println("No relations found.")
			return nil
		}
		for _, r := range relations {
			fmt.Printf("[%s] %s (%s) -> %s (%s)\n",
				r.RelationType, r.SourceName, r.SourceFile, r.TargetName, r.TargetFile)
		}
		fmt.Printf("\nTotal: %d relations\n", len(relations))
		return nil
	}

	relMaps := []map[string]interface{}{}
	for _, r := range relations {
		relMaps = append(relMaps, map[string]interface{}{
			"relation_type": r.RelationType,
			"source_name":   r.SourceName,
			"source_file":   r.SourceFile,
			"target_name":   r.TargetName,
			"target_file":   r.TargetFile,
			"confidence":    r.Confidence,
		})
	}

	return outputSuccess(map[string]interface{}{
		"symbol":    symbol,
		"direction": *direction,
		"count":     len(relations),
		"relations": relMaps,
	})
}

// --- build-dictionary ---

func (c *CLI) runBuildDictionary(args []string) error {
	fs := flag.NewFlagSet("build-dictionary", flag.ContinueOnError)
	sourceLang := fs.String("source-lang", "ja", "source language")
	document := fs.String("document", "", "specific document path (default: all)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	extractor := indexer.NewDictionaryExtractor()
	var allMappings []vectordb.WordMapping

	if *document != "" {
		if err := validatePath(*document); err != nil {
			return err
		}
		content, err := os.ReadFile(*document)
		if err != nil {
			return outputError("read_failed", fmt.Sprintf("failed to read document: %v", err), "document", "")
		}
		allMappings = extractor.ExtractFromContent(string(content), *document, *sourceLang)
	} else {
		docs, err := c.db.ListDocuments()
		if err != nil {
			return outputError("list_failed", fmt.Sprintf("failed to list documents: %v", err), "", "")
		}
		for docPath := range docs {
			content, err := os.ReadFile(docPath)
			if err != nil {
				continue
			}
			lang := indexer.DetectLanguage(string(content))
			if lang == "mixed" || lang == *sourceLang {
				mappings := extractor.ExtractFromContent(string(content), docPath, *sourceLang)
				allMappings = append(allMappings, mappings...)
			}
		}
	}

	if len(allMappings) > 0 {
		if err := c.db.InsertWordMappings(allMappings); err != nil {
			return outputError("insert_failed", fmt.Sprintf("failed to insert mappings: %v", err), "", "")
		}
	}

	totalCount, _ := c.db.GetWordMappingCount()

	return outputSuccess(map[string]interface{}{
		"extracted_count":  len(allMappings),
		"total_dictionary": totalCount,
	})
}

// --- add-frontmatter ---

func (c *CLI) runAddFrontmatter(args []string) error {
	fs := flag.NewFlagSet("add-frontmatter", flag.ContinueOnError)
	domain := fs.String("domain", "", "domain: frontend | backend | mobile | infrastructure | other")
	docType := fs.String("doc-type", "", "type: spec | design | api | guide | note | other")
	language := fs.String("language", "", "language: go | typescript | python | rust | java | etc")
	tags := fs.String("tags", "", "tags (comma-separated)")
	project := fs.String("project", "", "project name")
	paramsJSON := fs.String("params", "", "JSON object with all parameters (alternative to flags)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return outputError("missing_argument", "filepath is required", "filepath",
			"devrag add-frontmatter <filepath> [--domain ...] [--params '{...}']")
	}

	filePath := fs.Arg(0)
	if err := validatePath(filePath); err != nil {
		return err
	}

	metadata := &frontmatter.Metadata{}

	// Support JSON payload input (Agent DX)
	if *paramsJSON != "" {
		if err := json.Unmarshal([]byte(*paramsJSON), metadata); err != nil {
			return outputError("invalid_json", fmt.Sprintf("failed to parse --params: %v", err), "params",
				`Example: --params '{"domain":"backend","tags":["api","auth"]}'`)
		}
	} else {
		metadata.Domain = *domain
		metadata.DocType = *docType
		metadata.Language = *language
		metadata.Project = *project
		if *tags != "" {
			for _, tag := range strings.Split(*tags, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					metadata.Tags = append(metadata.Tags, tag)
				}
			}
		}
	}

	if err := frontmatter.AddFrontmatter(filePath, metadata); err != nil {
		return outputError("frontmatter_failed", fmt.Sprintf("failed to add frontmatter: %v", err), "", "")
	}

	return outputSuccess(map[string]interface{}{
		"message": "frontmatter added successfully",
		"file":    filePath,
	})
}

// --- update-frontmatter ---

func (c *CLI) runUpdateFrontmatter(args []string) error {
	fs := flag.NewFlagSet("update-frontmatter", flag.ContinueOnError)
	domain := fs.String("domain", "", "domain: frontend | backend | mobile | infrastructure | other")
	docType := fs.String("doc-type", "", "type: spec | design | api | guide | note | other")
	language := fs.String("language", "", "language: go | typescript | python | rust | java | etc")
	tags := fs.String("tags", "", "tags (comma-separated)")
	project := fs.String("project", "", "project name")
	paramsJSON := fs.String("params", "", "JSON object with all parameters (alternative to flags)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return outputError("missing_argument", "filepath is required", "filepath",
			"devrag update-frontmatter <filepath> [--domain ...] [--params '{...}']")
	}

	filePath := fs.Arg(0)
	if err := validatePath(filePath); err != nil {
		return err
	}

	metadata := &frontmatter.Metadata{}

	if *paramsJSON != "" {
		if err := json.Unmarshal([]byte(*paramsJSON), metadata); err != nil {
			return outputError("invalid_json", fmt.Sprintf("failed to parse --params: %v", err), "params", "")
		}
	} else {
		metadata.Domain = *domain
		metadata.DocType = *docType
		metadata.Language = *language
		metadata.Project = *project
		if *tags != "" {
			for _, tag := range strings.Split(*tags, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					metadata.Tags = append(metadata.Tags, tag)
				}
			}
		}
	}

	if err := frontmatter.UpdateFrontmatter(filePath, metadata); err != nil {
		return outputError("frontmatter_failed", fmt.Sprintf("failed to update frontmatter: %v", err), "", "")
	}

	return outputSuccess(map[string]interface{}{
		"message": "frontmatter updated successfully",
		"file":    filePath,
	})
}

// --- schema ---

func (c *CLI) runSchema(args []string) error {
	schema := map[string]interface{}{
		"name":    "devrag",
		"version": version.Version,
		"flag_syntax": map[string]interface{}{
			"prefix":          "--",
			"short_prefix":    "-",
			"help_flags":      []string{"-h", "--help", "-help"},
			"flag_position":   "before positional arguments",
			"important_note":  "Flags MUST be placed BEFORE positional arguments. Example: 'devrag delete --dry-run file.md' (correct), NOT 'devrag delete file.md --dry-run' (flags after args are ignored).",
		},
		"global_flags": []cmdFlag{
			{Name: "config", Type: "string", Default: "config.json", Description: "Path to configuration file"},
			{Name: "version", Type: "bool", Description: "Show version and exit"},
		},
		"commands": []map[string]interface{}{
			{
				"name":        "search",
				"description": "Semantic search over indexed documents",
				"args":        []cmdArg{{Name: "query", Type: "string", Required: true, Description: "Natural language search query"}},
				"flags": []cmdFlag{
					{Name: "top-k", Type: "int", Default: "5", Description: "Max number of results"},
					{Name: "directory", Type: "string", Description: "Filter by directory"},
					{Name: "file-pattern", Type: "string", Description: "Filter by filename glob pattern"},
					{Name: "fields", Type: "string", Description: "Comma-separated fields to include"},
					{Name: "output", Type: "string", Default: "json", Enum: []string{"json", "text"}, Description: "Output format"},
				},
				"destructive": false,
			},
			{
				"name":        "index",
				"description": "Index a markdown file",
				"args":        []cmdArg{{Name: "filepath", Type: "string", Required: true, Description: "Path to markdown file"}},
				"destructive":  false,
			},
			{
				"name":        "index-code",
				"description": "Index source code files using AST analysis",
				"args":        []cmdArg{{Name: "filepath", Type: "string", Required: false, Description: "Single file to index"}},
				"flags": []cmdFlag{
					{Name: "directory", Type: "string", Description: "Directory to index recursively"},
					{Name: "filepaths", Type: "string", Description: "Comma-separated list of file paths"},
					{Name: "force", Type: "bool", Default: "false", Description: "Force re-index regardless of modification time"},
				},
				"destructive": false,
			},
			{
				"name":        "list",
				"description": "List all indexed documents",
				"flags": []cmdFlag{
					{Name: "fields", Type: "string", Description: "Comma-separated fields to include"},
					{Name: "output", Type: "string", Default: "json", Enum: []string{"json", "text"}, Description: "Output format"},
				},
				"destructive": false,
			},
			{
				"name":        "delete",
				"description": "Remove a document from the index",
				"args":        []cmdArg{{Name: "filename", Type: "string", Required: true, Description: "Filename to delete"}},
				"flags": []cmdFlag{
					{Name: "remove-file", Type: "bool", Default: "false", Description: "Also delete the physical file"},
					{Name: "dry-run", Type: "bool", Default: "false", Description: "Show what would be deleted without executing"},
				},
				"destructive": true,
			},
			{
				"name":        "reindex",
				"description": "Delete and re-index a document",
				"args":        []cmdArg{{Name: "filename", Type: "string", Required: true, Description: "Filename to reindex"}},
				"flags": []cmdFlag{
					{Name: "dry-run", Type: "bool", Default: "false", Description: "Show what would be reindexed without executing"},
				},
				"destructive": true,
			},
			{
				"name":        "search-relations",
				"description": "Search code symbol relationships",
				"args":        []cmdArg{{Name: "symbol", Type: "string", Required: true, Description: "Symbol name to search"}},
				"flags": []cmdFlag{
					{Name: "type", Type: "string", Enum: []string{"calls", "imports", "inherits"}, Description: "Relation type filter"},
					{Name: "direction", Type: "string", Default: "both", Enum: []string{"outgoing", "incoming", "both"}, Description: "Search direction"},
					{Name: "output", Type: "string", Default: "json", Enum: []string{"json", "text"}, Description: "Output format"},
				},
				"destructive": false,
			},
			{
				"name":        "build-dictionary",
				"description": "Build multilingual word mapping dictionary from indexed documents",
				"flags": []cmdFlag{
					{Name: "source-lang", Type: "string", Default: "ja", Description: "Source language"},
					{Name: "document", Type: "string", Description: "Specific document path (default: all)"},
				},
				"destructive": false,
			},
			{
				"name":        "add-frontmatter",
				"description": "Add metadata (frontmatter) to a markdown file",
				"args":        []cmdArg{{Name: "filepath", Type: "string", Required: true, Description: "Path to markdown file"}},
				"flags": []cmdFlag{
					{Name: "domain", Type: "string", Enum: []string{"frontend", "backend", "mobile", "infrastructure", "other"}, Description: "Domain"},
					{Name: "doc-type", Type: "string", Enum: []string{"spec", "design", "api", "guide", "note", "other"}, Description: "Document type"},
					{Name: "language", Type: "string", Description: "Programming language"},
					{Name: "tags", Type: "string", Description: "Tags (comma-separated)"},
					{Name: "project", Type: "string", Description: "Project name"},
					{Name: "params", Type: "json", Description: "JSON object with all parameters (alternative to flags)"},
				},
				"destructive": false,
			},
			{
				"name":        "update-frontmatter",
				"description": "Update metadata (frontmatter) in a markdown file",
				"args":        []cmdArg{{Name: "filepath", Type: "string", Required: true, Description: "Path to markdown file"}},
				"flags": []cmdFlag{
					{Name: "domain", Type: "string", Enum: []string{"frontend", "backend", "mobile", "infrastructure", "other"}, Description: "Domain"},
					{Name: "doc-type", Type: "string", Enum: []string{"spec", "design", "api", "guide", "note", "other"}, Description: "Document type"},
					{Name: "language", Type: "string", Description: "Programming language"},
					{Name: "tags", Type: "string", Description: "Tags (comma-separated)"},
					{Name: "project", Type: "string", Description: "Project name"},
					{Name: "params", Type: "json", Description: "JSON object with all parameters (alternative to flags)"},
				},
				"destructive": false,
			},
			{
				"name":        "schema",
				"description": "Return CLI schema as machine-readable JSON",
				"destructive":  false,
			},
		},
	}

	return outputJSON(schema)
}

// --- helpers ---

type cmdArg struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

type cmdFlag struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Default     string   `json:"default,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Description string   `json:"description"`
}

// outputSuccess writes a success response to stdout
func outputSuccess(data interface{}) error {
	return outputJSON(Response{Status: "ok", Data: data})
}

// outputError writes an error response to stdout and returns an error
func outputError(code, message, field, hint string) error {
	resp := Response{
		Status: "error",
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Field:   field,
			Hint:    hint,
		},
	}
	outputJSON(resp)
	return fmt.Errorf("%s: %s", code, message)
}

func outputJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// validatePath checks for path traversal and control characters
func validatePath(p string) error {
	if p == "" {
		return outputError("invalid_path", "path is empty", "path", "")
	}
	// Reject control characters
	for _, r := range p {
		if unicode.IsControl(r) && r != '\t' {
			return outputError("invalid_path",
				"path contains control characters", "path",
				"Remove invisible control characters from the path")
		}
	}
	// Reject path traversal
	if strings.Contains(p, "..") {
		return outputError("invalid_path",
			"path contains '..' (path traversal not allowed)", "path",
			"Use absolute paths or paths without '..'")
	}
	return nil
}

// validateText rejects control characters in text input
func validateText(s, fieldName string) error {
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return outputError("invalid_input",
				fmt.Sprintf("%s contains control characters", fieldName), fieldName,
				"Remove invisible control characters from the input")
		}
	}
	return nil
}

// filterFields limits map entries to specified fields (comma-separated)
func filterFields(items []map[string]interface{}, fields string) []map[string]interface{} {
	if fields == "" {
		return items
	}
	allowed := make(map[string]bool)
	for _, f := range strings.Split(fields, ",") {
		allowed[strings.TrimSpace(f)] = true
	}

	filtered := make([]map[string]interface{}, len(items))
	for i, item := range items {
		m := make(map[string]interface{})
		for k, v := range item {
			if allowed[k] {
				m[k] = v
			}
		}
		filtered[i] = m
	}
	return filtered
}

func resultsToMaps(results []vectordb.SearchResult) []map[string]interface{} {
	maps := make([]map[string]interface{}, len(results))
	for i, r := range results {
		maps[i] = map[string]interface{}{
			"document":   r.DocumentName,
			"chunk":      r.ChunkContent,
			"similarity": r.Similarity,
			"position":   r.Position,
		}
	}
	return maps
}

func printUsage() {
	fmt.Print(`DevRag - Lightweight RAG for Claude Code

Usage:
  devrag [command] [options]

Commands:
  serve                 Start MCP server (default when no command given)
  search <query>        Semantic search over indexed documents
  index <filepath>      Index a markdown file
  index-code            Index source code files (AST analysis)
  list                  List all indexed documents
  delete <filename>     Remove a document from the index
  reindex <filename>    Re-index a document
  search-relations      Search code symbol relationships
  build-dictionary      Build multilingual word mapping dictionary
  add-frontmatter       Add metadata to a markdown file
  update-frontmatter    Update metadata in a markdown file
  schema                Show CLI schema as JSON (machine-readable)
  help                  Show this help

Global Options:
  --config <path>       Path to config file (default: config.json)
  --version             Show version and exit

Output:
  All commands output JSON by default. Use --output text for human-readable output.
  Use --fields to limit output fields. Use --dry-run on destructive operations.

Flag Syntax:
  Flags use -- prefix (e.g. --top-k 10). Short aliases: -h, -help for help.
  IMPORTANT: Flags MUST be placed BEFORE positional arguments.
    Correct:   devrag delete --dry-run file.md
    Incorrect: devrag delete file.md --dry-run  (--dry-run is ignored)

Run 'devrag schema' for full machine-readable command specifications.
`)
}
