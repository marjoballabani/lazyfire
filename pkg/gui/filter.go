package gui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/itchyny/gojq"
	"github.com/jesseduffield/gocui"
	"github.com/marjoballabani/lazyfire/pkg/firebase"
)

// doStartFilter opens the filter prompt for the focused panel (or the
// keybindings menu). Typing filters live; enter keeps the filter, esc drops it.
func (g *Gui) doStartFilter() error {
	ctx := g.currentContext()
	g.setCommittedFilter(ctx, "")
	g.filterInputActive = true
	g.filterInputPanel = ctx
	g.filterInputText = ""
	g.resetFilterSelection(ctx)
	return g.relayout()
}

// filterEditor handles keys typed into the filter prompt
func (g *Gui) filterEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) bool {
	switch key {
	case gocui.KeyEnter:
		g.commitFilter()
		return true
	case gocui.KeyEsc:
		g.cancelFilterInput()
		return true
	case gocui.KeyTab:
		return true
	case gocui.KeyArrowUp:
		g.moveFilterTarget(-1)
		return true
	case gocui.KeyArrowDown:
		g.moveFilterTarget(1)
		return true
	}

	matched := gocui.DefaultEditor.Edit(v, key, ch, mod)
	if text := v.TextArea.GetContent(); text != g.filterInputText {
		g.filterInputText = text
		g.resetFilterSelection(g.filterInputPanel)
	}
	return matched
}

// moveFilterTarget moves the selection in the list being filtered, so the
// arrow keys pick a match without leaving the prompt
func (g *Gui) moveFilterTarget(delta int) {
	ctx := g.filterInputPanel
	if ctx == CtxMenu {
		if g.helpPopup == nil {
			return
		}
		if delta < 0 {
			g.helpPopup.MoveUp()
		} else {
			g.helpPopup.MoveDown()
		}
		return
	}
	if ctx == CtxDetails {
		g.setDetailsCursor(g.detailsCursor + delta)
		return
	}
	if idx, n := g.listState(ctx); idx != nil && n > 0 {
		g.setSelection(ctx, max(0, min(*idx+delta, n-1)))
	}
}

// commitFilter keeps the typed filter and closes the prompt
func (g *Gui) commitFilter() {
	g.setCommittedFilter(g.filterInputPanel, g.filterInputText)
	g.closeFilterInput()
}

// cancelFilterInput closes the prompt without keeping a filter
func (g *Gui) cancelFilterInput() {
	ctx := g.filterInputPanel
	g.closeFilterInput()
	g.setCommittedFilter(ctx, "")
	g.resetFilterSelection(ctx)
}

func (g *Gui) closeFilterInput() {
	g.filterInputActive = false
	g.filterInputText = ""
	g.filterInputPanel = ""
	_ = g.relayout()
}

// doClearFilter removes the committed filter of the focused panel, keeping
// the selected item selected
func (g *Gui) doClearFilter() error {
	ctx := g.currentContext()
	key := g.selectedItemKey(ctx)
	g.setCommittedFilter(ctx, "")
	g.selectItemByKey(ctx, key)
	return nil
}

// resetFilterSelection moves the selection of ctx back to the top, used
// whenever the set of visible items changes under a filter
func (g *Gui) resetFilterSelection(ctx Context) {
	switch ctx {
	case CtxMenu:
		if g.helpPopup != nil {
			g.helpPopup.SetFilter(g.activeFilter(CtxMenu))
		}
	case CtxDetails:
		g.detailsCursor = 0
		g.detailsScrollPos = 0
	default:
		if idx, _ := g.listState(ctx); idx != nil {
			*idx = 0
		}
	}
}

// getCommittedFilter returns the filter kept for ctx after the prompt closed
func (g *Gui) getCommittedFilter(ctx Context) string {
	switch ctx {
	case CtxProjects:
		return g.projectsFilter
	case CtxDatabases:
		return g.databasesFilter
	case CtxCollections:
		return g.collectionsFilter
	case CtxFunctions:
		return g.functionsFilter
	case CtxStorage:
		return g.storageFilter
	case CtxAuth:
		return g.authFilter
	case CtxTree:
		return g.treeFilter
	case CtxDetails:
		return g.detailsFilter
	case CtxMenu:
		if g.helpPopup != nil {
			return g.helpPopup.Filter()
		}
	}
	return ""
}

func (g *Gui) setCommittedFilter(ctx Context, filter string) {
	switch ctx {
	case CtxProjects:
		g.projectsFilter = filter
	case CtxDatabases:
		g.databasesFilter = filter
	case CtxCollections:
		g.collectionsFilter = filter
	case CtxFunctions:
		g.functionsFilter = filter
	case CtxStorage:
		g.storageFilter = filter
	case CtxAuth:
		g.authFilter = filter
	case CtxTree:
		g.treeFilter = filter
	case CtxDetails:
		g.detailsFilter = filter
		g.detailsCursor = 0
		g.detailsScrollPos = 0
	case CtxMenu:
		if g.helpPopup != nil {
			g.helpPopup.SetFilter(filter)
		}
	}
}

