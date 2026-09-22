package gui

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/gocui"
	"github.com/marjoballabani/lazyfire/pkg/firebase"
)

// setFocus focuses a panel. Entering details remembers where we came from so
// esc/tab can return there.
func (g *Gui) setFocus(column string) error {
	if column == "details" && g.currentColumn != "details" {
		g.previousColumn = g.currentColumn
	}
	g.currentColumn = column
	// Databases load on first visit, like the collections panel tabs
	if column == "databases" && g.databases == nil && g.currentProject != "" && !g.databasesLoading {
		g.loadDatabases()
	}
	return g.relayout()
}

// relayout applies state changes to the views right away. Needed after focus
// changes so keys queued behind the current one reach the newly focused view.
func (g *Gui) relayout() error {
	if g.g == nil {
		return nil
	}
	return g.Layout(g.g)
}

// jumpTo returns a handler that focuses a side panel
func (g *Gui) jumpTo(column string) func() error {
	return func() error {
		return g.setFocus(column)
	}
}

func (g *Gui) focusDetails() error {
	return g.setFocus("details")
}

// Selection handlers - called by actions

func (g *Gui) selectProject() error {
	filtered := g.getFilteredProjects()
	if g.selectedProjectIndex >= len(filtered) {
		return nil
	}

	selectedProject := filtered[g.selectedProjectIndex]

	switch g.collectionsTab {
	case "functions":
		g.functionsLoading = true
	case "storage":
		g.storageLoading = true
	case "auth":
		g.authLoading = true
	case "rules":
		g.rulesLoading = true
	case "indexes":
		g.indexesLoading = true
	default:
		g.collectionsLoading = true
	}
	g.logCommand("api", fmt.Sprintf("SetProject(%s)...", selectedProject.ID), "running")

	go func() {
		err := g.firebaseClient.SetCurrentProject(selectedProject.ID)
		g.g.Update(func(gui *gocui.Gui) error {
			if err != nil {
				g.collectionsLoading = false
				g.functionsLoading = false
				g.storageLoading = false
				g.authLoading = false
				g.rulesLoading = false
				g.indexesLoading = false
				g.logCommand("api", fmt.Sprintf("SetProject failed: %v", err), "error")
				return nil
			}
			g.resetProjectState(selectedProject.ID)
			g.loadDatabases()
			g.loadActiveTab()
			return nil
		})
	}()

	return nil
}

// resetProjectState drops everything loaded for the previous project and
// goes back to its default database.
func (g *Gui) resetProjectState(projectID string) {
	g.projectGen++
	g.currentProject = projectID
	// Databases state
	g.databases = nil
	g.selectedDatabaseIdx = 0
	g.databasesFilter = ""
	g.databasesLoading = false
	g.currentDatabase = firebase.DefaultDatabase
	g.resetDatabaseState()
	// Functions state
	g.stopLogsRefresh()
	g.functions = nil
	g.currentFunction = nil
	g.functionLogs = nil
	g.selectedFunctionIdx = 0
	g.functionsLoading = false
	g.logsLoading = false
	// Storage state
	g.storageBuckets = nil
	g.storageObjects = nil
	g.currentBucket = ""
	g.storagePrefix = ""
	g.storagePrefixStack = nil
	g.selectedBucketIdx = 0
	g.selectedObjectIdx = 0
	g.storageLoading = false
	// Auth state
	g.authUsers = nil
	g.selectedAuthIdx = 0
	g.authLoading = false
	// Rules state
	g.firestoreRules = nil
	g.rulesLoading = false
	// Filters belong to the old project's data
	g.functionsFilter = ""
	g.storageFilter = ""
	g.authFilter = ""
}

// resetDatabaseState drops everything loaded from the previous Firestore
// database. Loads still running for it are dropped when they arrive, and
// caches are keyed by document path, which repeats across databases.
func (g *Gui) resetDatabaseState() {
	g.databaseGen++
	g.collectionsLoading = false
	g.treeLoading = false
	g.detailsLoading = false
	g.indexesLoading = false
	// Collections state
	g.collections = nil
	g.treeNodes = nil
	g.currentDocData = nil
	g.currentDocStats = nil
	g.currentCollection = ""
	g.currentDocPath = ""
	g.selectedCollectionIdx = 0
	g.selectedTreeIdx = 0
	g.expandedPaths = make(map[string]bool)
	g.selectMode = false
	g.selectedDocs = make(map[int]bool)
	g.queryResultMode = false
	g.docCache = make(map[string]map[string]any)
	g.statsCache = make(map[string]*firebase.DocStats)
	g.collectionCache = make(map[string][]string)
	g.compositeIndexCache = make(map[string]*bool)
	g.clearDetailsCache()
	// Indexes belong to a database
	g.firestoreIndexes = nil
	// Filters belong to the old database's data
	g.collectionsFilter = ""
	g.treeFilter = ""
	g.detailsFilter = ""
}

