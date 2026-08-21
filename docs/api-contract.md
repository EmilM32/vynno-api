# Vynno API contract

**Status:** Snapshot of the frontend-proposed contract — this API must implement it  
**Snapshot date:** 2026-08-14  
**Last updated:** 2026-08-21  
**Amended:** Profile writes + public avatar GET (ME-3 / ME-4 / ME-5); user-defined activity types (ADR-0012); session edit / delete / manual entry (LOG-6 / LOG-7)

This is the wire format the SvelteKit app already speaks. Implement these resources. Do not extend this file without a contract amendment ([working-agreement.md](./working-agreement.md) §6).

**Provenance.** Copied from the frontend repo ([`vynno`](https://github.com/EmilM32/vynno)) `docs/api-contract.md` as of the snapshot date. The **client executable source of truth** remains that repo’s `src/lib/api/schemas/` until this backend publishes its own schemas (or OpenAPI) and both sides agree.

If this doc and the frontend schemas drift, stop and reconcile — do not “fix” only one side.

---

## Conventions

| Rule | Value |
| --- | --- |
| Prefix | `/v1` |
| Format | JSON, camelCase |
| Lists | `{ "items": T[] }` |
| Errors | `{ "error": { "code": string, "message": string } }` |
| Timestamps | ISO-8601 (`Date.toISOString()`) |
| Absent optionals | JSON `null` (not omitted) |
| IDs | Opaque strings |
| Pagination | Not yet — `limit` query only |
| Auth | HttpOnly session cookie (see [Auth](#auth)) |
| Operator docs | `GET /swagger/` and `GET /openapi.json` — **not** SPA resources ([ADR-0013](./adr/0013-openapi-swagger.md)) |

Creates return **`201`**. Other successful writes return **`200`** with the updated resource. `DELETE` returns **`204`** with an empty body.

---

## Error codes

| Code | Status | When | Frontend UI string |
| --- | --- | --- | --- |
| `not_found` | 404 | Unknown project, session, or activity type id | `error_not_found` |
| `invalid_query` | 400 | Bad `status` / `limit` | fallback |
| `invalid_json` | 400 | Request body is not JSON | `error_invalid_response` |
| `invalid_body` | 400 | Write body failed the request schema / validation | fallback (`error_failed_*`) |
| `invalid_response` | 502 | Client-only: body did not match the response schema | `error_invalid_response` |
| `http_error` | 4xx/5xx | Client-only: non-OK without an envelope | fallback |
| `session_not_active` | 404 | `GET /sessions/active` when idle | `error_not_found` |
| `session_already_active` | 409 | `POST /sessions` while one is active/paused | `error_stop_before_start` |
| `project_archived` | 409 | Start against an archived project | `error_project_archived` |
| `code_in_use` | 409 | Project `code` not unique | `error_code_in_use` |
| `name_in_use` | 409 | Activity type `name` not unique for this user | `activity_types_name_in_use` |
| `last_active_project` | 409 | Archive/delete of the last active project | `error_last_active_project` |
| `project_has_sessions` | 409 | Hard-delete of a project that has logs | `projects_cannot_delete_has_sessions` |
| `activity_type_has_sessions` | 409 | Hard-delete of an activity type that has sessions | `activity_types_cannot_delete_has_sessions` |
| `invalid_transition` | 409 | Pause/resume/stop (or archive/restore) in a bad state | fallback |
| `unauthorized` | 401 | Missing, unknown, or expired session on a protected route | `error_unauthorized` |
| `invalid_credentials` | 401 | Login username/password do not match | `error_invalid_credentials` |
| `username_in_use` | 409 | Register with a taken username | fallback |

`invalid_response` and `http_error` are **not** codes this server should emit. Always send the envelope on failure so the client does not fall back to `http_error`.

Example envelope:

```json
{
	"error": {
		"code": "session_already_active",
		"message": "An active session already exists. Stop it before starting a new one."
	}
}
```

`message` is for logs / DevTools. The SPA maps `code` to Paraglide strings and does not show the raw English `message` for known codes.

---

## Business rules

These are product rules the API must enforce. Details: [domain-model.md](./domain-model.md), [ADR-0004](./adr/0004-project-lifecycle.md), [ADR-0005](./adr/0005-session-lifecycle.md).

1. **One live session.** At most one session with status `active` or `paused`. A second `POST /sessions` is `409 session_already_active`. The client requires an explicit stop — do not auto-stop.
2. **Restart is a new session.** Restart-from-recent sends `POST /sessions` with the same `projectId` / `note` / optional fields. It is not a resume of a stopped log.
3. **Session actions are verbs.** Use `/pause`, `/resume`, `/stop` — not a generic `PATCH status`. `PATCH /sessions/:id` updates fields; it must not send `status` or `pausedAt`.
4. **Elapsed time.** `pausedMs` is accumulated pause duration. On resume/stop-from-paused, add `now - pausedAt` into `pausedMs` and clear `pausedAt`.
5. **Default `GET /projects` omits archived.** Pass `includeArchived=true` for management UI. Archived projects must still resolve via `GET /projects/:id` so logs keep a label.
6. **Last active project.** Cannot archive or hard-delete the last non-archived project (`409 last_active_project`).
7. **Hard delete** only when **zero** sessions reference the project. Otherwise `409 project_has_sessions` — archive instead.
8. **Code uniqueness** is case-insensitive among all non-deleted projects, only when `code` is non-empty.
9. **Cannot start** on a missing (`404 not_found`) or archived (`409 project_archived`) project.
10. **Edit / delete.** `PATCH /sessions/:id` and `DELETE /sessions/:id` apply to any session, including the live timer. Deleting live returns idle. Stopped `endedAt` cannot be cleared. Live `endedAt` cannot be set (use `/stop`, then PATCH the stop time).
11. **Manual entry.** `POST /sessions/manual` creates a stopped log with `startedAt` and `endedAt`. Allowed while a live session exists. Archived projects are allowed. `session_already_active` and `project_archived` apply only to live `POST /sessions`.

---

## Auth

Mechanism: [ADR-0008](./adr/0008-authentication.md).

Login and register set an HttpOnly cookie `vynno_session`. The JSON body is `{ "profile": ProfileDto }` only — the session secret is not in the response.

Protected routes accept **either**:

1. Cookie `vynno_session=<token>` (what the SPA sends via `credentials: 'include'`), or
2. `Authorization: Bearer <token>` (tests, curl, non-browser clients)

Anything else on a protected route is `401 unauthorized`. A project or session id that belongs to another user is `404 not_found`.

| Method | Path | Auth | Body | Success | Typical errors |
| --- | --- | --- | --- | --- | --- |
| POST | `/auth/register` | no | `RegisterDto` | `{ profile }` `201` + `Set-Cookie` | `invalid_body`, `username_in_use` |
| POST | `/auth/login` | no | `LoginDto` | `{ profile }` `200` + `Set-Cookie` | `invalid_body`, `invalid_credentials` |
| POST | `/auth/logout` | yes | — | `204` + clear cookie | `unauthorized` |

`RegisterDto`:

```json
{ "username": "alexdev", "password": "a-long-enough-secret", "displayName": "Alex Dev", "rememberMe": true }
```

`displayName` and `rememberMe` may be omitted. Omitted `displayName` becomes the username. Omitted `rememberMe` is `true`.

`LoginDto`:

```json
{ "username": "alexdev", "password": "a-long-enough-secret", "rememberMe": true }
```

Username: trim, lowercase, `^[a-z0-9_]{3,32}$`. Password: 8–128 characters.

`rememberMe: true` (default) sets cookie `Max-Age` to 30 days. `false` sets a session cookie (cleared when the browser quits). The server still expires the token after 30 days.

Cookie flags: `HttpOnly`, `SameSite=Lax`, `Path=/`, `Secure` when the process is configured for HTTPS.

CORS is locked to the SPA origin(s) and allows credentials. Mutating cookie-backed requests must send an `Origin` (or `Referer`) in that allowlist.

Public: `POST /auth/login`, `POST /auth/register`, `GET /avatars/:id`. Every other `/v1` resource requires a session. `GET /healthz` is outside `/v1` and stays public. Operator Swagger UI (`GET /swagger/`, `GET /openapi.json`) is also outside `/v1` and public on this loopback process.

### Profile

| Method | Path | Auth | Body | Success | Errors |
| --- | --- | --- | --- | --- | --- |
| GET | `/me` | yes | — | `ProfileDto` | `unauthorized` |
| PATCH | `/me` | yes | `UpdateProfileDto` | `ProfileDto` `200` | `unauthorized`, `invalid_json`, `invalid_body` |
| PUT | `/me/avatar` | yes | `multipart/form-data` field `file` | `ProfileDto` `200` | `unauthorized`, `invalid_body` |
| DELETE | `/me/avatar` | yes | — | `ProfileDto` `200` | `unauthorized` |
| GET | `/avatars/:id` | **no** | — | raw image bytes | `not_found` |

```json
{
	"displayName": "Alex Dev",
	"handle": "@alexdev",
	"avatarUrl": null
}
```

`avatarUrl` is JSON `null` when absent. When set it is an absolute URL `{PUBLIC_API_ORIGIN}/v1/avatars/{uuid}` the browser can load as `<img src>`. The stored value is the path only; the origin is prefixed at read time.

`UpdateProfileDto` — all fields optional. Same present-vs-absent rule as `UpdateProjectDto`.

```json
{ "displayName": "Alex Dev" }
```

- `displayName`: trim; 1–80 characters. Omit = leave unchanged. `null` or `""` → `invalid_body`.
- Do not send `handle` or `avatarUrl` on this body. Handle stays derived from the username. Avatar is only `PUT` / `DELETE /me/avatar`.

`PUT /me/avatar`:

- `Content-Type: multipart/form-data`
- Field name: `file` (one part)
- Detected type (magic bytes, not the client `Content-Type` or filename): `image/jpeg`, `image/png`, `image/webp`
- Max decoded size: 1 MiB
- Replace deletes the previous row (if any), inserts a new UUID, updates `avatarUrl`, returns `ProfileDto`
- Missing part, empty file, unknown type, or oversize → `invalid_body`

`DELETE /me/avatar` when already null is still `200` with `avatarUrl: null`.

`GET /avatars/:id` is public (no cookie). Success is the raw bytes with `Content-Type` from the stored row and `Cache-Control: public, max-age=31536000, immutable`. Unknown id → `404` `{ "error": { "code": "not_found", "message": "…" } }`.

### Projects

| Method | Path | Body | Success | Typical errors |
| --- | --- | --- | --- | --- |
| GET | `/projects?includeArchived=boolean` | — | `{ items: ProjectDto[] }` | — |
| GET | `/projects/:id` | — | `ProjectDto` | `not_found` |
| POST | `/projects` | `CreateProjectDto` | `ProjectDto` `201` | `invalid_body`, `code_in_use` |
| PATCH | `/projects/:id` | `UpdateProjectDto` | `ProjectDto` | `not_found`, `invalid_body`, `code_in_use` |
| POST | `/projects/:id/archive` | — | `ProjectDto` | `not_found`, `last_active_project`, `invalid_transition` |
| POST | `/projects/:id/restore` | — | `ProjectDto` | `not_found`, `invalid_transition` |
| DELETE | `/projects/:id` | — | `204` | `not_found`, `last_active_project`, `project_has_sessions` |
| GET | `/projects/:id/session-count` | — | `{ "count": number }` | — |

`ProjectDto`:

```json
{
	"id": "proj-auth",
	"name": "Identity",
	"color": "#3b82f6",
	"code": "AUTH",
	"progressPercent": 60,
	"archived": false
}
```

`CreateProjectDto`:

```json
{ "name": "New tool", "color": "#3b82f6", "code": "TOOL" }
```

`code` may be `null` or omitted. `color` is a `#rrggbb` hex.

`UpdateProjectDto` — all fields optional; `code: null` clears the chip:

```json
{ "name": "Renamed", "code": null }
```

Optional timestamps `createdAt` / `updatedAt` (ISO-8601) are accepted by the client schema if present. The SPA does not require them. Do not add other extra fields.

### Activity types

Per-user dictionary. Empty until the user creates rows. [ADR-0012](./adr/0012-activity-types.md).

| Method | Path | Body | Success | Typical errors |
| --- | --- | --- | --- | --- |
| GET | `/activity-types` | — | `{ items: ActivityTypeDto[] }` name-sorted | — |
| GET | `/activity-types/:id` | — | `ActivityTypeDto` | `not_found` |
| POST | `/activity-types` | `CreateActivityTypeDto` | `ActivityTypeDto` `201` | `invalid_body`, `name_in_use` |
| PATCH | `/activity-types/:id` | `UpdateActivityTypeDto` | `ActivityTypeDto` | `not_found`, `invalid_body`, `name_in_use` |
| DELETE | `/activity-types/:id` | — | `204` | `not_found`, `activity_type_has_sessions` |
| GET | `/activity-types/:id/session-count` | — | `{ "count": number }` | `not_found` |

`ActivityTypeDto`:

```json
{
	"id": "8f3e0c1a-2b4d-4e6f-8a90-b1c2d3e4f567",
	"name": "coding",
	"color": "secondary"
}
```

`name` is a display label (trim, 1–80 characters, stored as typed), unique per user case-insensitively. The SPA shows this string; chips render it uppercase.

`color` is one of: `primary`, `secondary`, `tertiary`, `error`, `on-surface-variant`, `outline`, `primary-container`, `secondary-container`.

`CreateActivityTypeDto`:

```json
{ "name": "coding", "color": "secondary" }
```

`UpdateActivityTypeDto` — all fields optional:

```json
{ "name": "deep_work", "color": "primary" }
```

### Sessions

| Method | Path | Body | Success | Typical errors |
| --- | --- | --- | --- | --- |
| GET | `/sessions?status=active,paused&limit=n` | — | `{ items: SessionDto[] }` newest-first | `invalid_query` |
| GET | `/sessions/active` | — | `SessionDto` | `session_not_active` |
| GET | `/sessions/:id` | — | `SessionDto` | `not_found` |
| POST | `/sessions` | `StartSessionDto` | `SessionDto` `201` | `session_already_active`, `not_found`, `project_archived`, `invalid_body` |
| POST | `/sessions/manual` | `CreateManualSessionDto` | `SessionDto` `201` | `not_found`, `invalid_body` |
| PATCH | `/sessions/:id` | `UpdateSessionDto` | `SessionDto` | `not_found`, `invalid_body` |
| DELETE | `/sessions/:id` | — | `204` | `not_found` |
| POST | `/sessions/:id/pause` | — | `SessionDto` | `not_found`, `invalid_transition` |
| POST | `/sessions/:id/resume` | — | `SessionDto` | `not_found`, `invalid_transition` |
| POST | `/sessions/:id/stop` | — | `SessionDto` | `not_found`, `invalid_transition` |

`SessionDto`:

```json
{
	"id": "sess-today-1",
	"projectId": "proj-alpha",
	"note": "Database schema migration script",
	"ticketId": null,
	"activityTypeId": "8f3e0c1a-2b4d-4e6f-8a90-b1c2d3e4f567",
	"tags": [],
	"status": "stopped",
	"startedAt": "2026-03-11T08:00:00.000Z",
	"endedAt": "2026-03-11T10:15:00.000Z",
	"pausedMs": 0,
	"pausedAt": null,
	"targetDurationMs": null
}
```

`StartSessionDto`:

```json
{
	"projectId": "proj-auth",
	"note": "Refactoring Auth Service",
	"ticketId": null,
	"activityTypeId": null,
	"tags": [],
	"targetDurationMs": null
}
```

`activityTypeId`: UUID of an activity type this user owns, or JSON `null`. Unknown id is `404 not_found`.  
`status`: `active` \| `paused` \| `stopped`

`GET /sessions/active` returns the active **or paused** session. Idle → `404` `{ "error": { "code": "session_not_active", "message": "…" } }`.

`status` query is a comma-separated list of those enum values. `limit` is a positive integer. Anything else is `400 invalid_query`.

`UpdateSessionDto` — all fields optional. Same present-vs-absent rule as `UpdateProjectDto`. Do not send `status`, `pausedAt`, or `id` (`invalid_body`).

```json
{
	"projectId": "proj-auth",
	"note": "Renamed task",
	"ticketId": null,
	"activityTypeId": null,
	"tags": [],
	"startedAt": "2026-03-11T08:00:00.000Z",
	"endedAt": "2026-03-11T10:15:00.000Z",
	"pausedMs": 0,
	"targetDurationMs": null
}
```

- `note`: trim; empty → `"Untitled session"`.
- `projectId`: must exist for this user. Archived is allowed.
- `activityTypeId` / `ticketId` / `targetDurationMs`: `null` clears.
- `tags`: `null` or `[]` → `[]`.
- `endedAt`: required to stay set on stopped sessions; must stay `null` on live (use `/stop`).
- `pausedMs`: `>= 0` and must not exceed the interval (`endedAt - startedAt` when stopped; `pausedAt - startedAt` when paused; `now - startedAt` when active).

`CreateManualSessionDto` — always inserts `status=stopped`, `pausedAt=null`. Allowed while a live session exists. Archived projects are allowed.

```json
{
	"projectId": "proj-auth",
	"note": "Forgot to start the timer",
	"ticketId": null,
	"activityTypeId": null,
	"tags": [],
	"targetDurationMs": null,
	"startedAt": "2026-03-11T08:00:00.000Z",
	"endedAt": "2026-03-11T10:15:00.000Z",
	"pausedMs": 0
}
```

`projectId`, `startedAt`, and `endedAt` are required. `pausedMs` optional, default `0`. Same note / activity / target rules as start. `endedAt` must be after `startedAt`; `pausedMs` must fit the interval.

---

## Domain vs DTO

The SPA’s UI types use `isArchived` and omit absent optionals. DTOs use `archived` and JSON `null`. Implement the DTO column.

---

## Out of scope

Not in this contract. Do not invent them to “complete” the API without a contract amendment.

| Area | Client today |
| --- | --- |
| Prefs (daily target, default project) | In-memory `prefsStore` |
| Theme / locale | Device-local |
| Insights / dashboard totals | Computed on the client from the session list |
| Pagination / cursors | Full session list on boot (`limit` only) |
| Session target duration UI | Field exists on `StartSessionDto`; UI is P2 |

---

## Swap to a live API

See [frontend-handoff.md](./frontend-handoff.md). Short version:

1. Implement this contract.
2. Frontend sets `PUBLIC_API_BASE=https://…/v1`.
3. Frontend `ApiClient` sends `credentials: 'include'` (Phase 3 / [ADR-0008](./adr/0008-authentication.md)).
4. Frontend deletes its mock tree.

The SPA already uses `HttpTimeTrackingRepository` for every read and write. No view or store rewrite.
