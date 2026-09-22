# LazyFire Wiki

LazyFire is a keyboard-driven terminal app for Firebase, inspired by lazygit. The full documentation, with a live demo, is at https://marjoballabani.github.io/lazyfire/

## Features

- **Projects and databases** - Switch between Firebase projects and each project's Firestore databases
- **Tree navigation** - Browse collections, documents, and subcollections
- **Query Builder** - Interactive Firestore query builder with WHERE, ORDER BY, LIMIT
- **Cloud Functions** - Function details and recent logs, filterable by level
- **Storage, Auth, Rules and Indexes** - Browse buckets and files, Auth users, security rules and composite indexes (read-only)
- **Visual select mode** - Select multiple documents and fetch them together
- **Filtering and jq queries** - Filter any panel as you type; run jq on documents
- **Document stats** - View Firestore limits compliance
- **Keys that follow the panel** - `?` lists what works where you are
- **Customizable theme** - Configure colors and icons

## Quick Start

```bash
# Install
brew install marjoballabani/tap/lazyfire
# or
go install github.com/marjoballabani/lazyfire@latest

# Run
lazyfire
```

## Pages

- [Installation](Installation)
- [Navigation](Navigation)
- [Databases](Databases)
- [Cloud Functions](Cloud-Functions)
- [Storage, Auth, Rules & Indexes](Storage-Auth-Rules)
- [Query Builder](Query-Builder)
- [Filtering & jq Queries](Filtering)
- [Visual Select Mode](Select-Mode)
- [Document Stats](Document-Stats)
- [Configuration](Configuration)
- [Keybindings](Keybindings)

## Requirements

- Firebase CLI (`firebase-tools`) installed and authenticated
- A Nerd Font (optional, for icons)
- Go 1.25+ only if you build from source
