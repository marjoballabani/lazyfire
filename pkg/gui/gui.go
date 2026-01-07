package gui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/jesseduffield/gocui"
	"github.com/marjoballabani/lazyfire/pkg/config"
	"github.com/marjoballabani/lazyfire/pkg/firebase"
	"github.com/marjoballabani/lazyfire/pkg/gui/icons"
)

// ANSI color codes for JSON syntax highlighting
const (
	colorReset   = "\033[0m"
	colorKey     = "\033[36m"  // Cyan for keys
	colorString  = "\033[32m"  // Green for string values
	colorNumber  = "\033[33m"  // Yellow for numbers
	colorBool    = "\033[35m"  // Magenta for booleans
	colorNull    = "\033[31m"  // Red for null
	colorBracket = "\033[90m"  // Gray for brackets
)

// Precompiled regex pattern for JSON key detection
var jsonKeyPattern = regexp.MustCompile(`"([^"\\]|\\.)*"\s*:`)

// colorizeJSON adds ANSI color codes to JSON string for terminal display
func colorizeJSON(jsonStr string) string {
	var result strings.Builder
	lines := strings.Split(jsonStr, "\n")

	for i, line := range lines {
		colored := colorizeLine(line)
		result.WriteString(colored)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// colorizeLine applies syntax highlighting to a single line of JSON
func colorizeLine(line string) string {
	// Check for key-value patterns
	if match := jsonKeyPattern.FindStringIndex(line); match != nil {
		keyEnd := match[1]
		key := line[match[0]:keyEnd]
		rest := line[keyEnd:]

		// Colorize the key
		coloredKey := colorKey + key + colorReset

		// Colorize the value
		coloredValue := colorizeValue(rest)

		return line[:match[0]] + coloredKey + coloredValue
	}

	// No key found, just colorize brackets
	return colorizeBrackets(line)
}

// colorizeValue applies color to JSON values
func colorizeValue(s string) string {
	s = strings.TrimSpace(s)

	// Check for string value
	if strings.HasPrefix(s, `"`) {
		return " " + colorString + s + colorReset
	}

	// Check for number
	if len(s) > 0 && (s[0] == '-' || (s[0] >= '0' && s[0] <= '9')) {
		// Find end of number
		end := 0
		for end < len(s) && (s[end] == '-' || s[end] == '.' || s[end] == 'e' || s[end] == 'E' || s[end] == '+' || (s[end] >= '0' && s[end] <= '9')) {
			end++
		}
		if end > 0 {
			return " " + colorNumber + s[:end] + colorReset + s[end:]
		}
	}

	// Check for boolean
	if strings.HasPrefix(s, "true") {
		return " " + colorBool + "true" + colorReset + s[4:]
	}
	if strings.HasPrefix(s, "false") {
		return " " + colorBool + "false" + colorReset + s[5:]
	}

	// Check for null
	if strings.HasPrefix(s, "null") {
		return " " + colorNull + "null" + colorReset + s[4:]
	}

	// Check for array/object start
	if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
		return " " + colorizeBrackets(s)
	}

	return " " + s
}

// colorizeBrackets adds color to brackets and braces
func colorizeBrackets(s string) string {
	var result strings.Builder
	for _, ch := range s {
		switch ch {
		case '{', '}', '[', ']':
			result.WriteString(colorBracket)
			result.WriteRune(ch)
			result.WriteString(colorReset)
		default:
			result.WriteRune(ch)
		}
	}
	return result.String()
}

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

	// Details state
	currentDocPath     string
	currentDocData     map[string]interface{}
	currentProjectInfo *firebase.ProjectDetails
	detailsScrollPos   int

	// Cached rendered content (avoid re-rendering on every Layout)
	cachedDetailsContent string
	cachedDetailsDocPath string

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
	}

	// Current column: "projects", "collections", "tree"
	currentColumn string

	// Modal state
	modalOpen bool
	helpOpen  bool
	helpPopup *Popup

	// Loading state
	isLoading   bool
	loadingText string

	// Filter state
	filterInputActive bool   // true when typing in filter bar
	filterInputText   string // current input text
	filterInputPanel  string // which panel is being filtered

	// Committed filters (persist after Enter, cleared by Esc)
	projectsFilter    string
	collectionsFilter string
	treeFilter        string
	detailsFilter     string

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
	switch config.UI.NerdFontsVersion {
	case "2":
		icons.PatchForNerdFontsV2()
	case "3":
		// Default v3 icons, nothing to do
	default:
		// Disable icons for graceful fallback
		icons.SetEnabled(false)
	}

	gui := &Gui{
		g:              g,
		config:         config,
		firebaseClient: firebaseClient,
		version:        version,
		theme:          theme,
		currentProject: firebaseClient.GetCurrentProject(),
		currentColumn:  "projects",
		expandedPaths:  make(map[string]bool),
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
	timestamp := fmt.Sprintf("%s", time.Now().Format("15:04:05"))

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

func (g *Gui) Layout(gui *gocui.Gui) error {
	maxX, maxY := gui.Size()

	// Background view (covers entire screen, behind everything)
	if v, err := gui.SetView(g.views.background, -1, -1, maxX, maxY, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Frame = false
		v.BgColor = gocui.ColorDefault
		v.FgColor = gocui.ColorDefault
	}

	// Left panel width (1/3 of screen)
	leftWidth := maxX / 3

	// Calculate heights for left panels (3 stacked)
	leftHeight := maxY - 3 // Leave room for help bar

	var projectsEnd, collectionsEnd int
	collapsedSingleLine := 3 // Height for collapsed single-line panel (borders + 1 line)

	switch g.currentColumn {
	case "projects":
		// Projects expanded, others share remaining space
		expandedHeight := leftHeight / 2
		remainingHeight := leftHeight - expandedHeight
		projectsEnd = expandedHeight
		collectionsEnd = expandedHeight + remainingHeight/2
	case "collections":
		// Projects collapsed to 1 line, collections expanded
		remainingHeight := leftHeight - collapsedSingleLine
		expandedHeight := remainingHeight * 2 / 3
		projectsEnd = collapsedSingleLine
		collectionsEnd = collapsedSingleLine + expandedHeight
	case "tree":
		// Projects collapsed to 1 line, tree gets more space
		remainingHeight := leftHeight - collapsedSingleLine
		projectsEnd = collapsedSingleLine
		collectionsEnd = collapsedSingleLine + remainingHeight/3
	default: // details or other
		// Projects collapsed to 1 line, equal split for collections/tree
		remainingHeight := leftHeight - collapsedSingleLine
		projectsEnd = collapsedSingleLine
		collectionsEnd = collapsedSingleLine + remainingHeight/2
	}

	// Right side layout
	commandsHeight := 3

	// Projects panel (top-left)
	if v, err := gui.SetView(g.views.projects, 0, 0, leftWidth-1, projectsEnd-1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = " " + icons.PROJECT_ICON + " Projects "
		v.TitleColor = g.theme.InactiveBorderColor
		v.BgColor = gocui.ColorDefault
		v.FgColor = gocui.ColorDefault
		v.SelBgColor = g.theme.SelectedLineBgColor
		v.SelFgColor = gocui.ColorDefault
		v.FrameRunes = g.roundedFrameRunes
	}

	if v, err := gui.View(g.views.projects); err == nil {
		hasCommittedFilter := g.hasActiveFilter("projects")
		isTypingFilter := g.isFilteringPanel("projects")
		isFocused := g.currentColumn == "projects"

		// Title/border color: filter color when focused AND filter is committed (not while typing)
		if isFocused && hasCommittedFilter {
			// Must set global SelFrameColor because gocui uses it for focused views
			gui.SelFrameColor = g.theme.FilterBorderColor
			gui.SelFgColor = g.theme.FilterBorderColor
			v.TitleColor = g.theme.FilterBorderColor
			v.FrameColor = g.theme.FilterBorderColor
			v.Title = " " + icons.PROJECT_ICON + " Projects "
		} else if isFocused {
			gui.SelFrameColor = g.theme.ActiveBorderColor
			gui.SelFgColor = g.theme.ActiveBorderColor
			v.TitleColor = g.theme.ActiveBorderColor
			v.FrameColor = g.theme.ActiveBorderColor
			v.Title = " " + icons.PROJECT_ICON + " Projects "
		} else {
			v.TitleColor = g.theme.InactiveBorderColor
			v.FrameColor = g.theme.InactiveBorderColor
			v.Title = " " + icons.PROJECT_ICON + " Projects "
		}
		// Show footer only when expanded
		hasFilter := hasCommittedFilter || isTypingFilter
		if isFocused {
			filtered := g.getFilteredProjects()
			if hasFilter {
				v.Footer = fmt.Sprintf("%d/%d matched", len(filtered), len(g.projects))
			} else if len(g.projects) > 0 {
				v.Footer = fmt.Sprintf("%d of %d", g.selectedProjectIndex+1, len(g.projects))
			} else {
				v.Footer = "0 of 0"
			}
		} else {
			v.Footer = "" // Hide footer when collapsed
		}
		g.updateProjectsView(v)
	}

	// Collections panel (middle-left)
	if v, err := gui.SetView(g.views.collections, 0, projectsEnd, leftWidth-1, collectionsEnd-1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = " " + icons.COLLECTION_ICON + " Collections "
		v.TitleColor = g.theme.InactiveBorderColor
		v.BgColor = gocui.ColorDefault
		v.FgColor = gocui.ColorDefault
		v.SelBgColor = g.theme.SelectedLineBgColor
		v.SelFgColor = gocui.ColorDefault
		v.FrameRunes = g.roundedFrameRunes
	}

	if v, err := gui.View(g.views.collections); err == nil {
		hasCommittedFilter := g.hasActiveFilter("collections")
		isTypingFilter := g.isFilteringPanel("collections")
		isFocused := g.currentColumn == "collections"

		// Title/border color: filter color when focused AND filter is committed (not while typing)
		if isFocused && hasCommittedFilter {
			gui.SelFrameColor = g.theme.FilterBorderColor
			gui.SelFgColor = g.theme.FilterBorderColor
			v.TitleColor = g.theme.FilterBorderColor
			v.FrameColor = g.theme.FilterBorderColor
		} else if isFocused {
			gui.SelFrameColor = g.theme.ActiveBorderColor
			gui.SelFgColor = g.theme.ActiveBorderColor
			v.TitleColor = g.theme.ActiveBorderColor
			v.FrameColor = g.theme.ActiveBorderColor
		} else {
			v.TitleColor = g.theme.InactiveBorderColor
			v.FrameColor = g.theme.InactiveBorderColor
		}
		v.Title = " " + icons.COLLECTION_ICON + " Collections "
		// Set footer with count
		filtered := g.getFilteredCollections()
		hasFilter := hasCommittedFilter || isTypingFilter
		if hasFilter {
			v.Footer = fmt.Sprintf("%d/%d matched", len(filtered), len(g.collections))
		} else if len(g.collections) > 0 {
			v.Footer = fmt.Sprintf("%d of %d", g.selectedCollectionIdx+1, len(g.collections))
		} else {
			v.Footer = "0 of 0"
		}
		g.updateCollectionsView(v)
	}

	// Tree panel (bottom-left)
	if v, err := gui.SetView(g.views.tree, 0, collectionsEnd, leftWidth-1, maxY-3, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = " " + icons.TREE_ICON + " Tree "
		v.TitleColor = g.theme.InactiveBorderColor
		v.BgColor = gocui.ColorDefault
		v.FgColor = gocui.ColorDefault
		v.SelBgColor = g.theme.SelectedLineBgColor
		v.SelFgColor = gocui.ColorDefault
		v.FrameRunes = g.roundedFrameRunes
	}

	if v, err := gui.View(g.views.tree); err == nil {
		hasCommittedFilter := g.hasActiveFilter("tree")
		isTypingFilter := g.isFilteringPanel("tree")
		isFocused := g.currentColumn == "tree"

		// Title/border color: filter color when focused AND filter is committed (not while typing)
		if isFocused && hasCommittedFilter {
			gui.SelFrameColor = g.theme.FilterBorderColor
			gui.SelFgColor = g.theme.FilterBorderColor
			v.TitleColor = g.theme.FilterBorderColor
			v.FrameColor = g.theme.FilterBorderColor
		} else if isFocused {
			gui.SelFrameColor = g.theme.ActiveBorderColor
			gui.SelFgColor = g.theme.ActiveBorderColor
			v.TitleColor = g.theme.ActiveBorderColor
			v.FrameColor = g.theme.ActiveBorderColor
		} else {
			v.TitleColor = g.theme.InactiveBorderColor
			v.FrameColor = g.theme.InactiveBorderColor
		}
		v.Title = " " + icons.TREE_ICON + " Tree "
		// Set footer with count
		filtered := g.getFilteredTreeNodes()
		hasFilter := hasCommittedFilter || isTypingFilter
		if hasFilter {
			v.Footer = fmt.Sprintf("%d/%d matched", len(filtered), len(g.treeNodes))
		} else if len(g.treeNodes) > 0 {
			v.Footer = fmt.Sprintf("%d of %d", g.selectedTreeIdx+1, len(g.treeNodes))
		} else {
			v.Footer = "0 of 0"
		}
		g.updateTreeView(v)
	}

	// Details panel (top-right, big)
	if v, err := gui.SetView(g.views.details, leftWidth, 0, maxX-1, maxY-commandsHeight-3, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = " " + icons.DETAILS_ICON + " Details "
		v.TitleColor = g.theme.InactiveBorderColor
		v.Wrap = true
		v.BgColor = gocui.ColorDefault
		v.FgColor = gocui.ColorDefault
		v.SelBgColor = gocui.ColorDefault
		v.SelFgColor = gocui.ColorDefault
		v.FrameRunes = g.roundedFrameRunes
	}

	if v, err := gui.View(g.views.details); err == nil {
		hasCommittedFilter := g.hasActiveFilter("details")
		isFocused := g.currentColumn == "details"

		// Title/border color: filter color when focused AND filter is committed (not while typing)
		if isFocused && hasCommittedFilter {
			gui.SelFrameColor = g.theme.FilterBorderColor
			gui.SelFgColor = g.theme.FilterBorderColor
			v.Title = " " + icons.DETAILS_ICON + " Details (filtered) "
			v.TitleColor = g.theme.FilterBorderColor
			v.FrameColor = g.theme.FilterBorderColor
		} else if isFocused {
			gui.SelFrameColor = g.theme.ActiveBorderColor
			gui.SelFgColor = g.theme.ActiveBorderColor
			v.Title = " " + icons.DETAILS_ICON + " Details (j/k scroll) "
			v.TitleColor = g.theme.ActiveBorderColor
			v.FrameColor = g.theme.ActiveBorderColor
		} else {
			v.Title = " " + icons.DETAILS_ICON + " Details "
			v.TitleColor = g.theme.InactiveBorderColor
			v.FrameColor = g.theme.InactiveBorderColor
		}
		g.updateDetailsView(v)
		v.SetOrigin(0, g.detailsScrollPos)
	}

	// Commands panel (bottom-right, single row)
	if v, err := gui.SetView(g.views.commands, leftWidth, maxY-commandsHeight-2, maxX-1, maxY-3, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = " " + icons.COMMAND_ICON + " Commands "
		v.TitleColor = g.theme.InactiveBorderColor
		v.BgColor = gocui.ColorDefault
		v.FgColor = gocui.ColorDefault
		v.SelBgColor = gocui.ColorDefault
		v.SelFgColor = gocui.ColorDefault
		v.FrameRunes = g.roundedFrameRunes
	}

	if v, err := gui.View(g.views.commands); err == nil {
		g.updateCommandsView(v)
	}

	// Help bar (bottom, full width)
	if v, err := gui.SetView(g.views.help, 0, maxY-2, maxX-1, maxY, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Frame = false
		v.BgColor = gocui.ColorDefault
		v.FgColor = gocui.ColorDefault
		v.SelBgColor = gocui.ColorDefault
		v.SelFgColor = gocui.ColorDefault
	}

	if v, err := gui.View(g.views.help); err == nil {
		g.updateHelpView(v)
	}

	// Help modal (keyboard shortcuts)
	if g.helpOpen {
		modalWidth := 50
		modalHeight := 22
		if modalHeight > maxY-4 {
			modalHeight = maxY - 4
		}
		modalX := (maxX - modalWidth) / 2
		modalY := (maxY - modalHeight) / 2

		if v, err := gui.SetView(g.views.helpModal, modalX, modalY, modalX+modalWidth, modalY+modalHeight, 0); err != nil {
			if !errors.Is(err, gocui.ErrUnknownView) {
				return err
			}
			v.Title = " " + icons.KEYBOARD_ICON + " Keyboard Shortcuts "
			v.TitleColor = g.theme.ActiveBorderColor
			v.FrameColor = g.theme.ActiveBorderColor
			v.FrameRunes = g.roundedFrameRunes
			v.SelBgColor = g.theme.SelectedLineBgColor
			v.SelFgColor = gocui.ColorDefault
		}

		if v, err := gui.View(g.views.helpModal); err == nil {
			g.renderHelpContent(v)
			if _, err := gui.SetCurrentView(g.views.helpModal); err != nil {
				return fmt.Errorf("failed to set help view: %w", err)
			}
		}

		return nil
	} else {
		gui.DeleteView(g.views.helpModal)
	}

	// Modal (centered popup for command logs)
	if g.modalOpen {
		modalWidth := maxX - 10
		modalHeight := 15
		if modalHeight > maxY-6 {
			modalHeight = maxY - 6
		}
		modalX := (maxX - modalWidth) / 2
		modalY := (maxY - modalHeight) / 2

		if v, err := gui.SetView(g.views.modal, modalX, modalY, modalX+modalWidth, modalY+modalHeight, 0); err != nil {
			if !errors.Is(err, gocui.ErrUnknownView) {
				return err
			}
			v.Title = " Command Log "
			v.BgColor = gocui.ColorDefault
			v.FgColor = gocui.ColorDefault
			v.SelBgColor = gocui.ColorDefault
			v.SelFgColor = gocui.ColorDefault
			v.Wrap = true
		}

		if v, err := gui.View(g.views.modal); err == nil {
			v.Clear()
			if len(g.commandHistory) == 0 {
				fmt.Fprintln(v, "  No commands yet")
			} else {
				for _, cmd := range g.commandHistory {
					statusColor := "\033[32m" // Green
					if cmd.Status == "error" {
						statusColor = "\033[31m" // Red
					} else if cmd.Status == "running" {
						statusColor = "\033[33m" // Yellow
					}
					fmt.Fprintf(v, "  [%s] %s%s\033[0m: %s\n", cmd.Timestamp, statusColor, cmd.Command, cmd.Description)
				}
			}
			fmt.Fprintln(v, "")
			fmt.Fprintln(v, "  \033[36mPress Esc or @ to close\033[0m")
			if _, err := gui.SetCurrentView(g.views.modal); err != nil {
				return fmt.Errorf("failed to set modal view: %w", err)
			}
		}

		return nil
	} else {
		// Delete modal if it exists
		gui.DeleteView(g.views.modal)
	}

	// Set current view
	viewName := g.views.projects
	switch g.currentColumn {
	case "collections":
		viewName = g.views.collections
	case "tree":
		viewName = g.views.tree
	case "details":
		viewName = g.views.details
	}
	if _, err := gui.SetCurrentView(viewName); err != nil {
		return fmt.Errorf("failed to set current view '%s': %w", viewName, err)
	}

	return nil
}

func (g *Gui) updateProjectsView(v *gocui.View) {
	v.Clear()

	filtered := g.getFilteredProjects()

	// Enable highlight when this view is focused
	v.Highlight = g.currentColumn == "projects" && len(filtered) > 0

	// Project icon with spacing
	icon := icons.PROJECT_ICON
	if icon != "" {
		icon = icon + " "
	}

	// When collapsed (not focused), show only the selected project
	if g.currentColumn != "projects" {
		if len(filtered) > 0 && g.selectedProjectIndex < len(filtered) {
			project := filtered[g.selectedProjectIndex]
			fmt.Fprintf(v, "%s*\033[0m %s%s", g.getActiveColorCode(), icon, project.DisplayName)
		}
		return
	}

	// Expanded view - show filtered projects
	for _, project := range filtered {
		if project.ID == g.currentProject {
			fmt.Fprintf(v, "%s*\033[0m %s%s\n", g.getActiveColorCode(), icon, project.DisplayName)
		} else {
			fmt.Fprintf(v, "  %s%s\n", icon, project.DisplayName)
		}
	}

	// Handle scrolling and set cursor for highlight
	if len(filtered) > 0 {
		// Clamp selection to filtered list
		if g.selectedProjectIndex >= len(filtered) {
			g.selectedProjectIndex = len(filtered) - 1
		}
		v.FocusPoint(0, g.selectedProjectIndex, true)
	}
}

func (g *Gui) updateCollectionsView(v *gocui.View) {
	v.Clear()

	filtered := g.getFilteredCollections()

	// Enable highlight when this view is focused
	v.Highlight = g.currentColumn == "collections" && len(filtered) > 0

	if len(filtered) == 0 {
		return
	}

	for _, col := range filtered {
		icon := icons.COLLECTION_ICON
		if icon != "" {
			icon = icon + " "
		}
		if col.Name == g.currentCollection {
			fmt.Fprintf(v, "%s*\033[0m %s%s\n", g.getActiveColorCode(), icon, col.Name)
		} else {
			fmt.Fprintf(v, "  %s%s\n", icon, col.Name)
		}
	}

	// Handle scrolling and set cursor for highlight
	if len(filtered) > 0 {
		// Clamp selection to filtered list
		if g.selectedCollectionIdx >= len(filtered) {
			g.selectedCollectionIdx = len(filtered) - 1
		}
		v.FocusPoint(0, g.selectedCollectionIdx, true)
	}
}

func (g *Gui) updateTreeView(v *gocui.View) {
	v.Clear()

	filtered := g.getFilteredTreeNodes()

	// Enable highlight when this view is focused
	v.Highlight = g.currentColumn == "tree" && len(filtered) > 0

	if len(filtered) == 0 {
		return
	}

	for _, node := range filtered {
		// Build indentation
		indent := strings.Repeat("  ", node.Depth)

		// Choose icon based on type and expanded state
		icon := icons.DOCUMENT
		if node.Type == "collection" {
			if node.Expanded {
				icon = icons.FOLDER_OPEN
			} else {
				icon = icons.FOLDER_CLOSED
			}
		}

		// Add spacing after icon if present
		if icon != "" {
			icon = icon + " "
		}

		// Tree connector
		connector := ""
		if node.Depth > 0 {
			connector = "└─"
		}

		// Show colored * for currently selected document
		if node.Path == g.currentDocPath {
			fmt.Fprintf(v, "%s*%s%s%s%s%s\n", g.getActiveColorCode(), "\033[0m", indent, connector, icon, node.Name)
		} else {
			fmt.Fprintf(v, " %s%s%s%s\n", indent, connector, icon, node.Name)
		}
	}

	// Handle scrolling and set cursor for highlight
	if len(filtered) > 0 {
		// Clamp selection to filtered list
		if g.selectedTreeIdx >= len(filtered) {
			g.selectedTreeIdx = len(filtered) - 1
		}
		v.FocusPoint(0, g.selectedTreeIdx, true)
	}
}

func (g *Gui) updateDetailsView(v *gocui.View) {
	// Show document data if available (highest priority)
	if g.currentDocData != nil {
		// When filtering details, always re-render to apply filter
		detailsFilter := g.getDetailsFilter()
		if detailsFilter != "" {
			g.renderFilteredDetails(v)
			return
		}

		// Use cached content if document hasn't changed
		if g.cachedDetailsDocPath == g.currentDocPath && g.cachedDetailsContent != "" {
			v.SetContent(g.cachedDetailsContent)
			return
		}

		// Render and cache the colorized JSON
		data, err := json.MarshalIndent(g.currentDocData, "", "  ")
		if err != nil {
			v.SetContent(fmt.Sprintf("Error formatting data: %v\n", err))
			return
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("\033[36m─── %s ───\033[0m\n\n", g.currentDocPath))
		content.WriteString(colorizeJSON(string(data)))

		g.cachedDetailsContent = content.String()
		g.cachedDetailsDocPath = g.currentDocPath
		v.SetContent(g.cachedDetailsContent)
		return
	}

	// Clear cache when not showing document
	g.cachedDetailsContent = ""
	g.cachedDetailsDocPath = ""

	v.Clear()

	// Show fetched project details if available
	if g.currentProjectInfo != nil && g.currentColumn == "projects" {
		g.showFetchedProjectDetails(v)
		return
	}

	// Show contextual info based on current column
	switch g.currentColumn {
	case "projects":
		g.showProjectDetails(v)
	case "collections":
		g.showCollectionDetails(v)
	case "tree":
		g.showTreeNodeDetails(v)
	default:
		g.showWelcome(v)
	}
}

func (g *Gui) showProjectDetails(v *gocui.View) {
	filtered := g.getFilteredProjects()
	if len(filtered) == 0 || g.selectedProjectIndex >= len(filtered) {
		g.showWelcome(v)
		return
	}

	project := filtered[g.selectedProjectIndex]

	fmt.Fprintln(v, "\033[36m─── Project Info ───\033[0m")
	fmt.Fprintln(v, "")
	fmt.Fprintf(v, "  \033[33mID:\033[0m          %s\n", project.ID)
	fmt.Fprintf(v, "  \033[33mName:\033[0m        %s\n", project.DisplayName)
	fmt.Fprintln(v, "")
	fmt.Fprintln(v, "\033[90m  Press Enter for more details\033[0m")
	fmt.Fprintln(v, "\033[90m  Press Space to select project\033[0m")
}

func (g *Gui) showFetchedProjectDetails(v *gocui.View) {
	info := g.currentProjectInfo

	fmt.Fprintln(v, "\033[36m─── Project Details ───\033[0m")
	fmt.Fprintln(v, "")
	fmt.Fprintf(v, "  \033[33mProject ID:\033[0m      %s\n", info.ProjectID)
	fmt.Fprintf(v, "  \033[33mDisplay Name:\033[0m    %s\n", info.DisplayName)
	if info.ProjectNumber != "" {
		fmt.Fprintf(v, "  \033[33mProject Number:\033[0m  %s\n", info.ProjectNumber)
	}
	fmt.Fprintln(v, "")

	// Resources section
	if info.Resources.LocationID != "" || info.Resources.StorageBucket != "" ||
		info.Resources.HostingSite != "" || info.Resources.RealtimeDatabaseInstance != "" {
		fmt.Fprintln(v, "\033[36m─── Resources ───\033[0m")
		fmt.Fprintln(v, "")
		if info.Resources.LocationID != "" {
			fmt.Fprintf(v, "  \033[33mLocation:\033[0m        %s\n", info.Resources.LocationID)
		}
		if info.Resources.StorageBucket != "" {
			fmt.Fprintf(v, "  \033[33mStorage:\033[0m         %s\n", info.Resources.StorageBucket)
		}
		if info.Resources.HostingSite != "" {
			fmt.Fprintf(v, "  \033[33mHosting:\033[0m         %s\n", info.Resources.HostingSite)
		}
		if info.Resources.RealtimeDatabaseInstance != "" {
			fmt.Fprintf(v, "  \033[33mRTDB:\033[0m            %s\n", info.Resources.RealtimeDatabaseInstance)
		}
		fmt.Fprintln(v, "")
	}

	fmt.Fprintln(v, "\033[90m  Press Space to select project\033[0m")
}

func (g *Gui) showCollectionDetails(v *gocui.View) {
	filtered := g.getFilteredCollections()
	if len(filtered) == 0 || g.selectedCollectionIdx >= len(filtered) {
		fmt.Fprintln(v, "\033[36m─── Collections ───\033[0m")
		fmt.Fprintln(v, "")
		fmt.Fprintln(v, "\033[90m  No collections found\033[0m")
		fmt.Fprintln(v, "")
		fmt.Fprintln(v, "\033[90m  Select a project first\033[0m")
		return
	}

	collection := filtered[g.selectedCollectionIdx]

	fmt.Fprintln(v, "\033[36m─── Collection Info ───\033[0m")
	fmt.Fprintln(v, "")
	fmt.Fprintf(v, "  \033[33mName:\033[0m        %s\n", collection.Name)
	fmt.Fprintf(v, "  \033[33mPath:\033[0m        /%s\n", collection.Path)
	fmt.Fprintln(v, "")
	fmt.Fprintln(v, "\033[90m  Press Space to browse documents\033[0m")
}

func (g *Gui) showTreeNodeDetails(v *gocui.View) {
	filtered := g.getFilteredTreeNodes()
	if len(filtered) == 0 || g.selectedTreeIdx >= len(filtered) {
		fmt.Fprintln(v, "\033[36m─── Tree ───\033[0m")
		fmt.Fprintln(v, "")
		fmt.Fprintln(v, "\033[90m  No documents loaded\033[0m")
		fmt.Fprintln(v, "")
		fmt.Fprintln(v, "\033[90m  Select a collection first\033[0m")
		return
	}

	node := filtered[g.selectedTreeIdx]

	fmt.Fprintln(v, "\033[36m─── Node Info ───\033[0m")
	fmt.Fprintln(v, "")
	fmt.Fprintf(v, "  \033[33mName:\033[0m        %s\n", node.Name)
	fmt.Fprintf(v, "  \033[33mType:\033[0m        %s\n", node.Type)
	fmt.Fprintf(v, "  \033[33mPath:\033[0m        /%s\n", node.Path)
	fmt.Fprintln(v, "")
	if node.Type == "document" {
		fmt.Fprintln(v, "\033[90m  Press Space to view document data\033[0m")
	} else {
		fmt.Fprintln(v, "\033[90m  Press Space to expand collection\033[0m")
	}
}

func (g *Gui) showWelcome(v *gocui.View) {
	fmt.Fprintln(v, "")
	fmt.Fprintln(v, "\033[33m           ,")
	fmt.Fprintln(v, "\033[33m          /|\\")
	fmt.Fprintln(v, "\033[33m         / | \\")
	fmt.Fprintln(v, "\033[38;5;208m        /  |  \\")
	fmt.Fprintln(v, "\033[38;5;208m       /   |   \\")
	fmt.Fprintln(v, "\033[38;5;196m      /    |    \\")
	fmt.Fprintln(v, "\033[38;5;196m     /     |     \\")
	fmt.Fprintln(v, "\033[38;5;196m    (      |      )")
	fmt.Fprintln(v, "\033[38;5;208m     \\     |     /")
	fmt.Fprintln(v, "\033[33m      \\    |    /")
	fmt.Fprintln(v, "\033[33m       \\   |   /")
	fmt.Fprintln(v, "\033[0m        \\__|__/")
	fmt.Fprintln(v, "")
	fmt.Fprintf(v, "\033[36m  %s  L A Z Y F I R E\033[0m\n", icons.FIREBASE_ICON)
	fmt.Fprintln(v, "")
	fmt.Fprintln(v, "\033[90m   Select a project to start\033[0m")
}

func (g *Gui) updateCommandsView(v *gocui.View) {
	v.Clear()

	if len(g.commandHistory) == 0 {
		return
	}

	// Show last command
	cmd := g.commandHistory[len(g.commandHistory)-1]

	var statusIcon, statusColor string
	switch cmd.Status {
	case "running":
		statusIcon = icons.LOADING
		statusColor = "\033[33m" // Yellow
	case "error":
		statusIcon = icons.ERROR
		statusColor = "\033[31m" // Red
	case "success":
		statusIcon = icons.SUCCESS
		statusColor = "\033[32m" // Green
	default:
		statusIcon = "•"
		statusColor = "\033[0m"
	}

	fmt.Fprintf(v, "%s%s %s\033[0m %s",
		statusColor,
		statusIcon,
		cmd.Command,
		cmd.Description)
}

func (g *Gui) updateHelpView(v *gocui.View) {
	v.Clear()

	// Show filter input when typing
	if g.filterInputActive {
		panelName := g.getPanelNameFor(g.filterInputPanel)
		filterPrompt := fmt.Sprintf(" \033[33mFilter %s:\033[0m %s\033[7m \033[0m", panelName, g.filterInputText)
		hints := "  \033[90m(Enter to select, Esc to cancel)\033[0m"
		fmt.Fprintf(v, "%s%s", filterPrompt, hints)
		return
	}

	// Show filter status when panel has committed filter
	if filter := g.getFilterForPanel(g.currentColumn); filter != "" {
		panelName := g.getPanelNameFor(g.currentColumn)
		fmt.Fprintf(v, " \033[33m%s filtered:\033[0m '%s'  \033[90m(Esc to clear filter)\033[0m", panelName, filter)
		return
	}

	helpText := " \033[36m←/→\033[0m cols  \033[36mj/k\033[0m move  \033[33mspace\033[0m select  \033[32mc\033[0m copy  \033[32ms\033[0m save  \033[35m/\033[0m filter  \033[35m?\033[0m help  \033[31mq\033[0m quit"
	versionText := fmt.Sprintf("\033[90mv%s\033[0m ", g.version)

	// Calculate padding to right-align version
	width, _ := v.Size()
	helpLen := 85 // Approximate visible length without ANSI codes
	versionLen := len(g.version) + 2
	padding := width - helpLen - versionLen
	if padding < 1 {
		padding = 1
	}

	fmt.Fprintf(v, "%s%*s%s", helpText, padding, "", versionText)
}

func (g *Gui) setKeybindings() error {
	// Quit
	if err := g.g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, g.quit); err != nil {
		return err
	}
	if err := g.g.SetKeybinding("", 'q', gocui.ModNone, g.quit); err != nil {
		return err
	}

	// Refresh
	if err := g.g.SetKeybinding("", 'r', gocui.ModNone, g.refresh); err != nil {
		return err
	}

	// Escape
	if err := g.g.SetKeybinding("", gocui.KeyEsc, gocui.ModNone, g.handleEscape); err != nil {
		return err
	}

	// Navigation up/down
	if err := g.g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, g.cursorDown); err != nil {
		return err
	}
	if err := g.g.SetKeybinding("", 'j', gocui.ModNone, g.cursorDown); err != nil {
		return err
	}
	if err := g.g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, g.cursorUp); err != nil {
		return err
	}
	if err := g.g.SetKeybinding("", 'k', gocui.ModNone, g.cursorUp); err != nil {
		return err
	}

	// Column switching left/right
	if err := g.g.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone, g.columnLeft); err != nil {
		return err
	}
	if err := g.g.SetKeybinding("", 'h', gocui.ModNone, g.columnLeft); err != nil {
		return err
	}
	if err := g.g.SetKeybinding("", gocui.KeyArrowRight, gocui.ModNone, g.columnRight); err != nil {
		return err
	}
	if err := g.g.SetKeybinding("", 'l', gocui.ModNone, g.columnRight); err != nil {
		return err
	}

	// Tab to cycle columns
	if err := g.g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, g.nextColumn); err != nil {
		return err
	}

	// Space to select/expand
	if err := g.g.SetKeybinding("", gocui.KeySpace, gocui.ModNone, g.handleSpace); err != nil {
		return err
	}

	// Enter to view details
	if err := g.g.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, g.handleEnter); err != nil {
		return err
	}

	// @ to toggle command log modal
	if err := g.g.SetKeybinding("", '@', gocui.ModNone, g.toggleModal); err != nil {
		return err
	}

	// c to copy JSON to clipboard (when document is selected)
	if err := g.g.SetKeybinding("", 'c', gocui.ModNone, g.copyJSON); err != nil {
		return err
	}

	// s to save JSON to file (when document is selected)
	if err := g.g.SetKeybinding("", 's', gocui.ModNone, g.saveJSON); err != nil {
		return err
	}

	// ? to show help
	if err := g.g.SetKeybinding("", '?', gocui.ModNone, g.toggleHelp); err != nil {
		return err
	}

	// / to start filter
	if err := g.g.SetKeybinding("", '/', gocui.ModNone, g.startFilter); err != nil {
		return err
	}

	// Backspace for filter input
	if err := g.g.SetKeybinding("", gocui.KeyBackspace, gocui.ModNone, g.handleFilterBackspace); err != nil {
		return err
	}
	if err := g.g.SetKeybinding("", gocui.KeyBackspace2, gocui.ModNone, g.handleFilterBackspace); err != nil {
		return err
	}

	// Set up filter character input handlers (when in filter mode, these add to filter text)
	for _, ch := range "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_. " {
		ch := ch // capture for closure
		if err := g.g.SetKeybinding("", ch, gocui.ModNone, g.makeFilterCharHandler(ch)); err != nil {
			return err
		}
	}

	// Mouse click bindings for each panel
	if err := g.g.SetKeybinding(g.views.helpModal, gocui.MouseLeft, gocui.ModNone, g.handleHelpClick); err != nil {
		return err
	}
	if err := g.g.SetKeybinding(g.views.projects, gocui.MouseLeft, gocui.ModNone, g.handleProjectsClick); err != nil {
		return err
	}
	if err := g.g.SetKeybinding(g.views.collections, gocui.MouseLeft, gocui.ModNone, g.handleCollectionsClick); err != nil {
		return err
	}
	if err := g.g.SetKeybinding(g.views.tree, gocui.MouseLeft, gocui.ModNone, g.handleTreeClick); err != nil {
		return err
	}
	if err := g.g.SetKeybinding(g.views.details, gocui.MouseLeft, gocui.ModNone, g.handleDetailsClick); err != nil {
		return err
	}
	if err := g.g.SetKeybinding(g.views.commands, gocui.MouseLeft, gocui.ModNone, g.handleOutsideClick); err != nil {
		return err
	}
	if err := g.g.SetKeybinding(g.views.help, gocui.MouseLeft, gocui.ModNone, g.handleOutsideClick); err != nil {
		return err
	}
	if err := g.g.SetKeybinding(g.views.background, gocui.MouseLeft, gocui.ModNone, g.handleOutsideClick); err != nil {
		return err
	}

	return nil
}

