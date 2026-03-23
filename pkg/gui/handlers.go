package gui

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/gocui"
	"github.com/marjoballabani/lazyfire/pkg/firebase"
)

// State checking helpers

func (g *Gui) isModalOpen() bool {
	return g.modalOpen || g.helpOpen || g.confirmOpen
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

	switch g.collectionsTab {
	case "functions":
		g.logCommand("api", fmt.Sprintf("ListFunctions(%s) loading...", selectedProject.ID), "running")
		g.functionsLoading = true
	case "storage":
		g.logCommand("api", fmt.Sprintf("ListBuckets(%s) loading...", selectedProject.ID), "running")
		g.storageLoading = true
	case "auth":
		g.logCommand("api", fmt.Sprintf("ListAuthUsers(%s) loading...", selectedProject.ID), "running")
		g.authLoading = true
	case "rules":
		g.logCommand("api", fmt.Sprintf("GetRules(%s) loading...", selectedProject.ID), "running")
		g.rulesLoading = true
	case "indexes":
		g.logCommand("api", fmt.Sprintf("ListIndexes(%s) loading...", selectedProject.ID), "running")
		g.indexesLoading = true
	default:
		g.logCommand("api", fmt.Sprintf("ListCollections(%s) loading...", selectedProject.ID), "running")
		g.collectionsLoading = true
	}

	go func() {
		if err := g.firebaseClient.SetCurrentProject(selectedProject.ID); err != nil {
			g.g.Update(func(gui *gocui.Gui) error {
				g.collectionsLoading = false
				g.functionsLoading = false
				g.logCommand("api", fmt.Sprintf("SetProject failed: %v", err), "error")
				return nil
			})
			return
		}

		g.currentProject = selectedProject.ID
		// Clear collections state
		g.collections = nil
		g.treeNodes = nil
		g.currentDocData = nil
		g.currentDocStats = nil
		g.currentCollection = ""
		g.currentDocPath = ""
		g.selectedCollectionIdx = 0
		g.selectedTreeIdx = 0
		g.compositeIndexCache = make(map[string]*bool)
		// Clear functions state
		g.stopLogsRefresh()
		g.functions = nil
		g.currentFunction = nil
		g.functionLogs = nil
		g.selectedFunctionIdx = 0
		// Clear storage state
		g.storageBuckets = nil
		g.storageObjects = nil
		g.currentBucket = ""
		g.storagePrefix = ""
		g.storagePrefixStack = nil
		g.selectedBucketIdx = 0
		g.selectedObjectIdx = 0
		// Clear auth state
		g.authUsers = nil
		g.selectedAuthIdx = 0
		// Clear rules/indexes state
		g.firestoreRules = nil
		g.firestoreIndexes = nil

		// Load based on active tab
		switch g.collectionsTab {
		case "functions":
			g.loadFunctions()
		case "storage":
			g.loadStorageBuckets()
		case "auth":
			g.loadAuthUsers()
		case "rules":
			g.loadFirestoreRules()
		case "indexes":
			g.loadFirestoreIndexes()
		default:
			if err := g.loadCollections(); err != nil {
				g.g.Update(func(gui *gocui.Gui) error {
					g.collectionsLoading = false
					g.logCommand("api", fmt.Sprintf("ListCollections failed: %v", err), "error")
					return nil
				})
				return
			}

			g.g.Update(func(gui *gocui.Gui) error {
				g.collectionsLoading = false
				g.logCommand("api", fmt.Sprintf("ListCollections(%s) → %d collections", selectedProject.ID, len(g.collections)), "success")
				return nil
			})
		}
	}()

	return nil
}

