package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_NoFile(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(cfg.DocumentPatterns) != 1 || cfg.DocumentPatterns[0] != "./documents" {
		t.Errorf("Expected default document_patterns [./documents], got %v", cfg.DocumentPatterns)
	}

	if cfg.ChunkSize != 500 {
		t.Errorf("Expected default chunk_size 500, got %d", cfg.ChunkSize)
	}

	if cfg.SearchTopK != 5 {
		t.Errorf("Expected default search_top_k 5, got %d", cfg.SearchTopK)
	}

	if cfg.Compute.Device != "auto" {
		t.Errorf("Expected default device 'auto', got %s", cfg.Compute.Device)
	}

	if !cfg.Compute.FallbackToCPU {
		t.Error("Expected default fallback_to_cpu to be true")
	}

	if cfg.Model.Name != "multilingual-e5-small" {
		t.Errorf("Expected default model name, got %s", cfg.Model.Name)
	}

	if cfg.Model.Dimensions != 384 {
		t.Errorf("Expected default dimensions 384, got %d", cfg.Model.Dimensions)
	}
}

func TestLoadConfig_Valid(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	// Create test config
	testConfig := `{
  "document_patterns": ["./test_docs", "./other_docs/**/*.md"],
  "db_path": "./test.db",
  "chunk_size": 300,
  "search_top_k": 10,
  "compute": {
    "device": "cpu",
    "fallback_to_cpu": false
  },
  "model": {
    "name": "test-model",
    "dimensions": 256
  }
}`

	if err := os.WriteFile("config.json", []byte(testConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(cfg.DocumentPatterns) != 2 {
		t.Errorf("Expected 2 document patterns, got %d", len(cfg.DocumentPatterns))
	}
	if cfg.DocumentPatterns[0] != "./test_docs" {
		t.Errorf("Expected first pattern './test_docs', got %s", cfg.DocumentPatterns[0])
	}
	if cfg.DocumentPatterns[1] != "./other_docs/**/*.md" {
		t.Errorf("Expected second pattern './other_docs/**/*.md', got %s", cfg.DocumentPatterns[1])
	}

	if cfg.DBPath != "./test.db" {
		t.Errorf("Expected db_path './test.db', got %s", cfg.DBPath)
	}

	if cfg.ChunkSize != 300 {
		t.Errorf("Expected chunk_size 300, got %d", cfg.ChunkSize)
	}

	if cfg.SearchTopK != 10 {
		t.Errorf("Expected search_top_k 10, got %d", cfg.SearchTopK)
	}

	if cfg.Compute.Device != "cpu" {
		t.Errorf("Expected device 'cpu', got %s", cfg.Compute.Device)
	}

	if cfg.Compute.FallbackToCPU {
		t.Error("Expected fallback_to_cpu to be false")
	}

	if cfg.Model.Name != "test-model" {
		t.Errorf("Expected model name 'test-model', got %s", cfg.Model.Name)
	}

	if cfg.Model.Dimensions != 256 {
		t.Errorf("Expected dimensions 256, got %d", cfg.Model.Dimensions)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		modify    func(*Config)
		wantError bool
	}{
		{
			name:      "default config is valid",
			modify:    func(c *Config) {},
			wantError: false,
		},
		{
			name: "negative chunk_size",
			modify: func(c *Config) {
				c.ChunkSize = -1
			},
			wantError: true,
		},
		{
			name: "zero chunk_size",
			modify: func(c *Config) {
				c.ChunkSize = 0
			},
			wantError: true,
		},
		{
			name: "negative search_top_k",
			modify: func(c *Config) {
				c.SearchTopK = -1
			},
			wantError: true,
		},
		{
			name: "zero search_top_k",
			modify: func(c *Config) {
				c.SearchTopK = 0
			},
			wantError: true,
		},
		{
			name: "negative dimensions",
			modify: func(c *Config) {
				c.Model.Dimensions = -1
			},
			wantError: true,
		},
		{
			name: "zero dimensions",
			modify: func(c *Config) {
				c.Model.Dimensions = 0
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)

			err := cfg.Validate()
			if tt.wantError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.json"

	cfg := DefaultConfig()
	cfg.ChunkSize = 600

	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Read and verify content
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Config file is empty")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("DefaultConfig should be valid, got %v", err)
	}

	// Verify all default values
	if len(cfg.DocumentPatterns) != 1 || cfg.DocumentPatterns[0] != "./documents" {
		t.Errorf("Wrong default document_patterns: %v", cfg.DocumentPatterns)
	}
	if cfg.DBPath != "./vectors.db" {
		t.Errorf("Wrong default db_path: %s", cfg.DBPath)
	}
	if cfg.ChunkSize != 500 {
		t.Errorf("Wrong default chunk_size: %d", cfg.ChunkSize)
	}
	if cfg.SearchTopK != 5 {
		t.Errorf("Wrong default search_top_k: %d", cfg.SearchTopK)
	}
}

func TestLoadConfig_CustomPath(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Create custom config file
	customConfigPath := tmpDir + "/custom-config.json"
	testConfig := `{
  "documents_dir": "./custom_docs",
  "db_path": "./custom.db",
  "chunk_size": 1000,
  "search_top_k": 20,
  "compute": {
    "device": "cuda",
    "fallback_to_cpu": true
  },
  "model": {
    "name": "custom-model",
    "dimensions": 512
  }
}`

	if err := os.WriteFile(customConfigPath, []byte(testConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Load with custom path
	cfg, err := Load(customConfigPath)
func TestLoadConfig_BackwardsCompatibility(t *testing.T) {
	// Test that old documents_dir format is migrated to document_patterns
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	// Create test config with old format
	oldConfig := `{
  "documents_dir": "./old_docs",
  "db_path": "./test.db",
  "chunk_size": 300,
  "search_top_k": 10,
  "compute": {
    "device": "cpu",
    "fallback_to_cpu": false
  },
  "model": {
    "name": "test-model",
    "dimensions": 256
  }
}`

	if err := os.WriteFile("config.json", []byte(oldConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify custom values are loaded
	if cfg.DocumentsDir != "./custom_docs" {
		t.Errorf("Expected documents_dir './custom_docs', got %s", cfg.DocumentsDir)
	}

	if cfg.DBPath != "./custom.db" {
		t.Errorf("Expected db_path './custom.db', got %s", cfg.DBPath)
	}

	if cfg.ChunkSize != 1000 {
		t.Errorf("Expected chunk_size 1000, got %d", cfg.ChunkSize)
	}

	if cfg.SearchTopK != 20 {
		t.Errorf("Expected search_top_k 20, got %d", cfg.SearchTopK)
	}

	if cfg.Compute.Device != "cuda" {
		t.Errorf("Expected device 'cuda', got %s", cfg.Compute.Device)
	}

	if cfg.Model.Name != "custom-model" {
		t.Errorf("Expected model name 'custom-model', got %s", cfg.Model.Name)
	}

	if cfg.Model.Dimensions != 512 {
		t.Errorf("Expected dimensions 512, got %d", cfg.Model.Dimensions)
	}
}

func TestLoadConfig_CustomPath_NotFound(t *testing.T) {
	// Test loading from non-existent custom path
	cfg, err := Load("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("Expected no error (should return defaults), got %v", err)
	}

	// Should return default config when file doesn't exist
	if cfg.DocumentsDir != "./documents" {
		t.Errorf("Expected default documents_dir, got %s", cfg.DocumentsDir)
	}

	if cfg.ChunkSize != 500 {
		t.Errorf("Expected default chunk_size 500, got %d", cfg.ChunkSize)
	}
}

func TestLoadConfig_CustomPath_InvalidJSON(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Create invalid config file
	customConfigPath := tmpDir + "/invalid-config.json"
	invalidJSON := `{
  "documents_dir": "./docs",
  "invalid json here
}`

	if err := os.WriteFile(customConfigPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Should return default config and not error
	cfg, err := Load(customConfigPath)
	if err != nil {
		t.Fatalf("Expected no error (should fallback to defaults), got %v", err)
	}

	// Should have default values due to fallback
	if cfg.DocumentsDir != "./documents" {
		t.Errorf("Expected default documents_dir, got %s", cfg.DocumentsDir)
	}
}

func TestLoadConfig_RelativeAndAbsolutePaths(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	testConfig := `{
  "documents_dir": "./test_docs",
  "db_path": "./test.db",
  "chunk_size": 700,
  "search_top_k": 15
}`

	// Test with relative path
	relPath := "test-relative-config.json"
	fullPath := tmpDir + "/" + relPath

	if err := os.WriteFile(fullPath, []byte(testConfig), 0644); err != nil {
		t.Fatal(err)
	}

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	cfg, err := Load(relPath)
	if err != nil {
		t.Fatalf("Expected no error with relative path, got %v", err)
	}

	if cfg.ChunkSize != 700 {
		t.Errorf("Expected chunk_size 700, got %d", cfg.ChunkSize)
	}

	// Test with absolute path
	absPath := tmpDir + "/test-absolute-config.json"
	if err := os.WriteFile(absPath, []byte(testConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err = Load(absPath)
	if err != nil {
		t.Fatalf("Expected no error with absolute path, got %v", err)
	}

	if cfg.ChunkSize != 700 {
		t.Errorf("Expected chunk_size 700 from absolute path, got %d", cfg.ChunkSize)
	}
	// Verify migration
	if len(cfg.DocumentPatterns) != 1 {
		t.Errorf("Expected 1 document pattern after migration, got %d", len(cfg.DocumentPatterns))
	}
	if cfg.DocumentPatterns[0] != "./old_docs" {
		t.Errorf("Expected pattern './old_docs', got %s", cfg.DocumentPatterns[0])
	}
	if cfg.DocumentsDir != "" {
		t.Errorf("Expected deprecated DocumentsDir to be cleared, got %s", cfg.DocumentsDir)
	}
}

func TestGetDocumentFiles(t *testing.T) {
	// Create temporary test directory structure
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"docs/file1.md",
		"docs/subdir/file2.md",
		"notes/file3.md",
		"other/file4.txt", // not markdown
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(tmpDir, file)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte("# Test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name            string
		patterns        []string
		expectedCount   int
		shouldContain   []string
		shouldNotContain []string
	}{
		{
			name:          "single directory pattern",
			patterns:      []string{tmpDir + "/docs"},
			expectedCount: 2,
			shouldContain: []string{"file1.md", "file2.md"},
		},
		{
			name:          "glob pattern with **",
			patterns:      []string{tmpDir + "/docs/**/*.md"},
			expectedCount: 2,
			shouldContain: []string{"file1.md", "file2.md"},
		},
		{
			name:          "multiple patterns",
			patterns:      []string{tmpDir + "/docs", tmpDir + "/notes"},
			expectedCount: 3,
			shouldContain: []string{"file1.md", "file2.md", "file3.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				DocumentPatterns: tt.patterns,
			}

			files, err := cfg.GetDocumentFiles()
			if err != nil {
				t.Fatalf("GetDocumentFiles failed: %v", err)
			}

			if len(files) != tt.expectedCount {
				t.Errorf("Expected %d files, got %d: %v", tt.expectedCount, len(files), files)
			}

			// Check that expected files are present
			for _, expected := range tt.shouldContain {
				found := false
				for _, file := range files {
					if contains(file, expected) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find file containing '%s', but it was not found in %v", expected, files)
				}
			}
		})
	}
}

func TestGetBaseDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		patterns      []string
		expectedCount int
	}{
		{
			name:          "single directory",
			patterns:      []string{tmpDir + "/docs"},
			expectedCount: 1,
		},
		{
			name:          "glob pattern",
			patterns:      []string{tmpDir + "/docs/**/*.md"},
			expectedCount: 1,
		},
		{
			name:          "multiple patterns with subdirectory",
			patterns:      []string{tmpDir + "/docs/**/*.md", tmpDir + "/docs/subdir/*.md"},
			expectedCount: 2, // Different base directories (docs and docs/subdir)
		},
		{
			name:          "multiple patterns different bases",
			patterns:      []string{tmpDir + "/docs", tmpDir + "/notes"},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				DocumentPatterns: tt.patterns,
			}

			dirs := cfg.GetBaseDirectories()
			if len(dirs) != tt.expectedCount {
				t.Errorf("Expected %d base directories, got %d: %v", tt.expectedCount, len(dirs), dirs)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[len(s)-len(substr):] == substr || s[:len(substr)] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
