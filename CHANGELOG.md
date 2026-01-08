# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.34] - 2025-01-09

### Added
- **Open in editor** - press `e` in details panel to open JSON in external editor
  - Uses `$EDITOR` or `$VISUAL` environment variable
  - Falls back to `nvim` if installed, otherwise `vim`

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
