package firebase

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Collection represents a Firestore collection.
type Collection struct {
	Name string // Collection name (last segment of path)
	Path string // Full path from root
}

// DocStats holds Firestore document size statistics calculated from raw typed fields.
type DocStats struct {
	SizeBytes     int // Document size per Firestore calculation
	FieldCount    int // Total fields including nested
	LeafFields    int // Leaf fields only (for index entry estimation)
	MaxDepth      int // Maximum nesting depth
	MaxFieldName  int // Longest field name in bytes
	MaxFieldValue int // Largest field value in bytes
	DocNameSize   int // Document name size per Firestore calculation
}

// Document represents a Firestore document.
type Document struct {
	ID    string                 // Document ID
	Path  string                 // Full path from root
	Data  map[string]interface{} // Document fields as a map
	Stats *DocStats              // Accurate stats from raw Firestore response
}

// QueryFilter represents a where clause in a Firestore query.
type QueryFilter struct {
	Field     string
	Operator  string // EQUAL, NOT_EQUAL, LESS_THAN, LESS_THAN_OR_EQUAL, GREATER_THAN, GREATER_THAN_OR_EQUAL, ARRAY_CONTAINS, IN
	Value     interface{}
	ValueType string // string, integer, double, boolean, null (empty = auto-detect)
}

// QueryOptions contains all options for a Firestore query.
type QueryOptions struct {
	Filters  []QueryFilter
	OrderBy  string
	OrderDir string // ASCENDING or DESCENDING
	Limit    int
}

// getFirebaseToken retrieves the OAuth access token from Firebase CLI config.
// It reads from ~/.config/configstore/firebase-tools.json and refreshes
// the token if expired.
func (c *Client) getFirebaseToken() (string, error) {
	home, _ := os.UserHomeDir()
	configPath := home + "/.config/configstore/firebase-tools.json"

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("Firebase not logged in. Run 'firebase login' first")
	}

	var config struct {
		Tokens struct {
			RefreshToken string `json:"refresh_token"`
			AccessToken  string `json:"access_token"`
			ExpiresAt    int64  `json:"expires_at"`
		} `json:"tokens"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to parse Firebase config: %v", err)
	}

	// Check if token is still valid (expires_at is in milliseconds)
	now := time.Now().UnixMilli()
	if config.Tokens.AccessToken != "" && config.Tokens.ExpiresAt > now {
		return config.Tokens.AccessToken, nil
	}

	// Token expired, refresh it
	if config.Tokens.RefreshToken == "" {
		return "", fmt.Errorf("no Firebase token found. Run 'firebase login' first")
	}

	return c.refreshAccessToken(config.Tokens.RefreshToken)
}

// refreshAccessToken uses the OAuth refresh token to obtain a new access token.
func (c *Client) refreshAccessToken(refreshToken string) (string, error) {
	// Firebase CLI OAuth client ID (public, not a secret)
	clientID := "563584335869-fgrhgmd47bqnekij5i8b5pr03ho849e6.apps.googleusercontent.com"

	reqBody := fmt.Sprintf("client_id=%s&refresh_token=%s&grant_type=refresh_token", clientID, refreshToken)

	resp, err := http.Post(
		"https://oauth2.googleapis.com/token",
		"application/x-www-form-urlencoded",
		strings.NewReader(reqBody),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.Error != "" {
		return "", fmt.Errorf("token refresh failed: %s", result.Error)
	}

	return result.AccessToken, nil
}

// firestoreBaseURL returns the base Firestore REST API URL for the current project.
// In emulator mode, this points to the local emulator; otherwise to the production API.
func (c *Client) firestoreBaseURL() string {
	if c.emulatorMode {
		return fmt.Sprintf("http://%s/v1/projects/%s/databases/(default)/documents", c.firestoreHost, c.currentProject)
	}
	return fmt.Sprintf("https://firestore.googleapis.com/v1/projects/%s/databases/(default)/documents", c.currentProject)
}

// setAuthHeader adds the Authorization header if not in emulator mode.
func (c *Client) setAuthHeader(req *http.Request) error {
	if c.emulatorMode {
		return nil
	}
	token, err := c.getFirebaseToken()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

// firestoreRequest makes an authenticated request to the Firestore REST API.
func (c *Client) firestoreRequest(method, path string) ([]byte, error) {
	url := c.firestoreBaseURL() + path

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	if err := c.setAuthHeader(req); err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// ListCollections returns all root-level collections in the current project.
func (c *Client) ListCollections() ([]Collection, error) {
	if c.currentProject == "" {
		return nil, fmt.Errorf("no project selected")
	}

	url := c.firestoreBaseURL() + ":listCollectionIds"

	var collections []Collection
	pageToken := ""

	for {
		reqBody := map[string]any{"pageSize": 300}
		if pageToken != "" {
			reqBody["pageToken"] = pageToken
		}
		reqData, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", url, strings.NewReader(string(reqData)))
		if err != nil {
			return nil, err
		}

		if err := c.setAuthHeader(req); err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
		}

		var result struct {
			CollectionIds []string `json:"collectionIds"`
			NextPageToken string   `json:"nextPageToken"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		for _, id := range result.CollectionIds {
			collections = append(collections, Collection{
				Name: id,
				Path: id,
			})
		}

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	return collections, nil
}

