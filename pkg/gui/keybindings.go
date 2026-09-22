package gui

import "github.com/jesseduffield/gocui"

// getBindings defines every keybinding. Order matters: for a given key and
// context the first active binding wins, so state-specific bindings (those
// with When) come before the general ones they override.
func (g *Gui) getBindings() []*Binding {
	sidePanels := []Context{CtxProjects, CtxDatabases, CtxCollections, CtxFunctions, CtxStorage, CtxAuth, CtxRules, CtxIndexes, CtxTree}
	panels := append(append([]Context{}, sidePanels...), CtxDetails)
	filterable := []Context{CtxProjects, CtxDatabases, CtxCollections, CtxFunctions, CtxStorage, CtxAuth, CtxTree, CtxDetails}
	isSelectMode := func() bool { return g.selectMode }
	isInBucket := func() bool { return g.currentBucket != "" }
	isFunctionDetails := func() bool { return g.detailsSource() == "function" }
	hasFilterHere := func() bool { return g.getCommittedFilter(g.focusedPanel()) != "" }

	return []*Binding{
		// Global
		{Keys: []any{gocui.KeyCtrlC}, Handler: g.doQuit, Description: "Quit", AllowInPopup: true},
		{Keys: []any{'q'}, Handler: g.doQuit, Description: "Quit", Short: "quit"},
		{Keys: []any{'?'}, Handler: g.doToggleHelp, Description: "Keybindings", Short: "help"},
		{Keys: []any{'@'}, Handler: g.doToggleModal, Description: "Command log"},
		{Keys: []any{'1'}, Handler: g.jumpTo("projects"), Description: "Focus projects"},
		{Keys: []any{'2'}, Handler: g.jumpTo("databases"), Description: "Focus databases"},
		{Keys: []any{'3'}, Handler: g.jumpTo("collections"), Description: "Focus collections"},
		{Keys: []any{'4'}, Handler: g.jumpTo("tree"), Description: "Focus tree"},
		{Keys: []any{'0'}, Handler: g.focusDetails, Description: "Focus details"},
		{Keys: []any{'T'}, Handler: g.doToggleTimestamps, Description: "Toggle human-readable timestamps"},
		{Keys: []any{'x'}, Handler: g.doExportCachedDocs, Description: "Export cached documents to ~/Downloads", GetDisabledReason: g.noCachedDocsReason},
		{Keys: []any{'A'}, Handler: g.doFieldTypeAnalysis, Description: "Field type analysis (cached docs)", GetDisabledReason: g.noCollectionReason},
		{Keys: []any{'M'}, Handler: g.doCollectionMemoryEstimate, Description: "Collection memory estimate", GetDisabledReason: g.noCollectionReason},
		{Keys: []any{'i'}, Handler: g.doShowCacheStats, Description: "Cache statistics"},
		{Keys: []any{'R'}, Handler: g.doClearCache, Description: "Clear cache"},

		// Navigation
		{Keys: []any{'k', gocui.KeyArrowUp}, Contexts: panels, Tag: tagNavigation, Handler: func() error { return g.moveBy(-1) }, Description: "Move up"},
		{Keys: []any{'j', gocui.KeyArrowDown}, Contexts: panels, Tag: tagNavigation, Handler: func() error { return g.moveBy(1) }, Description: "Move down", Short: "move", ShortKey: "j/k"},
		{Keys: []any{gocui.KeyPgup}, Contexts: panels, Tag: tagNavigation, Handler: func() error { return g.moveBy(-g.pageSize()) }, Description: "Page up"},
		{Keys: []any{gocui.KeyPgdn}, Contexts: panels, Tag: tagNavigation, Handler: func() error { return g.moveBy(g.pageSize()) }, Description: "Page down"},
		{Keys: []any{gocui.KeyCtrlU}, Contexts: panels, Tag: tagNavigation, Handler: func() error { return g.moveBy(-g.halfPageSize()) }, Description: "Half page up"},
		{Keys: []any{gocui.KeyCtrlD}, Contexts: panels, Tag: tagNavigation, Handler: func() error { return g.moveBy(g.halfPageSize()) }, Description: "Half page down"},
		{Keys: []any{'K'}, Contexts: []Context{CtxDetails}, Tag: tagNavigation, Handler: func() error { return g.moveBy(-5) }, Description: "Up 5 lines"},
		{Keys: []any{'J'}, Contexts: []Context{CtxDetails}, Tag: tagNavigation, Handler: func() error { return g.moveBy(5) }, Description: "Down 5 lines"},
		{Keys: []any{'g', gocui.KeyHome}, Contexts: panels, Tag: tagNavigation, Handler: g.moveToTop, Description: "Go to top"},
		{Keys: []any{'G', gocui.KeyEnd}, Contexts: panels, Tag: tagNavigation, Handler: g.moveToBottom, Description: "Go to bottom"},
		{Keys: []any{'h', gocui.KeyArrowLeft, gocui.KeyBacktab}, Contexts: sidePanels, Tag: tagNavigation, Handler: g.doColumnLeft, Description: "Previous panel"},
		{Keys: []any{'l', gocui.KeyArrowRight, gocui.KeyTab}, Contexts: sidePanels, Tag: tagNavigation, Handler: g.doColumnRight, Description: "Next panel", Short: "panels", ShortKey: "tab"},
		{Keys: []any{gocui.KeyTab, gocui.KeyBacktab}, Contexts: []Context{CtxDetails}, Tag: tagNavigation, Handler: g.doBackFromDetails, Description: "Back to previous panel"},
		{Keys: []any{'['}, Contexts: collectionsTabContexts, Tag: tagNavigation, Handler: g.doSwitchTabPrev, Description: "Previous tab"},
		{Keys: []any{']'}, Contexts: collectionsTabContexts, Tag: tagNavigation, Handler: g.doSwitchTabNext, Description: "Next tab", Short: "tabs", ShortKey: "[/]"},
		{Keys: []any{'['}, Contexts: []Context{CtxDetails}, When: isFunctionDetails, Tag: tagNavigation, Handler: g.doSwitchTabPrev, Description: "Previous tab"},
		{Keys: []any{']'}, Contexts: []Context{CtxDetails}, When: isFunctionDetails, Tag: tagNavigation, Handler: g.doSwitchTabNext, Description: "Next tab", Short: "details/logs", ShortKey: "[/]"},

		// Escape: select mode, then filter, then panel-specific back
		{Keys: []any{gocui.KeyEsc}, Contexts: []Context{CtxTree}, When: isSelectMode, Handler: g.doExitSelectMode, Description: "Exit select mode", Short: "cancel"},
		{Keys: []any{gocui.KeyEsc}, Contexts: filterable, When: hasFilterHere, Handler: g.doClearFilter, Description: "Clear filter", Short: "clear filter"},
		{Keys: []any{gocui.KeyEsc}, Contexts: []Context{CtxDetails}, Handler: g.doBackFromDetails, Description: "Back to previous panel", Short: "back"},
		{Keys: []any{gocui.KeyEsc, gocui.KeyBackspace, gocui.KeyBackspace2}, Contexts: []Context{CtxStorage}, When: isInBucket, Handler: g.doStorageBack, Description: "Go up one level", Short: "back"},

		// Projects
		{Keys: []any{gocui.KeySpace}, Contexts: []Context{CtxProjects}, Handler: g.selectProject, Description: "Select project", Short: "select", GetDisabledReason: g.noItemReason},
		{Keys: []any{gocui.KeyEnter}, Contexts: []Context{CtxProjects}, Handler: g.fetchProjectDetails, Description: "Show project details", GetDisabledReason: g.noItemReason},
		{Keys: []any{'S'}, Contexts: []Context{CtxProjects}, Handler: g.doScanCollections, Description: "Scan collections health", Short: "scan", GetDisabledReason: g.scanDisabledReason},

		// Databases
		{Keys: []any{gocui.KeySpace}, Contexts: []Context{CtxDatabases}, Handler: g.selectDatabase, Description: "Use database", Short: "select", GetDisabledReason: g.databaseDisabledReason},
		{Keys: []any{gocui.KeyEnter}, Contexts: []Context{CtxDatabases}, Handler: g.openDatabase, Description: "Use database and focus collections", GetDisabledReason: g.databaseDisabledReason},

		// Collections tab
		{Keys: []any{gocui.KeySpace}, Contexts: []Context{CtxCollections}, Handler: g.selectCollection, Description: "Open collection", Short: "open", GetDisabledReason: g.noItemReason},
		{Keys: []any{gocui.KeyEnter}, Contexts: []Context{CtxCollections}, Handler: g.openCollection, Description: "Open collection and focus tree", GetDisabledReason: g.noItemReason},

		// Functions tab
		{Keys: []any{gocui.KeySpace}, Contexts: []Context{CtxFunctions}, Handler: g.selectFunction, Description: "Select function", Short: "select", GetDisabledReason: g.noItemReason},
		{Keys: []any{gocui.KeyEnter}, Contexts: []Context{CtxFunctions}, Handler: g.openFunction, Description: "Open function details", GetDisabledReason: g.noItemReason},
		{Keys: []any{'L'}, Contexts: []Context{CtxFunctions, CtxDetails}, When: isFunctionDetails, Handler: g.doCycleLogLevel, Description: "Cycle log level filter"},

		// Storage tab
		{Keys: []any{gocui.KeySpace, gocui.KeyEnter}, Contexts: []Context{CtxStorage}, Handler: g.doSelectStorageItem, Description: "Open bucket / folder", Short: "open", GetDisabledReason: g.noItemReason},

		// Auth tab
		{Keys: []any{gocui.KeySpace, gocui.KeyEnter}, Contexts: []Context{CtxAuth}, Handler: g.focusDetails, Description: "View user in details", Short: "details", GetDisabledReason: g.noItemReason},

		// Rules / Indexes tabs
		{Keys: []any{gocui.KeyEnter}, Contexts: []Context{CtxRules, CtxIndexes}, Handler: g.focusDetails, Description: "View in details", Short: "view"},

		// Tree
		{Keys: []any{gocui.KeySpace}, Contexts: []Context{CtxTree}, When: isSelectMode, Handler: g.doFetchSelectedDocs, Description: "Fetch selected documents", Short: "fetch"},
		{Keys: []any{gocui.KeySpace}, Contexts: []Context{CtxTree}, Handler: g.selectTreeNode, Description: "Expand / collapse", Short: "expand", GetDisabledReason: g.noItemReason},
		{Keys: []any{gocui.KeyEnter}, Contexts: []Context{CtxTree}, Handler: g.openTreeNode, Description: "Open document / expand collection", Short: "open", GetDisabledReason: g.noItemReason},
		{Keys: []any{'v'}, Contexts: []Context{CtxTree}, Handler: g.doToggleSelectMode, Description: "Toggle select mode", Short: "select"},
		{Keys: []any{'C'}, Contexts: []Context{CtxTree}, Handler: g.doCollapseAll, Description: "Collapse all"},
		{Keys: []any{'Q'}, Contexts: []Context{CtxTree}, Handler: g.doClearQueryResults, Description: "Clear query results", Short: "clear query", GetDisabledReason: g.noQueryResultsReason},

		// Shared panel actions
		{Keys: []any{'/'}, Contexts: filterable, Handler: g.doStartFilter, Description: "Filter", Short: "filter", GetDisabledReason: g.filterDisabledReason},
		{Keys: []any{'r'}, Contexts: panels, Handler: g.doRefresh, Description: "Refresh", Short: "refresh", GetDisabledReason: g.refreshDisabledReason},
		{Keys: []any{'F'}, Contexts: []Context{CtxCollections, CtxTree}, Handler: g.doOpenQuery, Description: "Query builder", Short: "query", GetDisabledReason: g.noCollectionForQueryReason},
		{Keys: []any{'c'}, Contexts: []Context{CtxTree, CtxDetails}, Handler: g.doCopyJSON, Description: "Copy JSON to clipboard", Short: "copy", GetDisabledReason: g.copyDisabledReason},
		{Keys: []any{'s'}, Contexts: []Context{CtxTree, CtxDetails}, Handler: g.doSaveJSON, Description: "Save JSON to ~/Downloads", GetDisabledReason: g.copyDisabledReason},
		{Keys: []any{'p'}, Contexts: []Context{CtxTree, CtxDetails}, Handler: g.doCopyPath, Description: "Copy path to clipboard", GetDisabledReason: g.copyPathDisabledReason},

		// Details
		{Keys: []any{'y'}, Contexts: []Context{CtxDetails}, Handler: g.doCopyFieldValue, Description: "Copy value on cursor line", Short: "copy value"},
		{Keys: []any{'B'}, Contexts: []Context{CtxDetails}, Handler: g.doToggleBase64Decode, Description: "Decode base64 on cursor line"},
		{Keys: []any{'n'}, Contexts: []Context{CtxDetails}, Handler: g.doNextSearchMatch, Description: "Next match", GetDisabledReason: g.searchDisabledReason},
		{Keys: []any{'N'}, Contexts: []Context{CtxDetails}, Handler: g.doPrevSearchMatch, Description: "Previous match", GetDisabledReason: g.searchDisabledReason},
		{Keys: []any{'t'}, Contexts: []Context{CtxDetails}, Handler: g.doToggleCompactJSON, Description: "Toggle compact JSON", Short: "compact", GetDisabledReason: g.noDocumentReason},
		{Keys: []any{'w'}, Contexts: []Context{CtxDetails}, Handler: g.doToggleWrap, Description: "Toggle word wrap", Short: "wrap"},
		{Keys: []any{'H'}, Contexts: []Context{CtxDetails}, Handler: g.doToggleLineNumbers, Description: "Toggle line numbers", GetDisabledReason: g.noDocumentReason},
		{Keys: []any{'D'}, Contexts: []Context{CtxDetails}, Handler: g.doFieldSizeBreakdown, Description: "Field size breakdown", GetDisabledReason: g.noDocumentReason},
		{Keys: []any{'e'}, Contexts: []Context{CtxDetails}, Handler: g.doEditInEditor, Description: "Open in $EDITOR", GetDisabledReason: g.noDocumentReason},

		// Keybindings menu
		{Keys: []any{'k', gocui.KeyArrowUp}, Contexts: []Context{CtxMenu}, Handler: g.helpMoveUp},
		{Keys: []any{'j', gocui.KeyArrowDown}, Contexts: []Context{CtxMenu}, Handler: g.helpMoveDown},
		{Keys: []any{gocui.KeyEnter}, Contexts: []Context{CtxMenu}, Handler: g.helpExecute, Short: "execute"},
		{Keys: []any{'/'}, Contexts: []Context{CtxMenu}, Handler: g.doStartFilter, Short: "filter"},
		{Keys: []any{gocui.KeyEsc, 'q', '?'}, Contexts: []Context{CtxMenu}, Handler: g.helpEscape, Short: "close"},

		// Command log
		{Keys: []any{gocui.KeyEsc, 'q', '@'}, Contexts: []Context{CtxCommandLog}, Handler: g.doToggleModal, Short: "close"},

		// Confirm dialog
		{Keys: []any{gocui.KeyEnter, 'y'}, Contexts: []Context{CtxConfirm}, Handler: g.doConfirmAccept, Short: "confirm"},
		{Keys: []any{gocui.KeyEsc, 'n', 'q'}, Contexts: []Context{CtxConfirm}, Handler: g.doConfirmCancel, Short: "cancel"},

		// Query builder
		{Keys: []any{'k', gocui.KeyArrowUp}, Contexts: []Context{CtxQuery}, Handler: g.queryMoveUp},
		{Keys: []any{'j', gocui.KeyArrowDown}, Contexts: []Context{CtxQuery}, Handler: g.queryMoveDown},
		{Keys: []any{'h', gocui.KeyArrowLeft}, Contexts: []Context{CtxQuery}, Handler: g.queryMoveLeft},
		{Keys: []any{'l', gocui.KeyArrowRight}, Contexts: []Context{CtxQuery}, Handler: g.queryMoveRight},
		{Keys: []any{gocui.KeyTab}, Contexts: []Context{CtxQuery}, Handler: g.queryNextField},
		{Keys: []any{gocui.KeyEnter}, Contexts: []Context{CtxQuery}, Handler: g.queryEnter},
		{Keys: []any{'a'}, Contexts: []Context{CtxQuery}, Handler: g.queryAddFilter},
		{Keys: []any{'d'}, Contexts: []Context{CtxQuery}, Handler: g.queryDeleteFilter},
		{Keys: []any{gocui.KeyEsc, 'q'}, Contexts: []Context{CtxQuery}, Handler: g.queryClose},

		// Query select popup
		{Keys: []any{'k', gocui.KeyArrowUp}, Contexts: []Context{CtxQuerySelect}, Handler: g.querySelectMoveUp},
		{Keys: []any{'j', gocui.KeyArrowDown}, Contexts: []Context{CtxQuerySelect}, Handler: g.querySelectMoveDown},
		{Keys: []any{gocui.KeyEnter}, Contexts: []Context{CtxQuerySelect}, Handler: g.querySelectConfirm},
		{Keys: []any{gocui.KeyEsc, 'q'}, Contexts: []Context{CtxQuerySelect}, Handler: g.querySelectClose},
	}
}

