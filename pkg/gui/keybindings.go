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
				ContextConfirm:     g.doConfirmCancel,
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
		// Page Up / Page Down
		{
			Key:         gocui.KeyPgup,
			Handler:     g.doPageUp,
			Description: "Page up",
			Contexts: map[Context]func() error{
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.blockAction,
			},
		},
		{
			Key:         gocui.KeyPgdn,
			Handler:     g.doPageDown,
			Description: "Page down",
			Contexts: map[Context]func() error{
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.blockAction,
			},
		},
		// Home / End
		{
			Key:         gocui.KeyHome,
			Handler:     g.doGoToTop,
			Description: "Go to top",
			Contexts: map[Context]func() error{
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.blockAction,
			},
		},
		{
			Key:         gocui.KeyEnd,
			Handler:     g.doGoToBottom,
			Description: "Go to bottom",
			Contexts: map[Context]func() error{
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.blockAction,
			},
		},
		// Half-page scroll in details
		{
			Key:         gocui.KeyCtrlD,
			Handler:     g.doHalfPageDown,
			Description: "Half page down",
			Contexts: map[Context]func() error{
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.blockAction,
			},
		},
		{
			Key:         gocui.KeyCtrlU,
			Handler:     g.doHalfPageUp,
			Description: "Half page up",
			Contexts: map[Context]func() error{
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.blockAction,
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
		// [ and ] - cycle tabs backward/forward
		{
			Key:         '[',
			Handler:     g.doSwitchTabPrev,
			Description: "Previous tab",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterInsertBracketLeft,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('['),
			},
		},
		{
			Key:         ']',
			Handler:     g.doSwitchTabNext,
			Description: "Next tab",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterInsertBracketRight,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar(']'),
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
				ContextConfirm:     g.doConfirmAccept,
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
	// Exclude chars that have dedicated context-aware bindings: hjkl, csrqveFQgGpCtwx, ?@/, [], 123
	filterChars := "afouzEIPVWXYZ456789"
	filterChars += "-_. "
	filterChars += "|(){}:\"'`,<>=!+*^$#~;&%\\"
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
		// Go to top (g) / Go to bottom (G)
		{
			Key:         'g',
			Handler:     g.doGoToTop,
			Description: "Go to top",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('g'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('g'),
			},
		},
		{
			Key:         'G',
			Handler:     g.doGoToBottom,
			Description: "Go to bottom",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('G'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('G'),
			},
		},
		// Number keys to jump panels
		{
			Key:         '1',
			Handler:     g.doJumpToProjects,
			Description: "Jump to Projects",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('1'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('1'),
			},
		},
		{
			Key:         '2',
			Handler:     g.doJumpToCollections,
			Description: "Jump to Collections",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('2'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('2'),
			},
		},
		{
			Key:         '3',
			Handler:     g.doJumpToTree,
			Description: "Jump to Tree",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('3'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('3'),
			},
		},
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
			Key:         'S',
			Handler:     g.doScanCollections,
			Description: "Scan collections health",
			Contexts: map[Context]func() error{
				ContextFilter: g.filterInsertUpperS,
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('S'),
			},
		},
		// Copy document path
		{
			Key:         'p',
			Handler:     g.doCopyPath,
			Description: "Copy path",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('p'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('p'),
			},
		},
		// Collapse all tree nodes
		{
			Key:         'C',
			Handler:     g.doCollapseAll,
			Description: "Collapse all",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('C'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('C'),
			},
		},
		// Toggle compact JSON view
		{
			Key:         't',
			Handler:     g.doToggleCompactJSON,
			Description: "Toggle compact JSON",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('t'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('t'),
			},
		},
		// Toggle word wrap
		{
			Key:         'w',
			Handler:     g.doToggleWrap,
			Description: "Toggle word wrap",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('w'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('w'),
			},
		},
		// Clear cache (Shift+R)
		{
			Key:         'R',
			Handler:     g.doClearCache,
			Description: "Clear cache",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('R'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('R'),
			},
		},
		// Show cache statistics
		{
			Key:         'i',
			Handler:     g.doShowCacheStats,
			Description: "Cache stats",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('i'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('i'),
			},
		},
		// Toggle timestamp humanization
		{
			Key:         'T',
			Handler:     g.doToggleTimestamps,
			Description: "Toggle timestamps",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('T'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('T'),
			},
		},
		// Export cached documents
		{
			Key:         'x',
			Handler:     g.doExportCachedDocs,
			Description: "Export cached docs",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('x'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('x'),
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
		// Field size breakdown
		{
			Key:         'D',
			Handler:     g.doFieldSizeBreakdown,
			Description: "Field size breakdown",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('D'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('D'),
			},
		},
		// Field type analysis
		{
			Key:         'A',
			Handler:     g.doFieldTypeAnalysis,
			Description: "Field type analysis",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('A'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('A'),
			},
		},
		// Next search match
		{
			Key:         'n',
			Handler:     g.doNextSearchMatch,
			Description: "Next match",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('n'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('n'),
			},
		},
		// Previous search match
		{
			Key:         'N',
			Handler:     g.doPrevSearchMatch,
			Description: "Previous match",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('N'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('N'),
			},
		},
		// Copy field value
		{
			Key:         'y',
			Handler:     g.doCopyFieldValue,
			Description: "Copy field value",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('y'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('y'),
			},
		},
		// Focus commands panel
		{
			Key:         '0',
			Handler:     g.doFocusCommands,
			Description: "Command log",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('0'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('0'),
			},
		},
		// Fast scroll down in details (J)
		{
			Key:         'J',
			Handler:     g.doFastScrollDown,
			Description: "Scroll down 5 lines",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('J'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('J'),
			},
		},
		// Fast scroll up in details (K)
		{
			Key:         'K',
			Handler:     g.doFastScrollUp,
			Description: "Scroll up 5 lines",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('K'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('K'),
			},
		},
		// Toggle line numbers
		{
			Key:         'H',
			Handler:     g.doToggleLineNumbers,
			Description: "Toggle line numbers",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('H'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('H'),
			},
		},
		// Cycle log level filter
		{
			Key:         'L',
			Handler:     g.doCycleLogLevel,
			Description: "Cycle log level filter",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('L'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('L'),
			},
		},
		// Collection memory estimate
		{
			Key:         'M',
			Handler:     g.doCollectionMemoryEstimate,
			Description: "Collection memory",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('M'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('M'),
			},
		},
		// Base64 decode
		{
			Key:         'B',
			Handler:     g.doToggleBase64Decode,
			Description: "Decode base64",
			Contexts: map[Context]func() error{
				ContextFilter: g.makeFilterCharAction('B'),
				ContextHelp:   g.blockAction,
				ContextModal:  g.blockAction,
				ContextQuery:  g.queryInsertChar('B'),
			},
		},
		// Remaining keys for filter: b, d, m, o, u
		{
			Key:         'b',
			Handler:     g.makeFilterCharAction('b'),
			Contexts: map[Context]func() error{
				ContextQuery: g.queryInsertChar('b'),
			},
		},
		{
			Key:         'd',
			Handler:     g.makeFilterCharAction('d'),
			Contexts: map[Context]func() error{
				ContextQuery: g.queryInsertChar('d'),
			},
		},
		{
			Key:         'm',
			Handler:     g.makeFilterCharAction('m'),
			Contexts: map[Context]func() error{
				ContextQuery: g.queryInsertChar('m'),
			},
		},
		{
			Key:         'o',
			Handler:     g.makeFilterCharAction('o'),
			Contexts: map[Context]func() error{
				ContextQuery: g.queryInsertChar('o'),
			},
		},
		{
			Key:         'u',
			Handler:     g.makeFilterCharAction('u'),
			Contexts: map[Context]func() error{
				ContextQuery: g.queryInsertChar('u'),
			},
		},
		{
			Key:         'O',
			Handler:     g.makeFilterCharAction('O'),
			Contexts: map[Context]func() error{
				ContextQuery: g.queryInsertChar('O'),
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
