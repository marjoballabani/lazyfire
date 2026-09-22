package gui

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/gocui"
)

// PopupItem represents an item in a popup list
type PopupItem struct {
	Key      string   // Shortcut key to display
	Label    string   // Item label/description
	IsHeader bool     // Headers are non-selectable section titles
	Binding  *Binding // Binding to run on Enter (optional)
}

// Popup represents a modal popup with selectable, filterable items
type Popup struct {
	Title       string
	Items       []PopupItem
	SelectedIdx int // index into Visible()
	Theme       *Theme
	filter      string
}

// NewPopup creates a new popup instance
func NewPopup(title string, items []PopupItem, theme *Theme) *Popup {
	p := &Popup{
		Title: title,
		Items: items,
		Theme: theme,
	}
	// Find first selectable item
	p.SelectedIdx = p.findNextSelectable(-1, 1)
	return p
}

func (p *Popup) Filter() string {
	return p.filter
}

// SetFilter narrows the visible items and selects the first match
func (p *Popup) SetFilter(filter string) {
	if filter == p.filter {
		return
	}
	p.filter = filter
	p.SelectedIdx = p.findNextSelectable(-1, 1)
}

// Visible returns the items matching the filter, keeping the header of each
// section that still has items
func (p *Popup) Visible() []PopupItem {
	if p.filter == "" {
		return p.Items
	}
	var visible []PopupItem
	var header *PopupItem
	for i := range p.Items {
		item := p.Items[i]
		if item.IsHeader {
			header = &p.Items[i]
			continue
		}
		if !item.matches(p.filter) {
			continue
		}
		if header != nil {
			visible = append(visible, *header)
			header = nil
		}
		visible = append(visible, item)
	}
	return visible
}

// matches reports whether the label contains filter or one of the keys is it
func (item PopupItem) matches(filter string) bool {
	for _, key := range strings.Split(item.Key, "/") {
		if key == filter {
			return true
		}
	}
	return MatchesFilter(item.Label, filter)
}

// findNextSelectable finds the next selectable item in the given direction
func (p *Popup) findNextSelectable(from int, direction int) int {
	visible := p.Visible()
	for i := from + direction; i >= 0 && i < len(visible); i += direction {
		if !visible[i].IsHeader {
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
	p.SelectedIdx = p.findNextSelectable(p.SelectedIdx, 1)
}

// GetSelectedItem returns the currently selected item
func (p *Popup) GetSelectedItem() *PopupItem {
	visible := p.Visible()
	if p.SelectedIdx >= 0 && p.SelectedIdx < len(visible) && !visible[p.SelectedIdx].IsHeader {
		return &visible[p.SelectedIdx]
	}
	return nil
}

// Render draws the popup content to the view using gocui's native highlighting
func (p *Popup) Render(v *gocui.View) {
	v.Clear()
	v.Highlight = true
	v.SelBgColor = p.Theme.SelectedLineBgColor
	v.SelFgColor = gocui.ColorDefault

	visible := p.Visible()
	for _, item := range visible {
		if item.IsHeader {
			fmt.Fprintf(v, "\033[36m ─── %s ───\033[0m\n", item.Label)
			continue
		}
		if item.Binding != nil {
			if reason := item.Binding.disabledReason(); reason != "" {
				// Disabled items stay selectable so enter explains why
				fmt.Fprintf(v, "  \033[90m%-14s %s (%s)\033[0m\n", item.Key, item.Label, reason)
				continue
			}
		}
		fmt.Fprintf(v, "  \033[33m%-14s\033[0m %s\n", item.Key, item.Label)
	}
	if len(visible) == 0 {
		fmt.Fprintf(v, "  \033[90mNo matching keybindings\033[0m\n")
	}

	if p.filter != "" {
		fmt.Fprintf(v, "\n\033[33m  filter: %s\033[0m", p.filter)
	}

	// Use FocusPoint to position the cursor and enable native highlighting
	v.FocusPoint(0, p.SelectedIdx, true)
}

// LineCount returns how many lines Render writes
func (p *Popup) LineCount() int {
	n := max(len(p.Visible()), 1)
	if p.filter != "" {
		n += 2
	}
	return n
}
