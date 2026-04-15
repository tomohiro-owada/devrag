package embedder

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	ort "github.com/yalue/onnxruntime_go"
)

// CoreMLTimeout is the maximum time to wait for CoreML model compilation
// during session creation. CoreML may need to compile the model for GPU/ANE
// on first run, which can hang indefinitely on some configurations.
var CoreMLTimeout = 60 * time.Second

type ONNXEmbedder struct {
	session    *ort.DynamicAdvancedSession
	tokenizer  *Tokenizer
	device     Device
	modelDir   string
	outputDim  int
	maxLength  int
}

// findBundledLibrary looks for the ONNX Runtime shared library bundled next
// to the executable. Returns empty string if not found.
func findBundledLibrary() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	exeDir := filepath.Dir(exe)

	// Platform-specific library names
	var names []string
	switch runtime.GOOS {
	case "darwin":
		names = []string{"libonnxruntime.dylib", "libonnxruntime.1.22.0.dylib"}
	case "linux":
		names = []string{"libonnxruntime.so", "libonnxruntime.so.1.22.0"}
	case "windows":
		names = []string{"onnxruntime.dll"}
	}

	for _, name := range names {
		p := filepath.Join(exeDir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// NewONNXEmbedder creates a new ONNX embedder
func NewONNXEmbedder(modelPath string, device Device) (*ONNXEmbedder, error) {
	fmt.Fprintf(os.Stderr, "[INFO] Initializing ONNX Runtime (%s)...\n", device)

	// Point ONNX Runtime to the bundled shared library if available.
	// This is required for CoreML (macOS) and CUDA (Linux) execution providers
	// which are only present in the official Microsoft releases, not Homebrew.
	if libPath := findBundledLibrary(); libPath != "" {
		fmt.Fprintf(os.Stderr, "[INFO] Using bundled ONNX Runtime: %s\n", libPath)
		ort.SetSharedLibraryPath(libPath)
	}

	// Initialize ONNX Runtime
	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}

	// Create session options
	options, err := ort.NewSessionOptions()
	if err != nil {
		return nil, fmt.Errorf("failed to create session options: %w", err)
	}
	defer options.Destroy()

	// Set execution provider based on device
	usedCoreML := false
	if device == GPU {
		fmt.Fprintf(os.Stderr, "[INFO] GPU execution provider requested\n")
		if runtime.GOOS == "darwin" {
			// Use the V2 API (recommended since ONNX Runtime 1.20.0).
			// Use CPUAndGPU instead of All to avoid Neural Engine
			// compilation hangs on some Apple Silicon configurations.
			coreMLOpts := map[string]string{
				"ModelFormat":    "MLProgram",
				"MLComputeUnits": "CPUAndGPU",
			}
			if err := options.AppendExecutionProviderCoreMLV2(coreMLOpts); err != nil {
				fmt.Fprintf(os.Stderr, "[WARN] CoreML not available, falling back to CPU: %v\n", err)
			} else {
				usedCoreML = true
				fmt.Fprintf(os.Stderr, "[INFO] CoreML execution provider enabled (MLProgram, CPU_AND_GPU)\n")
			}
		}
	}

	// Configure CPU thread count. Override with DEVRAG_THREADS env var.
	numThreads := 4
	if t := os.Getenv("DEVRAG_THREADS"); t != "" {
		if n, err := strconv.Atoi(t); err == nil && n > 0 {
			numThreads = n
		}
	}
	fmt.Fprintf(os.Stderr, "[INFO] CPU threads: %d (set DEVRAG_THREADS to override)\n", numThreads)
	if err := options.SetIntraOpNumThreads(numThreads); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] Failed to set intra-op threads: %v\n", err)
	}
	if err := options.SetInterOpNumThreads(numThreads); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] Failed to set inter-op threads: %v\n", err)
	}

	// Load model with timeout protection.
	// CoreML model compilation can hang indefinitely on some configurations,
	// so we run session creation in a goroutine with a timeout.
	inputNames := []string{"input_ids", "attention_mask", "token_type_ids"}
	outputNames := []string{"last_hidden_state"}

	session, err := createSessionWithTimeout(modelPath, inputNames, outputNames, options, usedCoreML)
	if err != nil && usedCoreML {
		// CoreML failed or timed out — retry with CPU only
		fmt.Fprintf(os.Stderr, "[WARN] CoreML session failed (%v), retrying with CPU only...\n", err)
		options.Destroy()

		options, err = ort.NewSessionOptions()
		if err != nil {
			return nil, fmt.Errorf("failed to create session options: %w", err)
		}
		defer options.Destroy()

		if err := options.SetIntraOpNumThreads(numThreads); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to set intra-op threads: %v\n", err)
		}
		if err := options.SetInterOpNumThreads(numThreads); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to set inter-op threads: %v\n", err)
		}

		session, err = createSessionWithTimeout(modelPath, inputNames, outputNames, options, false)
		if err != nil {
			return nil, fmt.Errorf("failed to load model (CPU fallback): %w", err)
		}
		device = CPU
		fmt.Fprintf(os.Stderr, "[INFO] Successfully fell back to CPU\n")
	} else if err != nil {
		return nil, fmt.Errorf("failed to load model: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[INFO] ONNX model loaded successfully\n")

	// Get model directory
	modelDir := filepath.Dir(modelPath)

	// Load tokenizer
	fmt.Fprintf(os.Stderr, "[INFO] Loading tokenizer...\n")
	tokenizer, err := NewTokenizerFromModelDir(modelDir)
	if err != nil {
		session.Destroy()
		return nil, fmt.Errorf("failed to load tokenizer: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[INFO] Tokenizer loaded successfully (vocab size: %d)\n", tokenizer.GetVocabSize())

	return &ONNXEmbedder{
		session:    session,
		tokenizer:  tokenizer,
		device:     device,
		modelDir:   modelDir,
		outputDim:  384, // multilingual-e5-small output dimension
		maxLength:  512,
	}, nil
}

// Embed embeds a single text
func (e *ONNXEmbedder) Embed(text string) (embedding []float32, err error) {
	// Add query prefix for e5 models (improves search quality)
	// For documents, no prefix is needed
	// text = "query: " + text

	// Recover from panics in the upstream tokenizer library.
	// sugarme/tokenizer v0.3.0 has known bugs with multi-byte Unicode
	// sequences at chunk boundaries (see upstream issues #77, #78).
	// The tokenizer methods already sanitize and recover, but we keep
	// this outer recovery as defense-in-depth.
	defer func() {
		if r := recover(); r != nil {
			embedding = nil
			err = fmt.Errorf("tokenizer panic (upstream bug): %v", r)
		}
	}()

	// Tokenize the text (sanitization happens inside TokenizeWithAttentionMask)
	inputIDs32, attentionMask32, err := e.tokenizer.TokenizeWithAttentionMask(text)
	if err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}

	// Prepare input tensors
	batchSize := 1
	seqLength := len(inputIDs32)

	// Convert int32 to int64 for ONNX model
	inputIDs := make([]int64, seqLength)
	attentionMask := make([]int64, seqLength)
	tokenTypeIDs := make([]int64, seqLength) // All zeros for single sequence
	for i := 0; i < seqLength; i++ {
		inputIDs[i] = int64(inputIDs32[i])
		attentionMask[i] = int64(attentionMask32[i])
		tokenTypeIDs[i] = 0 // Always 0 for single sequence
	}

	// Create input_ids tensor [batch_size, seq_length]
	inputIDsShape := []int64{int64(batchSize), int64(seqLength)}
	inputIDsTensor, err := ort.NewTensor(inputIDsShape, inputIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	defer inputIDsTensor.Destroy()

	// Create attention_mask tensor [batch_size, seq_length]
	attentionMaskShape := []int64{int64(batchSize), int64(seqLength)}
	attentionMaskTensor, err := ort.NewTensor(attentionMaskShape, attentionMask)
	if err != nil {
		return nil, fmt.Errorf("failed to create attention_mask tensor: %w", err)
	}
	defer attentionMaskTensor.Destroy()

	// Create token_type_ids tensor [batch_size, seq_length]
	tokenTypeIDsShape := []int64{int64(batchSize), int64(seqLength)}
	tokenTypeIDsTensor, err := ort.NewTensor(tokenTypeIDsShape, tokenTypeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create token_type_ids tensor: %w", err)
	}
	defer tokenTypeIDsTensor.Destroy()

	// Run inference
	// Output tensors will be allocated by the session
	outputs := []ort.Value{nil}
	err = e.session.Run([]ort.Value{inputIDsTensor, attentionMaskTensor, tokenTypeIDsTensor}, outputs)
	if err != nil {
		return nil, fmt.Errorf("inference failed: %w", err)
	}
	defer func() {
		for _, output := range outputs {
			if output != nil {
				output.Destroy()
			}
		}
	}()

	// Extract output tensor [batch_size, seq_length, hidden_size]
	if len(outputs) == 0 || outputs[0] == nil {
		return nil, fmt.Errorf("no output from model")
	}

	// Cast to Tensor[float32] to access the data
	outputTensor, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("unexpected output type: %T", outputs[0])
	}

	// Get the data
	outputFloat32 := outputTensor.GetData()

	// The output shape is [batch_size, seq_length, hidden_size]
	// We need to perform mean pooling over the sequence dimension
	// with attention mask to get a single vector per text
	embedding = meanPooling(outputFloat32, attentionMask, seqLength, e.outputDim)

	// Normalize the embedding (L2 normalization)
	embedding = normalize(embedding)

	return embedding, nil
}

