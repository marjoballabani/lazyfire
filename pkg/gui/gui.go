package gui

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jesseduffield/gocui"
	"github.com/marjoballabani/lazyfire/pkg/config"
	"github.com/marjoballabani/lazyfire/pkg/firebase"
	"github.com/marjoballabani/lazyfire/pkg/gui/icons"
)

// Spinner frames for loading animation
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type CommandExecution struct {
	Timestamp   string
	Command     string
	Description string
	Status      string
}

// TreeNode represents an item in the tree view (document or subcollection)
type TreeNode struct {
	Path        string // Full path e.g., "users/abc123/orders"
	Name        string // Display name (last segment)
	Type        string // "document" or "collection"
	Depth       int    // Indentation level
	HasChildren bool
	Expanded    bool
}

// ScanResult holds the health check result for a single collection.
type ScanResult struct {
	Collection string
	Status     string // "ok", "warning", "error", "skipped"
	Message    string // Human-readable summary
	DocPath    string // Path of the checked document
	Metrics    []string // All metric values (always shown)
	Warnings   []string // Metrics above 70% threshold
}

type Gui struct {
	g              *gocui.Gui
	config         *config.Config
	firebaseClient *firebase.Client
	version        string
	theme          *Theme

	// Projects state
	projects             []firebase.Project
	selectedProjectIndex int
	currentProject       string

	// Firestore databases of the current project
	databases           []firebase.Database
	selectedDatabaseIdx int
	currentDatabase     string
	databasesLoading    bool
	databasesFilter     string

	// Collections state
	collections           []firebase.Collection
	selectedCollectionIdx int
	currentCollection     string

	// Collections/Functions/Storage/Auth tab state
	collectionsTab      string // "collections", "functions", "storage", "auth", "rules", "indexes"
	functions           []firebase.CloudFunction
	selectedFunctionIdx int
	functionsFilter     string
	currentFunction     *firebase.CloudFunction
	functionLogs        []firebase.LogEntry
	functionsLoading    bool
	logsLoading         bool
	logsRefreshTicker   *time.Ticker

	// Storage state
	storageBuckets       []firebase.StorageBucket
	storageObjects       []firebase.StorageObject
	selectedBucketIdx    int
	selectedObjectIdx    int
	currentBucket        string
	storagePrefix        string // current "folder" prefix
	storagePrefixStack   []string // stack for navigating back
	storageLoading       bool
	storageFilter        string

	// Auth state
	authUsers            []firebase.AuthUser
	selectedAuthIdx      int
	authLoading          bool
	authFilter           string

	// Rules state
	firestoreRules       *firebase.FirestoreRules
	rulesLoading         bool
	collectionsScrollPos int // scroll offset of the rules/indexes tabs

	// Indexes state
	firestoreIndexes     []firebase.FirestoreIndex
	indexesLoading       bool

	// Tree state
	treeNodes       []TreeNode
	selectedTreeIdx int
	expandedPaths   map[string]bool
	docCache        map[string]map[string]any    // Cache of fetched documents by path
	statsCache      map[string]*firebase.DocStats // Cache of document stats by path
	collectionCache     map[string][]string       // Cache of document paths per collection
	compositeIndexCache map[string]*bool          // nil=not checked, false=no composites, true=has composites

	// Scan state
	scanResults   []ScanResult
	scanRunning   bool
	scanProgress  string // "3/12 collections"
	scanProjectID string // Project ID being scanned

	// Confirm dialog state
	confirmOpen     bool
	confirmTitle    string
	confirmMessage  string
	confirmCallback func()

	// Details state
	currentDocPath     string
	currentDocData     map[string]any
	currentDocStats    *firebase.DocStats
	currentProjectInfo *firebase.ProjectDetails
	detailsScrollPos   int
	detailsCursor      int    // highlighted line in details (view line, wrapping included)
	detailsLineCount   int    // view lines in details, updated by Layout
	detailsViewHeight  int    // visible lines in details, updated by Layout
	detailsTab         string // "details" or "logs"

	// Display mode
	compactJSON        bool // Toggle between compact and pretty JSON
	humanizeTimestamps bool // Format Firestore timestamps in human-readable form
	showLineNumbers    bool // Show line numbers in JSON view
	logLevelFilter     string // Filter function logs by severity (e.g., "ERROR")

	// Cached rendered content (avoid re-rendering on every Layout)
	cachedDetailsContent string
	cachedDetailsDocPath string
	cachedDetailsHeader  string   // Header (path + stats)
	detailsViewDirty     bool     // True when content needs to be pushed to view

	// Command execution tracking
	commandHistory []CommandExecution

	// View names
	views struct {
		background  string
		projects    string
		collections string
		tree        string
		details     string
		commands    string
		help        string
		modal       string
		helpModal   string
		databases   string
		queryModal  string
		queryInput  string
		querySelect string
		confirm     string
		filterPrefix string
		filterInput  string
	}

	// Current column: "projects", "collections", "tree", "details"
	currentColumn  string
	previousColumn string // Track previous column for returning from details

	// Modal state
	modalOpen bool
	helpOpen  bool
	helpPopup *Popup

	// Loading state
	isLoading          bool
	loadingText        string
	collectionsLoading bool
	treeLoading        bool
	detailsLoading     bool
	spinnerFrame       uint32 // Current spinner animation frame

	// Filter state
	filterInputActive bool    // true when typing in filter bar
	filterInputText   string  // current input text
	filterInputPanel  Context // which context is being filtered

	// Committed filters (persist after Enter, cleared by Esc)
	projectsFilter    string
	collectionsFilter string
	treeFilter        string
	detailsFilter     string

	// Select mode (visual selection in tree)
	selectMode     bool
	selectedDocs   map[int]bool // indices of selected tree nodes
	selectStartIdx int          // where selection started

	// Query builder state
	queryModalOpen   bool
	queryCollection  string // Collection path for query (can be subcollection)
	queryNodeIdx     int    // Index of collection node in tree (-1 for top-level)
	queryFilters     []firebase.QueryFilter
	queryOrderBy     string
	queryOrderDir    string // ASC or DESC
	queryLimit       int
	queryActiveRow   int    // Currently selected row in modal (0=filters, 1=orderBy, 2=limit, 3=buttons)
	queryActiveCol   int    // Currently selected column/field in row
	queryEditMode   bool   // True when editing a field value
	queryEditBuffer string // Buffer for editing field value
	queryResultMode bool   // True when showing query results instead of normal tree

	// Query select popup state (for operators and types)
	querySelectOpen     bool
	querySelectItems    []string
	querySelectIdx      int
	querySelectCallback func(string) // Called when item is selected

	// Frame styling
	roundedFrameRunes []rune

	// Keybindings, see keybindings.go
	bindings []*Binding

	// Bumped on every project switch so results loaded for the previous
	// project are dropped instead of mixed into the new one
	projectGen int
	// Same for Firestore data, bumped on every project or database switch
	databaseGen int

	// Last query run on a whole collection, re-run by refresh
	lastQueryCollection string
	lastQueryOptions    firebase.QueryOptions

	// Toast shown in the options bar for a few seconds
	toastText    string
	toastIsError bool
	toastSeq     int
}

