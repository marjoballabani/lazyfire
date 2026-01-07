package icons

// Nerd Font icons for lazyfire UI
// These require a Nerd Font to display correctly
// See: https://www.nerdfonts.com/cheat-sheet

var enabled = true

// IsEnabled returns whether icons are enabled
func IsEnabled() bool {
	return enabled
}

// SetEnabled enables or disables icons globally
func SetEnabled(e bool) {
	enabled = e
	if !e {
		disableAllIcons()
	}
}

var (
	// Panel title icons
	FIREBASE_ICON   = "\U000f0967" // 󰥧 (firebase)
	PROJECT_ICON    = "\U000f0766" // 󰝦 (package)
	COLLECTION_ICON = "\U000f024b" // 󰉋 (folder)
	TREE_ICON       = "\U000f0645" // 󰙅 (file-tree)
	DETAILS_ICON    = "\U000f0219" // 󰈙 (file-document)
	COMMAND_ICON    = "\U000f018d" // 󰆍 (console)
	KEYBOARD_ICON   = "\U000f030c" // 󰌌 (keyboard)

	// Tree view icons
	FOLDER_CLOSED = "\U000f024b" // 󰉋
	FOLDER_OPEN   = "\U000f0770" // 󰝰
	DOCUMENT      = "\U000f0219" // 󰈙
	DOCUMENT_JSON = "\U000f0626" // 󰘦

	// Status icons
	SELECTED = "\U000f012c" // 󰄬 (check)
	LOADING  = "\U000f0772" // 󰝲 (loading)
	ERROR    = "\U000f0159" // 󰅙 (close-circle)
	SUCCESS  = "\U000f0134" // 󰄴 (check-circle)
	WARNING  = "\U000f0026" // 󰀦 (alert)

	// Action icons
	REFRESH = "\U000f0450" // 󰑐 (refresh)
	COPY    = "\U000f018f" // 󰆏 (content-copy)
	SAVE    = "\U000f0193" // 󰆓 (content-save)
	SEARCH  = "\U000f0349" // 󰍉 (magnify)
	HELP    = "\U000f02d7" // 󰋗 (help-circle)
	QUIT    = "\U000f0156" // 󰅖 (close)

	// Navigation
	ARROW_RIGHT    = "\U000f0054" // 󰁔
	ARROW_DOWN     = "\U000f0047" // 󰁇
	ARROW_EXPAND   = "\U000f0142" // 󰅂
	ARROW_COLLAPSE = "\U000f0140" // 󰅀
)

// disableAllIcons sets all icons to empty strings for graceful fallback
func disableAllIcons() {
	FIREBASE_ICON = ""
	PROJECT_ICON = ""
	COLLECTION_ICON = ""
	TREE_ICON = ""
	DETAILS_ICON = ""
	COMMAND_ICON = ""
	KEYBOARD_ICON = ""
	FOLDER_CLOSED = ""
	FOLDER_OPEN = ""
	DOCUMENT = ""
	DOCUMENT_JSON = ""
	SELECTED = "✓"
	LOADING = "…"
	ERROR = "✗"
	SUCCESS = "✓"
	WARNING = "!"
	REFRESH = ""
	COPY = ""
	SAVE = ""
	SEARCH = ""
	HELP = ""
	QUIT = ""
	ARROW_RIGHT = ">"
	ARROW_DOWN = "v"
	ARROW_EXPAND = "+"
	ARROW_COLLAPSE = "-"
}

// PatchForNerdFontsV2 updates icons for Nerd Fonts v2 compatibility
func PatchForNerdFontsV2() {
	FIREBASE_ICON = "\uf6b1"
	FOLDER_CLOSED = "\uf07b"
	FOLDER_OPEN = "\uf07c"
	DOCUMENT = "\uf0f6"
}
