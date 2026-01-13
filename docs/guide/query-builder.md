# Query Builder

The Query Builder allows you to create Firestore queries with an interactive UI.

## Opening the Query Builder

Press `F` (Shift+F) on:
- A collection in the **Collections** panel
- A subcollection in the **Tree** panel

## Query Builder Interface

```
┌─ Query Builder ─────────────────────────────┐
│ Collection: users                           │
│                                             │
│ WHERE:                                      │
│   [field] [==] (auto) [value]               │
│                                             │
│ ORDER BY:  [field] [ASC]                    │
│ LIMIT:     [50]                             │
│                                             │
│ [ Execute ]  [ Clear ]                      │
└─────────────────────────────────────────────┘
```

## Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move to next row |
| `k` / `↑` | Move to previous row |
| `h` / `←` | Move to previous field |
| `l` / `→` | Move to next field |
| `Enter` | Edit selected field / Execute button |
| `Esc` | Close query builder |

## Adding Filters

| Key | Action |
|-----|--------|
| `a` | Add new WHERE filter |
| `d` | Delete current WHERE filter |

## Filter Fields

Each WHERE filter has four components:

1. **Field** - The document field to filter on (e.g., `status`, `age`, `createdAt`)
2. **Operator** - Comparison operator (opens popup selector)
3. **Type** - Value type (auto-detected or manual)
4. **Value** - The value to compare against

## Operators

| Operator | Description |
|----------|-------------|
| `==` | Equal to |
| `!=` | Not equal to |
| `<` | Less than |
| `<=` | Less than or equal |
| `>` | Greater than |
| `>=` | Greater than or equal |
| `in` | Value in array |
| `not-in` | Value not in array |
| `array-contains` | Array contains value |
| `array-contains-any` | Array contains any of values |

## Value Types

| Type | Description |
|------|-------------|
| `auto` | Auto-detect type from value |
| `string` | Text value |
| `integer` | Whole number |
| `double` | Decimal number |
| `boolean` | true/false |
| `null` | Null value |
| `array` | Array (for `in`, `not-in`, `array-contains-any`) |

## ORDER BY

Set the field to sort results by and the direction:
- `ASC` - Ascending (smallest first)
- `DESC` - Descending (largest first)

## LIMIT

Maximum number of documents to return (default: 50).

## Executing Queries

Press `Enter` on the **Execute** button to run the query.

### Top-level Collection Query
Results replace the entire tree view.

### Subcollection Query
Results appear under the subcollection node in the tree, preserving the rest of the tree structure.

## Clearing Queries

Press `Enter` on the **Clear** button to reset all filters, ORDER BY, and LIMIT to defaults.

## Examples

### Find Active Users
```
WHERE: [status] [==] (string) [active]
```

### Find Users Over 18, Sorted by Age
```
WHERE: [age] [>] (integer) [18]
ORDER BY: [age] [ASC]
```

### Find Recent Orders
```
WHERE: [createdAt] [>] (auto) [2024-01-01]
ORDER BY: [createdAt] [DESC]
LIMIT: [100]
```

## Tips

- Use `auto` type for most values - it will detect strings, numbers, and booleans
- For date fields, enter ISO format dates as strings
- Firestore requires composite indexes for some query combinations
- If a query fails, check the error message for index requirements
