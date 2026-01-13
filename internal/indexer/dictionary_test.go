package indexer

import (
	"testing"
)

func TestExtractFromContent(t *testing.T) {
	extractor := NewDictionaryExtractor()

	content := `
## 購入契約 (PurchaseContract)

購入契約を保存する処理を実装します。

### ユーザー (User)

ユーザー情報を管理するクラスです。

認証 (Authentication) の処理も含まれます。
`

	mappings := extractor.ExtractFromContent(content, "test.md", "ja")

	// Check that we extracted some mappings
	if len(mappings) == 0 {
		t.Fatal("No mappings extracted")
	}

	// Check for expected mappings
	expected := map[string]string{
		"購入契約": "purchasecontract",
		"ユーザー": "user",
		"認証":     "authentication",
	}

	found := make(map[string]bool)
	for _, m := range mappings {
		if expectedTarget, ok := expected[m.SourceWord]; ok {
			if m.TargetWord == expectedTarget {
				found[m.SourceWord] = true
			}
		}
		t.Logf("Mapping: %s -> %s (confidence: %.2f)", m.SourceWord, m.TargetWord, m.Confidence)
	}

	for word := range expected {
		if !found[word] {
			t.Errorf("Expected mapping for %s not found", word)
		}
	}
}

func TestSplitCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"PurchaseContract", []string{"purchase", "contract"}},
		{"UserService", []string{"user", "service"}},
		{"getUserById", []string{"get", "user", "by", "id"}},
		{"snake_case_name", []string{"snake", "case", "name"}},
		{"Simple", []string{"simple"}},
	}

	for _, tc := range tests {
		result := splitCamelCase(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("splitCamelCase(%s): expected %v, got %v", tc.input, tc.expected, result)
			continue
		}
		for i, v := range result {
			if v != tc.expected[i] {
				t.Errorf("splitCamelCase(%s)[%d]: expected %s, got %s", tc.input, i, tc.expected[i], v)
			}
		}
	}
}

func TestIsJapanese(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"購入契約", true},
		{"ユーザー", true},
		{"PurchaseContract", false},
		{"User123", false},
		{"日本語とEnglish", true},
		{"", false},
	}

	for _, tc := range tests {
		result := IsJapanese(tc.input)
		if result != tc.expected {
			t.Errorf("IsJapanese(%s): expected %v, got %v", tc.input, tc.expected, result)
		}
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"これは日本語のテキストです", "ja"},
		{"This is English text", "en"},
		{"これはmixed日本語andEnglishです", "ja"}, // Japanese content > 30% threshold
	}

	for _, tc := range tests {
		result := DetectLanguage(tc.input)
		if result != tc.expected {
			t.Errorf("DetectLanguage(%s): expected %s, got %s", tc.input, tc.expected, result)
		}
	}
}
