# Keybindings Reference

Complete reference of all keyboard shortcuts in LazyFire.

## Global Keys

| Key | Action |
|-----|--------|
| `q` | Quit application |
| `?` | Toggle help popup |
| `@` | Toggle command log |
| `Esc` | Cancel/close/go back |

## Navigation

### Panel Movement

| Key | Action |
|-----|--------|
| `h` | Move to left panel |
| `l` | Move to right panel |
| `Tab` | Move to next panel |

### Cursor Movement

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `g` | Go to first item |
| `G` | Go to last item |

## Selection & Actions

| Key | Action |
|-----|--------|
| `Enter` | Select/expand current item |
| `Space` | Fetch document data |
| `F` | Open query builder (collections/tree) |
| `r` | Refresh current view |

## Visual Select Mode (Tree Panel)

| Key | Action |
|-----|--------|
| `v` | Enter select mode |
| `j` | Extend selection down |
| `k` | Shrink selection up |
| `Space` | Fetch all selected documents |
| `Enter` | View fetched documents |
| `Esc` | Exit select mode |

## Filtering

| Key | Action |
|-----|--------|
| `/` | Start filter input |
| `Enter` | Apply filter |
| `Esc` | Cancel input / Clear filter |
| `Backspace` | Delete character |

## Details Panel

| Key | Action |
|-----|--------|
| `j` / `↓` | Scroll down |
| `k` / `↑` | Scroll up |
| `c` | Copy JSON to clipboard |
| `s` | Save JSON to file |
| `e` | Open in external editor ($EDITOR or vim) |
| `/` | Start filter/jq query |

## Query Builder

| Key | Action |
|-----|--------|
| `F` | Open query builder |
| `j` / `k` | Move between rows |
| `h` / `l` | Move between fields |
| `Enter` | Edit field / Execute |
| `a` | Add WHERE filter |
| `d` | Delete WHERE filter |
| `Esc` | Close query builder |

## Help Popup

| Key | Action |
|-----|--------|
| `?` | Close help |
| `Esc` | Close help |
| `j` / `k` | Scroll help content |

## Context-Sensitive Keys

Some keys behave differently based on context:

### In Filter Mode
- Letter keys insert characters
- `Enter` commits the filter
- `Esc` cancels filter input

### In Select Mode
- `j`/`k` extend/shrink selection
- `Esc` exits select mode (only in tree panel)

### In Details Panel
- `j`/`k` scroll content (not move cursor)
- `/` starts jq query mode

### In Query Builder
- `j`/`k` move between rows
- `h`/`l` move between fields
- `Enter` edits field or executes
- `a`/`d` add/delete filters

## Mouse Support

| Action | Effect |
|--------|--------|
| Click panel | Focus panel |
| Scroll | Scroll content |
| Click item | Select item |
