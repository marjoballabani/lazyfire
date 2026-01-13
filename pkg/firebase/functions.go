package firebase

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CloudFunction represents a deployed Cloud Function.
type CloudFunction struct {
	Name        string // Full resource name
	DisplayName string // Short name for display
	Status      string // ACTIVE, DEPLOYING, OFFLINE, DELETE_IN_PROGRESS, UNKNOWN
	Runtime     string // nodejs18, python311, go121, etc.
	Region      string // us-central1, europe-west1, etc.
	Memory      string // 256MB, 512MB, etc.
	Timeout     string // 60s, 540s, etc.
	TriggerType string // HTTP, Firestore, PubSub, Schedule, etc.
	TriggerURL  string // For HTTP triggers
	UpdatedAt   string // Last deployment time
	EntryPoint  string // Function entry point
}

// LogEntry represents a function log entry.
type LogEntry struct {
	Timestamp string
	Severity  string // INFO, WARNING, ERROR, DEBUG, DEFAULT
	Message   string
	Function  string
}

// ListFunctions fetches all Cloud Functions for the current project.
func (c *Client) ListFunctions() ([]CloudFunction, error) {
	if c.currentProject == "" {
		return nil, fmt.Errorf("no project selected")
	}

	token, err := c.getFirebaseToken()
	if err != nil {
		return nil, err
	}

	// Use "-" for location to list functions across all regions
	url := fmt.Sprintf("https://cloudfunctions.googleapis.com/v1/projects/%s/locations/-/functions", c.currentProject)

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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list functions: %s", string(body))
	}

	var result struct {
		Functions []struct {
			Name              string `json:"name"`
			Status            string `json:"status"`
			Runtime           string `json:"runtime"`
			AvailableMemoryMb int    `json:"availableMemoryMb"`
			Timeout           string `json:"timeout"`
			UpdateTime        string `json:"updateTime"`
			EntryPoint        string `json:"entryPoint"`
			HttpsTrigger      *struct {
				URL string `json:"url"`
			} `json:"httpsTrigger"`
			EventTrigger *struct {
				EventType string `json:"eventType"`
				Resource  string `json:"resource"`
			} `json:"eventTrigger"`
		} `json:"functions"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse functions: %v", err)
	}

	var functions []CloudFunction
	for _, f := range result.Functions {
		cf := CloudFunction{
			Name:       f.Name,
			Status:     f.Status,
			Runtime:    f.Runtime,
			Timeout:    f.Timeout,
			EntryPoint: f.EntryPoint,
			UpdatedAt:  f.UpdateTime,
		}

		// Extract display name and region from full name
		// Format: projects/{project}/locations/{region}/functions/{name}
		parts := strings.Split(f.Name, "/")
		if len(parts) >= 6 {
			cf.DisplayName = parts[5]
			cf.Region = parts[3]
		}

		// Format memory
		if f.AvailableMemoryMb > 0 {
			if f.AvailableMemoryMb >= 1024 {
				cf.Memory = fmt.Sprintf("%.1fGB", float64(f.AvailableMemoryMb)/1024)
			} else {
				cf.Memory = fmt.Sprintf("%dMB", f.AvailableMemoryMb)
			}
		}

		// Determine trigger type
		if f.HttpsTrigger != nil {
			cf.TriggerType = "HTTP"
			cf.TriggerURL = f.HttpsTrigger.URL
		} else if f.EventTrigger != nil {
			cf.TriggerType = parseTriggerType(f.EventTrigger.EventType)
		}

		functions = append(functions, cf)
	}

	return functions, nil
}

// parseTriggerType extracts a friendly trigger type from the event type.
func parseTriggerType(eventType string) string {
	switch {
	case strings.Contains(eventType, "firestore"):
		return "Firestore"
	case strings.Contains(eventType, "pubsub"):
		return "PubSub"
	case strings.Contains(eventType, "storage"):
		return "Storage"
	case strings.Contains(eventType, "auth"):
		return "Auth"
	case strings.Contains(eventType, "analytics"):
		return "Analytics"
	case strings.Contains(eventType, "scheduler"):
		return "Schedule"
	default:
		return "Event"
	}
}

// GetFunctionLogs fetches recent logs for a function using Cloud Logging API.
func (c *Client) GetFunctionLogs(functionName string, limit int) ([]LogEntry, error) {
	if c.currentProject == "" {
		return nil, fmt.Errorf("no project selected")
	}

	token, err := c.getFirebaseToken()
	if err != nil {
		return nil, err
	}

	// Build the filter for Cloud Functions logs
	filter := fmt.Sprintf(`resource.type="cloud_function" AND resource.labels.function_name="%s"`, functionName)

	requestBody := map[string]interface{}{
		"resourceNames": []string{fmt.Sprintf("projects/%s", c.currentProject)},
		"filter":        filter,
		"orderBy":       "timestamp desc",
		"pageSize":      limit,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://logging.googleapis.com/v2/entries:list", strings.NewReader(string(jsonBody)))
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get logs: %s", string(body))
	}

	var result struct {
		Entries []struct {
			Timestamp   string `json:"timestamp"`
			Severity    string `json:"severity"`
			TextPayload string `json:"textPayload"`
			JsonPayload *struct {
				Message string `json:"message"`
			} `json:"jsonPayload"`
			Resource *struct {
				Labels map[string]string `json:"labels"`
			} `json:"resource"`
		} `json:"entries"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse logs: %v", err)
	}

	var logs []LogEntry
	for _, e := range result.Entries {
		entry := LogEntry{
			Timestamp: formatLogTimestamp(e.Timestamp),
			Severity:  e.Severity,
			Function:  functionName,
		}

		// Get message from either textPayload or jsonPayload
		if e.TextPayload != "" {
			entry.Message = e.TextPayload
		} else if e.JsonPayload != nil && e.JsonPayload.Message != "" {
			entry.Message = e.JsonPayload.Message
		}

		logs = append(logs, entry)
	}

	return logs, nil
}

// formatLogTimestamp formats a timestamp for display.
func formatLogTimestamp(ts string) string {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return ts
	}
	return t.Local().Format("15:04:05")
}
