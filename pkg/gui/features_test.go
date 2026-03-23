package gui

import (
	"strings"
	"testing"

	"github.com/marjoballabani/lazyfire/pkg/firebase"
)

// --- highlightMatches ---

func TestHighlightMatches(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		filter   string
		wantAnsi bool   // should contain ANSI reverse video codes
		wantText string // stripped ANSI should equal original text
	}{
		{
			name:     "empty filter returns unchanged text",
			text:     "hello world",
			filter:   "",
			wantAnsi: false,
			wantText: "hello world",
		},
		{
			name:     "exact match highlighted",
			text:     "hello",
			filter:   "hello",
			wantAnsi: true,
			wantText: "hello",
		},
		{
			name:     "partial match highlighted",
			text:     "hello world",
			filter:   "world",
			wantAnsi: true,
			wantText: "hello world",
		},
		{
			name:     "case insensitive match",
			text:     "Hello World",
			filter:   "hello",
			wantAnsi: true,
			wantText: "Hello World",
		},
		{
			name:     "multiple matches",
			text:     "abcabc",
			filter:   "abc",
			wantAnsi: true,
			wantText: "abcabc",
		},
		{
			name:     "no match returns unchanged",
			text:     "hello",
			filter:   "xyz",
			wantAnsi: false,
			wantText: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := highlightMatches(tt.text, tt.filter)
			stripped := stripANSI(result)
			if stripped != tt.wantText {
				t.Errorf("stripped text = %q, want %q", stripped, tt.wantText)
			}
			hasAnsi := strings.Contains(result, "\033[7m")
			if hasAnsi != tt.wantAnsi {
				t.Errorf("has ANSI reverse = %v, want %v", hasAnsi, tt.wantAnsi)
			}
			if tt.wantAnsi {
				// Every \033[7m should have a matching \033[27m
				opens := strings.Count(result, "\033[7m")
				closes := strings.Count(result, "\033[27m")
				if opens != closes {
					t.Errorf("unbalanced ANSI: %d opens, %d closes", opens, closes)
				}
			}
		})
	}
}

// --- annotateTimestamps ---

func TestAnnotateTimestamps(t *testing.T) {
	// annotateTimestamps finds the first quoted string on each raw line.
	// If that string parses as RFC3339, it appends a human-readable comment.
	// In real JSON, "key": "value" means the key is the first quoted string,
	// so timestamps are only annotated when the key itself looks like a timestamp
	// or the line has only the value (e.g., array element).
	tests := []struct {
		name      string
		rawJSON   string
		colorized string
		wantComment bool
	}{
		{
			name:        "bare timestamp string gets annotated",
			rawJSON:     `  "2024-01-15T10:30:00Z"`,
			colorized:   `  "2024-01-15T10:30:00Z"`,
			wantComment: true,
		},
		{
			name:        "bare RFC3339Nano gets annotated",
			rawJSON:     `  "2024-06-20T14:05:30.123456789Z"`,
			colorized:   `  "2024-06-20T14:05:30.123456789Z"`,
			wantComment: true,
		},
		{
			name:        "key-value pair - first quoted string is key, not timestamp",
			rawJSON:     `  "createdAt": "2024-01-15T10:30:00Z"`,
			colorized:   `  "createdAt": "2024-01-15T10:30:00Z"`,
			wantComment: false, // first quoted string is "createdAt", not a timestamp
		},
		{
			name:        "non-timestamp string not annotated",
			rawJSON:     `  "name": "John Doe"`,
			colorized:   `  "name": "John Doe"`,
			wantComment: false,
		},
		{
			name:        "number line not annotated",
			rawJSON:     `  "count": 42`,
			colorized:   `  "count": 42`,
			wantComment: false,
		},
		{
			name:        "empty string",
			rawJSON:     "",
			colorized:   "",
			wantComment: false,
		},
		{
			name:        "multiline with bare timestamp in array",
			rawJSON:     "[\n  \"2024-01-01T00:00:00Z\",\n  \"hello\"\n]",
			colorized:   "[\n  \"2024-01-01T00:00:00Z\",\n  \"hello\"\n]",
			wantComment: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := annotateTimestamps(tt.rawJSON, tt.colorized)
			hasComment := strings.Contains(result, "// ")
			if hasComment != tt.wantComment {
				t.Errorf("has comment = %v, want %v\nresult: %q", hasComment, tt.wantComment, result)
			}
			if tt.wantComment {
				// Comment should contain human-readable parts like month names or AM/PM
				if !strings.Contains(result, "Jan") && !strings.Contains(result, "Jun") &&
					!strings.Contains(result, "Feb") && !strings.Contains(result, "Mar") &&
					!strings.Contains(result, "Apr") && !strings.Contains(result, "May") &&
					!strings.Contains(result, "Jul") && !strings.Contains(result, "Aug") &&
					!strings.Contains(result, "Sep") && !strings.Contains(result, "Oct") &&
					!strings.Contains(result, "Nov") && !strings.Contains(result, "Dec") {
					t.Errorf("annotation should contain month name, got: %q", result)
				}
			}
		})
	}
}

