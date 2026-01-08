package gui

import "testing"

func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		filter   string
		expected bool
	}{
		{
			name:     "empty filter matches everything",
			text:     "anything",
			filter:   "",
			expected: true,
		},
		{
			name:     "exact match",
			text:     "hello",
			filter:   "hello",
			expected: true,
		},
		{
			name:     "partial match",
			text:     "hello world",
			filter:   "world",
			expected: true,
		},
		{
			name:     "case insensitive match",
			text:     "Hello World",
			filter:   "hello",
			expected: true,
		},
		{
			name:     "case insensitive filter",
			text:     "hello world",
			filter:   "WORLD",
			expected: true,
		},
		{
			name:     "no match",
			text:     "hello world",
			filter:   "foo",
			expected: false,
		},
		{
			name:     "empty text with filter",
			text:     "",
			filter:   "test",
			expected: false,
		},
		{
			name:     "empty text empty filter",
			text:     "",
			filter:   "",
			expected: true,
		},
		{
			name:     "special characters",
			text:     "users/abc123/orders",
			filter:   "/abc",
			expected: true,
		},
		{
			name:     "unicode match",
			text:     "café résumé",
			filter:   "café",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchesFilter(tt.text, tt.filter)
			if result != tt.expected {
				t.Errorf("MatchesFilter(%q, %q) = %v, expected %v",
					tt.text, tt.filter, result, tt.expected)
			}
		})
	}
}
