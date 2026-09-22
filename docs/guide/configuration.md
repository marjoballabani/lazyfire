# Configuration

LazyFire works without a config file. To change colors, icons or emulator settings, create a YAML file with just the settings you want to change. Everything you leave out keeps its [default](#default-values).

## Config File Location

LazyFire uses the first file it finds:

1. The file in the `LAZYFIRE_CONFIG_FILE` environment variable
2. `~/.config/lazyfire/config.yml` (recommended), or `$XDG_CONFIG_HOME/lazyfire/config.yml` if that variable is set
3. `~/.lazyfire/config.yaml`
4. `config.yaml` in the directory you start LazyFire from

Both `.yml` and `.yaml` work. If `LAZYFIRE_CONFIG_FILE` points to a file that doesn't exist or isn't valid YAML, LazyFire stops with an error instead of silently using the defaults.

```bash
mkdir -p ~/.config/lazyfire
curl -o ~/.config/lazyfire/config.yml \
  https://raw.githubusercontent.com/marjoballabani/lazyfire/main/config.example.yaml
```

The [example config](https://github.com/marjoballabani/lazyfire/blob/main/config.example.yaml) lists every setting with its default value.

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
