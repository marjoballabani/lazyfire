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

	// Collections state
	collections           []firebase.Collection
	selectedCollectionIdx int
	currentCollection     string

	// Tree state
	treeNodes       []TreeNode
	selectedTreeIdx int
	expandedPaths   map[string]bool
	docCache        map[string]map[string]any // Cache of fetched documents by path
	collectionCache map[string][]string       // Cache of document paths per collection

	// Details state
	currentDocPath     string
	currentDocData     map[string]any
	currentProjectInfo *firebase.ProjectDetails
	detailsScrollPos   int

	// Cached rendered content (avoid re-rendering on every Layout)
	cachedDetailsContent string
	cachedDetailsDocPath string
	cachedDetailsLines   []string // Raw JSON lines for search
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
		queryModal  string
		queryInput  string
		querySelect string
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
	filterInputActive bool   // true when typing in filter bar
	filterInputText   string // current input text
	filterInputPanel  string // which panel is being filtered
	filterCursorPos   int    // cursor position in filter text

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
		currentColumn:  "projects",
		expandedPaths:   make(map[string]bool),
		selectedDocs:    make(map[int]bool),
		docCache:        make(map[string]map[string]any),
		collectionCache: make(map[string][]string),
	}

	// Set view names
	gui.views.projects = "projects"
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
	gui.views.background = "background"

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

	// Load projects asynchronously after UI starts
	go func() {
		// Show auth status
		authType := "service account"
		if g.firebaseClient.IsUsingLocalAuth() {
			authType = "local Firebase/gcloud"
		}
		g.g.Update(func(gui *gocui.Gui) error {
			g.logCommand("auth", fmt.Sprintf("Using %s authentication", authType), "success")
			return nil
		})

		// Load projects
		g.g.Update(func(gui *gocui.Gui) error {
			g.logCommand("load", "Loading projects...", "running")
			return nil
		})

		if err := g.loadProjects(); err != nil {
			g.g.Update(func(gui *gocui.Gui) error {
				g.isLoading = false
				g.loadingText = ""
				g.logCommand("load", fmt.Sprintf("Failed: %v", err), "error")
				return nil
			})
			return
		}

		g.g.Update(func(gui *gocui.Gui) error {
			g.isLoading = false
			g.loadingText = ""
			g.logCommand("load", fmt.Sprintf("Loaded %d projects", len(g.projects)), "success")
			return nil
		})
	}()

	if err := g.g.MainLoop(); err != nil && err != gocui.ErrQuit {
		return err
	}
	return nil
}

func (g *Gui) loadProjects() error {
	projects, err := g.firebaseClient.ListProjects()
	if err != nil {
		return err
	}
	g.projects = projects
	return nil
}

func (g *Gui) loadCollections() error {
	collections, err := g.firebaseClient.ListCollections()
	if err != nil {
		return err
	}
	g.collections = collections
	g.selectedCollectionIdx = 0
	return nil
}

// clearDetailsCache clears all cached details content and resets scroll
func (g *Gui) clearDetailsCache() {
	g.cachedDetailsContent = ""
	g.cachedDetailsDocPath = ""
	g.cachedDetailsLines = nil
	g.cachedDetailsHeader = ""
	g.detailsViewDirty = true
	g.detailsScrollPos = 0
}

// getLoadingText returns formatted loading text with animated spinner
func (g *Gui) getLoadingText(text string) string {
	frame := atomic.LoadUint32(&g.spinnerFrame)
	spinner := spinnerFrames[frame%uint32(len(spinnerFrames))]
	return fmt.Sprintf("\033[33m%s %s\033[0m", spinner, text)
}

// isAnyLoading returns true if any panel is currently loading
func (g *Gui) isAnyLoading() bool {
	return g.isLoading || g.collectionsLoading || g.treeLoading || g.detailsLoading
}
