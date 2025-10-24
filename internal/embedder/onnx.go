package embedder

import (
	"fmt"
	"os"
	"path/filepath"

	ort "github.com/yalue/onnxruntime_go"
)

type ONNXEmbedder struct {
	session    *ort.DynamicAdvancedSession
	tokenizer  *Tokenizer
	device     Device
	modelDir   string
	outputDim  int
	maxLength  int
}

// NewONNXEmbedder creates a new ONNX embedder
func NewONNXEmbedder(modelPath string, device Device) (*ONNXEmbedder, error) {
	fmt.Fprintf(os.Stderr, "[INFO] Initializing ONNX Runtime (%s)...\n", device)

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
	if device == GPU {
		fmt.Fprintf(os.Stderr, "[INFO] GPU execution provider requested\n")
		// Try to enable CoreML on macOS or CUDA on other platforms
		// Note: This may fail if the execution provider is not available
		if err := options.AppendExecutionProviderCoreML(0); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] CoreML not available, falling back to CPU: %v\n", err)
		}
	}

	// Configure session options for better performance
	if err := options.SetIntraOpNumThreads(4); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] Failed to set intra-op threads: %v\n", err)
	}
	if err := options.SetInterOpNumThreads(4); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] Failed to set inter-op threads: %v\n", err)
	}

	// Load model
	// For multilingual-e5-small, the inputs are: input_ids, attention_mask, token_type_ids
	// Output is: last_hidden_state
	session, err := ort.NewDynamicAdvancedSession(modelPath,
		[]string{"input_ids", "attention_mask", "token_type_ids"},
		[]string{"last_hidden_state"},
		options)
	if err != nil {
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
func (e *ONNXEmbedder) Embed(text string) ([]float32, error) {
	// Add query prefix for e5 models (improves search quality)
	// For documents, no prefix is needed
	// text = "query: " + text

	// Tokenize the text
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
	embedding := meanPooling(outputFloat32, attentionMask, seqLength, e.outputDim)

	// Normalize the embedding (L2 normalization)
	embedding = normalize(embedding)

	return embedding, nil
}

// EmbedBatch embeds multiple texts
func (e *ONNXEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// For now, process one by one
	// TODO: Implement true batch processing with padding
	results := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := e.Embed(text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		results[i] = embedding
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
