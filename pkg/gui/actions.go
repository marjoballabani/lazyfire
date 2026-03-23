package gui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/jesseduffield/gocui"
	"github.com/marjoballabani/lazyfire/pkg/firebase"
)

// Actions - clean handler functions without state checks.
// State checks are handled by the binding system's GetDisabledReason.

// doQuit exits the application
func (g *Gui) doQuit() error {
	return gocui.ErrQuit
}

// doEscape handles escape key - closes modals, cancels filter, returns from details
func (g *Gui) doEscape() error {
	// Priority: help popup > command modal > details panel > select mode (only in tree) > filter input > committed filter
	if g.helpOpen {
		g.helpOpen = false
		g.helpPopup = nil
		return g.Layout(g.g)
	}
	if g.modalOpen {
		g.modalOpen = false
		return g.Layout(g.g)
	}
	// Return from details to previous panel (keeps select mode)
	if g.currentColumn == "details" {
		g.scanResults = nil
		g.clearDetailsCache()
		target := g.previousColumn
		if target == "" {
			target = "tree"
		}
		return g.setFocus(g.g, target)
	}
	// Storage: go back up folder/bucket hierarchy
	if g.currentColumn == "collections" && g.collectionsTab == "storage" && (g.currentBucket != "" || g.storagePrefix != "") {
		return g.doStorageBack()
	}
	// Exit select mode only when in tree panel
	if g.selectMode && g.currentColumn == "tree" {
		return g.doExitSelectMode()
	}
	if g.filterInputActive {
		return g.cancelFilterInput(g.g)
	}
	if g.hasActiveFilter(g.currentColumn) {
		return g.clearCurrentFilter(g.g)
	}
	return nil
}

// doToggleHelp toggles the help popup
func (g *Gui) doToggleHelp() error {
	if g.helpOpen {
		g.helpOpen = false
		g.helpPopup = nil
	} else {
		g.buildHelpPopup()
		g.helpOpen = true
	}
	return g.Layout(g.g)
}

// doToggleModal toggles the command log modal
func (g *Gui) doToggleModal() error {
	g.modalOpen = !g.modalOpen
	return g.Layout(g.g)
}

// Context-specific handlers for help popup
func (g *Gui) helpMoveUp() error {
	if g.helpPopup != nil {
		g.helpPopup.MoveUp()
	}
	return g.Layout(g.g)
}

func (g *Gui) helpMoveDown() error {
	if g.helpPopup != nil {
		g.helpPopup.MoveDown()
	}
	return g.Layout(g.g)
}

func (g *Gui) helpClose() error {
	// Get selected item before closing
	var action func() error
	if g.helpPopup != nil {
		item := g.helpPopup.GetSelectedItem()
		if item != nil && item.Action != nil {
			action = item.Action
		}
	}

	// Close popup
	g.helpOpen = false
	g.helpPopup = nil

	// Execute action if any
	if action != nil {
		return action()
	}
	return g.Layout(g.g)
}

// Context-specific handlers for filter mode
func (g *Gui) filterCursorLeft() error {
	if g.filterCursorPos > 0 {
		g.filterCursorPos--
	}
	return g.Layout(g.g)
}

func (g *Gui) filterCursorRight() error {
	if g.filterCursorPos < len(g.filterInputText) {
		g.filterCursorPos++
	}
	return g.Layout(g.g)
}

// Block handler - does nothing (for modal context)
func (g *Gui) blockAction() error {
	return nil
}

// Filter char inserters for keys that have other bindings
func (g *Gui) filterInsertJ() error         { return g.insertFilterChar(g.g, 'j') }
func (g *Gui) filterInsertK() error         { return g.insertFilterChar(g.g, 'k') }
func (g *Gui) filterInsertH() error         { return g.insertFilterChar(g.g, 'h') }
func (g *Gui) filterInsertL() error         { return g.insertFilterChar(g.g, 'l') }
func (g *Gui) filterInsertQuestion() error  { return g.insertFilterChar(g.g, '?') }
func (g *Gui) filterInsertAt() error        { return g.insertFilterChar(g.g, '@') }
func (g *Gui) filterInsertC() error         { return g.insertFilterChar(g.g, 'c') }
func (g *Gui) filterInsertS() error         { return g.insertFilterChar(g.g, 's') }
func (g *Gui) filterInsertR() error         { return g.insertFilterChar(g.g, 'r') }
func (g *Gui) filterInsertQ() error      { return g.insertFilterChar(g.g, 'q') }
func (g *Gui) filterInsertUpperF() error { return g.insertFilterChar(g.g, 'F') }
func (g *Gui) filterInsertUpperS() error { return g.insertFilterChar(g.g, 'S') }

func (g *Gui) doConfirmAccept() error {
	if g.confirmCallback != nil {
		cb := g.confirmCallback
		g.confirmOpen = false
		g.confirmCallback = nil
		cb()
	}
	return g.Layout(g.g)
}

