package gui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/jesseduffield/gocui"
	"github.com/marjoballabani/lazyfire/pkg/firebase"
)

// Actions - handler functions. Which panel/tab they run in is decided by the
// binding contexts in keybindings.go; state checks live in GetDisabledReason.

// doQuit exits the application
func (g *Gui) doQuit() error {
	return gocui.ErrQuit
}

// doBackFromDetails returns focus to the panel details was opened from
func (g *Gui) doBackFromDetails() error {
	if g.scanResults != nil {
		g.scanResults = nil
		g.clearDetailsCache()
	}
	target := g.previousColumn
	if target == "" || target == "details" {
		target = "tree"
	}
	return g.setFocus(target)
}

// doToggleHelp toggles the keybindings menu
func (g *Gui) doToggleHelp() error {
	if g.helpOpen {
		g.helpOpen = false
		g.helpPopup = nil
	} else {
		g.buildHelpPopup()
		g.helpOpen = true
	}
	return g.relayout()
}

// doToggleModal toggles the command log modal
func (g *Gui) doToggleModal() error {
	g.modalOpen = !g.modalOpen
	return g.relayout()
}

func (g *Gui) helpMoveUp() error {
	if g.helpPopup != nil {
		g.helpPopup.MoveUp()
	}
	return nil
}

func (g *Gui) helpMoveDown() error {
	if g.helpPopup != nil {
		g.helpPopup.MoveDown()
	}
	return nil
}

// helpExecute closes the menu and runs the selected binding. A disabled
// binding keeps the menu open and explains why.
func (g *Gui) helpExecute() error {
	if g.helpPopup == nil {
		return nil
	}
	item := g.helpPopup.GetSelectedItem()
	if item == nil || item.Binding == nil {
		return nil
	}
	if reason := item.Binding.disabledReason(); reason != "" {
		g.toast(reason, true)
		return nil
	}
	g.helpOpen = false
	g.helpPopup = nil
	if err := item.Binding.Handler(); err != nil {
		return err
	}
	return g.relayout()
}

// helpEscape clears the menu filter, or closes the menu
func (g *Gui) helpEscape() error {
	if g.helpPopup != nil && g.helpPopup.Filter() != "" {
		g.helpPopup.SetFilter("")
		return nil
	}
	g.helpOpen = false
	g.helpPopup = nil
	return g.relayout()
}

func (g *Gui) doConfirmAccept() error {
	cb := g.confirmCallback
	g.confirmOpen = false
	g.confirmCallback = nil
	if cb != nil {
		cb()
	}
	return g.relayout()
}

func (g *Gui) doConfirmCancel() error {
	g.confirmOpen = false
	g.confirmCallback = nil
	return g.relayout()
}

// sidePanels lists the left panels top to bottom
var sidePanels = []string{"projects", "databases", "collections", "tree"}

// doColumnLeft switches to the side panel above, wrapping around
func (g *Gui) doColumnLeft() error {
	return g.cycleSidePanel(-1)
}

// doColumnRight switches to the side panel below, wrapping around
func (g *Gui) doColumnRight() error {
	return g.cycleSidePanel(1)
}

func (g *Gui) cycleSidePanel(dir int) error {
	for i, panel := range sidePanels {
		if panel == g.currentColumn {
			return g.setFocus(sidePanels[(i+dir+len(sidePanels))%len(sidePanels)])
		}
	}
	return nil
}

// listState returns the selection index and item count of a list context,
// or nil for contexts without a selectable list
func (g *Gui) listState(ctx Context) (*int, int) {
	switch ctx {
	case CtxProjects:
		return &g.selectedProjectIndex, len(g.getFilteredProjects())
	case CtxDatabases:
		return &g.selectedDatabaseIdx, len(g.getFilteredDatabases())
	case CtxCollections:
		return &g.selectedCollectionIdx, len(g.getFilteredCollections())
	case CtxFunctions:
		return &g.selectedFunctionIdx, len(g.getFilteredFunctions())
	case CtxStorage:
		if g.currentBucket == "" {
			return &g.selectedBucketIdx, len(g.getFilteredBuckets())
		}
		return &g.selectedObjectIdx, len(g.getFilteredObjects())
	case CtxAuth:
		return &g.selectedAuthIdx, len(g.getFilteredAuthUsers())
	case CtxTree:
		return &g.selectedTreeIdx, len(g.getFilteredTreeNodes())
	}
	return nil, 0
}

// setSelection selects item idx of a list context
func (g *Gui) setSelection(ctx Context, idx int) {
	sel, _ := g.listState(ctx)
	if sel == nil || *sel == idx {
		return
	}
	*sel = idx
	switch ctx {
	case CtxProjects:
		g.currentProjectInfo = nil
	case CtxTree:
		if g.selectMode {
			g.updateSelectRange()
		}
	}
}

// moveBy moves the focused panel's selection by delta items. Text panels
// (details, rules, indexes) move their cursor or scroll instead.
func (g *Gui) moveBy(delta int) error {
	ctx := g.currentContext()
	switch ctx {
	case CtxDetails:
		if delta > 1 || delta < -1 {
			// Page moves scroll along so the cursor keeps its row on screen
			g.detailsScrollPos = max(0, min(g.detailsScrollPos+delta, g.maxDetailsScroll()))
		}
		g.setDetailsCursor(g.detailsCursor + delta)
	case CtxRules, CtxIndexes:
		// Clamped to the content height by Layout
		g.collectionsScrollPos = max(0, g.collectionsScrollPos+delta)
	default:
		if sel, n := g.listState(ctx); sel != nil && n > 0 {
			g.setSelection(ctx, max(0, min(*sel+delta, n-1)))
		}
	}
	return nil
}

func (g *Gui) moveToTop() error {
	switch ctx := g.currentContext(); ctx {
	case CtxDetails:
		g.setDetailsCursor(0)
	case CtxRules, CtxIndexes:
		g.collectionsScrollPos = 0
	default:
		if sel, n := g.listState(ctx); sel != nil && n > 0 {
			g.setSelection(ctx, 0)
		}
	}
	return nil
}

func (g *Gui) moveToBottom() error {
	switch ctx := g.currentContext(); ctx {
	case CtxDetails:
		g.setDetailsCursor(g.detailsLineCount - 1)
	case CtxRules, CtxIndexes:
		g.collectionsScrollPos = 1 << 30 // clamped by Layout
	default:
		if sel, n := g.listState(ctx); sel != nil && n > 0 {
			g.setSelection(ctx, n-1)
		}
	}
	return nil
}

// pageSize is the visible height of the focused panel
func (g *Gui) pageSize() int {
	if g.g != nil {
		if v, err := g.g.View(g.contextView(g.currentContext())); err == nil && v.InnerHeight() > 0 {
			return v.InnerHeight()
		}
	}
	return 10
}

func (g *Gui) halfPageSize() int {
	return max(1, g.pageSize()/2)
}

