package gui

import (
	"fmt"

	"github.com/jesseduffield/gocui"
)

// State checking helpers

func (g *Gui) isModalOpen() bool {
	return g.modalOpen || g.helpOpen
}

// setFocus sets the current column and updates gocui's current view
func (g *Gui) setFocus(gui *gocui.Gui, column string) error {
	g.currentColumn = column
	if _, err := gui.SetCurrentView(column); err != nil {
		return err
	}
	return nil
}

// Selection handlers - called by actions

func (g *Gui) selectProject(gui *gocui.Gui) error {
	filtered := g.getFilteredProjects()
	if g.selectedProjectIndex >= len(filtered) {
		return nil
	}

	selectedProject := filtered[g.selectedProjectIndex]
	g.logCommand("api", fmt.Sprintf("ListCollections(%s) loading...", selectedProject.ID), "running")

	go func() {
		if err := g.firebaseClient.SetCurrentProject(selectedProject.ID); err != nil {
			g.g.Update(func(gui *gocui.Gui) error {
				g.logCommand("api", fmt.Sprintf("SetProject failed: %v", err), "error")
				return nil
			})
			return
		}

		g.currentProject = selectedProject.ID
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
	g.logCommand("api", fmt.Sprintf("ListDocuments(%s) loading...", collection.Name), "running")

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

	selectedNode := filtered[g.selectedTreeIdx]
	nodePath := selectedNode.Path
	nodeName := selectedNode.Name
	nodeDepth := selectedNode.Depth
	nodeType := selectedNode.Type

	originalIdx := g.getOriginalTreeNodeIndex(g.selectedTreeIdx)
	if originalIdx == -1 {
		return nil
	}
	node := &g.treeNodes[originalIdx]
	nodeIdx := originalIdx

	if nodeType == "document" {
		if node.Expanded {
			g.collapseNode(nodeIdx)
			node.Expanded = false
			return nil
		}

		g.logCommand("api", fmt.Sprintf("GetDocument(%s) loading...", nodePath), "running")

		go func() {
			doc, err := g.firebaseClient.GetDocument(nodePath)
			if err != nil {
				g.g.Update(func(gui *gocui.Gui) error {
					g.logCommand("api", fmt.Sprintf("GetDocument failed: %v", err), "error")
					return nil
				})
				return
			}

			subcols, err := g.firebaseClient.ListSubcollections(nodePath)

			g.g.Update(func(gui *gocui.Gui) error {
				g.currentDocPath = nodePath
				g.currentDocData = doc.Data

				if err != nil || len(subcols) == 0 {
					g.logCommand("api", fmt.Sprintf("GetDocument(%s) → loaded", nodeName), "success")
					return nil
				}

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
			g.collapseNode(nodeIdx)
			node.Expanded = false
			return nil
		}

		g.logCommand("api", fmt.Sprintf("ListDocuments(%s) loading...", nodePath), "running")

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
			g.currentDocData = nil
			g.logCommand("api", fmt.Sprintf("GetProjectDetails(%s) → success", project.ID), "success")
			return nil
		})
	}()

	return nil
}

func (g *Gui) collapseNode(idx int) {
	if idx >= len(g.treeNodes) {
		return
	}

	node := g.treeNodes[idx]
	nodeDepth := node.Depth

	endIdx := idx + 1
	for endIdx < len(g.treeNodes) && g.treeNodes[endIdx].Depth > nodeDepth {
		endIdx++
	}

	if endIdx > idx+1 {
		g.treeNodes = append(g.treeNodes[:idx+1], g.treeNodes[endIdx:]...)
	}
}

// Help popup builder

func (g *Gui) buildHelpPopup() {
	items := []PopupItem{
		{Key: "", Label: "Global", IsHeader: true},
		{Key: "←/→ h/l", Label: "Switch panels"},
		{Key: "↑/↓ j/k", Label: "Move up/down"},
		{Key: "Space", Label: "Select / Expand", Action: g.doSpace},
		{Key: "/", Label: "Filter / Search", Action: g.doStartFilter},
		{Key: "Esc", Label: "Back / Collapse / Close"},
		{Key: "r", Label: "Refresh", Action: g.doRefresh},
		{Key: "@", Label: "Command log", Action: g.doToggleModal},
		{Key: "?", Label: "This help"},
		{Key: "q", Label: "Quit", Action: g.doQuit},
		{Key: "", Label: g.getPanelName(), IsHeader: true},
	}

	switch g.currentColumn {
	case "projects":
		items = append(items,
			PopupItem{Key: "Enter", Label: "Fetch project details", Action: g.doEnter},
			PopupItem{Key: "Space", Label: "Select project", Action: g.doSpace},
		)
	case "collections":
		items = append(items,
			PopupItem{Key: "Space", Label: "Load documents", Action: g.doSpace},
		)
	case "tree":
		items = append(items,
			PopupItem{Key: "Space", Label: "View document / Expand", Action: g.doSpace},
			PopupItem{Key: "c", Label: "Copy JSON to clipboard", Action: g.doCopyJSON},
			PopupItem{Key: "s", Label: "Save JSON to Downloads", Action: g.doSaveJSON},
		)
	case "details":
		items = append(items,
			PopupItem{Key: "j/k", Label: "Scroll content"},
			PopupItem{Key: "c", Label: "Copy JSON to clipboard", Action: g.doCopyJSON},
			PopupItem{Key: "s", Label: "Save JSON to Downloads", Action: g.doSaveJSON},
		)
	}

	g.helpPopup = NewPopup("Keyboard Shortcuts", items, g.theme, g.views.helpModal)
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
