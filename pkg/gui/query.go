package gui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jesseduffield/gocui"
	"github.com/marjoballabani/lazyfire/pkg/firebase"
)

// Query builder row indices
const (
	queryRowFilters = iota
	queryRowOrderBy
	queryRowLimit
	queryRowButtons
)

// Available operators for query filters
var queryOperators = []string{"==", "!=", "<", "<=", ">", ">=", "in", "not-in", "array-contains", "array-contains-any"}

// Available value types for query filters
// For "in", "not-in", "array-contains-any" use array types
var queryValueTypes = []string{"auto", "string", "integer", "double", "boolean", "null", "array"}

// openQueryModal opens the query builder modal.
func (g *Gui) openQueryModal() error {
	// Determine collection path based on current panel
	collectionPath := ""
	nodeIdx := -1 // -1 means top-level query (replace whole tree)

	if g.currentColumn == "tree" {
		// Check if selected node is a collection
		filtered := g.getFilteredTreeNodes()
		if g.selectedTreeIdx < len(filtered) {
			node := filtered[g.selectedTreeIdx]
			if node.Type == "collection" {
				collectionPath = node.Path
				// Find the original index in unfiltered tree
				nodeIdx = g.getOriginalTreeNodeIndex(g.selectedTreeIdx)
			}
		}
	}

	// Fall back to current collection from collections panel
	if collectionPath == "" {
		collectionPath = g.currentCollection
		nodeIdx = -1
	}

	if collectionPath == "" {
		g.logCommand("F", "No collection selected", "error")
		return nil
	}

	g.queryCollection = collectionPath
	g.queryNodeIdx = nodeIdx
	g.queryModalOpen = true
	g.queryActiveRow = queryRowFilters
	g.queryActiveCol = 0
	g.queryEditMode = false
	g.queryEditBuffer = ""

	// Initialize with defaults if empty
	if g.queryLimit == 0 {
		g.queryLimit = 50
	}
	if g.queryOrderDir == "" {
		g.queryOrderDir = "ASC"
	}

	g.logCommand("F", fmt.Sprintf("Query: %s", collectionPath), "success")
	return nil
}

// queryInputEditor handles text input in the query input view.
func (g *Gui) queryInputEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) bool {
	switch key {
	case gocui.KeyEnter:
		// Commit the edit
		g.commitQueryEditFromView(v)
		return true
	case gocui.KeyEsc:
		// Cancel edit
		g.queryEditMode = false
		return true
	default:
		// Let default editor handle other keys
		return gocui.DefaultEditor.Edit(v, key, ch, mod)
	}
}

// getQueryEditFieldName returns the name of the field being edited.
func (g *Gui) getQueryEditFieldName() string {
	switch g.queryActiveRow {
	case queryRowFilters:
		if len(g.queryFilters) > 0 {
			idx := g.queryActiveCol / 4 // 4 columns per filter
			col := g.queryActiveCol % 4
			switch col {
			case 0:
				return fmt.Sprintf("Filter %d Field", idx+1)
			case 3:
				return fmt.Sprintf("Filter %d Value", idx+1)
			}
		}
		return "Filter"
	case queryRowOrderBy:
		return "Order By Field"
	case queryRowLimit:
		return "Limit"
	}
	return "Input"
}

// commitQueryEditFromView commits the edit from the editable view.
func (g *Gui) commitQueryEditFromView(v *gocui.View) {
	content := strings.TrimSpace(v.TextArea.GetContent())
	g.queryEditMode = false

	switch g.queryActiveRow {
	case queryRowFilters:
		if len(g.queryFilters) > 0 {
			idx := g.queryActiveCol / 4 // 4 columns per filter
			col := g.queryActiveCol % 4
			if idx < len(g.queryFilters) {
				switch col {
				case 0: // field
					g.queryFilters[idx].Field = content
				case 3: // value
					g.queryFilters[idx].Value = content // Store as string, type conversion happens at query time
				}
			}
		}

	case queryRowOrderBy:
		g.queryOrderBy = content

	case queryRowLimit:
		if limit, err := strconv.Atoi(content); err == nil && limit > 0 {
			g.queryLimit = limit
		}
	}
}

// closeQueryModal closes the query builder without executing.
func (g *Gui) closeQueryModal() error {
	g.queryModalOpen = false
	g.queryEditMode = false
	return nil
}

// clearQuery resets all query filters and options.
func (g *Gui) clearQuery() error {
	g.queryFilters = nil
	g.queryOrderBy = ""
	g.queryOrderDir = ""
	g.queryLimit = 50
	g.queryActiveRow = queryRowFilters
	g.queryActiveCol = 0
	g.queryResultMode = false
	return g.Layout(g.g)
}