func (g *Gui) doConfirmCancel() error {
	g.confirmOpen = false
	g.confirmCallback = nil
	return g.Layout(g.g)
}
func (g *Gui) filterInsertV() error         { return g.insertFilterChar(g.g, 'v') }
func (g *Gui) filterInsertE() error         { return g.insertFilterChar(g.g, 'e') }
func (g *Gui) filterInsertSlash() error        { return g.insertFilterChar(g.g, '/') }
func (g *Gui) filterInsertBracketLeft() error  { return g.insertFilterChar(g.g, '[') }
func (g *Gui) filterInsertBracketRight() error { return g.insertFilterChar(g.g, ']') }

// doColumnLeft switches to the panel on the left (skips details)
func (g *Gui) doColumnLeft() error {
	if g.currentColumn == "details" {
		return nil // Use Esc to leave details
	}
	var newColumn string
	switch g.currentColumn {
	case "projects":
		newColumn = "tree" // wrap to tree
	case "collections":
		newColumn = "projects"
	case "tree":
		newColumn = "collections"
	}
	return g.setFocus(g.g, newColumn)
}

// doColumnRight switches to the panel on the right (skips details)
func (g *Gui) doColumnRight() error {
	if g.currentColumn == "details" {
		return nil // Use Esc to leave details
	}
	var newColumn string
	switch g.currentColumn {
	case "projects":
		newColumn = "collections"
	case "collections":
		newColumn = "tree"
	case "tree":
		newColumn = "projects" // wrap to projects
	}
	return g.setFocus(g.g, newColumn)
}

// doPageUp jumps up 10 items in current panel
func (g *Gui) doPageUp() error {
	switch g.currentColumn {
	case "projects":
		g.selectedProjectIndex -= 10
		if g.selectedProjectIndex < 0 {
			g.selectedProjectIndex = 0
		}
		g.currentProjectInfo = nil
	case "collections":
		g.pageUpCollections(10)
	case "tree":
		g.selectedTreeIdx -= 10
		if g.selectedTreeIdx < 0 {
			g.selectedTreeIdx = 0
		}
	case "details":
		g.detailsScrollPos -= 10
		if g.detailsScrollPos < 0 {
			g.detailsScrollPos = 0
		}
	}
	return g.Layout(g.g)
}

// doPageDown jumps down 10 items in current panel
func (g *Gui) doPageDown() error {
	switch g.currentColumn {
	case "projects":
		filtered := g.getFilteredProjects()
		g.selectedProjectIndex += 10
		if g.selectedProjectIndex >= len(filtered) {
			g.selectedProjectIndex = len(filtered) - 1
		}
		if g.selectedProjectIndex < 0 {
			g.selectedProjectIndex = 0
		}
		g.currentProjectInfo = nil
	case "collections":
		g.pageDownCollections(10)
	case "tree":
		filtered := g.getFilteredTreeNodes()
		g.selectedTreeIdx += 10
		if g.selectedTreeIdx >= len(filtered) {
			g.selectedTreeIdx = len(filtered) - 1
		}
		if g.selectedTreeIdx < 0 {
			g.selectedTreeIdx = 0
		}
	case "details":
		g.detailsScrollPos += 10
	}
	return g.Layout(g.g)
}

// doCursorUp moves selection up in current panel
func (g *Gui) doCursorUp() error {
	switch g.currentColumn {
	case "projects":
		if g.selectedProjectIndex > 0 {
			g.selectedProjectIndex--
			g.currentProjectInfo = nil
		}
	case "collections":
		switch g.collectionsTab {
		case "functions":
			filtered := g.getFilteredFunctions()
			if g.selectedFunctionIdx > 0 && g.selectedFunctionIdx < len(filtered) {
				g.selectedFunctionIdx--
			}
		case "storage":
			if g.currentBucket == "" {
				if g.selectedBucketIdx > 0 {
					g.selectedBucketIdx--
				}
			} else {
				if g.selectedObjectIdx > 0 {
					g.selectedObjectIdx--
				}
			}
		case "auth":
			if g.selectedAuthIdx > 0 {
				g.selectedAuthIdx--
			}
		default:
			if g.selectedCollectionIdx > 0 {
				g.selectedCollectionIdx--
			}
		}
	case "tree":
		if g.selectedTreeIdx > 0 {
			g.selectedTreeIdx--
		}
	case "details":
		if g.detailsScrollPos > 0 {
			g.detailsScrollPos--
		}
	}
	return g.Layout(g.g)
}

// doCursorDown moves selection down in current panel
func (g *Gui) doCursorDown() error {
	switch g.currentColumn {
	case "projects":
		filtered := g.getFilteredProjects()
		if g.selectedProjectIndex < len(filtered)-1 {
			g.selectedProjectIndex++
			g.currentProjectInfo = nil
		}
	case "collections":
		switch g.collectionsTab {
		case "functions":
			filtered := g.getFilteredFunctions()
			if g.selectedFunctionIdx < len(filtered)-1 {
				g.selectedFunctionIdx++
			}
		case "storage":
			if g.currentBucket == "" {
				if g.selectedBucketIdx < len(g.storageBuckets)-1 {
					g.selectedBucketIdx++
				}
			} else {
				if g.selectedObjectIdx < len(g.storageObjects)-1 {
					g.selectedObjectIdx++
				}
			}
		case "auth":
			if g.selectedAuthIdx < len(g.authUsers)-1 {
				g.selectedAuthIdx++
			}
		default:
			filtered := g.getFilteredCollections()
			if g.selectedCollectionIdx < len(filtered)-1 {
				g.selectedCollectionIdx++
			}
		}
	case "tree":
		filtered := g.getFilteredTreeNodes()
		if g.selectedTreeIdx < len(filtered)-1 {
			g.selectedTreeIdx++
		}
	case "details":
		g.detailsScrollPos++
	}
	return g.Layout(g.g)
}

