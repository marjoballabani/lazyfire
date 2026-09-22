# Collections & Documents

LazyFire provides a visual browser for your Firestore collections and documents.

## Browsing Collections

1. Select a project from the **Projects** panel
2. If the project has more than one Firestore database, pick one in the **Databases** panel (see [Databases](/guide/databases))
3. Collections appear in the **Collections** panel
4. Use `j`/`k` to navigate, `Space` to open a collection, or `Enter` to open it and move to the tree

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

Press `Space` on a document to load it into the **Details** panel and list its subcollections, or `Enter` to open it and move into Details.

## Subcollections

Documents with subcollections show a folder icon and arrow:

- `▸` indicates collapsed subcollection
- `▾` indicates expanded subcollection

Navigate subcollections:
- `Space` expands or collapses a document's subcollections, or a subcollection's documents
- `Enter` on a subcollection expands it
- `C` collapses everything

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

### Moving in Details

A highlighted cursor line shows where you are.

| Key | Action |
|-----|--------|
| `j` / `k` | Move the cursor down / up |
| `J` / `K` | Move 5 lines |
| `Ctrl+d` / `Ctrl+u` | Half page down / up |
| `g` / `G` | Top / bottom |
| `y` | Copy the value on the cursor line |
| `Esc` or `Tab` | Go back to the tree |

## Select Mode

Select multiple documents for batch operations:

1. Press `v` to enter select mode
2. Use `j`/`k` to extend the selection
3. Press `Space` to fetch all selected documents
4. Press `Esc` to exit select mode

Selected documents are marked with `+` and counted in the bar at the bottom. See [Visual Select Mode](/guide/select-mode).
