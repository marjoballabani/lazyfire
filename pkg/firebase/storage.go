package firebase

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// StorageBucket represents a Cloud Storage bucket
type StorageBucket struct {
	Name         string
	Location     string
	StorageClass string
	TimeCreated  string
}

// StorageObject represents an object in Cloud Storage
type StorageObject struct {
	Name        string // Full path including "folders"
	DisplayName string // Short name (last segment)
	Size        int64
	ContentType string
	TimeCreated string
	Updated     string
	IsPrefix    bool // true for "folders" (prefixes)
}

// ListBuckets returns all Cloud Storage buckets for the current project.
func (c *Client) ListBuckets() ([]StorageBucket, error) {
	if c.currentProject == "" {
		return nil, fmt.Errorf("no project selected")
	}
	if c.emulatorMode {
		return nil, nil
	}

	token, err := c.getFirebaseToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b?project=%s", c.currentProject)
	req, err := http.NewRequest("GET", url, nil)
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

	var result struct {
		Items []struct {
			Name         string `json:"name"`
			Location     string `json:"location"`
			StorageClass string `json:"storageClass"`
			TimeCreated  string `json:"timeCreated"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var buckets []StorageBucket
	for _, b := range result.Items {
		buckets = append(buckets, StorageBucket{
			Name:         b.Name,
			Location:     b.Location,
			StorageClass: b.StorageClass,
			TimeCreated:  b.TimeCreated,
		})
	}
	return buckets, nil
}

// ListObjects returns objects in a bucket, optionally filtered by prefix.
// Uses delimiter "/" to get folder-like listing.
func (c *Client) ListObjects(bucket, prefix string, limit int) ([]StorageObject, error) {
	if c.emulatorMode {
		return nil, nil
	}

	token, err := c.getFirebaseToken()
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 100
	}

	url := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s/o?delimiter=/&maxResults=%d", bucket, limit)
	if prefix != "" {
		url += "&prefix=" + prefix
	}

	req, err := http.NewRequest("GET", url, nil)
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

	var result struct {
		Prefixes []string `json:"prefixes"`
		Items    []struct {
			Name        string `json:"name"`
			Size        string `json:"size"`
			ContentType string `json:"contentType"`
			TimeCreated string `json:"timeCreated"`
			Updated     string `json:"updated"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var objects []StorageObject

	// Add prefixes (folders) first
	for _, p := range result.Prefixes {
		display := p
		if prefix != "" {
			display = strings.TrimPrefix(p, prefix)
		}
		display = strings.TrimSuffix(display, "/")
		objects = append(objects, StorageObject{
			Name:        p,
			DisplayName: display,
			IsPrefix:    true,
		})
	}

	// Add objects (files)
	for _, o := range result.Items {
		display := o.Name
		if prefix != "" {
			display = strings.TrimPrefix(o.Name, prefix)
		}
		// Skip "folder marker" objects (empty name after stripping prefix, or ends with /)
		if display == "" || strings.HasSuffix(display, "/") {
			continue
		}
		size, _ := strconv.ParseInt(o.Size, 10, 64)
		objects = append(objects, StorageObject{
			Name:        o.Name,
			DisplayName: display,
			Size:        size,
			ContentType: o.ContentType,
			TimeCreated: o.TimeCreated,
			Updated:     o.Updated,
		})
	}

	return objects, nil
}
