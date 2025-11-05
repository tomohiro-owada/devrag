package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	// Deprecated: Use DocumentPatterns instead
	DocumentsDir     string   `json:"documents_dir,omitempty"`
	DocumentPatterns []string `json:"document_patterns,omitempty"`
	DBPath           string   `json:"db_path"`
	ChunkSize        int      `json:"chunk_size"`
	SearchTopK       int      `json:"search_top_k"`
	Compute          struct {
		Device        string `json:"device"`
		FallbackToCPU bool   `json:"fallback_to_cpu"`
	} `json:"compute"`
	Model struct {
		Name       string `json:"name"`
		Dimensions int    `json:"dimensions"`
	} `json:"model"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	cfg := &Config{
		DocumentPatterns: []string{"./documents"},
		DBPath:           "./vectors.db",
		ChunkSize:        500,
		SearchTopK:       5,
	}
	cfg.Compute.Device = "auto"
	cfg.Compute.FallbackToCPU = true
	cfg.Model.Name = "multilingual-e5-small"
	cfg.Model.Dimensions = 384
	return cfg
}

// Load reads config from file or returns default
func Load() (*Config, error) {
	const configFile = "config.json"

	// Check if config.json exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "[INFO] config.json not found, using defaults\n")
		cfg := DefaultConfig()

		// Generate template
		if err := cfg.Save(configFile); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to generate config template: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "[INFO] Generated config template: %s\n", configFile)
		}

		return cfg, nil
	}

	// Read existing config
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// First unmarshal to check which fields are present
	var rawConfig map[string]interface{}
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] Invalid JSON in config.json: %v\n", err)
		fmt.Fprintf(os.Stderr, "[WARN] Using default configuration\n")
		return DefaultConfig(), nil
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] Invalid JSON in config.json: %v\n", err)
		fmt.Fprintf(os.Stderr, "[WARN] Using default configuration\n")
		return cfg, nil
	}

	// Migrate old format to new format
	_, hasOldField := rawConfig["documents_dir"]
	_, hasNewField := rawConfig["document_patterns"]

	if hasOldField && !hasNewField {
		fmt.Fprintf(os.Stderr, "[INFO] Migrating from documents_dir to document_patterns\n")
		cfg.DocumentPatterns = []string{cfg.DocumentsDir}
		cfg.DocumentsDir = "" // Clear deprecated field
	}

	// Validate that at least one pattern is configured
	if len(cfg.DocumentPatterns) == 0 {
		cfg.DocumentPatterns = []string{"./documents"}
	}

	fmt.Fprintf(os.Stderr, "[INFO] Loaded configuration from %s\n", configFile)
	return cfg, nil
}

// Save writes config to file
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Validate checks config values
func (c *Config) Validate() error {
	if c.ChunkSize <= 0 {
		return fmt.Errorf("chunk_size must be positive")
	}
	if c.SearchTopK <= 0 {
		return fmt.Errorf("search_top_k must be positive")
	}
	if c.Model.Dimensions <= 0 {
		return fmt.Errorf("model.dimensions must be positive")
	}
	if len(c.DocumentPatterns) == 0 {
		return fmt.Errorf("at least one document pattern must be specified")
	}
	return nil
}

// GetDocumentFiles expands all document patterns and returns matching markdown files
func (c *Config) GetDocumentFiles() ([]string, error) {
	files := make(map[string]bool) // Use map to deduplicate

	for _, pattern := range c.DocumentPatterns {
		matches, err := c.expandPattern(pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to expand pattern %s: %v\n", pattern, err)
			continue
		}
		for _, match := range matches {
			files[match] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(files))
	for file := range files {
		result = append(result, file)
	}

	return result, nil
}

// expandPattern expands a single pattern to matching markdown files
// Supports both directory paths (e.g., "./docs") and glob patterns (e.g., "./docs/**/*.md")
func (c *Config) expandPattern(pattern string) ([]string, error) {
	var files []string

	// Check if pattern looks like a directory (no wildcards and no .md extension)
	if !strings.Contains(pattern, "*") && !strings.Contains(pattern, "?") {
		// Treat as directory - walk it for all .md files
		err := filepath.Walk(pattern, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Continue despite errors
			}
			if !info.IsDir() && filepath.Ext(path) == ".md" {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return files, nil
	}

	// Pattern contains wildcards - need to handle ** specially
	if strings.Contains(pattern, "**") {
		return c.expandDoubleStarPattern(pattern)
	}

	// Simple glob pattern (no **)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	// Filter to only markdown files
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if !info.IsDir() && filepath.Ext(match) == ".md" {
			files = append(files, match)
		}
	}

	return files, nil
}

// expandDoubleStarPattern handles patterns with ** (recursive directory matching)
func (c *Config) expandDoubleStarPattern(pattern string) ([]string, error) {
	var files []string

	// Split pattern at **
	parts := strings.SplitN(pattern, "**", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid ** pattern: %s", pattern)
	}

	baseDir := parts[0]
	suffix := parts[1]

	// Clean up baseDir
	if baseDir == "" {
		baseDir = "."
	} else {
		baseDir = strings.TrimSuffix(baseDir, string(filepath.Separator))
	}

	// Walk the base directory
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue despite errors
		}

		if info.IsDir() {
			return nil
		}

		// Check if path matches the suffix pattern
		if c.matchesSuffix(path, baseDir, suffix) {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// matchesSuffix checks if a file path matches the suffix pattern after **
func (c *Config) matchesSuffix(path, baseDir, suffix string) bool {
	// Remove baseDir from path
	relPath, err := filepath.Rel(baseDir, path)
	if err != nil {
		return false
	}

	// If suffix is empty or just a separator, match all .md files
	if suffix == "" || suffix == string(filepath.Separator) || suffix == "/" {
		return filepath.Ext(path) == ".md"
	}

	// Clean suffix
	suffix = strings.TrimPrefix(suffix, string(filepath.Separator))
	suffix = strings.TrimPrefix(suffix, "/")

	// If suffix is just *.md, match all markdown files
	if suffix == "*.md" {
		return filepath.Ext(path) == ".md"
	}

	// Check if relPath matches the suffix pattern
	matched, err := filepath.Match(suffix, filepath.Base(relPath))
	if err != nil {
		return false
	}

	if matched && filepath.Ext(path) == ".md" {
		return true
	}

	// For patterns like "subdir/*.md", check full relative path
	matched, err = filepath.Match(suffix, relPath)
	if err != nil {
		return false
	}

	return matched && filepath.Ext(path) == ".md"
}

// GetBaseDirectories returns the base directories from all patterns
// This is useful for path validation
func (c *Config) GetBaseDirectories() []string {
	dirs := make(map[string]bool)

	for _, pattern := range c.DocumentPatterns {
		// Extract base directory from pattern
		baseDir := c.extractBaseDir(pattern)
		if baseDir != "" {
			absDir, err := filepath.Abs(baseDir)
			if err == nil {
				dirs[absDir] = true
			}
		}
	}

	result := make([]string, 0, len(dirs))
	for dir := range dirs {
		result = append(result, dir)
	}

	return result
}

// extractBaseDir extracts the base directory from a pattern
func (c *Config) extractBaseDir(pattern string) string {
	// Find the first wildcard
	wildcardIndex := strings.IndexAny(pattern, "*?")
	if wildcardIndex == -1 {
		// No wildcards - entire pattern is base directory
		return pattern
	}

	// Get the directory part before the wildcard
	baseDir := pattern[:wildcardIndex]
	baseDir = filepath.Dir(baseDir)

	if baseDir == "." {
		return baseDir
	}

	return baseDir
}
