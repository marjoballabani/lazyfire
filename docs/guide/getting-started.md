# Getting Started

LazyFire is a terminal-based Firebase browser that lets you navigate your Firestore database, view Cloud Functions, and monitor logs without leaving your terminal.

## Prerequisites

- A Firebase project with Firestore enabled
- Firebase CLI installed and authenticated (`firebase login`)

## Quick Start

1. **Install LazyFire**

   ```bash
   # Homebrew (recommended)
   brew install marjoballabani/tap/lazyfire

   # Or via Go
   go install github.com/marjoballabani/lazyfire@latest
   ```

2. **Run it**

   ```bash
   lazyfire
   ```

3. **Navigate**
   - Use `j`/`k` to move up/down
   - Use `Tab` or `h`/`l` to switch panels, `0` to jump to Details
   - Press `Space` to select, `Enter` to open
   - Press `?` to see the keys for the panel you are in

## Interface Overview

The left side has four stacked panels, and the right side shows details:

```
╭─ Projects ────────╮╭─ Details ─────────────────────────╮
│ Your Firebase     ││ Document JSON, function details   │
│ projects          ││ and logs, storage and user info   │
╰───────────────────╯│                                   │
╭─ Databases ───────╮│                                   │
│ Firestore DBs     ││                                   │
╰───────────────────╯│                                   │
╭─ Collections ─────╮│                                   │
│ Collections and   │╰───────────────────────────────────╯
│ other tabs        │╭─ Commands ────────────────────────╮
╰───────────────────╯│ Status of the last API call       │
╭─ Tree ────────────╮╰───────────────────────────────────╯
│ Documents         │
╰───────────────────╯
 The bar at the bottom shows the keys for the focused panel
```

The collections panel has tabs for Collections, Functions, Storage, Auth, Rules and Indexes. See [Navigation](/guide/navigation) for how to move around.

## Next Steps

- [Installation](/guide/installation) - Detailed installation options
- [Navigation](/guide/navigation) - Learn the keybindings
- [Cloud Functions](/guide/cloud-functions) - Monitor your functions
