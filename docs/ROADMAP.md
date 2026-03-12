# Roadmap

Feature expansion plan for LazyFire.

## Completed

- **Query Builder** - Interactive query building with where clauses, ordering, and limits
- **Cloud Functions View** - List functions, view details, stream logs
- **Collection Health Scan** - Scan all collections against Firestore limits
- **Document Stats** - Size, field count, depth, index entries with color-coded warnings
- **Composite Index Detection** - Accurate index entry counts via Firestore Admin API
- **Emulator Support** - Connect to local Firebase Emulator
- **Customizable Themes** - YAML-based color configuration
- **Visual Select Mode** - Multi-select documents for batch operations

---

## Planned

### Realtime Database View

Browse Firebase Realtime Database with the same interface as Firestore.

- Toggle with a dedicated key
- Tree view for RTDB paths
- JSON view at any path
- Same filtering/copy/save as Firestore

### Storage Browser

Browse Cloud Storage buckets.

- List buckets and folders
- View file metadata (size, type, created)
- Download files
- Preview text/JSON files

### Hosting Sites View

View hosting deployments.

- List hosting sites
- Show deployment history
- View current deployment details

### Service Mode Switching

Quick switching between Firebase services:

- `1` / `D` - Firestore (current default)
- `2` / `R` - Realtime Database
- `3` / `S` - Storage
- `4` / `H` - Hosting

Status bar shows current mode: `[Firestore] project-name`
