# Filtering & Search

LazyFire includes powerful filtering to quickly find what you need.

## Basic Filtering

Press `/` to start filtering in any panel:

```
┌─ Collections ───────────────┐
│ Filter: user█               │  ← Type to filter
├─────────────────────────────┤
│ 📁 users                    │  ← Matching items
│ 📁 user_settings            │
└─────────────────────────────┘
```

## Filter Behavior

- Filtering is **case-insensitive**
- Matches anywhere in the name (not just prefix)
- Results update as you type
- Empty filter shows all items

## Filter Keybindings

| Key | Action |
|-----|--------|
| `/` | Start filtering |
| `Enter` | Apply filter and navigate |
| `Esc` | Cancel filter |
| `Backspace` | Delete character |
| `Ctrl+u` | Clear filter text |

## Panel-Specific Filtering

Each panel maintains its own filter:

| Panel | What it filters |
|-------|-----------------|
| Projects | Project names |
| Collections | Collection names or function names |
| Tree | Document IDs |
| Details | (scrolling, not filterable) |

## Filter Indicators

When a filter is active, the panel shows:
- Filter text in the header
- Count of matching items
- Visual indicator that filter is applied

```
┌─ Collections (3/15) ────────┐
│ Filter: user                │
├─────────────────────────────┤
│ 📁 users                    │
│ 📁 user_data                │
│ 📁 user_settings            │
└─────────────────────────────┘
```

## Clearing Filters

- Press `Esc` while typing to cancel
- Press `Esc` again to clear committed filter
- Switching panels preserves filters

## Tips

1. **Quick jump**: Type `/` then first few characters of what you want
2. **Partial match**: Filter `ord` matches `orders`, `order_items`, `records`
3. **Navigate while filtered**: Use `j`/`k` to move through filtered results
