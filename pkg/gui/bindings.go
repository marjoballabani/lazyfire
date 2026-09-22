package gui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jesseduffield/gocui"
)

// Context identifies what the user is interacting with. Panel contexts follow
// the focused panel (and the active tab inside the collections panel); popup
// contexts take precedence while a popup or prompt is open.
type Context string

const (
	CtxProjects    Context = "projects"
	CtxDatabases   Context = "databases"
	CtxCollections Context = "collections"
	CtxFunctions   Context = "functions"
	CtxStorage     Context = "storage"
	CtxAuth        Context = "auth"
	CtxRules       Context = "rules"
	CtxIndexes     Context = "indexes"
	CtxTree        Context = "tree"
	CtxDetails     Context = "details"
	CtxMenu        Context = "menu"
	CtxCommandLog  Context = "commandLog"
	CtxConfirm     Context = "confirm"
	CtxQuery       Context = "query"
	CtxQuerySelect Context = "querySelect"
	CtxQueryInput  Context = "queryInput"
	CtxFilter      Context = "filter"
)

// Contexts that share the collections panel, one per tab
var collectionsTabContexts = []Context{CtxCollections, CtxFunctions, CtxStorage, CtxAuth, CtxRules, CtxIndexes}

// tagNavigation groups a binding under "Navigation" in the keybindings menu
const tagNavigation = "navigation"

// Binding is a keybinding scoped to one or more contexts.
type Binding struct {
	Keys     []any     // gocui.Key or rune; all keys trigger the handler
	Contexts []Context // contexts the binding is active in; nil means global
	// When narrows the binding to a state within its contexts (e.g. select
	// mode in the tree). Bindings whose When returns false are skipped, so
	// another binding for the same key can take over.
	When    func() bool
	Handler func() error
	// Description is shown in the ? menu; bindings without one are hidden there
	Description string
	// Short is shown in the bottom options bar; empty keeps it off the bar
	Short string
	// ShortKey overrides the key label in the options bar (e.g. "j/k")
	ShortKey string
	Tag      string
	// AllowInPopup lets a global binding fire while a popup or prompt has focus
	AllowInPopup bool
	// GetDisabledReason returns why the binding can't run right now, or ""
	GetDisabledReason func() string
}

func (b *Binding) isActiveIn(ctx Context) bool {
	if b.When != nil && !b.When() {
		return false
	}
	if b.Contexts == nil {
		return true
	}
	for _, c := range b.Contexts {
		if c == ctx {
			return true
		}
	}
	return false
}

func (b *Binding) hasKey(key any) bool {
	for _, k := range b.Keys {
		if k == key {
			return true
		}
	}
	return false
}

func (b *Binding) disabledReason() string {
	if b.GetDisabledReason == nil {
		return ""
	}
	return b.GetDisabledReason()
}

// keyLabel joins the labels of all keys, e.g. "j/↓"
func (b *Binding) keyLabel() string {
	var labels []string
	for _, k := range b.Keys {
		// Backspace has two key codes with the same label
		if label := keyLabel(k); !slices.Contains(labels, label) {
			labels = append(labels, label)
		}
	}
	return strings.Join(labels, "/")
}

func (b *Binding) shortKeyLabel() string {
	if b.ShortKey != "" {
		return b.ShortKey
	}
	return keyLabel(b.Keys[0])
}

var keyLabels = map[gocui.Key]string{
	gocui.KeyArrowUp:    "↑",
	gocui.KeyArrowDown:  "↓",
	gocui.KeyArrowLeft:  "←",
	gocui.KeyArrowRight: "→",
	gocui.KeyEnter:      "enter",
	gocui.KeyEsc:        "esc",
	gocui.KeySpace:      "space",
	gocui.KeyTab:        "tab",
	gocui.KeyBacktab:    "shift+tab",
	gocui.KeyPgup:       "pgup",
	gocui.KeyPgdn:       "pgdn",
	gocui.KeyHome:       "home",
	gocui.KeyEnd:        "end",
	gocui.KeyCtrlC:      "ctrl+c",
	gocui.KeyCtrlD:      "ctrl+d",
	gocui.KeyCtrlU:      "ctrl+u",
	gocui.KeyBackspace:  "backspace",
	gocui.KeyBackspace2: "backspace",
}

func keyLabel(key any) string {
	switch k := key.(type) {
	case rune:
		return string(k)
	case gocui.Key:
		if label, ok := keyLabels[k]; ok {
			return label
		}
	}
	return fmt.Sprintf("%v", key)
}

// currentContext derives the active context from UI state. Layout focuses the
// matching view, so this is also the view gocui dispatches keys to.
func (g *Gui) currentContext() Context {
	switch {
	case g.confirmOpen:
		return CtxConfirm
	case g.querySelectOpen:
		return CtxQuerySelect
	case g.queryModalOpen && g.queryEditMode:
		return CtxQueryInput
	case g.queryModalOpen:
		return CtxQuery
	case g.filterInputActive:
		return CtxFilter
	case g.helpOpen:
		return CtxMenu
	case g.modalOpen:
		return CtxCommandLog
	}
	return g.panelContext(g.currentColumn)
}