// EmbedBatch embeds multiple texts.
// If individual texts fail (e.g. due to upstream tokenizer bugs), a zero vector
// is used as a fallback and a warning is logged. The batch as a whole does not fail.
func (e *ONNXEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// For now, process one by one
	// TODO: Implement true batch processing with padding
	results := make([][]float32, len(texts))
	skipped := 0
	for i, text := range texts {
		embedding, err := e.Embed(text)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Skipping chunk %d due to embedding error: %v\n", i, err)
			// Use zero vector as fallback so the chunk is still stored
			results[i] = make([]float32, e.outputDim)
			skipped++
			continue
		}
		results[i] = embedding
	}

	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "[WARN] %d/%d chunks used fallback zero vectors due to tokenizer errors\n", skipped, len(texts))
	}

	return results, nil
}

// meanPooling performs mean pooling over sequence dimension with attention mask
func meanPooling(hiddenStates []float32, attentionMask []int64, seqLength, hiddenSize int) []float32 {
	result := make([]float32, hiddenSize)

	// Sum all token embeddings weighted by attention mask
	maskSum := float32(0)
	for t := 0; t < seqLength; t++ {
		mask := float32(attentionMask[t])
		maskSum += mask

		for h := 0; h < hiddenSize; h++ {
			idx := t*hiddenSize + h
			result[h] += hiddenStates[idx] * mask
		}
	}

	// Average by number of real tokens (not padding)
	if maskSum > 0 {
		for h := 0; h < hiddenSize; h++ {
			result[h] /= maskSum
		}
	}

	return result
}