// Event handlers
func (g *Gui) quit(gui *gocui.Gui, v *gocui.View) error {
	if g.filterInputActive {
		return g.addFilterChar(gui, 'q')
	}
	return gocui.ErrQuit
}

// setFocus sets the current column and updates gocui's current view
func (g *Gui) setFocus(gui *gocui.Gui, column string) error {
	g.currentColumn = column
	if _, err := gui.SetCurrentView(column); err != nil {
		return err
	}
	return nil
}

func (g *Gui) columnLeft(gui *gocui.Gui, v *gocui.View) error {
	if g.filterInputActive {
		return g.addFilterChar(gui, 'h')
	}
	// Left goes up: details → tree → collections → projects → details (wrap)
	var newColumn string
	switch g.currentColumn {
	case "projects":
		newColumn = "details" // wrap to right
	case "collections":
		newColumn = "projects"
	case "tree":
		newColumn = "collections"
	case "details":
		newColumn = "tree"
	}
	if err := g.setFocus(gui, newColumn); err != nil {
		return err
	}
	return nil
}

func (g *Gui) columnRight(gui *gocui.Gui, v *gocui.View) error {
	if g.filterInputActive {
		return g.addFilterChar(gui, 'l')
	}
	// Right goes down: projects → collections → tree → details → projects (wrap)
	var newColumn string
	switch g.currentColumn {
	case "projects":
		newColumn = "collections"
	case "collections":
		newColumn = "tree"
	case "tree":
		newColumn = "details"
	case "details":
		newColumn = "projects" // wrap to left
	}
	if err := g.setFocus(gui, newColumn); err != nil {
		return err
	}
	return nil
}

