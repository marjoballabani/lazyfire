# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.36] - 2025-01-13

### Added
- **Cloud Functions v2 support** - Now fetches both 1st and 2nd generation Cloud Functions
  - Functions list shows both v1 and v2 functions with version indicator
  - Logs support both v1 (Cloud Functions) and v2 (Cloud Run) resource types
  - Combined logs sorted by timestamp

## [0.1.35] - 2025-01-13

### Added
- **Cloud Functions Browser** - View and monitor Cloud Functions for your Firebase projects
  - New **Functions tab** in Collections panel - press `[` / `]` to switch between Collections and Functions
  - View function details: name, status, runtime, region, memory, timeout, trigger type, URL
  - **Live Logs** - View function logs with color-coded severity (INFO/WARNING/ERROR/DEBUG)
  - **Details/Logs tabs** in Details panel when viewing Functions - press `[` / `]` to switch
  - Press `r` to refresh logs manually
  - Function state preserved when switching tabs
- **Separate details state per context** - Details panel remembers what you were viewing:
  - Collections/Tree context → shows document data
  - Functions context → shows function details/logs
  - Switching between contexts automatically swaps the Details view
- **Context-aware help popup** - `[` / `]` keybinding now shows per-panel descriptions

### Changed
- Select mode now only applies when focused on Tree panel (preserves selection when navigating)
- Selecting a different collection clears select mode (tree content changes)
- Tab switching (`[` / `]`) behavior is now panel-specific:
  - Collections panel: Switch Collections/Functions tabs
  - Details panel: Switch Details/Logs tabs (only when Functions tab is active)

## [0.1.34] - 2025-01-09

### Added
- **Query Builder** - Press `F` (Shift+F) to open interactive query builder
  - Filter documents with WHERE clauses (==, !=, <, <=, >, >=, in, array-contains)
  - ORDER BY field with ASC/DESC direction
  - LIMIT results
  - Works on collections panel and subcollections in tree
  - Clear button to reset all filters
  - Execute button to run query
  - Popup selector for operators and value types
- **Smart Document Caching**
  - Documents cached after fetch (yellow dot · indicator in tree)
  - Collection contents cached (re-expanding uses cache)
  - Multi-select uses cache for already-fetched documents
- **Tree View Improvements**
  - Arrow indicators (▶ collapsed, ▼ expanded) for collections
  - Cyan colored folder icons for collections
  - Green colored document icons
  - Proper indentation alignment for nested items
- **Projects Panel Improvements**
  - Firebase icon (orange) for projects list
  - Matches welcome screen branding
- **Collections Panel Improvements**
  - Cyan colored folder icons
- **Welcome Screen Improvements**
  - New flame ASCII art with hollow middle design
  - Version number display
  - Credits: "Created by Marjo Ballabani"
  - GitHub repository link
- **Open in editor** - press `e` in details panel to open JSON in external editor
  - Uses `$EDITOR` or `$VISUAL` environment variable
  - Falls back to `nvim` if installed, otherwise `vim`

### Changed
- **Visual Select Mode** now uses range-based selection (like vim visual mode)
  - Can select in either direction (up or down from start)
  - Dim yellow + marker for selected items
- Query builder keybinding changed from `Q` to `F` (Shift+F) to avoid accidental quit

## [0.1.33] - 2025-01-09

### Added
- **Unit tests** for core functionality (filter, JSON colorizer, document stats, config, icons)
- **Enhanced CI pipeline** with coverage reporting and linting
- **`showIcons` config option** to easily enable/disable icons
- **GitHub Wiki** documentation for all features

### Changed
- **New default theme** with Catppuccin-inspired colors (pink active border, muted inactive)

## [0.1.32] - 2025-01-08

### Added
- **Visual select mode** for multi-document operations in tree panel:
  - Press `v` to enter select mode
  - Move with `j`/`k` to extend/shrink selection
  - Press `Space` to fetch all selected documents in parallel
  - Press `Enter` to view fetched documents in details
  - Press `Esc` to exit select mode (only in tree panel)
  - Selection persists when viewing details
- **Parallel document fetching** for faster multi-document loads
- **Document stats display** showing Firestore limits compliance:
  - Document size (1 MiB limit)
  - Field count (20,000 limit)
  - Nesting depth (20 levels limit)
  - Largest field name (1,500 bytes limit)
  - Largest field value (~1 MiB limit)
  - Document path length (6 KiB limit)
- **Color-coded limit warnings** with 5 tiers (green/cyan/yellow/orange/red)
- **Animated loading spinner** in all panels (projects, collections, tree, details)

### Changed
- **Faster startup** - removed redundant Firebase API call during initialization
- **Improved syntax highlighting** - now using chroma library for faster, more accurate JSON colorization
- **Optimized details view** - cached rendering prevents redundant redraws on every layout
- Details scroll position now resets only when viewing a new document

### Fixed
- Escape from details panel now correctly returns to previous panel

## [0.1.31] - 2025-01-08

### Added
- **jq query support** for filtering JSON in details panel - use `.fieldName` syntax
- Copy/save now respects jq filter - exports filtered result when jq query is active
- Pagination for collections - now fetches all collections (was limited to 100)
- Filter input now supports all jq syntax characters (`[]|(){}:?` etc.)
- THIRD_PARTY_LICENSES file for open source compliance
