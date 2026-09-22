package gui

import (
	"strings"
	"testing"

	"github.com/jesseduffield/gocui"
	"github.com/marjoballabani/lazyfire/pkg/firebase"
)

func newBindingsTestGui() *Gui {
	g := newTestGui()
	g.theme = &Theme{}
	g.bindings = g.getBindings()
	return g
}

func bindingDescription(g *Gui, key any) string {
	if b := g.findBinding(g.currentContext(), key); b != nil {
		return b.Description
	}
	return ""
}

func TestSpaceDependsOnContext(t *testing.T) {
	tests := []struct {
		name       string
		column     string
		tab        string
		selectMode bool
		want       string
	}{
		{"projects", "projects", "collections", false, "Select project"},
		{"collections tab", "collections", "collections", false, "Open collection"},
		{"functions tab", "collections", "functions", false, "Select function"},
		{"storage tab", "collections", "storage", false, "Open bucket / folder"},
		{"auth tab does not open a collection", "collections", "auth", false, "View user in details"},
		{"rules tab has no space action", "collections", "rules", false, ""},
		{"indexes tab has no space action", "collections", "indexes", false, ""},
		{"tree", "tree", "collections", false, "Expand / collapse"},
		{"tree in select mode", "tree", "collections", true, "Fetch selected documents"},
		{"details has no space action", "details", "collections", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newBindingsTestGui()
			g.currentColumn = tt.column
			g.collectionsTab = tt.tab
			g.selectMode = tt.selectMode
			if got := bindingDescription(g, gocui.KeySpace); got != tt.want {
				t.Errorf("space = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEscapePriority(t *testing.T) {
	tests := []struct {
		name  string
		setup func(g *Gui)
		want  string
	}{
		{"details goes back", func(g *Gui) { g.currentColumn = "details" }, "Back to previous panel"},
		{"details clears its filter first", func(g *Gui) {
			g.currentColumn = "details"
			g.detailsFilter = "name"
		}, "Clear filter"},
		{"storage folder goes up", func(g *Gui) {
			g.currentColumn, g.collectionsTab = "collections", "storage"
			g.currentBucket = "b1"
		}, "Go up one level"},
		{"storage clears its filter before going up", func(g *Gui) {
			g.currentColumn, g.collectionsTab = "collections", "storage"
			g.currentBucket = "b1"
			g.storageFilter = "img"
		}, "Clear filter"},
		{"storage bucket list has nothing to go back to", func(g *Gui) {
			g.currentColumn, g.collectionsTab = "collections", "storage"
		}, ""},
		{"tree exits select mode before clearing the filter", func(g *Gui) {
			g.currentColumn = "tree"
			g.selectMode = true
			g.treeFilter = "u"
		}, "Exit select mode"},
		{"a filter on another panel is left alone", func(g *Gui) {
			g.currentColumn = "tree"
			g.projectsFilter = "p1"
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newBindingsTestGui()
			tt.setup(g)
			if got := bindingDescription(g, gocui.KeyEsc); got != tt.want {
				t.Errorf("esc = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnboundKeyInContextIsIgnored(t *testing.T) {
	// Esc is bound on the projects view (to clear a filter) but does nothing
	// without one. The handler must not return an error: gocui would end the
	// main loop with it.
	g := newBindingsTestGui()
	g.currentColumn = "projects"
	if err := g.contextHandler(gocui.KeyEsc)(nil, nil); err != nil {
		t.Errorf("esc without a filter returned %v", err)
	}

	// Space on the rules tab shares the collections view with tabs that bind it
	g.currentColumn = "collections"
	g.collectionsTab = "rules"
	if err := g.contextHandler(gocui.KeySpace)(nil, nil); err != nil {
		t.Errorf("space on rules returned %v", err)
	}
}

func TestPopupsTakeOverKeys(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "tree"

	g.filterInputActive = true
	if ctx := g.currentContext(); ctx != CtxFilter || !isPopupContext(ctx) {
		t.Fatalf("typing a filter should be a popup context, got %q", ctx)
	}
	g.filterInputActive = false

	g.helpOpen = true
	if b := g.findBinding(g.currentContext(), gocui.KeyEsc); b == nil || b.Contexts[0] != CtxMenu {
		t.Error("esc in the keybindings menu should belong to the menu")
	}
	g.helpOpen = false

	g.confirmOpen = true
	g.helpOpen = true
	if g.currentContext() != CtxConfirm {
		t.Error("confirm dialog should win over other popups")
	}
}

func TestEscWhileTypingFilterKeepsPanel(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "details"
	g.currentDocData = map[string]any{"a": 1}

	if err := g.doStartFilter(); err != nil {
		t.Fatal(err)
	}
	if !g.filterInputActive || g.filterInputPanel != CtxDetails {
		t.Fatalf("filter should target details, got active=%v panel=%q", g.filterInputActive, g.filterInputPanel)
	}
	g.filterInputText = "a"
	g.cancelFilterInput()

	if g.filterInputActive {
		t.Error("filter prompt should be closed")
	}
	if g.currentColumn != "details" {
		t.Errorf("esc should stay in details, got %q", g.currentColumn)
	}
	if g.detailsFilter != "" {
		t.Error("cancelled filter should not be kept")
	}
}

func TestCommitFilterKeepsSelection(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "projects"
	_ = g.doStartFilter()
	g.filterInputText = "Project 1" // Project 1, 10, 11, 12
	g.selectedProjectIndex = 2
	g.commitFilter()

	if g.projectsFilter != "Project 1" {
		t.Errorf("filter = %q, want %q", g.projectsFilter, "Project 1")
	}
	if g.selectedProjectIndex != 2 {
		t.Errorf("commit moved the selection to %d", g.selectedProjectIndex)
	}
}

func TestClearFilterKeepsSelectedItem(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "projects"
	g.projectsFilter = "Project 1" // Project 1, 10, 11, 12
	g.selectedProjectIndex = 2     // Project 11

	if err := g.doClearFilter(); err != nil {
		t.Fatal(err)
	}
	if g.projectsFilter != "" {
		t.Error("filter should be cleared")
	}
	if got := g.getFilteredProjects()[g.selectedProjectIndex].ID; got != "p11" {
		t.Errorf("selected %q after clearing, want p11", got)
	}
}

func TestStorageAndAuthFilters(t *testing.T) {
	g := newBindingsTestGui()
	g.storageBuckets = []firebase.StorageBucket{{Name: "prod-assets"}, {Name: "dev-assets"}, {Name: "prod-logs"}}
	g.storageObjects = []firebase.StorageObject{{Name: "a/img.png", DisplayName: "img.png"}, {Name: "a/doc.pdf", DisplayName: "doc.pdf"}}
	g.authUsers = []firebase.AuthUser{{UID: "u1", Email: "alice@example.com"}, {UID: "u2", Email: "bob@example.com"}}

	g.storageFilter = "prod"
	if got := len(g.getFilteredBuckets()); got != 2 {
		t.Errorf("buckets matching prod = %d, want 2", got)
	}
	g.storageFilter = "pdf"
	if got := g.getFilteredObjects(); len(got) != 1 || got[0].DisplayName != "doc.pdf" {
		t.Errorf("objects matching pdf = %v", got)
	}

	// Live text while typing wins over the committed filter
	g.authFilter = "alice"
	g.filterInputActive = true
	g.filterInputPanel = CtxAuth
	g.filterInputText = "bob"
	if got := g.getFilteredAuthUsers(); len(got) != 1 || got[0].UID != "u2" {
		t.Errorf("users matching bob = %v", got)
	}
}

func TestMoveByUsesFilteredLists(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "collections"
	g.collectionsTab = "auth"
	g.authUsers = []firebase.AuthUser{{UID: "u1", Email: "a@x"}, {UID: "u2", Email: "b@y"}, {UID: "u3", Email: "c@x"}}
	g.authFilter = "@x"

	_ = g.moveBy(100)
	if g.selectedAuthIdx != 1 {
		t.Errorf("move past the end = %d, want last filtered index 1", g.selectedAuthIdx)
	}
	_ = g.moveBy(-100)
	if g.selectedAuthIdx != 0 {
		t.Errorf("move past the start = %d, want 0", g.selectedAuthIdx)
	}
}

func TestDetailsCursorStaysInContent(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "details"
	g.detailsLineCount = 50
	g.detailsViewHeight = 10

	_ = g.moveToBottom()
	if g.detailsCursor != 49 || g.detailsScrollPos != 40 {
		t.Errorf("bottom: cursor=%d scroll=%d, want 49/40", g.detailsCursor, g.detailsScrollPos)
	}
	_ = g.moveBy(1)
	if g.detailsCursor != 49 {
		t.Errorf("moved past the last line: %d", g.detailsCursor)
	}
	_ = g.moveBy(-20)
	if g.detailsCursor != 29 || g.detailsScrollPos != 20 {
		t.Errorf("page up: cursor=%d scroll=%d, want 29/20", g.detailsCursor, g.detailsScrollPos)
	}
	_ = g.moveToTop()
	if g.detailsCursor != 0 || g.detailsScrollPos != 0 {
		t.Errorf("top: cursor=%d scroll=%d, want 0/0", g.detailsCursor, g.detailsScrollPos)
	}

	g.scrollDetails(100)
	if g.detailsScrollPos != 40 || g.detailsCursor != 40 {
		t.Errorf("wheel: scroll=%d cursor=%d, want 40/40", g.detailsScrollPos, g.detailsCursor)
	}
}

func TestBackFromDetailsAfterRepeatedFocus(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "tree"
	_ = g.setFocus("details")
	_ = g.setFocus("details") // e.g. clicking details while already there

	if err := g.doBackFromDetails(); err != nil {
		t.Fatal(err)
	}
	if g.currentColumn != "tree" {
		t.Errorf("back went to %q, want tree", g.currentColumn)
	}
}

func TestNumberKeysWorkFromDetails(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "details"
	b := g.findBinding(g.currentContext(), '1')
	if b != nil {
		t.Fatal("number keys should be global bindings")
	}
	for _, binding := range g.bindings {
		if binding.Contexts == nil && binding.hasKey('1') {
			_ = g.runBinding(binding)
		}
	}
	if g.currentColumn != "projects" {
		t.Errorf("1 from details focused %q, want projects", g.currentColumn)
	}
}

func TestDisabledBindingShowsReason(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "details"
	b := g.findBinding(CtxDetails, 't')
	if b == nil {
		t.Fatal("t should be bound in details")
	}

	_ = g.runBinding(b)
	if g.compactJSON {
		t.Error("handler ran without a document")
	}
	if g.toastText != "No document open" || !g.toastIsError {
		t.Errorf("toast = %q (error=%v)", g.toastText, g.toastIsError)
	}
}

func TestClearQueryResultsKey(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "tree"
	b := g.findBinding(CtxTree, 'Q')
	if b == nil {
		t.Fatal("Q should be bound in the tree")
	}
	if b.disabledReason() == "" {
		t.Error("Q should be disabled without query results")
	}
	g.queryResultMode = true
	if b.disabledReason() != "" {
		t.Error("Q should be enabled with query results")
	}
}

func TestResetProjectStateClearsCaches(t *testing.T) {
	g := newBindingsTestGui()
	g.docCache["users/u1"] = map[string]any{"name": "from old project"}
	g.collectionCache["users"] = []string{"users/u1"}
	g.treeFilter = "u1"
	g.queryResultMode = true

	g.resetProjectState("other")

	if g.currentProject != "other" {
		t.Errorf("currentProject = %q", g.currentProject)
	}
	if len(g.docCache) != 0 || len(g.collectionCache) != 0 {
		t.Error("document caches are keyed by path and must not leak across projects")
	}
	if g.treeFilter != "" || g.queryResultMode || g.treeNodes != nil {
		t.Error("tree state should be reset")
	}
}

func TestQueryTargetFollowsFocusedPanel(t *testing.T) {
	g := newBindingsTestGui()
	g.currentCollection = "users"

	g.currentColumn = "collections"
	g.selectedCollectionIdx = 1
	if got, idx := g.queryTarget(); got != "orders" || idx != -1 {
		t.Errorf("collections panel: %q %d, want highlighted orders -1", got, idx)
	}

	g.currentColumn = "tree"
	g.selectedTreeIdx = 3 // users/u3/orders
	if got, idx := g.queryTarget(); got != "users/u3/orders" || idx != 3 {
		t.Errorf("tree collection node: %q %d", got, idx)
	}

	g.selectedTreeIdx = 0 // a document
	if got, _ := g.queryTarget(); got != "users" {
		t.Errorf("tree document: %q, want the open collection", got)
	}
}

func TestOptionsBarFollowsContext(t *testing.T) {
	tests := []struct {
		name    string
		column  string
		tab     string
		want    []string
		notWant []string
	}{
		{"projects", "projects", "collections", []string{"select", "filter", "scan", "help"}, []string{"tabs", "copy"}},
		{"collections", "collections", "collections", []string{"open", "query", "tabs"}, []string{"copy"}},
		{"storage bucket list", "collections", "storage", []string{"open", "tabs"}, []string{"back"}},
		{"tree", "tree", "collections", []string{"expand", "open", "filter", "query", "select", "copy"}, []string{"tabs", "clear query"}},
		{"details without a document", "details", "collections", []string{"back", "wrap"}, []string{"compact", "tabs"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newBindingsTestGui()
			g.currentColumn = tt.column
			g.collectionsTab = tt.tab
			g.currentProject = "p1"
			g.currentCollection = "users"
			g.storageBuckets = []firebase.StorageBucket{{Name: "b1"}}
			bar := stripANSI(g.optionsBarHints(500))
			for _, w := range tt.want {
				if !strings.Contains(bar, " "+w) {
					t.Errorf("bar %q should contain %q", bar, w)
				}
			}
			for _, w := range tt.notWant {
				if strings.Contains(bar, " "+w+"  ") {
					t.Errorf("bar %q should not contain %q", bar, w)
				}
			}
		})
	}
}

func TestOptionsBarKeepsHelpWhenNarrow(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "tree"
	bar := stripANSI(g.optionsBarHints(30))
	if !strings.HasSuffix(bar, "? help") {
		t.Errorf("narrow bar %q should end with ? help", bar)
	}
	if !strings.Contains(bar, "…") {
		t.Errorf("narrow bar %q should mark cut hints", bar)
	}
	if len([]rune(bar)) > 30 {
		t.Errorf("bar is %d wide, limit 30", len([]rune(bar)))
	}
}

func TestHelpMenuListsContextBindings(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "details"
	g.buildHelpPopup()
	p := g.helpPopup

	if !p.Items[0].IsHeader || p.Items[0].Label != "Details" {
		t.Errorf("first section = %q, want Details", p.Items[0].Label)
	}
	var labels []string
	for _, item := range p.Items {
		labels = append(labels, item.Label)
	}
	all := strings.Join(labels, "|")
	for _, want := range []string{"Toggle compact JSON", "Navigation", "Global", "Keybindings"} {
		if !strings.Contains(all, want) {
			t.Errorf("menu should list %q", want)
		}
	}
	if strings.Contains(all, "Select project") {
		t.Error("menu in details should not list projects actions")
	}

	p.SetFilter("wrap")
	visible := p.Visible()
	if len(visible) != 2 || !visible[0].IsHeader || visible[1].Label != "Toggle word wrap" {
		t.Errorf("filtered menu = %+v", visible)
	}
	if item := p.GetSelectedItem(); item == nil || item.Label != "Toggle word wrap" {
		t.Error("filtering should select the first match")
	}
}

func TestKeyLabels(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "collections"
	g.collectionsTab = "storage"
	g.currentBucket = "b"
	if got := g.findBinding(g.currentContext(), gocui.KeyBackspace).keyLabel(); got != "esc/backspace" {
		t.Errorf("label = %q", got)
	}
	if got := g.findBinding(CtxTree, 'j').keyLabel(); got != "j/↓" {
		t.Errorf("label = %q", got)
	}
}

func TestExtractJSONLineValue(t *testing.T) {
	tests := []struct {
		line        string
		lineNumbers bool
		timestamps  bool
		want        string
	}{
		{`  "name": "Alice",`, false, false, "Alice"},
		{`  "count": 42,`, false, false, "42"},
		{`  "active": true`, false, false, "true"},
		{`  "quote": "say \"hi\""`, false, false, `say "hi"`},
		{`  "address": {`, false, false, ""},
		{`  },`, false, false, ""},
		{`    "tag",`, false, false, "tag"},
		{`  "empty": {},`, false, false, "{}"},
		{` 12   "name": "Bob",`, true, false, "Bob"},
		{`  "at": "2024-01-15T10:30:00Z",  // Jan 15, 2024 10:30:00 AM`, false, true, "2024-01-15T10:30:00Z"},
	}
	for _, tt := range tests {
		if got := extractJSONLineValue(tt.line, tt.lineNumbers, tt.timestamps); got != tt.want {
			t.Errorf("extractJSONLineValue(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestExtractInfoLineValue(t *testing.T) {
	tests := []struct{ line, want string }{
		{" Name:        myFunction", "myFunction"},
		{" URL:         https://example.com/fn", "https://example.com/fn"},
		{" Last Sign-In:  2024-01-01", "2024-01-01"},
		{"2024-01-01T00:00:00Z INFO    started", "2024-01-01T00:00:00Z INFO    started"},
	}
	for _, tt := range tests {
		if got := extractInfoLineValue(tt.line); got != tt.want {
			t.Errorf("extractInfoLineValue(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestCollectionsTabWindow(t *testing.T) {
	tests := []struct {
		tab       string
		wantStart int
		wantIdx   int
	}{
		{"collections", 0, 0},
		{"functions", 0, 1},
		{"storage", 1, 1},
		{"indexes", 3, 2},
	}
	for _, tt := range tests {
		tabs, idx := collectionsTabWindow(tt.tab)
		if start := collectionsTabWindowStart(tt.tab); start != tt.wantStart {
			t.Errorf("%s: start = %d, want %d", tt.tab, start, tt.wantStart)
		}
		if idx != tt.wantIdx || !strings.Contains(strings.ToLower(tabs[idx]), tt.tab) {
			t.Errorf("%s: tabs %v active %d", tt.tab, tabs, idx)
		}
	}
}

func menuItem(t *testing.T, g *Gui, label string) int {
	t.Helper()
	for i, item := range g.helpPopup.Visible() {
		if item.Label == label {
			return i
		}
	}
	t.Fatalf("menu has no %q", label)
	return -1
}

func TestMenuJudgesBindingsForItsPanel(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "tree"
	g.selectedTreeIdx = 3 // the expanded "orders" collection node
	_ = g.doToggleHelp()

	idx := menuItem(t, g, "Expand / collapse")
	if reason := g.helpPopup.Visible()[idx].Binding.disabledReason(); reason != "" {
		t.Fatalf("shown as disabled in the menu: %q", reason)
	}

	g.helpPopup.SelectedIdx = idx
	if err := g.helpExecute(); err != nil {
		t.Fatal(err)
	}
	if g.helpOpen {
		t.Error("menu should close after running an item")
	}
	if len(g.treeNodes) != 4 || g.treeNodes[3].Expanded {
		t.Error("running the item from the menu should collapse the node")
	}
}

func TestMenuFilterEscAndEnter(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "tree"
	_ = g.doToggleHelp()

	// Esc while typing drops the filter
	_ = g.doStartFilter()
	g.filterInputText = "collapse"
	g.resetFilterSelection(CtxMenu)
	g.cancelFilterInput()
	if f := g.helpPopup.Filter(); f != "" {
		t.Errorf("esc kept the menu filter %q", f)
	}

	// Enter keeps the item picked with the arrow keys
	_ = g.doStartFilter()
	g.filterInputText = "o"
	g.resetFilterSelection(CtxMenu)
	g.moveFilterTarget(1)
	picked := g.helpPopup.GetSelectedItem().Label
	g.commitFilter()
	if got := g.helpPopup.GetSelectedItem().Label; got != picked {
		t.Errorf("enter moved the selection from %q to %q", picked, got)
	}
}

func TestProjectSwitchDropsStaleLoads(t *testing.T) {
	g := newBindingsTestGui()
	gen := g.projectGen
	g.treeLoading = true
	g.functionsLoading = true

	g.resetProjectState("p3")

	if g.projectGen == gen {
		t.Error("switching projects should invalidate loads in flight")
	}
	if g.treeLoading || g.functionsLoading {
		t.Error("loading flags of dropped loads should be cleared")
	}
}

func runGlobalKey(g *Gui, key any) {
	for _, b := range g.bindings {
		if b.Contexts == nil && b.hasKey(key) {
			_ = g.globalHandler(b)(nil, nil)
			return
		}
	}
}

func TestTabCyclesSidePanelsLikeLazygit(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "projects"

	want := []string{"databases", "collections", "tree", "projects"}
	for _, column := range want {
		_ = g.contextHandler(gocui.KeyTab)(nil, nil)
		if g.currentColumn != column {
			t.Fatalf("tab went to %q, want %q", g.currentColumn, column)
		}
	}
	_ = g.contextHandler(gocui.KeyBacktab)(nil, nil)
	if g.currentColumn != "tree" {
		t.Errorf("shift+tab from projects went to %q, want tree", g.currentColumn)
	}

	// In details, tab goes back to the panel details was opened from
	_ = g.setFocus("details")
	_ = g.contextHandler(gocui.KeyTab)(nil, nil)
	if g.currentColumn != "tree" {
		t.Errorf("tab in details went to %q, want tree", g.currentColumn)
	}
}

func TestNumberKeysFollowScreenOrder(t *testing.T) {
	g := newBindingsTestGui()
	g.databases = []firebase.Database{{ID: firebase.DefaultDatabase}} // skip loading
	g.currentColumn = "tree"

	for key, want := range map[rune]string{'1': "projects", '2': "databases", '3': "collections", '4': "tree"} {
		runGlobalKey(g, key)
		if g.currentColumn != want {
			t.Errorf("%c focused %q, want %q", key, g.currentColumn, want)
		}
	}

	g.currentColumn = "collections"
	runGlobalKey(g, '0')
	if g.currentColumn != "details" || g.previousColumn != "collections" {
		t.Errorf("0 focused %q (previous %q), want details from collections", g.currentColumn, g.previousColumn)
	}
}

func TestDatabaseSwitchResetsOnlyFirestoreData(t *testing.T) {
	g := newBindingsTestGui()
	g.functions = []firebase.CloudFunction{{Name: "f1"}}
	g.authUsers = []firebase.AuthUser{{UID: "u1"}}
	g.docCache["users/u1"] = map[string]any{"from": "other database"}
	g.firestoreIndexes = []firebase.FirestoreIndex{{CollectionGroup: "users"}}
	projectGen, databaseGen := g.projectGen, g.databaseGen

	g.resetDatabaseState()

	if g.databaseGen == databaseGen || g.projectGen != projectGen {
		t.Error("a database switch should only drop Firestore loads in flight")
	}
	if g.collections != nil || g.treeNodes != nil || len(g.docCache) != 0 || g.firestoreIndexes != nil {
		t.Error("Firestore data of the old database should be cleared")
	}
	if len(g.functions) != 1 || len(g.authUsers) != 1 {
		t.Error("project-wide data (functions, auth) should be kept")
	}
}

func TestProjectSwitchGoesBackToDefaultDatabase(t *testing.T) {
	g := newBindingsTestGui()
	g.databases = []firebase.Database{{ID: firebase.DefaultDatabase}, {ID: "analytics"}}
	g.currentDatabase = "analytics"

	g.resetProjectState("p2")

	if g.currentDatabase != firebase.DefaultDatabase || g.databases != nil {
		t.Errorf("database = %q, list = %v", g.currentDatabase, g.databases)
	}
}

func TestDatabasesPanel(t *testing.T) {
	g := newBindingsTestGui()
	g.currentColumn = "databases"
	g.databases = []firebase.Database{
		{ID: firebase.DefaultDatabase, LocationID: "nam5", Type: "FIRESTORE_NATIVE"},
		{ID: "analytics", LocationID: "eur3", Type: "FIRESTORE_NATIVE"},
		{ID: "legacy", LocationID: "us-east1", Type: "DATASTORE_MODE"},
	}

	if got := bindingDescription(g, gocui.KeySpace); got != "Use database" {
		t.Errorf("space = %q", got)
	}

	g.databasesFilter = "eur"
	if got := g.getFilteredDatabases(); len(got) != 1 || got[0].ID != "analytics" {
		t.Errorf("filter by location = %v", got)
	}
	g.databasesFilter = ""

	g.selectedDatabaseIdx = 2
	if reason := g.databaseDisabledReason(); reason == "" {
		t.Error("datastore mode databases should not be selectable")
	}
	g.selectedDatabaseIdx = 1
	if reason := g.databaseDisabledReason(); reason != "" {
		t.Errorf("native database disabled: %q", reason)
	}
}
