# CLI Options

Command-line options for LazyFire.

## Usage

```bash
lazyfire [flags]
```

## Flags

### `--version`, `-v`

Show version information.

```bash
lazyfire --version
```

## Configuration

LazyFire loads configuration from `~/.config/lazyfire/config.yml`, falling back to `~/.lazyfire/config.yaml` and `./config.yaml`. It works without a config file.

See [Configuration](/guide/configuration) for details.

## Environment Variables

| Variable | Effect |
|----------|--------|
| `LAZYFIRE_CONFIG_FILE` | Use this config file instead of searching the usual locations |
| `XDG_CONFIG_HOME` | Look for `lazyfire/config.yml` here instead of `~/.config` |

## Authentication

LazyFire uses your existing Firebase CLI credentials. Make sure you're logged in:

```bash
firebase login
```

In [emulator mode](/guide/emulator-mode), authentication is skipped entirely.

## Examples

```bash
# Run LazyFire
lazyfire

# Check version
lazyfire -v
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success / Normal quit (q key) |
| 1 | Error |
