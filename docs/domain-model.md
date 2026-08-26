# Domain Model — Vynno API

**Status:** Draft  
**Last updated:** 2026-08-26

This is the conceptual model the **server** must implement. It is not a SQL schema and it is **not** the HTTP wire format.

Wire JSON lives in [api-contract.md](./api-contract.md). On the wire, projects use `archived` (not `isArchived`) and absent optionals are JSON `null`.

Inherited from the frontend domain model and from the mock engine the SPA already ships. If this file and the live mock disagree, treat the documented rules here plus [api-contract.md](./api-contract.md) as what the API must do.

---

## 1. Glossary

| Term | Meaning |
| --- | --- |
| **Project** | Named container for work. Has a color used in lists and charts. |
| **Session / time entry** | A timed interval. While `active` or `paused` it is the *live session*; when `stopped` it is a historical log entry. |
| **Task / note** | Free-text description on a session (`note`). Not a separate entity in v1. |
| **Activity type** | User-owned dictionary row (display `name` + token `color`). Optional on a session. |
| **Tag** | Secondary string labels on a session. Distinct from project color. |
| **Profile** | Display name, email, optional avatar. Display name and avatar are writable after register. Email is the login identifier. |
| **User** | Login account. Owns a profile, projects, and sessions. Not on the wire. |
| **Live session** | The at-most-one session whose status is `active` or `paused`. |

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
      ├── tags[]
      ├── status: active | paused | stopped
      ├── startedAt
      ├── endedAt?
      ├── pausedMs
      ├── pausedAt?
      └── targetDurationMs?
```

---

## 3. Session lifecycle

```
                    start
     ┌──────────────────────────────────┐
     │                                  ▼
  [idle] ──start──► [active] ◄──resume── [paused]
                       │                    ▲
                       │ pause              │
                       └────────────────────┘
                       │
                       │ stop
                       ▼
                   [stopped]
                 (log entry)
```

### Rules

| Rule | Description |
| --- | --- |
| **Single live session** | At most one session with status `active` or `paused`. A second start is `409 session_already_active`. **Do not auto-stop** the current one. |
| **Idle** | No live session. `GET /sessions/active` → `404 session_not_active`. |
| **Start** | Creates a new row, `status=active`, `startedAt=now` (UTC ISO), `pausedMs=0`. Project must exist and must not be archived. |
| **Pause** | Only from `active`. Sets `status=paused`, `pausedAt=now`. |
| **Resume** | Only from `paused`. Adds `now - pausedAt` to `pausedMs` (if positive), clears `pausedAt`, sets `status=active`. |
| **Stop** | From `active` or `paused`. If paused, apply the same pause-accounting as resume first. Sets `status=stopped`, `endedAt=now`. |
| **Invalid transition** | Any other verb (pause while paused, resume while active, stop while stopped, …) is `409 invalid_transition`. |
| **Restart** | Client sends a new `POST /sessions` with the same `projectId` / `note` / optionals. Never mutate a stopped row to make it live again. |
| **Patch** | Any session. Writable: `note`, `projectId`, `activityTypeId`, `ticketId`, `tags`, `startedAt`, `endedAt`, `pausedMs`, `targetDurationMs`. Not writable: `status`, `pausedAt`. Archived projects are allowed. |
| **Delete** | Any session, including live. Hard-delete. Idle after deleting live. |
| **Manual entry** | `POST /sessions/manual` inserts `stopped` with `startedAt`/`endedAt`. Allowed while a live session exists. Archived projects are allowed. |
| **Empty note** | Trim; if empty, store `"Untitled session"`. |
| **Time integrity** | Stopped: `endedAt > startedAt` and `0 <= pausedMs <= endedAt - startedAt`. Live: `endedAt` is null. Paused: `pausedAt >= startedAt`. `pausedMs` must fit the interval. |

### Elapsed time (derived, do not store as source of truth)

- `active`: `now - startedAt - pausedMs`
- `paused`: same formula using the instant of the pause (`pausedAt` is not yet folded into `pausedMs`)
- `stopped`: `endedAt - startedAt - pausedMs`

The client computes display labels. The server must keep `startedAt`, `endedAt`, `pausedMs`, and `pausedAt` consistent so those formulas work.

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
| **Code** | Optional. When set: trim, uppercase, `^[A-Z0-9-]{1,8}$`, unique case-insensitively among all non-deleted projects. Empty / null means “no code”. |
| **Name** | Required, trimmed, 1–80 characters. |
| **Color** | `#rrggbb`. The SPA palette is a UI concern; the API default is any valid hex ([open-questions.md](./open-questions.md) #5). |
| **progressPercent** | Optional 0–100. Not user-edited in project CRUD v1. May be `null`. |

---

## 5. Entity details

### 5.1 Project

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string | Opaque, stable |
| `name` | string | Required, trimmed, 1–80 |
| `color` | string | `#rrggbb` |
| `code` | string? | Chip code (`AUTH`); unique when set |
| `progressPercent` | number? | 0–100; mock metadata today |
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
| `tags` | string[] | Empty array on the wire when none |
| `status` | `active` \| `paused` \| `stopped` | |
| `startedAt` | ISO-8601 | UTC |
| `endedAt` | ISO-8601? | Set on stop |
| `pausedMs` | number | Accumulated completed pause time; `>= 0` |
| `pausedAt` | ISO-8601? | Set only while currently paused |
| `targetDurationMs` | number? | Optional session goal; UI is P2 |

### 5.3 ActivityType

Per-user dictionary. Empty until the user creates rows. Full decision: [ADR-0012](./adr/0012-activity-types.md).

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string | Opaque, stable |
| `name` | string | Display label. Trim, 1–80, stored as typed. Unique per user case-insensitively. The SPA shows this string; chips render it uppercase. |
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
| `email` | string | Login identifier. Unique, lowercase. Not writable after register. |
| `avatarUrl` | string? | JSON `null` when absent. Absolute public URL when set. |

Each account has its own profile. A fresh production database has no users; the first account is `POST /auth/register`. `scripts/reset` / `scripts/seed` are operator-only against `vynno_dev`.

Chrome shows `displayName` if non-empty, otherwise the raw email (no `@` prefix). There is no handle.

| Rule | Description |
| --- | --- |
| **Register** | Two steps. `POST /auth/register/code` sends a 6-digit code when the email is free. `POST /auth/register` with that code creates the profile (`avatarUrl` null). Omitted / empty `displayName` is stored `""`. No photo on register. No user row exists until the code is accepted. |
| **Display name** | `PATCH /me`. Omit leaves it unchanged. `""` clears it. `null` is `invalid_body`. |
| **Email** | Trim, lowercase, 3–254, a single address whose domain contains a `.`. Unique. Not accepted on `PATCH /me`. |
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

`invalid_json`, `invalid_response`, and `http_error` are transport/client codes. The server still returns the envelope for malformed JSON (`invalid_json` / `invalid_body` as appropriate).

---

## 7. Consistency decisions

1. **One live session** — enforced on the server, not only in the SPA store.
2. **Sessions are mutable.** PATCH and DELETE apply to any row. Status still changes only via pause/resume/stop. Manual create is always `stopped`.
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
- [plans/email.md](./plans/email.md)
