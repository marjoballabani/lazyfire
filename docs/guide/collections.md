# Collections & Documents

LazyFire provides a visual browser for your Firestore collections and documents.

## Browsing Collections

1. Select a project from the **Projects** panel
2. Collections appear in the **Collections** panel
3. Use `j`/`k` to navigate, `Space` to select

```
┌─ Collections ───────┐
│ 📁 users           │  ← Root collections
│ 📁 orders          │
│ 📁 products        │
│ 📁 analytics       │
└─────────────────────┘
```

## Viewing Documents

Select a collection to see its documents in the **Tree** panel:

```
┌─ Tree ──────────────┐
│ 📄 user_001        │  ← Documents
│ 📄 user_002        │
│ 📄 user_003        │
│ ▸ 📁 user_004      │  ← Has subcollection
└─────────────────────┘
```

Press `Space` on a document to view its data in the **Details** panel.

## Subcollections

Documents with subcollections show a folder icon and arrow:

- `▸` indicates collapsed subcollection
- `▾` indicates expanded subcollection

Navigate subcollections:
- `Space` or `l` to expand
- `h` to collapse / go to parent
- `Backspace` to go back in history

## Document Details

The **Details** panel shows the selected document's data as formatted JSON:

```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "created": "2024-01-15T10:30:00Z",
  "orders": 42,
  "metadata": {
    "source": "web",
    "verified": true
  }
}
```

### Scrolling Details

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll up/down |
| `Ctrl+d` | Page down |
| `Ctrl+u` | Page up |

## Select Mode

Select multiple documents for batch operations:

1. Press `v` to enter select mode
2. Use `j`/`k` to move, `Space` to toggle selection
3. Press `V` to select all visible
4. Press `Esc` to exit select mode

Selected documents are highlighted and counted in the status bar.