// doHalfPageDown scrolls details half a page down
func (g *Gui) doHalfPageDown() error {
	if g.currentColumn == "details" {
		g.detailsScrollPos += 20
	}
	return g.Layout(g.g)
}

// doHalfPageUp scrolls details half a page up
func (g *Gui) doHalfPageUp() error {
	if g.currentColumn == "details" {
		g.detailsScrollPos -= 20
		if g.detailsScrollPos < 0 {
			g.detailsScrollPos = 0
		}
	}
	return g.Layout(g.g)
}

// doJumpToProjects focuses the projects panel
func (g *Gui) doJumpToProjects() error {
	if g.currentColumn == "details" {
		return nil
	}
	return g.setFocus(g.g, "projects")
}

// doJumpToCollections focuses the collections panel
func (g *Gui) doJumpToCollections() error {
	if g.currentColumn == "details" {
		return nil
	}
	return g.setFocus(g.g, "collections")
}

// doJumpToTree focuses the tree panel
func (g *Gui) doJumpToTree() error {
	if g.currentColumn == "details" {
		return nil
	}
	return g.setFocus(g.g, "tree")
}

// doGoToTop jumps to the first item in current panel
func (g *Gui) doGoToTop() error {
	switch g.currentColumn {
	case "projects":
		g.selectedProjectIndex = 0
		g.currentProjectInfo = nil
	case "collections":
		switch g.collectionsTab {
		case "functions":
			g.selectedFunctionIdx = 0
		case "storage":
			if g.currentBucket == "" {
				g.selectedBucketIdx = 0
			} else {
				g.selectedObjectIdx = 0
			}
		case "auth":
			g.selectedAuthIdx = 0
		default:
			g.selectedCollectionIdx = 0
		}
	case "tree":
		g.selectedTreeIdx = 0
	case "details":
		g.detailsScrollPos = 0
	}
	return g.Layout(g.g)
}

// doGoToBottom jumps to the last item in current panel
func (g *Gui) doGoToBottom() error {
	switch g.currentColumn {
	case "projects":
		filtered := g.getFilteredProjects()
		if len(filtered) > 0 {
			g.selectedProjectIndex = len(filtered) - 1
		}
		g.currentProjectInfo = nil
	case "collections":
		switch g.collectionsTab {
		case "functions":
			filtered := g.getFilteredFunctions()
			if len(filtered) > 0 {
				g.selectedFunctionIdx = len(filtered) - 1
			}
		case "storage":
			if g.currentBucket == "" {
				if len(g.storageBuckets) > 0 {
					g.selectedBucketIdx = len(g.storageBuckets) - 1
				}
			} else {
				if len(g.storageObjects) > 0 {
					g.selectedObjectIdx = len(g.storageObjects) - 1
				}
			}
		case "auth":
			if len(g.authUsers) > 0 {
				g.selectedAuthIdx = len(g.authUsers) - 1
			}
		default:
			filtered := g.getFilteredCollections()
			if len(filtered) > 0 {
				g.selectedCollectionIdx = len(filtered) - 1
			}
		}
	case "tree":
		filtered := g.getFilteredTreeNodes()
		if len(filtered) > 0 {
			g.selectedTreeIdx = len(filtered) - 1
		}
	case "details":
		// Scroll to bottom - use a large number, layout will clamp
		g.detailsScrollPos = 99999
	}
	return g.Layout(g.g)
}

// doNextColumn - Tab goes to details panel from any panel
func (g *Gui) doNextColumn() error {
	if g.currentColumn == "details" {
		return nil // Already in details, do nothing
	}
	g.previousColumn = g.currentColumn
	return g.setFocus(g.g, "details")
}

// doSwitchTab switches tabs based on current panel ([ and ] keys)
// Collections panel: switch Collections/Functions tabs
// Details panel: switch Details/Logs tabs (only when Functions tab is active)
// doSwitchTabNext cycles to next tab (] key)
func (g *Gui) doSwitchTabNext() error {
	return g.doSwitchTabDir(1)
}

// doSwitchTabPrev cycles to previous tab ([ key)
func (g *Gui) doSwitchTabPrev() error {
	return g.doSwitchTabDir(-1)
}

func (g *Gui) doSwitchTab() error {
	return g.doSwitchTabDir(1)
}

