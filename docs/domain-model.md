# Domain Model — Vynno API

**Status:** Draft  
**Last updated:** 2026-08-17

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
| **Activity type** | Fixed category (Deep Work, Meeting, Coding, …). Optional on a session. |
| **Tag** | Secondary string labels on a session. Distinct from project color. |
| **Profile** | Display name, handle, optional avatar. Display name and avatar are writable after register. |
| **User** | Login account. Owns a profile, projects, and sessions. Not on the wire. |
| **Live session** | The at-most-one session whose status is `active` or `paused`. |

v1 does **not** have a Task table. “Recent tasks” on the client are reconstructed from recent sessions.

---

## 2. Entity relationship (conceptual)

```
User* (many personal accounts; isolated; no teams)
 │
 ├── username / password hash (not on the wire)
 ├── Profile
 │    ├── displayName
 │    ├── handle
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
 └── TimeSession*
      ├── id
      ├── projectId
      ├── note
      ├── ticketId?
      ├── activityType?
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
| **Stopped sessions** | Immutable in v1. Edit/delete is P2 and needs a contract amendment. |
| **Empty note** | Trim; if empty, store `"Untitled session"`. |

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
| `activityType` | ActivityType? | Fixed enum |
| `tags` | string[] | Empty array on the wire when none |
| `status` | `active` \| `paused` \| `stopped` | |
| `startedAt` | ISO-8601 | UTC |
| `endedAt` | ISO-8601? | Set on stop |
| `pausedMs` | number | Accumulated completed pause time; `>= 0` |
| `pausedAt` | ISO-8601? | Set only while currently paused |
| `targetDurationMs` | number? | Optional session goal; UI is P2 |

### 5.3 ActivityType

```
deep_work | meeting | maintenance | coding | debugging | docs | research | other
```

Display labels (`Deep Work`, `Debug`) are a frontend i18n concern.

### 5.4 Profile

| Field | Type | Notes |
| --- | --- | --- |
| `displayName` | string | Required, trimmed, 1–80. Writable via `PATCH /me`. |
| `handle` | string | Required, non-empty (client shows e.g. `@alexdev`). Derived from username; not user-editable. |
| `avatarUrl` | string? | JSON `null` when absent. Absolute public URL when set. |

Each account has its own profile. A fresh production database has no users; the first account is `POST /auth/register`. `scripts/reset` / `scripts/seed` are operator-only against `vynno_dev`.

| Rule | Description |
| --- | --- |
| **Register** | Creates the profile with `avatarUrl` null. No photo on `POST /auth/register`. |
| **Display name** | `PATCH /me`. Omit leaves it unchanged. Empty / null is `invalid_body`. |
| **Handle** | `@` + username. Not accepted on `PATCH /me`. |
| **Avatar upload** | `PUT /me/avatar`, multipart field `file`. JPEG / PNG / WebP by magic bytes. Max 1 MiB. Replacing allocates a new UUID and deletes the previous row. |
| **Avatar delete** | `DELETE /me/avatar`. Idempotent: already-null still succeeds. |
| **Avatar GET** | `GET /avatars/:id` is public. Unknown id is `404 not_found`. Bytes are not on the profile row. |

### 5.5 Aggregates

**Not stored and not served in v1.** The client computes today/week totals, insights KPIs, and charts from `GET /sessions`. Do not add aggregate endpoints without a contract amendment.

---

## 6. Error codes (domain)

These are the codes handlers must emit. HTTP mapping: [api-contract.md](./api-contract.md).

| Code | When |
| --- | --- |
| `not_found` | Unknown project or session id |
| `invalid_body` | Create/update failed validation |
| `invalid_query` | Bad `status` / `limit` |
| `session_not_active` | `GET /sessions/active` when idle |
| `session_already_active` | `POST /sessions` while one is live |
| `project_archived` | Start against an archived project |
| `code_in_use` | Project `code` not unique |
| `last_active_project` | Archive/delete of the last active project |
| `project_has_sessions` | Hard-delete of a project that has sessions |
| `invalid_transition` | Verb in a bad state (session or project) |
| `unauthorized` | Missing, unknown, or expired session |
| `invalid_credentials` | Login username/password do not match |
| `username_in_use` | Register with a taken username |

`invalid_json`, `invalid_response`, and `http_error` are transport/client codes. The server still returns the envelope for malformed JSON (`invalid_json` / `invalid_body` as appropriate).

---

## 7. Consistency decisions

1. **One live session** — enforced on the server, not only in the SPA store.
2. **Stopped sessions are immutable** until LOG-6 lands.
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