func (g *Gui) nextColumn(gui *gocui.Gui, v *gocui.View) error {
	if g.filterInputActive {
		return nil
	}
	var newColumn string
	switch g.currentColumn {
	case "projects":
		newColumn = "collections"
	case "collections":
		newColumn = "tree"
	case "tree":
		newColumn = "details"
	case "details":
		newColumn = "projects"
	}
	if err := g.setFocus(gui, newColumn); err != nil {
		return err
	}
	return nil
}

func (g *Gui) cursorDown(gui *gocui.Gui, v *gocui.View) error {
	if g.filterInputActive {
		return g.addFilterChar(gui, 'j')
	}
	// Handle help modal scrolling
	if g.helpOpen && g.helpPopup != nil {
		g.helpPopup.MoveDown()
		return g.Layout(gui)
	}

	switch g.currentColumn {
	case "projects":
		filtered := g.getFilteredProjects()
		if g.selectedProjectIndex < len(filtered)-1 {
			g.selectedProjectIndex++
			g.currentProjectInfo = nil // Clear details when changing selection
		}
	case "collections":
		filtered := g.getFilteredCollections()
		if g.selectedCollectionIdx < len(filtered)-1 {
			g.selectedCollectionIdx++
		}
	case "tree":
		filtered := g.getFilteredTreeNodes()
		if g.selectedTreeIdx < len(filtered)-1 {
			g.selectedTreeIdx++
		}
	case "details":
		g.detailsScrollPos++
	}
	return g.Layout(gui)
}

