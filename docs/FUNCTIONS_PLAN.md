# Cloud Functions View Implementation Plan

## Overview
Add Cloud Functions browsing to lazyfire using a tabbed view in the Collections panel.

## UI Design

```
┌─ Projects ────┬─ [Collections] [Functions] ─┬─ Tree ──────────┬─ Details ─────────┐
│               │                              │                 │                   │
│ 🔥 my-app     │ 📁 users                     │ ▼ 📁 users      │ Function Logs:    │
│   my-app-stg  │ 📁 products                  │     📄 user_001 │ 12:34:56 INFO ... │
│               │ 📁 orders                    │     📄 user_002 │ 12:34:57 INFO ... │
│               │                              │                 │ 12:34:58 ERROR ...|
└───────────────┴──────────────────────────────┴─────────────────┴───────────────────┘
```

When Functions tab is active:
```
┌─ Projects ────┬─ [Collections] [Functions] ─┬─ Function Details ─┬─ Logs ──────────┐
│               │                              │                    │                 │
│ 🔥 my-app     │ ⚡ processOrder              │ Name: processOrder │ 12:34:56 INFO   │
│               │ ⚡ sendEmail                 │ Runtime: nodejs18  │ Processing...   │
│               │ ⚡ onUserCreate              │ Region: us-central1│ 12:34:57 INFO   │
│               │ ⚡ scheduledCleanup          │ Memory: 256MB      │ Email sent      │
│               │                              │ Trigger: HTTP      │ 12:34:58 ERROR  │
└───────────────┴──────────────────────────────┴────────────────────┴─────────────────┘
```

## Implementation Steps

### Step 1: Add Firebase Functions API (`pkg/firebase/functions.go`)

```go
// CloudFunction represents a deployed Cloud Function
type CloudFunction struct {
    Name        string // Full resource name
    DisplayName string // Short name for display
    Status      string // ACTIVE, DEPLOYING, OFFLINE, etc.
    Runtime     string // nodejs18, python311, go121, etc.
    Region      string // us-central1, europe-west1, etc.
    Memory      string // 256MB, 512MB, etc.
    Timeout     string // 60s, 540s, etc.
    TriggerType string // HTTP, Firestore, PubSub, etc.
    TriggerURL  string // For HTTP triggers
    UpdatedAt   string // Last deployment time
}

// LogEntry represents a function log entry
type LogEntry struct {
    Timestamp string
    Severity  string // INFO, WARNING, ERROR, DEBUG
    Message   string
    Function  string
}

// ListFunctions fetches all Cloud Functions for current project
func (c *Client) ListFunctions() ([]CloudFunction, error)

// GetFunctionLogs fetches recent logs for a function
func (c *Client) GetFunctionLogs(functionName string, limit int) ([]LogEntry, error)
```

**API Endpoints:**
- List functions: `GET https://cloudfunctions.googleapis.com/v1/projects/{project}/locations/-/functions`
- Get logs: `POST https://logging.googleapis.com/v2/entries:list`

### Step 2: Add Functions State (`pkg/gui/gui.go`)

```go
// Add to Gui struct
collectionsTab      string              // "collections" or "functions"
functions           []firebase.CloudFunction
selectedFunctionIdx int
functionsFilter     string
currentFunction     *firebase.CloudFunction
functionLogs        []firebase.LogEntry
logsAutoRefresh     bool
logsRefreshTicker   *time.Ticker
```

### Step 3: Add Tab Switching (`pkg/gui/actions.go`)

```go
// switchCollectionsTab toggles between Collections and Functions tabs
func (g *Gui) switchCollectionsTab() error {
    if g.collectionsTab == "collections" {
        g.collectionsTab = "functions"
        g.loadFunctions()
    } else {
        g.collectionsTab = "collections"
    }
    return nil
}
```

**Keybinding:** `Tab` when focused on Collections panel switches tabs.

### Step 4: Update Layout (`pkg/gui/layout.go`)

1. **Render tab bar** at top of Collections panel:
   ```
   [Collections] [Functions]
   ```
   - Active tab: highlighted with active color
   - Inactive tab: dim