func (g *Gui) doSwitchTabDir(dir int) error {
	switch g.currentColumn {
	case "collections":
		tabs := []string{"collections", "functions", "storage", "auth", "rules", "indexes"}
		currentIdx := 0
		for i, t := range tabs {
			if t == g.collectionsTab {
				currentIdx = i
				break
			}
		}
		next := (currentIdx + dir + len(tabs)) % len(tabs)
		g.collectionsTab = tabs[next]

		// Reset view scroll position when switching tabs
		if v, err := g.g.View("collections"); err == nil {
			v.SetOrigin(0, 0)
		}

		// Load data for the new tab if needed
		switch g.collectionsTab {
		case "collections":
			g.stopLogsRefresh()
			if len(g.collections) == 0 && g.currentProject != "" {
				g.collectionsLoading = true
				go func() {
					if err := g.loadCollections(); err != nil {
						g.g.Update(func(gui *gocui.Gui) error {
							g.collectionsLoading = false
							g.logCommand("api", fmt.Sprintf("ListCollections failed: %v", err), "error")
							return nil
						})
						return
					}
					g.g.Update(func(gui *gocui.Gui) error {
						g.collectionsLoading = false
						g.logCommand("api", fmt.Sprintf("ListCollections → %d collections", len(g.collections)), "success")
						return nil
					})
				}()
			}
		case "functions":
			if len(g.functions) == 0 && g.currentProject != "" {
				g.loadFunctions()
			}
		case "storage":
			if len(g.storageBuckets) == 0 && g.currentProject != "" {
				g.loadStorageBuckets()
			}
		case "auth":
			if len(g.authUsers) == 0 && g.currentProject != "" {
				g.loadAuthUsers()
			}
		case "rules":
			if g.firestoreRules == nil && g.currentProject != "" {
				g.loadFirestoreRules()
			}
		case "indexes":
			if g.firestoreIndexes == nil && g.currentProject != "" {
				g.loadFirestoreIndexes()
			}
		}
	case "details":
		// Switch Details/Logs tabs (only when Functions tab is active)
		if g.collectionsTab != "functions" {
			return nil
		}
		if g.detailsTab == "details" {
			g.detailsTab = "logs"
			if g.currentFunction != nil && len(g.functionLogs) == 0 && !g.logsLoading {
				g.loadFunctionLogs()
			}
		} else {
			g.detailsTab = "details"
		}
	default:
		return nil
	}
	return g.Layout(g.g)
}

// doSpace handles space key - select/expand in current panel
// doSpace - normal mode space handler
func (g *Gui) doSpace() error {
	switch g.currentColumn {
	case "projects":
		return g.selectProject(g.g)
	case "collections":
		switch g.collectionsTab {
		case "functions":
			return g.selectFunction(g.g)
		case "storage":
			return g.doSelectStorageItem()
		default:
			return g.selectCollection(g.g)
		}
	case "tree":
		return g.selectTreeNode(g.g)
	}
	return nil
}

// filterInsertSpace inserts space in filter
func (g *Gui) filterInsertSpace() error {
	return g.insertFilterChar(g.g, ' ')
}

// doEnter - normal mode enter handler
func (g *Gui) doEnter() error {
	switch g.currentColumn {
	case "projects":
		return g.fetchProjectDetails(g.g)
	case "collections":
		switch g.collectionsTab {
		case "functions":
			// Select function and go to details to see logs
			if err := g.selectFunction(g.g); err != nil {
				return err
			}
			g.previousColumn = g.currentColumn
			return g.setFocus(g.g, "details")
		case "storage":
			return g.doSelectStorageItem()
		case "auth":
			// Go to details to view user info
			g.previousColumn = g.currentColumn
			return g.setFocus(g.g, "details")
		}
	case "tree":
		// In select mode with docs already loaded, just go to details
		if g.selectMode && g.currentDocData != nil {
			g.previousColumn = g.currentColumn
			return g.setFocus(g.g, "details")
		}
		// Select the node (loads document) then go to details
		if err := g.selectTreeNode(g.g); err != nil {
			return err
		}
		g.previousColumn = g.currentColumn
		return g.setFocus(g.g, "details")
	}
	return nil
}

// filterCommit commits the filter
func (g *Gui) filterCommit() error {
	return g.commitFilter(g.g)
}

// doStartFilter starts filter mode for current panel
func (g *Gui) doStartFilter() error {
	if g.filterInputActive {
		return nil
	}
	// Clear existing committed filter
	switch g.currentColumn {
	case "projects":
		g.projectsFilter = ""
	case "collections":
		switch g.collectionsTab {
		case "functions":
			g.functionsFilter = ""
		case "storage":
			g.storageFilter = ""
		case "auth":
			g.authFilter = ""
		default:
			g.collectionsFilter = ""
		}
	case "tree":
		g.treeFilter = ""
	case "details":
		g.detailsFilter = ""
	}
	g.filterInputActive = true
	g.filterInputPanel = g.currentColumn
	g.filterInputText = ""
	g.filterCursorPos = 0
	return g.Layout(g.g)
}

// doFilterBackspace handles backspace in filter mode
func (g *Gui) doFilterBackspace() error {
	if !g.filterInputActive {
		return nil
	}
	if g.filterCursorPos > 0 && len(g.filterInputText) > 0 {
		g.filterInputText = g.filterInputText[:g.filterCursorPos-1] + g.filterInputText[g.filterCursorPos:]
		g.filterCursorPos--
	}
	return g.Layout(g.g)
}

