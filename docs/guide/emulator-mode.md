# Emulator Mode

LazyFire can connect to a local Firebase Emulator instead of production Firestore. This is useful for development and testing without touching real data.

## Configuration

Add an `emulator` section to your config file (`~/.config/lazyfire/config.yml`, see [Configuration](/guide/configuration)):

```yaml
emulator:
  enabled: true
  projectId: "my-local-project"
  firestoreHost: "localhost:8080"
```

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `enabled` | yes | `false` | Enable emulator mode |
| `projectId` | yes | - | Project ID to use (can be any string) |
| `firestoreHost` | no | `localhost:8080` | Emulator host and port |

## How It Works

When emulator mode is enabled:

- LazyFire connects directly to the local Firestore emulator
- Firebase CLI authentication is skipped
- The project list shows only your configured `projectId`
- The Databases panel shows only `(default)`
- Composite index detection is disabled (Admin API is not available locally)
- All read operations work the same as production

## Starting the Emulator

Make sure the Firestore emulator is running before starting LazyFire:

```bash
firebase emulators:start --only firestore
```

Then start LazyFire normally:

```bash
lazyfire
```

## Limitations

- Only Firestore is supported in emulator mode
- The Functions, Storage, Auth and Indexes tabs stay empty, and the Rules tab shows a placeholder
- Only the default database is listed
- Composite index detection always reports unknown
- Project details (region, type) show minimal information