// executeQuery runs the query and displays results in the tree.
func (g *Gui) executeQuery() error {
	if g.queryCollection == "" {
		return nil
	}

	g.queryModalOpen = false
	g.treeLoading = true
	g.logCommand("query", fmt.Sprintf("Query on %s...", g.queryCollection), "running")

	collectionPath := g.queryCollection
	nodeIdx := g.queryNodeIdx
	go func() {
		opts := firebase.QueryOptions{
			Filters:  g.queryFilters,
			OrderBy:  g.queryOrderBy,
			OrderDir: g.queryOrderDir,
			Limit:    g.queryLimit,
		}

		docs, err := g.firebaseClient.RunQuery(collectionPath, opts)

		g.g.Update(func(gui *gocui.Gui) error {
			g.treeLoading = false

			if err != nil {
				g.logCommand("query", fmt.Sprintf("Error: %v", err), "error")
				return nil
			}

			// Cache documents
			for _, doc := range docs {
				g.docCache[doc.Path] = doc.Data
			}

			if nodeIdx == -1 {
				// Top-level query: replace entire tree
				g.queryResultMode = true
				g.treeNodes = nil
				for _, doc := range docs {
					g.treeNodes = append(g.treeNodes, TreeNode{
						Path:        doc.Path,
						Name:        doc.ID,
						Type:        "document",
						Depth:       0,
						HasChildren: true,
						Expanded:    false,
					})
				}
				g.selectedTreeIdx = 0
			} else {
				// Subcollection query: insert results under the collection node
				if nodeIdx < len(g.treeNodes) {
					parentNode := g.treeNodes[nodeIdx]
					parentDepth := parentNode.Depth

					// First collapse any existing children
					g.collapseNode(nodeIdx)

					// Build new nodes for query results
					newChildren := make([]TreeNode, 0, len(docs))
					for _, doc := range docs {
						newChildren = append(newChildren, TreeNode{
							Path:        doc.Path,
							Name:        doc.ID,
							Type:        "document",
							Depth:       parentDepth + 1,
							HasChildren: true,
							Expanded:    false,
						})
					}

					// Insert children after parent node
					if len(newChildren) > 0 {
						newNodes := make([]TreeNode, 0, len(g.treeNodes)+len(newChildren))
						newNodes = append(newNodes, g.treeNodes[:nodeIdx+1]...)
						newNodes = append(newNodes, newChildren...)
						newNodes = append(newNodes, g.treeNodes[nodeIdx+1:]...)
						g.treeNodes = newNodes
						g.treeNodes[nodeIdx].Expanded = true
					}

					g.selectedTreeIdx = nodeIdx + 1
				}
			}

			g.logCommand("query", fmt.Sprintf("Found %d documents", len(docs)), "success")
			return nil
		})
	}()

	return nil
}

// addQueryFilter adds a new empty filter to the query.
func (g *Gui) addQueryFilter() {
	g.queryFilters = append(g.queryFilters, firebase.QueryFilter{
		Field:     "",
		Operator:  "==",
		Value:     "",
		ValueType: "auto",
	})
	g.queryActiveRow = queryRowFilters
	g.queryActiveCol = (len(g.queryFilters) - 1) * 4 // 4 columns per filter: field, op, type, value
}

// removeQueryFilter removes the currently selected filter.
func (g *Gui) removeQueryFilter() {
	if len(g.queryFilters) == 0 {
		return
	}
	idx := g.queryActiveCol / 4 // 4 columns per filter
	if idx >= len(g.queryFilters) {
		idx = len(g.queryFilters) - 1
	}
	g.queryFilters = append(g.queryFilters[:idx], g.queryFilters[idx+1:]...)
	// Adjust column to stay in bounds
	maxCol := g.getMaxColForRow()
	if g.queryActiveCol > maxCol {
		g.queryActiveCol = maxCol
	}
}


// handleQueryEnter handles Enter key in query modal.
func (g *Gui) handleQueryEnter() error {
	switch g.queryActiveRow {
	case queryRowFilters:
		if len(g.queryFilters) == 0 {
			g.addQueryFilter()
			return nil
		}
		// Start editing filter field
		g.startQueryEdit()

	case queryRowOrderBy:
		g.startQueryEdit()

	case queryRowLimit:
		g.startQueryEdit()

	case queryRowButtons:
		if g.queryActiveCol == 0 {
			return g.executeQuery()
		} else {
			return g.clearQuery()
		}
	}

	return nil
}