// Disabled reasons. They check the focused panel rather than the current
// context, so the ? menu judges each binding for the panel it was opened from.

func (g *Gui) noItemReason() string {
	if _, n := g.listState(g.focusedPanel()); n == 0 {
		return "Nothing selected"
	}
	return ""
}

// databaseDisabledReason blocks databases the Firestore document API can't read
func (g *Gui) databaseDisabledReason() string {
	if reason := g.noItemReason(); reason != "" {
		return reason
	}
	databases := g.getFilteredDatabases()
	if g.selectedDatabaseIdx < len(databases) && databases[g.selectedDatabaseIdx].Type == "DATASTORE_MODE" {
		return "Datastore mode databases can't be browsed"
	}
	return ""
}

func (g *Gui) noDocumentReason() string {
	if g.detailsSource() != "document" {
		return "No document open"
	}
	return ""
}

func (g *Gui) noCollectionReason() string {
	if g.currentCollection == "" {
		return "No collection open"
	}
	return ""
}

func (g *Gui) noCachedDocsReason() string {
	if len(g.docCache) == 0 {
		return "No cached documents"
	}
	return ""
}

func (g *Gui) noQueryResultsReason() string {
	if !g.queryResultMode {
		return "No query results to clear"
	}
	return ""
}

func (g *Gui) noCollectionForQueryReason() string {
	if collection, _ := g.queryTarget(); collection == "" {
		return "No collection to query"
	}
	return ""
}

