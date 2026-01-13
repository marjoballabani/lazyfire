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

// ListFunctions fetches all Cloud Functions (v1 and v2) for the current project.
func (c *Client) ListFunctions() ([]CloudFunction, error) {
	if c.currentProject == "" {
		return nil, fmt.Errorf("no project selected")
	}

	token, err := c.getFirebaseToken()
	if err != nil {
		return nil, err
	}

	var allFunctions []CloudFunction

	// Fetch v1 functions
	v1Functions, _ := c.listFunctionsV1(token)
	allFunctions = append(allFunctions, v1Functions...)

	// Fetch v2 functions
	v2Functions, _ := c.listFunctionsV2(token)
	allFunctions = append(allFunctions, v2Functions...)

	if len(allFunctions) == 0 && len(v1Functions) == 0 && len(v2Functions) == 0 {
		return allFunctions, nil
	}

	return allFunctions, nil
}

// listFunctionsV1 fetches Cloud Functions 1st generation.
func (c *Client) listFunctionsV1(token string) ([]CloudFunction, error) {
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
		return nil, fmt.Errorf("failed to list v1 functions: %s", string(body))
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
		return nil, fmt.Errorf("failed to parse v1 functions: %v", err)
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
			cf.DisplayName = parts[5] + " (v1)"
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

// listFunctionsV2 fetches Cloud Functions 2nd generation.
func (c *Client) listFunctionsV2(token string) ([]CloudFunction, error) {
	url := fmt.Sprintf("https://cloudfunctions.googleapis.com/v2/projects/%s/locations/-/functions", c.currentProject)

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
		return nil, fmt.Errorf("failed to list v2 functions: %s", string(body))
	}

	var result struct {
		Functions []struct {
			Name        string `json:"name"`
			State       string `json:"state"` // v2 uses "state" not "status"
			Environment string `json:"environment"`
			UpdateTime  string `json:"updateTime"`
			BuildConfig *struct {
				Runtime    string `json:"runtime"`
				EntryPoint string `json:"entryPoint"`
			} `json:"buildConfig"`
			ServiceConfig *struct {
				Uri                  string `json:"uri"`
				AvailableMemory      string `json:"availableMemory"`
				TimeoutSeconds       int    `json:"timeoutSeconds"`
				AvailableCpu         string `json:"availableCpu"`
				MaxInstanceCount     int    `json:"maxInstanceCount"`
				MinInstanceCount     int    `json:"minInstanceCount"`
			} `json:"serviceConfig"`
			EventTrigger *struct {
				TriggerRegion string `json:"triggerRegion"`
				EventType     string `json:"eventType"`
				PubsubTopic   string `json:"pubsubTopic"`
			} `json:"eventTrigger"`
		} `json:"functions"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse v2 functions: %v", err)
	}

	var functions []CloudFunction
	for _, f := range result.Functions {
		cf := CloudFunction{
			Name:      f.Name,
			Status:    f.State,
			UpdatedAt: f.UpdateTime,
		}

		// Extract display name and region from full name
		parts := strings.Split(f.Name, "/")
		if len(parts) >= 6 {
			cf.DisplayName = parts[5] + " (v2)"
			cf.Region = parts[3]
		}

		// Get build config
		if f.BuildConfig != nil {
			cf.Runtime = f.BuildConfig.Runtime
			cf.EntryPoint = f.BuildConfig.EntryPoint
		}

		// Get service config
		if f.ServiceConfig != nil {
			cf.Memory = f.ServiceConfig.AvailableMemory
			if f.ServiceConfig.TimeoutSeconds > 0 {
				cf.Timeout = fmt.Sprintf("%ds", f.ServiceConfig.TimeoutSeconds)
			}
			if f.ServiceConfig.Uri != "" {
				cf.TriggerType = "HTTP"
				cf.TriggerURL = f.ServiceConfig.Uri
			}
		}

		// Check event trigger
		if f.EventTrigger != nil {
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
// Supports both v1 (Cloud Functions) and v2 (Cloud Run) functions.
func (c *Client) GetFunctionLogs(functionName string, limit int) ([]LogEntry, error) {
	if c.currentProject == "" {
		return nil, fmt.Errorf("no project selected")
	}

	token, err := c.getFirebaseToken()
	if err != nil {
		return nil, err
	}

	// Strip version suffix from display name (e.g., "myFunc (v2)" -> "myFunc")
	cleanName := functionName
	if name, found := strings.CutSuffix(functionName, " (v1)"); found {
		cleanName = name
	} else if name, found := strings.CutSuffix(functionName, " (v2)"); found {
		cleanName = name
	}

	var allLogs []LogEntry

	// Fetch v1 function logs
	v1Logs, _ := c.getFunctionLogsV1(token, cleanName, limit)
	allLogs = append(allLogs, v1Logs...)

	// Fetch v2 function logs (Cloud Run)
	v2Logs, _ := c.getFunctionLogsV2(token, cleanName, limit)
	allLogs = append(allLogs, v2Logs...)

	// Sort by timestamp descending and limit
	sortLogsByTimestamp(allLogs)
	if len(allLogs) > limit {
		allLogs = allLogs[:limit]
	}

	return allLogs, nil
}

// getFunctionLogsV1 fetches logs for v1 Cloud Functions.
func (c *Client) getFunctionLogsV1(token, functionName string, limit int) ([]LogEntry, error) {
	filter := fmt.Sprintf(`resource.type="cloud_function" AND resource.labels.function_name="%s"`, functionName)
	return c.fetchLogs(token, functionName, filter, limit)
}

// getFunctionLogsV2 fetches logs for v2 Cloud Functions (Cloud Run).
func (c *Client) getFunctionLogsV2(token, functionName string, limit int) ([]LogEntry, error) {
	filter := fmt.Sprintf(`resource.type="cloud_run_revision" AND resource.labels.service_name="%s"`, functionName)
	return c.fetchLogs(token, functionName, filter, limit)
}

// fetchLogs executes a Cloud Logging query with the given filter.
func (c *Client) fetchLogs(token, functionName, filter string, limit int) ([]LogEntry, error) {
	requestBody := map[string]any{
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

// sortLogsByTimestamp sorts log entries by timestamp in descending order.
func sortLogsByTimestamp(logs []LogEntry) {
	for i := 0; i < len(logs)-1; i++ {
		for j := i + 1; j < len(logs); j++ {
			if logs[i].Timestamp < logs[j].Timestamp {
				logs[i], logs[j] = logs[j], logs[i]
			}
		}
	}
}

// formatLogTimestamp formats a timestamp for display.
func formatLogTimestamp(ts string) string {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return ts
	}
	return t.Local().Format("15:04:05")
}
