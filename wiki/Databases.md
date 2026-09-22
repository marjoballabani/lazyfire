# Databases

A Firestore project can have more than one database: the `(default)` one and any named databases you created. The **Databases** panel, under Projects, lists them and sets which one LazyFire reads from.

```
╭─ Databases ──────────────────────────╮
│ * (default) (nam5)                   │
│   analytics (eur3)                   │
│   legacy (us-east1, datastore mode)  │
╰───────────────────────────────1 of 3─╯
```

The `*` marks the database in use. When the panel isn't focused it shrinks to one line showing just that database.

## Switching databases

1. Press `2` to focus the Databases panel (or move there with `Tab`)
2. Use `j`/`k` to pick a database
3. Press `Space` to use it, or `Enter` to use it and jump to its collections

The database list loads the first time you open the panel after selecting a project. Press `r` to reload it and `/` to filter it by ID or location.

## What follows the database

These all read from the selected database:

- Collections, documents and subcollections
- The query builder
- The collection health scan
- Document stats, including composite index detection
- The Indexes tab

Functions, Storage and Auth belong to the whole project, so they don't change. The Rules tab shows the project's newest ruleset.

Switching to another database clears the tree, the open document, cached documents and filters, since the same document paths can exist in several databases. Selecting a project again goes back to its `(default)` database.

A named database also appears in the breadcrumb at the bottom right, for example `my-app > analytics > users`.

## Limitations

- Datastore mode databases are listed but can't be browsed. Pressing `Space` on one explains why.
- If your account isn't allowed to list databases, the error goes to the command log (`@`) and `(default)` is still offered.
- In [emulator mode](https://marjoballabani.github.io/lazyfire/guide/emulator-mode) only `(default)` is listed.