// selectDatabase switches to the highlighted Firestore database
func (g *Gui) selectDatabase() error {
	databases := g.getFilteredDatabases()
	if g.selectedDatabaseIdx >= len(databases) {
		return nil
	}
	db := databases[g.selectedDatabaseIdx]
	if db.ID == g.currentDatabase && g.collections != nil {
		return nil
	}

	g.firebaseClient.SetCurrentDatabase(db.ID)
	g.currentDatabase = db.ID
	g.resetDatabaseState()
	g.loadCollections()
	if g.collectionsTab == "indexes" {
		g.loadFirestoreIndexes()
	}
	g.logCommand("database", fmt.Sprintf("Using database %s", db.ID), "success")
	return nil
}

// openDatabase switches to the highlighted database and focuses its collections
func (g *Gui) openDatabase() error {
	if err := g.selectDatabase(); err != nil {
		return err
	}
	if g.collectionsTab != "collections" {
		if err := g.switchCollectionsTab("collections"); err != nil {
			return err
		}
	}
	return g.setFocus("collections")
}

// loadActiveTab loads the data shown by the active collections panel tab
func (g *Gui) loadActiveTab() {
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
		g.loadCollections()
	}
}

func (g *Gui) selectCollection() error {
	gen := g.databaseGen
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
	g.queryResultMode = false
	g.treeFilter = ""
	g.logCommand("api", fmt.Sprintf("ListDocuments(%s) loading...", collection.Name), "running")
	g.treeLoading = true

	go func() {
		docs, err := g.firebaseClient.ListDocuments(collection.Name, 50)
		if err != nil {
			g.g.Update(func(gui *gocui.Gui) error {
				if gen != g.databaseGen {
					return nil // loaded for another project or database
				}
				g.treeLoading = false
				g.logCommand("api", fmt.Sprintf("ListDocuments failed: %v", err), "error")
				return nil
			})
			return
		}

		g.g.Update(func(gui *gocui.Gui) error {
			if gen != g.databaseGen {
				return nil // loaded for another project or database
			}
			g.setTreeDocuments(collection.Path, docs)
			g.treeLoading = false
			g.logCommand("api", fmt.Sprintf("ListDocuments(%s) → %d docs", collection.Name, len(docs)), "success")
			return nil
		})
	}()

	return nil
}

// setTreeDocuments replaces the tree with the top-level documents of a
// collection. An empty collectionPath (query results) skips the collection cache.
func (g *Gui) setTreeDocuments(collectionPath string, docs []firebase.Document) {
	g.treeNodes = nil
	g.expandedPaths = make(map[string]bool)
	var docPaths []string
	for _, doc := range docs {
		g.docCache[doc.Path] = doc.Data
		docPaths = append(docPaths, doc.Path)
		g.treeNodes = append(g.treeNodes, TreeNode{
			Path:        doc.Path,
			Name:        doc.ID,
			Type:        "document",
			Depth:       0,
			HasChildren: true,
			Expanded:    false,
		})
	}
	if collectionPath != "" {
		g.collectionCache[collectionPath] = docPaths
	}
	g.selectedTreeIdx = 0
}

// openCollection opens the selected collection and focuses the tree
func (g *Gui) openCollection() error {
	if err := g.selectCollection(); err != nil {
		return err
	}
	return g.setFocus("tree")
}

// selectedTreeNode returns the highlighted tree node
func (g *Gui) selectedTreeNode() (TreeNode, bool) {
	filtered := g.getFilteredTreeNodes()
	if g.selectedTreeIdx < 0 || g.selectedTreeIdx >= len(filtered) {
		return TreeNode{}, false
	}
	return filtered[g.selectedTreeIdx], true
}

// openTreeNode shows a document in details, or expands a collection
func (g *Gui) openTreeNode() error {
	node, ok := g.selectedTreeNode()
	if !ok {
		return nil
	}
	if node.Type == "collection" {
		return g.selectTreeNode()
	}
	// In select mode with docs already loaded, just go to details
	if g.selectMode && g.currentDocData != nil {
		return g.setFocus("details")
	}
	if idx := g.treeNodeIndex(node.Path); idx >= 0 && g.treeNodes[idx].Expanded {
		// Already expanded: show it without collapsing its subcollections
		g.showDocument(node.Path)
	} else if err := g.selectTreeNode(); err != nil {
		return err
	}
	return g.setFocus("details")
}