// startQueryEdit starts editing the currently selected field.
// For text fields: sets queryEditMode=true and stores initial value in queryEditBuffer.
// For operators/types: opens a selection popup.
func (g *Gui) startQueryEdit() {
	switch g.queryActiveRow {
	case queryRowFilters:
		if len(g.queryFilters) > 0 {
			idx := g.queryActiveCol / 4 // Each filter has 4 columns: field, operator, type, value
			col := g.queryActiveCol % 4
			if idx < len(g.queryFilters) {
				switch col {
				case 0: // field - text edit
					g.queryEditBuffer = g.queryFilters[idx].Field
					g.queryEditMode = true
				case 1: // operator - open select popup
					g.openQuerySelect(queryOperators, g.queryFilters[idx].Operator, func(selected string) {
						g.queryFilters[idx].Operator = selected
					})
				case 2: // type - open select popup
					g.openQuerySelect(queryValueTypes, g.queryFilters[idx].ValueType, func(selected string) {
						g.queryFilters[idx].ValueType = selected
					})
				case 3: // value - text edit
					if s, ok := g.queryFilters[idx].Value.(string); ok {
						g.queryEditBuffer = s
					} else {
						g.queryEditBuffer = fmt.Sprintf("%v", g.queryFilters[idx].Value)
					}
					g.queryEditMode = true
				}
			}
		}

	case queryRowOrderBy:
		if g.queryActiveCol == 0 {
			g.queryEditBuffer = g.queryOrderBy
			g.queryEditMode = true
		} else {
			// direction - open select popup
			g.openQuerySelect([]string{"ASC", "DESC"}, g.queryOrderDir, func(selected string) {
				g.queryOrderDir = selected
			})
		}

	case queryRowLimit:
		g.queryEditBuffer = strconv.Itoa(g.queryLimit)
		g.queryEditMode = true
	}
}

// openQuerySelect opens the selection popup with given items.
func (g *Gui) openQuerySelect(items []string, current string, callback func(string)) {
	g.querySelectItems = items
	g.querySelectCallback = callback
	g.querySelectOpen = true

	// Find current item index
	g.querySelectIdx = 0
	for i, item := range items {
		if item == current {
			g.querySelectIdx = i
			break
		}
	}
}

// closeQuerySelect closes the selection popup without selecting.
func (g *Gui) closeQuerySelect() {
	g.querySelectOpen = false
	g.querySelectItems = nil
	g.querySelectCallback = nil
}

// confirmQuerySelect confirms the selection and closes the popup.
func (g *Gui) confirmQuerySelect() {
	if g.querySelectCallback != nil && g.querySelectIdx < len(g.querySelectItems) {
		g.querySelectCallback(g.querySelectItems[g.querySelectIdx])
	}
	g.closeQuerySelect()
}

// renderQuerySelect renders the selection popup content.
func (g *Gui) renderQuerySelect(v *gocui.View) {
	v.Clear()

	for _, item := range g.querySelectItems {
		fmt.Fprintf(v, " %s\n", item)
	}

	// Set cursor to selected item (gocui handles highlighting)
	v.SetCursor(0, g.querySelectIdx)
}

// getMaxColForRow returns the maximum column index for the current row.
func (g *Gui) getMaxColForRow() int {
	switch g.queryActiveRow {
	case queryRowFilters:
		if len(g.queryFilters) == 0 {
			return 0
		}
		return len(g.queryFilters)*4 - 1 // field, operator, type, value for each filter

	case queryRowOrderBy:
		return 1 // field, direction

	case queryRowLimit:
		return 0

	case queryRowButtons:
		return 1 // Execute, Clear
	}
	return 0
}

