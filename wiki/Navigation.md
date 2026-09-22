# Navigation

LazyFire is driven by the keyboard, with vim-style keys. Keys depend on the focused panel, so press `?` whenever you want to see what works right now.

## Panel layout

The left side has four stacked panels. Projects and Databases shrink to one line when you are not in them. Details and Commands are on the right, and the bar at the bottom shows the keys for the focused panel.

```
╭─ Projects ─────────╮╭─ Details ──────────────────────────╮
│ * my-app-prod      ││ ─── users/u_alice ───              │
╰────────────────────╯│ Size: 312 B / 1MB  Depth: 1 / 20   │
╭─ Databases ────────╮│                                    │
│ * (default)        ││ {                                  │
╰────────────────────╯│   "name": "Alice",                 │
╭─ Collections ──────╮│   "plan": "pro"                    │
│ * users            ││ }                                  │
│   orders           │╰────────────────────────────────────╯
╰────────────────────╯╭─ Commands ─────────────────────────╮
╭─ Tree ─────────────╮│ ✓ api ListDocuments(users) → 3 docs│
│ * u_alice          │╰────────────────────────────────────╯
╰────────────────────╯
 space expand  enter open  / filter  F query  tab panels  ? help
```

## Moving between panels

| Key | Action |
|-----|--------|
| `Tab` / `l` / `→` | Next left panel |
| `Shift+Tab` / `h` / `←` | Previous left panel |
| `1` / `2` / `3` / `4` | Focus Projects / Databases / Collections / Tree |
| `0` | Focus Details |
| `Esc` or `Tab` in Details | Go back to the panel you came from |

`Enter` on an item also moves you along: it opens a database's collections, a collection's documents, or a document in Details.

## Moving within a panel

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `g` / `Home` | Go to top |
| `G` / `End` | Go to bottom |
| `PgDn` / `PgUp` | Page down / up |
| `Ctrl+d` / `Ctrl+u` | Half page down / up |
| `J` / `K` | Down / up 5 lines (Details) |

In Details these keys move a highlighted cursor line. `y` copies the value on that line and `B` decodes it as base64.

## Selecting and opening

| Key | Action |
|-----|--------|
| `Space` | Select a project or database, open a collection, expand a document |
| `Enter` | Open the item and move to the next panel |
| `v` | Toggle select mode in the tree (see [Visual Select Mode](Select-Mode)) |

## Tabs

The collections panel has six tabs: Collections, Functions, Storage, Auth, Rules and Indexes. Press `[` or `]` to switch between them. Only three tab titles fit at once, so arrows in the title show that more tabs are hidden.

When you look at a function in Details, `[` and `]` switch between its Details and Logs tabs.

## Tree

| Key | Action |
|-----|--------|
| `Space` | Expand or collapse a document's subcollections, or a subcollection's documents |
| `Enter` | Open a document in Details, or expand a subcollection |
| `C` | Collapse everything |
| `Q` | Go back from query results to the collection's documents |

## Keybindings menu

Press `?` to open the list of keys for the focused panel. It is split into the panel's own actions, navigation and global keys.

- Type `/` to filter the list, then `Enter` to run the selected key.
- Keys that can't run right now are dimmed, with the reason next to them.
- `Esc`, `q` or `?` closes the menu.

When you press a key that can't run, the reason also shows at the bottom right of the screen, for example "No document open".

## Mouse

| Action | Effect |
|--------|--------|
| Click | Focus a panel and select the row |
| Double click | Open the row, same as `Enter` |
| Wheel | Move the selection, or scroll Details |
| Click a tab title | Switch to that tab |

## Other keys

| Key | Action |
|-----|--------|
| `/` | Filter the focused panel (see [Filtering & Search](Filtering)) |
| `r` | Refresh the focused panel |
| `@` | Command log |
| `q` / `Ctrl+c` | Quit |

The full list is in the [keybindings reference](Keybindings).