// setDetailsCursor moves the details cursor to line and scrolls it into view
func (g *Gui) setDetailsCursor(line int) {
	g.detailsCursor = max(0, min(line, g.detailsLineCount-1))
	if g.detailsCursor < g.detailsScrollPos {
		g.detailsScrollPos = g.detailsCursor
	} else if h := g.detailsViewHeight; h > 0 && g.detailsCursor >= g.detailsScrollPos+h {
		g.detailsScrollPos = g.detailsCursor - h + 1
	}
}

// scrollDetails scrolls details by delta lines, dragging the cursor along
// when it would leave the screen
func (g *Gui) scrollDetails(delta int) {
	g.detailsScrollPos = max(0, min(g.detailsScrollPos+delta, g.maxDetailsScroll()))
	if g.detailsCursor < g.detailsScrollPos {
		g.detailsCursor = g.detailsScrollPos
	}
	if h := g.detailsViewHeight; h > 0 && g.detailsCursor >= g.detailsScrollPos+h {
		g.detailsCursor = g.detailsScrollPos + h - 1
	}
}

func (g *Gui) maxDetailsScroll() int {
	return max(0, g.detailsLineCount-g.detailsViewHeight)
}

// collectionsTabs lists the collections panel tabs in display order
var collectionsTabs = []string{"collections", "functions", "storage", "auth", "rules", "indexes"}

// doSwitchTabNext cycles to next tab (] key)
func (g *Gui) doSwitchTabNext() error {
	return g.doSwitchTabDir(1)
}

// doSwitchTabPrev cycles to previous tab ([ key)
func (g *Gui) doSwitchTabPrev() error {
	return g.doSwitchTabDir(-1)
}

func (g *Gui) doSwitchTabDir(dir int) error {
	if g.currentColumn == "details" {
		// Details and Logs are the only two tabs, so both directions toggle
		return g.toggleDetailsTab()
	}
	idx := 0
	for i, t := range collectionsTabs {
		if t == g.collectionsTab {
			idx = i
			break
		}
	}
	return g.switchCollectionsTab(collectionsTabs[(idx+dir+len(collectionsTabs))%len(collectionsTabs)])
}

// switchCollectionsTab activates a collections panel tab, loading its data on
// first visit
func (g *Gui) switchCollectionsTab(tab string) error {
	g.collectionsTab = tab
	g.collectionsScrollPos = 0

	// Reset view scroll position when switching tabs
	if g.g != nil {
		if v, err := g.g.View(g.views.collections); err == nil {
			v.SetOrigin(0, 0)
		}
	}

	if tab == "collections" {
		g.stopLogsRefresh()
	}
	if g.currentProject == "" {
		return nil
	}
	switch tab {
	case "collections":
		if len(g.collections) == 0 && !g.collectionsLoading {
			g.loadCollections()
		}
	case "functions":
		if len(g.functions) == 0 && !g.functionsLoading {
			g.loadFunctions()
		}
	case "storage":
		if len(g.storageBuckets) == 0 && !g.storageLoading {
			g.loadStorageBuckets()
		}
	case "auth":
		if len(g.authUsers) == 0 && !g.authLoading {
			g.loadAuthUsers()
		}
	case "rules":
		if g.firestoreRules == nil && !g.rulesLoading {
			g.loadFirestoreRules()
		}
	case "indexes":
		if g.firestoreIndexes == nil && !g.indexesLoading {
			g.loadFirestoreIndexes()
		}
	}
	return nil
}

// toggleDetailsTab switches the function details panel between Details and Logs
func (g *Gui) toggleDetailsTab() error {
	if g.detailsTab == "details" {
		g.detailsTab = "logs"
		if g.currentFunction != nil && len(g.functionLogs) == 0 && !g.logsLoading {
			g.loadFunctionLogs()
		}
	} else {
		g.detailsTab = "details"
	}
	g.detailsCursor = 0
	g.detailsScrollPos = 0
	return nil
}

// doCopyJSON copies current document to clipboard
func (g *Gui) doCopyJSON() error {
	if g.currentColumn == "details" && g.detailsSource() == "scan" {
		return g.copyScanReport()
	}
	return g.copyJSONAction()
}

// doClearCache clears all document and collection caches
func (g *Gui) doClearCache() error {
	docCount := len(g.docCache)
	g.docCache = make(map[string]map[string]any)
	g.statsCache = make(map[string]*firebase.DocStats)
	g.collectionCache = make(map[string][]string)
	g.compositeIndexCache = make(map[string]*bool)
	g.clearDetailsCache()
	g.logCommand("cache", fmt.Sprintf("Cleared %d cached documents", docCount), "success")
	return g.Layout(g.g)
}

// doShowCacheStats shows cache statistics in the command log
func (g *Gui) doShowCacheStats() error {
	docCount := len(g.docCache)
	collCount := len(g.collectionCache)
	statsCount := len(g.statsCache)

	// Estimate memory usage
	totalDocs := 0
	for _, paths := range g.collectionCache {
		totalDocs += len(paths)
	}

	g.logCommand("cache",
		fmt.Sprintf("Docs: %d cached, Collections: %d cached (%d paths), Stats: %d cached",
			docCount, collCount, totalDocs, statsCount),
		"success")
	return g.Layout(g.g)
}

// doToggleTimestamps toggles human-readable timestamp annotations
func (g *Gui) doToggleTimestamps() error {
	g.humanizeTimestamps = !g.humanizeTimestamps
	g.clearDetailsCache()
	if g.humanizeTimestamps {
		g.logCommand("view", "Timestamps humanized", "success")
	} else {
		g.logCommand("view", "Timestamps raw", "success")
	}
	return g.Layout(g.g)
}

// doExportCachedDocs exports all cached documents to a single JSON file
func (g *Gui) doExportCachedDocs() error {
	if len(g.docCache) == 0 {
		g.logCommand("export", "No cached documents to export", "error")
		return nil
	}

	data, err := json.MarshalIndent(g.docCache, "", "  ")
	if err != nil {
		g.logCommand("export", fmt.Sprintf("Failed: %v", err), "error")
		return nil
	}

	home, _ := os.UserHomeDir()
	projectName := g.currentProject
	if projectName == "" {
		projectName = "lazyfire"
	}
	filename := fmt.Sprintf("lazyfire-export_%s.json", projectName)
	fullPath := filepath.Join(home, "Downloads", filename)

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		g.logCommand("export", fmt.Sprintf("Failed: %v", err), "error")
		return nil
	}

	g.logCommand("export", fmt.Sprintf("Exported %d docs to %s", len(g.docCache), fullPath), "success")
	return nil
}

// doToggleWrap toggles word wrap in the details panel
func (g *Gui) doToggleWrap() error {
	if g.currentColumn != "details" {
		return nil
	}
	v, err := g.g.View("details")
	if err != nil {
		return nil
	}
	v.Wrap = !v.Wrap
	if v.Wrap {
		g.logCommand("view", "Word wrap on", "success")
	} else {
		g.logCommand("view", "Word wrap off", "success")
	}
	return g.Layout(g.g)
}