const (
	PROJECT_COLOR    = gocui.ColorCyan
	COLLECTION_COLOR = gocui.ColorYellow
	SELECTED_BG      = gocui.ColorDefault
	SELECTED_FG      = gocui.ColorDefault
	ERROR_COLOR      = gocui.ColorRed
	SUCCESS_COLOR    = gocui.ColorGreen
	WARNING_COLOR    = gocui.ColorYellow
	FOCUS_COLOR      = gocui.ColorCyan
)

func NewGui(config *config.Config, firebaseClient *firebase.Client, version string) (*Gui, error) {
	g, err := gocui.NewGui(gocui.NewGuiOpts{
		OutputMode:      gocui.OutputTrue,
		SupportOverlaps: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gui: %v", err)
	}

	// Create theme from config
	theme := NewTheme(config.UI.Theme)

	// Initialize icons based on config
	if !config.UI.ShowIcons {
		icons.SetEnabled(false)
	} else {
		switch config.UI.NerdFontsVersion {
		case "2":
			icons.PatchForNerdFontsV2()
		case "3":
			// Default v3 icons, nothing to do
		default:
			// Disable icons for graceful fallback
			icons.SetEnabled(false)
		}
	}

	gui := &Gui{
		g:              g,
		config:         config,
		firebaseClient: firebaseClient,
		version:        version,
		theme:          theme,
		currentProject: firebaseClient.GetCurrentProject(),
		currentDatabase: firebaseClient.GetCurrentDatabase(),
		currentColumn:  "projects",
		collectionsTab: "collections",
		detailsTab:     "details",
		expandedPaths:   make(map[string]bool),
		selectedDocs:    make(map[int]bool),
		docCache:        make(map[string]map[string]any),
		statsCache:      make(map[string]*firebase.DocStats),
		collectionCache:     make(map[string][]string),
		compositeIndexCache: make(map[string]*bool),
	}

	// Set view names
	gui.views.projects = "projects"
	gui.views.databases = "databases"
	gui.views.collections = "collections"
	gui.views.tree = "tree"
	gui.views.details = "details"
	gui.views.commands = "commands"
	gui.views.help = "help"
	gui.views.modal = "modal"
	gui.views.helpModal = "helpModal"
	gui.views.queryModal = "queryModal"
	gui.views.queryInput = "queryInput"
	gui.views.querySelect = "querySelect"
	gui.views.confirm = "confirm"
	gui.views.background = "background"
	gui.views.filterPrefix = "filterPrefix"
	gui.views.filterInput = "filterInput"

	// Configure gocui
	g.Cursor = false
	g.Mouse = true
	g.InputEsc = true
	g.ShowListFooter = true // Show "X of Y" footer

	// Set colors for frames from theme
	g.BgColor = gocui.ColorDefault
	g.FgColor = gocui.ColorDefault
	g.FrameColor = gui.theme.InactiveBorderColor
	g.SelFrameColor = gui.theme.ActiveBorderColor
	g.SelFgColor = gui.theme.ActiveBorderColor
	g.Highlight = true

	// Rounded frame characters: ─ │ ╭ ╮ ╰ ╯
	gui.roundedFrameRunes = []rune{'─', '│', '╭', '╮', '╰', '╯'}

	// Set layout function
	g.SetManagerFunc(func(g *gocui.Gui) error {
		return gui.Layout(g)
	})

	// Set up keybindings
	if err := gui.setKeybindings(); err != nil {
		return nil, err
	}

	// Set initial loading state
	gui.isLoading = true
	gui.loadingText = "Starting..."
	gui.logCommand("init", "LazyFire starting...", "running")

	return gui, nil
}

func (g *Gui) getActiveColorCode() string {
	return g.theme.GetAnsiColorCode()
}

func (g *Gui) logCommand(command, description, status string) {
	timestamp := time.Now().Format("15:04:05")

	cmdExec := CommandExecution{
		Timestamp:   timestamp,
		Command:     command,
		Description: description,
		Status:      status,
	}

	g.commandHistory = append(g.commandHistory, cmdExec)

	// Keep only last 10 commands
	if len(g.commandHistory) > 10 {
		g.commandHistory = g.commandHistory[1:]
	}
}

func (g *Gui) Run() error {
	defer g.g.Close()

	// Start spinner animation ticker
	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			atomic.AddUint32(&g.spinnerFrame, 1)
			if g.isAnyLoading() {
				g.g.Update(func(gui *gocui.Gui) error {
					return nil
				})
			}
		}
	}()

	authType := "service account"
	if g.firebaseClient.IsUsingLocalAuth() {
		authType = "local Firebase/gcloud"
	}
	g.logCommand("auth", fmt.Sprintf("Using %s authentication", authType), "success")
	g.loadProjects()

	if err := g.g.MainLoop(); err != nil && err != gocui.ErrQuit {
		return err
	}
	return nil
}

