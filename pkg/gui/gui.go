package gui

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jesseduffield/gocui"
	"github.com/mballabani/lazyfire/pkg/config"
	"github.com/mballabani/lazyfire/pkg/firebase"
)

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
	currentDocPath   string
	currentDocData   map[string]interface{}
	detailsScrollPos int

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
	}

	// Current column: "projects", "collections", "tree"
	currentColumn string

	// Modal state
	modalOpen bool

	// Loading state
	isLoading   bool
	loadingText string

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
	gui.views.background = "background"

	// Configure gocui
	g.Cursor = false
	g.Mouse = false
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
	// Focused panel gets more space (50%), unfocused share the rest (25% each)
	leftHeight := maxY - 3 // Leave room for help bar

	var projectsEnd, collectionsEnd int
	expandedHeight := leftHeight / 2      // 50% for focused
	collapsedHeight := leftHeight / 4     // 25% for unfocused

	switch g.currentColumn {
	case "projects":
		projectsEnd = expandedHeight
		collectionsEnd = expandedHeight + collapsedHeight
	case "collections":
		projectsEnd = collapsedHeight
		collectionsEnd = collapsedHeight + expandedHeight
	case "tree":
		projectsEnd = collapsedHeight
		collectionsEnd = collapsedHeight * 2
	default: // details or other
		// Equal distribution when not in left panels
		panelHeight := leftHeight / 3
		projectsEnd = panelHeight
		collectionsEnd = panelHeight * 2
	}

	// Right side layout
	commandsHeight := 3

	// Projects panel (top-left)
	if v, err := gui.SetView(g.views.projects, 0, 0, leftWidth-1, projectsEnd-1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = "󰈸 Projects"
		v.TitleColor = g.theme.InactiveBorderColor
		v.BgColor = gocui.ColorDefault
		v.FgColor = gocui.ColorDefault
		v.SelBgColor = g.theme.SelectedLineBgColor
		v.SelFgColor = gocui.ColorDefault
		v.FrameRunes = g.roundedFrameRunes
	}

	if v, err := gui.View(g.views.projects); err == nil {
		// Set footer with count
		if len(g.projects) > 0 {
			v.Footer = fmt.Sprintf("%d of %d", g.selectedProjectIndex+1, len(g.projects))
		} else {
			v.Footer = "0 of 0"
		}
		// Title color matches border (active/inactive based on focus)
		if g.currentColumn == "projects" {
			v.TitleColor = g.theme.ActiveBorderColor
		} else {
			v.TitleColor = g.theme.InactiveBorderColor
		}
		g.updateProjectsView(v)
	}

	// Collections panel (middle-left)
	if v, err := gui.SetView(g.views.collections, 0, projectsEnd, leftWidth-1, collectionsEnd-1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = " Collections"
		v.TitleColor = g.theme.InactiveBorderColor
		v.BgColor = gocui.ColorDefault
		v.FgColor = gocui.ColorDefault
		v.SelBgColor = g.theme.SelectedLineBgColor
		v.SelFgColor = gocui.ColorDefault
		v.FrameRunes = g.roundedFrameRunes
	}

	if v, err := gui.View(g.views.collections); err == nil {
		// Set footer with count
		if len(g.collections) > 0 {
			v.Footer = fmt.Sprintf("%d of %d", g.selectedCollectionIdx+1, len(g.collections))
		} else {
			v.Footer = "0 of 0"
		}
		// Title color matches border (active/inactive based on focus)
		if g.currentColumn == "collections" {
			v.TitleColor = g.theme.ActiveBorderColor
		} else {
			v.TitleColor = g.theme.InactiveBorderColor
		}
		g.updateCollectionsView(v)
	}

	// Tree panel (bottom-left)
	if v, err := gui.SetView(g.views.tree, 0, collectionsEnd, leftWidth-1, maxY-3, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = "󰙅 Tree"
		v.TitleColor = g.theme.InactiveBorderColor
		v.BgColor = gocui.ColorDefault
		v.FgColor = gocui.ColorDefault
		v.SelBgColor = g.theme.SelectedLineBgColor
		v.SelFgColor = gocui.ColorDefault
		v.FrameRunes = g.roundedFrameRunes
	}

	if v, err := gui.View(g.views.tree); err == nil {
		// Set footer with count
		if len(g.treeNodes) > 0 {
			v.Footer = fmt.Sprintf("%d of %d", g.selectedTreeIdx+1, len(g.treeNodes))
		} else {
			v.Footer = "0 of 0"
		}
		// Title color matches border (active/inactive based on focus)
		if g.currentColumn == "tree" {
			v.TitleColor = g.theme.ActiveBorderColor
		} else {
			v.TitleColor = g.theme.InactiveBorderColor
		}
		g.updateTreeView(v)
	}

	// Details panel (top-right, big)
	if v, err := gui.SetView(g.views.details, leftWidth, 0, maxX-1, maxY-commandsHeight-3, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = "Details"
		v.TitleColor = g.theme.InactiveBorderColor
		v.Wrap = true
		v.BgColor = gocui.ColorDefault
		v.FgColor = gocui.ColorDefault
		v.SelBgColor = gocui.ColorDefault
		v.SelFgColor = gocui.ColorDefault
		v.FrameRunes = g.roundedFrameRunes
	}

	if v, err := gui.View(g.views.details); err == nil {
		if g.currentColumn == "details" {
			v.Title = "  Details (j/k scroll) "
			v.TitleColor = g.theme.ActiveBorderColor
		} else {
			v.Title = "  Details "
			v.TitleColor = g.theme.InactiveBorderColor
		}
		g.updateDetailsView(v)
		v.SetOrigin(0, g.detailsScrollPos)
	}

	// Commands panel (bottom-right, single row)
	if v, err := gui.SetView(g.views.commands, leftWidth, maxY-commandsHeight-2, maxX-1, maxY-3, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = "  Commands "
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

	// Enable highlight when this view is focused
	v.Highlight = g.currentColumn == "projects" && len(g.projects) > 0

	for _, project := range g.projects {
		if project.ID == g.currentProject {
			fmt.Fprintf(v, "%s*\033[0m %s\n", g.getActiveColorCode(), project.DisplayName)
		} else {
			fmt.Fprintf(v, "  %s\n", project.DisplayName)
		}
	}

	// Handle scrolling and set cursor for highlight
	if len(g.projects) > 0 {
		v.FocusPoint(0, g.selectedProjectIndex, true)
	}
}

