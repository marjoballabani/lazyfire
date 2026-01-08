# Navigation

LazyFire uses vim-style navigation throughout the interface.

## Panel Layout

```
┌─────────────┬────────────────────────────┐
│  Projects   │                            │
├─────────────┤         Details            │
│ Collections │                            │
├─────────────┤                            │
│    Tree     │                            │
└─────────────┴────────────────────────────┘
```

## Moving Between Panels

| Key | Action |
|-----|--------|
| `h` | Move to left panel |
| `l` | Move to right panel |
| `Tab` | Next panel |

## Moving Within a Panel

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `g` | Go to top |
| `G` | Go to bottom |

## Selection & Expansion

| Key | Action |
|-----|--------|
| `Enter` | Select/expand item |
| `Space` | Fetch document data |

## Panel-Specific Actions

### Projects Panel
- `Enter` - Select project and load its collections

### Collections Panel
- `Enter` - Select collection and load documents

### Tree Panel
- `Enter` - Expand document (show subcollections)
- `Space` - Fetch and view document data
- `v` - Enter [select mode](Select-Mode)

### Details Panel
- `j`/`k` - Scroll content
- `/` - Start [filter/jq query](Filtering)
- `c` - Copy JSON to clipboard
- `s` - Save JSON to file
- `e` - Open in external editor (uses `$EDITOR` or vim)

## Global Keys

| Key | Action |
|-----|--------|
| `q` | Quit |
| `?` | Show help |
| `@` | Show command log |
| `r` | Refresh current view |
| `Esc` | Cancel/go back |
