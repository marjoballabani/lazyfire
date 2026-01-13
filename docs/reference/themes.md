# Themes

LazyFire supports custom color themes via configuration.

## Configuring Colors

Edit `~/.lazyfire/config.yaml`:

```yaml
ui:
  theme:
    activeBorderColor: ["#ed8796", "bold"]
    inactiveBorderColor: ["#5f626b"]
    optionsTextColor: ["#8aadf4"]
    selectedLineBgColor: ["#494d64", "bold"]
```

## Color Format

Colors can be specified as:

| Format | Example | Description |
|--------|---------|-------------|
| Named | `cyan`, `red` | Basic terminal colors |
| Hex | `#ed8796` | RGB hex colors |
| 256-color | `125` | 256-color palette index |

### Named Colors

`black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`, `default`

### Attributes

Add attributes after the color:
- `bold`
- `underline`
- `reverse`

Example: `["cyan", "bold"]`

## Theme Properties

| Property | Description |
|----------|-------------|
| `activeBorderColor` | Border and title of focused panel |
| `inactiveBorderColor` | Border of unfocused panels |
| `optionsTextColor` | Help text in the footer |
| `selectedLineBgColor` | Background of highlighted row |

## Example Themes

### Default (Catppuccin Macchiato)

```yaml
ui:
  theme:
    activeBorderColor: ["#ed8796", "bold"]
    inactiveBorderColor: ["#5f626b"]
    optionsTextColor: ["#8aadf4"]
    selectedLineBgColor: ["#494d64", "bold"]
```

### Nord

```yaml
ui:
  theme:
    activeBorderColor: ["#88c0d0", "bold"]
    inactiveBorderColor: ["#4c566a"]
    optionsTextColor: ["#81a1c1"]
    selectedLineBgColor: ["#3b4252", "bold"]
```

### Dracula

```yaml
ui:
  theme:
    activeBorderColor: ["#bd93f9", "bold"]
    inactiveBorderColor: ["#6272a4"]
    optionsTextColor: ["#8be9fd"]
    selectedLineBgColor: ["#44475a", "bold"]
```

### Gruvbox

```yaml
ui:
  theme:
    activeBorderColor: ["#fabd2f", "bold"]
    inactiveBorderColor: ["#665c54"]
    optionsTextColor: ["#83a598"]
    selectedLineBgColor: ["#3c3836", "bold"]
```

### Tokyo Night

```yaml
ui:
  theme:
    activeBorderColor: ["#7aa2f7", "bold"]
    inactiveBorderColor: ["#565f89"]
    optionsTextColor: ["#bb9af7"]
    selectedLineBgColor: ["#292e42", "bold"]
```

### Monokai

```yaml
ui:
  theme:
    activeBorderColor: ["#f92672", "bold"]
    inactiveBorderColor: ["#75715e"]
    optionsTextColor: ["#66d9ef"]
    selectedLineBgColor: ["#3e3d32", "bold"]
```

## UI Indicators

Throughout the interface, colors indicate status:

| Color | Meaning |
|-------|---------|
| Green | Active, success, INFO |
| Yellow | Pending, warning, DEPLOYING |
| Red | Error, offline, failed |
| Cyan | Selected, highlighted |
| Gray | Disabled, secondary info |
