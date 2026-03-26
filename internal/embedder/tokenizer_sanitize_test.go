package embedder

import (
	"strings"
	"testing"
)

func TestSanitizeForTokenizer_BoxDrawingChars(t *testing.T) {
	// Box Drawing characters (U+2500–U+257F) that trigger tokenizer panics
	input := "┌──────────┐\n│  Header  │\n└──────────┘"
	result := SanitizeForTokenizer(input)

	// Should not contain any box-drawing characters
	for _, r := range result {
		if r >= 0x2500 && r <= 0x257F {
			t.Errorf("Box drawing character U+%04X still present after sanitization", r)
		}
	}

	// Should preserve non-problematic content
	if !strings.Contains(result, "Header") {
		t.Error("Sanitized text should preserve 'Header'")
	}
}

func TestSanitizeForTokenizer_BlockElements(t *testing.T) {
	input := "Progress: ▓▓▓▓░░░░ 50%"
	result := SanitizeForTokenizer(input)

	if !strings.Contains(result, "Progress:") {
		t.Error("Should preserve 'Progress:'")
	}
	if !strings.Contains(result, "50%") {
		t.Error("Should preserve '50%'")
	}

	for _, r := range result {
		if r >= 0x2580 && r <= 0x259F {
			t.Errorf("Block element U+%04X still present", r)
		}
	}
}

func TestSanitizeForTokenizer_GeometricShapes(t *testing.T) {
	input := "■ Item 1\n□ Item 2\n▲ Warning"
	result := SanitizeForTokenizer(input)

	if !strings.Contains(result, "Item 1") {
		t.Error("Should preserve 'Item 1'")
	}
	if !strings.Contains(result, "Warning") {
		t.Error("Should preserve 'Warning'")
	}
}

func TestSanitizeForTokenizer_BraillePatterns(t *testing.T) {
	input := "⠋⠙⠹⠸ loading..."
	result := SanitizeForTokenizer(input)

	if !strings.Contains(result, "loading...") {
		t.Error("Should preserve 'loading...'")
	}

	for _, r := range result {
		if r >= 0x2800 && r <= 0x28FF {
			t.Errorf("Braille pattern U+%04X still present", r)
		}
	}
}

func TestSanitizeForTokenizer_ReplacementChar(t *testing.T) {
	input := "Hello \uFFFD World"
	result := SanitizeForTokenizer(input)

	if strings.ContainsRune(result, '\uFFFD') {
		t.Error("Replacement character should be removed")
	}
	if !strings.Contains(result, "Hello") || !strings.Contains(result, "World") {
		t.Error("Should preserve surrounding text")
	}
}

func TestSanitizeForTokenizer_PreservesNormalText(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"ASCII", "Hello world, this is a test!"},
		{"Japanese", "これはテストです。日本語のテキスト。"},
		{"Emoji", "Hello 👋 World 🌍"},
		{"CJK + ASCII", "Go言語でプログラミング programming"},
		{"Mixed with newlines", "# Title\n\nParagraph with **bold** text.\n"},
		{"Empty string", ""},
		{"Numbers and punctuation", "v1.2.3 (2024-01-15) — release notes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForTokenizer(tt.input)
			if result != tt.input {
				t.Errorf("Normal text was modified:\n  input:  %q\n  result: %q", tt.input, result)
			}
		})
	}
}

func TestSanitizeForTokenizer_MixedContent(t *testing.T) {
	// Simulates the kind of content from issue #18:
	// markdown tables with Unicode border characters alongside Japanese text
	input := "## アーキテクチャ\n\n┌─────────┬─────────┐\n│ サービス │ ポート  │\n├─────────┼─────────┤\n│ API     │ 8080    │\n└─────────┴─────────┘\n\n✅ 完了"

	result := SanitizeForTokenizer(input)

	// Should preserve Japanese text
	if !strings.Contains(result, "アーキテクチャ") {
		t.Error("Should preserve Japanese text 'アーキテクチャ'")
	}
	if !strings.Contains(result, "サービス") {
		t.Error("Should preserve Japanese text 'サービス'")
	}
	if !strings.Contains(result, "API") {
		t.Error("Should preserve 'API'")
	}
	if !strings.Contains(result, "8080") {
		t.Error("Should preserve '8080'")
	}
	// ✅ is an emoji, not in the problematic ranges
	if !strings.Contains(result, "✅") {
		t.Error("Should preserve emoji ✅")
	}

	// Should not contain box-drawing characters
	for _, r := range result {
		if r >= 0x2500 && r <= 0x257F {
			t.Errorf("Box drawing character U+%04X still present", r)
		}
	}
}

