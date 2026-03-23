package firebase

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// FirestoreRules represents the current Firestore security rules.
type FirestoreRules struct {
	Rules     string // The rules source text
	UpdatedAt string // When rules were last deployed
}

// GetFirestoreRules fetches the current Firestore security rules.
func (c *Client) GetFirestoreRules() (*FirestoreRules, error) {
	if c.currentProject == "" {
		return nil, fmt.Errorf("no project selected")
	}
	if c.emulatorMode {
		return &FirestoreRules{Rules: "// Emulator mode - rules not available"}, nil
	}

	token, err := c.getFirebaseToken()
	if err != nil {
		return nil, err
	}

	// First, list rulesets to get the latest one
	url := fmt.Sprintf("https://firebaserules.googleapis.com/v1/projects/%s/rulesets?pageSize=1", c.currentProject)
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

	var listResult struct {
		Rulesets []struct {
			Name       string `json:"name"`
			CreateTime string `json:"createTime"`
			Source     struct {
				Files []struct {
					Content string `json:"content"`
					Name    string `json:"name"`
				} `json:"files"`
			} `json:"source"`
		} `json:"rulesets"`
	}
	if err := json.Unmarshal(body, &listResult); err != nil {
		return nil, err
	}

	if len(listResult.Rulesets) == 0 {
		return &FirestoreRules{Rules: "// No rules deployed"}, nil
	}

	latest := listResult.Rulesets[0]
	rules := ""
	for _, f := range latest.Source.Files {
		if rules != "" {
			rules += "\n\n// --- " + f.Name + " ---\n"
		}
		rules += f.Content
	}

	return &FirestoreRules{
		Rules:     rules,
		UpdatedAt: latest.CreateTime,
	}, nil
}

// FirestoreIndex represents a Firestore composite index.
type FirestoreIndex struct {
	CollectionGroup string
	QueryScope      string
	Fields          []IndexField
	State           string // CREATING, READY, NEEDS_REPAIR
}

// IndexField represents a field in a composite index.
type IndexField struct {
	FieldPath string
	Order     string // ASCENDING, DESCENDING, or CONTAINS (for array)
}

// ListFirestoreIndexes returns all composite indexes for the current project.
func (c *Client) ListFirestoreIndexes() ([]FirestoreIndex, error) {
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

	url := fmt.Sprintf("https://firestore.googleapis.com/v1/projects/%s/databases/(default)/collectionGroups/-/indexes", c.currentProject)
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
		Indexes []struct {
			Name       string `json:"name"`
			QueryScope string `json:"queryScope"`
			State      string `json:"state"`
			Fields     []struct {
				FieldPath   string `json:"fieldPath"`
				Order       string `json:"order"`
				ArrayConfig string `json:"arrayConfig"`
			} `json:"fields"`
		} `json:"indexes"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var indexes []FirestoreIndex
	for _, idx := range result.Indexes {
		// Skip single-field indexes (auto-created) - composite have 2+ fields
		if len(idx.Fields) < 2 {
			continue
		}

		// Extract collection group from name
		// Format: projects/{project}/databases/(default)/collectionGroups/{group}/indexes/{id}
		parts := strings.Split(idx.Name, "/")
		collGroup := ""
		for i, p := range parts {
			if p == "collectionGroups" && i+1 < len(parts) {
				collGroup = parts[i+1]
				break
			}
		}

		fi := FirestoreIndex{
			CollectionGroup: collGroup,
			QueryScope:      idx.QueryScope,
			State:           idx.State,
		}
		for _, f := range idx.Fields {
			order := f.Order
			if f.ArrayConfig != "" {
				order = f.ArrayConfig
			}
			fi.Fields = append(fi.Fields, IndexField{
				FieldPath: f.FieldPath,
				Order:     order,
			})
		}
		indexes = append(indexes, fi)
	}

	return indexes, nil
}