// ListDocuments returns documents in a collection, limited to the specified count.
func (c *Client) ListDocuments(collectionPath string, limit int) ([]Document, error) {
	if c.currentProject == "" {
		return nil, fmt.Errorf("no project selected")
	}

	if limit <= 0 {
		limit = 50
	}

	body, err := c.firestoreRequest("GET", fmt.Sprintf("/%s?pageSize=%d", collectionPath, limit))
	if err != nil {
		return nil, err
	}

	var result struct {
		Documents []struct {
			Name   string                 `json:"name"`
			Fields map[string]interface{} `json:"fields"`
		} `json:"documents"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var documents []Document
	for _, doc := range result.Documents {
		// Extract doc ID from full path: projects/x/databases/x/documents/collection/docId
		parts := strings.Split(doc.Name, "/")
		docID := parts[len(parts)-1]

		docPath := strings.Join(parts[5:], "/")
		documents = append(documents, Document{
			ID:    docID,
			Path:  docPath,
			Data:  parseFirestoreFields(doc.Fields),
			Stats: calculateDocStats(doc.Fields, docPath),
		})
	}

	return documents, nil
}

// GetDocument retrieves a single document by its path.
func (c *Client) GetDocument(docPath string) (*Document, error) {
	if c.currentProject == "" {
		return nil, fmt.Errorf("no project selected")
	}

	body, err := c.firestoreRequest("GET", "/"+docPath)
	if err != nil {
		return nil, err
	}

	var result struct {
		Name   string                 `json:"name"`
		Fields map[string]interface{} `json:"fields"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	parts := strings.Split(result.Name, "/")
	docID := parts[len(parts)-1]

	return &Document{
		ID:    docID,
		Path:  docPath,
		Data:  parseFirestoreFields(result.Fields),
		Stats: calculateDocStats(result.Fields, docPath),
	}, nil
}

// ListSubcollections returns all subcollections of a document.
func (c *Client) ListSubcollections(docPath string) ([]Collection, error) {
	if c.currentProject == "" {
		return nil, fmt.Errorf("no project selected")
	}

	url := c.firestoreBaseURL() + "/" + docPath + ":listCollectionIds"

	req, err := http.NewRequest("POST", url, strings.NewReader("{}"))
	if err != nil {
		return nil, err
	}

	if err := c.setAuthHeader(req); err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		// No subcollections is not an error
		return nil, nil
	}

	var result struct {
		CollectionIds []string `json:"collectionIds"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var collections []Collection
	for _, id := range result.CollectionIds {
		collections = append(collections, Collection{
			Name: id,
			Path: docPath + "/" + id,
		})
	}

	return collections, nil
}

// parseFirestoreFields converts Firestore's typed field format to a simple map.
// Firestore returns fields like {"stringValue": "hello"} which we convert to just "hello".
func parseFirestoreFields(fields map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range fields {
		if valueMap, ok := value.(map[string]interface{}); ok {
			result[key] = extractFirestoreValue(valueMap)
		}
	}

	return result
}

// extractFirestoreValue extracts the actual value from Firestore's typed format.
// Handles all Firestore types: string, integer, double, boolean, null, timestamp,
// map, array, reference, and geoPoint.
func extractFirestoreValue(field map[string]interface{}) interface{} {
	if v, ok := field["stringValue"]; ok {
		return v
	}
	if v, ok := field["integerValue"]; ok {
		return v
	}
	if v, ok := field["doubleValue"]; ok {
		return v
	}
	if v, ok := field["booleanValue"]; ok {
		return v
	}
	if v, ok := field["nullValue"]; ok {
		return v
	}
	if v, ok := field["timestampValue"]; ok {
		return v
	}
	if v, ok := field["mapValue"]; ok {
		if mapFields, ok := v.(map[string]interface{})["fields"].(map[string]interface{}); ok {
			return parseFirestoreFields(mapFields)
		}
	}
	if v, ok := field["arrayValue"]; ok {
		if values, ok := v.(map[string]interface{})["values"].([]interface{}); ok {
			var arr []interface{}
			for _, item := range values {
				if itemMap, ok := item.(map[string]interface{}); ok {
					arr = append(arr, extractFirestoreValue(itemMap))
				}
			}
			return arr
		}
	}
	if v, ok := field["referenceValue"]; ok {
		return v
	}
	if v, ok := field["geoPointValue"]; ok {
		return v
	}

	return field
}

// RunQuery executes a structured query on a collection and returns matching documents.
func (c *Client) RunQuery(collectionPath string, opts QueryOptions) ([]Document, error) {
	if c.currentProject == "" {
		return nil, fmt.Errorf("no project selected")
	}

	// Build the structured query
	query := buildStructuredQuery(collectionPath, opts)

	reqData, err := json.Marshal(map[string]interface{}{
		"structuredQuery": query,
	})
	if err != nil {
		return nil, err
	}

	url := c.firestoreBaseURL() + ":runQuery"

	req, err := http.NewRequest("POST", url, strings.NewReader(string(reqData)))
	if err != nil {
		return nil, err
	}

	if err := c.setAuthHeader(req); err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("query error %d: %s", resp.StatusCode, string(body))
	}

	// Parse query results (array of objects with "document" field)
	var results []struct {
		Document struct {
			Name   string                 `json:"name"`
			Fields map[string]interface{} `json:"fields"`
		} `json:"document"`
		ReadTime string `json:"readTime"`
	}

	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("failed to parse query results: %v", err)
	}

	var documents []Document
	for _, result := range results {
		if result.Document.Name == "" {
			continue // Skip empty results
		}
		parts := strings.Split(result.Document.Name, "/")
		docID := parts[len(parts)-1]

		docPath := strings.Join(parts[5:], "/")
		documents = append(documents, Document{
			ID:    docID,
			Path:  docPath,
			Data:  parseFirestoreFields(result.Document.Fields),
			Stats: calculateDocStats(result.Document.Fields, docPath),
		})
	}

	return documents, nil
}

// buildStructuredQuery constructs a Firestore structured query from QueryOptions.
func buildStructuredQuery(collectionPath string, opts QueryOptions) map[string]interface{} {
	// Extract collection ID from path (last segment)
	parts := strings.Split(collectionPath, "/")
	collectionID := parts[len(parts)-1]

	query := map[string]interface{}{
		"from": []map[string]interface{}{
			{"collectionId": collectionID},
		},
	}

	// Add where filters
	if len(opts.Filters) > 0 {
		if len(opts.Filters) == 1 {
			query["where"] = buildFieldFilter(opts.Filters[0])
		} else {
			// Multiple filters need composite filter
			var filters []map[string]interface{}
			for _, f := range opts.Filters {
				filters = append(filters, buildFieldFilter(f))
			}
			query["where"] = map[string]interface{}{
				"compositeFilter": map[string]interface{}{
					"op":      "AND",
					"filters": filters,
				},
			}
		}
	}

	// Add orderBy
	if opts.OrderBy != "" {
		dir := "ASCENDING"
		if opts.OrderDir == "DESC" || opts.OrderDir == "DESCENDING" {
			dir = "DESCENDING"
		}
		query["orderBy"] = []map[string]interface{}{
			{
				"field":     map[string]string{"fieldPath": opts.OrderBy},
				"direction": dir,
			},
		}
	}

	// Add limit
	if opts.Limit > 0 {
		query["limit"] = opts.Limit
	}

	return query
}

// buildFieldFilter creates a field filter for a QueryFilter.
func buildFieldFilter(f QueryFilter) map[string]interface{} {
	return map[string]interface{}{
		"fieldFilter": map[string]interface{}{
			"field": map[string]string{"fieldPath": f.Field},
			"op":    convertOperator(f.Operator),
			"value": toFirestoreValue(f.Value, f.ValueType),
		},
	}
}

// convertOperator converts user-friendly operators to Firestore API operators.
func convertOperator(op string) string {
	switch op {
	case "==", "EQUAL":
		return "EQUAL"
	case "!=", "NOT_EQUAL":
		return "NOT_EQUAL"
	case "<", "LESS_THAN":
		return "LESS_THAN"
	case "<=", "LESS_THAN_OR_EQUAL":
		return "LESS_THAN_OR_EQUAL"
	case ">", "GREATER_THAN":
		return "GREATER_THAN"
	case ">=", "GREATER_THAN_OR_EQUAL":
		return "GREATER_THAN_OR_EQUAL"
	case "in", "IN":
		return "IN"
	case "not-in", "NOT_IN":
		return "NOT_IN"
	case "array-contains", "ARRAY_CONTAINS":
		return "ARRAY_CONTAINS"
	case "array-contains-any", "ARRAY_CONTAINS_ANY":
		return "ARRAY_CONTAINS_ANY"
	default:
		return "EQUAL"
	}
}

// toFirestoreValue converts a Go value to Firestore's typed value format.
// If valueType is specified (and not "auto"), it forces that type; otherwise auto-detects.
func toFirestoreValue(v interface{}, valueType string) map[string]interface{} {
	strVal := fmt.Sprintf("%v", v)

	// If explicit type specified (not auto), convert accordingly
	if valueType != "" && valueType != "auto" {
		switch valueType {
		case "string":
			return map[string]interface{}{"stringValue": strVal}
		case "integer":
			return map[string]interface{}{"integerValue": strVal}
		case "double":
			return map[string]interface{}{"doubleValue": strVal}
		case "boolean":
			boolVal := strings.ToLower(strVal) == "true" || strVal == "1"
			return map[string]interface{}{"booleanValue": boolVal}
		case "null":
			return map[string]interface{}{"nullValue": nil}
		case "array":
			return parseArrayValue(strVal)
		}
	}

	// Auto-detect type from string value
	strVal = strings.TrimSpace(strVal)

	// Try null
	if strVal == "null" || strVal == "" {
		return map[string]interface{}{"nullValue": nil}
	}

	// Try boolean
	lower := strings.ToLower(strVal)
	if lower == "true" {
		return map[string]interface{}{"booleanValue": true}
	}
	if lower == "false" {
		return map[string]interface{}{"booleanValue": false}
	}

	// Try integer
	if i, err := strconv.ParseInt(strVal, 10, 64); err == nil {
		return map[string]interface{}{"integerValue": fmt.Sprintf("%d", i)}
	}

	// Try float
	if f, err := strconv.ParseFloat(strVal, 64); err == nil {
		return map[string]interface{}{"doubleValue": f}
	}

	// Default to string
	return map[string]interface{}{"stringValue": strVal}
}

// parseArrayValue parses a comma-separated string into a Firestore arrayValue.
// Each element is auto-typed (integers, booleans, etc. are detected).
// Example: "a,b,c" -> arrayValue with 3 stringValues
// Example: "1,2,3" -> arrayValue with 3 integerValues
func parseArrayValue(s string) map[string]interface{} {
	parts := strings.Split(s, ",")
	var values []map[string]interface{}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Auto-detect type for each element
		values = append(values, toFirestoreValue(part, "auto"))
	}

	return map[string]interface{}{
		"arrayValue": map[string]interface{}{
			"values": values,
		},
	}
}

// HasCompositeIndexes checks if a collection has any composite indexes
// by calling the Firestore Admin API. Returns false in emulator mode.
func (c *Client) HasCompositeIndexes(collectionID string) (bool, error) {
	if c.emulatorMode {
		return false, nil
	}
	if c.currentProject == "" {
		return false, fmt.Errorf("no project selected")
	}

	url := fmt.Sprintf("https://firestore.googleapis.com/v1/projects/%s/databases/(default)/collectionGroups/%s/indexes", c.currentProject, collectionID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	if err := c.setAuthHeader(req); err != nil {
		return false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Indexes []struct {
			QueryScope string `json:"queryScope"`
			Fields     []any  `json:"fields"`
		} `json:"indexes"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}

	// Filter out single-field indexes (automatic) - composite indexes have 2+ fields
	for _, idx := range result.Indexes {
		if len(idx.Fields) >= 2 {
			return true, nil
		}
	}

	return false, nil
}

