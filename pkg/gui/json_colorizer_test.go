package gui

import (
	"strings"
	"testing"
)

func TestColorizeJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string // Substrings that should be present
	}{
		{
			name:  "simple object",
			input: `{"key": "value"}`,
			contains: []string{
				"key",
				"value",
				"\033[", // Should contain ANSI codes
			},
		},
		{
			name:  "number",
			input: `{"count": 42}`,
			contains: []string{
				"count",
				"42",
			},
		},
		{
			name:  "boolean",
			input: `{"active": true}`,
			contains: []string{
				"active",
				"true",
			},
		},
		{
			name:  "null",
			input: `{"value": null}`,
			contains: []string{
				"null",
			},
		},
		{
			name:  "nested object",
			input: `{"outer": {"inner": "deep"}}`,
			contains: []string{
				"outer",
				"inner",
				"deep",
			},
		},
		{
			name:  "array",
			input: `{"items": [1, 2, 3]}`,
			contains: []string{
				"items",
				"1", "2", "3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := colorizeJSON(tt.input)

			// Should not be empty
			if result == "" {
				t.Error("colorizeJSON returned empty string")
			}

			// Should contain expected substrings
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("colorizeJSON() result should contain %q", substr)
				}
			}
		})
	}
}

func TestColorizeJSONPreservesContent(t *testing.T) {
	input := `{
  "name": "John",
  "age": 30,
  "active": true,
  "address": null
}`
	result := colorizeJSON(input)

	// Strip ANSI codes and verify content is preserved
	stripped := stripANSI(result)
	if stripped != input {
		t.Errorf("Content not preserved.\nExpected:\n%s\nGot:\n%s", input, stripped)
	}
}

func TestColorizeLine(t *testing.T) {
	line := `  "fieldName": "value",`
	result := colorizeLine(line)

	if !strings.Contains(result, "fieldName") {
		t.Error("colorizeLine should contain field name")
	}
	if !strings.Contains(result, "value") {
		t.Error("colorizeLine should contain value")
	}
}

func TestColorizeJSONEmptyInput(t *testing.T) {
	result := colorizeJSON("")
	if result != "" {
		t.Errorf("Expected empty string for empty input, got %q", result)
	}
}

func TestColorizeJSONInvalidJSON(t *testing.T) {
	// Should handle invalid JSON gracefully (return as-is or best effort)
	input := `{invalid json`
	result := colorizeJSON(input)

	// Should not panic and should return something
	if result == "" {
		t.Error("Should return non-empty result for invalid JSON")
	}
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false

	for i := 0; i < len(s); i++ {
		if s[i] == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}

	return result.String()
}