// doToggleCompactJSON toggles between compact and pretty JSON view
func (g *Gui) doToggleCompactJSON() error {
	if g.currentColumn != "details" || g.currentDocData == nil {
		return nil
	}
	g.compactJSON = !g.compactJSON
	g.clearDetailsCache()
	if g.compactJSON {
		g.logCommand("view", "Compact JSON", "success")
	} else {
		g.logCommand("view", "Pretty JSON", "success")
	}
	return g.Layout(g.g)
}

// doCollapseAll collapses all expanded tree nodes
func (g *Gui) doCollapseAll() error {
	if g.currentColumn != "tree" || len(g.treeNodes) == 0 {
		return nil
	}
	// Remove all children - keep only depth-0 nodes
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
	g.logCommand("tree", fmt.Sprintf("Collapsed all (%d nodes)", len(topLevel)), "success")
	return g.Layout(g.g)
}

// doCopyPath copies the current document/node path to clipboard
func (g *Gui) doCopyPath() error {
	path := g.currentDocPath
	if g.currentColumn == "tree" {
		node, ok := g.selectedTreeNode()
		if !ok {
			return nil
		}
		path = node.Path
	}
	if err := copyToClipboard(path); err != nil {
		g.logCommand("path", fmt.Sprintf("Failed: %v", err), "error")
		return nil
	}
	g.logCommand("path", fmt.Sprintf("Copied: %s", path), "success")
	return nil
}

// copyToClipboard puts text on the system clipboard
func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// doSaveJSON saves current document to file
func (g *Gui) doSaveJSON() error {
	if g.currentColumn == "details" && g.detailsSource() == "scan" {
		return g.saveScanReport()
	}
	return g.saveJSONAction()
}

// doEditInEditor opens current document in external editor
func (g *Gui) doEditInEditor() error {
	if g.currentColumn != "details" {
		return nil
	}

	if g.currentDocData == nil {
		g.logCommand("e", "No document loaded", "error")
		return nil
	}

	g.logCommand("e", "Opening editor...", "running")

	// Get editor from environment, try nvim then vim as fallback
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Check if nvim is available, otherwise use vim
		if _, err := exec.LookPath("nvim"); err == nil {
			editor = "nvim"
		} else {
			editor = "vim"
		}
	}

	// Format JSON
	jsonData, err := json.MarshalIndent(g.currentDocData, "", "  ")
	if err != nil {
		g.logCommand("e", fmt.Sprintf("JSON error: %v", err), "error")
		return nil
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "lazyfire-*.json")
	if err != nil {
		g.logCommand("e", fmt.Sprintf("Temp file error: %v", err), "error")
		return nil
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(jsonData); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		g.logCommand("e", fmt.Sprintf("Write error: %v", err), "error")
		return nil
	}
	tmpFile.Close()

	// Run editor synchronously (blocks until editor closes)
	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	_ = g.g.Suspend()
	err = cmd.Run()
	_ = g.g.Resume()

	// Clean up temp file
	os.Remove(tmpPath)

	if err != nil {
		g.logCommand("e", fmt.Sprintf("Editor error: %v", err), "error")
	} else {
		g.logCommand("e", fmt.Sprintf("Opened in %s", editor), "success")
	}

	return g.Layout(g.g)
}

// doRefresh reloads whatever the focused panel (or tab) shows
func (g *Gui) doRefresh() error {
	switch g.currentContext() {
	case CtxProjects:
		g.loadProjects()
	case CtxDatabases:
		g.loadDatabases()
	case CtxCollections:
		g.loadCollections()
	case CtxFunctions:
		g.loadFunctions()
	case CtxStorage:
		if g.currentBucket == "" {
			g.loadStorageBuckets()
		} else {
			g.loadStorageObjects(g.selectedItemKey(CtxStorage))
		}
	case CtxAuth:
		g.loadAuthUsers()
	case CtxRules:
		g.loadFirestoreRules()
	case CtxIndexes:
		g.loadFirestoreIndexes()
	case CtxTree:
		if g.queryResultMode {
			g.runQuery(g.lastQueryCollection, -1, g.lastQueryOptions)
			return nil
		}
		g.reloadTree()
	case CtxDetails:
		switch g.detailsSource() {
		case "function":
			g.logCommand("r", "Refreshing logs...", "running")
			g.loadFunctionLogs()
		case "document":
			g.refetchDocument()
		}
	}
	return nil
}

// reloadTree reloads the open collection's documents into the tree
func (g *Gui) reloadTree() {
	gen := g.databaseGen
	collection := g.currentCollection
	g.logCommand("r", "Refreshing documents...", "running")
	g.treeLoading = true
	go func() {
		docs, err := g.firebaseClient.ListDocuments(collection, 50)
		g.g.Update(func(gui *gocui.Gui) error {
			if gen != g.databaseGen {
				return nil // loaded for another project or database
			}
			g.treeLoading = false
			if err != nil {
				g.logCommand("r", fmt.Sprintf("Failed: %v", err), "error")
				return nil
			}
			g.queryResultMode = false
			g.setTreeDocuments(collection, docs)
			g.logCommand("r", fmt.Sprintf("Loaded %d documents", len(docs)), "success")
			return nil
		})
	}()
}

// doClearQueryResults replaces query results with the collection's documents
func (g *Gui) doClearQueryResults() error {
	g.queryResultMode = false
	if g.currentCollection == "" {
		g.treeNodes = nil
		g.selectedTreeIdx = 0
		return nil
	}
	g.reloadTree()
	return nil
}

// Mouse handlers

// onPanelClick focuses the clicked panel and selects the clicked row. A
// double click also opens the row, like enter.
func (g *Gui) onPanelClick(column string, opts gocui.ViewMouseBindingOpts) error {
	if g.closeDismissablePopup() || isPopupContext(g.currentContext()) {
		return nil
	}
	// Collapsed panels only show the active project or database, so a click
	// there just expands them
	wasCollapsed := (column == "projects" || column == "databases") && g.currentColumn != column
	if err := g.setFocus(column); err != nil {
		return err
	}
	if wasCollapsed {
		return nil
	}

	ctx := g.currentContext()
	if ctx == CtxDetails {
		g.setDetailsCursor(opts.Y)
	} else if sel, n := g.listState(ctx); sel != nil && opts.Y >= 0 && opts.Y < n {
		g.setSelection(ctx, opts.Y)
	} else {
		return nil
	}

	if opts.IsDoubleClick {
		if b := g.findBinding(ctx, gocui.KeyEnter); b != nil {
			return g.runBinding(b)
		}
	}
	return nil
}

// onPanelWheel scrolls a panel under the mouse without focusing it
func (g *Gui) onPanelWheel(column string, dir int) error {
	if isPopupContext(g.currentContext()) {
		return nil
	}
	if (column == "projects" || column == "databases") && g.currentColumn != column {
		return nil // collapsed
	}
	ctx := g.panelContext(column)
	switch ctx {
	case CtxDetails:
		g.scrollDetails(3 * dir)
	case CtxRules, CtxIndexes:
		g.collectionsScrollPos = max(0, g.collectionsScrollPos+3*dir)
	default:
		if sel, n := g.listState(ctx); sel != nil && n > 0 {
			g.setSelection(ctx, max(0, min(*sel+dir, n-1)))
		}
	}
	return nil
}

