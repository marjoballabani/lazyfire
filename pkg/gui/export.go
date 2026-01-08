package gui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/itchyny/gojq"
)

// copyJSONAction copies current document to clipboard
func (g *Gui) copyJSONAction() error {
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

// saveJSONAction saves current document to file
func (g *Gui) saveJSONAction() error {
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
// If a jq filter is active on details, returns the filtered result.
func (g *Gui) getDocumentToCopy() (map[string]any, string, error) {
	filtered := g.getFilteredTreeNodes()
	if g.currentColumn == "tree" && len(filtered) > 0 && g.selectedTreeIdx < len(filtered) {
		node := filtered[g.selectedTreeIdx]
		if node.Type == "document" {
			doc, err := g.firebaseClient.GetDocument(node.Path)
			if err != nil {
				return nil, "", fmt.Errorf("Failed to fetch document: %v", err)
			}
			g.currentDocData = doc.Data
			g.currentDocPath = node.Path
			return doc.Data, node.Path, nil
		}
		return nil, "", fmt.Errorf("Selected item is a collection, not a document")
	}

	if g.currentDocData != nil {
		// Check if jq filter is active on details - return filtered result
		if g.currentColumn == "details" {
			if jqResult, path, ok := g.getJqFilteredResult(); ok {
				return jqResult, path, nil
			}
		}
		return g.currentDocData, g.currentDocPath, nil
	}

	return nil, "", fmt.Errorf("No document selected")
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
