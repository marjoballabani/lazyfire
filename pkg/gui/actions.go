package gui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/jesseduffield/gocui"
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
func (g *Gui) filterInsertV() error         { return g.insertFilterChar(g.g, 'v') }
func (g *Gui) filterInsertE() error         { return g.insertFilterChar(g.g, 'e') }
func (g *Gui) filterInsertSlash() error     { return g.insertFilterChar(g.g, '/') }

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
		if g.selectedCollectionIdx > 0 {
			g.selectedCollectionIdx--
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
		filtered := g.getFilteredCollections()
		if g.selectedCollectionIdx < len(filtered)-1 {
			g.selectedCollectionIdx++
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

// doNextColumn - Tab goes to details panel (keeps existing content)
func (g *Gui) doNextColumn() error {
	if g.currentColumn == "details" {
		return nil // Already in details, do nothing
	}
	g.previousColumn = g.currentColumn
	return g.setFocus(g.g, "details")
}

// doSpace handles space key - select/expand in current panel
// doSpace - normal mode space handler
func (g *Gui) doSpace() error {
	switch g.currentColumn {
	case "projects":
		return g.selectProject(g.g)
	case "collections":
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
		g.collectionsFilter = ""
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
	return g.copyJSONAction()
}

// doSaveJSON saves current document to file
func (g *Gui) doSaveJSON() error {
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

// doRefresh reloads all data
func (g *Gui) doRefresh() error {
	g.logCommand("r", "Refreshing...", "running")

	if err := g.loadProjects(); err != nil {
		g.logCommand("r", fmt.Sprintf("Failed: %v", err), "error")
		return g.Layout(g.g)
	}

	g.collections = nil
	g.treeNodes = nil
	g.currentDocData = nil
	g.currentDocPath = ""
	g.currentProjectInfo = nil
	g.selectedProjectIndex = 0
	g.selectedCollectionIdx = 0
	g.selectedTreeIdx = 0

	g.logCommand("r", fmt.Sprintf("Loaded %d projects", len(g.projects)), "success")
	return g.Layout(g.g)
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