// closeDismissablePopup closes the keybindings menu or the command log, which
// a click outside of them dismisses. Reports whether it closed anything.
func (g *Gui) closeDismissablePopup() bool {
	if g.filterInputActive || g.confirmOpen || g.queryModalOpen || (!g.helpOpen && !g.modalOpen) {
		return false
	}
	g.helpOpen = false
	g.helpPopup = nil
	g.modalOpen = false
	_ = g.relayout()
	return true
}

func (g *Gui) onOutsideClick(gocui.ViewMouseBindingOpts) error {
	g.closeDismissablePopup()
	return nil
}

// onHelpClick runs the clicked keybindings menu item
func (g *Gui) onHelpClick(opts gocui.ViewMouseBindingOpts) error {
	if g.helpPopup == nil || g.currentContext() != CtxMenu {
		return nil
	}
	visible := g.helpPopup.Visible()
	if opts.Y < 0 || opts.Y >= len(visible) || visible[opts.Y].IsHeader {
		return nil
	}
	g.helpPopup.SelectedIdx = opts.Y
	return g.helpExecute()
}

func (g *Gui) onQuerySelectClick(opts gocui.ViewMouseBindingOpts) error {
	if !g.querySelectOpen || opts.Y < 0 || opts.Y >= len(g.querySelectItems) {
		return nil
	}
	g.querySelectIdx = opts.Y
	return g.querySelectConfirm()
}

// onCollectionsTabClick switches to the clicked tab of the collections panel
func (g *Gui) onCollectionsTabClick(tabIdx int) error {
	if isPopupContext(g.currentContext()) {
		return nil
	}
	idx := collectionsTabWindowStart(g.collectionsTab) + tabIdx
	if idx < 0 || idx >= len(collectionsTabs) {
		return nil
	}
	if err := g.switchCollectionsTab(collectionsTabs[idx]); err != nil {
		return err
	}
	return g.setFocus("collections")
}

// onDetailsTabClick switches between the function Details and Logs tabs
func (g *Gui) onDetailsTabClick(tabIdx int) error {
	if isPopupContext(g.currentContext()) || g.detailsSource() != "function" {
		return nil
	}
	if (tabIdx == 1) != (g.detailsTab == "logs") {
		return g.toggleDetailsTab()
	}
	return nil
}

// Select mode functions

// doToggleSelectMode toggles visual selection mode in tree
func (g *Gui) doToggleSelectMode() error {
	if g.currentColumn != "tree" {
		return nil
	}
	if g.selectMode {
		// Exit select mode
		g.selectMode = false
		g.selectedDocs = make(map[int]bool)
	} else {
		// Enter select mode
		g.selectMode = true
		g.selectStartIdx = g.selectedTreeIdx
		g.selectedDocs = make(map[int]bool)
		// Select current item if it's a document
		filtered := g.getFilteredTreeNodes()
		if g.selectedTreeIdx < len(filtered) && filtered[g.selectedTreeIdx].Type == "document" {
			g.selectedDocs[g.selectedTreeIdx] = true
		}
	}
	return g.Layout(g.g)
}

// doExitSelectMode exits select mode without fetching
func (g *Gui) doExitSelectMode() error {
	g.selectMode = false
	g.selectedDocs = make(map[int]bool)
	return g.Layout(g.g)
}

// updateSelectRange updates selectedDocs based on range from selectStartIdx to selectedTreeIdx
func (g *Gui) updateSelectRange() {
	filtered := g.getFilteredTreeNodes()
	g.selectedDocs = make(map[int]bool)

	start, end := g.selectStartIdx, g.selectedTreeIdx
	if start > end {
		start, end = end, start
	}

	for i := start; i <= end; i++ {
		if i < len(filtered) && filtered[i].Type == "document" {
			g.selectedDocs[i] = true
		}
	}
}

// doFetchSelectedDocs fetches all selected documents in parallel
func (g *Gui) doFetchSelectedDocs() error {
	gen := g.databaseGen
	if len(g.selectedDocs) == 0 {
		return nil
	}

	filtered := g.getFilteredTreeNodes()

	// Collect all selected paths and check cache
	combined := make(map[string]any)
	var toFetch []string
	for idx := range g.selectedDocs {
		if idx < len(filtered) && filtered[idx].Type == "document" {
			path := filtered[idx].Path
			if cachedData, ok := g.docCache[path]; ok {
				combined[path] = cachedData
			} else {
				toFetch = append(toFetch, path)
			}
		}
	}

	showCombined := func() {
		if len(combined) > 0 {
			g.currentDocData = combined
			g.currentDocStats = nil // No stats for combined multi-doc view
			g.currentDocPath = fmt.Sprintf("%d documents selected", len(combined))
			g.clearDetailsCache()
		}
	}

	// If all docs are cached, no need to fetch
	if len(toFetch) == 0 {
		showCombined()
		g.logCommand("cache", fmt.Sprintf("Using %d cached documents", len(combined)), "success")
		return nil
	}

	g.logCommand("api", fmt.Sprintf("Fetching %d documents (%d cached)...", len(toFetch), len(combined)), "running")
	g.detailsLoading = true

	// Fetch uncached documents in parallel, off the UI thread
	type result struct {
		path string
		data map[string]any
		err  error
	}
	go func() {
		results := make([]result, len(toFetch))
		var wg sync.WaitGroup
		for i, path := range toFetch {
			wg.Add(1)
			go func(idx int, docPath string) {
				defer wg.Done()
				doc, err := g.firebaseClient.GetDocument(docPath)
				if err != nil {
					results[idx] = result{path: docPath, err: err}
				} else {
					results[idx] = result{path: docPath, data: doc.Data}
				}
			}(i, path)
		}
		wg.Wait()

		g.g.Update(func(gui *gocui.Gui) error {
			if gen != g.databaseGen {
				return nil // loaded for another project or database
			}
			g.detailsLoading = false
			for _, r := range results {
				if r.err != nil {
					g.logCommand("api", fmt.Sprintf("Error fetching %s: %v", r.path, r.err), "error")
					continue
				}
				combined[r.path] = r.data
				g.docCache[r.path] = r.data
			}
			showCombined()
			// Stay in select mode - only Esc exits
			g.logCommand("api", fmt.Sprintf("Loaded %d documents", len(combined)), "success")
			return nil
		})
	}()
	return nil
}

// Query builder action handlers

// doOpenQuery opens the query builder modal
func (g *Gui) doOpenQuery() error {
	return g.openQueryModal()
}

// queryClose closes the query modal
func (g *Gui) queryClose() error {
	_ = g.closeQueryModal()
	return g.relayout()
}

// queryMoveUp moves up in the query modal
func (g *Gui) queryMoveUp() error {
	if g.queryActiveRow == queryRowFilters && len(g.queryFilters) > 0 {
		// In filters: move to previous filter, or wrap to buttons if at first
		filterIdx := g.queryActiveCol / 4
		colInFilter := g.queryActiveCol % 4
		if filterIdx > 0 {
			// Move to previous filter, keep same column within filter
			g.queryActiveCol = (filterIdx-1)*4 + colInFilter
			return g.Layout(g.g)
		}
	}
	// Move to previous row
	g.queryActiveRow--
	if g.queryActiveRow < 0 {
		g.queryActiveRow = queryRowButtons
	}
	// When entering filters from above, go to last filter
	if g.queryActiveRow == queryRowFilters && len(g.queryFilters) > 0 {
		g.queryActiveCol = (len(g.queryFilters) - 1) * 4
	} else {
		g.queryActiveCol = 0
	}
	return g.Layout(g.g)
}

