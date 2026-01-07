# 🔥 LazyFire

A terminal UI for browsing Firebase Firestore, inspired by [lazygit](https://github.com/jesseduffield/lazygit).

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Platform](https://img.shields.io/badge/Platform-macOS%20|%20Linux-lightgrey)

## Features

- 📂 Browse Firestore collections and documents
- 🌳 Expandable tree view for nested subcollections
- 📄 View document data as formatted JSON
- ⌨️ Vim-style keybindings (h/j/k/l)
- 🎨 Customizable theme (hex colors, 256-color, bold)
- 🔐 Uses existing Firebase CLI authentication
- 📐 Dynamic panel sizing (focused panel expands)

## Installation

### Homebrew (macOS/Linux)

```bash
brew install mballabani/tap/lazyfire
```

### From Source

```bash
git clone https://github.com/mballabani/lazyfire.git
cd lazyfire
go build -o lazyfire .
```

### Using Go Install

```bash
go install github.com/mballabani/lazyfire@latest
```

### Download Binary

Download pre-built binaries from the [releases page](https://github.com/mballabani/lazyfire/releases).

## Prerequisites

You must be authenticated with Firebase CLI:

```bash
firebase login
```

## Usage

```bash
./lazyfire
```

## Layout

```
╭─────────────╮╭──────────────────────────────────────╮
│  Projects   ││                                      │
│             ││                                      │
│   dev       ││           Details                    │
│ * prod      ││                                      │
│   staging   ││   {                                  │
├─────────────┤│     "name": "John",                  │
│ Collections ││     "email": "john@example.com",     │
│             ││     "age": 30                        │
│   users     ││   }                                  │
│ * orders    ││                                      │
│   products  ││                                      │
├─────────────┤├──────────────────────────────────────┤
│    Tree     ││  Commands                            │
│             ││  ✓ ListCollections → 3 collections   │
│  ├─ abc123  │╰──────────────────────────────────────╯
│  └─ def456  │
╰─────────────╯
╭────────────────────────────────────────────────────╮
│ ←/→ panels  j/k move  Space select  @ logs  q quit │
╰────────────────────────────────────────────────────╯
```

**Left side (stacked):**
- **Projects** - Your Firebase projects
- **Collections** - Root collections in selected project
- **Tree** - Documents and subcollections (expandable)

**Right side:**
- **Details** - Document JSON data
- **Commands** - API call status

## Keybindings

| Key | Action |
|-----|--------|
| `h` `←` | Move to left panel |
| `l` `→` | Move to right panel |
| `j` `↓` | Move down in list |
| `k` `↑` | Move up in list |
| `Space` | Select / Expand |
| `Esc` | Collapse node |
| `r` | Refresh |
| `@` | Show command history |
| `q` | Quit |

## Configuration

Create `~/.lazyfire/config.yaml`:

```yaml
ui:
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

- Go 1.21+
- Firebase CLI (`npm install -g firebase-tools`)
- Terminal with true color support (recommended)

## Contributing

Contributions welcome! Please open an issue or PR.

## License

MIT - see [LICENSE](LICENSE)

## Acknowledgments

- [lazygit](https://github.com/jesseduffield/lazygit) - UI inspiration
- [gocui](https://github.com/jesseduffield/gocui) - Terminal UI library