// firestoreDocNameSize calculates document name size per Firestore rules:
// sum of each path component's UTF-8 size + 1 byte per component + 16 bytes
func firestoreDocNameSize(docPath string) int {
	parts := strings.Split(docPath, "/")
	size := 16
	for _, part := range parts {
		size += len(part) + 1
	}
	return size
}

// rawFieldValueSize returns the Firestore storage size of a typed field value.
func rawFieldValueSize(field map[string]interface{}) int {
	if _, ok := field["nullValue"]; ok {
		return 1
	}
	if _, ok := field["booleanValue"]; ok {
		return 1
	}
	if _, ok := field["integerValue"]; ok {
		return 8
	}
	if _, ok := field["doubleValue"]; ok {
		return 8
	}
	if _, ok := field["timestampValue"]; ok {
		return 8
	}
	if v, ok := field["stringValue"]; ok {
		if s, ok := v.(string); ok {
			return len(s) + 1
		}
		return 1
	}
	if v, ok := field["bytesValue"]; ok {
		if s, ok := v.(string); ok {
			// Base64 encoded, actual byte length is ~3/4 of string length
			return len(s) * 3 / 4
		}
		return 0
	}
	if v, ok := field["referenceValue"]; ok {
		if s, ok := v.(string); ok {
			// Reference is stored as a document name, extract path after "documents/"
			if idx := strings.Index(s, "/documents/"); idx >= 0 {
				return firestoreDocNameSize(s[idx+len("/documents/"):])
			}
			return len(s) + 1
		}
		return 0
	}
	if _, ok := field["geoPointValue"]; ok {
		return 16
	}
	if v, ok := field["mapValue"]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			if fields, ok := m["fields"].(map[string]interface{}); ok {
				return rawFieldsSize(fields)
			}
		}
		return 0
	}
	if v, ok := field["arrayValue"]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			if values, ok := m["values"].([]interface{}); ok {
				size := 0
				for _, item := range values {
					if itemMap, ok := item.(map[string]interface{}); ok {
						size += rawFieldValueSize(itemMap)
					}
				}
				return size
			}
		}
		return 0
	}
	return 0
}

