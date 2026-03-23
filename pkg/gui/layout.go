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
		v.Title = " " + icons.FIREBASE_ICON + " Projects "
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
			v.Title = " " + icons.FIREBASE_ICON + " Projects "
		} else if isFocused {
			gui.SelFrameColor = g.theme.ActiveBorderColor
			gui.SelFgColor = g.theme.ActiveBorderColor
			v.TitleColor = g.theme.ActiveBorderColor
			v.FrameColor = g.theme.ActiveBorderColor
			v.Title = " " + icons.FIREBASE_ICON + " Projects "
		} else {
			v.TitleColor = g.theme.InactiveBorderColor
			v.FrameColor = g.theme.InactiveBorderColor
			v.Title = " " + icons.FIREBASE_ICON + " Projects "
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

	// Collections/Functions panel (middle-left)
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
		// Active tab always uses ActiveBorderColor (reddish) regardless of focus
		v.SelFgColor = g.theme.ActiveBorderColor
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

		// Sliding window of 3 tabs using gocui's built-in tab rendering
		allTabs := []string{"Collections", "Functions", "Storage", "Auth", "Rules", "Indexes"}
		allKeys := []string{"collections", "functions", "storage", "auth", "rules", "indexes"}
		activeIdx := 0
		for i, k := range allKeys {
			if k == g.collectionsTab {
				activeIdx = i
				break
			}
		}
		start := activeIdx - 1
		if start < 0 {
			start = 0
		}
		if start+3 > len(allTabs) {
			start = len(allTabs) - 3
		}
		windowTabs := make([]string, 3)
		for i := 0; i < 3; i++ {
			name := allTabs[start+i]
			if start > 0 && i == 0 {
				name = "< " + name
			}
			if start+3 < len(allTabs) && i == 2 {
				name = name + " >"
			}
			windowTabs[i] = name
		}
		v.Tabs = windowTabs
		v.TabIndex = activeIdx - start

		switch g.collectionsTab {
		case "functions":
			filtered := g.getFilteredFunctions()
			hasFilter := hasCommittedFilter || isTypingFilter
			if hasFilter {
				v.Footer = fmt.Sprintf("%d/%d matched", len(filtered), len(g.functions))
			} else if len(g.functions) > 0 {
				v.Footer = fmt.Sprintf("%d of %d", g.selectedFunctionIdx+1, len(g.functions))
			} else {
				v.Footer = "0 of 0"
			}
		case "storage":
			if g.currentBucket == "" {
				if len(g.storageBuckets) > 0 {
					v.Footer = fmt.Sprintf("%d of %d", g.selectedBucketIdx+1, len(g.storageBuckets))
				} else {
					v.Footer = "0 of 0"
				}
			} else {
				if len(g.storageObjects) > 0 {
					v.Footer = fmt.Sprintf("%d of %d", g.selectedObjectIdx+1, len(g.storageObjects))
				} else {
					v.Footer = "0 of 0"
				}
			}
		case "auth":
			if len(g.authUsers) > 0 {
				v.Footer = fmt.Sprintf("%d of %d", g.selectedAuthIdx+1, len(g.authUsers))
			} else {
				v.Footer = "0 of 0"
			}
		case "rules":
			if g.firestoreRules != nil {
				v.Footer = "loaded"
			} else {
				v.Footer = ""
			}
		case "indexes":
			if len(g.firestoreIndexes) > 0 {
				v.Footer = fmt.Sprintf("%d indexes", len(g.firestoreIndexes))
			} else {
				v.Footer = "0 indexes"
			}
		default: // collections
			filtered := g.getFilteredCollections()
			hasFilter := hasCommittedFilter || isTypingFilter
			if hasFilter {
				v.Footer = fmt.Sprintf("%d/%d matched", len(filtered), len(g.collections))
			} else if len(g.collections) > 0 {
				v.Footer = fmt.Sprintf("%d of %d", g.selectedCollectionIdx+1, len(g.collections))
			} else {
				v.Footer = "0 of 0"
			}
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
		// Show query mode in title
		if g.queryResultMode {
			v.Title = " " + icons.TREE_ICON + " Query Results (Q to clear) "
		} else {
			v.Title = " " + icons.TREE_ICON + " Tree "
		}
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

		// Title/border color
		// Active tab always uses ActiveBorderColor regardless of focus
		v.SelFgColor = g.theme.ActiveBorderColor
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

		// Show active tab in title when Functions tab is active
		if g.collectionsTab == "functions" && (g.currentColumn == "collections" || (g.currentColumn == "details" && g.previousColumn == "collections")) {
			v.Tabs = nil
			if g.detailsTab == "logs" {
				v.Title = " Logs [/] "
			} else {
				v.Title = " Details [/] "
			}
		} else {
			v.Tabs = nil
			// Only reset detailsTab when on Tree or Projects (not when switching to Collections tab)
			if g.currentColumn == "tree" || g.currentColumn == "projects" {
				g.detailsTab = "details"
			}
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

	// Query builder modal
	// Confirm dialog
	if g.confirmOpen {
		modalWidth := 50
		modalHeight := 10
		modalX := (maxX - modalWidth) / 2
		modalY := (maxY - modalHeight) / 2

		if v, err := gui.SetView(g.views.confirm, modalX, modalY, modalX+modalWidth, modalY+modalHeight, 0); err != nil {
			if !errors.Is(err, gocui.ErrUnknownView) {
				return err
			}
			v.Title = " " + g.confirmTitle + " "
			v.TitleColor = g.theme.ActiveBorderColor
			v.FrameColor = g.theme.ActiveBorderColor
			v.FrameRunes = g.roundedFrameRunes
			v.BgColor = gocui.ColorDefault
			v.FgColor = gocui.ColorDefault
		}

		if v, err := gui.View(g.views.confirm); err == nil {
			v.Clear()
			v.Title = " " + g.confirmTitle + " "
			fmt.Fprintln(v, "")
			for _, line := range strings.Split(g.confirmMessage, "\n") {
				fmt.Fprintf(v, "  %s\n", line)
			}
			fmt.Fprintln(v, "")
			fmt.Fprintln(v, "  \033[32mEnter\033[0m confirm    \033[31mEsc\033[0m cancel")
			if _, err := gui.SetCurrentView(g.views.confirm); err != nil {
				return fmt.Errorf("failed to set confirm view: %w", err)
			}
		}

		return nil
	}
	_ = gui.DeleteView(g.views.confirm)

	if g.queryModalOpen {
		modalWidth := 50
		modalHeight := 20
		if modalHeight > maxY-4 {
			modalHeight = maxY - 4
		}
		modalX := (maxX - modalWidth) / 2
		modalY := (maxY - modalHeight) / 2

		if v, err := gui.SetView(g.views.queryModal, modalX, modalY, modalX+modalWidth, modalY+modalHeight, 0); err != nil {
			if !errors.Is(err, gocui.ErrUnknownView) {
				return err
			}
			v.Title = " Query Builder "
			v.TitleColor = g.theme.ActiveBorderColor
			v.FrameColor = g.theme.ActiveBorderColor
			v.FrameRunes = g.roundedFrameRunes
			v.BgColor = gocui.ColorDefault
			v.FgColor = gocui.ColorDefault
		}

		if v, err := gui.View(g.views.queryModal); err == nil {
			g.renderQueryModal(v)
		}

		// Create editable input view when in edit mode
		if g.queryEditMode {
			inputX := modalX + 2
			inputY := modalY + modalHeight - 4
			inputWidth := modalWidth - 4

			if v, err := gui.SetView(g.views.queryInput, inputX, inputY, inputX+inputWidth, inputY+2, 0); err != nil {
				if !errors.Is(err, gocui.ErrUnknownView) {
					return err
				}
				v.Title = " " + g.getQueryEditFieldName() + " "
				v.TitleColor = g.theme.ActiveBorderColor
				v.FrameColor = g.theme.ActiveBorderColor
				v.FrameRunes = g.roundedFrameRunes
				v.Editable = true
				v.Editor = gocui.EditorFunc(g.queryInputEditor)
				// Initialize TextArea with current value
				v.TextArea.Clear()
				v.TextArea.TypeString(g.queryEditBuffer)
				v.RenderTextArea()
			}

			if v, err := gui.View(g.views.queryInput); err == nil {
				v.Title = " " + g.getQueryEditFieldName() + " "
				gui.Cursor = true // Show cursor when editing
				if _, err := gui.SetCurrentView(g.views.queryInput); err != nil {
					return fmt.Errorf("failed to set query input view: %w", err)
				}
			}
		} else {
			gui.Cursor = false // Hide cursor when not editing
			_ = gui.DeleteView(g.views.queryInput)
		}

		// Create select popup when selecting operator/type
		if g.querySelectOpen {
			selectWidth := 20
			selectHeight := len(g.querySelectItems) + 2
			if selectHeight > 12 {
				selectHeight = 12
			}
			selectX := modalX + (modalWidth-selectWidth)/2
			selectY := modalY + 4

			if v, err := gui.SetView(g.views.querySelect, selectX, selectY, selectX+selectWidth, selectY+selectHeight, 0); err != nil {
				if !errors.Is(err, gocui.ErrUnknownView) {
					return err
				}
				v.Title = " Select "
				v.TitleColor = g.theme.ActiveBorderColor
				v.FrameColor = g.theme.ActiveBorderColor
				v.FrameRunes = g.roundedFrameRunes
				v.Highlight = true
				v.SelBgColor = g.theme.SelectedLineBgColor
				v.SelFgColor = gocui.ColorDefault
			}

			if v, err := gui.View(g.views.querySelect); err == nil {
				g.renderQuerySelect(v)
				if _, err := gui.SetCurrentView(g.views.querySelect); err != nil {
					return fmt.Errorf("failed to set query select view: %w", err)
				}
			}
		} else {
			_ = gui.DeleteView(g.views.querySelect)
			if !g.queryEditMode {
				if _, err := gui.SetCurrentView(g.views.queryModal); err != nil {
					return fmt.Errorf("failed to set query view: %w", err)
				}
			}
		}

		return nil
	} else {
		_ = gui.DeleteView(g.views.queryModal)
		_ = gui.DeleteView(g.views.queryInput)
		_ = gui.DeleteView(g.views.querySelect)
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
		_ = gui.DeleteView(g.views.helpModal)
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
		_ = gui.DeleteView(g.views.modal)
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

	// Project icon with spacing and orange color
	icon := icons.FIREBASE_ICON
	if icon != "" {
		icon = "\033[38;5;208m" + icon + "\033[0m " // Orange Firebase icon
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

	// Show content based on active tab
	switch g.collectionsTab {
	case "functions":
		g.renderFunctionsContent(v)
	case "storage":
		g.renderStorageContent(v)
	case "auth":
		g.renderAuthContent(v)
	case "rules":
		g.renderRulesContent(v)
	case "indexes":
		g.renderIndexesContent(v)
	default:
		g.renderCollectionsContent(v)
	}
}

func (g *Gui) renderCollectionsContent(v *gocui.View) {
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
			icon = "\033[36m" + icon + "\033[0m " // Cyan folder icon
		}
		// Show cached doc count if available
		countStr := ""
		if cachedPaths, ok := g.collectionCache[col.Path]; ok {
			countStr = fmt.Sprintf(" \033[90m(%d)\033[0m", len(cachedPaths))
		}
		if col.Name == g.currentCollection {
			fmt.Fprintf(v, "%s*\033[0m %s%s%s\n", g.getActiveColorCode(), icon, col.Name, countStr)
		} else {
			fmt.Fprintf(v, "  %s%s%s\n", icon, col.Name, countStr)
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

func (g *Gui) renderFunctionsContent(v *gocui.View) {
	// Show loading indicator when functions are being loaded
	if g.functionsLoading {
		v.Highlight = false
		fmt.Fprint(v, g.getLoadingText("Loading functions..."))
		return
	}

	filtered := g.getFilteredFunctions()

	// Enable highlight when this view is focused
	v.Highlight = g.currentColumn == "collections" && len(filtered) > 0

	if len(filtered) == 0 {
		fmt.Fprint(v, "\033[90mNo functions deployed\033[0m")
		return
	}

	activeColor := g.getActiveColorCode()
	resetColor := "\033[0m"
	greenColor := "\033[32m"
	yellowColor := "\033[33m"
	redColor := "\033[31m"

	for _, fn := range filtered {
		// Status color
		statusColor := greenColor
		switch fn.Status {
		case "DEPLOYING", "DELETE_IN_PROGRESS":
			statusColor = yellowColor
		case "OFFLINE", "UNKNOWN":
			statusColor = redColor
		}

		// Function icon (lightning bolt)
		icon := "⚡"

		// Mark current function
		marker := "  "
		if g.currentFunction != nil && fn.DisplayName == g.currentFunction.DisplayName {
			marker = activeColor + "* " + resetColor
		}

		// Format: ⚡ functionName (region) [STATUS]
		fmt.Fprintf(v, "%s%s %s %s(%s)%s %s[%s]%s\n",
			marker, icon, fn.DisplayName,
			"\033[90m", fn.Region, resetColor,
			statusColor, fn.Status, resetColor)
	}

	// Handle scrolling and set cursor for highlight
	if len(filtered) > 0 {
		// Clamp selection to filtered list
		if g.selectedFunctionIdx >= len(filtered) {
			g.selectedFunctionIdx = len(filtered) - 1
		}
		v.FocusPoint(0, g.selectedFunctionIdx, true)
	}
}

func (g *Gui) renderStorageContent(v *gocui.View) {
	if g.storageLoading {
		v.Highlight = false
		fmt.Fprint(v, g.getLoadingText("Loading storage..."))
		return
	}

	isFocused := g.currentColumn == "collections"
	dimColor := "\033[90m"
	resetColor := "\033[0m"
	activeColor := g.getActiveColorCode()

	if g.currentBucket == "" {
		// Show bucket list
		v.Highlight = isFocused && len(g.storageBuckets) > 0
		if len(g.storageBuckets) == 0 {
			fmt.Fprintf(v, "%sNo storage buckets found%s", dimColor, resetColor)
			return
		}
		for i, b := range g.storageBuckets {
			marker := "  "
			if i == g.selectedBucketIdx {
				marker = activeColor + "* " + resetColor
			}
			fmt.Fprintf(v, "%s📦 %s %s(%s, %s)%s\n", marker, b.Name, dimColor, b.Location, b.StorageClass, resetColor)
		}
		if len(g.storageBuckets) > 0 {
			if g.selectedBucketIdx >= len(g.storageBuckets) {
				g.selectedBucketIdx = len(g.storageBuckets) - 1
			}
			v.FocusPoint(0, g.selectedBucketIdx, true)
		}
	} else {
		// Show objects in current bucket/prefix
		v.Highlight = isFocused && len(g.storageObjects) > 0
		if len(g.storageObjects) == 0 {
			fmt.Fprintf(v, "%sEmpty%s", dimColor, resetColor)
			return
		}
		for i, o := range g.storageObjects {
			marker := "  "
			if i == g.selectedObjectIdx {
				marker = activeColor + "* " + resetColor
			}
			if o.IsPrefix {
				fmt.Fprintf(v, "%s📁 %s/\n", marker, o.DisplayName)
			} else {
				sizeStr := formatBytes(int(o.Size))
				fmt.Fprintf(v, "%s📄 %s %s(%s, %s)%s\n", marker, o.DisplayName, dimColor, sizeStr, o.ContentType, resetColor)
			}
		}
		if len(g.storageObjects) > 0 {
			if g.selectedObjectIdx >= len(g.storageObjects) {
				g.selectedObjectIdx = len(g.storageObjects) - 1
			}
			v.FocusPoint(0, g.selectedObjectIdx, true)
		}
	}
}

func (g *Gui) renderAuthContent(v *gocui.View) {
	if g.authLoading {
		v.Highlight = false
		fmt.Fprint(v, g.getLoadingText("Loading users..."))
		return
	}

	isFocused := g.currentColumn == "collections"
	dimColor := "\033[90m"
	resetColor := "\033[0m"
	activeColor := g.getActiveColorCode()

	v.Highlight = isFocused && len(g.authUsers) > 0
	if len(g.authUsers) == 0 {
		fmt.Fprintf(v, "%sNo users found%s", dimColor, resetColor)
		return
	}

	for i, u := range g.authUsers {
		marker := "  "
		if i == g.selectedAuthIdx {
			marker = activeColor + "* " + resetColor
		}
		name := u.Email
		if name == "" {
			name = u.UID
		}
		status := ""
		if u.Disabled {
			status = " \033[31m[disabled]\033[0m"
		}
		providers := ""
		if len(u.Providers) > 0 {
			providers = fmt.Sprintf(" %s(%s)%s", dimColor, strings.Join(u.Providers, ", "), resetColor)
		}
		fmt.Fprintf(v, "%s👤 %s%s%s\n", marker, name, status, providers)
	}

	if len(g.authUsers) > 0 {
		if g.selectedAuthIdx >= len(g.authUsers) {
			g.selectedAuthIdx = len(g.authUsers) - 1
		}
		v.FocusPoint(0, g.selectedAuthIdx, true)
	}
}

func (g *Gui) renderRulesContent(v *gocui.View) {
	if g.rulesLoading {
		v.Highlight = false
		fmt.Fprint(v, g.getLoadingText("Loading rules..."))
		return
	}

	v.Highlight = false
	dimColor := "\033[90m"
	resetColor := "\033[0m"
	cyanColor := "\033[36m"

	if g.firestoreRules == nil {
		fmt.Fprintf(v, "%sNo rules loaded%s", dimColor, resetColor)
		return
	}

	if g.firestoreRules.UpdatedAt != "" {
		fmt.Fprintf(v, "%sLast deployed: %s%s\n\n", dimColor, g.firestoreRules.UpdatedAt, resetColor)
	}

	// Syntax highlight the rules
	for _, line := range strings.Split(g.firestoreRules.Rules, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			fmt.Fprintf(v, "%s%s%s\n", dimColor, line, resetColor)
		} else if strings.HasPrefix(trimmed, "rules_version") || strings.HasPrefix(trimmed, "service") {
			fmt.Fprintf(v, "%s%s%s\n", cyanColor, line, resetColor)
		} else if strings.Contains(trimmed, "allow") {
			fmt.Fprintf(v, "\033[33m%s%s\n", line, resetColor)
		} else if strings.Contains(trimmed, "match") {
			fmt.Fprintf(v, "\033[32m%s%s\n", line, resetColor)
		} else {
			fmt.Fprintf(v, "%s\n", line)
		}
	}
}

func (g *Gui) renderIndexesContent(v *gocui.View) {
	if g.indexesLoading {
		v.Highlight = false
		fmt.Fprint(v, g.getLoadingText("Loading indexes..."))
		return
	}

	dimColor := "\033[90m"
	resetColor := "\033[0m"
	cyanColor := "\033[36m"
	greenColor := "\033[32m"
	yellowColor := "\033[33m"

	v.Highlight = false
	if len(g.firestoreIndexes) == 0 {
		fmt.Fprintf(v, "%sNo composite indexes found%s", dimColor, resetColor)
		return
	}

	for i, idx := range g.firestoreIndexes {
		// State color
		stateColor := greenColor
		if idx.State == "CREATING" {
			stateColor = yellowColor
		} else if idx.State == "NEEDS_REPAIR" {
			stateColor = "\033[31m"
		}

		// Collection group header
		fmt.Fprintf(v, "%s%d.%s %s%s%s %s[%s]%s %s(%s)%s\n",
			cyanColor, i+1, resetColor,
			"", idx.CollectionGroup, "",
			stateColor, idx.State, resetColor,
			dimColor, idx.QueryScope, resetColor)

		// Fields
		for _, f := range idx.Fields {
			order := f.Order
			if order == "" {
				order = "AUTO"
			}
			fmt.Fprintf(v, "   %s%s%s %s\n", dimColor, f.FieldPath, resetColor, order)
		}
	}
}

func (g *Gui) renderFunctionDetails(v *gocui.View) {
	fn := g.currentFunction
	if fn == nil {
		return
	}

	resetColor := "\033[0m"
	cyanColor := "\033[36m"
	dimColor := "\033[90m"
	greenColor := "\033[32m"
	yellowColor := "\033[33m"

	fmt.Fprintf(v, "%s─── Function Details ───%s\n\n", cyanColor, resetColor)

	// Name
	fmt.Fprintf(v, " %sName:%s        %s\n", dimColor, resetColor, fn.DisplayName)

	// Status with color
	statusColor := greenColor
	if fn.Status == "DEPLOYING" || fn.Status == "DELETE_IN_PROGRESS" {
		statusColor = yellowColor
	} else if fn.Status == "OFFLINE" || fn.Status == "UNKNOWN" {
		statusColor = "\033[31m" // Red
	}
	fmt.Fprintf(v, " %sStatus:%s      %s%s%s\n", dimColor, resetColor, statusColor, fn.Status, resetColor)

	// Runtime
	fmt.Fprintf(v, " %sRuntime:%s     %s\n", dimColor, resetColor, fn.Runtime)

	// Region
	fmt.Fprintf(v, " %sRegion:%s      %s\n", dimColor, resetColor, fn.Region)

	// Memory
	if fn.Memory != "" {
		fmt.Fprintf(v, " %sMemory:%s      %s\n", dimColor, resetColor, fn.Memory)
	}

	// Timeout
	if fn.Timeout != "" {
		fmt.Fprintf(v, " %sTimeout:%s     %s\n", dimColor, resetColor, fn.Timeout)
	}

	// Trigger
	fmt.Fprintf(v, " %sTrigger:%s     %s\n", dimColor, resetColor, fn.TriggerType)

	// URL for HTTP triggers
	if fn.TriggerURL != "" {
		fmt.Fprintf(v, " %sURL:%s         %s\n", dimColor, resetColor, fn.TriggerURL)
	}

	// Entry point
	if fn.EntryPoint != "" {
		fmt.Fprintf(v, " %sEntry:%s       %s\n", dimColor, resetColor, fn.EntryPoint)
	}

	// Last updated
	if fn.UpdatedAt != "" {
		fmt.Fprintf(v, " %sUpdated:%s     %s\n", dimColor, resetColor, fn.UpdatedAt)
	}
}

func (g *Gui) renderFunctionLogs(v *gocui.View) {
	v.Clear()

	resetColor := "\033[0m"
	cyanColor := "\033[36m"
	dimColor := "\033[90m"
	greenColor := "\033[32m"
	yellowColor := "\033[33m"
	redColor := "\033[31m"

	if g.currentFunction == nil {
		fmt.Fprintf(v, "%s─── Logs ───%s\n\n", cyanColor, resetColor)
		fmt.Fprintf(v, "%sNo function selected%s\n\n", dimColor, resetColor)
		fmt.Fprintf(v, "%sSelect a function from the Functions tab%s\n", dimColor, resetColor)
		fmt.Fprintf(v, "%s(press '[' or ']' to switch tabs)%s\n", dimColor, resetColor)
		return
	}

	// Header
	fmt.Fprintf(v, "%s─── Logs: %s ───%s\n", cyanColor, g.currentFunction.DisplayName, resetColor)
	fmt.Fprintf(v, "%s(press 'r' to refresh)%s\n\n", dimColor, resetColor)

	// Show loading indicator
	if g.logsLoading && len(g.functionLogs) == 0 {
		fmt.Fprint(v, g.getLoadingText("Loading logs..."))
		return
	}

	if len(g.functionLogs) == 0 {
		fmt.Fprintf(v, "%sNo logs available%s\n", dimColor, resetColor)
		return
	}

	// Show active log level filter
	if g.logLevelFilter != "" {
		fmt.Fprintf(v, "%sFilter: %s%s\n\n", yellowColor, g.logLevelFilter, resetColor)
	}

	// Render logs (newest first based on API response)
	for _, log := range g.functionLogs {
		// Apply log level filter
		if g.logLevelFilter != "" && log.Severity != g.logLevelFilter {
			continue
		}
		// Severity color
		severityColor := dimColor
		switch log.Severity {
		case "INFO":
			severityColor = greenColor
		case "WARNING":
			severityColor = yellowColor
		case "ERROR":
			severityColor = redColor
		case "DEBUG":
			severityColor = dimColor
		}

		// Format: timestamp SEVERITY message
		fmt.Fprintf(v, "%s%s%s %s%-7s%s %s\n",
			dimColor, log.Timestamp, resetColor,
			severityColor, log.Severity, resetColor,
			log.Message)
	}
}

func (g *Gui) renderStorageDetails(v *gocui.View) {
	dimColor := "\033[90m"
	resetColor := "\033[0m"
	cyanColor := "\033[36m"

	if g.currentBucket == "" {
		// Show bucket details
		if len(g.storageBuckets) == 0 || g.selectedBucketIdx >= len(g.storageBuckets) {
			fmt.Fprintf(v, "%sSelect a bucket to view details%s", dimColor, resetColor)
			return
		}
		b := g.storageBuckets[g.selectedBucketIdx]
		fmt.Fprintf(v, "%s─── Bucket Details ───%s\n\n", cyanColor, resetColor)
		fmt.Fprintf(v, " %sName:%s         %s\n", dimColor, resetColor, b.Name)
		fmt.Fprintf(v, " %sLocation:%s     %s\n", dimColor, resetColor, b.Location)
		fmt.Fprintf(v, " %sClass:%s        %s\n", dimColor, resetColor, b.StorageClass)
		fmt.Fprintf(v, " %sCreated:%s      %s\n", dimColor, resetColor, b.TimeCreated)
	} else {
		// Show object details
		if len(g.storageObjects) == 0 || g.selectedObjectIdx >= len(g.storageObjects) {
			fmt.Fprintf(v, "%sNo object selected%s", dimColor, resetColor)
			return
		}
		o := g.storageObjects[g.selectedObjectIdx]
		fmt.Fprintf(v, "%s─── Object Details ───%s\n\n", cyanColor, resetColor)
		fmt.Fprintf(v, " %sName:%s         %s\n", dimColor, resetColor, o.Name)
		if o.IsPrefix {
			fmt.Fprintf(v, " %sType:%s         folder\n", dimColor, resetColor)
		} else {
			fmt.Fprintf(v, " %sSize:%s         %s\n", dimColor, resetColor, formatBytes(int(o.Size)))
			fmt.Fprintf(v, " %sType:%s         %s\n", dimColor, resetColor, o.ContentType)
			fmt.Fprintf(v, " %sCreated:%s      %s\n", dimColor, resetColor, o.TimeCreated)
			fmt.Fprintf(v, " %sUpdated:%s      %s\n", dimColor, resetColor, o.Updated)
		}
		fmt.Fprintf(v, "\n %sBucket:%s       %s\n", dimColor, resetColor, g.currentBucket)
		if g.storagePrefix != "" {
			fmt.Fprintf(v, " %sPrefix:%s       %s\n", dimColor, resetColor, g.storagePrefix)
		}
	}
}

func (g *Gui) renderAuthDetails(v *gocui.View) {
	dimColor := "\033[90m"
	resetColor := "\033[0m"
	cyanColor := "\033[36m"

	if len(g.authUsers) == 0 || g.selectedAuthIdx >= len(g.authUsers) {
		fmt.Fprintf(v, "%sSelect a user to view details%s", dimColor, resetColor)
		return
	}

	u := g.authUsers[g.selectedAuthIdx]
	fmt.Fprintf(v, "%s─── User Details ───%s\n\n", cyanColor, resetColor)
	fmt.Fprintf(v, " %sUID:%s           %s\n", dimColor, resetColor, u.UID)
	if u.Email != "" {
		fmt.Fprintf(v, " %sEmail:%s         %s\n", dimColor, resetColor, u.Email)
	}
	if u.DisplayName != "" {
		fmt.Fprintf(v, " %sName:%s          %s\n", dimColor, resetColor, u.DisplayName)
	}
	fmt.Fprintf(v, " %sVerified:%s      %v\n", dimColor, resetColor, u.EmailVerified)
	fmt.Fprintf(v, " %sDisabled:%s      %v\n", dimColor, resetColor, u.Disabled)
	if u.CreatedAt != "" {
		fmt.Fprintf(v, " %sCreated:%s       %s\n", dimColor, resetColor, u.CreatedAt)
	}
	if u.LastSignIn != "" {
		fmt.Fprintf(v, " %sLast Sign-In:%s  %s\n", dimColor, resetColor, u.LastSignIn)
	}
	if len(u.Providers) > 0 {
		fmt.Fprintf(v, " %sProviders:%s     %s\n", dimColor, resetColor, strings.Join(u.Providers, ", "))
	}
	if u.PhotoURL != "" {
		fmt.Fprintf(v, " %sPhoto URL:%s     %s\n", dimColor, resetColor, u.PhotoURL)
	}
}

func (g *Gui) renderRulesDetails(v *gocui.View) {
	dimColor := "\033[90m"
	resetColor := "\033[0m"
	cyanColor := "\033[36m"

	fmt.Fprintf(v, "%s─── Firestore Security Rules ───%s\n\n", cyanColor, resetColor)
	if g.firestoreRules == nil {
		fmt.Fprintf(v, "%sNo rules loaded. Switch to Rules tab to load.%s", dimColor, resetColor)
		return
	}
	fmt.Fprintf(v, "%s%s%s", dimColor, "Rules are shown in the Rules tab panel.", resetColor)
}

func (g *Gui) renderIndexesDetails(v *gocui.View) {
	dimColor := "\033[90m"
	resetColor := "\033[0m"
	cyanColor := "\033[36m"

	fmt.Fprintf(v, "%s─── Composite Indexes ───%s\n\n", cyanColor, resetColor)
	if len(g.firestoreIndexes) == 0 {
		fmt.Fprintf(v, "%sNo composite indexes. Switch to Indexes tab to load.%s", dimColor, resetColor)
		return
	}
	fmt.Fprintf(v, "%s%d composite indexes loaded. View in Indexes tab.%s", dimColor, len(g.firestoreIndexes), resetColor)
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
		// Build indentation with visual guides
		var indent string
		if node.Depth > 0 {
			// Build guide lines: use │ for each depth level except the last
			for d := 0; d < node.Depth-1; d++ {
				// Check if there's a sibling at this depth below
				hasMoreAtDepth := false
				for j := i + 1; j < len(filtered); j++ {
					if filtered[j].Depth <= d {
						break
					}
					if filtered[j].Depth == d+1 {
						hasMoreAtDepth = true
						break
					}
				}
				if hasMoreAtDepth {
					indent += "\033[90m│\033[0m "
				} else {
					indent += "  "
				}
			}
		}

		// Arrow and icon based on type and expanded state
		arrow := ""
		icon := icons.DOCUMENT
		iconColor := "\033[32m" // Green for documents
		resetColor := "\033[0m"

		if node.Type == "collection" {
			iconColor = "\033[36m" // Cyan for collections
			if node.Expanded {
				arrow = " ▼ "
				icon = icons.FOLDER_OPEN
			} else {
				arrow = " ▶ "
				icon = icons.FOLDER_CLOSED
			}
		} else if node.Depth > 0 {
			// Nested documents get spacing to align with arrows
			arrow = "   "
		}

		// Add spacing after icon if present
		if icon != "" {
			icon = icon + " "
		}

		// Tree connector with proper guide character
		connector := ""
		if node.Depth > 0 {
			// Check if there's a sibling after this node at the same depth
			hasMoreSiblings := false
			for j := i + 1; j < len(filtered); j++ {
				if filtered[j].Depth < node.Depth {
					break
				}
				if filtered[j].Depth == node.Depth {
					hasMoreSiblings = true
					break
				}
			}
			if hasMoreSiblings {
				connector = "\033[90m├─\033[0m"
			} else {
				connector = "\033[90m└─\033[0m"
			}
		}

		// Determine marker: * for current doc, + for selected in select mode, space otherwise
		marker := "  "
		isSelected := g.selectMode && g.selectedDocs[i]
		if isSelected {
			marker = "\033[2;33m+ \033[0m" // Dim yellow + for selected
		} else if node.Path == g.currentDocPath {
			marker = g.getActiveColorCode() + "* " + "\033[0m"
		}

		// Check if document is cached and show stats
		cachedIndicator := ""
		if node.Type == "document" {
			if _, ok := g.docCache[node.Path]; ok {
				if stats, hasStats := g.statsCache[node.Path]; hasStats && stats != nil {
					cachedIndicator = fmt.Sprintf(" \033[90m%dF %s\033[0m", stats.FieldCount, formatBytes(stats.SizeBytes))
				} else {
					cachedIndicator = " \033[33m·\033[0m" // Yellow dot for cached (no stats yet)
				}
			}
		}

		// Format: marker + indent + connector + arrow + colored_icon + name + cachedIndicator
		if isSelected {
			fmt.Fprintf(v, "%s%s%s%s%s%s%s\033[33m%s\033[0m%s\n", marker, indent, connector, arrow, iconColor, icon, resetColor, node.Name, cachedIndicator)
		} else {
			fmt.Fprintf(v, "%s%s%s%s%s%s%s%s%s\n", marker, indent, connector, arrow, iconColor, icon, resetColor, node.Name, cachedIndicator)
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
	// Scan results / progress (takes priority over everything)
	if g.scanRunning {
		v.Clear()
		fmt.Fprint(v, g.getLoadingText(fmt.Sprintf("Scanning collections... %s", g.scanProgress)))
		return
	}
	if g.scanResults != nil && g.currentDocData == nil {
		g.renderScanResults(v)
		return
	}

	// Determine context based on current panel (or previous if in details)
	activeTab := g.collectionsTab
	isFromCollections := g.currentColumn == "collections" || (g.currentColumn == "details" && g.previousColumn == "collections")

	// Functions context: show function details or logs
	if activeTab == "functions" && isFromCollections {
		if g.detailsTab == "logs" {
			g.renderFunctionLogs(v)
			return
		}
		if g.currentFunction != nil {
			v.Clear()
			g.renderFunctionDetails(v)
			return
		}
		v.Clear()
		fmt.Fprint(v, "\033[90mSelect a function to view details\033[0m")
		return
	}

	// Storage context: show object metadata
	if activeTab == "storage" && isFromCollections {
		v.Clear()
		g.renderStorageDetails(v)
		return
	}

	// Auth context: show user details
	if activeTab == "auth" && isFromCollections {
		v.Clear()
		g.renderAuthDetails(v)
		return
	}

	// Rules context: show rules in details panel too
	if activeTab == "rules" && isFromCollections {
		v.Clear()
		g.renderRulesDetails(v)
		return
	}

	// Indexes context
	if activeTab == "indexes" && isFromCollections {
		v.Clear()
		g.renderIndexesDetails(v)
		return
	}

	// Document/Collections context: show document data
	// Show loading indicator when details are being loaded
	if g.detailsLoading {
		v.Clear()
		fmt.Fprint(v, g.getLoadingText("Loading document..."))
		return
	}

	// Show document data if available
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

		// Format JSON (compact or pretty)
		var data []byte
		var err error
		if g.compactJSON {
			data, err = json.Marshal(g.currentDocData)
		} else {
			data, err = json.MarshalIndent(g.currentDocData, "", "  ")
		}
		if err != nil {
			v.SetContent(fmt.Sprintf("Error formatting data: %v\n", err))
			return
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("\033[36m─── %s ───\033[0m\n", g.currentDocPath))

		// Show stats for actual documents
		if strings.Contains(g.currentDocPath, "/") {
			var stats docStats
			if g.currentDocStats != nil {
				stats = docStats{
					sizeBytes:     g.currentDocStats.SizeBytes,
					fieldCount:    g.currentDocStats.FieldCount,
					leafFields:    g.currentDocStats.LeafFields,
					maxDepth:      g.currentDocStats.MaxDepth,
					maxFieldName:  g.currentDocStats.MaxFieldName,
					maxFieldValue: g.currentDocStats.MaxFieldValue,
					docPathLen:    g.currentDocStats.DocNameSize,
				}
			} else {
				stats = calculateDocStats(g.currentDocData, g.currentDocPath)
			}
			// Determine index certainty from composite index cache
			idxCert := indexCertaintyUnknown
			pathParts := strings.Split(g.currentDocPath, "/")
			if len(pathParts) >= 2 {
				collID := pathParts[len(pathParts)-2]
				if val, ok := g.compositeIndexCache[collID]; ok {
					if *val {
						idxCert = indexCertaintyApproximate
					} else {
						idxCert = indexCertaintyExact
					}
				}
			}
			content.WriteString(formatDocStats(stats, g.currentDocStats != nil, idxCert))
			content.WriteString("\n")
		}
		// Show schema summary (field names and types)
		schema := buildSchemaSummary(g.currentDocData)
		if schema != "" {
			content.WriteString(fmt.Sprintf("\033[90m%s\033[0m\n", schema))
		}
		content.WriteString("\n")

		// Colorize JSON, then annotate timestamps
		colorized := colorizeJSON(string(data))
		if g.humanizeTimestamps {
			// Annotate on raw text, then apply. We use the raw JSON lines to find timestamps
			colorized = annotateTimestamps(string(data), colorized)
		}

		// Optionally add line numbers
		if g.showLineNumbers {
			lines := strings.Split(colorized, "\n")
			width := len(fmt.Sprintf("%d", len(lines)))
			var numbered strings.Builder
			for i, line := range lines {
				fmt.Fprintf(&numbered, "\033[90m%*d \033[0m%s", width, i+1, line)
				if i < len(lines)-1 {
					numbered.WriteString("\n")
				}
			}
			colorized = numbered.String()
		}

		content.WriteString(colorized)

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
	fmt.Fprintln(v, "\033[33m            ▄")
	fmt.Fprintln(v, "\033[33m           ▄█▄")
	fmt.Fprintln(v, "\033[33m          ▄███▄")
	fmt.Fprintln(v, "\033[38;5;208m         ▄█▀▀▀█▄")
	fmt.Fprintln(v, "\033[38;5;208m        ▄█▀   ▀█▄")
	fmt.Fprintln(v, "\033[38;5;196m       ▄█▀     ▀█▄")
	fmt.Fprintln(v, "\033[38;5;196m      ▄█▀  ▄▄▄  ▀█▄")
	fmt.Fprintln(v, "\033[38;5;196m      █▀  ▄█▀█▄  ▀█")
	fmt.Fprintln(v, "\033[38;5;208m      █  ▄█▀ ▀█▄  █")
	fmt.Fprintln(v, "\033[38;5;208m      ▀█▄▀     ▀▄█▀")
	fmt.Fprintln(v, "\033[33m        ▀▀▄▄▄▄▄▀▀\033[0m")
	fmt.Fprintln(v, "")
	fmt.Fprintf(v, "\033[38;5;208m     %s  L A Z Y F I R E\033[0m\n", icons.FIREBASE_ICON)
	fmt.Fprintf(v, "\033[90m          v%s\033[0m\n", g.version)
	fmt.Fprintln(v, "")
	fmt.Fprintln(v, "\033[90m     Select a project to start\033[0m")
	fmt.Fprintln(v, "")
	fmt.Fprintln(v, "\033[90m     Created by Marjo Ballabani\033[0m")
	fmt.Fprintln(v, "\033[36m     github.com/marjoballabani/lazyfire\033[0m")
}

func (g *Gui) renderScanResults(v *gocui.View) {
	if g.cachedDetailsContent != "" && g.cachedDetailsDocPath == "__scan__" {
		if g.detailsViewDirty {
			v.SetContent(g.cachedDetailsContent)
			g.detailsViewDirty = false
		}
		return
	}

	g.detailsScrollPos = 0

	var content strings.Builder

	okCount, warnCount, skipCount := 0, 0, 0
	for _, r := range g.scanResults {
		switch r.Status {
		case "ok":
			okCount++
		case "warning":
			warnCount++
		case "skipped":
			skipCount++
		}
	}

	content.WriteString("\033[36m--- Collection Health Scan ---\033[0m\n\n")
	content.WriteString(fmt.Sprintf("  \033[90mProject:\033[0m  %s\n", g.currentProject))
	content.WriteString(fmt.Sprintf("  \033[90mScanned:\033[0m  %d collections\n", len(g.scanResults)))
	content.WriteString(fmt.Sprintf("  \033[32mHealthy:\033[0m  %d", okCount))
	if warnCount > 0 {
		content.WriteString(fmt.Sprintf("  \033[33mWarnings:\033[0m %d", warnCount))
	}
	if skipCount > 0 {
		content.WriteString(fmt.Sprintf("  \033[31mSkipped:\033[0m  %d", skipCount))
	}
	content.WriteString("\n\n")

	// Show warnings first, then ok, then skipped
	for _, status := range []string{"warning", "ok", "skipped"} {
		for _, r := range g.scanResults {
			if r.Status != status {
				continue
			}

			var icon, color string
			switch r.Status {
			case "ok":
				icon = "\033[32m[OK]\033[0m"
				color = "\033[0m"
			case "warning":
				icon = "\033[33m[!!]\033[0m"
				color = "\033[33m"
			case "skipped":
				icon = "\033[31m[--]\033[0m"
				color = "\033[90m"
			}

			content.WriteString(fmt.Sprintf("  %s %s%s\033[0m  %s%s\033[0m\n", icon, color, r.Collection, "\033[90m", r.Message))

			// Build warning set for highlighting
			warningSet := make(map[string]bool)
			for _, w := range r.Warnings {
				warningSet[w] = true
			}
			for _, m := range r.Metrics {
				if warningSet[m] {
					content.WriteString(fmt.Sprintf("       \033[33m! %s\033[0m\n", m))
				} else {
					content.WriteString(fmt.Sprintf("       \033[90m- %s\033[0m\n", m))
				}
			}
		}
	}

	content.WriteString("\n\033[90m  Press Esc to dismiss\033[0m\n")

	g.cachedDetailsContent = content.String()
	g.cachedDetailsDocPath = "__scan__"
	g.cachedDetailsLines = nil
	g.cachedDetailsHeader = ""
	v.SetContent(g.cachedDetailsContent)
	g.detailsViewDirty = false
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

	helpText := g.getContextHelpText()

	// Build breadcrumb path
	breadcrumb := g.buildBreadcrumb()
	versionText := fmt.Sprintf("\033[90mv%s\033[0m ", g.version)

	// Calculate padding to right-align breadcrumb + version
	width, _ := v.Size()
	helpLen := g.visibleLength(helpText)
	rightSide := breadcrumb + "  " + versionText
	rightLen := g.visibleLength(breadcrumb) + 2 + len(g.version) + 2
	padding := width - helpLen - rightLen
	if padding < 1 {
		padding = 1
	}

	fmt.Fprintf(v, "%s%*s%s", helpText, padding, "", rightSide)
}

func (g *Gui) getContextHelpText() string {
	c := "\033[36m" // cyan for keys
	y := "\033[33m" // yellow
	g2 := "\033[32m" // green
	m := "\033[35m" // magenta
	r := "\033[31m" // red
	x := "\033[0m"  // reset

	common := c + "←/→" + x + " cols  " + c + "j/k" + x + " move  " + c + "[/]" + x + " tabs  "

	switch g.currentColumn {
	case "projects":
		return " " + common + y + "space" + x + " select  " + m + "/" + x + " filter  " + c + "S" + x + " scan  " + m + "?" + x + " help  " + r + "q" + x + " quit"
	case "collections":
		switch g.collectionsTab {
		case "functions":
			return " " + common + y + "space" + x + " select  " + m + "/" + x + " filter  " + c + "r" + x + " refresh  " + m + "?" + x + " help  " + r + "q" + x + " quit"
		case "storage":
			return " " + common + y + "space" + x + " open  " + c + "esc" + x + " back  " + m + "?" + x + " help  " + r + "q" + x + " quit"
		case "auth":
			return " " + common + y + "space" + x + " details  " + m + "?" + x + " help  " + r + "q" + x + " quit"
		case "rules":
			return " " + common + m + "?" + x + " help  " + r + "q" + x + " quit"
		case "indexes":
			return " " + common + m + "?" + x + " help  " + r + "q" + x + " quit"
		default:
			return " " + common + y + "space" + x + " select  " + m + "/" + x + " filter  " + y + "F" + x + " query  " + g2 + "c" + x + " copy  " + m + "?" + x + " help  " + r + "q" + x + " quit"
		}
	case "tree":
		return " " + common + y + "space" + x + " expand  " + c + "enter" + x + " details  " + m + "/" + x + " filter  " + y + "F" + x + " query  " + g2 + "c" + x + " copy  " + m + "?" + x + " help  " + r + "q" + x + " quit"
	case "details":
		return " " + c + "j/k" + x + " scroll  " + c + "J/K" + x + " fast  " + c + "esc" + x + " back  " + c + "t" + x + " compact  " + c + "w" + x + " wrap  " + g2 + "c" + x + " copy  " + m + "/" + x + " search  " + m + "?" + x + " help  " + r + "q" + x + " quit"
	default:
		return " " + common + y + "space" + x + " select  " + m + "/" + x + " filter  " + m + "?" + x + " help  " + r + "q" + x + " quit"
	}
}

// buildBreadcrumb returns a breadcrumb path showing current navigation context
func (g *Gui) buildBreadcrumb() string {
	parts := []string{}
	if g.currentProject != "" {
		parts = append(parts, g.currentProject)
	}

	switch g.collectionsTab {
	case "storage":
		parts = append(parts, "storage")
		if g.currentBucket != "" {
			parts = append(parts, g.currentBucket)
			if g.storagePrefix != "" {
				parts = append(parts, g.storagePrefix)
			}
		}
	case "auth":
		parts = append(parts, "auth")
		if g.selectedAuthIdx < len(g.authUsers) && len(g.authUsers) > 0 {
			u := g.authUsers[g.selectedAuthIdx]
			if u.Email != "" {
				parts = append(parts, u.Email)
			} else {
				parts = append(parts, u.UID)
			}
		}
	case "rules":
		parts = append(parts, "rules")
	case "indexes":
		parts = append(parts, "indexes")
	case "functions":
		parts = append(parts, "functions")
		if g.currentFunction != nil {
			parts = append(parts, g.currentFunction.DisplayName)
		}
	default:
		if g.currentCollection != "" {
			parts = append(parts, g.currentCollection)
		}
		if g.currentDocPath != "" && !strings.HasPrefix(g.currentDocPath, "__") && !strings.Contains(g.currentDocPath, " selected") {
			docParts := strings.Split(g.currentDocPath, "/")
			if len(docParts) > 0 {
				parts = append(parts, docParts[len(docParts)-1])
			}
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return "\033[90m" + strings.Join(parts, " > ") + "\033[0m"
}

// visibleLength returns the visible length of a string (excluding ANSI codes)
func (g *Gui) visibleLength(s string) int {
	inEscape := false
	length := 0
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		length++
	}
	return length
}

// Firestore limits (https://firebase.google.com/docs/firestore/quotas)
const (
	maxDocSizeBytes    = 1048576         // 1 MiB
	maxIndexEntries    = 40000           // Firestore limit: 40k index entries per document
	maxDepth           = 20              // Maximum depth of nested maps/arrays
	maxFieldNameBytes  = 1500            // Maximum field name size
	maxFieldValueBytes = 1048576 - 89    // 1 MiB - 89 bytes
	maxDocNameBytes    = 6 * 1024        // 6 KiB for document path
)

// docStats holds document statistics
type docStats struct {
	sizeBytes       int
	fieldCount      int
	leafFields      int // leaf fields only (for index entry estimation)
	maxDepth        int
	maxFieldName    int // longest field name in bytes
	maxFieldValue   int // largest field value in bytes
	docPathLen      int // document name size per Firestore calculation
}

// calculateDocStats calculates all document statistics
func calculateDocStats(data map[string]any, docPath string) docStats {
	maxName, maxValue := findMaxFieldSizes(data)
	return docStats{
		sizeBytes:     firestoreDocSize(data, docPath),
		fieldCount:    countFields(data),
		leafFields:    countLeafFields(data),
		maxDepth:      calculateDepth(data),
		maxFieldName:  maxName,
		maxFieldValue: maxValue,
		docPathLen:    firestoreDocNameSize(docPath),
	}
}

// firestoreDocNameSize calculates the document name size per Firestore rules:
// sum of each path component's UTF-8 size + 1 byte per component + 16 bytes
func firestoreDocNameSize(docPath string) int {
	parts := strings.Split(docPath, "/")
	size := 16
	for _, part := range parts {
		size += len(part) + 1
	}
	return size
}

// firestoreValueSize returns the size of a value per Firestore storage rules.
func firestoreValueSize(val any) int {
	switch v := val.(type) {
	case nil:
		return 1
	case bool:
		return 1
	case float64:
		return 8 // double
	case string:
		return len(v) + 1
	case map[string]any:
		// Check if it's a GeoPoint (has exactly latitude + longitude)
		if len(v) == 2 {
			if _, hasLat := v["latitude"]; hasLat {
				if _, hasLng := v["longitude"]; hasLng {
					return 16
				}
			}
		}
		size := 0
		for key, val := range v {
			size += len(key) + 1 + firestoreValueSize(val)
		}
		return size
	case []any:
		size := 0
		for _, item := range v {
			size += firestoreValueSize(item)
		}
		return size
	default:
		return 0
	}
}

// firestoreDocSize calculates the document size per Firestore rules:
// document name size + sum of (field name size + 1 + field value size) + 32 bytes
func firestoreDocSize(data map[string]any, docPath string) int {
	size := firestoreDocNameSize(docPath) + 32
	for key, val := range data {
		size += len(key) + 1 + firestoreValueSize(val)
	}
	return size
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
			valSize := firestoreValueSize(val)
			if valSize > maxValue {
				maxValue = valSize
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

// countLeafFields counts only leaf fields (non-map, non-array values).
// Firestore only creates index entries for leaf fields, not intermediate map keys.
func countLeafFields(data any) int {
	switch v := data.(type) {
	case map[string]any:
		count := 0
		for _, val := range v {
			switch val.(type) {
			case map[string]any:
				count += countLeafFields(val)
			case []any:
				count += countLeafFields(val)
			default:
				count++ // leaf field
			}
		}
		return count
	case []any:
		count := 0
		for _, item := range v {
			count += countLeafFields(item)
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

// indexCertainty represents the certainty level of the index entries count.
// -1 = unknown (not checked yet or error), 0 = exact (no composite indexes), 1 = approximate (has composite indexes)
type indexCertainty int

const (
	indexCertaintyUnknown     indexCertainty = -1
	indexCertaintyExact       indexCertainty = 0
	indexCertaintyApproximate indexCertainty = 1
)

// formatDocStats returns a formatted string showing document stats with warnings
func formatDocStats(stats docStats, accurate bool, idxCertainty indexCertainty) string {
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

	// Line 1: Size, Index Entries, Depth
	indexEntries := stats.leafFields * 2 // default: 2 single-field indexes per leaf field (asc + desc)
	var indexLabel string
	switch idxCertainty {
	case indexCertaintyExact:
		indexLabel = fmt.Sprintf("%d", indexEntries)
	case indexCertaintyApproximate:
		indexLabel = fmt.Sprintf("~%d+", indexEntries)
	default:
		indexLabel = fmt.Sprintf("~%d", indexEntries)
	}
	line1 := fmt.Sprintf("\033[90mSize:\033[0m %s%s / 1MB\033[0m  \033[90mIndex Entries:\033[0m %s%s / %d\033[0m  \033[90mDepth:\033[0m %s%d / %d\033[0m",
		getColor(stats.sizeBytes, maxDocSizeBytes), formatBytes(stats.sizeBytes),
		getColor(indexEntries, maxIndexEntries), indexLabel, maxIndexEntries,
		getColor(stats.maxDepth, maxDepth), stats.maxDepth, maxDepth)

	// Line 2: Field Name, Field Value, Doc Path
	line2 := fmt.Sprintf("\033[90mField Name:\033[0m %s%d / %d B\033[0m  \033[90mField Value:\033[0m %s%s / 1MB\033[0m  \033[90mPath:\033[0m %s%d / %d B\033[0m",
		getColor(stats.maxFieldName, maxFieldNameBytes), stats.maxFieldName, maxFieldNameBytes,
		getColor(stats.maxFieldValue, maxFieldValueBytes), formatBytes(stats.maxFieldValue),
		getColor(stats.docPathLen, maxDocNameBytes), stats.docPathLen, maxDocNameBytes)

	source := "\033[90m[estimated]\033[0m"
	if accurate {
		source = "\033[90m[accurate]\033[0m"
	}

	return line1 + "\n" + line2 + "  " + source
}

// buildSchemaSummary creates a one-line summary of field names and their types
func buildSchemaSummary(data map[string]any) string {
	if len(data) == 0 {
		return ""
	}
	var parts []string
	for key, val := range data {
		typeName := inferType(val)
		parts = append(parts, key+":"+typeName)
	}
	// Sort for consistent output
	sortStrings(parts)
	summary := "Schema: " + strings.Join(parts, ", ")
	// Truncate if too long
	if len(summary) > 120 {
		summary = summary[:117] + "..."
	}
	return summary
}

// inferType returns a short type name for a value
func inferType(val any) string {
	if val == nil {
		return "null"
	}
	switch v := val.(type) {
	case string:
		// Check if it looks like a timestamp
		if len(v) > 18 && strings.Contains(v, "T") && strings.Contains(v, ":") {
			return "timestamp"
		}
		return "string"
	case float64:
		if v == float64(int64(v)) {
			return "int"
		}
		return "float"
	case bool:
		return "bool"
	case map[string]any:
		return "map"
	case []any:
		return "array"
	default:
		// Firestore integerValue comes as string from REST API
		s := fmt.Sprintf("%v", val)
		if _, err := fmt.Sscanf(s, "%d", new(int)); err == nil {
			return "int"
		}
		return "string"
	}
}

// sortStrings sorts a string slice in place (simple insertion sort)
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
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
