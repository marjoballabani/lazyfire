package gui

import "github.com/jesseduffield/gocui"

// Context represents the current UI context/mode
type Context string

const (
	ContextNormal  Context = "normal"
	ContextFilter  Context = "filter"
	ContextHelp    Context = "help"
	ContextModal   Context = "modal"
	ContextSelect  Context = "select" // Visual selection mode
)

// Binding represents a keybinding with context-aware handling
type Binding struct {
	Key         interface{} // gocui.Key or rune
	Modifier    gocui.Modifier
	ViewName    string // Empty for global, specific view name otherwise
	Handler     func() error
	Description string
	// GetDisabledReason returns nil if enabled, or a reason string if disabled
	GetDisabledReason func() string
	// Contexts maps specific contexts to different handlers (optional)
	// If current context has a handler here, it's used instead of Handler
	Contexts map[Context]func() error
}

// BindingGroup represents a set of related keybindings
type BindingGroup struct {
	Name     string
	Bindings []*Binding
}

// Guards provides guard functions that wrap handlers with state checks
type Guards struct {
	NoPopup        func(func() error) func() error
	NoFilter       func(func() error) func() error
	NoPopupOrFilter func(func() error) func() error
}

// newGuards creates the guard functions for the GUI
func (g *Gui) newGuards() Guards {
	return Guards{
		NoPopup: func(f func() error) func() error {
			return func() error {
				if g.isModalOpen() {
					return nil
				}
				return f()
			}
		},
		NoFilter: func(f func() error) func() error {
			return func() error {
				if g.filterInputActive {
					return nil
				}
				return f()
			}
		},
		NoPopupOrFilter: func(f func() error) func() error {
			return func() error {
				if g.isModalOpen() || g.filterInputActive {
					return nil
				}
				return f()
			}
		},
	}
}

// DisabledReasons provides common disable-reason check functions
type DisabledReasons struct {
	PopupOpen   func() string
	FilterActive func() string
	NoDocument  func() string
}

// newDisabledReasons creates the disabled-reason check functions
func (g *Gui) newDisabledReasons() DisabledReasons {
	return DisabledReasons{
		PopupOpen: func() string {
			if g.isModalOpen() {
				return "Close popup first"
			}
			return ""
		},
		FilterActive: func() string {
			if g.filterInputActive {
				return "Exit filter mode first"
			}
			return ""
		},
		NoDocument: func() string {
			if g.currentDocData == nil {
				return "No document selected"
			}
			return ""
		},
	}
}

// require combines multiple disable-reason checks into one
// Returns first non-empty reason, or empty string if all pass
func require(checks ...func() string) func() string {
	return func() string {
		for _, check := range checks {
			if reason := check(); reason != "" {
				return reason
			}
		}
		return ""
	}
}

// getContext returns the current UI context
func (g *Gui) getContext() Context {
	if g.helpOpen {
		return ContextHelp
	}
	if g.modalOpen {
		return ContextModal
	}
	if g.filterInputActive {
		return ContextFilter
	}
	if g.selectMode {
		return ContextSelect
	}
	return ContextNormal
}

// KeybindingManager handles registration and execution of keybindings
type KeybindingManager struct {
	gui      *Gui
	bindings []*Binding
	guards   Guards
	disabled DisabledReasons
}

// newKeybindingManager creates a new keybinding manager
func (g *Gui) newKeybindingManager() *KeybindingManager {
	return &KeybindingManager{
		gui:      g,
		bindings: make([]*Binding, 0),
		guards:   g.newGuards(),
		disabled: g.newDisabledReasons(),
	}
}

// Register adds a binding to the manager
func (km *KeybindingManager) Register(b *Binding) {
	km.bindings = append(km.bindings, b)
}

// RegisterAll adds multiple bindings
func (km *KeybindingManager) RegisterAll(bindings []*Binding) {
	km.bindings = append(km.bindings, bindings...)
}

// Apply registers all bindings with gocui
func (km *KeybindingManager) Apply() error {
	for _, b := range km.bindings {
		handler := km.wrapHandler(b)

		var err error
		switch key := b.Key.(type) {
		case gocui.Key:
			err = km.gui.g.SetKeybinding(b.ViewName, key, b.Modifier, handler)
		case rune:
			err = km.gui.g.SetKeybinding(b.ViewName, key, b.Modifier, handler)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// wrapHandler creates a gocui-compatible handler that checks context and disabled state
func (km *KeybindingManager) wrapHandler(b *Binding) func(*gocui.Gui, *gocui.View) error {
	return func(gui *gocui.Gui, v *gocui.View) error {
		// Check for context-specific handler first
		ctx := km.gui.getContext()
		if b.Contexts != nil {
			if contextHandler, ok := b.Contexts[ctx]; ok {
				return contextHandler()
			}
		}

		// Check if binding is disabled for this context
		if b.GetDisabledReason != nil {
			if reason := b.GetDisabledReason(); reason != "" {
				// Could show a toast/notification here in future
				return nil
			}
		}
		return b.Handler()
	}
}
