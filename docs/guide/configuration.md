# Configuration

LazyFire can be configured via a YAML configuration file.

## Config File Location

LazyFire searches for `config.yaml` in:
1. `~/.lazyfire/config.yaml` (recommended)
2. `./config.yaml` (current directory)

## Configuration Options

```yaml
ui:
  showIcons: true           # Enable/disable icons
  nerdFontsVersion: "3"     # "2", "3", or "" to disable
  theme:
    activeBorderColor: ["#ed8796", "bold"]
    inactiveBorderColor: ["#5f626b"]
    optionsTextColor: ["#8aadf4"]
    selectedLineBgColor: ["#494d64", "bold"]
```

### UI Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `showIcons` | bool | `true` | Show icons in the UI |
| `nerdFontsVersion` | string | `"3"` | Nerd Fonts version ("2", "3", or "" to disable) |

### Theme Colors

Colors can be specified as:
- Named colors: `cyan`, `blue`, `red`, `green`, `yellow`, `magenta`, `white`, `black`, `default`
- Hex colors: `#ed8796`
- 256-color numbers: `0` to `255`
- Attributes: `bold`, `underline`, `reverse`

| Option | Description |
|--------|-------------|
| `activeBorderColor` | Focused panel border and title |
| `inactiveBorderColor` | Unfocused panel borders |
| `optionsTextColor` | Help text in footer |
| `selectedLineBgColor` | Highlighted row background |

## Example Configurations

### Minimal (Disable Icons)

```yaml
ui:
  showIcons: false
  nerdFontsVersion: ""
```

### Custom Theme

```yaml
ui:
  theme:
    activeBorderColor: ["cyan", "bold"]
    inactiveBorderColor: ["#444444"]
    optionsTextColor: ["green"]
    selectedLineBgColor: ["#333333"]
```

### Catppuccin-style

```yaml
ui:
  theme:
    activeBorderColor: ["#f5c2e7", "bold"]
    inactiveBorderColor: ["#6c7086"]
    optionsTextColor: ["#89b4fa"]
    selectedLineBgColor: ["#313244", "bold"]
```

## Emulator Mode

Connect to a local Firebase Emulator instead of production:

```yaml
emulator:
  enabled: true
  projectId: "my-local-project"
  firestoreHost: "localhost:8080"
```

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `enabled` | yes | `false` | Enable emulator mode |
| `projectId` | yes | - | Project ID (can be any string) |
| `firestoreHost` | no | `localhost:8080` | Emulator host and port |

See [Emulator Mode](/guide/emulator-mode) for details.

## Default Values

If no config file exists, LazyFire uses these defaults:

```yaml
ui:
  showIcons: true
  nerdFontsVersion: "3"
  theme:
    activeBorderColor: ["#ed8796", "bold"]
    inactiveBorderColor: ["#5f626b"]
    optionsTextColor: ["#8aadf4"]
    selectedLineBgColor: ["#494d64", "bold"]
```
