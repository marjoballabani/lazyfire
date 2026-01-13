# Visual Select Mode

Visual select mode allows you to select multiple documents in the tree panel for batch operations.

## Entering Select Mode

1. Focus the **Tree panel**
2. Press `v` to enter select mode
3. The current document is automatically selected

## Selection Behavior

| Key | Action |
|-----|--------|
| `v` | Enter select mode |
| `j` / `↓` | Move down and extend selection |
| `k` / `↑` | Move up and shrink selection |
| `Space` | Fetch all selected documents (parallel) |
| `Enter` | View fetched documents in details |
| `Esc` | Exit select mode (only in tree panel) |

## Visual Indicators

- Selected items show a `+` marker with yellow background
- Current position is highlighted
- Selection persists when viewing details

## Workflow Example

### Fetching Multiple Documents

1. Navigate to Tree panel
2. Press `v` to start selection
3. Press `j` multiple times to select documents below
4. Press `Space` to fetch all selected documents in parallel
5. Press `Enter` to view each document's data

### Comparing Documents

1. Select multiple documents with `v` and `j`/`k`
2. Press `Space` to fetch them
3. Use `Enter` to cycle through and view each document
4. Selection stays active for quick switching

## Parallel Fetching

When you press `Space` with multiple documents selected:
- All documents are fetched in parallel
- Progress is shown in the command log
- Faster than fetching one by one

## Tips

- Selection only works in the Tree panel
- Moving to Details panel keeps your selection
- Press `Esc` in Tree panel to clear selection
- Press `Esc` in Details panel returns to Tree (selection preserved)
