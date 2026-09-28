# Domain Model — Vynno API

**Status:** Accepted  
**Last updated:** 2026-09-10

This is the conceptual model the **server** must implement. It is not a SQL schema and it is **not** the HTTP wire format.

Wire JSON lives in [api-contract.md](./api-contract.md). On the wire, projects use `archived` (not `isArchived`) and absent optionals are JSON `null`.

If this file and the live API disagree, treat the documented rules here plus [api-contract.md](./api-contract.md) as what the API must do.

---

## 1. Glossary

| Term | Meaning |
| --- | --- |
| **Project** | Named container for work. Has a color used in lists and charts. |
| **Session / time entry** | A continuous timed interval. While `active` it is the *live session*; when `stopped` it is a historical log entry. |
| **Task / note** | Free-text description on a session (`note`). Not a separate entity in v1. |
| **Activity type** | User-owned dictionary row (display `name` + token `color`). Optional on a session. |
| **Profile** | Display name, email, optional avatar. Display name and avatar are writable after register. Email is the login identifier. |
| **User** | Login account. Owns a profile, projects, and sessions. Not on the wire. |
| **Live session** | The at-most-one session whose status is `active`. |

v1 does **not** have a Task table. “Recent tasks” on the client are reconstructed from recent sessions.

---

## 2. Entity relationship (conceptual)

```
User* (many personal accounts; isolated; no teams)
 │
 ├── email / password hash (email is on ProfileDto; hash is not)
 ├── Profile
 │    ├── displayName
 │    ├── email
 │    └── avatarUrl?
 │
 ├── Project*
 │    ├── id
 │    ├── name
 │    ├── color
 │    ├── code?
 │    ├── progressPercent?
 │    └── archived
 │
 ├── ActivityType*
 │    ├── id
 │    ├── name
 │    └── color
 │
 └── TimeSession*
      ├── id
      ├── projectId
      ├── note
      ├── ticketId?
      ├── activityTypeId?
      ├── status: active | stopped
      ├── startedAt
      ├── endedAt?
      └── targetDurationMs?
```

---

## 3. Session lifecycle

```
  [idle] ──start──► [active] ──stop──► [stopped]
                                         (log entry)
```

### Rules

| Rule | Description |
| --- | --- |
| **Single live session** | At most one session with status `active`. A second start is `409 session_already_active`. **Do not auto-stop** the current one. |
| **Idle** | No live session. `GET /sessions/active` → `404 session_not_active`. |
| **Start** | Creates a new row, `status=active`, `startedAt=now` (UTC ISO). Project must exist and must not be archived. |
| **Stop** | Only from `active`. Sets `status=stopped`, `endedAt=now`. |
| **Invalid transition** | Stop while stopped (or any other illegal verb) is `409 invalid_transition`. |
| **Restart** | Client sends a new `POST /sessions` with the same `projectId` / `note` / optionals. Never mutate a stopped row to make it live again. A break is stop, then start. |
| **Patch** | Any session. Writable: `note`, `projectId`, `activityTypeId`, `ticketId`, `startedAt`, `endedAt`, `targetDurationMs`. Not writable: `status`. Archived projects are allowed. |
| **Delete** | Any session, including live. Hard-delete. Idle after deleting live. |
| **Manual entry** | `POST /sessions/manual` inserts `stopped` with `startedAt`/`endedAt`. Allowed while a live session exists. Archived projects are allowed. |
| **Empty note** | NFC and trim; if empty, store `"Untitled session"`. Notes ≤ 500 code points after trim; ticketId ≤ 64; one emoji counts as 1. Tabs, LF, CR allowed in notes. Other Cc and bidi controls rejected. Existing oversized notes still load; a patch that omits `note` still succeeds. |
| **Time integrity** | Stopped: `endedAt > startedAt` at microsecond precision. Live: `endedAt` is null. Session instants are compared at microsecond precision. `startedAt >= 2000-01-01T00:00:00Z`. `startedAt` and `endedAt` ≤ now+5min. Duration ≤ 7 days. A patch that omits both instants does not re-check bounds. `targetDurationMs` is an integer from 0 through 9007199254740991. |

### Elapsed time (derived, do not store as source of truth)

- `active`: `now - startedAt`
- `stopped`: `endedAt - startedAt`

The client computes display labels. The server must keep `startedAt` and `endedAt` consistent so those formulas work.

---

## 4. Project lifecycle

Full decision: [ADR-0004](./adr/0004-project-lifecycle.md).

