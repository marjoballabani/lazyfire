package gui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/itchyny/gojq"
	"github.com/jesseduffield/gocui"
	"github.com/marjoballabani/lazyfire/pkg/firebase"
)

func (g *Gui) commitFilter(gui *gocui.Gui) error {
	filterText := g.filterInputText
	panel := g.filterInputPanel

	// Save filter and exit input mode (filter stays active)
	switch panel {
	case "projects":
		g.projectsFilter = filterText
		g.selectedProjectIndex = 0 // Reset to first filtered item
	case "collections":
		if g.collectionsTab == "functions" {
			g.functionsFilter = filterText
			g.selectedFunctionIdx = 0
		} else {
			g.collectionsFilter = filterText
			g.selectedCollectionIdx = 0
		}
	case "tree":
		g.treeFilter = filterText
		g.selectedTreeIdx = 0
	case "details":
		g.detailsFilter = filterText
		g.detailsScrollPos = 0
	}

	// Exit input mode but keep filter active
	g.filterInputActive = false
	g.filterInputText = ""
	g.filterInputPanel = ""
	g.filterCursorPos = 0

	return g.Layout(gui)
}

func (g *Gui) isFilteringPanel(panel string) bool {
	return g.filterInputActive && g.filterInputPanel == panel
}

func (g *Gui) getFilterForPanel(panel string) string {
	switch panel {
	case "projects":
		return g.projectsFilter
	case "collections":
		if g.collectionsTab == "functions" {
			return g.functionsFilter
		}
		return g.collectionsFilter
	case "tree":
		return g.treeFilter
	case "details":
		return g.detailsFilter
	}
	return ""
}

func (g *Gui) hasActiveFilter(panel string) bool {
	return g.getFilterForPanel(panel) != ""
}

func (g *Gui) clearCurrentFilter(gui *gocui.Gui) error {
	switch g.currentColumn {
	case "projects":
		g.projectsFilter = ""
		g.selectedProjectIndex = 0
	case "collections":
		if g.collectionsTab == "functions" {
			g.functionsFilter = ""
			g.selectedFunctionIdx = 0
		} else {
			g.collectionsFilter = ""
			g.selectedCollectionIdx = 0
		}
	case "tree":
		g.treeFilter = ""
		g.selectedTreeIdx = 0
	case "details":
		g.detailsFilter = ""
		g.detailsScrollPos = 0
	}
	return g.Layout(gui)
}

func (g *Gui) cancelFilterInput(gui *gocui.Gui) error {
	g.filterInputActive = false
	g.filterInputText = ""
	g.filterInputPanel = ""
	g.filterCursorPos = 0
	return g.Layout(gui)
}

// insertFilterChar inserts a character at the cursor position
func (g *Gui) insertFilterChar(gui *gocui.Gui, ch rune) error {
	// Insert character at cursor position
	g.filterInputText = g.filterInputText[:g.filterCursorPos] + string(ch) + g.filterInputText[g.filterCursorPos:]
	g.filterCursorPos++
	return g.Layout(gui)
}

// matchesFilter checks if text contains the filter string (case-insensitive)
func (g *Gui) matchesFilter(text, filter string) bool {
	return MatchesFilter(text, filter)
}

// MatchesFilter checks if text contains the filter string (case-insensitive)
func MatchesFilter(text, filter string) bool {
	if filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(text), strings.ToLower(filter))
}

// getFilteredProjects returns projects matching the current filter
func (g *Gui) getFilteredProjects() []firebase.Project {
	// Use input text while typing, otherwise use committed filter
	filter := g.projectsFilter
	if g.filterInputActive && g.filterInputPanel == "projects" {
		filter = g.filterInputText
	}
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

// getFilteredCollections returns collections matching the current filter
func (g *Gui) getFilteredCollections() []firebase.Collection {
	filter := g.collectionsFilter
	if g.filterInputActive && g.filterInputPanel == "collections" {
		filter = g.filterInputText
	}
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
	filter := g.functionsFilter
	if g.filterInputActive && g.filterInputPanel == "collections" && g.collectionsTab == "functions" {
		filter = g.filterInputText
	}
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

// getFilteredTreeNodes returns tree nodes matching the current filter
func (g *Gui) getFilteredTreeNodes() []TreeNode {
	filter := g.treeFilter
	if g.filterInputActive && g.filterInputPanel == "tree" {
		filter = g.filterInputText
	}
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
	if g.filterInputActive && g.filterInputPanel == "details" {
		return g.filterInputText
	}
	return g.detailsFilter
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
	targetPath := filtered[filteredIdx].Path
	for i, node := range g.treeNodes {
		if node.Path == targetPath {
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
	if strings.HasPrefix(filter, ".") {
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
