# LazyFire

A terminal UI for browsing Firebase Firestore, inspired by [lazygit](https://github.com/jesseduffield/lazygit).

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Platform](https://img.shields.io/badge/Platform-macOS%20|%20Linux-lightgrey)

## Features

- Browse Firestore collections and documents in a terminal UI
- Expandable tree view for nested subcollections
- View document data as formatted JSON
- Vim-style keybindings (h/j/k/l navigation)
- Customizable theme with hex color support
- Uses existing Firebase/gcloud authentication
- Dynamic panel sizing (focused panel expands)

## Installation

### Using Go

```bash
go install github.com/mballabani/lazyfire@latest
```

### Building from Source

```bash
git clone https://github.com/mballabani/lazyfire.git
cd lazyfire
go build -o lazyfire .
```

## Prerequisites

LazyFire requires authentication with Firebase or Google Cloud:

```bash
# Option 1: Firebase CLI (recommended)
firebase login

# Option 2: Google Cloud SDK
gcloud auth application-default login
```

## Usage

```bash
lazyfire
```

### Keybindings

| Key | Action |
|-----|--------|
| `h` / `←` | Move to left panel |
| `l` / `→` | Move to right panel |
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Space` | Select project/collection, expand tree node |
| `Esc` | Collapse tree node |
| `r` | Refresh |
| `@` | Show command history |
| `q` | Quit |

### Panels

```
┌─────────┬──────────┬─────────────┬─────────────────────┐
│Projects │Collections│ Tree        │ Details             │
│         │          │             │                     │
│  dev    │  users   │ └─ users    │  {                  │
│* prod   │* orders  │   ├─abc123  │    "name": "John",  │
│  stage  │  products│   └─def456  │    "email": "j@x"   │
│         │          │             │  }                  │
├─────────┴──────────┴─────────────┴─────────────────────┤
│ Commands                                               │
├────────────────────────────────────────────────────────┤
│ ←/→ panels  j/k move  Space select  @ history  q quit  │
└────────────────────────────────────────────────────────┘
```

## Configuration

Create `~/.lazyfire/config.yaml`:

```yaml
ui:
  theme:
    # Supports: named colors, hex (#ed8796), 256-color (0-255)
    # Attributes: bold, underline, reverse
    activeBorderColor:
      - cyan
    inactiveBorderColor:
      - default
    optionsTextColor:
      - cyan
    selectedLineBgColor:
      - blue
```

### Theme Examples

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
- Firebase CLI or Google Cloud SDK (for authentication)
- Terminal with true color support (recommended)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- [lazygit](https://github.com/jesseduffield/lazygit) - Inspiration for the UI design
- [gocui](https://github.com/jesseduffield/gocui) - Terminal UI library
