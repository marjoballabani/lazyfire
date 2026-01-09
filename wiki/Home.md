# LazyFire Wiki

Welcome to the LazyFire wiki! LazyFire is a terminal UI for browsing Firebase Firestore databases.

## Features

- **Multi-project support** - Switch between Firebase projects
- **Tree navigation** - Browse collections, documents, and subcollections
- **Query Builder** - Interactive Firestore query builder with WHERE, ORDER BY, LIMIT
- **Visual select mode** - Select multiple documents for batch operations
- **Smart caching** - Documents and collections cached with visual indicator
- **jq query support** - Filter JSON with jq syntax
- **Document stats** - View Firestore limits compliance
- **Customizable theme** - Configure colors and icons

## Quick Start

```bash
# Install
go install github.com/marjoballabani/lazyfire@latest

# Run
lazyfire
```

## Pages

- [Installation](Installation)
- [Navigation](Navigation)
- [Query Builder](Query-Builder)
- [Filtering & jq Queries](Filtering)
- [Visual Select Mode](Select-Mode)
- [Document Stats](Document-Stats)
- [Configuration](Configuration)
- [Keybindings](Keybindings)

## Requirements

- Go 1.21+
- Firebase CLI (`firebase-tools`) installed and authenticated
- A Nerd Font (optional, for icons)
