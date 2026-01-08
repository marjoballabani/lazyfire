package gui

import (
	"testing"
)

func TestCountFields(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		expected int
	}{
		{
			name:     "empty map",
			data:     map[string]any{},
			expected: 0,
		},
		{
			name:     "simple map",
			data:     map[string]any{"a": 1, "b": 2, "c": 3},
			expected: 3,
		},
		{
			name: "nested map",
			data: map[string]any{
				"a": 1,
				"b": map[string]any{
					"c": 2,
					"d": 3,
				},
			},
			expected: 4, // a, b, c, d
		},
		{
			name: "map with array",
			data: map[string]any{
				"items": []any{
					map[string]any{"id": 1},
					map[string]any{"id": 2},
				},
			},
			expected: 3, // items, id, id
		},
		{
			name:     "primitive value",
			data:     "string",
			expected: 0,
		},
		{
			name:     "nil",
			data:     nil,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countFields(tt.data)
			if result != tt.expected {
				t.Errorf("countFields() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestCalculateDepth(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		expected int
	}{
		{
			name:     "empty map",
			data:     map[string]any{},
			expected: 1,
		},
		{
			name:     "flat map",
			data:     map[string]any{"a": 1, "b": "two"},
			expected: 1,
		},
		{
			name: "nested map depth 2",
			data: map[string]any{
				"level1": map[string]any{
					"level2": "value",
				},
			},
			expected: 2,
		},
		{
			name: "nested map depth 3",
			data: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": "deep",
					},
				},
			},
			expected: 3,
		},
		{
			name: "array depth",
			data: map[string]any{
				"items": []any{
					map[string]any{"nested": "value"},
				},
			},
			expected: 3, // map -> array -> map
		},
		{
			name:     "primitive",
			data:     42,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateDepth(tt.data)
			if result != tt.expected {
				t.Errorf("calculateDepth() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{10240, "10.0 KB"},
		{1048576, "1.00 MB"},
		{1572864, "1.50 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %q, expected %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFindMaxFieldSizes(t *testing.T) {
	tests := []struct {
		name         string
		data         any
		expectedName int
		expectedVal  int
	}{
		{
			name:         "empty map",
			data:         map[string]any{},
			expectedName: 0,
			expectedVal:  0,
		},
		{
			name:         "simple fields",
			data:         map[string]any{"a": 1, "longname": "x"},
			expectedName: 8, // "longname"
			expectedVal:  3, // "x" as JSON
		},
		{
			name: "nested finds deeper",
			data: map[string]any{
				"short": map[string]any{
					"verylongfieldname": "value",
				},
			},
			expectedName: 17, // "verylongfieldname"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxName, _ := findMaxFieldSizes(tt.data)
			if maxName != tt.expectedName {
				t.Errorf("findMaxFieldSizes() maxName = %d, expected %d", maxName, tt.expectedName)
			}
		})
	}
}

func TestCalculateDocStats(t *testing.T) {
	data := map[string]any{
		"name": "test",
		"nested": map[string]any{
			"value": 123,
		},
	}
	docPath := "collection/doc123"

	stats := calculateDocStats(data, docPath)

	if stats.fieldCount != 3 { // name, nested, value
		t.Errorf("fieldCount = %d, expected 3", stats.fieldCount)
	}
	if stats.maxDepth != 2 {
		t.Errorf("maxDepth = %d, expected 2", stats.maxDepth)
	}
	if stats.docPathLen != len(docPath) {
		t.Errorf("docPathLen = %d, expected %d", stats.docPathLen, len(docPath))
	}
	if stats.sizeBytes <= 0 {
		t.Error("sizeBytes should be > 0")
	}
}