// rawFieldsSize returns the total size of a Firestore fields map.
func rawFieldsSize(fields map[string]interface{}) int {
	size := 0
	for key, val := range fields {
		size += len(key) + 1
		if valMap, ok := val.(map[string]interface{}); ok {
			size += rawFieldValueSize(valMap)
		}
	}
	return size
}

// rawFieldCount counts all fields recursively from raw Firestore typed fields.
func rawFieldCount(fields map[string]interface{}) int {
	count := len(fields)
	for _, val := range fields {
		valMap, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		if v, ok := valMap["mapValue"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				if subFields, ok := m["fields"].(map[string]interface{}); ok {
					count += rawFieldCount(subFields)
				}
			}
		}
		if v, ok := valMap["arrayValue"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				if values, ok := m["values"].([]interface{}); ok {
					count += rawArrayFieldCount(values)
				}
			}
		}
	}
	return count
}

// rawArrayFieldCount counts fields in array elements.
func rawArrayFieldCount(values []interface{}) int {
	count := 0
	for _, item := range values {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if v, ok := itemMap["mapValue"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				if subFields, ok := m["fields"].(map[string]interface{}); ok {
					count += rawFieldCount(subFields)
				}
			}
		}
		if v, ok := itemMap["arrayValue"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				if vals, ok := m["values"].([]interface{}); ok {
					count += rawArrayFieldCount(vals)
				}
			}
		}
	}
	return count
}

