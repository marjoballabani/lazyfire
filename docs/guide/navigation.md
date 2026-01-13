# Navigation

LazyFire uses vim-style keybindings for efficient keyboard navigation.

## Panel Layout

```
┌─ Projects ──┬─ Collections ──┬─ Tree ──────┬─ Details ────┐
│             │                │             │              │
│  Panel 1    │    Panel 2     │   Panel 3   │   Panel 4    │
│             │                │             │              │
└─────────────┴────────────────┴─────────────┴──────────────┘
```

## Moving Between Panels

| Key | Action |
|-----|--------|
| `Tab` | Move to next panel |
| `Shift+Tab` | Move to previous panel |
| `h` | Move left |
| `l` | Move right |
| `1-4` | Jump to panel by number |

## Moving Within Lists

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `gg` | Jump to top |
| `G` | Jump to bottom |
| `Ctrl+d` | Page down |
| `Ctrl+u` | Page up |

## Selection

| Key | Action |
|-----|--------|
| `Space` | Select item / expand |
| `Enter` | Open / view details |
| `v` | Toggle select mode |
| `V` | Select all visible |

## Tab Switching

The Collections and Details panels support tabs:

### Collections Panel
| Key | Action |
|-----|--------|
| `[` | Switch to Collections tab |
| `]` | Switch to Functions tab |

### Details Panel (when viewing Functions)
| Key | Action |
|-----|--------|
| `[` | Switch to Details tab |
| `]` | Switch to Logs tab |

## Tree Navigation

When browsing documents with subcollections:

| Key | Action |
|-----|--------|
| `Space` | Expand/collapse subcollection |
| `h` | Go to parent |
| `l` | Enter subcollection |
| `Backspace` | Go back in history |

## Quick Actions

| Key | Action |
|-----|--------|
| `?` | Show help popup |
| `q` | Quit |
| `/` | Start filtering |
| `Esc` | Cancel / close popup |
| `r` | Refresh current view |
