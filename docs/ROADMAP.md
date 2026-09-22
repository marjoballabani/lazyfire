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
- **Multiple Databases** - Databases panel to switch between a project's Firestore databases
- **Storage, Auth, Rules and Indexes** - Read-only tabs for Cloud Storage buckets and files, Auth users, security rules and composite indexes
- **Keybindings Menu** - `?` lists the keys for the focused panel, filterable and runnable
- **Mouse Support** - Click, double click, wheel scrolling and tab switching

---

## Planned

### Realtime Database View

Browse Firebase Realtime Database with the same interface as Firestore.

- Toggle with a dedicated key
- Tree view for RTDB paths
- JSON view at any path
- Same filtering/copy/save as Firestore

### Storage Downloads

The Storage tab already lists buckets, folders and file metadata. Next:

- Download files
- Preview text/JSON files

### Hosting Sites View

View hosting deployments.

- List hosting sites
- Show deployment history
- View current deployment details