| Rule | Description |
| --- | --- |
| **Active** | `archived=false`. Appears in default `GET /projects` and is eligible to start a session. |
| **Archive** | Soft-hide. Excluded from default list. Still returned by `GET /projects/:id`. Cannot start a session on it (`409 project_archived`). |
| **Restore** | `archived=false` again. Restore on a non-archived project is `409 invalid_transition`. Archive on an already-archived project is the same. |
| **Last active** | Cannot archive or hard-delete the last non-archived project (`409 last_active_project`). |
| **Hard delete** | Permanent remove, only when **zero** sessions reference the project. Otherwise `409 project_has_sessions`. |
| **Code** | Optional. Project code: ASCII only, `^[A-Z0-9-]{1,8}$` with at least one letter or digit. `---` and `ı` are 400. `A-1` is accepted. Empty / null means “no code”. Unique case-insensitively among all non-deleted projects. |
| **Name** | Names: the text pipeline (reject U+FFFD, NFC, reject Cc and bidi controls, strip zero-width characters, trim), 1–80 code points. |
| **Color** | `#rrggbb`. The SPA palette is a UI concern; the API accepts any valid hex unless [ADR-0004](./adr/0004-project-lifecycle.md) is amended. |
| **progressPercent** | Optional 0–100. Writable on create/update. `null` means unset. Not derived from estimates. |

---

## 5. Entity details

### 5.1 Project

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string | Opaque, stable |
| `name` | string | Required, trimmed, 1–80 |
| `color` | string | `#rrggbb` |
| `code` | string? | Chip code (`AUTH`); unique when set |
| `progressPercent` | number? | 0–100; optional dashboard metadata |
| `archived` | boolean | Soft-hide flag |

The frontend domain type uses `isArchived`. The wire and this API use `archived`.

### 5.2 TimeSession

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string | Opaque |
| `projectId` | string | Required |
| `note` | string | Task description; default `"Untitled session"` |
| `ticketId` | string? | e.g. `DEV-842` |
| `activityTypeId` | string? | Optional FK to an activity type this user owns |
| `status` | `active` \| `stopped` | |
| `startedAt` | ISO-8601 | UTC |
| `endedAt` | ISO-8601? | Set on stop |
| `targetDurationMs` | number? | Optional session goal; UI is P2 |

### 5.3 ActivityType

Per-user dictionary. Empty until the user creates rows. Full decision: [ADR-0012](./adr/0012-activity-types.md).

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string | Opaque, stable |
| `name` | string | Display label. Names: the text pipeline above, 1–80 code points, stored as typed. Unique per user case-insensitively. `"D\u200bUP"` normalizes to `"DUP"` (`409 name_in_use`). The SPA shows this string; chips render it uppercase. |
| `color` | string | Theme token: `primary` \| `secondary` \| `tertiary` \| `error` \| `on-surface-variant` \| `outline` \| `primary-container` \| `secondary-container`. Chip CSS lives on the client. |

| Rule | Description |
| --- | --- |
| **Optional on session** | `activityTypeId` may be null. Unknown or other-user id is `404 not_found`. |
| **Hard delete** | Only when zero sessions reference the row (`409 activity_type_has_sessions`). No archive. |
| **Duplicate name** | `409 name_in_use`. |
| **Empty list** | Allowed. Register does not seed types. |

### 5.4 Profile

| Field | Type | Notes |
| --- | --- | --- |
| `displayName` | string | Trimmed, at most 80. May be empty. Writable via `PATCH /me`. |
| `email` | string | Login identifier. Emails are stored NFC, lowercased, domain in IDNA punycode. NFC and NFD are one account. An IDN domain and its punycode form are one stored email. Cc/Cf anywhere → `400 invalid_body`. Local part longer than 64 octets → 400; 64 is accepted. Non-ASCII local parts are allowed. Existing rows are rechecked; collisions are reported, not auto-merged. Not writable after register. |
| `avatarUrl` | string? | JSON `null` when absent. `avatarUrl` stays the absolute URL `{PUBLIC_API_ORIGIN}/v1/avatars/{uuid}` (internal origin on local prod). The SPA rewrites it to a same-origin path before rendering. This is intentional. |

Each account has its own profile. A fresh production database has no users; the first account is `POST /auth/register`. `scripts/reset` / `scripts/seed` are operator-only against `vynno_dev`.

Chrome shows `displayName` if non-empty, otherwise the raw email (no `@` prefix). There is no handle.