// loadProjects lists projects, keeping the highlighted project (or the
// active one on first load) selected
func (g *Gui) loadProjects() {
	g.isLoading = true
	g.loadingText = "Loading projects..."
	g.logCommand("load", "Loading projects...", "running")
	selectedID := g.selectedItemKey(CtxProjects)
	if selectedID == "" {
		selectedID = g.currentProject
	}
	go func() {
		projects, err := g.firebaseClient.ListProjects()
		g.g.Update(func(gui *gocui.Gui) error {
			g.isLoading = false
			g.loadingText = ""
			if err != nil {
				g.logCommand("load", fmt.Sprintf("Failed: %v", err), "error")
				return nil
			}
			g.projects = projects
			g.selectItemByKey(CtxProjects, selectedID)
			g.logCommand("load", fmt.Sprintf("Loaded %d projects", len(projects)), "success")
			return nil
		})
	}()
}

// loadDatabases lists the current project's Firestore databases, keeping the
// highlighted one (or the active one) selected. If listing fails, e.g. for
// lack of permission, the default database is still offered.
func (g *Gui) loadDatabases() {
	gen := g.projectGen
	g.databasesLoading = true
	selectedID := g.selectedItemKey(CtxDatabases)
	if selectedID == "" {
		selectedID = g.currentDatabase
	}
	go func() {
		databases, err := g.firebaseClient.ListDatabases()
		g.g.Update(func(gui *gocui.Gui) error {
			if gen != g.projectGen {
				return nil // loaded for the previous project
			}
			g.databasesLoading = false
			if err != nil {
				g.logCommand("databases", fmt.Sprintf("ListDatabases failed: %v", err), "error")
				databases = []firebase.Database{{ID: firebase.DefaultDatabase}}
			} else {
				g.logCommand("databases", fmt.Sprintf("Loaded %d databases", len(databases)), "success")
			}
			g.databases = databases
			g.selectItemByKey(CtxDatabases, selectedID)
			return nil
		})
	}()
}