func (g *Gui) updateCollectionsView(v *gocui.View) {
	v.Clear()

	// Enable highlight when this view is focused
	v.Highlight = g.currentColumn == "collections" && len(g.collections) > 0

	if len(g.collections) == 0 {
		// fmt.Fprintln(v, "\033[90m Select a project\033[0m")
		// fmt.Fprintln(v, "\033[90m and press Space\033[0m")
		return
	}

	for _, col := range g.collections {
		if col.Name == g.currentCollection {
			fmt.Fprintf(v, "%s*\033[0m %s\n", g.getActiveColorCode(), col.Name)
		} else {
			fmt.Fprintf(v, "  %s\n", col.Name)
		}
	}

	// Handle scrolling and set cursor for highlight
	if len(g.collections) > 0 {
		v.FocusPoint(0, g.selectedCollectionIdx, true)
	}
}

func (g *Gui) updateTreeView(v *gocui.View) {
	v.Clear()

	// Enable highlight when this view is focused
	v.Highlight = g.currentColumn == "tree" && len(g.treeNodes) > 0

	if len(g.treeNodes) == 0 {
		// fmt.Fprintln(v, "\033[90m Select a collection\033[0m")
		// fmt.Fprintln(v, "\033[90m and press Space\033[0m")
		return
	}

	for _, node := range g.treeNodes {
		// Build indentation
		indent := strings.Repeat("  ", node.Depth)

		// Choose icon based on type and expanded state
		icon := "" // Document icon (nerd font)
		if node.Type == "collection" {
			if node.Expanded {
				icon = "" // Open folder icon
			} else {
				icon = "" // Closed folder icon
			}
		}

		// Tree connector
		connector := ""
		if node.Depth > 0 {
			connector = "└─"
		}

		// Show colored * for currently selected document
		if node.Path == g.currentDocPath {
			fmt.Fprintf(v, "%s*%s%s%s%s %s\n", g.getActiveColorCode(), "\033[0m", indent, connector, icon, node.Name)
		} else {
			fmt.Fprintf(v, " %s%s%s %s\n", indent, connector, icon, node.Name)
		}
	}

	// Handle scrolling and set cursor for highlight
	if len(g.treeNodes) > 0 {
		v.FocusPoint(0, g.selectedTreeIdx, true)
	}
}