// activeFilter returns the text being typed for ctx, or its committed filter
func (g *Gui) activeFilter(ctx Context) string {
	if g.filterInputActive && g.filterInputPanel == ctx {
		return g.filterInputText
	}
	return g.getCommittedFilter(ctx)
}

// selectedItemKey returns a stable identifier for the selected item of a list
func (g *Gui) selectedItemKey(ctx Context) string {
	idx, n := g.listState(ctx)
	if idx == nil || *idx < 0 || *idx >= n {
		return ""
	}
	i := *idx
	switch ctx {
	case CtxProjects:
		return g.getFilteredProjects()[i].ID
	case CtxDatabases:
		return g.getFilteredDatabases()[i].ID
	case CtxCollections:
		return g.getFilteredCollections()[i].Name
	case CtxFunctions:
		return g.getFilteredFunctions()[i].Name
	case CtxStorage:
		if g.currentBucket == "" {
			return g.getFilteredBuckets()[i].Name
		}
		return g.getFilteredObjects()[i].Name
	case CtxAuth:
		return g.getFilteredAuthUsers()[i].UID
	case CtxTree:
		return g.getFilteredTreeNodes()[i].Path
	}
	return ""
}

// selectItemByKey selects the item identified by key, or the first item
func (g *Gui) selectItemByKey(ctx Context, key string) {
	idx, n := g.listState(ctx)
	if idx == nil {
		return
	}
	*idx = 0
	for i := 0; i < n; i++ {
		*idx = i
		if g.selectedItemKey(ctx) == key {
			return
		}
	}
	*idx = 0
}

// MatchesFilter checks if text contains the filter string (case-insensitive)
func MatchesFilter(text, filter string) bool {
	if filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(text), strings.ToLower(filter))
}

func (g *Gui) matchesFilter(text, filter string) bool {
	return MatchesFilter(text, filter)
}

