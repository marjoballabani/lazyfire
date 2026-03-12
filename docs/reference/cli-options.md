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

LazyFire loads configuration from `~/.lazyfire/config.yaml`.

See [Configuration](/guide/configuration) for details.

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
