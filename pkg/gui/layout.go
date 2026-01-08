package gui

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jesseduffield/gocui"
	"github.com/marjoballabani/lazyfire/pkg/gui/icons"
)

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
					switch cmd.Status {
					case "error":
						statusColor = "\033[31m" // Red
					case "running":
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

	// Show loading indicator when projects are being loaded
	if g.isLoading && len(g.projects) == 0 {
		v.Highlight = false
		fmt.Fprint(v, g.getLoadingText("Loading projects..."))
		return
	}

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

	// Show loading indicator when collections are being loaded
	if g.collectionsLoading {
		v.Highlight = false
		fmt.Fprint(v, g.getLoadingText("Loading collections..."))
		return
	}

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

	// Show loading indicator when tree is being loaded
	if g.treeLoading {
		v.Highlight = false
		fmt.Fprint(v, g.getLoadingText("Loading documents..."))
		return
	}

	filtered := g.getFilteredTreeNodes()

	// Enable highlight when this view is focused
	v.Highlight = g.currentColumn == "tree" && len(filtered) > 0

	if len(filtered) == 0 {
		return
	}

	for i, node := range filtered {
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

		// Determine marker: * for current doc, + for selected in select mode, space otherwise
		marker := " "
		isSelected := g.selectMode && g.selectedDocs[i]
		if isSelected {
			marker = "\033[30;43m+\033[0m" // Black on yellow background for selected
		} else if node.Path == g.currentDocPath {
			marker = g.getActiveColorCode() + "*" + "\033[0m"
		}

		// Highlight selected items in select mode
		if isSelected {
			fmt.Fprintf(v, "%s%s%s%s\033[33m%s\033[0m\n", marker, indent, connector, icon, node.Name)
		} else {
			fmt.Fprintf(v, "%s%s%s%s%s\n", marker, indent, connector, icon, node.Name)
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
	// Show loading indicator when details are being loaded
	if g.detailsLoading {
		v.Clear()
		fmt.Fprint(v, g.getLoadingText("Loading document..."))
		return
	}

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
			// Only call SetContent if view is dirty (avoids expensive redraw)
			if g.detailsViewDirty {
				v.SetContent(g.cachedDetailsContent)
				g.detailsViewDirty = false
			}
			return
		}

		// New document - reset scroll position
		g.detailsScrollPos = 0

		// Format JSON
		data, err := json.MarshalIndent(g.currentDocData, "", "  ")
		if err != nil {
			v.SetContent(fmt.Sprintf("Error formatting data: %v\n", err))
			return
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("\033[36m─── %s ───\033[0m\n", g.currentDocPath))

		// Show stats for actual documents
		if strings.Contains(g.currentDocPath, "/") {
			stats := calculateDocStats(g.currentDocData, g.currentDocPath)
			content.WriteString(formatDocStats(stats))
			content.WriteString("\n")
		}
		content.WriteString("\n")

		// Syntax highlighting with chroma
		content.WriteString(colorizeJSON(string(data)))

		g.cachedDetailsLines = strings.Split(string(data), "\n")
		g.cachedDetailsHeader = ""
		g.cachedDetailsContent = content.String()
		g.cachedDetailsDocPath = g.currentDocPath
		v.SetContent(g.cachedDetailsContent)
		g.detailsViewDirty = false
		return
	}

	// Clear cache when not showing document
	g.clearDetailsCache()

	v.Clear()

	// Show fetched project details if available
	if g.currentProjectInfo != nil {
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
		// Show text with cursor at correct position
		beforeCursor := g.filterInputText[:g.filterCursorPos]
		afterCursor := g.filterInputText[g.filterCursorPos:]
		// Cursor shown as reverse video - highlight char at cursor or space if at end
		var cursorChar, rest string
		if len(afterCursor) > 0 {
			cursorChar = string(afterCursor[0])
			rest = afterCursor[1:]
		} else {
			cursorChar = " "
			rest = ""
		}
		filterPrompt := fmt.Sprintf(" \033[33mFilter %s:\033[0m %s\033[7m%s\033[0m%s", panelName, beforeCursor, cursorChar, rest)
		hints := "  \033[90m(Enter to select, Esc to cancel)\033[0m"
		fmt.Fprintf(v, "%s%s", filterPrompt, hints)
		return
	}

	// Show select mode status
	if g.selectMode {
		count := len(g.selectedDocs)
		fmt.Fprintf(v, " \033[33m-- SELECT MODE --\033[0m  %d selected  \033[90m(j/k to extend, Space to fetch, Esc to cancel)\033[0m", count)
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

// Firestore limits (https://firebase.google.com/docs/firestore/quotas)
const (
	maxDocSizeBytes    = 1048576         // 1 MiB
	maxFieldCount      = 20000           // Due to 40k index entries limit (2 per field)
	maxDepth           = 20              // Maximum depth of nested maps/arrays
	maxFieldNameBytes  = 1500            // Maximum field name size
	maxFieldValueBytes = 1048576 - 89    // 1 MiB - 89 bytes
	maxDocNameBytes    = 6 * 1024        // 6 KiB for document path
)

// docStats holds document statistics
type docStats struct {
	sizeBytes       int
	fieldCount      int
	maxDepth        int
	maxFieldName    int // longest field name in bytes
	maxFieldValue   int // largest field value in bytes
	docPathLen      int // document path length
}

// calculateDocStats calculates all document statistics
func calculateDocStats(data map[string]any, docPath string) docStats {
	jsonBytes, _ := json.Marshal(data)
	maxName, maxValue := findMaxFieldSizes(data)
	return docStats{
		sizeBytes:     len(jsonBytes),
		fieldCount:    countFields(data),
		maxDepth:      calculateDepth(data),
		maxFieldName:  maxName,
		maxFieldValue: maxValue,
		docPathLen:    len(docPath),
	}
}

// findMaxFieldSizes finds the largest field name and value sizes
func findMaxFieldSizes(data any) (maxName int, maxValue int) {
	switch v := data.(type) {
	case map[string]any:
		for key, val := range v {
			nameLen := len(key)
			if nameLen > maxName {
				maxName = nameLen
			}
			// Calculate value size
			valBytes, _ := json.Marshal(val)
			if len(valBytes) > maxValue {
				maxValue = len(valBytes)
			}
			// Recurse into nested structures
			nestedName, nestedValue := findMaxFieldSizes(val)
			if nestedName > maxName {
				maxName = nestedName
			}
			if nestedValue > maxValue {
				maxValue = nestedValue
			}
		}
	case []any:
		for _, item := range v {
			nestedName, nestedValue := findMaxFieldSizes(item)
			if nestedName > maxName {
				maxName = nestedName
			}
			if nestedValue > maxValue {
				maxValue = nestedValue
			}
		}
	}
	return
}

// countFields counts all fields including nested ones
func countFields(data any) int {
	switch v := data.(type) {
	case map[string]any:
		count := len(v)
		for _, val := range v {
			count += countFields(val)
		}
		return count
	case []any:
		count := 0
		for _, item := range v {
			count += countFields(item)
		}
		return count
	default:
		return 0
	}
}

// calculateDepth calculates the maximum nesting depth
func calculateDepth(data any) int {
	switch v := data.(type) {
	case map[string]any:
		maxChildDepth := 0
		for _, val := range v {
			d := calculateDepth(val)
			if d > maxChildDepth {
				maxChildDepth = d
			}
		}
		return 1 + maxChildDepth
	case []any:
		maxChildDepth := 0
		for _, item := range v {
			d := calculateDepth(item)
			if d > maxChildDepth {
				maxChildDepth = d
			}
		}
		return 1 + maxChildDepth
	default:
		return 0
	}
}

// formatDocStats returns a formatted string showing document stats with warnings
func formatDocStats(stats docStats) string {
	// Helper to get color based on percentage of limit
	// Tiers: green <50%, cyan 50-70%, yellow 70-85%, orange 85-100%, red >100%
	getColor := func(value, limit int) string {
		pct := value * 100 / limit
		if pct > 100 {
			return "\033[31m" // red - over limit
		} else if pct > 85 {
			return "\033[38;5;208m" // orange - critical
		} else if pct > 70 {
			return "\033[33m" // yellow - warning
		} else if pct > 50 {
			return "\033[36m" // cyan - moderate
		}
		return "\033[32m" // green - ok
	}

	// Line 1: Size, Fields, Depth
	line1 := fmt.Sprintf("\033[90mSize:\033[0m %s%s / 1MB\033[0m  \033[90mFields:\033[0m %s%d / %d\033[0m  \033[90mDepth:\033[0m %s%d / %d\033[0m",
		getColor(stats.sizeBytes, maxDocSizeBytes), formatBytes(stats.sizeBytes),
		getColor(stats.fieldCount, maxFieldCount), stats.fieldCount, maxFieldCount,
		getColor(stats.maxDepth, maxDepth), stats.maxDepth, maxDepth)

	// Line 2: Field Name, Field Value, Doc Path
	line2 := fmt.Sprintf("\033[90mField Name:\033[0m %s%d / %d B\033[0m  \033[90mField Value:\033[0m %s%s / 1MB\033[0m  \033[90mPath:\033[0m %s%d / %d B\033[0m",
		getColor(stats.maxFieldName, maxFieldNameBytes), stats.maxFieldName, maxFieldNameBytes,
		getColor(stats.maxFieldValue, maxFieldValueBytes), formatBytes(stats.maxFieldValue),
		getColor(stats.docPathLen, maxDocNameBytes), stats.docPathLen, maxDocNameBytes)

	return line1 + "\n" + line2
}

// formatBytes formats bytes into human readable string
func formatBytes(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
}