// panelContext maps a panel to its context. The collections panel's context
// is its active tab; tab names match the context names.
func (g *Gui) panelContext(column string) Context {
	if column == "collections" {
		return Context(g.collectionsTab)
	}
	return Context(column)
}

// focusedPanel returns the context of the focused panel, ignoring popups
func (g *Gui) focusedPanel() Context {
	return g.panelContext(g.currentColumn)
}

func isPopupContext(ctx Context) bool {
	switch ctx {
	case CtxMenu, CtxCommandLog, CtxConfirm, CtxQuery, CtxQuerySelect, CtxQueryInput, CtxFilter:
		return true
	}
	return false
}

// contextView returns the gocui view that receives keys for a context
func (g *Gui) contextView(ctx Context) string {
	switch ctx {
	case CtxProjects:
		return g.views.projects
	case CtxDatabases:
		return g.views.databases
	case CtxCollections, CtxFunctions, CtxStorage, CtxAuth, CtxRules, CtxIndexes:
		return g.views.collections
	case CtxTree:
		return g.views.tree
	case CtxDetails:
		return g.views.details
	case CtxMenu:
		return g.views.helpModal
	case CtxCommandLog:
		return g.views.modal
	case CtxConfirm:
		return g.views.confirm
	case CtxQuery:
		return g.views.queryModal
	case CtxQuerySelect:
		return g.views.querySelect
	case CtxQueryInput:
		return g.views.queryInput
	case CtxFilter:
		return g.views.filterInput
	}
	return ""
}

func (g *Gui) setKeybindings() error {
	g.bindings = g.getBindings()

	for _, b := range g.bindings {
		if b.Contexts != nil {
			continue
		}
		for _, key := range b.Keys {
			if err := g.g.SetKeybinding("", key, gocui.ModNone, g.globalHandler(b)); err != nil {
				return err
			}
		}
	}

	// One gocui binding per (view, key). It resolves the binding for the
	// current context at press time, so e.g. space means "select project"
	// in projects and "open bucket" on the storage tab.
	type viewKey struct {
		view string
		key  any
	}
	registered := make(map[viewKey]bool)
	for _, b := range g.bindings {
		for _, ctx := range b.Contexts {
			for _, key := range b.Keys {
				vk := viewKey{g.contextView(ctx), key}
				if registered[vk] {
					continue
				}
				registered[vk] = true
				if err := g.g.SetKeybinding(vk.view, key, gocui.ModNone, g.contextHandler(key)); err != nil {
					return err
				}
			}
		}
	}

	return g.setMouseBindings()
}

func (g *Gui) globalHandler(b *Binding) func(*gocui.Gui, *gocui.View) error {
	return func(*gocui.Gui, *gocui.View) error {
		if isPopupContext(g.currentContext()) && !b.AllowInPopup {
			return nil
		}
		if b.When != nil && !b.When() {
			return nil
		}
		return g.runBinding(b)
	}
}

func (g *Gui) contextHandler(key any) func(*gocui.Gui, *gocui.View) error {
	return func(gui *gocui.Gui, v *gocui.View) error {
		if b := g.findBinding(g.currentContext(), key); b != nil {
			return g.runBinding(b)
		}
		// Nothing bound here in this context: fall back to a global binding.
		// Returning gocui.ErrKeybindingNotHandled instead would end the main
		// loop whenever gocui has no global binding of its own to try.
		for _, b := range g.bindings {
			if b.Contexts == nil && b.hasKey(key) {
				return g.globalHandler(b)(gui, v)
			}
		}
		return nil
	}
}

// findBinding returns the first context-specific binding for key that is
// active in ctx
func (g *Gui) findBinding(ctx Context, key any) *Binding {
	for _, b := range g.bindings {
		if b.Contexts != nil && b.hasKey(key) && b.isActiveIn(ctx) {
			return b
		}
	}
	return nil
}

// runBinding runs a binding's handler, or explains why it can't run
func (g *Gui) runBinding(b *Binding) error {
	if reason := b.disabledReason(); reason != "" {
		g.toast(reason, true)
		return nil
	}
	return b.Handler()
}

// contextBindings splits the bindings active in ctx the way the ? menu shows
// them: panel-specific, navigation, and global
func (g *Gui) contextBindings(ctx Context) (local, nav, global []*Binding) {
	for _, b := range g.bindings {
		if !b.isActiveIn(ctx) {
			continue
		}
		switch {
		case b.Contexts == nil:
			global = append(global, b)
		case b.Tag == tagNavigation:
			nav = append(nav, b)
		default:
			local = append(local, b)
		}
	}
	return local, nav, global
}