func (g *Gui) scanDisabledReason() string {
	if g.scanRunning {
		return "Scan already in progress"
	}
	return g.noItemReason()
}

func (g *Gui) filterDisabledReason() string {
	if g.focusedPanel() == CtxDetails && g.detailsSource() != "document" {
		return "Filtering works on documents only"
	}
	return ""
}

func (g *Gui) searchDisabledReason() string {
	filter := g.getDetailsFilter()
	if filter == "" {
		return "No active filter, press / first"
	}
	if isJqQuery(filter) {
		return "n/N don't apply to jq queries"
	}
	return ""
}

func (g *Gui) refreshDisabledReason() string {
	ctx := g.focusedPanel()
	if ctx != CtxProjects && g.currentProject == "" {
		return "Select a project first"
	}
	switch ctx {
	case CtxTree:
		if g.currentCollection == "" && !g.queryResultMode {
			return "No collection open"
		}
	case CtxDetails:
		switch g.detailsSource() {
		case "document":
			if !isDocumentPath(g.currentDocPath) {
				return "Can't refresh a multi-document view"
			}
		case "function":
			if g.detailsTab != "logs" {
				return "Switch to the Logs tab to refresh logs"
			}
		default:
			return "Nothing to refresh here"
		}
	}
	return ""
}

