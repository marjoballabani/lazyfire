# Cloud Functions

LazyFire lets you browse Cloud Functions and view their logs in real-time.

## Accessing Functions

1. Select a project in the **Projects** panel
2. Navigate to the **Collections** panel
3. Press `]` to switch to the **Functions** tab

```
┌─ [Collections] [Functions] ─┐
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

Press `Space` on a function to view its details:

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

## Live Logs

View function execution logs in real-time:

1. Select a function with `Space`
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

- Logs auto-refresh every 3 seconds when viewing
- Press `r` to manually refresh

## Keybindings Summary

| Key | Action |
|-----|--------|
| `]` | Switch to Functions tab (Collections panel) |
| `[` | Switch to Collections tab |
| `Space` | Select function, show details |
| `]` | Switch to Logs tab (Details panel) |
| `[` | Switch to Details tab |
| `r` | Refresh logs |
| `/` | Filter functions by name |

## State Preservation

LazyFire maintains separate states for Collections and Functions:

- Switching tabs preserves your selections
- Selected function and logs are remembered
- Document selections remain independent
