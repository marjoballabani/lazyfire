package firebase

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// AuthUser represents a Firebase Auth user
type AuthUser struct {
	UID           string
	Email         string
	DisplayName   string
	PhotoURL      string
	Disabled      bool
	EmailVerified bool
	CreatedAt     string
	LastSignIn    string
	Providers     []string // Provider IDs (google.com, password, etc.)
}

// ListAuthUsers returns Firebase Auth users for the current project.
func (c *Client) ListAuthUsers(limit int) ([]AuthUser, error) {
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

	if limit <= 0 {
		limit = 50
	}

	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/projects/%s/accounts:batchGet?maxResults=%d", c.currentProject, limit)

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
		Users []struct {
			LocalID       string `json:"localId"`
			Email         string `json:"email"`
			DisplayName   string `json:"displayName"`
			PhotoURL      string `json:"photoUrl"`
			Disabled      bool   `json:"disabled"`
			EmailVerified bool   `json:"emailVerified"`
			CreatedAt     string `json:"createdAt"`   // Milliseconds since epoch
			LastLoginAt   string `json:"lastLoginAt"` // Milliseconds since epoch
			ProviderUserInfo []struct {
				ProviderID string `json:"providerId"`
			} `json:"providerUserInfo"`
		} `json:"users"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var users []AuthUser
	for _, u := range result.Users {
		user := AuthUser{
			UID:           u.LocalID,
			Email:         u.Email,
			DisplayName:   u.DisplayName,
			PhotoURL:      u.PhotoURL,
			Disabled:      u.Disabled,
			EmailVerified: u.EmailVerified,
			CreatedAt:     formatEpochMillis(u.CreatedAt),
			LastSignIn:    formatEpochMillis(u.LastLoginAt),
		}
		for _, p := range u.ProviderUserInfo {
			user.Providers = append(user.Providers, p.ProviderID)
		}
		users = append(users, user)
	}

	return users, nil
}

// formatEpochMillis converts epoch milliseconds string to human-readable format.
func formatEpochMillis(ms string) string {
	if ms == "" {
		return ""
	}
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return ms
	}
	t := time.Unix(msInt/1000, (msInt%1000)*1000000)
	return t.Local().Format("Jan 2, 2006 3:04 PM")
}