// queryMoveDown moves down in the query modal
func (g *Gui) queryMoveDown() error {
	if g.queryActiveRow == queryRowFilters && len(g.queryFilters) > 0 {
		// In filters: move to next filter, or to orderBy if at last
		filterIdx := g.queryActiveCol / 4
		colInFilter := g.queryActiveCol % 4
		if filterIdx < len(g.queryFilters)-1 {
			// Move to next filter, keep same column within filter
			g.queryActiveCol = (filterIdx+1)*4 + colInFilter
			return g.Layout(g.g)
		}
	}
	// Move to next row
	g.queryActiveRow++
	if g.queryActiveRow > queryRowButtons {
		g.queryActiveRow = queryRowFilters
	}
	g.queryActiveCol = 0
	return g.Layout(g.g)
}

// queryMoveLeft moves left in the query modal
func (g *Gui) queryMoveLeft() error {
	g.queryActiveCol--
	if g.queryActiveCol < 0 {
		g.queryActiveCol = g.getMaxColForRow()
	}
	return g.Layout(g.g)
}

// queryMoveRight moves right in the query modal
func (g *Gui) queryMoveRight() error {
	g.queryActiveCol++
	if g.queryActiveCol > g.getMaxColForRow() {
		g.queryActiveCol = 0
	}
	return g.Layout(g.g)
}

// queryNextField moves to the next field, wrapping to next row at end
func (g *Gui) queryNextField() error {
	maxCol := g.getMaxColForRow()

	if g.queryActiveCol < maxCol {
		// Move to next column in same row
		g.queryActiveCol++
	} else {
		// Move to first column of next row
		g.queryActiveCol = 0
		g.queryActiveRow++
		if g.queryActiveRow > queryRowButtons {
			g.queryActiveRow = queryRowFilters
		}
	}

	return g.Layout(g.g)
}

// queryEnter handles enter key in query modal
func (g *Gui) queryEnter() error {
	return g.handleQueryEnter()
}

// queryAddFilter adds a filter row in the query modal
func (g *Gui) queryAddFilter() error {
	g.addQueryFilter()
	return nil
}

// queryDeleteFilter removes the selected filter row in the query modal
func (g *Gui) queryDeleteFilter() error {
	if g.queryActiveRow == queryRowFilters && len(g.queryFilters) > 0 {
		g.removeQueryFilter()
	}
	return nil
}

// Query select popup handlers

// querySelectMoveUp moves selection up in the select popup
func (g *Gui) querySelectMoveUp() error {
	if g.querySelectIdx > 0 {
		g.querySelectIdx--
	}
	return g.Layout(g.g)
}

// querySelectMoveDown moves selection down in the select popup
func (g *Gui) querySelectMoveDown() error {
	if g.querySelectIdx < len(g.querySelectItems)-1 {
		g.querySelectIdx++
	}
	return g.Layout(g.g)
}

// querySelectConfirm confirms selection and closes popup
func (g *Gui) querySelectConfirm() error {
	g.confirmQuerySelect()
	return g.Layout(g.g)
}

// querySelectClose closes the select popup without selecting
func (g *Gui) querySelectClose() error {
	g.closeQuerySelect()
	return g.Layout(g.g)
}

// doScanCollections shows a confirmation dialog before scanning collections.
// Only works from the projects panel.
func (g *Gui) doScanCollections() error {
	if g.currentColumn != "projects" {
		return nil
	}
	if g.scanRunning {
		g.logCommand("scan", "Scan already in progress", "error")
		return g.Layout(g.g)
	}

	// Always use the focused project in the list
	filtered := g.getFilteredProjects()
	if g.selectedProjectIndex >= len(filtered) || len(filtered) == 0 {
		g.logCommand("scan", "No project available", "error")
		return g.Layout(g.g)
	}
	projectID := filtered[g.selectedProjectIndex].ID

	g.scanProjectID = projectID
	g.confirmOpen = true
	g.confirmTitle = "Collection Health Scan"
	g.confirmMessage = fmt.Sprintf("This will fetch documents from every collection\nin project '%s' to check Firestore limits.\n\nThis may be slow and consume read quota.", projectID)
	g.confirmCallback = g.executeScan
	return g.Layout(g.g)
}