func (g *Gui) cursorUp(gui *gocui.Gui, v *gocui.View) error {
	if g.filterInputActive {
		return g.addFilterChar(gui, 'k')
	}
	// Handle help modal scrolling
	if g.helpOpen && g.helpPopup != nil {
		g.helpPopup.MoveUp()
		return g.Layout(gui)
	}

	switch g.currentColumn {
	case "projects":
		if g.selectedProjectIndex > 0 {
			g.selectedProjectIndex--
			g.currentProjectInfo = nil // Clear details when changing selection
		}
	case "collections":
		if g.selectedCollectionIdx > 0 {
			g.selectedCollectionIdx--
		}
	case "tree":
		if g.selectedTreeIdx > 0 {
			g.selectedTreeIdx--
		}
	case "details":
		if g.detailsScrollPos > 0 {
			g.detailsScrollPos--
		}
	}
	return g.Layout(gui)
}

func (g *Gui) copyJSON(gui *gocui.Gui, v *gocui.View) error {
	if g.filterInputActive {
		return g.addFilterChar(gui, 'c')
	}
	docData, docPath, err := g.getDocumentToCopy()
	if err != nil {
		g.logCommand("copy", err.Error(), "error")
		return nil
	}

	data, err := json.MarshalIndent(docData, "", "  ")
	if err != nil {
		g.logCommand("copy", fmt.Sprintf("Failed to marshal JSON: %v", err), "error")
		return nil
	}

	// Copy to clipboard using platform-specific command
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		g.logCommand("copy", "Clipboard not supported on this platform", "error")
		return nil
	}

	cmd.Stdin = strings.NewReader(string(data))
	if err := cmd.Run(); err != nil {
		g.logCommand("copy", fmt.Sprintf("Failed to copy: %v", err), "error")
		return nil
	}

	g.logCommand("copy", fmt.Sprintf("Copied %s to clipboard", docPath), "success")
	return nil
}

