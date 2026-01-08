// Package firebase provides a client for interacting with Firebase services.
// It uses the Firebase CLI for authentication and project management,
// and the Google Cloud Firestore SDK for database operations.
package firebase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"

	"github.com/marjoballabani/lazyfire/pkg/config"
)

// Client manages Firebase connections and operations.
// It wraps the Firebase CLI and Firestore SDK.
type Client struct {
	ctx            context.Context
	config         *config.Config
	currentProject string
	usingLocalAuth bool
}

// Project represents a Firebase project.
type Project struct {
	ID          string // Firebase project ID
	DisplayName string // Human-readable project name
	Environment string // Environment identifier (same as ID for now)
}

// ProjectDetails contains extended information about a Firebase project.
type ProjectDetails struct {
	ProjectID     string   `json:"projectId"`
	ProjectNumber string   `json:"projectNumber"`
	DisplayName   string   `json:"displayName"`
	Resources     struct {
		HostingSite       string `json:"hostingSite"`
		RealtimeDatabaseInstance string `json:"realtimeDatabaseInstance"`
		StorageBucket     string `json:"storageBucket"`
		LocationID        string `json:"locationId"`
	} `json:"resources"`
}

// NewClient creates a new Firebase client using existing CLI authentication.
// Authentication is verified lazily when ListProjects is called.
func NewClient(ctx context.Context, cfg *config.Config) (*Client, error) {
	// Just verify firebase CLI is installed (fast check)
	if _, err := exec.LookPath("firebase"); err != nil {
		return nil, fmt.Errorf("firebase CLI not found. Please install it: npm install -g firebase-tools")
	}

	return &Client{
		ctx:            ctx,
		config:         cfg,
		usingLocalAuth: true,
	}, nil
}

// ListProjects returns all Firebase projects accessible to the authenticated user.
// It calls 'firebase projects:list' and parses the JSON output.
func (c *Client) ListProjects() ([]Project, error) {
	cmd := exec.Command("firebase", "projects:list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %v", err)
	}

	// Parse Firebase CLI JSON response
	var result struct {
		Status   string `json:"status"`
		Projects []struct {
			ProjectID   string `json:"projectId"`
			DisplayName string `json:"displayName"`
		} `json:"result"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse projects: %v", err)
	}

	var projects []Project
	for _, p := range result.Projects {
		projects = append(projects, Project{
			ID:          p.ProjectID,
			DisplayName: p.DisplayName,
			Environment: p.ProjectID,
		})
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("no projects found")
	}

	return projects, nil
}

// SetCurrentProject switches the active Firebase project.
// This affects which Firestore database is queried via REST API.
func (c *Client) SetCurrentProject(projectID string) error {
	c.currentProject = projectID
	return nil
}

// GetCurrentProject returns the currently active project ID.
func (c *Client) GetCurrentProject() string {
	return c.currentProject
}

// IsUsingLocalAuth returns true if using Firebase CLI authentication.
func (c *Client) IsUsingLocalAuth() bool {
	return c.usingLocalAuth
}

// GetProjectDetails fetches extended information about a Firebase project.
func (c *Client) GetProjectDetails(projectID string) (*ProjectDetails, error) {
	token, err := c.getFirebaseToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://firebase.googleapis.com/v1beta1/projects/%s", projectID)

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

	var details ProjectDetails
	if err := json.Unmarshal(body, &details); err != nil {
		return nil, err
	}

	return &details, nil
}
