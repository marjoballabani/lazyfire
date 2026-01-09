package firebase

import (
	"reflect"
	"testing"
)

func TestConvertOperator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"equal symbol", "==", "EQUAL"},
		{"equal word", "EQUAL", "EQUAL"},
		{"not equal symbol", "!=", "NOT_EQUAL"},
		{"not equal word", "NOT_EQUAL", "NOT_EQUAL"},
		{"less than symbol", "<", "LESS_THAN"},
		{"less than word", "LESS_THAN", "LESS_THAN"},
		{"less than or equal symbol", "<=", "LESS_THAN_OR_EQUAL"},
		{"less than or equal word", "LESS_THAN_OR_EQUAL", "LESS_THAN_OR_EQUAL"},
		{"greater than symbol", ">", "GREATER_THAN"},
		{"greater than word", "GREATER_THAN", "GREATER_THAN"},
		{"greater than or equal symbol", ">=", "GREATER_THAN_OR_EQUAL"},
		{"greater than or equal word", "GREATER_THAN_OR_EQUAL", "GREATER_THAN_OR_EQUAL"},
		{"in lowercase", "in", "IN"},
		{"in uppercase", "IN", "IN"},
		{"not-in lowercase", "not-in", "NOT_IN"},
		{"not-in uppercase", "NOT_IN", "NOT_IN"},
		{"array-contains lowercase", "array-contains", "ARRAY_CONTAINS"},
		{"array-contains uppercase", "ARRAY_CONTAINS", "ARRAY_CONTAINS"},
		{"array-contains-any lowercase", "array-contains-any", "ARRAY_CONTAINS_ANY"},
		{"array-contains-any uppercase", "ARRAY_CONTAINS_ANY", "ARRAY_CONTAINS_ANY"},
		{"unknown operator defaults to EQUAL", "unknown", "EQUAL"},
		{"empty string defaults to EQUAL", "", "EQUAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertOperator(tt.input)
			if result != tt.expected {
				t.Errorf("convertOperator(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToFirestoreValue(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		valueType string
		expected  map[string]interface{}
	}{
		// Auto-detect tests
		{
			name:      "auto detect string",
			value:     "hello",
			valueType: "auto",
			expected:  map[string]interface{}{"stringValue": "hello"},
		},
		{
			name:      "auto detect integer",
			value:     "42",
			valueType: "auto",
			expected:  map[string]interface{}{"integerValue": "42"},
		},
		{
			name:      "auto detect negative integer",
			value:     "-123",
			valueType: "auto",
			expected:  map[string]interface{}{"integerValue": "-123"},
		},
		{
			name:      "auto detect float",
			value:     "3.14",
			valueType: "auto",
			expected:  map[string]interface{}{"doubleValue": 3.14},
		},
		{
			name:      "auto detect true",
			value:     "true",
			valueType: "auto",
			expected:  map[string]interface{}{"booleanValue": true},
		},
		{
			name:      "auto detect TRUE uppercase",
			value:     "TRUE",
			valueType: "auto",
			expected:  map[string]interface{}{"booleanValue": true},
		},
		{
			name:      "auto detect false",
			value:     "false",
			valueType: "auto",
			expected:  map[string]interface{}{"booleanValue": false},
		},
		{
			name:      "auto detect null",
			value:     "null",
			valueType: "auto",
			expected:  map[string]interface{}{"nullValue": nil},
		},
		{
			name:      "auto detect empty string as null",
			value:     "",
			valueType: "auto",
			expected:  map[string]interface{}{"nullValue": nil},
		},
		// Explicit type tests
		{
			name:      "explicit string type",
			value:     "42",
			valueType: "string",
			expected:  map[string]interface{}{"stringValue": "42"},
		},
		{
			name:      "explicit integer type",
			value:     "100",
			valueType: "integer",
			expected:  map[string]interface{}{"integerValue": "100"},
		},
		{
			name:      "explicit double type",
			value:     "3.14",
			valueType: "double",
			expected:  map[string]interface{}{"doubleValue": "3.14"},
		},
		{
			name:      "explicit boolean type true",
			value:     "true",
			valueType: "boolean",
			expected:  map[string]interface{}{"booleanValue": true},
		},
		{
			name:      "explicit boolean type 1",
			value:     "1",
			valueType: "boolean",
			expected:  map[string]interface{}{"booleanValue": true},
		},
		{
			name:      "explicit boolean type false",
			value:     "false",
			valueType: "boolean",
			expected:  map[string]interface{}{"booleanValue": false},
		},
		{
			name:      "explicit null type",
			value:     "anything",
			valueType: "null",
			expected:  map[string]interface{}{"nullValue": nil},
		},
		// Empty type defaults to auto
		{
			name:      "empty type defaults to auto detect",
			value:     "hello",
			valueType: "",
			expected:  map[string]interface{}{"stringValue": "hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toFirestoreValue(tt.value, tt.valueType)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("toFirestoreValue(%v, %q) = %v, expected %v",
					tt.value, tt.valueType, result, tt.expected)
			}
		})
	}
}

func TestParseArrayValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{
			name:  "string array",
			input: "a,b,c",
			expected: map[string]interface{}{
				"arrayValue": map[string]interface{}{
					"values": []map[string]interface{}{
						{"stringValue": "a"},
						{"stringValue": "b"},
						{"stringValue": "c"},
					},
				},
			},
		},
		{
			name:  "integer array",
			input: "1,2,3",
			expected: map[string]interface{}{
				"arrayValue": map[string]interface{}{
					"values": []map[string]interface{}{
						{"integerValue": "1"},
						{"integerValue": "2"},
						{"integerValue": "3"},
					},
				},
			},
		},
		{
			name:  "mixed types array",
			input: "hello,42,true",
			expected: map[string]interface{}{
				"arrayValue": map[string]interface{}{
					"values": []map[string]interface{}{
						{"stringValue": "hello"},
						{"integerValue": "42"},
						{"booleanValue": true},
					},
				},
			},
		},
		{
			name:  "array with spaces trimmed",
			input: " a , b , c ",
			expected: map[string]interface{}{
				"arrayValue": map[string]interface{}{
					"values": []map[string]interface{}{
						{"stringValue": "a"},
						{"stringValue": "b"},
						{"stringValue": "c"},
					},
				},
			},
		},
		{
			name:  "single element",
			input: "single",
			expected: map[string]interface{}{
				"arrayValue": map[string]interface{}{
					"values": []map[string]interface{}{
						{"stringValue": "single"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseArrayValue(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseArrayValue(%q) = %v, expected %v",
					tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildFieldFilter(t *testing.T) {
	tests := []struct {
		name     string
		filter   QueryFilter
		expected map[string]interface{}
	}{
		{
			name: "simple equality filter",
			filter: QueryFilter{
				Field:     "status",
				Operator:  "==",
				Value:     "active",
				ValueType: "string",
			},
			expected: map[string]interface{}{
				"fieldFilter": map[string]interface{}{
					"field": map[string]string{"fieldPath": "status"},
					"op":    "EQUAL",
					"value": map[string]interface{}{"stringValue": "active"},
				},
			},
		},
		{
			name: "greater than filter with integer",
			filter: QueryFilter{
				Field:     "age",
				Operator:  ">",
				Value:     "18",
				ValueType: "integer",
			},
			expected: map[string]interface{}{
				"fieldFilter": map[string]interface{}{
					"field": map[string]string{"fieldPath": "age"},
					"op":    "GREATER_THAN",
					"value": map[string]interface{}{"integerValue": "18"},
				},
			},
		},
		{
			name: "array-contains filter",
			filter: QueryFilter{
				Field:     "tags",
				Operator:  "array-contains",
				Value:     "featured",
				ValueType: "auto",
			},
			expected: map[string]interface{}{
				"fieldFilter": map[string]interface{}{
					"field": map[string]string{"fieldPath": "tags"},
					"op":    "ARRAY_CONTAINS",
					"value": map[string]interface{}{"stringValue": "featured"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildFieldFilter(tt.filter)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("buildFieldFilter(%+v) = %v, expected %v",
					tt.filter, result, tt.expected)
			}
		})
	}
}

func TestBuildStructuredQuery(t *testing.T) {
	tests := []struct {
		name           string
		collectionPath string
		opts           QueryOptions
		checkFn        func(t *testing.T, result map[string]interface{})
	}{
		{
			name:           "simple collection query",
			collectionPath: "users",
			opts:           QueryOptions{},
			checkFn: func(t *testing.T, result map[string]interface{}) {
				from := result["from"].([]map[string]interface{})
				if from[0]["collectionId"] != "users" {
					t.Errorf("expected collectionId 'users', got %v", from[0]["collectionId"])
				}
			},
		},
		{
			name:           "subcollection path",
			collectionPath: "users/doc123/orders",
			opts:           QueryOptions{},
			checkFn: func(t *testing.T, result map[string]interface{}) {
				from := result["from"].([]map[string]interface{})
				if from[0]["collectionId"] != "orders" {
					t.Errorf("expected collectionId 'orders', got %v", from[0]["collectionId"])
				}
			},
		},
		{
			name:           "query with single filter",
			collectionPath: "users",
			opts: QueryOptions{
				Filters: []QueryFilter{
					{Field: "status", Operator: "==", Value: "active", ValueType: "string"},
				},
			},
			checkFn: func(t *testing.T, result map[string]interface{}) {
				where := result["where"].(map[string]interface{})
				fieldFilter := where["fieldFilter"].(map[string]interface{})
				field := fieldFilter["field"].(map[string]string)
				if field["fieldPath"] != "status" {
					t.Errorf("expected fieldPath 'status', got %v", field["fieldPath"])
				}
				if fieldFilter["op"] != "EQUAL" {
					t.Errorf("expected op 'EQUAL', got %v", fieldFilter["op"])
				}
			},
		},
		{
			name:           "query with multiple filters (composite)",
			collectionPath: "users",
			opts: QueryOptions{
				Filters: []QueryFilter{
					{Field: "status", Operator: "==", Value: "active", ValueType: "string"},
					{Field: "age", Operator: ">", Value: "18", ValueType: "integer"},
				},
			},
			checkFn: func(t *testing.T, result map[string]interface{}) {
				where := result["where"].(map[string]interface{})
				composite := where["compositeFilter"].(map[string]interface{})
				if composite["op"] != "AND" {
					t.Errorf("expected composite op 'AND', got %v", composite["op"])
				}
				filters := composite["filters"].([]map[string]interface{})
				if len(filters) != 2 {
					t.Errorf("expected 2 filters, got %d", len(filters))
				}
			},
		},
		{
			name:           "query with orderBy ascending",
			collectionPath: "users",
			opts: QueryOptions{
				OrderBy:  "created",
				OrderDir: "ASC",
			},
			checkFn: func(t *testing.T, result map[string]interface{}) {
				orderBy := result["orderBy"].([]map[string]interface{})
				if orderBy[0]["direction"] != "ASCENDING" {
					t.Errorf("expected direction 'ASCENDING', got %v", orderBy[0]["direction"])
				}
				field := orderBy[0]["field"].(map[string]string)
				if field["fieldPath"] != "created" {
					t.Errorf("expected fieldPath 'created', got %v", field["fieldPath"])
				}
			},
		},
		{
			name:           "query with orderBy descending",
			collectionPath: "users",
			opts: QueryOptions{
				OrderBy:  "created",
				OrderDir: "DESC",
			},
			checkFn: func(t *testing.T, result map[string]interface{}) {
				orderBy := result["orderBy"].([]map[string]interface{})
				if orderBy[0]["direction"] != "DESCENDING" {
					t.Errorf("expected direction 'DESCENDING', got %v", orderBy[0]["direction"])
				}
			},
		},
		{
			name:           "query with limit",
			collectionPath: "users",
			opts: QueryOptions{
				Limit: 100,
			},
			checkFn: func(t *testing.T, result map[string]interface{}) {
				if result["limit"] != 100 {
					t.Errorf("expected limit 100, got %v", result["limit"])
				}
			},
		},
		{
			name:           "full query with all options",
			collectionPath: "users",
			opts: QueryOptions{
				Filters: []QueryFilter{
					{Field: "status", Operator: "==", Value: "active", ValueType: "string"},
				},
				OrderBy:  "created",
				OrderDir: "DESC",
				Limit:    50,
			},
			checkFn: func(t *testing.T, result map[string]interface{}) {
				// Check all parts exist
				if result["where"] == nil {
					t.Error("expected where clause")
				}
				if result["orderBy"] == nil {
					t.Error("expected orderBy clause")
				}
				if result["limit"] != 50 {
					t.Errorf("expected limit 50, got %v", result["limit"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildStructuredQuery(tt.collectionPath, tt.opts)
			tt.checkFn(t, result)
		})
	}
}

func TestParseFirestoreFields(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "string field",
			input: map[string]interface{}{
				"name": map[string]interface{}{"stringValue": "John"},
			},
			expected: map[string]interface{}{"name": "John"},
		},
		{
			name: "integer field",
			input: map[string]interface{}{
				"age": map[string]interface{}{"integerValue": "25"},
			},
			expected: map[string]interface{}{"age": "25"},
		},
		{
			name: "boolean field",
			input: map[string]interface{}{
				"active": map[string]interface{}{"booleanValue": true},
			},
			expected: map[string]interface{}{"active": true},
		},
		{
			name: "null field",
			input: map[string]interface{}{
				"deleted": map[string]interface{}{"nullValue": nil},
			},
			expected: map[string]interface{}{"deleted": nil},
		},
		{
			name: "multiple fields",
			input: map[string]interface{}{
				"name":   map[string]interface{}{"stringValue": "John"},
				"age":    map[string]interface{}{"integerValue": "25"},
				"active": map[string]interface{}{"booleanValue": true},
			},
			expected: map[string]interface{}{
				"name":   "John",
				"age":    "25",
				"active": true,
			},
		},
		{
			name:     "empty fields",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFirestoreFields(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseFirestoreFields(%v) = %v, expected %v",
					tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractFirestoreValue(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected interface{}
	}{
		{
			name:     "string value",
			input:    map[string]interface{}{"stringValue": "hello"},
			expected: "hello",
		},
		{
			name:     "integer value",
			input:    map[string]interface{}{"integerValue": "42"},
			expected: "42",
		},
		{
			name:     "double value",
			input:    map[string]interface{}{"doubleValue": 3.14},
			expected: 3.14,
		},
		{
			name:     "boolean true",
			input:    map[string]interface{}{"booleanValue": true},
			expected: true,
		},
		{
			name:     "boolean false",
			input:    map[string]interface{}{"booleanValue": false},
			expected: false,
		},
		{
			name:     "null value",
			input:    map[string]interface{}{"nullValue": nil},
			expected: nil,
		},
		{
			name:     "timestamp value",
			input:    map[string]interface{}{"timestampValue": "2024-01-01T00:00:00Z"},
			expected: "2024-01-01T00:00:00Z",
		},
		{
			name:     "reference value",
			input:    map[string]interface{}{"referenceValue": "projects/test/databases/(default)/documents/users/123"},
			expected: "projects/test/databases/(default)/documents/users/123",
		},
		{
			name: "geoPoint value",
			input: map[string]interface{}{"geoPointValue": map[string]interface{}{
				"latitude":  40.7128,
				"longitude": -74.0060,
			}},
			expected: map[string]interface{}{"latitude": 40.7128, "longitude": -74.0060},
		},
		{
			name:     "unknown type returns raw",
			input:    map[string]interface{}{"unknownType": "value"},
			expected: map[string]interface{}{"unknownType": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFirestoreValue(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("extractFirestoreValue(%v) = %v, expected %v",
					tt.input, result, tt.expected)
			}
		})
	}
}