func (g *Gui) saveJSON(gui *gocui.Gui, v *gocui.View) error {
	if g.filterInputActive {
		return g.addFilterChar(gui, 's')
	}
	docData, docPath, err := g.getDocumentToCopy()
	if err != nil {
		g.logCommand("save", err.Error(), "error")
		return nil
	}

	data, err := json.MarshalIndent(docData, "", "  ")
	if err != nil {
		g.logCommand("save", fmt.Sprintf("Failed to marshal JSON: %v", err), "error")
		return nil
	}

	// Create filename from document path
	safePath := strings.ReplaceAll(docPath, "/", "_")
	filename := fmt.Sprintf("%s.json", safePath)

	// Save to Downloads directory
	home, _ := os.UserHomeDir()
	downloadDir := filepath.Join(home, "Downloads")
	fullPath := filepath.Join(downloadDir, filename)

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		g.logCommand("save", fmt.Sprintf("Failed to save: %v", err), "error")
		return nil
	}

	g.logCommand("save", fmt.Sprintf("Saved to %s", fullPath), "success")
	return nil
}

// getDocumentToCopy returns the document data to copy/save.
// If on tree panel with a document highlighted, fetches that document and displays it.
// Otherwise returns the currently displayed document.
func (g *Gui) getDocumentToCopy() (map[string]interface{}, string, error) {
	// If on tree panel with a document node selected, fetch that document
	filtered := g.getFilteredTreeNodes()
	if g.currentColumn == "tree" && len(filtered) > 0 && g.selectedTreeIdx < len(filtered) {
		node := filtered[g.selectedTreeIdx]
		if node.Type == "document" {
			// Fetch the document
			doc, err := g.firebaseClient.GetDocument(node.Path)
			if err != nil {
				return nil, "", fmt.Errorf("Failed to fetch document: %v", err)
			}
			// Also display it in Details panel
			g.currentDocData = doc.Data
			g.currentDocPath = node.Path
			return doc.Data, node.Path, nil
		}
		return nil, "", fmt.Errorf("Selected item is a collection, not a document")
	}

	// Otherwise use the currently displayed document
	if g.currentDocData != nil {
		return g.currentDocData, g.currentDocPath, nil
	}

	return nil, "", fmt.Errorf("No document selected")
}

