# Document Stats

LazyFire displays document statistics to help you monitor Firestore limits compliance.

## Firestore Limits

Firestore has specific limits for documents:

| Metric | Limit |
|--------|-------|
| Document size | 1 MiB (1,048,576 bytes) |
| Field count | 20,000 fields |
| Nesting depth | 20 levels |
| Field name size | 1,500 bytes |
| Field value size | ~1 MiB |
| Document path | 6 KiB (6,144 bytes) |

## Stats Display

When viewing a document, stats are shown at the top of the Details panel:

```
Size: 45.2 KB / 1MB  Fields: 156 / 20000  Depth: 4 / 20
Field Name: 24 / 1500 B  Field Value: 2.1 KB / 1MB  Path: 48 / 6144 B
```

## Color-Coded Warnings

Stats are color-coded based on percentage of limit:

| Color | Usage | Meaning |
|-------|-------|---------|
| Green | < 50% | Safe |
| Cyan | 50-70% | Moderate |
| Yellow | 70-85% | Warning |
| Orange | 85-100% | Critical |
| Red | > 100% | Over limit |

## What Each Stat Means

### Size
Total document size in JSON format. Large embedded data (images, long strings) increases this.

### Fields
Total number of fields, including nested fields. Each key in a map counts as a field.

### Depth
Maximum nesting level. `{a: {b: {c: 1}}}` has depth 3.

### Field Name
Longest field name in bytes. Relevant for deeply nested structures with long keys.

### Field Value
Largest single field value size. Watch for large strings or embedded JSON.

### Path
Document path length. `collection/doc/subcol/subdoc` - relevant for deeply nested documents.

## Use Cases

- **Pre-migration checks** - Verify documents won't exceed limits
- **Debugging** - Find why writes are failing
- **Optimization** - Identify documents that need restructuring
- **Monitoring** - Track document growth over time