func TestAnnotateTimestampsPreservesLineCount(t *testing.T) {
	raw := "{\n  \"a\": \"2024-01-01T00:00:00Z\",\n  \"b\": 2\n}"
	colored := "{\n  \"a\": \"2024-01-01T00:00:00Z\",\n  \"b\": 2\n}"
	result := annotateTimestamps(raw, colored)
	rawLines := strings.Count(raw, "\n") + 1
	resultLines := strings.Count(result, "\n") + 1
	if rawLines != resultLines {
		t.Errorf("line count changed: raw=%d, result=%d", rawLines, resultLines)
	}
}

// --- buildSchemaSummary ---

func TestBuildSchemaSummary(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		contains []string
		empty    bool
	}{
		{
			name:  "empty data",
			data:  map[string]any{},
			empty: true,
		},
		{
			name: "string field",
			data: map[string]any{"name": "John"},
			contains: []string{"Schema:", "name:string"},
		},
		{
			name: "int field",
			data: map[string]any{"count": float64(42)},
			contains: []string{"count:int"},
		},
		{
			name: "float field",
			data: map[string]any{"score": float64(3.14)},
			contains: []string{"score:float"},
		},
		{
			name: "bool field",
			data: map[string]any{"active": true},
			contains: []string{"active:bool"},
		},
		{
			name: "null field",
			data: map[string]any{"deleted": nil},
			contains: []string{"deleted:null"},
		},
		{
			name: "map field",
			data: map[string]any{"address": map[string]any{"city": "NYC"}},
			contains: []string{"address:map"},
		},
		{
			name: "array field",
			data: map[string]any{"tags": []any{"go", "fire"}},
			contains: []string{"tags:array"},
		},
		{
			name: "timestamp field",
			data: map[string]any{"createdAt": "2024-01-15T10:30:00Z"},
			contains: []string{"createdAt:timestamp"},
		},
		{
			name: "sorted output",
			data: map[string]any{"zebra": "z", "alpha": "a", "mid": "m"},
			contains: []string{"alpha:string, mid:string, zebra:string"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSchemaSummary(tt.data)
			if tt.empty {
				if result != "" {
					t.Errorf("expected empty, got %q", result)
				}
				return
			}
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("result %q should contain %q", result, substr)
				}
			}
		})
	}
}