// normalize performs L2 normalization
func normalize(vec []float32) []float32 {
	var norm float32
	for _, v := range vec {
		norm += v * v
	}

	if norm == 0 {
		return vec
	}

	norm = float32(1.0) / float32(sqrt(float64(norm)))
	result := make([]float32, len(vec))
	for i, v := range vec {
		result[i] = v * norm
	}

	return result
}

// sqrt computes square root
func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	// Use Newton's method for square root
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// sessionResult holds the result of an async session creation
type sessionResult struct {
	session *ort.DynamicAdvancedSession
	err     error
}

// createSessionWithTimeout creates an ONNX session with a timeout.
// When CoreML is involved, model compilation can hang indefinitely.
func createSessionWithTimeout(modelPath string, inputNames, outputNames []string, options *ort.SessionOptions, withCoreML bool) (*ort.DynamicAdvancedSession, error) {
	timeout := CoreMLTimeout
	if !withCoreML {
		// CPU-only sessions are fast; use a shorter timeout
		timeout = 30 * time.Second
	}

	ch := make(chan sessionResult, 1)
	go func() {
		session, err := ort.NewDynamicAdvancedSession(modelPath, inputNames, outputNames, options)
		ch <- sessionResult{session, err}
	}()

	select {
	case result := <-ch:
		return result.session, result.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("session creation timed out after %v (CoreML model compilation may be hanging)", timeout)
	}
}

// Close closes the embedder and releases resources
func (e *ONNXEmbedder) Close() error {
	if e.tokenizer != nil {
		e.tokenizer.Close()
	}
	if e.session != nil {
		e.session.Destroy()
	}
	// Note: DestroyEnvironment should be called when the application exits
	// ort.DestroyEnvironment()
	return nil
}
