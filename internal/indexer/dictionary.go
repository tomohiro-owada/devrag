package indexer

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/tomohiro-owada/devrag/internal/vectordb"
)

// DictionaryExtractor extracts word mappings from documents
type DictionaryExtractor struct {
	// Patterns for extracting word pairs
	parenthesisPattern *regexp.Regexp // 日本語 (English) or 日本語（English）
	bracketPattern     *regexp.Regexp // 日本語 [English]
}

// NewDictionaryExtractor creates a new dictionary extractor
func NewDictionaryExtractor() *DictionaryExtractor {
	return &DictionaryExtractor{
		// Match: 日本語 (English) or 日本語（English）
		parenthesisPattern: regexp.MustCompile(`([\p{Han}\p{Hiragana}\p{Katakana}ー]+)\s*[（(]([a-zA-Z][a-zA-Z0-9_]*)[)）]`),
		// Match: 日本語 [English]
		bracketPattern: regexp.MustCompile(`([\p{Han}\p{Hiragana}\p{Katakana}ー]+)\s*\[([a-zA-Z][a-zA-Z0-9_]*)\]`),
	}
}

// ExtractFromContent extracts word mappings from document content
func (de *DictionaryExtractor) ExtractFromContent(content string, sourceDoc string, sourceLang string) []vectordb.WordMapping {
	var mappings []vectordb.WordMapping
	seen := make(map[string]bool)

	// Extract from parenthesis patterns: 日本語 (English)
	matches := de.parenthesisPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			jaWord := strings.TrimSpace(match[1])
			enWord := strings.TrimSpace(match[2])
			key := jaWord + ":" + enWord
			if !seen[key] && jaWord != "" && enWord != "" {
				seen[key] = true
				mappings = append(mappings, vectordb.WordMapping{
					SourceWord:     jaWord,
					TargetWord:     strings.ToLower(enWord),
					SourceLang:     sourceLang,
					Confidence:     1.0,
					SourceDocument: sourceDoc,
				})
				// Also add split words from camelCase
				splitWords := splitCamelCase(enWord)
				for _, sw := range splitWords {
					swKey := jaWord + ":" + sw
					if !seen[swKey] && sw != enWord {
						seen[swKey] = true
						mappings = append(mappings, vectordb.WordMapping{
							SourceWord:     jaWord,
							TargetWord:     strings.ToLower(sw),
							SourceLang:     sourceLang,
							Confidence:     0.8,
							SourceDocument: sourceDoc,
						})
					}
				}
			}
		}
	}

	// Extract from bracket patterns: 日本語 [English]
	matches = de.bracketPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			jaWord := strings.TrimSpace(match[1])
			enWord := strings.TrimSpace(match[2])
			key := jaWord + ":" + enWord
			if !seen[key] && jaWord != "" && enWord != "" {
				seen[key] = true
				mappings = append(mappings, vectordb.WordMapping{
					SourceWord:     jaWord,
					TargetWord:     strings.ToLower(enWord),
					SourceLang:     sourceLang,
					Confidence:     0.9,
					SourceDocument: sourceDoc,
				})
			}
		}
	}

	return mappings
}

// ExtractFromSymbolsAndComments extracts mappings by analyzing code comments near symbols
func (de *DictionaryExtractor) ExtractFromSymbolsAndComments(content string, symbols []string, sourceDoc string, sourceLang string) []vectordb.WordMapping {
	var mappings []vectordb.WordMapping
	seen := make(map[string]bool)

	// Pattern: // 日本語コメント for symbol or /* 日本語 */ near symbol
	commentPattern := regexp.MustCompile(`(?://|#)\s*([\p{Han}\p{Hiragana}\p{Katakana}ー]+)`)

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		// Check if line contains a symbol definition
		for _, symbol := range symbols {
			if strings.Contains(line, symbol) {
				// Look at previous line for comment
				if i > 0 {
					prevLine := lines[i-1]
					matches := commentPattern.FindAllStringSubmatch(prevLine, -1)
					for _, match := range matches {
						if len(match) >= 2 {
							jaWord := strings.TrimSpace(match[1])
							// Map the Japanese comment to the symbol name parts
							symbolParts := splitCamelCase(symbol)
							for _, part := range symbolParts {
								key := jaWord + ":" + part
								if !seen[key] && len(jaWord) > 1 && len(part) > 1 {
									seen[key] = true
									mappings = append(mappings, vectordb.WordMapping{
										SourceWord:     jaWord,
										TargetWord:     strings.ToLower(part),
										SourceLang:     sourceLang,
										Confidence:     0.6,
										SourceDocument: sourceDoc,
									})
								}
							}
						}
					}
				}
			}
		}
	}

	return mappings
}

// splitCamelCase splits a camelCase or PascalCase string into words
func splitCamelCase(s string) []string {
	// Handle snake_case first
	if strings.Contains(s, "_") {
		parts := strings.Split(s, "_")
		var result []string
		for _, p := range parts {
			if p != "" {
				result = append(result, strings.ToLower(p))
			}
		}
		return result
	}

	// Handle camelCase/PascalCase
	var words []string
	var currentWord strings.Builder

	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			if currentWord.Len() > 0 {
				words = append(words, strings.ToLower(currentWord.String()))
				currentWord.Reset()
			}
		}
		currentWord.WriteRune(r)
	}

	if currentWord.Len() > 0 {
		words = append(words, strings.ToLower(currentWord.String()))
	}

	return words
}

// IsJapanese checks if a string contains Japanese characters
func IsJapanese(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana) {
			return true
		}
	}
	return false
}

// DetectLanguage detects the primary language of text
func DetectLanguage(s string) string {
	jaCount := 0
	enCount := 0
	totalCount := 0

	for _, r := range s {
		if unicode.IsLetter(r) {
			totalCount++
			if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana) {
				jaCount++
			} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				enCount++
			}
		}
	}

	if totalCount == 0 {
		return "unknown"
	}

	jaRatio := float64(jaCount) / float64(totalCount)
	enRatio := float64(enCount) / float64(totalCount)

	if jaRatio > 0.3 {
		return "ja"
	} else if enRatio > 0.8 {
		return "en"
	}
	return "mixed"
}