// rawLeafFieldCount counts only leaf fields from raw Firestore typed fields.
func rawLeafFieldCount(fields map[string]interface{}) int {
	count := 0
	for _, val := range fields {
		valMap, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		if v, ok := valMap["mapValue"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				if subFields, ok := m["fields"].(map[string]interface{}); ok {
					count += rawLeafFieldCount(subFields)
					continue
				}
			}
		}
		if v, ok := valMap["arrayValue"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				if values, ok := m["values"].([]interface{}); ok {
					count += rawArrayLeafCount(values)
					continue
				}
			}
		}
		count++ // leaf field
	}
	return count
}

// rawArrayLeafCount counts leaf fields in array elements.
func rawArrayLeafCount(values []interface{}) int {
	count := 0
	for _, item := range values {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if v, ok := itemMap["mapValue"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				if subFields, ok := m["fields"].(map[string]interface{}); ok {
					count += rawLeafFieldCount(subFields)
					continue
				}
			}
		}
		if v, ok := itemMap["arrayValue"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				if vals, ok := m["values"].([]interface{}); ok {
					count += rawArrayLeafCount(vals)
					continue
				}
			}
		}
		count++ // leaf value in array
	}
	return count
}

// rawDepth calculates the maximum nesting depth from raw Firestore fields.
func rawDepth(fields map[string]interface{}) int {
	maxChild := 0
	for _, val := range fields {
		valMap, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		if v, ok := valMap["mapValue"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				if subFields, ok := m["fields"].(map[string]interface{}); ok {
					d := rawDepth(subFields)
					if d > maxChild {
						maxChild = d
					}
				}
			}
		}
		if v, ok := valMap["arrayValue"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				if values, ok := m["values"].([]interface{}); ok {
					d := rawArrayDepth(values)
					if d > maxChild {
						maxChild = d
					}
				}
			}
		}
	}
	return 1 + maxChild
}

