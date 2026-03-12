# Collection Health Scan

LazyFire can scan all collections in a project and check documents against Firestore limits. This helps identify collections with documents approaching or exceeding size, field count, depth, and other constraints.

## How to Use

1. Navigate to the **Projects** panel
2. Focus on the project you want to scan
3. Press `S` (Shift+S)
4. Confirm the scan in the dialog

LazyFire selects the project, switches to the **Details** panel, and begins scanning. The collections panel also updates to reflect the scanned project.

## What Gets Checked

For each collection, LazyFire fetches the first document and measures it against Firestore limits:

| Metric | Limit |
|--------|-------|
| Document size | 1 MiB |
| Field count | 20,000 |
| Index entries | 40,000 |
| Nesting depth | 20 levels |
| Field name size | 1,500 bytes |
| Field value size | 1 MiB |
| Document path | 6,144 bytes |

If the first document triggers a warning (above 70% of any limit), LazyFire fetches a second document to confirm the pattern.

## Reading Results

Each collection shows a status icon and all metric values:

| Status | Meaning |
|--------|---------|
| OK | All metrics below 70% of limits |
| Warning | One or more metrics above 70% |
| Error | Failed to fetch or check the document |
| Skipped | Collection is empty |

Metrics that exceed 70% are highlighted with a warning marker.

## Export Results

While viewing scan results in the Details panel:

- Press `c` to copy the report as Markdown to your clipboard
- Press `s` to save the report as a `.md` file

The exported report includes a summary table with status icons and a detailed breakdown per collection with collapsible sections for healthy collections.

## Tips

- The scan only reads documents -- it does not modify anything
- Index entry counts use the Firestore Admin API for composite index detection when available
- In emulator mode, composite index checks are skipped
- Press `Esc` to dismiss scan results and return to the previous panel
