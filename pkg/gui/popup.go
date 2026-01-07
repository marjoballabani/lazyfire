package gui

import (
	"fmt"

	"github.com/jesseduffield/gocui"
)

// PopupItem represents an item in a popup list
type PopupItem struct {
	Key         string                                              // Shortcut key to display
	Label       string                                              // Item label/description
	IsHeader    bool                                                // Headers are non-selectable section titles
	Action      func(g *gocui.Gui, v *gocui.View) error             // Action to execute on Enter (optional)
}

// Popup represents a modal popup with selectable items
type Popup struct {
	Title       string
	Items       []PopupItem
	SelectedIdx int
	Theme       *Theme
	viewName    string
}

// NewPopup creates a new popup instance
func NewPopup(title string, items []PopupItem, theme *Theme, viewName string) *Popup {
	p := &Popup{
		Title:    title,
		Items:    items,
		Theme:    theme,
		viewName: viewName,
	}
	// Find first selectable item
	p.SelectedIdx = p.findNextSelectable(-1, 1)
	return p
}

// findNextSelectable finds the next selectable item in the given direction
func (p *Popup) findNextSelectable(from int, direction int) int {
	for i := from + direction; i >= 0 && i < len(p.Items); i += direction {
		if !p.Items[i].IsHeader {
			return i
		}
	}
	return from // Stay in place if no selectable found
}

// MoveUp moves selection up to the previous selectable item
func (p *Popup) MoveUp() {
	newIdx := p.findNextSelectable(p.SelectedIdx, -1)
	if newIdx >= 0 {
		p.SelectedIdx = newIdx
	}
}

// MoveDown moves selection down to the next selectable item
func (p *Popup) MoveDown() {
	newIdx := p.findNextSelectable(p.SelectedIdx, 1)
	if newIdx < len(p.Items) {
		p.SelectedIdx = newIdx
	}
}

// GetSelectedItem returns the currently selected item
func (p *Popup) GetSelectedItem() *PopupItem {
	if p.SelectedIdx >= 0 && p.SelectedIdx < len(p.Items) {
		return &p.Items[p.SelectedIdx]
	}
	return nil
}

// Render draws the popup content to the view using gocui's native highlighting
func (p *Popup) Render(v *gocui.View) {
	v.Clear()
	v.Highlight = true
	v.SelBgColor = p.Theme.SelectedLineBgColor
	v.SelFgColor = gocui.ColorDefault

	// Build display lines - each item gets its own line
	for _, item := range p.Items {
		if item.IsHeader {
			// Header line - cyan section divider
			fmt.Fprintf(v, "\033[36m ─── %s ───\033[0m\n", item.Label)
		} else {
			// Regular item - key in yellow, description in default
			fmt.Fprintf(v, "  \033[33m%-12s\033[0m %s\n", item.Key, item.Label)
		}
	}

	// Footer
	fmt.Fprintf(v, "\n\033[90m  Enter to execute · Esc to close\033[0m")

	// Use FocusPoint to position the cursor and enable native highlighting
	v.FocusPoint(0, p.SelectedIdx, true)
}

// SelectableCount returns the number of selectable (non-header) items
func (p *Popup) SelectableCount() int {
	count := 0
	for _, item := range p.Items {
		if !item.IsHeader {
			count++
		}
	}
	return count
}
