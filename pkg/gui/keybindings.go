package gui

import "github.com/jesseduffield/gocui"

func (g *Gui) setKeybindings() error {
	km := g.newKeybindingManager()

	// Define all bindings
	km.RegisterAll(g.globalBindings(km))
	km.RegisterAll(g.navigationBindings(km))
	km.RegisterAll(g.filterBindings(km))
	km.RegisterAll(g.actionBindings(km))
	km.RegisterAll(g.mouseBindings())

	return km.Apply()
}

// globalBindings - always available (quit, escape, help)
func (g *Gui) globalBindings(km *KeybindingManager) []*Binding {
	return []*Binding{
		{
			Key:         gocui.KeyCtrlC,
			Handler:     g.doQuit,
			Description: "Force quit",
		},
		{
			Key:         'q',
			Handler:     g.doQuit,
			Description: "Quit",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterInsertQ,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('q'),
			},
		},
		{
			Key:         gocui.KeyEsc,
			Handler:     g.doEscape,
			Description: "Close/Cancel",
			Contexts: map[Context]func() error{
				ContextQuery:       g.queryClose,
				ContextQuerySelect: g.querySelectClose,
			},
		},
		{
			Key:         '?',
			Handler:     g.doToggleHelp,
			Description: "Show help",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterInsertQuestion,
				ContextQuery:  g.queryInsertChar('?'),
			},
		},
		{
			Key:         '@',
			Handler:     g.doToggleModal,
			Description: "Command log",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterInsertAt,
				ContextQuery:  g.queryInsertChar('@'),
			},
		},
	}
}

// navigationBindings - panel and list navigation
func (g *Gui) navigationBindings(km *KeybindingManager) []*Binding {
	return []*Binding{
		// Arrow up/down - context aware
		{
			Key:         gocui.KeyArrowUp,
			Handler:     g.doCursorUp,
			Description: "Move up",
			Contexts: map[Context]func() error{
				ContextHelp:        g.helpMoveUp,
				ContextModal:       g.blockAction,
				ContextSelect:      g.selectMoveUp,
				ContextQuery:       g.queryMoveUp,
				ContextQuerySelect: g.querySelectMoveUp,
			},
		},
		{
			Key:         gocui.KeyArrowDown,
			Handler:     g.doCursorDown,
			Description: "Move down",
			Contexts: map[Context]func() error{
				ContextHelp:        g.helpMoveDown,
				ContextModal:       g.blockAction,
				ContextSelect:      g.selectMoveDown,
				ContextQuery:       g.queryMoveDown,
				ContextQuerySelect: g.querySelectMoveDown,
			},
		},
		// Arrow left/right - context aware
		{
			Key:         gocui.KeyArrowLeft,
			Handler:     g.doColumnLeft,
			Description: "Move left",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterCursorLeft,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryMoveLeft,
			},
		},
		{
			Key:         gocui.KeyArrowRight,
			Handler:     g.doColumnRight,
			Description: "Move right",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterCursorRight,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryMoveRight,
			},
		},
		// Vim keys - context aware
		{
			Key:         'j',
			Handler:     g.doCursorDown,
			Description: "Move down",
			Contexts: map[Context]func() error{
				ContextFilter:      g.filterInsertJ,
				ContextHelp:        g.helpMoveDown,
				ContextModal:       g.blockAction,
				ContextSelect:      g.selectMoveDown,
				ContextQuery:       g.queryKeyJ,
				ContextQuerySelect: g.querySelectMoveDown,
			},
		},
		{
			Key:         'k',
			Handler:     g.doCursorUp,
			Description: "Move up",
			Contexts: map[Context]func() error{
				ContextFilter:      g.filterInsertK,
				ContextHelp:        g.helpMoveUp,
				ContextModal:       g.blockAction,
				ContextSelect:      g.selectMoveUp,
				ContextQuery:       g.queryKeyK,
				ContextQuerySelect: g.querySelectMoveUp,
			},
		},
		{
			Key:         'h',
			Handler:     g.doColumnLeft,
			Description: "Move left",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterInsertH,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryKeyH,
			},
		},
		{
			Key:         'l',
			Handler:     g.doColumnRight,
			Description: "Move right",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterInsertL,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryKeyL,
			},
		},
		// Tab
		{
			Key:         gocui.KeyTab,
			Handler:     g.doNextColumn,
			Description: "Next panel",
			Contexts: map[Context]func() error{
				ContextFilter: g.blockAction,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryNextField,
			},
		},
		// Space - context aware
		{
			Key:         gocui.KeySpace,
			Handler:     g.doSpace,
			Description: "Select/Expand",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterInsertSpace,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextSelect: g.doFetchSelectedDocs,
				ContextQuery:  g.blockAction,
			},
		},
		// Enter - context aware
		{
			Key:         gocui.KeyEnter,
			Handler:     g.doEnter,
			Description: "Confirm/Details",
			Contexts: map[Context]func() error{
				ContextFilter:      g.filterCommit,
				ContextHelp:        g.helpClose,
				ContextQuery:       g.queryEnter,
				ContextQuerySelect: g.querySelectConfirm,
			},
		},
	}
}