func TestIsProblematicRune(t *testing.T) {
	tests := []struct {
		name     string
		r        rune
		expected bool
	}{
		// Problematic runes
		{"Box drawing horizontal", '─', true},     // U+2500
		{"Box drawing vertical", '│', true},        // U+2502
		{"Box drawing corner", '┌', true},          // U+250C
		{"Box drawing cross", '┼', true},           // U+253C
		{"Block upper half", '▀', true},            // U+2580
		{"Block full", '█', true},                  // U+2588
		{"Block light shade", '░', true},           // U+2591
		{"Geometric black square", '■', true},      // U+25A0
		{"Geometric white square", '□', true},      // U+25A1
		{"Geometric triangle", '▲', true},          // U+25B2
		{"Braille pattern", '⠋', true},             // U+280B
		{"Replacement char", '\uFFFD', true},

		// Safe runes
		{"ASCII letter", 'A', false},
		{"ASCII digit", '0', false},
		{"Space", ' ', false},
		{"Japanese hiragana", 'あ', false},
		{"Japanese katakana", 'ア', false},
		{"CJK ideograph", '漢', false},
		{"Emoji check mark", '✅', false},
		{"Emoji wave", '👋', false},
		{"Em dash", '—', false},                    // U+2014, not in problematic range
		{"Ellipsis", '…', false},                   // U+2026
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isProblematicRune(tt.r)
			if got != tt.expected {
				t.Errorf("isProblematicRune(U+%04X) = %v, want %v", tt.r, got, tt.expected)
			}
		})
	}
}

func TestSanitizeForTokenizer_LargeInput(t *testing.T) {
	// Build a large string with mixed problematic and safe characters
	var b strings.Builder
	for i := 0; i < 1000; i++ {
		b.WriteString("日本語テキスト with ┌─┐ box drawing │ and text\n")
	}
	input := b.String()

	result := SanitizeForTokenizer(input)

	// Should not panic and should complete
	if len(result) == 0 {
		t.Error("Result should not be empty for non-empty input")
	}

	// Content should be preserved (just box-drawing replaced with spaces)
	if !strings.Contains(result, "日本語テキスト") {
		t.Error("Japanese text should be preserved")
	}
}

func TestSanitizeForTokenizer_ConsecutiveSpaces(t *testing.T) {
	// Issue #18 follow-up: consecutive spaces (4+) trigger the real
	// sentencepiece normalizer bug in sugarme/tokenizer v0.3.0
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"4 spaces collapsed to 3",
			"a    b",
			"a   b",
		},
		{
			"8 spaces collapsed to 3",
			"a        b",
			"a   b",
		},
		{
			"Deep indentation (48 spaces)",
			"                                                code",
			"   code",
		},
		{
			"3 spaces preserved",
			"a   b",
			"a   b",
		},
		{
			"2 spaces preserved",
			"a  b",
			"a  b",
		},
		{
			"1 space preserved",
			"a b",
			"a b",
		},
		{
			"Multiple groups of spaces",
			"a    b      c  d",
			"a   b   c  d",
		},
		{
			"Indented code block",
			"func main() {\n        fmt.Println()\n}",
			"func main() {\n   fmt.Println()\n}",
		},
		{
			"Tabs are not affected",
			"a\t\t\t\tb",
			"a\t\t\t\tb",
		},
		{
			"Mixed tabs and spaces",
			"a\t    b",
			"a\t   b",
		},
		{
			"Only spaces",
			"          ",
			"   ",
		},
		{
			"Spaces at start and end",
			"      hello      ",
			"   hello   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForTokenizer(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeForTokenizer(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeForTokenizer_FlowchartIndentation(t *testing.T) {
	// Simulates the exact pattern from the user's reproduction case:
	// deeply nested flowchart-style text with 4-space indentation
	input := "Start\n" +
		"    Step 1\n" +
		"        Step 1.1\n" +
		"            Step 1.1.1\n" +
		"                Step 1.1.1.1\n" +
		"                    Step 1.1.1.1.1\n" +
		"                        Step 1.1.1.1.1.1\n" +
		"                            Step 1.1.1.1.1.1.1\n" +
		"                                Step 1.1.1.1.1.1.1.1\n" +
		"                                    Step 1.1.1.1.1.1.1.1.1\n" +
		"                                        Step 1.1.1.1.1.1.1.1.1.1\n"

	result := SanitizeForTokenizer(input)

	// Should not contain 4+ consecutive spaces
	if strings.Contains(result, "    ") {
		t.Error("Result should not contain 4+ consecutive spaces")
	}

	// Should preserve all step labels
	for _, step := range []string{"Start", "Step 1.1.1.1.1.1.1.1.1.1"} {
		if !strings.Contains(result, step) {
			t.Errorf("Should preserve %q", step)
		}
	}
}
