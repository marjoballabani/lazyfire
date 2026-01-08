# Filtering & jq Queries

LazyFire supports filtering in all panels, with special jq query support in the details panel.

## Basic Filtering

Press `/` in any panel to start filtering. Type your filter text and press `Enter` to apply.

### Filter Behavior

- **Case-insensitive** - "user" matches "User", "USER", "user"
- **Substring matching** - "abc" matches "xabcy"
- **Real-time preview** - Results update as you type

### Filter Keys

| Key | Action |
|-----|--------|
| `/` | Start filter |
| `Enter` | Apply filter |
| `Esc` | Cancel filter input |
| `Esc` (after applied) | Clear filter |
| `Backspace` | Delete character |

## Panel Filters

### Projects Panel
Filter by project name or ID.

### Collections Panel
Filter by collection name.

### Tree Panel
Filter by document ID or path.

### Details Panel
Filter JSON content by line or use jq queries.

## jq Query Support

In the **Details panel**, filters starting with `.` are treated as jq queries.

### Examples

```bash
# Get a specific field
.name

# Get nested field
.user.email

# Get array element
.items[0]

# Get all emails from array
.users[].email

# Filter array
.items | map(select(.active == true))

# Get keys
. | keys

# Count items
.items | length
```

### jq Query Results

- Results are syntax-highlighted
- Copy/save exports the filtered result, not the full document
- Invalid queries show an error message

## Filter Indicators

When a filter is active:
- Panel border changes color (magenta by default)
- Footer shows match count: "5/20 matched"

## Examples

### Finding a specific project
1. Focus Projects panel
2. Press `/`
3. Type "prod"
4. Press `Enter`
5. Only projects containing "prod" are shown

### Extracting user emails with jq
1. Select a document with user data
2. Focus Details panel
3. Press `/`
4. Type `.users[].email`
5. Press `Enter`
6. Only email addresses are displayed
