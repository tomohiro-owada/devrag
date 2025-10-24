package embedder

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
)

// Tokenizer wraps the HuggingFace tokenizer for text tokenization
type Tokenizer struct {
	tk            *tokenizer.Tokenizer
	maxLength     int
	padTokenID    int32
	clsTokenID    int32
	sepTokenID    int32
	maskTokenID   int32
	attentionMask bool
}

// TokenizerConfig holds tokenizer configuration
type TokenizerConfig struct {
	MaxLength     int
	PadTokenID    int32
	ClsTokenID    int32
	SepTokenID    int32
	MaskTokenID   int32
	AttentionMask bool
}

// NewTokenizer creates a new tokenizer from a tokenizer.json file
func NewTokenizer(tokenizerPath string, config TokenizerConfig) (*Tokenizer, error) {
	// Load tokenizer from JSON file using pretrained package
	tk, err := pretrained.FromFile(tokenizerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load tokenizer from %s: %w", tokenizerPath, err)
	}

	// Configure truncation
	if config.MaxLength > 0 {
		tk.WithTruncation(&tokenizer.TruncationParams{
			MaxLength: config.MaxLength,
		})
	}

	// Configure padding
	tk.WithPadding(&tokenizer.PaddingParams{
		PadToken: "<pad>",
	})

	return &Tokenizer{
		tk:            tk,
		maxLength:     config.MaxLength,
		padTokenID:    config.PadTokenID,
		clsTokenID:    config.ClsTokenID,
		sepTokenID:    config.SepTokenID,
		maskTokenID:   config.MaskTokenID,
		attentionMask: config.AttentionMask,
	}, nil
}

// NewTokenizerFromModelDir creates a tokenizer from the models directory
func NewTokenizerFromModelDir(modelDir string) (*Tokenizer, error) {
	tokenizerPath := filepath.Join(modelDir, "tokenizer.json")

	// Check if tokenizer file exists
	if _, err := os.Stat(tokenizerPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("tokenizer.json not found in %s", modelDir)
	}

	// Default configuration for XLM-RoBERTa
	// Based on tokenizer_config.json:
	// - <s> (BOS/CLS): 0
	// - <pad>: 1
	// - </s> (EOS/SEP): 2
	// - <unk>: 3
	// - <mask>: 250001
	config := TokenizerConfig{
		MaxLength:     512,
		PadTokenID:    1,
		ClsTokenID:    0,
		SepTokenID:    2,
		MaskTokenID:   250001,
		AttentionMask: true,
	}

	return NewTokenizer(tokenizerPath, config)
}

// Tokenize converts text to token IDs
func (t *Tokenizer) Tokenize(text string) ([]int32, error) {
	// Encode the text
	encoding, err := t.tk.EncodeSingle(text, true)
	if err != nil {
		return nil, fmt.Errorf("failed to encode text: %w", err)
	}

	// Get token IDs
	ids := encoding.GetIds()
	result := make([]int32, len(ids))
	for i, id := range ids {
		result[i] = int32(id)
	}

	return result, nil
}

// TokenizeBatch converts multiple texts to token IDs
func (t *Tokenizer) TokenizeBatch(texts []string) ([][]int32, error) {
	// Convert to EncodeInput slice
	inputs := make([]tokenizer.EncodeInput, len(texts))
	for i, text := range texts {
		inputs[i] = tokenizer.NewSingleEncodeInput(tokenizer.NewInputSequence(text))
	}

	// Encode batch
	encodings, err := t.tk.EncodeBatch(inputs, true)
	if err != nil {
		return nil, fmt.Errorf("failed to encode batch: %w", err)
	}

	// Convert to [][]int32
	result := make([][]int32, len(encodings))
	for i, enc := range encodings {
		ids := enc.GetIds()
		result[i] = make([]int32, len(ids))
		for j, id := range ids {
			result[i][j] = int32(id)
		}
	}

	return result, nil
}

// TokenizeWithAttentionMask tokenizes text and returns token IDs and attention mask
func (t *Tokenizer) TokenizeWithAttentionMask(text string) ([]int32, []int32, error) {
	encoding, err := t.tk.EncodeSingle(text, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encode text: %w", err)
	}

	// Get token IDs
	ids := encoding.GetIds()
	tokenIDs := make([]int32, len(ids))
	for i, id := range ids {
		tokenIDs[i] = int32(id)
	}

	// Get attention mask
	attentionMask := encoding.GetAttentionMask()
	mask := make([]int32, len(attentionMask))
	for i, m := range attentionMask {
		mask[i] = int32(m)
	}

	return tokenIDs, mask, nil
}

// TokenizeBatchWithAttentionMask tokenizes multiple texts with attention masks
func (t *Tokenizer) TokenizeBatchWithAttentionMask(texts []string) ([][]int32, [][]int32, error) {
	// Convert to EncodeInput slice
	inputs := make([]tokenizer.EncodeInput, len(texts))
	for i, text := range texts {
		inputs[i] = tokenizer.NewSingleEncodeInput(tokenizer.NewInputSequence(text))
	}

	encodings, err := t.tk.EncodeBatch(inputs, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encode batch: %w", err)
	}

	tokenIDs := make([][]int32, len(encodings))
	attentionMasks := make([][]int32, len(encodings))

	for i, enc := range encodings {
		// Token IDs
		ids := enc.GetIds()
		tokenIDs[i] = make([]int32, len(ids))
		for j, id := range ids {
			tokenIDs[i][j] = int32(id)
		}

		// Attention mask
		attentionMask := enc.GetAttentionMask()
		attentionMasks[i] = make([]int32, len(attentionMask))
		for j, m := range attentionMask {
			attentionMasks[i][j] = int32(m)
		}
	}

	return tokenIDs, attentionMasks, nil
}

// Decode converts token IDs back to text
func (t *Tokenizer) Decode(tokenIDs []int32, skipSpecialTokens bool) (string, error) {
	ids := make([]int, len(tokenIDs))
	for i, id := range tokenIDs {
		ids[i] = int(id)
	}

	text := t.tk.Decode(ids, skipSpecialTokens)

	return text, nil
}

// GetVocabSize returns the vocabulary size
func (t *Tokenizer) GetVocabSize() int {
	vocab := t.tk.GetVocab(false) // false = don't include added tokens
	return len(vocab)
}

// Close releases tokenizer resources
func (t *Tokenizer) Close() error {
	// The tokenizer doesn't require explicit cleanup in this implementation
	return nil
}

// SimpleTokenizer provides basic tokenization (for backwards compatibility)
// Note: This is a simple fallback. Use Tokenizer for production.
type SimpleTokenizer struct {
	vocabSize int
}

// NewSimpleTokenizer creates a simple tokenizer
func NewSimpleTokenizer(vocabSize int) *SimpleTokenizer {
	return &SimpleTokenizer{
		vocabSize: vocabSize,
	}
}

// Tokenize converts text to token IDs (simplified approach)
func (st *SimpleTokenizer) Tokenize(text string) []int32 {
	// This is a very basic implementation for testing purposes
	// In production, always use the proper Tokenizer above

	runes := []rune(text)
	tokens := make([]int32, 0, len(runes))

	for _, r := range runes {
		// Simple character-based tokenization
		tokens = append(tokens, int32(r)%int32(st.vocabSize))
	}

	// Limit to reasonable length
	if len(tokens) > 512 {
		tokens = tokens[:512]
	}

	return tokens
}
