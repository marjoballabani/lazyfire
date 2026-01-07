package gui

import (
	"fmt"

	"github.com/jesseduffield/gocui"
)

// Actions - clean handler functions without state checks.
// State checks are handled by the binding system's GetDisabledReason.

// doQuit exits the application
func (g *Gui) doQuit() error {
	return gocui.ErrQuit
}

// doEscape handles escape key - closes modals, cancels filter, or clears filter
func (g *Gui) doEscape() error {
	// Priority: help popup > command modal > filter input > committed filter
	if g.helpOpen {
		g.helpOpen = false
		g.helpPopup = nil
		return g.Layout(g.g)
	}
	if g.modalOpen {
		g.modalOpen = false
		return g.Layout(g.g)
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

// Filter char inserters for vim keys
func (g *Gui) filterInsertJ() error { return g.insertFilterChar(g.g, 'j') }
func (g *Gui) filterInsertK() error { return g.insertFilterChar(g.g, 'k') }
func (g *Gui) filterInsertH() error { return g.insertFilterChar(g.g, 'h') }
func (g *Gui) filterInsertL() error { return g.insertFilterChar(g.g, 'l') }

// doColumnLeft switches to the panel on the left
func (g *Gui) doColumnLeft() error {
	var newColumn string
	switch g.currentColumn {
	case "projects":
		newColumn = "details"
	case "collections":
		newColumn = "projects"
	case "tree":
		newColumn = "collections"
	case "details":
		newColumn = "tree"
	}
	return g.setFocus(g.g, newColumn)
}

// doColumnRight switches to the panel on the right
func (g *Gui) doColumnRight() error {
	var newColumn string
	switch g.currentColumn {
	case "projects":
		newColumn = "collections"
	case "collections":
		newColumn = "tree"
	case "tree":
		newColumn = "details"
	case "details":
		newColumn = "projects"
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

// doNextColumn cycles to the next panel
func (g *Gui) doNextColumn() error {
	var newColumn string
	switch g.currentColumn {
	case "projects":
		newColumn = "collections"
	case "collections":
		newColumn = "tree"
	case "tree":
		newColumn = "details"
	case "details":
		newColumn = "projects"
	}
	return g.setFocus(g.g, newColumn)
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