// showDocument opens a document in details, fetching it if not cached
func (g *Gui) showDocument(path string) {
	gen := g.databaseGen
	if data, ok := g.docCache[path]; ok {
		g.currentDocPath = path
		g.currentDocData = data
		g.currentDocStats = g.statsCache[path]
		g.clearDetailsCache()
		return
	}

	g.detailsLoading = true
	g.logCommand("api", fmt.Sprintf("GetDocument(%s) loading...", path), "running")
	go func() {
		doc, err := g.firebaseClient.GetDocument(path)
		g.g.Update(func(gui *gocui.Gui) error {
			if gen != g.databaseGen {
				return nil // loaded for another project or database
			}
			g.detailsLoading = false
			if err != nil {
				g.logCommand("api", fmt.Sprintf("GetDocument failed: %v", err), "error")
				return nil
			}
			g.docCache[path] = doc.Data
			g.statsCache[path] = doc.Stats
			g.currentDocPath = path
			g.currentDocData = doc.Data
			g.currentDocStats = doc.Stats
			g.clearDetailsCache()
			g.logCommand("api", fmt.Sprintf("GetDocument(%s) → loaded", path), "success")
			return nil
		})
	}()
}

// refetchDocument reloads the open document, bypassing the cache
func (g *Gui) refetchDocument() {
	path := g.currentDocPath
	delete(g.docCache, path)
	g.showDocument(path)
}

// insertTreeChildren inserts nodes under the node at path and marks it
// expanded. The path is looked up again because the tree may have changed
// while the children were loading.
func (g *Gui) insertTreeChildren(path string, children []TreeNode) {
	nodeIdx := g.treeNodeIndex(path)
	if nodeIdx == -1 {
		return
	}
	g.collapseNode(nodeIdx)
	newNodes := make([]TreeNode, 0, len(g.treeNodes)+len(children))
	newNodes = append(newNodes, g.treeNodes[:nodeIdx+1]...)
	newNodes = append(newNodes, children...)
	newNodes = append(newNodes, g.treeNodes[nodeIdx+1:]...)
	g.treeNodes = newNodes
	g.treeNodes[nodeIdx].Expanded = true
}

