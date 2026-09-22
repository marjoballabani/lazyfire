# Storage, Auth, Rules & Indexes

Besides Collections and Functions, the collections panel has four more tabs. Press `]` and `[` to move between them. All of them are read-only.

```
╭─ < Storage - Auth - Rules > ────╮
```

## Storage

Browse Cloud Storage buckets and the files in them.

- The first level lists the project's buckets with their location and storage class.
- `Space` or `Enter` opens a bucket or folder.
- `Esc` or `Backspace` goes up one level, keeping the folder you came from selected.
- Details shows the selected bucket (location, storage class, creation time) or file (size, content type, created and updated times).
- `/` filters the buckets or the files in the current folder.

Each folder shows up to 100 items.

## Auth

List Firebase Authentication users.

- Each row shows the email (or UID when there is no email), sign-in providers, and whether the account is disabled.
- `Space` or `Enter` moves to Details, which shows the UID, email, name, whether the email is verified, whether the account is disabled, when it was created, the last sign-in, providers and photo URL.
- `/` filters by email, UID or name.

The tab lists up to 100 users.

## Rules

Shows the newest Firestore security ruleset in the project, with the time it was deployed. `rules_version` and `service` lines are cyan, `match` lines green, `allow` lines yellow and comments dim.

`j`/`k` scroll the rules in the panel. Press `Enter` to read them in the wider Details panel.

## Indexes

Lists the composite indexes of the selected [database](/guide/databases): the collection group, query scope, fields and their order, and the state.

| Color | State |
|-------|-------|
| Green | READY |
| Yellow | CREATING |
| Red | NEEDS_REPAIR |

`j`/`k` scroll the list, and `Enter` shows it in Details.

## Keys

| Key | Action |
|-----|--------|
| `[` / `]` | Previous / next tab |
| `Space` / `Enter` | Open a bucket or folder (Storage), view a user (Auth) |
| `Enter` | View in Details (Rules, Indexes) |
| `Esc` / `Backspace` | Up one level (Storage) |
| `/` | Filter (Storage, Auth) |
| `r` | Reload the tab |