func (g *Gui) handleSpace(gui *gocui.Gui, v *gocui.View) error {
	if g.filterInputActive {
		// Add space to filter text
		g.filterInputText += " "
		return g.Layout(gui)
	}
	switch g.currentColumn {
	case "projects":
		return g.selectProject(gui)
	case "collections":
		return g.selectCollection(gui)
	case "tree":
		return g.selectTreeNode(gui)
	}
	return nil
}

func (g *Gui) handleEnter(gui *gocui.Gui, v *gocui.View) error {
	// Commit filter if in filter input mode
	if g.filterInputActive {
		return g.commitFilter(gui)
	}

	// Handle help popup action execution
	if g.helpOpen && g.helpPopup != nil {
		item := g.helpPopup.GetSelectedItem()
		if item != nil && item.Action != nil {
			g.helpOpen = false
			g.helpPopup = nil
			return item.Action(gui, v)
		}
		return nil
	}

	switch g.currentColumn {
	case "projects":
		return g.fetchProjectDetails(gui)
	}
	return nil
}

func (g *Gui) fetchProjectDetails(gui *gocui.Gui) error {
	filtered := g.getFilteredProjects()
	if g.selectedProjectIndex >= len(filtered) {
		return nil
	}

	project := filtered[g.selectedProjectIndex]
	g.logCommand("api", fmt.Sprintf("GetProjectDetails(%s)...", project.ID), "running")

	go func() {
		details, err := g.firebaseClient.GetProjectDetails(project.ID)
		g.g.Update(func(gui *gocui.Gui) error {
			if err != nil {
				g.logCommand("api", fmt.Sprintf("GetProjectDetails failed: %v", err), "error")
				return nil
			}
			g.currentProjectInfo = details
			g.currentDocData = nil // Clear doc data so project info shows
			g.logCommand("api", fmt.Sprintf("GetProjectDetails(%s) → success", project.ID), "success")
			return nil
		})
	}()

	return nil
}

func (g *Gui) selectProject(gui *gocui.Gui) error {
	filtered := g.getFilteredProjects()
	if g.selectedProjectIndex >= len(filtered) {
		return nil
	}

	selectedProject := filtered[g.selectedProjectIndex]

	// Show loading state immediately
	g.logCommand("api", fmt.Sprintf("ListCollections(%s) loading...", selectedProject.ID), "running")

	// Run API call in goroutine so UI updates immediately
	go func() {
		if err := g.firebaseClient.SetCurrentProject(selectedProject.ID); err != nil {
			g.g.Update(func(gui *gocui.Gui) error {
				g.logCommand("api", fmt.Sprintf("SetProject failed: %v", err), "error")
				return nil
			})
			return
		}

		g.currentProject = selectedProject.ID

		// Clear state
		g.collections = nil
		g.treeNodes = nil
		g.currentDocData = nil
		g.currentCollection = ""
		g.currentDocPath = ""
		g.selectedCollectionIdx = 0
		g.selectedTreeIdx = 0

		if err := g.loadCollections(); err != nil {
			g.g.Update(func(gui *gocui.Gui) error {
				g.logCommand("api", fmt.Sprintf("ListCollections failed: %v", err), "error")
				return nil
			})
			return
		}

		g.g.Update(func(gui *gocui.Gui) error {
			g.logCommand("api", fmt.Sprintf("ListCollections(%s) → %d collections", selectedProject.ID, len(g.collections)), "success")
			return nil
		})
	}()

	return nil
}

func (g *Gui) selectCollection(gui *gocui.Gui) error {
	filtered := g.getFilteredCollections()
	if g.selectedCollectionIdx >= len(filtered) {
		return nil
	}

	collection := filtered[g.selectedCollectionIdx]
	g.currentCollection = collection.Name

	// Show loading state immediately
	g.logCommand("api", fmt.Sprintf("ListDocuments(%s) loading...", collection.Name), "running")

	// Run API call in goroutine so UI updates immediately
	go func() {
		docs, err := g.firebaseClient.ListDocuments(collection.Name, 50)
		if err != nil {
			g.g.Update(func(gui *gocui.Gui) error {
				g.logCommand("api", fmt.Sprintf("ListDocuments failed: %v", err), "error")
				return nil
			})
			return
		}

		g.g.Update(func(gui *gocui.Gui) error {
			// Build tree nodes from documents
			g.treeNodes = nil
			g.expandedPaths = make(map[string]bool)

			for _, doc := range docs {
				node := TreeNode{
					Path:        doc.Path,
					Name:        doc.ID,
					Type:        "document",
					Depth:       0,
					HasChildren: true,
					Expanded:    false,
				}
				g.treeNodes = append(g.treeNodes, node)
			}

			g.selectedTreeIdx = 0
			g.logCommand("api", fmt.Sprintf("ListDocuments(%s) → %d docs", collection.Name, len(docs)), "success")
			return nil
		})
	}()

	return nil
}

func (g *Gui) selectTreeNode(gui *gocui.Gui) error {
	filtered := g.getFilteredTreeNodes()
	if g.selectedTreeIdx >= len(filtered) {
		return nil
	}

	// Get node from filtered list
	selectedNode := filtered[g.selectedTreeIdx]
	nodePath := selectedNode.Path
	nodeName := selectedNode.Name
	nodeDepth := selectedNode.Depth
	nodeType := selectedNode.Type

	// Find original index for modifications
	originalIdx := g.getOriginalTreeNodeIndex(g.selectedTreeIdx)
	if originalIdx == -1 {
		return nil
	}
	node := &g.treeNodes[originalIdx]
	nodeIdx := originalIdx

	if nodeType == "document" {
		if node.Expanded {
			// Collapse: remove children (synchronous, no API call)
			g.collapseNode(nodeIdx)
			node.Expanded = false
			return nil
		}

		// Show loading state immediately
		g.logCommand("api", fmt.Sprintf("GetDocument(%s) loading...", nodePath), "running")

		// Run API calls in goroutine
		go func() {
			// Load document data
			doc, err := g.firebaseClient.GetDocument(nodePath)
			if err != nil {
				g.g.Update(func(gui *gocui.Gui) error {
					g.logCommand("api", fmt.Sprintf("GetDocument failed: %v", err), "error")
					return nil
				})
				return
			}

			// Load subcollections
			subcols, err := g.firebaseClient.ListSubcollections(nodePath)

			g.g.Update(func(gui *gocui.Gui) error {
				g.currentDocPath = nodePath
				g.currentDocData = doc.Data

				if err != nil || len(subcols) == 0 {
					g.logCommand("api", fmt.Sprintf("GetDocument(%s) → loaded", nodeName), "success")
					return nil
				}

				// Insert subcollection nodes after current node
				if nodeIdx < len(g.treeNodes) {
					newNodes := make([]TreeNode, 0, len(g.treeNodes)+len(subcols))
					newNodes = append(newNodes, g.treeNodes[:nodeIdx+1]...)

					for _, sub := range subcols {
						subNode := TreeNode{
							Path:        sub.Path,
							Name:        sub.Name,
							Type:        "collection",
							Depth:       nodeDepth + 1,
							HasChildren: true,
							Expanded:    false,
						}
						newNodes = append(newNodes, subNode)
					}

					newNodes = append(newNodes, g.treeNodes[nodeIdx+1:]...)
					g.treeNodes = newNodes
					if nodeIdx < len(g.treeNodes) {
						g.treeNodes[nodeIdx].Expanded = true
					}
				}

				g.logCommand("api", fmt.Sprintf("GetDocument(%s) → %d subcols", nodeName, len(subcols)), "success")
				return nil
			})
		}()

	} else if nodeType == "collection" {
		if node.Expanded {
			// Collapse (synchronous, no API call)
			g.collapseNode(nodeIdx)
			node.Expanded = false
			return nil
		}

		// Show loading state immediately
		g.logCommand("api", fmt.Sprintf("ListDocuments(%s) loading...", nodePath), "running")

		// Run API call in goroutine
		go func() {
			docs, err := g.firebaseClient.ListDocuments(nodePath, 50)
			if err != nil {
				g.g.Update(func(gui *gocui.Gui) error {
					g.logCommand("api", fmt.Sprintf("ListDocuments failed: %v", err), "error")
					return nil
				})
				return
			}

			g.g.Update(func(gui *gocui.Gui) error {
				if len(docs) == 0 {
					g.logCommand("api", fmt.Sprintf("ListDocuments(%s) → empty", nodeName), "success")
					return nil
				}

				// Insert document nodes after current node
				if nodeIdx < len(g.treeNodes) {
					newNodes := make([]TreeNode, 0, len(g.treeNodes)+len(docs))
					newNodes = append(newNodes, g.treeNodes[:nodeIdx+1]...)

					for _, doc := range docs {
						docNode := TreeNode{
							Path:        doc.Path,
							Name:        doc.ID,
							Type:        "document",
							Depth:       nodeDepth + 1,
							HasChildren: true,
							Expanded:    false,
						}
						newNodes = append(newNodes, docNode)
					}

					newNodes = append(newNodes, g.treeNodes[nodeIdx+1:]...)
					g.treeNodes = newNodes
					if nodeIdx < len(g.treeNodes) {
						g.treeNodes[nodeIdx].Expanded = true
					}
				}

				g.logCommand("api", fmt.Sprintf("ListDocuments(%s) → %d docs", nodeName, len(docs)), "success")
				return nil
			})
		}()
	}

	return nil
}