// loadCollections lists the current database's root collections
func (g *Gui) loadCollections() {
	gen := g.databaseGen
	g.collectionsLoading = true
	go func() {
		collections, err := g.firebaseClient.ListCollections()
		g.g.Update(func(gui *gocui.Gui) error {
			if gen != g.databaseGen {
				return nil // loaded for another project or database
			}
			g.collectionsLoading = false
			if err != nil {
				g.logCommand("api", fmt.Sprintf("ListCollections failed: %v", err), "error")
				return nil
			}
			g.collections = collections
			g.logCommand("api", fmt.Sprintf("ListCollections(%s) → %d collections", g.currentProject, len(collections)), "success")
			return nil
		})
	}()
}

// clearDetailsCache clears all cached details content and resets scroll
func (g *Gui) clearDetailsCache() {
	g.cachedDetailsContent = ""
	g.cachedDetailsDocPath = ""
	g.cachedDetailsHeader = ""
	g.detailsViewDirty = true
	g.detailsScrollPos = 0
	g.detailsCursor = 0
}

// toast shows a short message in the options bar for a few seconds
func (g *Gui) toast(msg string, isError bool) {
	g.toastText = msg
	g.toastIsError = isError
	g.toastSeq++
	if g.g == nil {
		return
	}
	seq := g.toastSeq
	duration := 2500 * time.Millisecond
	if isError {
		duration = 4 * time.Second
	}
	time.AfterFunc(duration, func() {
		g.g.Update(func(*gocui.Gui) error {
			if g.toastSeq == seq {
				g.toastText = ""
			}
			return nil
		})
	})
}

// getLoadingText returns formatted loading text with animated spinner
func (g *Gui) getLoadingText(text string) string {
	frame := atomic.LoadUint32(&g.spinnerFrame)
	spinner := spinnerFrames[frame%uint32(len(spinnerFrames))]
	return fmt.Sprintf("\033[33m%s %s\033[0m", spinner, text)
}

// isAnyLoading returns true if any panel is currently loading
func (g *Gui) isAnyLoading() bool {
	return g.isLoading || g.databasesLoading || g.collectionsLoading || g.treeLoading || g.detailsLoading || g.functionsLoading || g.logsLoading || g.storageLoading || g.authLoading || g.rulesLoading || g.indexesLoading
}

// loadFunctions loads Cloud Functions for the current project
func (g *Gui) loadFunctions() {
	gen := g.projectGen
	g.functionsLoading = true
	go func() {
		functions, err := g.firebaseClient.ListFunctions()
		g.g.Update(func(gui *gocui.Gui) error {
			if gen != g.projectGen {
				return nil // loaded for the previous project
			}
			g.functionsLoading = false
			if err != nil {
				g.logCommand("functions", fmt.Sprintf("Error: %v", err), "error")
				return nil
			}
			g.functions = functions
			g.logCommand("functions", fmt.Sprintf("Loaded %d functions", len(functions)), "success")
			return nil
		})
	}()
}

// loadFunctionLogs loads logs for the current function
func (g *Gui) loadFunctionLogs() {
	gen := g.projectGen
	if g.currentFunction == nil {
		return
	}
	g.logsLoading = true
	name := g.currentFunction.DisplayName
	go func() {
		logs, err := g.firebaseClient.GetFunctionLogs(name, 50)
		g.g.Update(func(gui *gocui.Gui) error {
			if gen != g.projectGen {
				return nil // loaded for the previous project
			}
			// Logs of a function the user moved away from; the newer request
			// for the current function owns logsLoading
			if g.currentFunction == nil || g.currentFunction.DisplayName != name {
				return nil
			}
			g.logsLoading = false
			if err != nil {
				g.logCommand("logs", fmt.Sprintf("Error: %v", err), "error")
				return nil
			}
			g.functionLogs = logs
			return nil
		})
	}()
}

