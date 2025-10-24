package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	DocumentsDir string `json:"documents_dir"`
	DBPath       string `json:"db_path"`
	ChunkSize    int    `json:"chunk_size"`
	SearchTopK   int    `json:"search_top_k"`
	Compute      struct {
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
		DocumentsDir: "./documents",
		DBPath:       "./vectors.db",
		ChunkSize:    500,
		SearchTopK:   5,
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

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] Invalid JSON in config.json: %v\n", err)
		fmt.Fprintf(os.Stderr, "[WARN] Using default configuration\n")
		return cfg, nil
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
	return nil
}