func (g *Gui) selectCollection(gui *gocui.Gui) error {
	filtered := g.getFilteredCollections()
	if g.selectedCollectionIdx >= len(filtered) {
		return nil
	}

	// Clear select mode - tree will show different documents
	if g.selectMode {
		g.selectMode = false
		g.selectedDocs = make(map[int]bool)
	}

	collection := filtered[g.selectedCollectionIdx]
	g.currentCollection = collection.Name
	g.logCommand("api", fmt.Sprintf("ListDocuments(%s) loading...", collection.Name), "running")
	g.treeLoading = true

	go func() {
		docs, err := g.firebaseClient.ListDocuments(collection.Name, 50)
		if err != nil {
			g.g.Update(func(gui *gocui.Gui) error {
				g.treeLoading = false
				g.logCommand("api", fmt.Sprintf("ListDocuments failed: %v", err), "error")
				return nil
			})
			return
		}

		g.g.Update(func(gui *gocui.Gui) error {
			g.treeNodes = nil
			g.expandedPaths = make(map[string]bool)

			// Cache all fetched documents
			for _, doc := range docs {
				g.docCache[doc.Path] = doc.Data
			}

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
			g.treeLoading = false
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
			g.treeNodes[nodeIdx].Expanded = false
			return nil
		}

		// Check cache for document data
		cachedData, isCached := g.docCache[nodePath]
		if isCached {
			g.currentDocPath = nodePath
			g.currentDocData = cachedData
			g.currentDocStats = g.statsCache[nodePath]
			g.clearDetailsCache()
			g.logCommand("cache", fmt.Sprintf("Using cached %s", nodeName), "success")
			// Don't return - still need to load subcollections
		}

		// Load subcollections (and document if not cached)
		if !isCached {
			g.logCommand("api", fmt.Sprintf("GetDocument(%s) loading...", nodePath), "running")
			g.detailsLoading = true
		}

		go func() {
			var docData map[string]any
			var docStats *firebase.DocStats
			if isCached {
				docData = cachedData
				docStats = g.statsCache[nodePath]
			} else {
				doc, err := g.firebaseClient.GetDocument(nodePath)
				if err != nil {
					g.g.Update(func(gui *gocui.Gui) error {
						g.detailsLoading = false
						g.logCommand("api", fmt.Sprintf("GetDocument failed: %v", err), "error")
						return nil
					})
					return
				}
				docData = doc.Data
				docStats = doc.Stats
			}

			subcols, err := g.firebaseClient.ListSubcollections(nodePath)

			g.g.Update(func(gui *gocui.Gui) error {
				g.detailsLoading = false
				g.currentDocPath = nodePath
				g.currentDocData = docData
				g.currentDocStats = docStats
				g.docCache[nodePath] = docData // Cache for future use
				g.statsCache[nodePath] = docStats
				g.clearDetailsCache()

				// Async check for composite indexes if not cached
				parts := strings.Split(nodePath, "/")
				if len(parts) >= 2 {
					collID := parts[len(parts)-2]
					if _, ok := g.compositeIndexCache[collID]; !ok {
						go func() {
							hasComposite, err := g.firebaseClient.HasCompositeIndexes(collID)
							g.g.Update(func(gui *gocui.Gui) error {
								if err == nil {
									val := hasComposite
									g.compositeIndexCache[collID] = &val
									g.clearDetailsCache()
								}
								return nil
							})
						}()
					}
				}

				if err != nil || len(subcols) == 0 {
					if !isCached {
						g.logCommand("api", fmt.Sprintf("GetDocument(%s) → loaded", nodeName), "success")
					}
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

				if !isCached {
					g.logCommand("api", fmt.Sprintf("GetDocument(%s) → %d subcols", nodeName, len(subcols)), "success")
				}
				return nil
			})
		}()

	} else if nodeType == "collection" {
		if node.Expanded {
			g.collapseNode(nodeIdx)
			g.treeNodes[nodeIdx].Expanded = false
			return nil
		}

		// Check if collection contents are cached
		if cachedPaths, ok := g.collectionCache[nodePath]; ok {
			// Rebuild tree nodes from cache
			if nodeIdx < len(g.treeNodes) {
				newNodes := make([]TreeNode, 0, len(g.treeNodes)+len(cachedPaths))
				newNodes = append(newNodes, g.treeNodes[:nodeIdx+1]...)

				for _, docPath := range cachedPaths {
					// Extract doc ID from path
					parts := strings.Split(docPath, "/")
					docID := parts[len(parts)-1]
					docNode := TreeNode{
						Path:        docPath,
						Name:        docID,
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
			g.logCommand("cache", fmt.Sprintf("Using cached %s → %d docs", nodeName, len(cachedPaths)), "success")
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

				// Cache document data and collection contents
				var docPaths []string
				for _, doc := range docs {
					g.docCache[doc.Path] = doc.Data
					docPaths = append(docPaths, doc.Path)
				}
				g.collectionCache[nodePath] = docPaths

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

func (g *Gui) selectFunction(gui *gocui.Gui) error {
	filtered := g.getFilteredFunctions()
	if g.selectedFunctionIdx >= len(filtered) {
		return nil
	}

	selectedFunc := filtered[g.selectedFunctionIdx]
	isSameFunction := g.currentFunction != nil && g.currentFunction.Name == selectedFunc.Name

	g.currentFunction = &selectedFunc
	g.clearDetailsCache()

	// Only fetch logs if selecting a different function or no logs yet (and not already loading)
	if !isSameFunction {
		g.functionLogs = nil
		g.loadFunctionLogs()
	} else if len(g.functionLogs) == 0 && !g.logsLoading {
		g.loadFunctionLogs()
	}

	g.logCommand("functions", fmt.Sprintf("Selected %s", selectedFunc.DisplayName), "success")
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
			g.currentDocStats = nil
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
		{Key: "g/G", Label: "Go to top/bottom"},
		{Key: "PgUp/PgDn", Label: "Page up/down"},
		{Key: "1/2/3", Label: "Jump to panel"},
		{Key: "Space", Label: "Select / Expand", Action: g.doSpace},
		{Key: "/", Label: "Filter / Search", Action: g.doStartFilter},
		{Key: "Esc", Label: "Back / Collapse / Close"},
		{Key: "r", Label: "Refresh", Action: g.doRefresh},
		{Key: "R", Label: "Clear cache", Action: g.doClearCache},
		{Key: "i", Label: "Cache stats", Action: g.doShowCacheStats},
		{Key: "M", Label: "Collection memory", Action: g.doCollectionMemoryEstimate},
		{Key: "A", Label: "Field type analysis", Action: g.doFieldTypeAnalysis},
		{Key: "L", Label: "Cycle log level", Action: g.doCycleLogLevel},
		{Key: "0", Label: "Command log", Action: g.doFocusCommands},
		{Key: "@", Label: "Command log (modal)", Action: g.doToggleModal},
		{Key: "?", Label: "This help"},
		{Key: "q", Label: "Quit", Action: g.doQuit},
		{Key: "", Label: g.getPanelName(), IsHeader: true},
	}

	switch g.currentColumn {
	case "projects":
		items = append(items,
			PopupItem{Key: "Enter", Label: "Fetch project details", Action: g.doEnter},
			PopupItem{Key: "Space", Label: "Select project", Action: g.doSpace},
			PopupItem{Key: "S", Label: "Scan collections health", Action: g.doScanCollections},
		)
	case "collections":
		items = append(items,
			PopupItem{Key: "[ / ]", Label: "Cycle tabs (6 tabs)", Action: g.doSwitchTab},
			PopupItem{Key: "Space", Label: "Select / Navigate", Action: g.doSpace},
			PopupItem{Key: "Esc", Label: "Back (storage folders)"},
			PopupItem{Key: "F", Label: "Query builder", Action: g.doOpenQuery},
		)
	case "tree":
		items = append(items,
			PopupItem{Key: "Space", Label: "Expand / Collapse", Action: g.doSpace},
			PopupItem{Key: "Enter", Label: "Open in details", Action: g.doEnter},
			PopupItem{Key: "v", Label: "Select mode (multi-select)", Action: g.doToggleSelectMode},
			PopupItem{Key: "C", Label: "Collapse all nodes", Action: g.doCollapseAll},
			PopupItem{Key: "F", Label: "Query builder", Action: g.doOpenQuery},
			PopupItem{Key: "p", Label: "Copy path to clipboard", Action: g.doCopyPath},
			PopupItem{Key: "c", Label: "Copy JSON to clipboard", Action: g.doCopyJSON},
			PopupItem{Key: "s", Label: "Save JSON to Downloads", Action: g.doSaveJSON},
			PopupItem{Key: "x", Label: "Export all cached docs", Action: g.doExportCachedDocs},
			PopupItem{Key: "A", Label: "Field type analysis", Action: g.doFieldTypeAnalysis},
			PopupItem{Key: "M", Label: "Collection memory estimate", Action: g.doCollectionMemoryEstimate},
		)
	case "details":
		// Show [ / ] only when Functions tab is active
		if g.collectionsTab == "functions" {
			items = append(items, PopupItem{Key: "[ / ]", Label: "Switch Details/Logs", Action: g.doSwitchTab})
		}
		items = append(items,
			PopupItem{Key: "j/k", Label: "Scroll content"},
			PopupItem{Key: "J/K", Label: "Scroll 5 lines"},
			PopupItem{Key: "Ctrl+d/u", Label: "Half-page scroll"},
			PopupItem{Key: "Esc", Label: "Go back"},
			PopupItem{Key: "t", Label: "Toggle compact JSON", Action: g.doToggleCompactJSON},
			PopupItem{Key: "w", Label: "Toggle word wrap", Action: g.doToggleWrap},
			PopupItem{Key: "T", Label: "Toggle timestamps", Action: g.doToggleTimestamps},
			PopupItem{Key: "H", Label: "Toggle line numbers", Action: g.doToggleLineNumbers},
			PopupItem{Key: "n/N", Label: "Next/prev search match"},
			PopupItem{Key: "y", Label: "Copy field value", Action: g.doCopyFieldValue},
			PopupItem{Key: "D", Label: "Field size breakdown", Action: g.doFieldSizeBreakdown},
			PopupItem{Key: "B", Label: "Decode base64 value", Action: g.doToggleBase64Decode},
			PopupItem{Key: "p", Label: "Copy path to clipboard", Action: g.doCopyPath},
			PopupItem{Key: "c", Label: "Copy JSON to clipboard", Action: g.doCopyJSON},
			PopupItem{Key: "s", Label: "Save JSON to Downloads", Action: g.doSaveJSON},
			PopupItem{Key: "e", Label: "Open in editor", Action: g.doEditInEditor},
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