// executeScan runs the actual collection scan after confirmation.
func (g *Gui) executeScan() {
	// Select the project being scanned
	filtered := g.getFilteredProjects()
	for i, p := range filtered {
		if p.ID == g.scanProjectID {
			g.selectedProjectIndex = i
			break
		}
	}

	// Move to details panel
	g.previousColumn = g.currentColumn
	g.currentColumn = "details"

	g.scanRunning = true
	g.scanResults = nil
	g.scanProgress = "loading collections..."
	g.currentDocData = nil
	g.currentDocStats = nil
	g.currentDocPath = ""
	g.clearDetailsCache()
	g.logCommand("scan", fmt.Sprintf("Scanning %s...", g.scanProjectID), "running")

	projectID := g.scanProjectID

	go func() {
		// Set project if needed
		if g.currentProject != projectID {
			if err := g.firebaseClient.SetCurrentProject(projectID); err != nil {
				g.g.Update(func(gui *gocui.Gui) error {
					g.scanRunning = false
					g.logCommand("scan", fmt.Sprintf("Failed to set project: %v", err), "error")
					return nil
				})
				return
			}
			g.g.Update(func(gui *gocui.Gui) error {
				g.resetProjectState(projectID)
				// The scan fills the collections tab; other tabs load now
				if g.collectionsTab != "collections" {
					g.loadActiveTab()
				}
				return nil
			})
		}

		// Fetch collections
		collections, err := g.firebaseClient.ListCollections()
		if err != nil {
			g.g.Update(func(gui *gocui.Gui) error {
				g.scanRunning = false
				g.logCommand("scan", fmt.Sprintf("Failed to list collections: %v", err), "error")
				return nil
			})
			return
		}

		if len(collections) == 0 {
			g.g.Update(func(gui *gocui.Gui) error {
				g.scanRunning = false
				g.scanResults = []ScanResult{}
				g.logCommand("scan", "No collections found", "success")
				return nil
			})
			return
		}

		collNames := make([]string, len(collections))
		for i, c := range collections {
			collNames[i] = c.Name
		}

		g.g.Update(func(gui *gocui.Gui) error {
			// Update collections panel to reflect scanned project
			g.collections = collections
			g.selectedCollectionIdx = 0
			g.currentCollection = ""
			// Clear tree
			g.treeNodes = nil
			g.selectedTreeIdx = 0
			g.expandedPaths = make(map[string]bool)
			g.queryResultMode = false

			g.scanProgress = fmt.Sprintf("0/%d collections", len(collNames))
			return nil
		})

		var results []ScanResult

		for i, collName := range collNames {
			// Update progress
			g.g.Update(func(gui *gocui.Gui) error {
				g.scanProgress = fmt.Sprintf("%d/%d collections", i+1, len(collNames))
				return nil
			})

			docs, err := g.firebaseClient.ListDocuments(collName, 2)
			if err != nil {
				results = append(results, ScanResult{
					Collection: collName,
					Status:     "skipped",
					Message:    fmt.Sprintf("Failed to list: %v", err),
				})
				continue
			}

			if len(docs) == 0 {
				results = append(results, ScanResult{
					Collection: collName,
					Status:     "ok",
					Message:    "Empty collection",
				})
				continue
			}

			// Check first document
			doc := docs[0]
			metrics, warnings := checkDocLimits(doc.Stats, doc.Path)

			if len(warnings) == 0 {
				results = append(results, ScanResult{
					Collection: collName,
					Status:     "ok",
					Message:    fmt.Sprintf("%s - all metrics healthy", doc.ID),
					DocPath:    doc.Path,
					Metrics:    metrics,
				})
				continue
			}

			// First doc has warnings - check second if available
			if len(docs) > 1 {
				doc2 := docs[1]
				_, warnings2 := checkDocLimits(doc2.Stats, doc2.Path)
				if len(warnings2) > 0 {
					// Both docs have warnings - likely a collection-wide pattern
					results = append(results, ScanResult{
						Collection: collName,
						Status:     "warning",
						Message:    fmt.Sprintf("Pattern confirmed across docs (%s, %s)", doc.ID, doc2.ID),
						DocPath:    doc.Path,
						Metrics:    metrics,
						Warnings:   warnings,
					})
					continue
				}
			}

			// Only first doc has issues
			results = append(results, ScanResult{
				Collection: collName,
				Status:     "warning",
				Message:    fmt.Sprintf("%s has issues", doc.ID),
				DocPath:    doc.Path,
				Metrics:    metrics,
				Warnings:   warnings,
			})
		}

		g.g.Update(func(gui *gocui.Gui) error {
			g.scanRunning = false
			g.scanResults = results
			g.clearDetailsCache()

			warningCount := 0
			for _, r := range results {
				if r.Status == "warning" {
					warningCount++
				}
			}
			if warningCount > 0 {
				g.logCommand("scan", fmt.Sprintf("Done: %d/%d collections have warnings", warningCount, len(results)), "error")
			} else {
				g.logCommand("scan", fmt.Sprintf("Done: all %d collections healthy", len(results)), "success")
			}
			return nil
		})
	}()

	g.g.Update(func(gui *gocui.Gui) error { return nil })
}

// formatScanReportMarkdown generates a markdown scan report.
func (g *Gui) formatScanReportMarkdown() string {
	var b strings.Builder

	okCount, warnCount, skipCount := 0, 0, 0
	for _, r := range g.scanResults {
		switch r.Status {
		case "ok":
			okCount++
		case "warning":
			warnCount++
		case "skipped":
			skipCount++
		}
	}

	b.WriteString(fmt.Sprintf("# Collection Health Scan\n\n"))
	b.WriteString(fmt.Sprintf("**Project:** `%s`\n\n", g.scanProjectID))

	// Summary
	summary := fmt.Sprintf("| Scanned | Healthy | Warnings | Skipped |\n")
	summary += fmt.Sprintf("|:-------:|:-------:|:--------:|:-------:|\n")
	summary += fmt.Sprintf("| %d | %d | %d | %d |\n", len(g.scanResults), okCount, warnCount, skipCount)
	b.WriteString(summary)
	b.WriteString("\n")

	// Warnings section
	hasWarnings := false
	for _, r := range g.scanResults {
		if r.Status != "warning" {
			continue
		}
		if !hasWarnings {
			b.WriteString("## Warnings\n\n")
			hasWarnings = true
		}
		b.WriteString(fmt.Sprintf("### %s `%s`\n\n", warningIcon, r.Collection))
		b.WriteString(fmt.Sprintf("> %s\n\n", r.Message))

		warningSet := make(map[string]bool)
		for _, w := range r.Warnings {
			warningSet[w] = true
		}

		b.WriteString("| Metric | Value | Limit | Usage |\n")
		b.WriteString("|--------|------:|------:|------:|\n")
		for _, m := range r.Metrics {
			name, value, limit, pct := parseMetricLine(m)
			flag := ""
			if warningSet[m] {
				flag = " " + warningIcon
			}
			b.WriteString(fmt.Sprintf("| %s%s | %s | %s | %s |\n", name, flag, value, limit, pct))
		}
		b.WriteString("\n")
	}

	// Healthy section
	hasHealthy := false
	for _, r := range g.scanResults {
		if r.Status != "ok" {
			continue
		}
		if !hasHealthy {
			b.WriteString("## Healthy\n\n")
			hasHealthy = true
		}

		if len(r.Metrics) == 0 {
			b.WriteString(fmt.Sprintf("- %s **%s** - %s\n", checkIcon, r.Collection, r.Message))
			continue
		}

		b.WriteString(fmt.Sprintf("<details>\n<summary>%s <strong>%s</strong></summary>\n\n", checkIcon, r.Collection))
		b.WriteString("| Metric | Value | Limit | Usage |\n")
		b.WriteString("|--------|------:|------:|------:|\n")
		for _, m := range r.Metrics {
			name, value, limit, pct := parseMetricLine(m)
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", name, value, limit, pct))
		}
		b.WriteString("\n</details>\n\n")
	}

	// Skipped section
	hasSkipped := false
	for _, r := range g.scanResults {
		if r.Status != "skipped" {
			continue
		}
		if !hasSkipped {
			b.WriteString("## Skipped\n\n")
			hasSkipped = true
		}
		b.WriteString(fmt.Sprintf("- %s **%s** - %s\n", skipIcon, r.Collection, r.Message))
	}

	b.WriteString("\n---\n*Generated by [LazyFire](https://github.com/marjoballabani/lazyfire)*\n")

	return b.String()
}

const (
	warningIcon = "\u26a0\ufe0f" // warning sign
	checkIcon   = "\u2705"       // check mark
	skipIcon    = "\u23ed\ufe0f" // skip
)

// parseMetricLine extracts name, value, limit, pct from "Name: 123/456 (27%)"
func parseMetricLine(m string) (name, value, limit, pct string) {
	// Format: "Size: 123/456 (27%)"
	colonIdx := strings.Index(m, ": ")
	if colonIdx == -1 {
		return m, "", "", ""
	}
	name = m[:colonIdx]
	rest := m[colonIdx+2:]

	slashIdx := strings.Index(rest, "/")
	if slashIdx == -1 {
		return name, rest, "", ""
	}
	value = rest[:slashIdx]

	parenIdx := strings.Index(rest, " (")
	if parenIdx == -1 {
		limit = rest[slashIdx+1:]
		return
	}
	limit = rest[slashIdx+1 : parenIdx]
	pct = rest[parenIdx+2 : len(rest)-1] // strip trailing ")"
	return
}