// renderQueryModal renders the query builder modal.
func (g *Gui) renderQueryModal(v *gocui.View) {
	v.Clear()

	activeColor := g.getActiveColorCode()
	resetColor := "\033[0m"
	dimColor := "\033[90m"
	cyanColor := "\033[36m"
	yellowColor := "\033[33m"
	highlightBg := g.theme.GetSelectedBgAnsiCode()

	// Collection name
	fmt.Fprintf(v, " %sCollection:%s %s\n\n", dimColor, resetColor, g.queryCollection)

	// WHERE section
	whereLabel := "WHERE:"
	if g.queryActiveRow == queryRowFilters && !g.queryEditMode {
		whereLabel = fmt.Sprintf("%sWHERE:%s", activeColor, resetColor)
	}
	fmt.Fprintf(v, " %s\n", whereLabel)

	if len(g.queryFilters) == 0 {
		hint := "(a) add filter"
		if g.queryActiveRow == queryRowFilters {
			hint = fmt.Sprintf("%s %s %s", highlightBg, hint, resetColor)
		} else {
			hint = fmt.Sprintf("%s%s%s", dimColor, hint, resetColor)
		}
		fmt.Fprintf(v, "   %s\n", hint)
	} else {
		for i, f := range g.queryFilters {
			fieldStr := f.Field
			if fieldStr == "" {
				fieldStr = "field"
			}
			opStr := f.Operator
			typeStr := f.ValueType
			if typeStr == "" {
				typeStr = "auto"
			}
			valueStr := fmt.Sprintf("%v", f.Value)
			if valueStr == "" {
				valueStr = "value"
			}

			// Format operator in brackets with cyan color
			opDisplay := fmt.Sprintf("%s[%s]%s", cyanColor, opStr, resetColor)
			// Format type in parentheses with yellow/dim color
			typeDisplay := fmt.Sprintf("%s(%s)%s", yellowColor, typeStr, resetColor)

			// Highlight selected parts (4 columns: field, op, type, value)
			if g.queryActiveRow == queryRowFilters && !g.queryEditMode {
				baseIdx := i * 4
				if g.queryActiveCol == baseIdx {
					fieldStr = fmt.Sprintf("%s %s %s", highlightBg, fieldStr, resetColor)
				}
				if g.queryActiveCol == baseIdx+1 {
					opDisplay = fmt.Sprintf("%s [%s] %s", highlightBg, opStr, resetColor)
				}
				if g.queryActiveCol == baseIdx+2 {
					typeDisplay = fmt.Sprintf("%s (%s) %s", highlightBg, typeStr, resetColor)
				}
				if g.queryActiveCol == baseIdx+3 {
					valueStr = fmt.Sprintf("%s %s %s", highlightBg, valueStr, resetColor)
				}
			}

			fmt.Fprintf(v, "   %s %s %s %s\n", fieldStr, opDisplay, typeDisplay, valueStr)
		}
	}
	fmt.Fprintln(v)

	// ORDER BY section
	orderLabel := "ORDER BY:"
	if g.queryActiveRow == queryRowOrderBy && !g.queryEditMode {
		orderLabel = fmt.Sprintf("%sORDER BY:%s", activeColor, resetColor)
	}
	orderByStr := g.queryOrderBy
	if orderByStr == "" {
		orderByStr = "field"
	}
	// Format direction in brackets with cyan color
	dirDisplay := fmt.Sprintf("%s[%s]%s", cyanColor, g.queryOrderDir, resetColor)

	if g.queryActiveRow == queryRowOrderBy && !g.queryEditMode {
		if g.queryActiveCol == 0 {
			orderByStr = fmt.Sprintf("%s %s %s", highlightBg, orderByStr, resetColor)
		}
		if g.queryActiveCol == 1 {
			dirDisplay = fmt.Sprintf("%s [%s] %s", highlightBg, g.queryOrderDir, resetColor)
		}
	}
	fmt.Fprintf(v, " %s  %s  %s\n\n", orderLabel, orderByStr, dirDisplay)

	// LIMIT section
	limitLabel := "LIMIT:"
	if g.queryActiveRow == queryRowLimit && !g.queryEditMode {
		limitLabel = fmt.Sprintf("%sLIMIT:%s", activeColor, resetColor)
	}
	limitStr := strconv.Itoa(g.queryLimit)
	if g.queryActiveRow == queryRowLimit && !g.queryEditMode {
		limitStr = fmt.Sprintf("%s %s %s", highlightBg, limitStr, resetColor)
	}
	fmt.Fprintf(v, " %s  %s\n\n", limitLabel, limitStr)

	// Buttons: Execute, Clear
	execBtn := "Execute"
	clearBtn := "Clear"
	if g.queryActiveRow == queryRowButtons && !g.queryEditMode {
		if g.queryActiveCol == 0 {
			execBtn = fmt.Sprintf("%s Execute %s", highlightBg, resetColor)
		} else {
			clearBtn = fmt.Sprintf("%s Clear %s", highlightBg, resetColor)
		}
	}
	fmt.Fprintf(v, " [ %s ]  [ %s ]\n\n", execBtn, clearBtn)

	// Help
	fmt.Fprintf(v, "%s ─────────────────────────────────────%s\n", dimColor, resetColor)
	if g.queryEditMode {
		fmt.Fprintf(v, "%s Enter: confirm  Esc: cancel%s\n", dimColor, resetColor)
	} else {
		fmt.Fprintf(v, "%s j/k: rows  h/l: cols  Enter: edit%s\n", dimColor, resetColor)
		fmt.Fprintf(v, "%s a: add filter  d: delete  Esc: close%s\n", dimColor, resetColor)
	}
}