func (g *Gui) collapseNode(idx int) {
	if idx >= len(g.treeNodes) {
		return
	}

	node := g.treeNodes[idx]
	nodeDepth := node.Depth

	// Find all children (nodes with greater depth that follow)
	endIdx := idx + 1
	for endIdx < len(g.treeNodes) && g.treeNodes[endIdx].Depth > nodeDepth {
		endIdx++
	}

	// Remove children
	if endIdx > idx+1 {
		g.treeNodes = append(g.treeNodes[:idx+1], g.treeNodes[endIdx:]...)
	}
}

func (g *Gui) handleEscape(gui *gocui.Gui, v *gocui.View) error {
	// Close help modal if open (first priority)
	if g.helpOpen {
		g.helpOpen = false
		g.helpPopup = nil
		return nil
	}

	// Close command modal if open
	if g.modalOpen {
		g.modalOpen = false
		return nil
	}

	// Cancel filter input if typing
	if g.filterInputActive {
		return g.cancelFilterInput(gui)
	}

	// Clear committed filter for current panel if it has one
	if g.hasActiveFilter(g.currentColumn) {
		return g.clearCurrentFilter(gui)
	}

	// Collapse currently selected tree node if expanded
	if g.currentColumn == "tree" && g.selectedTreeIdx < len(g.treeNodes) {
		node := &g.treeNodes[g.selectedTreeIdx]
		if node.Expanded {
			g.collapseNode(g.selectedTreeIdx)
			node.Expanded = false
			g.logCommand("Esc", "Collapsed node", "success")
			return nil
		}
	}

	return nil
}

func (g *Gui) toggleModal(gui *gocui.Gui, v *gocui.View) error {
	if g.filterInputActive {
		return nil
	}
	g.modalOpen = !g.modalOpen
	return g.Layout(gui)
}

func (g *Gui) toggleHelp(gui *gocui.Gui, v *gocui.View) error {
	if g.filterInputActive {
		return nil
	}
	g.helpOpen = !g.helpOpen
	if g.helpOpen {
		g.helpPopup = g.buildHelpPopup()
	} else {
		g.helpPopup = nil
	}
	return g.Layout(gui)
}

// buildHelpPopup creates the help popup with global and context-specific shortcuts
func (g *Gui) buildHelpPopup() *Popup {
	items := []PopupItem{
		{Key: "", Label: "Global", IsHeader: true},
		{Key: "←/→ h/l", Label: "Switch panels", Action: nil},
		{Key: "↑/↓ j/k", Label: "Move up/down", Action: nil},
		{Key: "Space", Label: "Select / Expand", Action: g.handleSpace},
		{Key: "/", Label: "Filter / Search", Action: g.startFilter},
		{Key: "Esc", Label: "Back / Collapse / Close", Action: g.handleEscape},
		{Key: "r", Label: "Refresh", Action: g.refresh},
		{Key: "@", Label: "Command log", Action: g.toggleModal},
		{Key: "?", Label: "This help", Action: nil},
		{Key: "q", Label: "Quit", Action: g.quit},
		{Key: "", Label: g.getPanelName(), IsHeader: true},
	}

	// Add context-specific items with their actions
	switch g.currentColumn {
	case "projects":
		items = append(items,
			PopupItem{Key: "Enter", Label: "Fetch project details", Action: g.fetchProjectDetailsAction},
			PopupItem{Key: "Space", Label: "Select project", Action: g.selectProjectAction},
		)
	case "collections":
		items = append(items,
			PopupItem{Key: "Space", Label: "Load documents", Action: g.selectCollectionAction},
		)
	case "tree":
		items = append(items,
			PopupItem{Key: "Space", Label: "View document / Expand", Action: g.selectTreeNodeAction},
			PopupItem{Key: "c", Label: "Copy JSON to clipboard", Action: g.copyJSON},
			PopupItem{Key: "s", Label: "Save JSON to Downloads", Action: g.saveJSON},
		)
	case "details":
		items = append(items,
			PopupItem{Key: "j/k", Label: "Scroll content", Action: nil},
			PopupItem{Key: "c", Label: "Copy JSON to clipboard", Action: g.copyJSON},
			PopupItem{Key: "s", Label: "Save JSON to Downloads", Action: g.saveJSON},
		)
	}

	return NewPopup("Keyboard Shortcuts", items, g.theme, g.views.helpModal)
}

// Action wrappers that close help first
func (g *Gui) selectProjectAction(gui *gocui.Gui, v *gocui.View) error {
	g.helpOpen = false
	g.helpPopup = nil
	return g.selectProject(gui)
}

func (g *Gui) selectCollectionAction(gui *gocui.Gui, v *gocui.View) error {
	g.helpOpen = false
	g.helpPopup = nil
	return g.selectCollection(gui)
}

func (g *Gui) selectTreeNodeAction(gui *gocui.Gui, v *gocui.View) error {
	g.helpOpen = false
	g.helpPopup = nil
	return g.selectTreeNode(gui)
}

func (g *Gui) fetchProjectDetailsAction(gui *gocui.Gui, v *gocui.View) error {
	g.helpOpen = false
	g.helpPopup = nil
	return g.fetchProjectDetails(gui)
}

func (g *Gui) renderHelpContent(v *gocui.View) {
	if g.helpPopup == nil {
		return
	}
	g.helpPopup.Render(v)
}

func (g *Gui) getPanelName() string {
	return g.getPanelNameFor(g.currentColumn)
}

func (g *Gui) getPanelNameFor(panel string) string {
	switch panel {
	case "projects":
		return "Projects"
	case "collections":
		return "Collections"
	case "tree":
		return "Tree"
	case "details":
		return "Details"
	default:
		return "Panel"
	}
}

func (g *Gui) refresh(gui *gocui.Gui, v *gocui.View) error {
	if g.filterInputActive {
		return g.addFilterChar(gui, 'r')
	}
	g.logCommand("r", "Refreshing...", "running")

	// Reload projects
	if err := g.loadProjects(); err != nil {
		g.logCommand("r", fmt.Sprintf("Failed: %v", err), "error")
		return g.Layout(gui)
	}

	// Reload collections if we have a project selected
	if g.currentProject != "" {
		if err := g.loadCollections(); err != nil {
			// Not fatal, just log it
			g.logCommand("r", "Collections reload failed", "error")
		}
	}

	g.logCommand("r", "Refreshed", "success")
	return g.Layout(gui)
}

// Mouse click handlers

func (g *Gui) handleHelpClick(gui *gocui.Gui, v *gocui.View) error {
	if g.helpPopup == nil {
		return nil
	}

	_, cy := v.Cursor()
	_, oy := v.Origin()
	clickedLine := cy + oy

	// Map clicked line to popup item index (accounting for the line content)
	if clickedLine >= 0 && clickedLine < len(g.helpPopup.Items) {
		item := &g.helpPopup.Items[clickedLine]
		if !item.IsHeader {
			g.helpPopup.SelectedIdx = clickedLine
		}
	}

	return g.Layout(gui)
}