// rawArrayDepth calculates depth within array elements.
func rawArrayDepth(values []interface{}) int {
	maxChild := 0
	for _, item := range values {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if v, ok := itemMap["mapValue"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				if subFields, ok := m["fields"].(map[string]interface{}); ok {
					d := rawDepth(subFields)
					if d > maxChild {
						maxChild = d
					}
				}
			}
		}
		if v, ok := itemMap["arrayValue"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				if vals, ok := m["values"].([]interface{}); ok {
					d := rawArrayDepth(vals)
					if d > maxChild {
						maxChild = d
					}
				}
			}
		}
	}
	return 1 + maxChild
}

// rawMaxFieldSizes finds the largest field name and value sizes from raw fields.
func rawMaxFieldSizes(fields map[string]interface{}) (maxName int, maxValue int) {
	for key, val := range fields {
		nameLen := len(key)
		if nameLen > maxName {
			maxName = nameLen
		}
		valMap, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		valSize := rawFieldValueSize(valMap)
		if valSize > maxValue {
			maxValue = valSize
		}
		// Recurse into maps
		if v, ok := valMap["mapValue"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				if subFields, ok := m["fields"].(map[string]interface{}); ok {
					nn, nv := rawMaxFieldSizes(subFields)
					if nn > maxName {
						maxName = nn
					}
					if nv > maxValue {
						maxValue = nv
					}
				}
			}
		}
		// Recurse into arrays
		if v, ok := valMap["arrayValue"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				if values, ok := m["values"].([]interface{}); ok {
					nn, nv := rawArrayMaxFieldSizes(values)
					if nn > maxName {
						maxName = nn
					}
					if nv > maxValue {
						maxValue = nv
					}
				}
			}
		}
	}
	return
}

// rawArrayMaxFieldSizes finds max field sizes in array elements.
func rawArrayMaxFieldSizes(values []interface{}) (maxName int, maxValue int) {
	for _, item := range values {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if v, ok := itemMap["mapValue"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				if subFields, ok := m["fields"].(map[string]interface{}); ok {
					nn, nv := rawMaxFieldSizes(subFields)
					if nn > maxName {
						maxName = nn
					}
					if nv > maxValue {
						maxValue = nv
					}
				}
			}
		}
	}
	return
}

// calculateDocStats computes accurate document stats from raw Firestore typed fields.
func calculateDocStats(rawFields map[string]interface{}, docPath string) *DocStats {
	docNameSize := firestoreDocNameSize(docPath)
	maxName, maxValue := rawMaxFieldSizes(rawFields)
	return &DocStats{
		SizeBytes:     docNameSize + rawFieldsSize(rawFields) + 32,
		FieldCount:    rawFieldCount(rawFields),
		LeafFields:    rawLeafFieldCount(rawFields),
		MaxDepth:      rawDepth(rawFields),
		MaxFieldName:  maxName,
		MaxFieldValue: maxValue,
		DocNameSize:   docNameSize,
	}
}