2. **Conditional rendering:**
   - If `collectionsTab == "collections"`: show collections list (existing)
   - If `collectionsTab == "functions"`: show functions list

3. **Tree panel changes:**
   - Collections mode: show documents/subcollections (existing)
   - Functions mode: show function metadata

4. **Details panel changes:**
   - Collections mode: show document JSON (existing)
   - Functions mode: show live logs

### Step 5: Functions List Rendering (`pkg/gui/layout.go`)

```go
func (g *Gui) updateFunctionsView(v *gocui.View) {
    // Show function list with status indicators
    // ⚡ functionName (region) [ACTIVE]
    // Color-code by status: green=ACTIVE, yellow=DEPLOYING, red=OFFLINE
}
```

### Step 6: Live Logs Implementation (`pkg/gui/functions.go`)

```go
// startLogsRefresh starts auto-refreshing logs every 3 seconds
func (g *Gui) startLogsRefresh() {
    g.logsRefreshTicker = time.NewTicker(3 * time.Second)
    go func() {
        for range g.logsRefreshTicker.C {
            g.refreshFunctionLogs()
        }
    }()
}

// stopLogsRefresh stops the auto-refresh
func (g *Gui) stopLogsRefresh() {
    if g.logsRefreshTicker != nil {
        g.logsRefreshTicker.Stop()
    }
}

// refreshFunctionLogs fetches latest logs and updates UI
func (g *Gui) refreshFunctionLogs() {
    if g.currentFunction == nil {
        return
    }
    logs, err := g.firebaseClient.GetFunctionLogs(g.currentFunction.DisplayName, 50)
    if err == nil {
        g.functionLogs = logs
        g.g.Update(func(*gocui.Gui) error {
            return g.Layout(g.g)
        })
    }
}
```

### Step 7: Logs Rendering (`pkg/gui/layout.go`)

```go
func (g *Gui) updateLogsView(v *gocui.View) {
    // Render logs with color-coded severity
    // 12:34:56 INFO  Processing order abc123
    // 12:34:57 ERROR Failed to send email: timeout
    //
    // Colors: INFO=green, WARNING=yellow, ERROR=red, DEBUG=dim
}
```

### Step 8: Keybindings (`pkg/gui/keybindings.go`)

| Key | Context | Action |
|-----|---------|--------|
| `Tab` | Collections panel | Switch between Collections/Functions tabs |
| `j/k` | Functions tab | Navigate function list |
| `Enter` | Functions tab | Select function, show logs |
| `r` | Functions tab (logs) | Manual refresh logs |
| `/` | Functions tab | Filter functions by name |

## Files to Modify

| File | Changes |
|------|---------|
| `pkg/firebase/functions.go` | **NEW** - Functions API client |
| `pkg/gui/gui.go` | Add functions state, tab state |
| `pkg/gui/layout.go` | Tab rendering, functions list, logs view |
| `pkg/gui/handlers.go` | `selectFunction()` handler |
| `pkg/gui/actions.go` | Tab switching, logs refresh control |
| `pkg/gui/keybindings.go` | Tab key binding for Collections panel |
| `pkg/gui/filter.go` | Add `getFilteredFunctions()` |

## Data Flow

```
1. User selects project
   └─> loadFunctions() fetches function list

2. User presses Tab on Collections panel
   └─> switchCollectionsTab() toggles to Functions
   └─> UI re-renders with functions list

3. User selects a function (Enter)
   └─> selectFunction() sets currentFunction
   └─> startLogsRefresh() begins polling
   └─> Details panel shows live logs

4. User navigates away from Functions
   └─> stopLogsRefresh() stops polling
```

## Verification

1. Launch lazyfire, select a project with deployed functions
2. Navigate to Collections panel, press `Tab`
3. Verify tab switches to Functions, functions list loads
4. Navigate functions with `j/k`, filter with `/`
5. Press `Enter` on a function
6. Verify logs appear in details panel
7. Verify logs auto-refresh (new entries appear)
8. Press `Tab` to switch back to Collections
9. Verify Firestore browsing still works normally