func (g *Gui) copyDisabledReason() string {
	if g.focusedPanel() == CtxTree {
		if node, ok := g.selectedTreeNode(); !ok || node.Type != "document" {
			return "Select a document"
		}
		return ""
	}
	if g.detailsSource() == "scan" && g.scanResults != nil {
		return ""
	}
	return g.noDocumentReason()
}

func (g *Gui) copyPathDisabledReason() string {
	if g.focusedPanel() == CtxTree {
		if _, ok := g.selectedTreeNode(); !ok {
			return "Nothing selected"
		}
		return ""
	}
	if g.detailsSource() != "document" || !isDocumentPath(g.currentDocPath) {
		return "No document open"
	}
	return ""
}

// isDocumentPath reports whether path is a real document path rather than a
// label like "3 documents selected"
func isDocumentPath(path string) bool {
	for _, r := range path {
		if r == ' ' {
			return false
		}
	}
	return path != ""
}

// Mouse

func (g *Gui) setMouseBindings() error {
	lists := map[string]string{
		g.views.projects:    "projects",
		g.views.databases:   "databases",
		g.views.collections: "collections",
		g.views.tree:        "tree",
		g.views.details:     "details",
	}
	for view, column := range lists {
		column := column
		bindings := []*gocui.ViewMouseBinding{
			{ViewName: view, Key: gocui.MouseLeft, Handler: func(opts gocui.ViewMouseBindingOpts) error { return g.onPanelClick(column, opts) }},
			{ViewName: view, Key: gocui.MouseWheelUp, Handler: func(gocui.ViewMouseBindingOpts) error { return g.onPanelWheel(column, -1) }},
			{ViewName: view, Key: gocui.MouseWheelDown, Handler: func(gocui.ViewMouseBindingOpts) error { return g.onPanelWheel(column, 1) }},
		}
		for _, b := range bindings {
			if err := g.g.SetViewClickBinding(b); err != nil {
				return err
			}
		}
	}

	popupBindings := []*gocui.ViewMouseBinding{
		{ViewName: g.views.helpModal, Key: gocui.MouseLeft, Handler: g.onHelpClick},
		{ViewName: g.views.helpModal, Key: gocui.MouseWheelUp, Handler: func(gocui.ViewMouseBindingOpts) error { return g.helpMoveUp() }},
		{ViewName: g.views.helpModal, Key: gocui.MouseWheelDown, Handler: func(gocui.ViewMouseBindingOpts) error { return g.helpMoveDown() }},
		{ViewName: g.views.querySelect, Key: gocui.MouseLeft, Handler: g.onQuerySelectClick},
	}
	for _, view := range []string{g.views.commands, g.views.help, g.views.background} {
		popupBindings = append(popupBindings, &gocui.ViewMouseBinding{ViewName: view, Key: gocui.MouseLeft, Handler: g.onOutsideClick})
	}
	for _, b := range popupBindings {
		if err := g.g.SetViewClickBinding(b); err != nil {
			return err
		}
	}

	if err := g.g.SetTabClickBinding(g.views.collections, g.onCollectionsTabClick); err != nil {
		return err
	}
	return g.g.SetTabClickBinding(g.views.details, g.onDetailsTabClick)
}