func (g *Gui) updateDetailsView(v *gocui.View) {
	v.Clear()

	if g.currentDocData == nil {
		// Firestore fire ASCII art
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
		fmt.Fprintln(v, "\033[36m       L A Z Y F I R E\033[0m")
		fmt.Fprintln(v, "")
		fmt.Fprintln(v, "\033[90m   Select a document to view\033[0m")
		return
	}

	fmt.Fprintf(v, "\033[36m─── %s ───\033[0m\n\n", g.currentDocPath)

	// Pretty print JSON
	data, err := json.MarshalIndent(g.currentDocData, "", "  ")
	if err != nil {
		fmt.Fprintf(v, "Error formatting data: %v\n", err)
		return
	}

	fmt.Fprintln(v, string(data))
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
		statusIcon = "⏳"
		statusColor = "\033[33m" // Yellow
	case "error":
		statusIcon = "✗"
		statusColor = "\033[31m" // Red
	case "success":
		statusIcon = "✓"
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
	fmt.Fprint(v, " \033[36m←/→\033[0m columns  \033[36mj/k\033[0m move  \033[33mspace\033[0m select  \033[33mEsc\033[0m back  \033[35mr\033[0m refresh  \033[31mq\033[0m quit")
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
	if err := g.g.SetKeybinding("", gocui.KeySpace, gocui.ModNone, g.handleEnter); err != nil {
		return err
	}

	// @ to toggle command log modal
	if err := g.g.SetKeybinding("", '@', gocui.ModNone, g.toggleModal); err != nil {
		return err
	}

	return nil
}

// Event handlers
func (g *Gui) quit(gui *gocui.Gui, v *gocui.View) error {
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
	switch g.currentColumn {
	case "projects":
		if g.selectedProjectIndex < len(g.projects)-1 {
			g.selectedProjectIndex++
		}
	case "collections":
		if g.selectedCollectionIdx < len(g.collections)-1 {
			g.selectedCollectionIdx++
		}
	case "tree":
		if g.selectedTreeIdx < len(g.treeNodes)-1 {
			g.selectedTreeIdx++
		}
	case "details":
		g.detailsScrollPos++
	}
	return g.Layout(gui)
}

func (g *Gui) cursorUp(gui *gocui.Gui, v *gocui.View) error {
	switch g.currentColumn {
	case "projects":
		if g.selectedProjectIndex > 0 {
			g.selectedProjectIndex--
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

func (g *Gui) handleEnter(gui *gocui.Gui, v *gocui.View) error {
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

func (g *Gui) selectProject(gui *gocui.Gui) error {
	if g.selectedProjectIndex >= len(g.projects) {
		return nil
	}

	selectedProject := g.projects[g.selectedProjectIndex]

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
	if g.selectedCollectionIdx >= len(g.collections) {
		return nil
	}

	collection := g.collections[g.selectedCollectionIdx]
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
	if g.selectedTreeIdx >= len(g.treeNodes) {
		return nil
	}

	node := &g.treeNodes[g.selectedTreeIdx]
	nodeIdx := g.selectedTreeIdx
	nodePath := node.Path
	nodeName := node.Name
	nodeDepth := node.Depth
	nodeType := node.Type

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
	// Close modal if open
	if g.modalOpen {
		g.modalOpen = false
		return nil
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

	// Otherwise go back a column
	return g.columnLeft(gui, v)
}

func (g *Gui) toggleModal(gui *gocui.Gui, v *gocui.View) error {
	g.modalOpen = !g.modalOpen
	return g.Layout(gui)
}

func (g *Gui) refresh(gui *gocui.Gui, v *gocui.View) error {
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