// makeFilterCharAction creates a handler for a specific character
func (g *Gui) makeFilterCharAction(ch rune) func() error {
	return func() error {
		if !g.filterInputActive {
			return nil
		}
		return g.insertFilterChar(g.g, ch)
	}
}

// doCopyJSON copies current document to clipboard
func (g *Gui) doCopyJSON() error {
	if g.scanResults != nil && g.currentDocData == nil {
		return g.copyScanReport()
	}
	if g.currentColumn != "tree" && g.currentColumn != "details" {
		return nil
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
	var path string
	switch g.currentColumn {
	case "tree":
		filtered := g.getFilteredTreeNodes()
		if g.selectedTreeIdx < len(filtered) {
			path = filtered[g.selectedTreeIdx].Path
		}
	case "details":
		path = g.currentDocPath
	default:
		return nil
	}
	if path == "" {
		g.logCommand("path", "No path to copy", "error")
		return nil
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		g.logCommand("path", "Clipboard not supported", "error")
		return nil
	}
	cmd.Stdin = strings.NewReader(path)
	if err := cmd.Run(); err != nil {
		g.logCommand("path", fmt.Sprintf("Failed: %v", err), "error")
		return nil
	}
	g.logCommand("path", fmt.Sprintf("Copied: %s", path), "success")
	return nil
}

// doSaveJSON saves current document to file
func (g *Gui) doSaveJSON() error {
	if g.scanResults != nil && g.currentDocData == nil {
		return g.saveScanReport()
	}
	if g.currentColumn != "tree" && g.currentColumn != "details" {
		return nil
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

// doRefresh reloads data based on current panel and tab
func (g *Gui) doRefresh() error {
	switch g.currentColumn {
	case "details":
		// In details view with Logs tab: refresh logs
		if g.detailsTab == "logs" && g.currentFunction != nil {
			g.logCommand("r", "Refreshing logs...", "running")
			g.loadFunctionLogs()
			return g.Layout(g.g)
		}
	case "collections":
		// In collections panel with functions tab: refresh functions
		if g.collectionsTab == "functions" {
			g.logCommand("r", "Refreshing functions...", "running")
			g.loadFunctions()
			return g.Layout(g.g)
		}
		// In collections panel with collections tab: refresh collections
		if g.currentProject != "" {
			g.logCommand("r", "Refreshing collections...", "running")
			g.collectionsLoading = true
			go func() {
				if err := g.loadCollections(); err != nil {
					g.g.Update(func(gui *gocui.Gui) error {
						g.collectionsLoading = false
						g.logCommand("r", fmt.Sprintf("Failed: %v", err), "error")
						return nil
					})
					return
				}
				g.g.Update(func(gui *gocui.Gui) error {
					g.collectionsLoading = false
					g.logCommand("r", fmt.Sprintf("Loaded %d collections", len(g.collections)), "success")
					return nil
				})
			}()
			return g.Layout(g.g)
		}
	case "projects":
		// In projects panel: refresh projects
		g.logCommand("r", "Refreshing projects...", "running")
		if err := g.loadProjects(); err != nil {
			g.logCommand("r", fmt.Sprintf("Failed: %v", err), "error")
			return g.Layout(g.g)
		}
		g.logCommand("r", fmt.Sprintf("Loaded %d projects", len(g.projects)), "success")
		return g.Layout(g.g)
	case "tree":
		// In tree panel: refresh current collection documents
		if g.currentCollection != "" {
			g.logCommand("r", "Refreshing documents...", "running")
			g.treeLoading = true
			go func() {
				docs, err := g.firebaseClient.ListDocuments(g.currentCollection, 50)
				g.g.Update(func(gui *gocui.Gui) error {
					g.treeLoading = false
					if err != nil {
						g.logCommand("r", fmt.Sprintf("Failed: %v", err), "error")
						return nil
					}
					g.treeNodes = nil
					for _, doc := range docs {
						g.docCache[doc.Path] = doc.Data
						node := TreeNode{
							Path:        doc.Path,
							Name:        doc.ID,
							Type:        "document",
							Depth:       0,
							HasChildren: true,
							Expanded:    false,
						}
						g.treeNodes = append(g.treeNodes, node)
					}
					g.selectedTreeIdx = 0
					g.logCommand("r", fmt.Sprintf("Loaded %d documents", len(docs)), "success")
					return nil
				})
			}()
			return g.Layout(g.g)
		}
	}

	return nil
}

// Mouse click handlers

func (g *Gui) doHelpClick() error {
	if g.helpPopup == nil {
		return nil
	}
	v, _ := g.g.View("helpModal")
	if v == nil {
		return nil
	}
	_, cy := v.Cursor()
	_, oy := v.Origin()
	clickedLine := cy + oy

	if clickedLine >= 0 && clickedLine < len(g.helpPopup.Items) {
		item := &g.helpPopup.Items[clickedLine]
		if !item.IsHeader {
			g.helpPopup.SelectedIdx = clickedLine
		}
	}
	return g.Layout(g.g)
}

func (g *Gui) doProjectsClick() error {
	if g.helpOpen {
		g.helpOpen = false
		g.helpPopup = nil
		return g.Layout(g.g)
	}
	g.currentColumn = "projects"
	v, _ := g.g.View("projects")
	if v == nil {
		return g.Layout(g.g)
	}
	_, cy := v.Cursor()
	_, oy := v.Origin()
	clickedLine := cy + oy

	filtered := g.getFilteredProjects()
	if clickedLine >= 0 && clickedLine < len(filtered) {
		g.selectedProjectIndex = clickedLine
		g.currentProjectInfo = nil
	}
	return g.Layout(g.g)
}

func (g *Gui) doCollectionsClick() error {
	if g.helpOpen {
		g.helpOpen = false
		g.helpPopup = nil
		return g.Layout(g.g)
	}
	g.currentColumn = "collections"
	v, _ := g.g.View("collections")
	if v == nil {
		return g.Layout(g.g)
	}
	_, cy := v.Cursor()
	_, oy := v.Origin()
	clickedLine := cy + oy

	filtered := g.getFilteredCollections()
	if clickedLine >= 0 && clickedLine < len(filtered) {
		g.selectedCollectionIdx = clickedLine
	}
	return g.Layout(g.g)
}

func (g *Gui) doTreeClick() error {
	if g.helpOpen {
		g.helpOpen = false
		g.helpPopup = nil
		return g.Layout(g.g)
	}
	g.currentColumn = "tree"
	v, _ := g.g.View("tree")
	if v == nil {
		return g.Layout(g.g)
	}
	_, cy := v.Cursor()
	_, oy := v.Origin()
	clickedLine := cy + oy

	filtered := g.getFilteredTreeNodes()
	if clickedLine >= 0 && clickedLine < len(filtered) {
		g.selectedTreeIdx = clickedLine
	}
	return g.Layout(g.g)
}

func (g *Gui) doDetailsClick() error {
	if g.helpOpen {
		g.helpOpen = false
		g.helpPopup = nil
		return g.Layout(g.g)
	}
	g.previousColumn = g.currentColumn
	g.currentColumn = "details"
	return g.Layout(g.g)
}

func (g *Gui) doOutsideClick() error {
	if g.helpOpen {
		g.helpOpen = false
		g.helpPopup = nil
		return g.Layout(g.g)
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

// selectMoveDown moves down in select mode, extending selection
func (g *Gui) selectMoveDown() error {
	if !g.selectMode || g.currentColumn != "tree" {
		return g.doCursorDown()
	}
	filtered := g.getFilteredTreeNodes()
	if g.selectedTreeIdx < len(filtered)-1 {
		g.selectedTreeIdx++
		g.updateSelectRange()
	}
	return g.Layout(g.g)
}

// selectMoveUp moves up in select mode, extending selection
func (g *Gui) selectMoveUp() error {
	if !g.selectMode || g.currentColumn != "tree" {
		return g.doCursorUp()
	}
	if g.selectedTreeIdx > 0 {
		g.selectedTreeIdx--
		g.updateSelectRange()
	}
	return g.Layout(g.g)
}

// doFetchSelectedDocs fetches all selected documents in parallel
func (g *Gui) doFetchSelectedDocs() error {
	if !g.selectMode || len(g.selectedDocs) == 0 {
		return g.doSpace()
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

	// If all docs are cached, no need to fetch
	if len(toFetch) == 0 {
		if len(combined) > 0 {
			g.currentDocData = combined
			g.currentDocStats = nil // No stats for combined multi-doc view
			g.currentDocPath = fmt.Sprintf("%d documents selected", len(combined))
			g.clearDetailsCache()
			g.logCommand("cache", fmt.Sprintf("Using %d cached documents", len(combined)), "success")
		}
		return g.Layout(g.g)
	}

	g.logCommand("api", fmt.Sprintf("Fetching %d documents (%d cached)...", len(toFetch), len(combined)), "running")

	// Fetch uncached documents in parallel
	type result struct {
		path string
		data map[string]any
		err  error
	}

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

	// Add fetched results to combined and cache them
	for _, r := range results {
		if r.err != nil {
			g.logCommand("api", fmt.Sprintf("Error fetching %s: %v", r.path, r.err), "error")
		} else {
			combined[r.path] = r.data
			g.docCache[r.path] = r.data
		}
	}

	if len(combined) > 0 {
		g.currentDocData = combined
		g.currentDocStats = nil // No stats for combined multi-doc view
		g.currentDocPath = fmt.Sprintf("%d documents selected", len(combined))
		g.clearDetailsCache()
		g.logCommand("api", fmt.Sprintf("Loaded %d documents", len(combined)), "success")
	}

	// Stay in select mode - only Esc exits
	return g.Layout(g.g)
}

// Query builder action handlers

// doOpenQuery opens the query builder modal
func (g *Gui) doOpenQuery() error {
	return g.openQueryModal()
}

// queryClose closes the query modal
func (g *Gui) queryClose() error {
	return g.closeQueryModal()
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

// queryKeyJ handles j key in query modal (navigation only)
func (g *Gui) queryKeyJ() error {
	return g.queryMoveDown()
}

// queryKeyK handles k key in query modal (navigation only)
func (g *Gui) queryKeyK() error {
	return g.queryMoveUp()
}

// queryKeyH handles h key in query modal (navigation only)
func (g *Gui) queryKeyH() error {
	return g.queryMoveLeft()
}

// queryKeyL handles l key in query modal (navigation only)
func (g *Gui) queryKeyL() error {
	return g.queryMoveRight()
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

// queryBackspace is no longer needed - editable view handles it
func (g *Gui) queryBackspace() error {
	return nil
}

// queryInsertChar is no longer needed for text input - editable view handles it
// Only handles special action keys when not in edit mode
func (g *Gui) queryInsertChar(ch rune) func() error {
	return func() error {
		switch ch {
		case 'a':
			g.addQueryFilter()
			return g.Layout(g.g)
		case 'd':
			if g.queryActiveRow == queryRowFilters && len(g.queryFilters) > 0 {
				g.removeQueryFilter()
			}
			return g.Layout(g.g)
		}
		return nil
	}
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
				g.currentProject = projectID
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

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		g.logCommand("copy", "Clipboard not supported on this platform", "error")
		return nil
	}

	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
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

// doNextSearchMatch scrolls to next filter match in details
func (g *Gui) doNextSearchMatch() error {
	if g.currentColumn != "details" {
		return nil
	}
	filter := g.getDetailsFilter()
	if filter == "" || strings.HasPrefix(filter, ".") {
		return nil
	}
	if g.cachedDetailsLines == nil {
		return nil
	}

	lowerFilter := strings.ToLower(filter)
	startLine := g.detailsCursorLine + 1
	for i := 0; i < len(g.cachedDetailsLines); i++ {
		idx := (startLine + i) % len(g.cachedDetailsLines)
		if strings.Contains(strings.ToLower(g.cachedDetailsLines[idx]), lowerFilter) {
			g.detailsCursorLine = idx
			g.detailsScrollPos = idx
			return g.Layout(g.g)
		}
	}
	g.logCommand("search", "No more matches", "error")
	return nil
}

// doPrevSearchMatch scrolls to previous filter match in details
func (g *Gui) doPrevSearchMatch() error {
	if g.currentColumn != "details" {
		return nil
	}
	filter := g.getDetailsFilter()
	if filter == "" || strings.HasPrefix(filter, ".") {
		return nil
	}
	if g.cachedDetailsLines == nil {
		return nil
	}

	lowerFilter := strings.ToLower(filter)
	startLine := g.detailsCursorLine - 1
	if startLine < 0 {
		startLine = len(g.cachedDetailsLines) - 1
	}
	for i := 0; i < len(g.cachedDetailsLines); i++ {
		idx := (startLine - i + len(g.cachedDetailsLines)) % len(g.cachedDetailsLines)
		if strings.Contains(strings.ToLower(g.cachedDetailsLines[idx]), lowerFilter) {
			g.detailsCursorLine = idx
			g.detailsScrollPos = idx
			return g.Layout(g.g)
		}
	}
	g.logCommand("search", "No more matches", "error")
	return nil
}

// doCopyFieldValue copies the JSON value of the field at the current scroll position
func (g *Gui) doCopyFieldValue() error {
	if g.currentColumn != "details" || g.currentDocData == nil {
		return nil
	}
	if g.cachedDetailsLines == nil || len(g.cachedDetailsLines) == 0 {
		return nil
	}

	// Get the line at current scroll position
	lineIdx := g.detailsScrollPos
	if lineIdx >= len(g.cachedDetailsLines) {
		lineIdx = len(g.cachedDetailsLines) - 1
	}
	if lineIdx < 0 {
		lineIdx = 0
	}

	line := strings.TrimSpace(g.cachedDetailsLines[lineIdx])
	if line == "" || line == "{" || line == "}" || line == "[" || line == "]" {
		g.logCommand("copy", "No field value on this line", "error")
		return nil
	}

	// Extract value after the colon (for key: value lines)
	value := line
	if colonIdx := strings.Index(line, ": "); colonIdx >= 0 {
		value = strings.TrimSpace(line[colonIdx+2:])
		value = strings.TrimSuffix(value, ",")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		g.logCommand("copy", "Clipboard not supported", "error")
		return nil
	}
	cmd.Stdin = strings.NewReader(value)
	if err := cmd.Run(); err != nil {
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

// doFocusCommands focuses the command log panel
func (g *Gui) doFocusCommands() error {
	if g.modalOpen || g.helpOpen {
		return nil
	}
	g.modalOpen = true
	return g.Layout(g.g)
}

// doFastScrollDown scrolls details 5 lines down (J)
func (g *Gui) doFastScrollDown() error {
	if g.currentColumn == "details" {
		g.detailsScrollPos += 5
	}
	return g.Layout(g.g)
}

// doFastScrollUp scrolls details 5 lines up (K)
func (g *Gui) doFastScrollUp() error {
	if g.currentColumn == "details" {
		g.detailsScrollPos -= 5
		if g.detailsScrollPos < 0 {
			g.detailsScrollPos = 0
		}
	}
	return g.Layout(g.g)
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

// doToggleBase64Decode toggles inline base64 decoding for string values
func (g *Gui) doToggleBase64Decode() error {
	if g.currentColumn != "details" || g.currentDocData == nil {
		return nil
	}
	if g.cachedDetailsLines == nil || len(g.cachedDetailsLines) == 0 {
		return nil
	}

	lineIdx := g.detailsScrollPos
	if lineIdx >= len(g.cachedDetailsLines) {
		lineIdx = len(g.cachedDetailsLines) - 1
	}
	if lineIdx < 0 {
		lineIdx = 0
	}

	line := g.cachedDetailsLines[lineIdx]
	// Try to find a base64-encoded string value
	if idx := strings.Index(line, `": "`); idx >= 0 {
		rest := line[idx+4:]
		endIdx := strings.LastIndex(rest, `"`)
		if endIdx > 0 {
			val := rest[:endIdx]
			decoded, err := base64.StdEncoding.DecodeString(val)
			if err != nil {
				decoded, err = base64.RawStdEncoding.DecodeString(val)
			}
			if err == nil && len(decoded) > 0 {
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
		}
	}

	g.logCommand("base64", "No base64 string found on this line", "error")
	return nil
}

// pageUpCollections moves up N items in the collections panel based on current tab
func (g *Gui) pageUpCollections(n int) {
	switch g.collectionsTab {
	case "functions":
		g.selectedFunctionIdx -= n
		if g.selectedFunctionIdx < 0 {
			g.selectedFunctionIdx = 0
		}
	case "storage":
		if g.currentBucket == "" {
			g.selectedBucketIdx -= n
			if g.selectedBucketIdx < 0 {
				g.selectedBucketIdx = 0
			}
		} else {
			g.selectedObjectIdx -= n
			if g.selectedObjectIdx < 0 {
				g.selectedObjectIdx = 0
			}
		}
	case "auth":
		g.selectedAuthIdx -= n
		if g.selectedAuthIdx < 0 {
			g.selectedAuthIdx = 0
		}
	default:
		g.selectedCollectionIdx -= n
		if g.selectedCollectionIdx < 0 {
			g.selectedCollectionIdx = 0
		}
	}
}

// pageDownCollections moves down N items in the collections panel based on current tab
func (g *Gui) pageDownCollections(n int) {
	switch g.collectionsTab {
	case "functions":
		filtered := g.getFilteredFunctions()
		g.selectedFunctionIdx += n
		if g.selectedFunctionIdx >= len(filtered) {
			g.selectedFunctionIdx = len(filtered) - 1
		}
		if g.selectedFunctionIdx < 0 {
			g.selectedFunctionIdx = 0
		}
	case "storage":
		if g.currentBucket == "" {
			g.selectedBucketIdx += n
			if g.selectedBucketIdx >= len(g.storageBuckets) {
				g.selectedBucketIdx = len(g.storageBuckets) - 1
			}
			if g.selectedBucketIdx < 0 {
				g.selectedBucketIdx = 0
			}
		} else {
			g.selectedObjectIdx += n
			if g.selectedObjectIdx >= len(g.storageObjects) {
				g.selectedObjectIdx = len(g.storageObjects) - 1
			}
			if g.selectedObjectIdx < 0 {
				g.selectedObjectIdx = 0
			}
		}
	case "auth":
		g.selectedAuthIdx += n
		if g.selectedAuthIdx >= len(g.authUsers) {
			g.selectedAuthIdx = len(g.authUsers) - 1
		}
		if g.selectedAuthIdx < 0 {
			g.selectedAuthIdx = 0
		}
	default:
		filtered := g.getFilteredCollections()
		g.selectedCollectionIdx += n
		if g.selectedCollectionIdx >= len(filtered) {
			g.selectedCollectionIdx = len(filtered) - 1
		}
		if g.selectedCollectionIdx < 0 {
			g.selectedCollectionIdx = 0
		}
	}
}

// doSelectStorageItem handles space/enter for storage tab
func (g *Gui) doSelectStorageItem() error {
	if g.currentBucket == "" {
		// Select a bucket
		if g.selectedBucketIdx < len(g.storageBuckets) {
			g.currentBucket = g.storageBuckets[g.selectedBucketIdx].Name
			g.storagePrefix = ""
			g.storagePrefixStack = nil
			g.loadStorageObjects()
		}
	} else {
		// Navigate into folder or select object
		if g.selectedObjectIdx < len(g.storageObjects) {
			obj := g.storageObjects[g.selectedObjectIdx]
			if obj.IsPrefix {
				g.storagePrefixStack = append(g.storagePrefixStack, g.storagePrefix)
				g.storagePrefix = obj.Name
				g.loadStorageObjects()
			}
		}
	}
	return g.Layout(g.g)
}

// doStorageBack goes up one level in storage navigation (Esc or Backspace)
func (g *Gui) doStorageBack() error {
	if g.storagePrefix != "" {
		// Go up one folder level
		if len(g.storagePrefixStack) > 0 {
			g.storagePrefix = g.storagePrefixStack[len(g.storagePrefixStack)-1]
			g.storagePrefixStack = g.storagePrefixStack[:len(g.storagePrefixStack)-1]
		} else {
			g.storagePrefix = ""
		}
		g.loadStorageObjects()
	} else if g.currentBucket != "" {
		// Go back to bucket list
		g.currentBucket = ""
		g.storageObjects = nil
	}
	return g.Layout(g.g)
}
