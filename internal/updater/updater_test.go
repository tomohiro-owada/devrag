package updater

import "testing"

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"v1.2.0", "1.2.0", false},
		{"1.2.0", "1.2.0", false},
		{"v0.1.0", "0.1.0", false},
		{"v10.20.30", "10.20.30", false},
		{"invalid", "", true},
		{"v1.2", "", true},
		{"1.2.3-beta", "1.2.3", false}, // prefix match is ok
		{"", "", true},
		{"vX.Y.Z", "", true},
	}

	for _, tt := range tests {
		result, err := normalizeVersion(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("normalizeVersion(%q) expected error, got %q", tt.input, result)
			}
		} else {
			if err != nil {
				t.Errorf("normalizeVersion(%q) unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("normalizeVersion(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		}
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		latest   string
		current  string
		expected bool
		wantErr  bool
	}{
		{"1.3.0", "1.2.0", true, false},
		{"1.2.1", "1.2.0", true, false},
		{"2.0.0", "1.9.9", true, false},
		{"1.2.0", "1.2.0", false, false},
		{"1.1.0", "1.2.0", false, false},
		{"1.2.0", "1.2.1", false, false},
		{"v1.3.0", "v1.2.0", true, false},
		{"1.3.0", "v1.2.0", true, false},
		{"invalid", "1.2.0", false, true},
		{"1.2.0", "invalid", false, true},
	}

	for _, tt := range tests {
		result, err := isNewerVersion(tt.latest, tt.current)
		if tt.wantErr {
			if err == nil {
				t.Errorf("isNewerVersion(%q, %q) expected error", tt.latest, tt.current)
			}
		} else {
			if err != nil {
				t.Errorf("isNewerVersion(%q, %q) unexpected error: %v", tt.latest, tt.current, err)
			}
			if result != tt.expected {
				t.Errorf("isNewerVersion(%q, %q) = %v, want %v", tt.latest, tt.current, result, tt.expected)
			}
		}
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
		wantErr  bool
	}{
		{"1.2.0", []int{1, 2, 0}, false},
		{"1.2.3", []int{1, 2, 3}, false},
		{"10.20.30", []int{10, 20, 30}, false},
		{"1.2.abc", nil, true},
		{"1.2.-1", nil, true},
		{"", []int{}, true},
	}

	for _, tt := range tests {
		result, err := parseVersion(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseVersion(%q) expected error", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseVersion(%q) unexpected error: %v", tt.input, err)
			}
			if len(result) != len(tt.expected) {
				t.Errorf("parseVersion(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
				continue
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("parseVersion(%q)[%d] = %d, want %d", tt.input, i, result[i], tt.expected[i])
				}
			}
		}
	}
}

func TestSemverRegex(t *testing.T) {
	tests := []struct {
		input   string
		matches bool
	}{
		{"v1.2.3", true},
		{"1.2.3", true},
		{"v0.0.0", true},
		{"v10.20.30", true},
		{"1.2.3-beta", true},
		{"v1.2.3-rc1", true},
		{"v1.2", false},
		{"v1", false},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		matches := semverRegex.MatchString(tt.input)
		if matches != tt.matches {
			t.Errorf("semverRegex.MatchString(%q) = %v, want %v", tt.input, matches, tt.matches)
		}
	}
}
