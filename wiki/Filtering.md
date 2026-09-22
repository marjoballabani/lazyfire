# Filtering & Search

Press `/` in a panel to filter it. The list narrows as you type.

```
╭─ Collections ──────────────╮
│ * users                    │
│   user_settings            │
╰───────────────2/15 matched─╯
 Filter Collections: user█
```

## How filtering works

- Matching is case-insensitive and finds the text anywhere in the name.
- The prompt at the bottom is a normal text input: move with `←`/`→`, delete a word with `Ctrl+w`, clear with `Ctrl+u`, and paste.
- `↑` and `↓` move through the matches while you type.
- `Enter` keeps the filter and closes the prompt. The panel border turns yellow while a filter is kept.
- `Esc` while typing cancels. `Esc` later clears a kept filter, and the item you had selected stays selected.

## What each panel filters

| Panel | Matches on |
|-------|------------|
| Projects | Project name and ID |
| Databases | Database ID and location |
| Collections tab | Collection name |
| Functions tab | Function name and region |
| Storage tab | Bucket or file name in the current folder |
| Auth tab | Email, UID and display name |
| Tree | Document ID and path |
| Details | Lines of the open document, or a jq query |

Each panel keeps its own filter, so switching panels doesn't lose it.

## Filtering a document

In Details, the filter shows only the lines of the JSON that contain your text, with the matches highlighted. `n` and `N` move the cursor to the next and previous match.

Start the filter with `.` to run a [jq](https://jqlang.github.io/jq/) query on the document instead:

| Filter | Result |
|--------|--------|
| `name` | Lines containing "name" |
| `.name` | The `name` field |
| `.address.city` | A nested field |
| `.tags[0]` | The first element of an array |
| `.items \| length` | The number of items |

`c` and `s` copy or save the jq result instead of the whole document while a jq filter is active.

## Keys

| Key | Action |
|-----|--------|
| `/` | Start filtering the focused panel |
| `Enter` | Keep the filter |
| `Esc` | Cancel while typing, or clear a kept filter |
| `↑` / `↓` | Move through matches while typing |
| `n` / `N` | Next / previous match in Details |
