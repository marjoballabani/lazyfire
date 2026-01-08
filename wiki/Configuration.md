# Configuration

LazyFire can be customized via a YAML configuration file.

## Config File Location

LazyFire looks for `config.yaml` in:
1. `~/.lazyfire/config.yaml` (recommended)
2. `./config.yaml` (current directory)

## Default Configuration

```yaml
ui:
  showIcons: true
  nerdFontsVersion: "3"  # "2", "3", or "" to disable
  theme:
    activeBorderColor:
      - "#ed8796"
      - bold
    inactiveBorderColor:
      - "#5f626b"
    optionsTextColor:
      - "#8aadf4"
    selectedLineBgColor:
      - "#494d64"
      - bold
```

## Configuration Options

### UI Settings

| Option | Type | Description |
|--------|------|-------------|
| `showIcons` | boolean | Enable/disable icons |
| `nerdFontsVersion` | string | "2", "3", or "" to disable |

### Theme Colors

Colors can be specified as:
- **Named colors**: `cyan`, `blue`, `red`, `green`, `yellow`, `magenta`, `white`, `black`, `default`
- **Hex colors**: `#ed8796`, `#ff0000`
- **256-color numbers**: `0` to `255`
- **Attributes**: `bold`, `underline`, `reverse`

| Option | Description |
|--------|-------------|
| `activeBorderColor` | Focused panel border color |
| `inactiveBorderColor` | Unfocused panel border color |
| `optionsTextColor` | Help text color in footer |
| `selectedLineBgColor` | Highlighted row background |

## Theme Examples

### Catppuccin Macchiato

```yaml
ui:
  theme:
    activeBorderColor:
      - "#ed8796"
      - bold
    inactiveBorderColor:
      - "#5f626b"
    optionsTextColor:
      - "#8aadf4"
    selectedLineBgColor:
      - "#494d64"
```

### Dracula

```yaml
ui:
  theme:
    activeBorderColor:
      - "#bd93f9"
      - bold
    inactiveBorderColor:
      - "#6272a4"
    optionsTextColor:
      - "#8be9fd"
    selectedLineBgColor:
      - "#44475a"
```

### Nord

```yaml
ui:
  theme:
    activeBorderColor:
      - "#88c0d0"
      - bold
    inactiveBorderColor:
      - "#4c566a"
    optionsTextColor:
      - "#81a1c1"
    selectedLineBgColor:
      - "#3b4252"
```

### Simple (No Colors)

```yaml
ui:
  showIcons: false
  theme:
    activeBorderColor:
      - default
      - bold
    inactiveBorderColor:
      - default
    optionsTextColor:
      - default
    selectedLineBgColor:
      - reverse
```

## Disabling Icons

If you don't have a Nerd Font installed:

```yaml
ui:
  showIcons: false
```

Or use Nerd Fonts v2:

```yaml
ui:
  nerdFontsVersion: "2"
```
