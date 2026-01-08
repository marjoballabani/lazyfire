package icons

import "testing"

func TestSetEnabled(t *testing.T) {
	// Save original state
	originalEnabled := enabled

	// Test enabling
	SetEnabled(true)
	if !IsEnabled() {
		t.Error("IsEnabled() should be true after SetEnabled(true)")
	}

	// Test disabling
	SetEnabled(false)
	if IsEnabled() {
		t.Error("IsEnabled() should be false after SetEnabled(false)")
	}

	// Verify icons are cleared when disabled
	if PROJECT_ICON != "" {
		t.Error("PROJECT_ICON should be empty when disabled")
	}
	if COLLECTION_ICON != "" {
		t.Error("COLLECTION_ICON should be empty when disabled")
	}

	// Verify fallback icons are set
	if SELECTED != "✓" {
		t.Errorf("SELECTED should be '✓' when disabled, got %q", SELECTED)
	}
	if ERROR != "✗" {
		t.Errorf("ERROR should be '✗' when disabled, got %q", ERROR)
	}
	if ARROW_RIGHT != ">" {
		t.Errorf("ARROW_RIGHT should be '>' when disabled, got %q", ARROW_RIGHT)
	}

	// Restore original state
	enabled = originalEnabled
}

func TestPatchForNerdFontsV2(t *testing.T) {
	// Save original values
	origFolder := FOLDER_CLOSED
	origDocument := DOCUMENT

	PatchForNerdFontsV2()

	// Verify v2 icons are set
	if FOLDER_CLOSED != "\uf07b" {
		t.Errorf("FOLDER_CLOSED should be patched for v2, got %q", FOLDER_CLOSED)
	}
	if FOLDER_OPEN != "\uf07c" {
		t.Errorf("FOLDER_OPEN should be patched for v2, got %q", FOLDER_OPEN)
	}
	if DOCUMENT != "\uf0f6" {
		t.Errorf("DOCUMENT should be patched for v2, got %q", DOCUMENT)
	}

	// Restore original values
	FOLDER_CLOSED = origFolder
	DOCUMENT = origDocument
}
