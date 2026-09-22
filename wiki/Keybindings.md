# Keybindings Reference

Complete list of all keyboard shortcuts in LazyFire.

Keys depend on the focused panel and, in the collections panel, on the active tab. Press `?` anywhere to see the keys that apply right now. In that menu, `/` filters the list and `Enter` runs the selected key. The bar at the bottom of the screen shows the most useful keys for the current panel. If a key can't run right now, a short message says why.

## Global

| Key | Action |
|-----|--------|
| `q` / `Ctrl+c` | Quit |
| `?` | Keybindings menu |
| `@` | Command log |
| `1` / `2` / `3` / `4` | Focus Projects / Databases / Collections / Tree |
| `0` | Focus Details |
| `T` | Toggle human-readable timestamps |
| `x` | Export cached documents to ~/Downloads |
| `A` | Field type analysis (cached docs of the open collection) |
| `M` | Collection memory estimate |
| `i` | Cache statistics |
| `R` | Clear cache |

## Navigation

Works in every panel. In Details, and in the Rules and Indexes tabs, these keys scroll the text instead of moving through a list.

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `PgDn` / `PgUp` | Page down / up |
| `Ctrl+d` / `Ctrl+u` | Half page down / up |
| `g` / `Home` | Go to top |
| `G` / `End` | Go to bottom |
| `l` / `→` / `Tab` | Next left panel |
| `h` / `←` / `Shift+Tab` | Previous left panel |
| `[` / `]` | Previous / next tab |

## Filtering

| Key | Action |
|-----|--------|
| `/` | Filter the focused panel (Projects, Databases, Collections, Functions, Storage, Auth, Tree, Details) |
| `Enter` | Keep the filter |
| `Esc` | Cancel while typing, or clear a kept filter |
| `↑` / `↓` | Move through matches while typing |

The filter prompt is a normal text input: `Ctrl+a` / `Ctrl+e`, `Ctrl+w`, `Ctrl+u` and pasting work. In Details, a filter starting with `.` is a jq query.

## Projects Panel

| Key | Action |
|-----|--------|
| `Space` | Select project |
| `Enter` | Show project details |
| `S` | Scan collections health |
| `r` | Refresh projects |

## Databases Panel

Lists the Firestore databases of the selected project. Collections, documents, queries, the health scan and the Indexes tab all use the selected database. When the panel isn't focused it shows only the database in use.

| Key | Action |
|-----|--------|
| `Space` | Use database |
| `Enter` | Use database and focus collections |
| `r` | Refresh databases |

Datastore mode databases are listed but can't be browsed.

## Collections Panel

### Collections tab

| Key | Action |
|-----|--------|
| `Space` | Open collection |
| `Enter` | Open collection and focus the tree |
| `F` | Query builder on the highlighted collection |

### Functions tab

| Key | Action |
|-----|--------|
| `Space` | Select function |
| `Enter` | Open function details |
| `L` | Cycle log level filter |

### Storage tab

| Key | Action |
|-----|--------|
| `Space` / `Enter` | Open bucket / folder |
| `Esc` / `Backspace` | Go up one level |

### Auth tab

| Key | Action |
|-----|--------|
| `Space` / `Enter` | View user in details |

### Rules and Indexes tabs

| Key | Action |
|-----|--------|
| `Enter` | View in details |

`r` refreshes the active tab.

## Tree Panel

| Key | Action |
|-----|--------|
| `Space` | Expand / collapse |
| `Enter` | Open document in details / expand collection |
| `v` | Toggle select mode |
| `C` | Collapse all |
| `F` | Query builder (on a collection node, or the open collection) |
| `Q` | Clear query results |
| `c` | Copy document JSON |
| `s` | Save document JSON to ~/Downloads |
| `p` | Copy document path |
| `r` | Refresh documents (re-runs the query when showing query results) |

## Select Mode

| Key | Action |
|-----|--------|
| `j` / `k` | Extend selection |
| `Space` | Fetch selected documents |
| `Esc` / `v` | Exit select mode |

## Details Panel

The highlighted line is the cursor; `y` and `B` act on it.

| Key | Action |
|-----|--------|
| `J` / `K` | Down / up 5 lines |
| `Esc` / `Tab` | Back to the previous panel |
| `y` | Copy the value on the cursor line |
| `B` | Decode base64 on the cursor line |
| `n` / `N` | Next / previous filter match |
| `c` | Copy content (JSON, jq result, or scan report as Markdown) |
| `s` | Save content (JSON, jq result, or scan report as Markdown) |
| `p` | Copy document path |
| `t` | Toggle compact JSON |
| `w` | Toggle word wrap |
| `H` | Toggle line numbers |
| `D` | Field size breakdown |
| `e` | Open in $EDITOR |
| `r` | Reload document, or refresh function logs |
| `[` / `]` | Switch Details / Logs (functions) |

## Query Builder

| Key | Action |
|-----|--------|
| `j` / `k` | Move between rows |
| `h` / `l` | Move between fields |
| `Tab` | Next field |
| `Enter` | Edit field / run button |
| `a` | Add filter |
| `d` | Delete filter |
| `Esc` | Close query builder |

## Confirm Dialog

| Key | Action |
|-----|--------|
| `Enter` / `y` | Confirm |
| `Esc` / `n` | Cancel |

## Mouse

| Action | Effect |
|--------|--------|
| Click | Focus a panel and select the row |
| Double click | Open the row (same as `Enter`) |
| Wheel | Move the selection, or scroll Details |
| Click a tab title | Switch tab |
