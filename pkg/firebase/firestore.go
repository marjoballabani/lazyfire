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

// Document represents a Firestore document.
type Document struct {
	ID   string                 // Document ID
	Path string                 // Full path from root
	Data map[string]interface{} // Document fields as a map
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

// firestoreRequest makes an authenticated request to the Firestore REST API.
func (c *Client) firestoreRequest(method, path string) ([]byte, error) {
	token, err := c.getFirebaseToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://firestore.googleapis.com/v1/projects/%s/databases/(default)/documents%s", c.currentProject, path)

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

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

	token, err := c.getFirebaseToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://firestore.googleapis.com/v1/projects/%s/databases/(default)/documents:listCollectionIds", c.currentProject)

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

		req.Header.Set("Authorization", "Bearer "+token)
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

		documents = append(documents, Document{
			ID:   docID,
			Path: strings.Join(parts[5:], "/"), // Path after "documents/"
			Data: parseFirestoreFields(doc.Fields),
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
		ID:   docID,
		Path: docPath,
		Data: parseFirestoreFields(result.Fields),
	}, nil
}

// ListSubcollections returns all subcollections of a document.
func (c *Client) ListSubcollections(docPath string) ([]Collection, error) {
	if c.currentProject == "" {
		return nil, fmt.Errorf("no project selected")
	}

	token, err := c.getFirebaseToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://firestore.googleapis.com/v1/projects/%s/databases/(default)/documents/%s:listCollectionIds", c.currentProject, docPath)

	req, err := http.NewRequest("POST", url, strings.NewReader("{}"))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
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

	token, err := c.getFirebaseToken()
	if err != nil {
		return nil, err
	}

	// Build the structured query
	query := buildStructuredQuery(collectionPath, opts)

	reqData, err := json.Marshal(map[string]interface{}{
		"structuredQuery": query,
	})
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://firestore.googleapis.com/v1/projects/%s/databases/(default)/documents:runQuery", c.currentProject)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(reqData)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
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

		documents = append(documents, Document{
			ID:   docID,
			Path: strings.Join(parts[5:], "/"),
			Data: parseFirestoreFields(result.Document.Fields),
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