func (g *Gui) handleProjectsClick(gui *gocui.Gui, v *gocui.View) error {
	// Close popup if open and ignore click
	if g.helpOpen {
		g.helpOpen = false
		g.helpPopup = nil
		return g.Layout(gui)
	}

	g.currentColumn = "projects"

	_, cy := v.Cursor()
	_, oy := v.Origin()
	clickedLine := cy + oy

	filtered := g.getFilteredProjects()
	if clickedLine >= 0 && clickedLine < len(filtered) {
		g.selectedProjectIndex = clickedLine
		g.currentProjectInfo = nil
	}

	return g.Layout(gui)
}

func (g *Gui) handleCollectionsClick(gui *gocui.Gui, v *gocui.View) error {
	// Close popup if open and ignore click
	if g.helpOpen {
		g.helpOpen = false
		g.helpPopup = nil
		return g.Layout(gui)
	}

	g.currentColumn = "collections"

	_, cy := v.Cursor()
	_, oy := v.Origin()
	clickedLine := cy + oy

	filtered := g.getFilteredCollections()
	if clickedLine >= 0 && clickedLine < len(filtered) {
		g.selectedCollectionIdx = clickedLine
	}

	return g.Layout(gui)
}

func (g *Gui) handleTreeClick(gui *gocui.Gui, v *gocui.View) error {
	// Close popup if open and ignore click
	if g.helpOpen {
		g.helpOpen = false
		g.helpPopup = nil
		return g.Layout(gui)
	}

	g.currentColumn = "tree"

	_, cy := v.Cursor()
	_, oy := v.Origin()
	clickedLine := cy + oy

	filtered := g.getFilteredTreeNodes()
	if clickedLine >= 0 && clickedLine < len(filtered) {
		g.selectedTreeIdx = clickedLine
	}

	return g.Layout(gui)
}

func (g *Gui) handleDetailsClick(gui *gocui.Gui, v *gocui.View) error {
	// Close popup if open and ignore click
	if g.helpOpen {
		g.helpOpen = false
		g.helpPopup = nil
		return g.Layout(gui)
	}

	g.currentColumn = "details"
	return g.Layout(gui)
}

func (g *Gui) handleOutsideClick(gui *gocui.Gui, v *gocui.View) error {
	// Close popup if open
	if g.helpOpen {
		g.helpOpen = false
		g.helpPopup = nil
		return g.Layout(gui)
	}
	return nil
}

// Filter functions

func (g *Gui) startFilter(gui *gocui.Gui, v *gocui.View) error {
	// Don't start filter if modal/help is open or already typing filter
	if g.helpOpen || g.modalOpen || g.filterInputActive {
		return nil
	}
	// Clear any existing committed filter for this panel
	switch g.currentColumn {
	case "projects":
		g.projectsFilter = ""
	case "collections":
		g.collectionsFilter = ""
	case "tree":
		g.treeFilter = ""
	case "details":
		g.detailsFilter = ""
	}
	g.filterInputActive = true
	g.filterInputPanel = g.currentColumn
	g.filterInputText = ""
	return g.Layout(gui)
}

func (g *Gui) commitFilter(gui *gocui.Gui) error {
	filterText := g.filterInputText
	panel := g.filterInputPanel

	// Save filter and exit input mode (filter stays active)
	switch panel {
	case "projects":
		g.projectsFilter = filterText
		g.selectedProjectIndex = 0 // Reset to first filtered item
	case "collections":
		g.collectionsFilter = filterText
		g.selectedCollectionIdx = 0
	case "tree":
		g.treeFilter = filterText
		g.selectedTreeIdx = 0
	case "details":
		g.detailsFilter = filterText
		g.detailsScrollPos = 0
	}

	// Exit input mode but keep filter active
	g.filterInputActive = false
	g.filterInputText = ""
	g.filterInputPanel = ""

	return g.Layout(gui)
}

func (g *Gui) isFilteringPanel(panel string) bool {
	return g.filterInputActive && g.filterInputPanel == panel
}

func (g *Gui) getFilterForPanel(panel string) string {
	switch panel {
	case "projects":
		return g.projectsFilter
	case "collections":
		return g.collectionsFilter
	case "tree":
		return g.treeFilter
	case "details":
		return g.detailsFilter
	}
	return ""
}

func (g *Gui) hasActiveFilter(panel string) bool {
	return g.getFilterForPanel(panel) != ""
}

func (g *Gui) clearCurrentFilter(gui *gocui.Gui) error {
	switch g.currentColumn {
	case "projects":
		g.projectsFilter = ""
		g.selectedProjectIndex = 0
	case "collections":
		g.collectionsFilter = ""
		g.selectedCollectionIdx = 0
	case "tree":
		g.treeFilter = ""
		g.selectedTreeIdx = 0
	case "details":
		g.detailsFilter = ""
		g.detailsScrollPos = 0
	}
	return g.Layout(gui)
}

func (g *Gui) cancelFilterInput(gui *gocui.Gui) error {
	g.filterInputActive = false
	g.filterInputText = ""
	g.filterInputPanel = ""
	return g.Layout(gui)
}

func (g *Gui) handleFilterBackspace(gui *gocui.Gui, v *gocui.View) error {
	if !g.filterInputActive {
		return nil
	}
	if len(g.filterInputText) > 0 {
		g.filterInputText = g.filterInputText[:len(g.filterInputText)-1]
	}
	return g.Layout(gui)
}

func (g *Gui) makeFilterCharHandler(ch rune) func(*gocui.Gui, *gocui.View) error {
	return func(gui *gocui.Gui, v *gocui.View) error {
		if !g.filterInputActive {
			return nil // Let normal keybindings handle it
		}
		g.filterInputText += string(ch)
		return g.Layout(gui)
	}
}

// addFilterChar adds a character to the filter input and refreshes layout
func (g *Gui) addFilterChar(gui *gocui.Gui, ch rune) error {
	g.filterInputText += string(ch)
	return g.Layout(gui)
}

// matchesFilter checks if text contains the filter string (case-insensitive)
func (g *Gui) matchesFilter(text, filter string) bool {
	if filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(text), strings.ToLower(filter))
}

// getFilteredProjects returns projects matching the current filter
func (g *Gui) getFilteredProjects() []firebase.Project {
	// Use input text while typing, otherwise use committed filter
	filter := g.projectsFilter
	if g.filterInputActive && g.filterInputPanel == "projects" {
		filter = g.filterInputText
	}
	if filter == "" {
		return g.projects
	}
	var filtered []firebase.Project
	for _, p := range g.projects {
		if g.matchesFilter(p.DisplayName, filter) || g.matchesFilter(p.ID, filter) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// getFilteredCollections returns collections matching the current filter
func (g *Gui) getFilteredCollections() []firebase.Collection {
	filter := g.collectionsFilter
	if g.filterInputActive && g.filterInputPanel == "collections" {
		filter = g.filterInputText
	}
	if filter == "" {
		return g.collections
	}
	var filtered []firebase.Collection
	for _, c := range g.collections {
		if g.matchesFilter(c.Name, filter) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// getFilteredTreeNodes returns tree nodes matching the current filter
func (g *Gui) getFilteredTreeNodes() []TreeNode {
	filter := g.treeFilter
	if g.filterInputActive && g.filterInputPanel == "tree" {
		filter = g.filterInputText
	}
	if filter == "" {
		return g.treeNodes
	}
	var filtered []TreeNode
	for _, n := range g.treeNodes {
		if g.matchesFilter(n.Name, filter) || g.matchesFilter(n.Path, filter) {
			filtered = append(filtered, n)
		}
	}
	return filtered
}

// getDetailsFilter returns the active filter for details panel
func (g *Gui) getDetailsFilter() string {
	if g.filterInputActive && g.filterInputPanel == "details" {
		return g.filterInputText
	}
	return g.detailsFilter
}

// getOriginalTreeNodeIndex maps a filtered index back to the original treeNodes index
func (g *Gui) getOriginalTreeNodeIndex(filteredIdx int) int {
	filtered := g.getFilteredTreeNodes()
	if filteredIdx < 0 || filteredIdx >= len(filtered) {
		return -1
	}
	targetPath := filtered[filteredIdx].Path
	for i, node := range g.treeNodes {
		if node.Path == targetPath {
			return i
		}
	}
	return -1
}

// renderFilteredDetails shows only JSON lines that match the filter
func (g *Gui) renderFilteredDetails(v *gocui.View) {
	filter := g.getDetailsFilter()
	data, err := json.MarshalIndent(g.currentDocData, "", "  ")
	if err != nil {
		v.SetContent(fmt.Sprintf("Error formatting data: %v\n", err))
		return
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("\033[36m─── %s (filtered) ───\033[0m\n\n", g.currentDocPath))

	lines := strings.Split(string(data), "\n")
	matchCount := 0
	for _, line := range lines {
		if g.matchesFilter(line, filter) {
			content.WriteString(colorizeLine(line))
			content.WriteString("\n")
			matchCount++
		}
	}

	if matchCount == 0 {
		content.WriteString("\033[90mNo matching lines\033[0m\n")
	}

	v.SetContent(content.String())
}