// getFilteredProjects returns projects matching the current filter
func (g *Gui) getFilteredProjects() []firebase.Project {
	filter := g.activeFilter(CtxProjects)
	if filter == "" {
		return g.projects
	}
	var filtered []firebase.Project
	for _, p := range g.projects {
		if g.matchesFilter(p.DisplayName, filter) || g.matchesFilter(p.ID, filter) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// getFilteredDatabases returns databases matching the current filter
func (g *Gui) getFilteredDatabases() []firebase.Database {
	filter := g.activeFilter(CtxDatabases)
	if filter == "" {
		return g.databases
	}
	var filtered []firebase.Database
	for _, db := range g.databases {
		if g.matchesFilter(db.ID, filter) || g.matchesFilter(db.LocationID, filter) {
			filtered = append(filtered, db)
		}
	}
	return filtered
}

// getFilteredCollections returns collections matching the current filter
func (g *Gui) getFilteredCollections() []firebase.Collection {
	filter := g.activeFilter(CtxCollections)
	if filter == "" {
		return g.collections
	}
	var filtered []firebase.Collection
	for _, c := range g.collections {
		if g.matchesFilter(c.Name, filter) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// getFilteredFunctions returns functions matching the current filter
func (g *Gui) getFilteredFunctions() []firebase.CloudFunction {
	filter := g.activeFilter(CtxFunctions)
	if filter == "" {
		return g.functions
	}
	var filtered []firebase.CloudFunction
	for _, f := range g.functions {
		if g.matchesFilter(f.DisplayName, filter) || g.matchesFilter(f.Region, filter) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// getFilteredBuckets returns storage buckets matching the current filter
func (g *Gui) getFilteredBuckets() []firebase.StorageBucket {
	filter := g.activeFilter(CtxStorage)
	if filter == "" {
		return g.storageBuckets
	}
	var filtered []firebase.StorageBucket
	for _, b := range g.storageBuckets {
		if g.matchesFilter(b.Name, filter) {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

// getFilteredObjects returns objects in the current folder matching the filter
func (g *Gui) getFilteredObjects() []firebase.StorageObject {
	filter := g.activeFilter(CtxStorage)
	if filter == "" {
		return g.storageObjects
	}
	var filtered []firebase.StorageObject
	for _, o := range g.storageObjects {
		if g.matchesFilter(o.DisplayName, filter) {
			filtered = append(filtered, o)
		}
	}
	return filtered
}

// getFilteredAuthUsers returns auth users matching the current filter
func (g *Gui) getFilteredAuthUsers() []firebase.AuthUser {
	filter := g.activeFilter(CtxAuth)
	if filter == "" {
		return g.authUsers
	}
	var filtered []firebase.AuthUser
	for _, u := range g.authUsers {
		if g.matchesFilter(u.Email, filter) || g.matchesFilter(u.UID, filter) || g.matchesFilter(u.DisplayName, filter) {
			filtered = append(filtered, u)
		}
	}
	return filtered
}

// getFilteredTreeNodes returns tree nodes matching the current filter
func (g *Gui) getFilteredTreeNodes() []TreeNode {
	filter := g.activeFilter(CtxTree)
	if filter == "" {
		return g.treeNodes
	}
	var filtered []TreeNode
	for _, n := range g.treeNodes {
		if g.matchesFilter(n.Name, filter) || g.matchesFilter(n.Path, filter) {
			filtered = append(filtered, n)
		}
	}
	return filtered
}

// getDetailsFilter returns the active filter for details panel
func (g *Gui) getDetailsFilter() string {
	return g.activeFilter(CtxDetails)
}

// isJqQuery reports whether a details filter is a jq query rather than text
func isJqQuery(filter string) bool {
	return strings.HasPrefix(filter, ".")
}

// highlightMatches wraps matching text in reverse video ANSI codes
func highlightMatches(text, filter string) string {
	if filter == "" {
		return text
	}
	lowerText := strings.ToLower(text)
	lowerFilter := strings.ToLower(filter)

	var result strings.Builder
	i := 0
	for i < len(text) {
		idx := strings.Index(lowerText[i:], lowerFilter)
		if idx == -1 {
			result.WriteString(text[i:])
			break
		}
		// Write text before match
		result.WriteString(text[i : i+idx])
		// Write highlighted match
		result.WriteString("\033[7m") // Reverse video
		result.WriteString(text[i+idx : i+idx+len(filter)])
		result.WriteString("\033[27m") // Reset reverse
		i = i + idx + len(filter)
	}
	return result.String()
}

// getOriginalTreeNodeIndex maps a filtered index back to the original treeNodes index
func (g *Gui) getOriginalTreeNodeIndex(filteredIdx int) int {
	filtered := g.getFilteredTreeNodes()
	if filteredIdx < 0 || filteredIdx >= len(filtered) {
		return -1
	}
	return g.treeNodeIndex(filtered[filteredIdx].Path)
}

// treeNodeIndex returns the index of the node with path in treeNodes, or -1
func (g *Gui) treeNodeIndex(path string) int {
	for i, node := range g.treeNodes {
		if node.Path == path {
			return i
		}
	}
	return -1
}

// renderFilteredDetails shows only JSON lines that match the filter
// If filter starts with "." it's treated as a jq query
func (g *Gui) renderFilteredDetails(v *gocui.View) {
	filter := g.getDetailsFilter()

	// If filter starts with ".", treat as jq query
	if isJqQuery(filter) {
		g.renderJqFilteredDetails(v, filter)
		return
	}

	// Otherwise, do line-based string matching
	data, err := json.MarshalIndent(g.currentDocData, "", "  ")
	if err != nil {
		v.SetContent(fmt.Sprintf("Error formatting data: %v\n", err))
		return
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("\033[36m─── %s (filtered) ───\033[0m\n\n", g.currentDocPath))

	lines := strings.Split(string(data), "\n")
	matchCount := 0
	for _, line := range lines {
		if g.matchesFilter(line, filter) {
			colored := colorizeLine(line)
			// Highlight matching text with reverse video
			colored = highlightMatches(colored, filter)
			content.WriteString(colored)
			content.WriteString("\n")
			matchCount++
		}
	}

	if matchCount == 0 {
		content.WriteString("\033[90mNo matching lines\033[0m\n")
	}

	v.SetContent(content.String())
}

// renderJqFilteredDetails applies a jq query to the document
func (g *Gui) renderJqFilteredDetails(v *gocui.View, query string) {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("\033[36m─── %s (jq: %s) ───\033[0m\n\n", g.currentDocPath, query))

	// Parse jq query
	jqQuery, err := gojq.Parse(query)
	if err != nil {
		content.WriteString(fmt.Sprintf("\033[31mjq parse error: %v\033[0m\n", err))
		v.SetContent(content.String())
		return
	}

	// Run query
	iter := jqQuery.Run(g.currentDocData)
	hasResults := false

	for {
		result, ok := iter.Next()
		if !ok {
			break
		}

		if err, isErr := result.(error); isErr {
			content.WriteString(fmt.Sprintf("\033[31mjq error: %v\033[0m\n", err))
			break
		}

		hasResults = true
		// Format result as JSON
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			content.WriteString(fmt.Sprintf("%v\n", result))
		} else {
			content.WriteString(colorizeJSON(string(data)))
			content.WriteString("\n")
		}
	}

	if !hasResults {
		content.WriteString("\033[90mnull\033[0m\n")
	}

	v.SetContent(content.String())
}