func (g *Gui) copyScanReport() error {
	text := g.formatScanReportMarkdown()
	if err := copyToClipboard(text); err != nil {
		g.logCommand("copy", fmt.Sprintf("Failed to copy: %v", err), "error")
		return nil
	}

	g.logCommand("copy", "Scan report copied to clipboard", "success")
	return nil
}

func (g *Gui) saveScanReport() error {
	text := g.formatScanReportMarkdown()

	home, _ := os.UserHomeDir()
	filename := fmt.Sprintf("lazyfire-scan_%s.md", g.scanProjectID)
	fullPath := filepath.Join(home, "Downloads", filename)

	if err := os.WriteFile(fullPath, []byte(text), 0644); err != nil {
		g.logCommand("save", fmt.Sprintf("Failed to save: %v", err), "error")
		return nil
	}

	g.logCommand("save", fmt.Sprintf("Saved to %s", fullPath), "success")
	return nil
}

// checkDocLimits checks a document's stats against Firestore limits.
// Returns all metrics as strings, and warnings for any metric above 70%.
func checkDocLimits(stats *firebase.DocStats, docPath string) (metrics []string, warnings []string) {
	if stats == nil {
		return nil, nil
	}

	indexEntries := stats.LeafFields * 2

	type metricDef struct {
		name  string
		value int
		limit int
	}
	defs := []metricDef{
		{"Size", stats.SizeBytes, maxDocSizeBytes},
		{"Index entries", indexEntries, maxIndexEntries},
		{"Depth", stats.MaxDepth, maxDepth},
		{"Field name", stats.MaxFieldName, maxFieldNameBytes},
		{"Field value", stats.MaxFieldValue, maxFieldValueBytes},
		{"Doc path", stats.DocNameSize, maxDocNameBytes},
	}

	for _, d := range defs {
		pct := d.value * 100 / d.limit
		label := fmt.Sprintf("%s: %d/%d (%d%%)", d.name, d.value, d.limit, pct)
		metrics = append(metrics, label)
		if pct > 100 {
			warnings = append(warnings, fmt.Sprintf("%s: %d/%d (OVER LIMIT)", d.name, d.value, d.limit))
		} else if pct > 70 {
			warnings = append(warnings, label)
		}
	}

	return metrics, warnings
}

// --- Sprint 2 features ---

// doFieldSizeBreakdown shows field-by-field size breakdown in details
func (g *Gui) doFieldSizeBreakdown() error {
	if g.currentColumn != "details" || g.currentDocData == nil {
		return nil
	}

	type fieldSize struct {
		name string
		size int
	}

	var fields []fieldSize
	for k, v := range g.currentDocData {
		size := firestoreValueSize(v) + len(k) + 1
		fields = append(fields, fieldSize{name: k, size: size})
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].size > fields[j].size
	})

	var parts []string
	for i, f := range fields {
		if i >= 10 {
			parts = append(parts, fmt.Sprintf("... +%d more", len(fields)-10))
			break
		}
		parts = append(parts, fmt.Sprintf("%s=%s", f.name, formatBytes(f.size)))
	}

	g.logCommand("breakdown", strings.Join(parts, ", "), "success")
	return nil
}

// doFieldTypeAnalysis shows field type distribution across cached docs in current collection
func (g *Gui) doFieldTypeAnalysis() error {
	if g.currentCollection == "" {
		g.logCommand("analysis", "No collection selected", "error")
		return nil
	}

	// Gather all cached docs for the current collection
	typeCounts := make(map[string]map[string]int) // field -> type -> count
	docCount := 0

	for path, data := range g.docCache {
		if !strings.HasPrefix(path, g.currentCollection+"/") {
			continue
		}
		// Only top-level collection docs (not subcollection docs)
		parts := strings.Split(path, "/")
		if len(parts) != 2 {
			continue
		}
		docCount++
		for k, v := range data {
			if typeCounts[k] == nil {
				typeCounts[k] = make(map[string]int)
			}
			typeCounts[k][inferType(v)]++
		}
	}

	if docCount == 0 {
		g.logCommand("analysis", "No cached docs for "+g.currentCollection, "error")
		return nil
	}

	// Sort fields alphabetically
	var fieldNames []string
	for k := range typeCounts {
		fieldNames = append(fieldNames, k)
	}
	sort.Strings(fieldNames)

	var parts []string
	for _, name := range fieldNames {
		types := typeCounts[name]
		var typeStrs []string
		for t, c := range types {
			if c == docCount {
				typeStrs = append(typeStrs, t)
			} else {
				typeStrs = append(typeStrs, fmt.Sprintf("%s(%d)", t, c))
			}
		}
		sort.Strings(typeStrs)
		parts = append(parts, fmt.Sprintf("%s:%s", name, strings.Join(typeStrs, "/")))
	}

	summary := fmt.Sprintf("[%d docs] %s", docCount, strings.Join(parts, ", "))
	if len(summary) > 200 {
		summary = summary[:197] + "..."
	}
	g.logCommand("analysis", summary, "success")
	return nil
}

// doNextSearchMatch moves the details cursor to the next line matching the filter
func (g *Gui) doNextSearchMatch() error {
	return g.jumpToMatch(1)
}

// doPrevSearchMatch moves the details cursor to the previous matching line
func (g *Gui) doPrevSearchMatch() error {
	return g.jumpToMatch(-1)
}

func (g *Gui) jumpToMatch(dir int) error {
	lines := g.detailsViewLines()
	filter := strings.ToLower(g.getDetailsFilter())
	n := len(lines)
	for i := 1; i <= n; i++ {
		idx := ((g.detailsCursor+dir*i)%n + n) % n
		if strings.Contains(strings.ToLower(lines[idx]), filter) {
			g.setDetailsCursor(idx)
			return nil
		}
	}
	g.toast("No matches", true)
	return nil
}

// detailsViewLines returns the text of each (wrapped) line shown in details
func (g *Gui) detailsViewLines() []string {
	if g.g == nil {
		return nil
	}
	v, err := g.g.View(g.views.details)
	if err != nil {
		return nil
	}
	return v.ViewBufferLines()
}

// detailsCursorLine returns the full, unwrapped content line under the cursor
func (g *Gui) detailsCursorLine() string {
	if g.g == nil {
		return ""
	}
	v, err := g.g.View(g.views.details)
	if err != nil {
		return ""
	}
	_, oy := v.Origin()
	line, _ := v.Line(g.detailsCursor - oy)
	return line
}

// cursorLineValue returns the value on the details cursor line
func (g *Gui) cursorLineValue() string {
	line := g.detailsCursorLine()
	switch source := g.detailsSource(); {
	case source == "document":
		return extractJSONLineValue(line, g.showLineNumbers, g.humanizeTimestamps)
	case source == "function" && g.detailsTab == "logs":
		return strings.TrimSpace(line)
	case source == "function", source == "storage", source == "auth", source == "info":
		return extractInfoLineValue(line)
	}
	return strings.TrimSpace(line)
}

