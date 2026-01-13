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
   - Use `h`/`l` or `Tab` to switch panels
   - Press `Space` to select items
   - Press `?` for help

## Interface Overview

LazyFire uses a four-panel layout:

```
┌─ Projects ──┬─ Collections ──┬─ Tree ──────┬─ Details ────┐
│             │                │             │              │
│ Your        │ Collections    │ Documents   │ Document     │
│ Firebase    │ or Functions   │ in the      │ data in      │
│ projects    │ in project     │ collection  │ JSON format  │
│             │                │             │              │
└─────────────┴────────────────┴─────────────┴──────────────┘
```

## Next Steps

- [Installation](/guide/installation) - Detailed installation options
- [Navigation](/guide/navigation) - Learn the keybindings
- [Cloud Functions](/guide/cloud-functions) - Monitor your functions
