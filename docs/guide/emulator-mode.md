# Emulator Mode

LazyFire can connect to a local Firebase Emulator instead of production Firestore. This is useful for development and testing without touching real data.

## Configuration

Add an `emulator` section to your `~/.lazyfire/config.yaml`:

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
- Cloud Functions view is not available
- Composite index detection always reports unknown
- Project details (region, type) show minimal information