var (
	lineNumberPrefix = regexp.MustCompile(`^\s*\d+ `)
	jsonKeyValue     = regexp.MustCompile(`^"(?:[^"\\]|\\.)*":\s*(.*)$`)
	infoKeyValue     = regexp.MustCompile(`^[A-Za-z][A-Za-z -]*:\s+(.+)$`)
)

// extractJSONLineValue returns the value of a pretty-printed JSON line:
// `"name": "Alice",` gives `Alice`. Lines that only open or close an object
// or array have no single-line value and give "".
func extractJSONLineValue(line string, hasLineNumbers, hasTimestampNotes bool) string {
	if hasLineNumbers {
		line = lineNumberPrefix.ReplaceAllString(line, "")
	}
	if hasTimestampNotes {
		if i := strings.Index(line, "  // "); i >= 0 {
			line = line[:i]
		}
	}
	line = strings.TrimSpace(line)
	if m := jsonKeyValue.FindStringSubmatch(line); m != nil {
		line = m[1]
	}
	line = strings.TrimSuffix(line, ",")
	switch line {
	case "", "{", "}", "[", "]":
		return ""
	}
	var s string
	if strings.HasPrefix(line, `"`) && json.Unmarshal([]byte(line), &s) == nil {
		return s
	}
	return line
}

// extractInfoLineValue returns the value of a "Label:   value" line, or the
// whole line when it has no label
func extractInfoLineValue(line string) string {
	line = strings.TrimSpace(line)
	if m := infoKeyValue.FindStringSubmatch(line); m != nil {
		return strings.TrimSpace(m[1])
	}
	return line
}

// doCopyFieldValue copies the value on the details cursor line
func (g *Gui) doCopyFieldValue() error {
	value := g.cursorLineValue()
	if value == "" {
		g.toast("No single-line value here (c copies the whole JSON)", true)
		return nil
	}
	if err := copyToClipboard(value); err != nil {
		g.logCommand("copy", fmt.Sprintf("Failed: %v", err), "error")
		return nil
	}

	display := value
	if len(display) > 60 {
		display = display[:57] + "..."
	}
	g.logCommand("copy", fmt.Sprintf("Copied value: %s", display), "success")
	return nil
}

// doToggleLineNumbers toggles line numbers in details JSON view
func (g *Gui) doToggleLineNumbers() error {
	if g.currentColumn != "details" {
		return nil
	}
	g.showLineNumbers = !g.showLineNumbers
	g.clearDetailsCache()
	if g.showLineNumbers {
		g.logCommand("view", "Line numbers on", "success")
	} else {
		g.logCommand("view", "Line numbers off", "success")
	}
	return g.Layout(g.g)
}

// doCycleLogLevel cycles through log level filters for function logs
func (g *Gui) doCycleLogLevel() error {
	levels := []string{"", "ERROR", "WARNING", "INFO", "DEBUG"}
	currentIdx := 0
	for i, l := range levels {
		if l == g.logLevelFilter {
			currentIdx = i
			break
		}
	}
	g.logLevelFilter = levels[(currentIdx+1)%len(levels)]
	g.clearDetailsCache()
	if g.logLevelFilter == "" {
		g.logCommand("logs", "Showing all log levels", "success")
	} else {
		g.logCommand("logs", fmt.Sprintf("Filter: %s only", g.logLevelFilter), "success")
	}
	return g.Layout(g.g)
}

// doCollectionMemoryEstimate shows estimated memory for current collection from cache
func (g *Gui) doCollectionMemoryEstimate() error {
	if g.currentCollection == "" {
		g.logCommand("memory", "No collection selected", "error")
		return nil
	}

	totalSize := 0
	docCount := 0
	for path := range g.docCache {
		if !strings.HasPrefix(path, g.currentCollection+"/") {
			continue
		}
		parts := strings.Split(path, "/")
		if len(parts) != 2 {
			continue
		}
		docCount++
		if stats, ok := g.statsCache[path]; ok && stats != nil {
			totalSize += stats.SizeBytes
		} else {
			// Estimate from JSON marshal
			if data, err := json.Marshal(g.docCache[path]); err == nil {
				totalSize += len(data)
			}
		}
	}

	if docCount == 0 {
		g.logCommand("memory", "No cached docs for "+g.currentCollection, "error")
		return nil
	}

	avg := totalSize / docCount
	g.logCommand("memory",
		fmt.Sprintf("%s: %d docs, total ~%s, avg ~%s/doc",
			g.currentCollection, docCount, formatBytes(totalSize), formatBytes(avg)),
		"success")
	return nil
}

// doToggleBase64Decode decodes the base64 string on the details cursor line
func (g *Gui) doToggleBase64Decode() error {
	value := g.cursorLineValue()
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(value)
	}
	if value == "" || err != nil || len(decoded) == 0 {
		g.toast("No base64 string on this line", true)
		return nil
	}

	display := string(decoded)
	if len(display) > 100 {
		display = display[:97] + "..."
	}
	// Check if decoded content is printable
	printable := true
	for _, b := range decoded {
		if b < 0x20 && b != '\n' && b != '\r' && b != '\t' {
			printable = false
			break
		}
	}
	if printable {
		g.logCommand("base64", fmt.Sprintf("Decoded: %s", display), "success")
	} else {
		g.logCommand("base64", fmt.Sprintf("Binary data, %d bytes", len(decoded)), "success")
	}
	return nil
}

// doSelectStorageItem opens the selected bucket or folder
func (g *Gui) doSelectStorageItem() error {
	if g.currentBucket == "" {
		buckets := g.getFilteredBuckets()
		if g.selectedBucketIdx < len(buckets) {
			g.currentBucket = buckets[g.selectedBucketIdx].Name
			g.storagePrefix = ""
			g.storagePrefixStack = nil
			g.storageFilter = ""
			g.selectedObjectIdx = 0
			g.loadStorageObjects("")
		}
		return nil
	}

	objects := g.getFilteredObjects()
	if g.selectedObjectIdx < len(objects) {
		obj := objects[g.selectedObjectIdx]
		if obj.IsPrefix {
			g.storagePrefixStack = append(g.storagePrefixStack, g.storagePrefix)
			g.storagePrefix = obj.Name
			g.storageFilter = ""
			g.selectedObjectIdx = 0
			g.loadStorageObjects("")
		}
	}
	return nil
}

// doStorageBack goes up one folder, or back to the bucket list, keeping the
// folder or bucket we came from selected
func (g *Gui) doStorageBack() error {
	g.storageFilter = ""
	g.storageLoading = false // a folder still loading is dropped when it arrives
	if g.storagePrefix != "" {
		leaving := g.storagePrefix
		if len(g.storagePrefixStack) > 0 {
			g.storagePrefix = g.storagePrefixStack[len(g.storagePrefixStack)-1]
			g.storagePrefixStack = g.storagePrefixStack[:len(g.storagePrefixStack)-1]
		} else {
			g.storagePrefix = ""
		}
		g.loadStorageObjects(leaving)
	} else if g.currentBucket != "" {
		leaving := g.currentBucket
		g.currentBucket = ""
		g.storageObjects = nil
		g.selectItemByKey(CtxStorage, leaving)
	}
	return nil
}
