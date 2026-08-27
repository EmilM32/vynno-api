# ADR-0012: User-defined activity types

**Status:** Accepted  
**Date:** 2026-08-20  
**Deciders:** Project owner

## Context

v1 treated `activityType` as a closed enum (`deep_work` | `meeting` | …). Display labels lived in SPA i18n; chip colors were a hardcoded CSS map. The session column was unconstrained `TEXT`.

The product needs a per-user dictionary: add, rename, recolor, delete. Labels must keep working when the UI locale switches. Colors must stay on the design-system tokens, not arbitrary hex.

## Decision

1. **Dedicated `activity_types` table**, owned by `user_id`. Not a generic dictionaries framework.
2. **`name` is a display string.** Trim, 1–80 characters, stored as typed (case preserved). Unique per user case-insensitively. The SPA shows this string; the activity chip renders it in uppercase. There is no i18n lookup — user-defined names cannot be added to translation JSON at runtime.
3. **`color` is a token name** from a closed set the SPA already styles (`primary`, `secondary`, `tertiary`, `error`, `on-surface-variant`, `outline`, `primary-container`, `secondary-container`). The API rejects anything else (`invalid_body`). Hex is for projects only.
4. **Sessions store `activity_type_id`** (nullable UUID FK). Wire field is `activityTypeId`. Null remains allowed.
5. **Empty on register.** Do not seed the old eight types. Existing session slugs are migrated to rows so history keeps a type; new accounts start with `{ items: [] }`.
6. **Hard delete** only when zero sessions reference the row (`409 activity_type_has_sessions`). No archive, no cascade, no auto-null.
7. **Duplicate name** is `409 name_in_use`. Unknown id (including another user’s) is `404 not_found`.

## Consequences

### Positive

- Users configure the list from scratch.
- Display copy is whatever the user typed; no missing-translation gap.
- Chip and chart colors stay on theme tokens.

### Negative / tradeoffs

- Locale switch does not translate activity type names; they stay in the language they were saved in.
- Breaking wire change: `activityType` slug → `activityTypeId`. SPA must ship with the API.
- Insights cannot assume an `other` bucket.

## Alternatives considered

| Option | Why not |
| --- | --- |
| Keep the closed enum | Blocks add/delete. |
| Technical key + i18n JSON | User-created names cannot be added to `messages/*.json` at runtime. |
| Hex colors | Diverges from the existing chip token map; theme changes would not follow. |
| Generic `dictionaries` table | Only activity types need this. |
| Seed the old eight on register | Owner wants an empty list. |
| Snapshot name+color on the session | Rename would fork history; dictionary edit is the point. |

## Related

- [../domain-model.md](../domain-model.md)
- [../api-contract.md](../api-contract.md)

## Amendment (2026-08-20)

`name` is a per-user display label, not an i18n key. Trim only; do not lowercase. Chip uppercase is CSS, not storage.