| Rule | Description |
| --- | --- |
| **Register** | Two steps. `POST /auth/register/code` sends a 6-digit code when the email is free. `POST /auth/register` with that code creates the profile (`avatarUrl` null). Omitted / empty `displayName` is stored `""`. No photo on register. No user row exists until the code is accepted. |
| **Display name** | Same text pipeline as names, 0–80 code points. `PATCH /me`. Omit leaves it unchanged. `""` clears it. `null` is `invalid_body`. |
| **Email** | Emails are stored NFC, lowercased, domain in IDNA punycode. NFC and NFD are one account. An IDN domain and its punycode form are one stored email. Cc/Cf anywhere → `400 invalid_body`. Local part longer than 64 octets → 400; 64 is accepted. Non-ASCII local parts are allowed. Existing rows are rechecked; collisions are reported, not auto-merged. Still one address whose domain contains a `.`, 3–254 characters. Not accepted on `PATCH /me`. |
| **One-time code** | Six digits. 15 minute TTL. SHA-256 at rest. One active challenge per email+purpose (`register` \| `password_reset`). Resend replaces. 60 s cooldown; 5 sends / hour; 5 guesses then spent. Never on the wire except in the mail body. |
| **Password reset** | `POST /auth/password/forgot` always succeeds for a well-formed email; mail only if the account exists. `POST /auth/password/reset` sets a new hash and deletes every session token for that user. No cookie. Login afterwards. |
| **Avatar upload** | `PUT /me/avatar`, multipart field `file`. JPEG / PNG / WebP by magic bytes. Max 1 MiB. Replacing allocates a new UUID and deletes the previous row. |
| **Avatar delete** | `DELETE /me/avatar`. Idempotent: already-null still succeeds. |
| **Avatar GET** | `GET /avatars/:id` is public. Unknown id is `404 not_found`. Bytes are not on the profile row. |

### 5.5 Aggregates

**Not stored and not served in v1.** The client computes today/week totals, insights KPIs, and charts from loaded `GET /sessions` pages. Do not add aggregate endpoints without a contract amendment.

---

## 6. Error codes (domain)

These are the codes handlers must emit. HTTP mapping: [api-contract.md](./api-contract.md).

| Code | When |
| --- | --- |
| `not_found` | Unknown project or session id |
| `invalid_body` | Create/update failed validation |
| `invalid_query` | Bad `status` / `limit` / `cursor` |
| `session_not_active` | `GET /sessions/active` when idle |
| `session_already_active` | `POST /sessions` while one is live |
| `project_archived` | Start against an archived project |
| `code_in_use` | Project `code` not unique |
| `name_in_use` | Activity type `name` not unique for this user |
| `last_active_project` | Archive/delete of the last active project |
| `project_has_sessions` | Hard-delete of a project that has sessions |
| `activity_type_has_sessions` | Hard-delete of an activity type that has sessions |
| `invalid_transition` | Verb in a bad state (session or project) |
| `unauthorized` | Missing, unknown, or expired session |
| `invalid_credentials` | Login email/password do not match |
| `email_in_use` | Register with a taken email |
| `invalid_code` | Wrong, expired, or already used one-time code |
| `rate_limited` | Send cooldown, send cap, or too many code guesses |
| `internal_error` | 500 for unhandled faults (not `invalid_body`) |

`internal_error` is 500 for unhandled faults (not `invalid_body`).

Unknown route and wrong method: JSON `404` `not_found`.

Body over the BFF 2 MB limit: `413` with envelope code `invalid_body` and message `Request body is too large.` (the BFF emits this).

Wrong JSON type → `400 invalid_body`. Malformed JSON and trailing data → `400 invalid_json`. Unknown fields are rejected on POST and PATCH (`400 invalid_body`). Empty `status` query means no filter.

`invalid_json`, `invalid_response`, and `http_error` are transport/client codes. The server still returns the envelope for malformed JSON (`invalid_json` / `invalid_body` as appropriate).

---

## 7. Consistency decisions

1. **One live session** — enforced on the server, not only in the SPA store.
2. **Sessions are mutable.** PATCH and DELETE apply to any row. Status still changes only via stop. Manual create is always `stopped`.
3. **Duration precision** — milliseconds. Display formatting is the client.
4. **`user_id` is internal** — not on the wire. Accounts are isolated; there are no team workspaces ([ADR-0006](./adr/0006-single-user-tenancy.md)).
5. **UTC on the wire.** Day grouping and local clocks are the client.
6. **IDs are opaque.** The mock’s `proj-` / `sess-` prefixes are not a contract.

---

## 8. Related documents

- [prd.md](./prd.md)
- [api-contract.md](./api-contract.md)
- [adr/0004-project-lifecycle.md](./adr/0004-project-lifecycle.md)
- [adr/0005-session-lifecycle.md](./adr/0005-session-lifecycle.md)
- [adr/0008-authentication.md](./adr/0008-authentication.md)
- [adr/0015-outbound-email.md](./adr/0015-outbound-email.md)
