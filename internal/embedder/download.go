package embedder

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	modelURL      = "https://huggingface.co/intfloat/multilingual-e5-small/resolve/main/onnx/model.onnx"
	tokenizerURL  = "https://huggingface.co/intfloat/multilingual-e5-small/resolve/main/tokenizer.json"
	configURL     = "https://huggingface.co/intfloat/multilingual-e5-small/resolve/main/config.json"
	specialTokensURL = "https://huggingface.co/intfloat/multilingual-e5-small/resolve/main/special_tokens_map.json"
	tokenizerConfigURL = "https://huggingface.co/intfloat/multilingual-e5-small/resolve/main/tokenizer_config.json"
)

// DownloadModelFiles downloads model files from Hugging Face if they don't exist
func DownloadModelFiles(modelDir string) error {
	fmt.Fprintf(os.Stderr, "[INFO] Checking model files in %s...\n", modelDir)

	// Create models directory if it doesn't exist
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create models directory: %w", err)
	}

	files := map[string]string{
		"model.onnx":               modelURL,
		"tokenizer.json":           tokenizerURL,
		"config.json":              configURL,
		"special_tokens_map.json":  specialTokensURL,
		"tokenizer_config.json":    tokenizerConfigURL,
	}

	needsDownload := false
	for filename := range files {
		path := filepath.Join(modelDir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			needsDownload = true
			break
		}
	}

	if !needsDownload {
		fmt.Fprintf(os.Stderr, "[INFO] All model files found, skipping download\n")
		return nil
	}

	fmt.Fprintf(os.Stderr, "[INFO] Downloading model files from Hugging Face...\n")
	fmt.Fprintf(os.Stderr, "[INFO] This is a one-time download (~450MB), please wait...\n")

	for filename, url := range files {
		path := filepath.Join(modelDir, filename)

		// Skip if file already exists
		if _, err := os.Stat(path); err == nil {
			fmt.Fprintf(os.Stderr, "[INFO] File already exists: %s\n", filename)
			continue
		}

		fmt.Fprintf(os.Stderr, "[INFO] Downloading %s...\n", filename)
		if err := downloadFile(path, url); err != nil {
			return fmt.Errorf("failed to download %s: %w", filename, err)
		}
		fmt.Fprintf(os.Stderr, "[INFO] Downloaded %s\n", filename)
	}

	fmt.Fprintf(os.Stderr, "[INFO] Model download complete!\n")
	return nil
}

// downloadFile downloads a file from URL to destination with progress
func downloadFile(filepath string, url string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer with progress
	size := resp.ContentLength
	downloaded := int64(0)
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)

			// Print progress every 10MB
			if size > 0 && downloaded%(10*1024*1024) == 0 {
				progress := float64(downloaded) / float64(size) * 100
				fmt.Fprintf(os.Stderr, "  Progress: %.1f%% (%d MB / %d MB)\r",
					progress, downloaded/(1024*1024), size/(1024*1024))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	if size > 0 {
		fmt.Fprintf(os.Stderr, "  Progress: 100.0%% (%d MB / %d MB)\n",
			size/(1024*1024), size/(1024*1024))
	}

	return nil
}
