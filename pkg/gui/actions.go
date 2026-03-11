package gui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// doCursorUp moves selection up in current panel
func (g *Gui) doCursorUp() error {
	switch g.currentColumn {
	case "projects":
		if g.selectedProjectIndex > 0 {
			g.selectedProjectIndex--
			g.currentProjectInfo = nil
		}
	case "collections":
		if g.collectionsTab == "functions" {
			filtered := g.getFilteredFunctions()
			if g.selectedFunctionIdx > 0 && g.selectedFunctionIdx < len(filtered) {
				g.selectedFunctionIdx--
			}
		} else {
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
		if g.collectionsTab == "functions" {
			filtered := g.getFilteredFunctions()
			if g.selectedFunctionIdx < len(filtered)-1 {
				g.selectedFunctionIdx++
			}
		} else {
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
func (g *Gui) doSwitchTab() error {
	switch g.currentColumn {
	case "collections":
		// Switch Collections/Functions tabs (preserve state for both)
		if g.collectionsTab == "collections" {
			g.collectionsTab = "functions"
			if len(g.functions) == 0 && g.currentProject != "" {
				g.loadFunctions()
			}
		} else {
			g.collectionsTab = "collections"
			g.stopLogsRefresh()
			// Keep currentFunction and functionLogs - don't clear them
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
		if g.collectionsTab == "functions" {
			return g.selectFunction(g.g)
		}
		return g.selectCollection(g.g)
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
		if g.collectionsTab == "functions" {
			// Select function and go to details to see logs
			if err := g.selectFunction(g.g); err != nil {
				return err
			}
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
		if g.collectionsTab == "functions" {
			g.functionsFilter = ""
		} else {
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
