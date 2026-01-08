# 🔥 LazyFire

A terminal UI for browsing Firebase Firestore, inspired by [lazygit](https://github.com/jesseduffield/lazygit).

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Platform](https://img.shields.io/badge/Platform-macOS%20|%20Linux-lightgrey)

## Features

- Browse Firestore collections and documents
- Expandable tree view for nested subcollections
- View document data as syntax-highlighted JSON
- Filter/search across all panels
- **jq query support** for filtering JSON in details panel
- Vim-style keybindings (h/j/k/l)
- Mouse support (click to select, navigate)
- Customizable theme (hex colors, 256-color, bold)
- Nerd Font icons (optional, with graceful fallback)
- Uses existing Firebase CLI authentication
- Dynamic panel sizing (focused panel expands)
- Copy/save document JSON to clipboard or file

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap marjoballabani/tap
brew install lazyfire
```

### Using Go Install

```bash
go install github.com/marjoballabani/lazyfire@latest
```

### From Source

```bash
git clone https://github.com/marjoballabani/lazyfire.git
cd lazyfire
go build -o lazyfire .
```

### Download Binary

Download pre-built binaries from the [releases page](https://github.com/marjoballabani/lazyfire/releases).

## Quick Start

1. **Login to Firebase** (if not already):
   ```bash
   firebase login
   ```

2. **Run LazyFire:**
   ```bash
   lazyfire
   ```

3. **Navigate:** Use arrow keys or `h/j/k/l` to browse your Firestore data.

## Preview

![LazyFire Preview](assets/preview.gif)

**Panels:**
- **Projects** - Your Firebase projects
- **Collections** - Root collections in selected project
- **Tree** - Documents and subcollections (expandable)
- **Details** - Document JSON data with syntax highlighting
- **Commands** - API call status

## Keybindings

| Key | Action |
|-----|--------|
| `h` `←` | Move to left panel |
| `l` `→` | Move to right panel |
| `j` `↓` | Move down in list |
| `k` `↑` | Move up in list |
| `Tab` | Go to details panel |
| `Enter` | Open document in details / Fetch project info |
| `Space` | Select / Expand / Collapse |
| `/` | Filter current panel |
| `c` | Copy JSON to clipboard (respects jq filter) |
| `s` | Save JSON to ~/Downloads (respects jq filter) |
| `Esc` | Back: close popup / cancel filter / clear filter / exit details |
| `r` | Refresh |
| `?` | Show keyboard shortcuts |
| `@` | Show command history |
| `q` | Quit |

### Mouse

- **Click** on any panel to focus and select item
- **Click** outside popup to close it

## Filtering & jq Queries

Press `/` to filter any panel. In the **details panel**, you can use jq queries:

| Filter | Description |
|--------|-------------|
| `name` | Text search - shows lines containing "name" |
| `.name` | jq query - extracts the `name` field |
| `.users[0]` | jq query - first element of users array |
| `.data \| keys` | jq query - list keys of data object |

When a jq filter is active, `c` and `s` copy/save the filtered result instead of the full document.

## Configuration

Create `~/.lazyfire/config.yaml`:

```yaml
ui:
  # Icons: "3" (Nerd Fonts v3), "2" (v2), or "" (disable)
  nerdFontsVersion: "3"

  theme:
    activeBorderColor:
      - cyan
    inactiveBorderColor:
      - default
    optionsTextColor:
      - cyan
    selectedLineBgColor:
      - blue
```

### Icons

LazyFire uses [Nerd Fonts](https://www.nerdfonts.com/) icons by default. If icons don't display correctly:

```yaml
# Use Nerd Fonts v2 (older version)
ui:
  nerdFontsVersion: "2"

# Or disable icons entirely
ui:
  nerdFontsVersion: ""
```

### Color Options

- **Named colors:** `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`, `default`
- **Hex colors:** `#ed8796`, `#ff79c6`
- **256-color:** `0` - `255`
- **Attributes:** `bold`, `underline`, `reverse`

### Example Themes

**Catppuccin Macchiato:**
```yaml
ui:
  theme:
    activeBorderColor: ["#ed8796", "bold"]
    inactiveBorderColor: ["#5f626b"]
    optionsTextColor: ["#8aadf4"]
    selectedLineBgColor: ["#494d64"]
```

**Dracula:**
```yaml
ui:
  theme:
    activeBorderColor: ["#ff79c6", "bold"]
    inactiveBorderColor: ["#6272a4"]
    optionsTextColor: ["#8be9fd"]
    selectedLineBgColor: ["#44475a"]
```

## Requirements

- Firebase CLI (`npm install -g firebase-tools`)
- Terminal with true color support (recommended)
- [Nerd Font](https://www.nerdfonts.com/) for icons (optional)
- Go 1.21+ (only if building from source)

## Contributing

Contributions welcome! Please open an issue or PR.

## License

MIT - see [LICENSE](LICENSE)

## Acknowledgments

- [lazygit](https://github.com/jesseduffield/lazygit) - UI inspiration
- [gocui](https://github.com/jesseduffield/gocui) - Terminal UI library
