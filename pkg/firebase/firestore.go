package firebase

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