// startLogsRefresh starts auto-refreshing logs every 3 seconds
func (g *Gui) startLogsRefresh() {
	g.stopLogsRefresh() // Stop any existing ticker
	g.logsRefreshTicker = time.NewTicker(3 * time.Second)
	go func() {
		for range g.logsRefreshTicker.C {
			if g.currentFunction != nil && g.collectionsTab == "functions" {
				g.loadFunctionLogs()
			}
		}
	}()
}

// stopLogsRefresh stops the auto-refresh ticker
func (g *Gui) stopLogsRefresh() {
	if g.logsRefreshTicker != nil {
		g.logsRefreshTicker.Stop()
		g.logsRefreshTicker = nil
	}
}

// loadStorageBuckets loads Cloud Storage buckets for the current project
func (g *Gui) loadStorageBuckets() {
	gen := g.projectGen
	g.storageLoading = true
	go func() {
		buckets, err := g.firebaseClient.ListBuckets()
		g.g.Update(func(gui *gocui.Gui) error {
			if gen != g.projectGen {
				return nil // loaded for the previous project
			}
			g.storageLoading = false
			if err != nil {
				g.logCommand("storage", fmt.Sprintf("Error: %v", err), "error")
				return nil
			}
			g.storageBuckets = buckets
			g.logCommand("storage", fmt.Sprintf("Loaded %d buckets", len(buckets)), "success")
			return nil
		})
	}()
}

// loadStorageObjects loads objects in a bucket with the current prefix and
// selects the object named selectName, if any
func (g *Gui) loadStorageObjects(selectName string) {
	gen := g.projectGen
	if g.currentBucket == "" {
		return
	}
	g.storageLoading = true
	bucket, prefix := g.currentBucket, g.storagePrefix
	go func() {
		objects, err := g.firebaseClient.ListObjects(bucket, prefix, 100)
		g.g.Update(func(gui *gocui.Gui) error {
			if gen != g.projectGen {
				return nil // loaded for the previous project
			}
			if g.currentBucket != bucket || g.storagePrefix != prefix {
				return nil // the user already left this folder
			}
			g.storageLoading = false
			if err != nil {
				g.logCommand("storage", fmt.Sprintf("Error: %v", err), "error")
				return nil
			}
			g.storageObjects = objects
			if selectName != "" {
				g.selectItemByKey(CtxStorage, selectName)
			}
			g.logCommand("storage", fmt.Sprintf("Listed %d items in %s", len(objects), g.currentBucket), "success")
			return nil
		})
	}()
}

// loadAuthUsers loads Firebase Auth users
func (g *Gui) loadAuthUsers() {
	gen := g.projectGen
	g.authLoading = true
	go func() {
		users, err := g.firebaseClient.ListAuthUsers(100)
		g.g.Update(func(gui *gocui.Gui) error {
			if gen != g.projectGen {
				return nil // loaded for the previous project
			}
			g.authLoading = false
			if err != nil {
				g.logCommand("auth", fmt.Sprintf("Error: %v", err), "error")
				return nil
			}
			g.authUsers = users
			g.logCommand("auth", fmt.Sprintf("Loaded %d users", len(users)), "success")
			return nil
		})
	}()
}

// loadFirestoreRules loads current Firestore security rules
func (g *Gui) loadFirestoreRules() {
	gen := g.projectGen
	g.rulesLoading = true
	go func() {
		rules, err := g.firebaseClient.GetFirestoreRules()
		g.g.Update(func(gui *gocui.Gui) error {
			if gen != g.projectGen {
				return nil // loaded for the previous project
			}
			g.rulesLoading = false
			if err != nil {
				g.logCommand("rules", fmt.Sprintf("Error: %v", err), "error")
				return nil
			}
			g.firestoreRules = rules
			g.logCommand("rules", "Loaded Firestore rules", "success")
			return nil
		})
	}()
}

// loadFirestoreIndexes loads Firestore composite indexes
func (g *Gui) loadFirestoreIndexes() {
	gen := g.databaseGen
	g.indexesLoading = true
	go func() {
		indexes, err := g.firebaseClient.ListFirestoreIndexes()
		g.g.Update(func(gui *gocui.Gui) error {
			if gen != g.databaseGen {
				return nil // loaded for another project or database
			}
			g.indexesLoading = false
			if err != nil {
				g.logCommand("indexes", fmt.Sprintf("Error: %v", err), "error")
				return nil
			}
			g.firestoreIndexes = indexes
			g.logCommand("indexes", fmt.Sprintf("Loaded %d composite indexes", len(indexes)), "success")
			return nil
		})
	}()
}