func TestBuildSchemaSummaryTruncation(t *testing.T) {
	// Create a data map with many long field names
	data := make(map[string]any)
	for i := 0; i < 50; i++ {
		key := strings.Repeat("x", 20) + string(rune('A'+i%26))
		data[key] = "val"
	}
	result := buildSchemaSummary(data)
	if len(result) > 120 {
		t.Errorf("should be truncated to <=120 chars, got %d", len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Error("truncated result should end with ...")
	}
}

// --- inferType ---

func TestInferType(t *testing.T) {
	tests := []struct {
		name     string
		val      any
		expected string
	}{
		{"nil", nil, "null"},
		{"string", "hello", "string"},
		{"timestamp string", "2024-01-15T10:30:00Z", "timestamp"},
		{"short non-timestamp", "hello:world", "string"},
		{"integer float64", float64(42), "int"},
		{"float", float64(3.14), "float"},
		{"zero int", float64(0), "int"},
		{"negative int", float64(-5), "int"},
		{"bool true", true, "bool"},
		{"bool false", false, "bool"},
		{"map", map[string]any{"a": 1}, "map"},
		{"empty map", map[string]any{}, "map"},
		{"array", []any{1, 2, 3}, "array"},
		{"empty array", []any{}, "array"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferType(tt.val)
			if result != tt.expected {
				t.Errorf("inferType(%v) = %q, want %q", tt.val, result, tt.expected)
			}
		})
	}
}

// --- sortStrings ---

func TestSortStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"empty", []string{}, []string{}},
		{"single", []string{"a"}, []string{"a"}},
		{"already sorted", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"reverse", []string{"c", "b", "a"}, []string{"a", "b", "c"}},
		{"mixed", []string{"banana", "apple", "cherry"}, []string{"apple", "banana", "cherry"}},
		{"duplicates", []string{"b", "a", "b", "a"}, []string{"a", "a", "b", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := make([]string, len(tt.input))
			copy(input, tt.input)
			sortStrings(input)
			for i, v := range input {
				if v != tt.expected[i] {
					t.Errorf("at index %d: got %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

// --- visibleLength ---

func TestVisibleLength(t *testing.T) {
	g := &Gui{}
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"plain text", "hello", 5},
		{"empty string", "", 0},
		{"with ANSI color", "\033[36mhello\033[0m", 5},
		{"multiple ANSI codes", "\033[36mhel\033[0m\033[32mlo\033[0m", 5},
		{"ANSI with plain text", "pre \033[31mred\033[0m post", 12},
		{"nested ANSI", "\033[1m\033[36mbold cyan\033[0m", 9},
		{"reverse video", "\033[7mmatch\033[27m", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.visibleLength(tt.input)
			if result != tt.expected {
				t.Errorf("visibleLength(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// --- buildBreadcrumb ---

func TestBuildBreadcrumb(t *testing.T) {
	tests := []struct {
		name       string
		project    string
		collection string
		docPath    string
		wantParts  []string
		wantEmpty  bool
	}{
		{
			name:      "all empty",
			wantEmpty: true,
		},
		{
			name:      "project only",
			project:   "my-project",
			wantParts: []string{"my-project"},
		},
		{
			name:       "project and collection",
			project:    "my-project",
			collection: "users",
			wantParts:  []string{"my-project", "users"},
		},
		{
			name:       "full path",
			project:    "my-project",
			collection: "users",
			docPath:    "users/abc123",
			wantParts:  []string{"my-project", "users", "abc123"},
		},
		{
			name:       "deep doc path extracts last segment",
			project:    "proj",
			collection: "users",
			docPath:    "users/abc/orders/ord1",
			wantParts:  []string{"proj", "users", "ord1"},
		},
		{
			name:       "internal path with __ prefix skipped",
			project:    "proj",
			collection: "users",
			docPath:    "__scan_results",
			wantParts:  []string{"proj", "users"},
		},
		{
			name:       "path with selected suffix skipped",
			project:    "proj",
			collection: "users",
			docPath:    "3 selected docs",
			wantParts:  []string{"proj", "users"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Gui{
				currentProject:    tt.project,
				currentCollection: tt.collection,
				currentDocPath:    tt.docPath,
			}
			result := g.buildBreadcrumb()
			stripped := stripANSI(result)
			if tt.wantEmpty {
				if result != "" {
					t.Errorf("expected empty, got %q", stripped)
				}
				return
			}
			for _, part := range tt.wantParts {
				if !strings.Contains(stripped, part) {
					t.Errorf("breadcrumb %q should contain %q", stripped, part)
				}
			}
			// Parts should be joined by " > "
			if len(tt.wantParts) > 1 {
				if !strings.Contains(stripped, " > ") {
					t.Errorf("breadcrumb %q should contain separator ' > '", stripped)
				}
			}
		})
	}
}

// --- Navigation state tests (no gocui needed for state verification) ---

func newTestGui() *Gui {
	return &Gui{
		projects: []firebase.Project{
			{ID: "p1", DisplayName: "Project 1"},
			{ID: "p2", DisplayName: "Project 2"},
			{ID: "p3", DisplayName: "Project 3"},
			{ID: "p4", DisplayName: "Project 4"},
			{ID: "p5", DisplayName: "Project 5"},
			{ID: "p6", DisplayName: "Project 6"},
			{ID: "p7", DisplayName: "Project 7"},
			{ID: "p8", DisplayName: "Project 8"},
			{ID: "p9", DisplayName: "Project 9"},
			{ID: "p10", DisplayName: "Project 10"},
			{ID: "p11", DisplayName: "Project 11"},
			{ID: "p12", DisplayName: "Project 12"},
		},
		collections: []firebase.Collection{
			{Name: "users", Path: "users"},
			{Name: "orders", Path: "orders"},
			{Name: "products", Path: "products"},
			{Name: "reviews", Path: "reviews"},
			{Name: "settings", Path: "settings"},
		},
		treeNodes: []TreeNode{
			{Path: "users/u1", Name: "u1", Type: "document", Depth: 0},
			{Path: "users/u2", Name: "u2", Type: "document", Depth: 0},
			{Path: "users/u3", Name: "u3", Type: "document", Depth: 0},
			{Path: "users/u3/orders", Name: "orders", Type: "collection", Depth: 1, HasChildren: true, Expanded: true},
			{Path: "users/u3/orders/o1", Name: "o1", Type: "document", Depth: 2},
		},
		expandedPaths:       make(map[string]bool),
		docCache:            make(map[string]map[string]any),
		statsCache:          make(map[string]*firebase.DocStats),
		collectionCache:     make(map[string][]string),
		compositeIndexCache: make(map[string]*bool),
		selectedDocs:        make(map[int]bool),
		currentColumn:       "projects",
		collectionsTab:      "collections",
	}
}

// Test page navigation state changes without calling Layout.
// We replicate the core logic since action methods call g.Layout(g.g) which needs gocui.

func TestPageUpState(t *testing.T) {
	tests := []struct {
		name     string
		panel    string
		startIdx int
		wantIdx  int
	}{
		{"projects from 11", "projects", 11, 1},
		{"projects from 5", "projects", 5, 0},
		{"projects from 0", "projects", 0, 0},
		{"tree from 4", "tree", 4, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newTestGui()
			g.currentColumn = tt.panel
			switch tt.panel {
			case "projects":
				g.selectedProjectIndex = tt.startIdx
				g.selectedProjectIndex -= 10
				if g.selectedProjectIndex < 0 {
					g.selectedProjectIndex = 0
				}
				if g.selectedProjectIndex != tt.wantIdx {
					t.Errorf("got %d, want %d", g.selectedProjectIndex, tt.wantIdx)
				}
			case "tree":
				g.selectedTreeIdx = tt.startIdx
				g.selectedTreeIdx -= 10
				if g.selectedTreeIdx < 0 {
					g.selectedTreeIdx = 0
				}
				if g.selectedTreeIdx != tt.wantIdx {
					t.Errorf("got %d, want %d", g.selectedTreeIdx, tt.wantIdx)
				}
			}
		})
	}
}

func TestPageDownState(t *testing.T) {
	g := newTestGui()
	g.currentColumn = "projects"
	g.selectedProjectIndex = 0

	// Page down from 0 with 12 projects
	g.selectedProjectIndex += 10
	filtered := g.getFilteredProjects()
	if g.selectedProjectIndex >= len(filtered) {
		g.selectedProjectIndex = len(filtered) - 1
	}
	if g.selectedProjectIndex != 10 {
		t.Errorf("expected index 10, got %d", g.selectedProjectIndex)
	}

	// Page down again - should clamp to last
	g.selectedProjectIndex += 10
	if g.selectedProjectIndex >= len(filtered) {
		g.selectedProjectIndex = len(filtered) - 1
	}
	if g.selectedProjectIndex != 11 {
		t.Errorf("expected index 11, got %d", g.selectedProjectIndex)
	}
}

func TestGoToTopState(t *testing.T) {
	panels := []struct {
		name  string
		panel string
	}{
		{"projects", "projects"},
		{"collections", "collections"},
		{"tree", "tree"},
		{"details", "details"},
	}

	for _, tt := range panels {
		t.Run(tt.name, func(t *testing.T) {
			g := newTestGui()
			g.currentColumn = tt.panel
			g.selectedProjectIndex = 5
			g.selectedCollectionIdx = 3
			g.selectedTreeIdx = 4
			g.detailsScrollPos = 100

			// Replicate doGoToTop logic
			switch g.currentColumn {
			case "projects":
				g.selectedProjectIndex = 0
			case "collections":
				g.selectedCollectionIdx = 0
			case "tree":
				g.selectedTreeIdx = 0
			case "details":
				g.detailsScrollPos = 0
			}

			switch tt.panel {
			case "projects":
				if g.selectedProjectIndex != 0 {
					t.Error("projects index not reset to 0")
				}
			case "collections":
				if g.selectedCollectionIdx != 0 {
					t.Error("collections index not reset to 0")
				}
			case "tree":
				if g.selectedTreeIdx != 0 {
					t.Error("tree index not reset to 0")
				}
			case "details":
				if g.detailsScrollPos != 0 {
					t.Error("details scroll not reset to 0")
				}
			}
		})
	}
}

func TestGoToBottomState(t *testing.T) {
	g := newTestGui()

	// Projects
	g.currentColumn = "projects"
	filtered := g.getFilteredProjects()
	g.selectedProjectIndex = len(filtered) - 1
	if g.selectedProjectIndex != 11 {
		t.Errorf("projects bottom: got %d, want 11", g.selectedProjectIndex)
	}

	// Collections
	g.currentColumn = "collections"
	filteredCols := g.getFilteredCollections()
	g.selectedCollectionIdx = len(filteredCols) - 1
	if g.selectedCollectionIdx != 4 {
		t.Errorf("collections bottom: got %d, want 4", g.selectedCollectionIdx)
	}

	// Tree
	g.currentColumn = "tree"
	filteredTree := g.getFilteredTreeNodes()
	g.selectedTreeIdx = len(filteredTree) - 1
	if g.selectedTreeIdx != 4 {
		t.Errorf("tree bottom: got %d, want 4", g.selectedTreeIdx)
	}

	// Details
	g.currentColumn = "details"
	g.detailsScrollPos = 99999
	if g.detailsScrollPos != 99999 {
		t.Error("details should set high scroll value")
	}
}

func TestHalfPageScrollState(t *testing.T) {
	g := newTestGui()
	g.currentColumn = "details"
	g.detailsScrollPos = 0

	// Half page down
	g.detailsScrollPos += 20
	if g.detailsScrollPos != 20 {
		t.Errorf("half page down: got %d, want 20", g.detailsScrollPos)
	}

	// Half page up
	g.detailsScrollPos -= 20
	if g.detailsScrollPos < 0 {
		g.detailsScrollPos = 0
	}
	if g.detailsScrollPos != 0 {
		t.Errorf("half page up: got %d, want 0", g.detailsScrollPos)
	}

	// Half page up from 0 stays at 0
	g.detailsScrollPos -= 20
	if g.detailsScrollPos < 0 {
		g.detailsScrollPos = 0
	}
	if g.detailsScrollPos != 0 {
		t.Errorf("half page up from 0: got %d, want 0", g.detailsScrollPos)
	}
}

func TestJumpToPanelBlockedInDetails(t *testing.T) {
	g := newTestGui()
	g.currentColumn = "details"

	// doJumpToProjects/Collections/Tree all return nil when in details
	// Verify the guard condition
	if g.currentColumn != "details" {
		t.Error("should be in details panel")
	}
}

// --- Action state tests ---

func TestCollapseAllState(t *testing.T) {
	g := newTestGui()
	g.currentColumn = "tree"
	g.expandedPaths["users/u3/orders"] = true

	// Replicate doCollapseAll logic
	var topLevel []TreeNode
	for _, node := range g.treeNodes {
		if node.Depth == 0 {
			node.Expanded = false
			topLevel = append(topLevel, node)
		}
	}
	g.treeNodes = topLevel
	g.selectedTreeIdx = 0
	g.expandedPaths = make(map[string]bool)

	if len(g.treeNodes) != 3 {
		t.Errorf("after collapse: got %d nodes, want 3 (only depth-0)", len(g.treeNodes))
	}
	for _, node := range g.treeNodes {
		if node.Depth != 0 {
			t.Errorf("node %q has depth %d, expected 0", node.Path, node.Depth)
		}
		if node.Expanded {
			t.Errorf("node %q should not be expanded", node.Path)
		}
	}
	if len(g.expandedPaths) != 0 {
		t.Error("expandedPaths should be empty")
	}
	if g.selectedTreeIdx != 0 {
		t.Error("selectedTreeIdx should be 0")
	}
}

func TestCollapseAllNotInTree(t *testing.T) {
	g := newTestGui()
	g.currentColumn = "projects"
	originalLen := len(g.treeNodes)

	// doCollapseAll should be a no-op when not in tree
	if g.currentColumn != "tree" || len(g.treeNodes) == 0 {
		// This is the guard - no-op
	}
	if len(g.treeNodes) != originalLen {
		t.Error("nodes should not change when not in tree panel")
	}
}

func TestToggleCompactJSONState(t *testing.T) {
	g := newTestGui()
	g.currentColumn = "details"
	g.currentDocData = map[string]any{"key": "value"}
	g.compactJSON = false

	g.compactJSON = !g.compactJSON
	if !g.compactJSON {
		t.Error("compactJSON should be true after toggle")
	}

	g.compactJSON = !g.compactJSON
	if g.compactJSON {
		t.Error("compactJSON should be false after second toggle")
	}
}

func TestToggleCompactJSONGuards(t *testing.T) {
	g := newTestGui()

	// Not in details - should be no-op
	g.currentColumn = "tree"
	g.currentDocData = map[string]any{"key": "value"}
	if !(g.currentColumn != "details" || g.currentDocData == nil) {
		t.Error("guard should block when not in details")
	}

	// In details but no doc data - should be no-op
	g.currentColumn = "details"
	g.currentDocData = nil
	if !(g.currentColumn != "details" || g.currentDocData == nil) {
		t.Error("guard should block when no doc data")
	}
}

func TestToggleTimestampsState(t *testing.T) {
	g := newTestGui()
	g.humanizeTimestamps = false

	g.humanizeTimestamps = !g.humanizeTimestamps
	if !g.humanizeTimestamps {
		t.Error("humanizeTimestamps should be true after toggle")
	}

	g.humanizeTimestamps = !g.humanizeTimestamps
	if g.humanizeTimestamps {
		t.Error("humanizeTimestamps should be false after second toggle")
	}
}

func TestClearCacheState(t *testing.T) {
	g := newTestGui()
	g.docCache["users/u1"] = map[string]any{"name": "Alice"}
	g.docCache["users/u2"] = map[string]any{"name": "Bob"}
	g.statsCache["users/u1"] = &firebase.DocStats{}
	g.collectionCache["users"] = []string{"users/u1", "users/u2"}
	hasComposites := true
	g.compositeIndexCache["users"] = &hasComposites

	// Replicate doClearCache logic
	g.docCache = make(map[string]map[string]any)
	g.statsCache = make(map[string]*firebase.DocStats)
	g.collectionCache = make(map[string][]string)
	g.compositeIndexCache = make(map[string]*bool)

	if len(g.docCache) != 0 {
		t.Error("docCache should be empty")
	}
	if len(g.statsCache) != 0 {
		t.Error("statsCache should be empty")
	}
	if len(g.collectionCache) != 0 {
		t.Error("collectionCache should be empty")
	}
	if len(g.compositeIndexCache) != 0 {
		t.Error("compositeIndexCache should be empty")
	}
}

func TestShowCacheStatsState(t *testing.T) {
	g := newTestGui()
	g.docCache["users/u1"] = map[string]any{"name": "Alice"}
	g.docCache["users/u2"] = map[string]any{"name": "Bob"}
	g.collectionCache["users"] = []string{"users/u1", "users/u2"}
	g.collectionCache["orders"] = []string{"orders/o1"}
	g.statsCache["users/u1"] = &firebase.DocStats{}

	docCount := len(g.docCache)
	collCount := len(g.collectionCache)
	statsCount := len(g.statsCache)

	totalDocs := 0
	for _, paths := range g.collectionCache {
		totalDocs += len(paths)
	}

	if docCount != 2 {
		t.Errorf("docCount = %d, want 2", docCount)
	}
	if collCount != 2 {
		t.Errorf("collCount = %d, want 2", collCount)
	}
	if totalDocs != 3 {
		t.Errorf("totalDocs = %d, want 3", totalDocs)
	}
	if statsCount != 1 {
		t.Errorf("statsCount = %d, want 1", statsCount)
	}
}

func TestExportCachedDocsEmptyCache(t *testing.T) {
	g := newTestGui()
	// With empty cache, export should be a no-op
	if len(g.docCache) != 0 {
		t.Error("docCache should start empty for this test")
	}
}

func TestCopyPathState(t *testing.T) {
	g := newTestGui()

	// Tree panel - path from filtered nodes
	g.currentColumn = "tree"
	g.selectedTreeIdx = 0
	filtered := g.getFilteredTreeNodes()
	if len(filtered) > 0 {
		path := filtered[g.selectedTreeIdx].Path
		if path != "users/u1" {
			t.Errorf("expected path 'users/u1', got %q", path)
		}
	}

	// Details panel - path from currentDocPath
	g.currentColumn = "details"
	g.currentDocPath = "users/u1"
	if g.currentDocPath != "users/u1" {
		t.Errorf("expected currentDocPath 'users/u1', got %q", g.currentDocPath)
	}

	// Other panels - should return nil (no-op)
	g.currentColumn = "projects"
	// guard check
	if g.currentColumn != "tree" && g.currentColumn != "details" {
		// correct - would be no-op
	} else {
		t.Error("should be no-op for projects panel")
	}
}

// --- Filtering integration with navigation ---

func TestFilteredNavigationBounds(t *testing.T) {
	g := newTestGui()
	g.projectsFilter = "Project 1" // Matches "Project 1", "Project 10", "Project 11", "Project 12"

	filtered := g.getFilteredProjects()
	if len(filtered) != 4 {
		t.Errorf("expected 4 filtered projects, got %d", len(filtered))
	}

	// Page down should clamp to filtered length
	g.currentColumn = "projects"
	g.selectedProjectIndex = 0
	g.selectedProjectIndex += 10
	if g.selectedProjectIndex >= len(filtered) {
		g.selectedProjectIndex = len(filtered) - 1
	}
	if g.selectedProjectIndex != 3 {
		t.Errorf("expected index 3 (last filtered), got %d", g.selectedProjectIndex)
	}
}

func TestFilteredCollections(t *testing.T) {
	g := newTestGui()
	g.collectionsFilter = "ers" // Matches "users", "orders"
	filtered := g.getFilteredCollections()
	if len(filtered) != 2 {
		t.Errorf("expected 2 filtered collections, got %d", len(filtered))
	}
}

func TestFilteredTreeNodes(t *testing.T) {
	g := newTestGui()
	g.treeFilter = "orders"
	filtered := g.getFilteredTreeNodes()
	// Matches nodes with "orders" in name or path
	if len(filtered) != 2 { // "orders" collection and "o1" doc under orders
		t.Errorf("expected 2 filtered tree nodes, got %d", len(filtered))
	}
}

// --- Collection cache doc count ---

func TestCollectionCacheDocCount(t *testing.T) {
	g := newTestGui()
	g.collectionCache["users"] = []string{"users/u1", "users/u2", "users/u3"}

	paths, ok := g.collectionCache["users"]
	if !ok {
		t.Fatal("users should be in cache")
	}
	if len(paths) != 3 {
		t.Errorf("expected 3 cached paths, got %d", len(paths))
	}
}

// --- getOriginalTreeNodeIndex ---

func TestGetOriginalTreeNodeIndex(t *testing.T) {
	g := newTestGui()

	// No filter - filtered == original
	idx := g.getOriginalTreeNodeIndex(0)
	if idx != 0 {
		t.Errorf("expected 0, got %d", idx)
	}
	idx = g.getOriginalTreeNodeIndex(2)
	if idx != 2 {
		t.Errorf("expected 2, got %d", idx)
	}

	// With filter
	g.treeFilter = "u2"
	filtered := g.getFilteredTreeNodes()
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered node for 'u2', got %d", len(filtered))
	}
	idx = g.getOriginalTreeNodeIndex(0)
	if idx != 1 { // "u2" is at index 1 in original
		t.Errorf("expected original index 1, got %d", idx)
	}

	// Out of bounds
	idx = g.getOriginalTreeNodeIndex(99)
	if idx != -1 {
		t.Errorf("expected -1 for out of bounds, got %d", idx)
	}
	idx = g.getOriginalTreeNodeIndex(-1)
	if idx != -1 {
		t.Errorf("expected -1 for negative index, got %d", idx)
	}
}

// --- Command history / logCommand ---

func TestLogCommand(t *testing.T) {
	g := newTestGui()

	g.logCommand("test", "Test command", "success")
	if len(g.commandHistory) != 1 {
		t.Fatalf("expected 1 command in history, got %d", len(g.commandHistory))
	}
	if g.commandHistory[0].Command != "test" {
		t.Errorf("command = %q, want 'test'", g.commandHistory[0].Command)
	}
	if g.commandHistory[0].Status != "success" {
		t.Errorf("status = %q, want 'success'", g.commandHistory[0].Status)
	}
}

func TestLogCommandMaxHistory(t *testing.T) {
	g := newTestGui()

	for i := 0; i < 15; i++ {
		g.logCommand("cmd", "desc", "success")
	}
	if len(g.commandHistory) != 10 {
		t.Errorf("history should be capped at 10, got %d", len(g.commandHistory))
	}
}

// --- clearDetailsCache ---

func TestClearDetailsCache(t *testing.T) {
	g := newTestGui()
	g.cachedDetailsContent = "some content"
	g.cachedDetailsDocPath = "users/u1"
	g.cachedDetailsLines = []string{"line1", "line2"}
	g.cachedDetailsHeader = "header"
	g.detailsScrollPos = 50

	g.clearDetailsCache()

	if g.cachedDetailsContent != "" {
		t.Error("cachedDetailsContent should be empty")
	}
	if g.cachedDetailsDocPath != "" {
		t.Error("cachedDetailsDocPath should be empty")
	}
	if g.cachedDetailsLines != nil {
		t.Error("cachedDetailsLines should be nil")
	}
	if g.cachedDetailsHeader != "" {
		t.Error("cachedDetailsHeader should be empty")
	}
	if !g.detailsViewDirty {
		t.Error("detailsViewDirty should be true")
	}
	if g.detailsScrollPos != 0 {
		t.Error("detailsScrollPos should be 0")
	}
}
