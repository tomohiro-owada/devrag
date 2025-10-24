package config

import (
	"os"
	"testing"
)

func TestLoadConfig_NoFile(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.DocumentsDir != "./documents" {
		t.Errorf("Expected default documents_dir, got %s", cfg.DocumentsDir)
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
  "documents_dir": "./test_docs",
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

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.DocumentsDir != "./test_docs" {
		t.Errorf("Expected documents_dir './test_docs', got %s", cfg.DocumentsDir)
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
	if cfg.DocumentsDir != "./documents" {
		t.Errorf("Wrong default documents_dir: %s", cfg.DocumentsDir)
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
