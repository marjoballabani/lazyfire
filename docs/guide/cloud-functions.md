# Cloud Functions

LazyFire lets you browse Cloud Functions and read their recent logs.

## Accessing Functions

1. Select a project in the **Projects** panel
2. Move to the **Collections** panel (`3`)
3. Press `]` to switch to the **Functions** tab

```
┌─ Collections - Functions ───┐
│ ⚡ processOrder             │
│ ⚡ sendEmail                │
│ ⚡ onUserCreate             │
│ ⚡ scheduledCleanup         │
└─────────────────────────────┘
```

## Function List

Each function displays:
- Function name
- Status indicator (color-coded)

### Status Colors

| Color | Status |
|-------|--------|
| Green | ACTIVE |
| Yellow | DEPLOYING |
| Red | OFFLINE / DELETE_IN_PROGRESS |

## Function Details

Press `Space` on a function to select it and load its details, or `Enter` to also move into the Details panel:

```
┌─ Function Details ──────────┐
│ Name:    processOrder       │
│ Status:  ACTIVE             │
│ Runtime: nodejs18           │
│ Region:  us-central1        │
│ Memory:  256MB              │
│ Timeout: 60s                │
│ Trigger: HTTP               │
│ URL:     https://...        │
└─────────────────────────────┘
```

## Logs

View the latest 50 log entries of a function:

1. Open a function with `Enter`
2. Press `]` to switch to the **Logs** tab

```
┌─ [Details] [Logs] ──────────┐
│ 12:34:56 INFO  Starting...  │
│ 12:34:57 INFO  Processing   │
│ 12:34:58 ERROR Failed: ...  │
│ 12:34:59 INFO  Retrying...  │
└─────────────────────────────┘
```

### Log Severity Colors

| Color | Severity |
|-------|----------|
| Green | INFO |
| Yellow | WARNING |
| Red | ERROR |
| Gray | DEBUG |

### Refreshing Logs

Press `r` on the Logs tab to load the latest entries.

### Filtering by Severity

Press `L` to cycle the log level filter: all levels, then ERROR, WARNING, INFO and DEBUG. The active filter is shown above the log lines.

## Keybindings Summary

| Key | Action |
|-----|--------|
| `[` / `]` | Switch collections panel tabs (Functions is the second) |
| `Space` | Select function |
| `Enter` | Select function and focus Details |
| `[` / `]` | Switch between Details and Logs (in Details) |
| `r` | Refresh functions, or logs on the Logs tab |
| `L` | Cycle log level filter |
| `/` | Filter functions by name or region |

## State Preservation

LazyFire maintains separate states for Collections and Functions:

- Switching tabs preserves your selections
- Selected function and logs are remembered
- Document selections remain independent
