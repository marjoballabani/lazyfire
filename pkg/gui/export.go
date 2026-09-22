package gui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/itchyny/gojq"
	"github.com/jesseduffield/gocui"
)

// copyJSONAction copies current document to clipboard
func (g *Gui) copyJSONAction() error {
	g.withDocumentToCopy("copy", func(docData any, docPath string) {
		data, err := json.MarshalIndent(docData, "", "  ")
		if err != nil {
			g.logCommand("copy", fmt.Sprintf("Failed to marshal JSON: %v", err), "error")
			return
		}
		if err := copyToClipboard(string(data)); err != nil {
			g.logCommand("copy", fmt.Sprintf("Failed to copy: %v", err), "error")
			return
		}
		g.logCommand("copy", fmt.Sprintf("Copied %s to clipboard", docPath), "success")
	})
	return nil
}

// saveJSONAction saves current document to file
func (g *Gui) saveJSONAction() error {
	g.withDocumentToCopy("save", func(docData any, docPath string) {
		data, err := json.MarshalIndent(docData, "", "  ")
		if err != nil {
			g.logCommand("save", fmt.Sprintf("Failed to marshal JSON: %v", err), "error")
			return
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
			return
		}

		g.logCommand("save", fmt.Sprintf("Saved to %s", fullPath), "success")
	})
	return nil
}

// withDocumentToCopy calls fn with the document to copy or save: the
// highlighted document in the tree, else the one in details (its jq result
// if a jq filter is active). A tree document missing from the cache is
// fetched in the background first.
func (g *Gui) withDocumentToCopy(action string, fn func(data any, path string)) {
	gen := g.databaseGen
	if g.currentColumn == "tree" {
		node, ok := g.selectedTreeNode()
		if !ok || node.Type != "document" {
			g.logCommand(action, "Selected item is a collection, not a document", "error")
			return
		}
		if data, ok := g.docCache[node.Path]; ok {
			fn(data, node.Path)
			return
		}
		g.logCommand(action, fmt.Sprintf("GetDocument(%s) loading...", node.Path), "running")
		go func() {
			doc, err := g.firebaseClient.GetDocument(node.Path)
			g.g.Update(func(*gocui.Gui) error {
				if gen != g.databaseGen {
					return nil // loaded for another project or database
				}
				if err != nil {
					g.logCommand(action, fmt.Sprintf("Failed to fetch document: %v", err), "error")
					return nil
				}
				g.docCache[node.Path] = doc.Data
				g.statsCache[node.Path] = doc.Stats
				fn(doc.Data, node.Path)
				return nil
			})
		}()
		return
	}

	if g.currentDocData == nil {
		g.logCommand(action, "No document selected", "error")
		return
	}
	if jqResult, path, ok := g.getJqFilteredResult(); ok {
		fn(jqResult, path)
		return
	}
	fn(g.currentDocData, g.currentDocPath)
}

// getJqFilteredResult returns the jq-filtered result if a jq filter is active
func (g *Gui) getJqFilteredResult() (map[string]any, string, bool) {
	filter := g.getDetailsFilter()
	if !strings.HasPrefix(filter, ".") {
		return nil, "", false
	}

	jqQuery, err := gojq.Parse(filter)
	if err != nil {
		return nil, "", false
	}

	iter := jqQuery.Run(g.currentDocData)
	result, ok := iter.Next()
	if !ok {
		return nil, "", false
	}

	if _, isErr := result.(error); isErr {
		return nil, "", false
	}

	// Convert result to map if possible
	if resultMap, ok := result.(map[string]any); ok {
		path := fmt.Sprintf("%s (jq: %s)", g.currentDocPath, filter)
		return resultMap, path, true
	}

	// Wrap non-map results
	path := fmt.Sprintf("%s (jq: %s)", g.currentDocPath, filter)
	return map[string]any{"result": result}, path, true
}
