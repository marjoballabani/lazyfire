// Package firebase provides a client for interacting with Firebase services.
// It uses the Firebase CLI for authentication and project management,
// and the Google Cloud Firestore SDK for database operations.
package firebase

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/mballabani/lazyfire/pkg/config"
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

// NewClient creates a new Firebase client using existing CLI authentication.
// It verifies that the user is logged in via 'firebase login'.
func NewClient(ctx context.Context, cfg *config.Config) (*Client, error) {
	client := &Client{
		ctx:            ctx,
		config:         cfg,
		usingLocalAuth: true,
	}

	// Verify Firebase CLI is logged in
	cmd := exec.Command("firebase", "projects:list", "--json")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("not logged in. Please run 'firebase login' first")
	}

	return client, nil
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
// This affects which Firestore database is queried.
func (c *Client) SetCurrentProject(projectID string) error {
	cmd := exec.Command("firebase", "use", projectID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to switch to project %s: %v\nOutput: %s", projectID, err, string(output))
	}

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
