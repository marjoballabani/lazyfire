# Lazyfire Feature Expansion Plan

User priority: **Query Builder** + expand beyond Firestore to other Firebase services.

---

## Priority 1: Query Builder (Firestore)

Interactive query building for Firestore collections.

### UI Design
```
┌─ Query Builder ─────────────────────────────┐
│ Collection: users                           │
│                                             │
│ WHERE:                                      │
│   [status] [==] [active]         [+ Add]    │
│   [age]    [>]  [18]             [x Remove] │
│                                             │
│ ORDER BY: [created] [DESC ▼]                │
│ LIMIT: [50]                                 │
│                                             │
│ Preview: .where("status","==","active")     │
│          .where("age",">",18)               │
│          .orderBy("created","desc")         │
│          .limit(50)                         │
│                                             │
│        [Execute]  [Save Query]  [Cancel]    │
└─────────────────────────────────────────────┘
```

### Implementation
- `Q` key opens query builder modal on collections panel
- Operators: `==`, `!=`, `<`, `<=`, `>`, `>=`, `in`, `array-contains`
- OrderBy: field + direction (ASC/DESC)
- Limit: number input
- Save queries to config file for reuse
- Execute replaces tree view with query results

### Files to Modify
- `pkg/gui/query.go` (new) - Query builder modal and logic
- `pkg/firebase/client.go` - Add `RunQuery()` method
- `pkg/gui/keybindings.go` - Add `Q` binding
- `pkg/config/config.go` - Saved queries storage

### Detailed Implementation Steps

**Step 1: Add Firestore Query API** (`pkg/firebase/client.go`)
```go
type QueryFilter struct {
    Field    string
    Operator string  // ==, !=, <, <=, >, >=, in, array-contains
    Value    interface{}
}

type QueryOptions struct {
    Filters []QueryFilter
    OrderBy string
    OrderDir string  // ASC or DESC
    Limit   int
}

func (c *Client) RunQuery(collectionPath string, opts QueryOptions) ([]Document, error)
```
- Use Firestore REST API's `:runQuery` endpoint
- Build structured query from QueryOptions
- Return documents matching query

**Step 2: Add Query State to Gui** (`pkg/gui/gui.go`)
```go
// In Gui struct
queryModalOpen    bool
queryFilters      []QueryFilter
queryOrderBy      string
queryOrderDir     string
queryLimit        int
queryActiveField  int  // Which field is being edited
queryResults      []Document  // Cached query results
```

**Step 3: Create Query Modal** (`pkg/gui/query.go`)
- `openQueryModal()` - Initialize modal with current collection
- `renderQueryModal()` - Draw the query builder UI
- `handleQueryInput()` - Handle keypresses in modal
- `addQueryFilter()` - Add new where clause
- `removeQueryFilter()` - Remove where clause
- `executeQuery()` - Run query and display results
- `closeQueryModal()` - Close without executing

**Step 4: Add Keybinding** (`pkg/gui/keybindings.go`)
- `Q` key opens query modal (only when collection is selected)
- Modal navigation: Tab between fields, Enter to confirm
- Escape to close modal

**Step 5: Update Layout** (`pkg/gui/layout.go`)
- Add query modal view rendering
- Show query results in tree panel (replace normal docs)
- Indicator in tree title when showing query results

**Step 6: Optional - Save Queries** (`pkg/config/config.go`)
```yaml
savedQueries:
  - name: "Active users"
    collection: "users"
    filters:
      - field: "status"
        operator: "=="
        value: "active"
    orderBy: "created"
    orderDir: "DESC"
    limit: 100
```

---

## Priority 2: Cloud Functions View

New view mode for Cloud Functions (toggle with `F` key).

### Features
- List all deployed functions for project
- Show function status (active, deploying, failed)
- View function logs (streaming or last N lines)
- Function details: trigger type, runtime, region, memory

### UI Layout
```
┌─ Projects ─┬─ Functions ──────────┬─ Logs / Details ─────┐
│ project-1  │ processOrder         │ 2024-01-09 12:34:56  │
│ project-2  │ sendEmail            │ INFO: Processing...  │
│ project-3  │ onUserCreate         │ INFO: Email sent     │
│            │ scheduledCleanup     │ ERROR: Timeout       │
└────────────┴──────────────────────┴──────────────────────┘
```

### Implementation
- `F` key toggles between Firestore mode and Functions mode
- Uses `firebase functions:list` or REST API
- `firebase functions:log` for logs
- Filter functions by name
- Details panel shows: trigger, runtime, region, memory, timeout

### Files to Create
- `pkg/firebase/functions.go` - Functions API client
- `pkg/gui/functions_view.go` - Functions UI handling

---

## Priority 3: Realtime Database View

Similar to Firestore but for RTDB (toggle with `R` key).

### Features
- Browse RTDB tree structure
- View JSON at any path
- Same filtering/copy/save as Firestore

### Implementation
- Uses `firebase database:get` or REST API
- Tree view shows RTDB paths
- Details shows JSON value at path

---

## Priority 4: Storage Browser

Browse Cloud Storage buckets (toggle with `S` key).

### Features
- List buckets and folders
- View file metadata (size, type, created)
- Download files to ~/Downloads
- Preview text/JSON files

---

## Priority 5: Hosting Sites View

View hosting deployments (toggle with `H` key).

### Features
- List hosting sites
- Show deployment history
- View current deployment details

---

## Mode Switching

Add service mode switching:
- `1` or `D` - Firestore (Database) - current default
- `2` or `F` - Functions
- `3` or `R` - Realtime Database
- `4` or `S` - Storage
- `5` or `H` - Hosting

Status bar shows current mode: `[Firestore] project-name`

---

## Implementation Order

| Phase | Feature | Complexity |
|-------|---------|------------|
| 1 | Query Builder | Medium |
| 2 | Functions View | Medium |
| 3 | RTDB View | Low (similar to Firestore) |
| 4 | Storage Browser | Medium |
| 5 | Hosting View | Low |

---

## Verification

### Query Builder
1. Open lazyfire, select a project and collection
2. Press `Q` to open query builder
3. Add where clause, set orderBy and limit
4. Execute and verify filtered results in tree
5. Save query, close, reopen builder, load saved query

### Functions View
1. Press `F` to switch to functions mode
2. Verify functions list loads
3. Select function, view details
4. View logs in details panel