// filterBindings - filter mode specific
func (g *Gui) filterBindings(km *KeybindingManager) []*Binding {
	bindings := []*Binding{
		{
			Key:         '/',
			Handler:     g.doStartFilter,
			Description: "Start filter",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterInsertSlash,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('/'),
			},
		},
		{
			Key:     gocui.KeyBackspace,
			Handler: g.doFilterBackspace,
			Contexts: map[Context]func() error{
				ContextQuery: g.queryBackspace,
			},
		},
		{
			Key:     gocui.KeyBackspace2,
			Handler: g.doFilterBackspace,
			Contexts: map[Context]func() error{
				ContextQuery: g.queryBackspace,
			},
		},
	}

	// Character handlers for filter input (includes jq syntax chars)
	// Exclude chars that have dedicated context-aware bindings: hjkl, csrqveFQ, ?@/
	filterChars := "abdfgimnoptuwxyzABCDEGHIJKLMNOPRSTUVWXYZ0123456789"
	filterChars += "-_. "
	filterChars += "[]|(){}:\"'`,<>=!+*^$#~;&%\\"
	for _, ch := range filterChars {
		c := ch // capture for closure
		bindings = append(bindings, &Binding{
			Key:     c,
			Handler: g.makeFilterCharAction(c),
			Contexts: map[Context]func() error{
				ContextQuery: g.queryInsertChar(c),
			},
		})
	}

	return bindings
}

// actionBindings - document actions
func (g *Gui) actionBindings(km *KeybindingManager) []*Binding {
	return []*Binding{
		{
			Key:         'F',
			Handler:     g.doOpenQuery,
			Description: "Query builder",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterInsertUpperF,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('F'),
			},
		},
		{
			Key:         'c',
			Handler:     g.doCopyJSON,
			Description: "Copy JSON",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterInsertC,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('c'),
			},
		},
		{
			Key:         's',
			Handler:     g.doSaveJSON,
			Description: "Save JSON",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterInsertS,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('s'),
			},
		},
		{
			Key:         'r',
			Handler:     g.doRefresh,
			Description: "Refresh",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterInsertR,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('r'),
			},
		},
		{
			Key:         'v',
			Handler:     g.doToggleSelectMode,
			Description: "Select mode",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterInsertV,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextSelect: g.doToggleSelectMode, // Toggle off
				ContextQuery:  g.queryInsertChar('v'),
			},
		},
		{
			Key:         'e',
			Handler:     g.doEditInEditor,
			Description: "Edit in $EDITOR",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterInsertE,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('e'),
			},
		},
	}
}

// mouseBindings - click handlers
func (g *Gui) mouseBindings() []*Binding {
	return []*Binding{
		{Key: gocui.MouseLeft, ViewName: "helpModal", Handler: g.doHelpClick},
		{Key: gocui.MouseLeft, ViewName: "projects", Handler: g.doProjectsClick},
		{Key: gocui.MouseLeft, ViewName: "collections", Handler: g.doCollectionsClick},
		{Key: gocui.MouseLeft, ViewName: "tree", Handler: g.doTreeClick},
		{Key: gocui.MouseLeft, ViewName: "details", Handler: g.doDetailsClick},
		{Key: gocui.MouseLeft, ViewName: "commands", Handler: g.doOutsideClick},
		{Key: gocui.MouseLeft, ViewName: "help", Handler: g.doOutsideClick},
		{Key: gocui.MouseLeft, ViewName: "background", Handler: g.doOutsideClick},
	}
}