func (g *Gui) selectTreeNode() error {
	gen := g.databaseGen
	selectedNode, ok := g.selectedTreeNode()
	if !ok {
		return nil
	}
	nodePath := selectedNode.Path
	nodeName := selectedNode.Name
	nodeDepth := selectedNode.Depth

	nodeIdx := g.treeNodeIndex(nodePath)
	if nodeIdx == -1 {
		return nil
	}
	node := &g.treeNodes[nodeIdx]

	if selectedNode.Type == "document" {
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
			} else {
				doc, err := g.firebaseClient.GetDocument(nodePath)
				if err != nil {
					g.g.Update(func(gui *gocui.Gui) error {
						if gen != g.databaseGen {
							return nil // loaded for another project or database
						}
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
				if gen != g.databaseGen {
					return nil // loaded for another project or database
				}
				g.detailsLoading = false
				if !isCached {
					g.statsCache[nodePath] = docStats
				}
				g.currentDocPath = nodePath
				g.currentDocData = docData
				g.currentDocStats = g.statsCache[nodePath]
				g.docCache[nodePath] = docData // Cache for future use
				g.clearDetailsCache()

				// Async check for composite indexes if not cached
				parts := strings.Split(nodePath, "/")
				if len(parts) >= 2 {
					collID := parts[len(parts)-2]
					if _, ok := g.compositeIndexCache[collID]; !ok {
						go func() {
							hasComposite, err := g.firebaseClient.HasCompositeIndexes(collID)
							g.g.Update(func(gui *gocui.Gui) error {
								if gen != g.databaseGen {
									return nil // loaded for another project or database
								}
								if err == nil {
									val := hasComposite
									g.compositeIndexCache[collID] = &val
									// Re-render the header; keeps cursor and scroll
									g.cachedDetailsContent = ""
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

				children := make([]TreeNode, 0, len(subcols))
				for _, sub := range subcols {
					children = append(children, TreeNode{
						Path:        sub.Path,
						Name:        sub.Name,
						Type:        "collection",
						Depth:       nodeDepth + 1,
						HasChildren: true,
						Expanded:    false,
					})
				}
				g.insertTreeChildren(nodePath, children)

				if !isCached {
					g.logCommand("api", fmt.Sprintf("GetDocument(%s) → %d subcols", nodeName, len(subcols)), "success")
				}
				return nil
			})
		}()

	} else if selectedNode.Type == "collection" {
		if node.Expanded {
			g.collapseNode(nodeIdx)
			g.treeNodes[nodeIdx].Expanded = false
			return nil
		}

		// Check if collection contents are cached
		if cachedPaths, ok := g.collectionCache[nodePath]; ok {
			children := make([]TreeNode, 0, len(cachedPaths))
			for _, docPath := range cachedPaths {
				// Extract doc ID from path
				parts := strings.Split(docPath, "/")
				children = append(children, TreeNode{
					Path:        docPath,
					Name:        parts[len(parts)-1],
					Type:        "document",
					Depth:       nodeDepth + 1,
					HasChildren: true,
					Expanded:    false,
				})
			}
			g.insertTreeChildren(nodePath, children)
			g.logCommand("cache", fmt.Sprintf("Using cached %s → %d docs", nodeName, len(cachedPaths)), "success")
			return nil
		}

		g.logCommand("api", fmt.Sprintf("ListDocuments(%s) loading...", nodePath), "running")

		go func() {
			docs, err := g.firebaseClient.ListDocuments(nodePath, 50)
			if err != nil {
				g.g.Update(func(gui *gocui.Gui) error {
					if gen != g.databaseGen {
						return nil // loaded for another project or database
					}
					g.logCommand("api", fmt.Sprintf("ListDocuments failed: %v", err), "error")
					return nil
				})
				return
			}

			g.g.Update(func(gui *gocui.Gui) error {
				if gen != g.databaseGen {
					return nil // loaded for another project or database
				}
				if len(docs) == 0 {
					g.logCommand("api", fmt.Sprintf("ListDocuments(%s) → empty", nodeName), "success")
					return nil
				}

				// Cache document data and collection contents
				var docPaths []string
				children := make([]TreeNode, 0, len(docs))
				for _, doc := range docs {
					g.docCache[doc.Path] = doc.Data
					docPaths = append(docPaths, doc.Path)
					children = append(children, TreeNode{
						Path:        doc.Path,
						Name:        doc.ID,
						Type:        "document",
						Depth:       nodeDepth + 1,
						HasChildren: true,
						Expanded:    false,
					})
				}
				g.collectionCache[nodePath] = docPaths
				g.insertTreeChildren(nodePath, children)

				g.logCommand("api", fmt.Sprintf("ListDocuments(%s) → %d docs", nodeName, len(docs)), "success")
				return nil
			})
		}()
	}

	return nil
}

func (g *Gui) selectFunction() error {
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

// openFunction selects the function and focuses details to see its logs
func (g *Gui) openFunction() error {
	if err := g.selectFunction(); err != nil {
		return err
	}
	return g.setFocus("details")
}

func (g *Gui) fetchProjectDetails() error {
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

// Keybindings menu

// buildHelpPopup lists the bindings active in the focused panel, grouped like
// lazygit's keybindings menu: panel actions, navigation, then global keys
func (g *Gui) buildHelpPopup() {
	ctx := g.currentContext()
	local, nav, global := g.contextBindings(ctx)

	var items []PopupItem
	addSection := func(title string, bindings []*Binding) {
		var section []PopupItem
		seen := make(map[string]bool)
		for _, b := range bindings {
			if b.Description == "" || seen[b.Description] {
				continue
			}
			seen[b.Description] = true
			section = append(section, PopupItem{Key: b.keyLabel(), Label: b.Description, Binding: b})
		}
		if len(section) == 0 {
			return
		}
		items = append(items, PopupItem{Label: title, IsHeader: true})
		items = append(items, section...)
	}
	addSection(g.getContextName(ctx), local)
	addSection("Navigation", nav)
	addSection("Global", global)

	g.helpPopup = NewPopup("Keybindings", items, g.theme)
}

func (g *Gui) renderHelpContent(v *gocui.View) {
	if g.helpPopup == nil {
		return
	}
	g.helpPopup.Render(v)
}

// getContextName returns a display name for a context
func (g *Gui) getContextName(ctx Context) string {
	switch ctx {
	case CtxProjects:
		return "Projects"
	case CtxDatabases:
		return "Databases"
	case CtxCollections:
		return "Collections"
	case CtxFunctions:
		return "Functions"
	case CtxStorage:
		return "Storage"
	case CtxAuth:
		return "Auth"
	case CtxRules:
		return "Rules"
	case CtxIndexes:
		return "Indexes"
	case CtxTree:
		return "Tree"
	case CtxDetails:
		return "Details"
	case CtxMenu:
		return "Keybindings"
	default:
		return "Panel"
	}
}
