package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tomohiro-owada/devrag/internal/cli"
	"github.com/tomohiro-owada/devrag/internal/config"
	"github.com/tomohiro-owada/devrag/internal/embedder"
	"github.com/tomohiro-owada/devrag/internal/indexer"
	"github.com/tomohiro-owada/devrag/internal/mcp"
	"github.com/tomohiro-owada/devrag/internal/updater"
	"github.com/tomohiro-owada/devrag/internal/vectordb"
	"github.com/tomohiro-owada/devrag/internal/version"
)

func main() {
	// Extract global flags before subcommand
	configPath := "config.json"
	cliModelDir := ""
	args := os.Args[1:]

	// Parse global flags manually (before subcommand)
	var filteredArgs []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--version", "-version":
			fmt.Printf("devrag version %s\n", version.Version)
			os.Exit(0)
		case "--config", "-config":
			if i+1 < len(args) {
				configPath = args[i+1]
				i++ // skip next arg
			} else {
				fmt.Fprintf(os.Stderr, "[FATAL] --config requires a path argument\n")
				os.Exit(1)
			}
		case "--model-dir":
			if i+1 < len(args) {
				cliModelDir = args[i+1]
				i++ // skip next arg
			} else {
				fmt.Fprintf(os.Stderr, "[FATAL] --model-dir requires a path argument\n")
				os.Exit(1)
			}
		default:
			filteredArgs = append(filteredArgs, args[i])
		}
	}

	// Determine mode: MCP server (default/serve) or CLI subcommand
	isMCP := len(filteredArgs) == 0 || filteredArgs[0] == "serve"

	if isMCP {
		fmt.Fprintf(os.Stderr, "[INFO] DevRag v%s starting (MCP mode)...\n", version.Version)
	} else {
		fmt.Fprintf(os.Stderr, "[INFO] DevRag v%s\n", version.Version)
	}

	// Check for updates (synchronous, shown immediately after startup message)
	updater.CheckForUpdate(version.Version, "")

	// 1. Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] Failed to load config: %v\n", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	if isMCP {
		fmt.Fprintf(os.Stderr, "[INFO] Configuration loaded successfully\n")
		fmt.Fprintf(os.Stderr, "[INFO] Document patterns: %v\n", cfg.DocumentPatterns)
		fmt.Fprintf(os.Stderr, "[INFO] Database path: %s\n", cfg.DBPath)
		fmt.Fprintf(os.Stderr, "[INFO] Model: %s (dimensions: %d)\n", cfg.Model.Name, cfg.Model.Dimensions)
		fmt.Fprintf(os.Stderr, "[INFO] Device: %s\n", cfg.Compute.Device)
	}

	// 2. Resolve model directory and download model files if needed
	modelCacheDir := config.ResolveModelDir(cliModelDir, cfg)
	modelDir := filepath.Join(modelCacheDir, "multilingual-e5-small")
	fmt.Fprintf(os.Stderr, "[INFO] Model directory: %s\n", modelDir)
	if err := embedder.DownloadModelFiles(modelDir); err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] Failed to download model files: %v\n", err)
		os.Exit(1)
	}

	// 3. Detect device
	device := embedder.DetectDevice(cfg.Compute.Device, cfg.Compute.FallbackToCPU)
	if isMCP {
		fmt.Fprintf(os.Stderr, "[INFO] Using device: %s\n", device)
	}

	// 4. Initialize components
	baseDirs := cfg.GetBaseDirectories()
	for _, dir := range baseDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to create directory %s: %v\n", dir, err)
		}
	}

	db, err := vectordb.Init(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	var emb embedder.Embedder
	modelPath := filepath.Join(modelDir, "model.onnx")
	if _, err := os.Stat(modelPath); err == nil {
		emb, err = embedder.NewONNXEmbedder(modelPath, device)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[FATAL] Failed to initialize embedder: %v\n", err)
			os.Exit(1)
		}
		defer emb.Close()
		if isMCP {
			fmt.Fprintf(os.Stderr, "[INFO] Loaded ONNX model from %s\n", modelPath)
		}
	} else {
		fmt.Fprintf(os.Stderr, "[WARN] Model not found at %s, using mock embedder\n", modelPath)
		emb = &embedder.MockEmbedder{}
	}

	idx := indexer.NewIndexer(db, emb, cfg)

	// 5. Sync documents (MCP mode does full sync; CLI skips for speed)
	if isMCP {
		fmt.Fprintf(os.Stderr, "[INFO] Syncing documents...\n")
		syncResult, err := idx.Sync()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Sync error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "[INFO] Sync complete: +%d, ~%d, -%d\n",
				len(syncResult.Added),
				len(syncResult.Updated),
				len(syncResult.Deleted))
		}

		// 6. Start MCP server
		fmt.Fprintf(os.Stderr, "[INFO] Starting MCP server...\n")
		server := mcp.NewMCPServer(idx, db, emb, cfg)
		if err := server.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "[FATAL] MCP server error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// CLI mode
		c := cli.New(db, emb, idx, cfg)
		if err := c.Run(filteredArgs); err != nil {
			os.Exit(1)
		}
	}
}
