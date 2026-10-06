# Vynno API contract

**Status:** Living — canonical copy. The frontend `docs/api-contract.md` is generated from this file by `scripts/sync-contract`.  
**Snapshot date:** 2026-08-14  
**Last updated:** 2026-09-30  
**Amended:** Profile writes + public avatar GET; user-defined activity types ([ADR-0012](./adr/0012-activity-types.md)); session edit / delete / manual entry; session list cursor pagination ([ADR-0014](./adr/0014-session-list-pagination.md)); email login identifier; register confirmation + password reset ([ADR-0015](./adr/0015-outbound-email.md)); account preferences ([ADR-0017](./adr/0017-account-prefs.md)); signed-in password and email change ([ADR-0008](./adr/0008-authentication.md) amendment 2026-09-30); day totals ([ADR-0018](./adr/0018-day-totals.md))

This is the wire format the SvelteKit app already speaks. Implement these resources. Do not extend this file without a contract amendment ([working-agreement.md](./working-agreement.md) §6).

**Provenance.** First copied from the frontend repo ([`vynno`](https://github.com/EmilM32/vynno)) as of the snapshot date. This file in `vynno-api` is now the one to edit; `scripts/sync-contract` writes the frontend copy and `scripts/sync-contract --check` fails on drift. The **client executable source of truth** is the frontend’s `src/lib/api/schemas/`.

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
| Pagination | `GET /sessions` only: `limit` + opaque `cursor`; `{ items, nextCursor }` ([ADR-0014](./adr/0014-session-list-pagination.md)) |
| Auth | HttpOnly session cookie (see [Auth](#auth)) |
| Operator docs | `GET /swagger/` and `GET /openapi.json` — **not** SPA resources ([ADR-0013](./adr/0013-openapi-swagger.md)) |

Creates return **`201`**. Other successful writes return **`200`** with the updated resource. `DELETE` returns **`204`** with an empty body.

---

## Error codes

| Code | Status | When | Frontend UI string |
| --- | --- | --- | --- |
| `not_found` | 404 | Unknown project, session, or activity type id | `error_not_found` |
| `invalid_query` | 400 | Bad `status` / `limit` / `cursor`, or bad `from` / `to` / `timeZone` on `/stats/days` | fallback |
| `invalid_json` | 400 | Request body is not JSON | `error_invalid_response` |
| `invalid_body` | 400 | Write body failed the request schema / validation | fallback (`error_failed_*`) |
| `invalid_response` | 502 | Client-only: body did not match the response schema | `error_invalid_response` |
| `http_error` | 4xx/5xx | Client-only: non-OK without an envelope | fallback |
| `session_not_active` | 404 | `GET /sessions/active` when idle | `error_not_found` |
| `session_already_active` | 409 | `POST /sessions` while one is active | `error_stop_before_start` |
| `project_archived` | 409 | Start against an archived project | `error_project_archived` |
| `code_in_use` | 409 | Project `code` not unique | `error_code_in_use` |
| `name_in_use` | 409 | Activity type `name` not unique for this user | `activity_types_name_in_use` |
| `last_active_project` | 409 | Archive/delete of the last active project | `error_last_active_project` |
| `project_has_sessions` | 409 | Hard-delete of a project that has logs | `projects_cannot_delete_has_sessions` |
| `activity_type_has_sessions` | 409 | Hard-delete of an activity type that has sessions | `activity_types_cannot_delete_has_sessions` |
| `invalid_transition` | 409 | Stop (or archive/restore) in a bad state | fallback |
| `unauthorized` | 401 | Missing, unknown, or expired session on a protected route | `error_unauthorized` |
| `invalid_credentials` | 401 | Login email/password do not match, or the current password on a signed-in credential change is wrong | `error_invalid_credentials` |
| `email_in_use` | 409 | Register or email change to a taken email | `error_email_in_use` |
| `invalid_code` | 401 | Wrong, expired, or already used one-time code | `error_invalid_code` |
| `rate_limited` | 429 | Login failure caps, register/reset send cooldown, send cap, too many guesses, and too many requests from one client | `error_rate_limited` |
| `internal_error` | 500 | Unhandled faults (not `invalid_body`) | fallback |

`invalid_response` and `http_error` are **not** codes this server should emit. Always send the envelope on failure so the client does not fall back to `http_error`.

`rate_limited` (429) covers login failure caps, register/reset send cooldown, send cap, too many guesses, and too many requests from one client. `POST /auth/login` lists `rate_limited`. 429 responses include `Retry-After` (seconds). 10 failures / 15 min per email (11th is 429 even if the password is then correct, until the window passes; success resets that counter). 30 failures / 15 min per client IP. A wrong current password on `/auth/password/change` or `/auth/email/code` counts as a login failure for that account's email and the client IP. 5 `register/code`, `password/forgot`, or `email/code` sends / 10 min per client IP; the 6th is 429 and sends no mail. Client IP is taken from `X-Forwarded-For` only when the TCP peer is a trusted proxy.

`internal_error` is 500 for unhandled faults (not `invalid_body`).

Unknown route and wrong method: JSON `404` `not_found`.

Body over the BFF 2 MB limit: `413` with envelope code `invalid_body` and message `Request body is too large.` (the BFF emits this).

JSON bodies must be sent with `Content-Type: application/json` (parameters such as `charset` are allowed). Any other or missing type → `400 invalid_body` (`Content-Type must be application/json.`). The API itself caps a JSON body at 64 KiB → `400 invalid_body` (`Request body is too large.`); the avatar upload keeps its own multipart limit.

Wrong JSON type → `400 invalid_body`. Malformed JSON and trailing data → `400 invalid_json`. Unknown fields are rejected on POST and PATCH (`400 invalid_body`). Empty `status` query means no filter.

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

1. **One live session.** At most one session with status `active`. A second `POST /sessions` is `409 session_already_active`. The client requires an explicit stop — do not auto-stop.
2. **Restart is a new session.** Restart-from-recent sends `POST /sessions` with the same `projectId` / `note` / optional fields. It is not a resume of a stopped log. A break is stop, then start.
3. **Session actions are verbs.** Use `/stop` — not a generic `PATCH status`. `PATCH /sessions/:id` updates fields; it must not send `status`.
4. **Elapsed time.** Duration is `endedAt - startedAt` (or `now - startedAt` while active). Sessions are continuous intervals; there is no pause accounting.
5. **Default `GET /projects` omits archived.** Pass `includeArchived=true` for management UI. Archived projects must still resolve via `GET /projects/:id` so logs keep a label.
6. **Last active project.** Cannot archive or hard-delete the last non-archived project (`409 last_active_project`).
7. **Hard delete** only when **zero** sessions reference the project. Otherwise `409 project_has_sessions` — archive instead.
8. **Code uniqueness** is case-insensitive among all non-deleted projects, only when `code` is non-empty.
9. **Cannot start** on a missing (`404 not_found`) or archived (`409 project_archived`) project.
10. **Edit / delete.** `PATCH /sessions/:id` and `DELETE /sessions/:id` apply to any session, including the live timer. Deleting live returns idle. Stopped `endedAt` cannot be cleared. Live `endedAt` cannot be set (use `/stop`, then PATCH the stop time).
11. **Manual entry.** `POST /sessions/manual` creates a stopped log with `startedAt` and `endedAt`. Allowed while a live session exists. Archived projects are allowed. `session_already_active` and `project_archived` apply only to live `POST /sessions`.

---

## Auth

Mechanism: [ADR-0008](./adr/0008-authentication.md). Mail: [ADR-0015](./adr/0015-outbound-email.md). Register is two-step (code, then create). Password reset is two-step (forgot, then reset).

Login and register set an HttpOnly cookie `vynno_session`. The JSON body is `{ "profile": ProfileDto }` only — the session secret is not in the response.

Protected routes accept **either**:

1. Cookie `vynno_session=<token>` (what the SPA sends via `credentials: 'include'`), or
2. `Authorization: Bearer <token>` (tests, curl, non-browser clients)

Anything else on a protected route is `401 unauthorized`. A project or session id that belongs to another user is `404 not_found`.

| Method | Path | Auth | Body | Success | Typical errors |
| --- | --- | --- | --- | --- | --- |
| POST | `/auth/register/code` | no | `{ "email" }` | `204` empty | `invalid_body`, `email_in_use`, `rate_limited` |
| POST | `/auth/register` | no | `RegisterDto` | `{ profile }` `201` + `Set-Cookie` | `invalid_body`, `email_in_use`, `invalid_code`, `rate_limited` |
| POST | `/auth/login` | no | `LoginDto` | `{ profile }` `200` + `Set-Cookie` | `invalid_body`, `invalid_credentials`, `rate_limited` |
| POST | `/auth/logout` | yes | — | `204` + clear cookie | `unauthorized` |
| POST | `/auth/password/forgot` | no | `{ "email" }` | `204` empty | `invalid_body`, `rate_limited` |
| POST | `/auth/password/reset` | no | `ResetPasswordDto` | `204` empty | `invalid_body`, `invalid_code`, `rate_limited` |
| POST | `/auth/password/change` | yes | `ChangePasswordDto` | `204` empty | `unauthorized`, `invalid_body`, `invalid_credentials`, `rate_limited` |
| POST | `/auth/email/code` | yes | `{ "email", "password" }` | `204` empty | `unauthorized`, `invalid_body`, `invalid_credentials`, `email_in_use`, `rate_limited` |
| POST | `/auth/email/change` | yes | `{ "email", "code" }` | `ProfileDto` `200` | `unauthorized`, `invalid_body`, `invalid_code`, `email_in_use`, `rate_limited` |

Register is two steps. `POST /auth/register/code` emails a 6-digit code (15 minute TTL) when the address is free. Taken email is `409 email_in_use`. The account is **not** created until `POST /auth/register` accepts that code.

`RegisterDto`:

```json
{ "email": "alex@example.com", "password": "a-long-enough-secret", "code": "123456", "displayName": "Alex Dev", "rememberMe": true }
```

`code` is required: exactly six digits, matching the unused register challenge for that email. `displayName` and `rememberMe` may be omitted. Omitted `displayName`, or one that is empty after the text pipeline (see [Profile](#profile)), is stored as `""` (the SPA shows the email). Omitted `rememberMe` is `true`. Do not send `username`. Do not return `code` in any JSON body.

`LoginDto`:

```json
{ "email": "alex@example.com", "password": "a-long-enough-secret", "rememberMe": true }
```

Emails are stored NFC, lowercased, domain in IDNA punycode. NFC and NFD are one account. An IDN domain and its punycode form are one stored email. Cc/Cf anywhere → `400 invalid_body`. Local part longer than 64 octets → 400; 64 is accepted. Non-ASCII local parts are allowed. The address is still one address (`net/mail.ParseAddress` equals the whole string) whose domain contains a `.`, 3–254 characters. Unique among accounts. Password: 8–128 characters and at most 72 bytes of UTF-8 (the bcrypt input limit); longer is `400 invalid_body`. Login with a malformed email is `invalid_credentials` (same as unknown email).

`rememberMe: true` (default) sets cookie `Max-Age` to 30 days. `false` sets a session cookie (cleared when the browser quits). The server still expires the token after 30 days.

Password reset is also two steps. `POST /auth/password/forgot` always returns `204` for a well-formed email (including unknown addresses) and sends a code only when the account exists. `POST /auth/password/reset` sets a new password and revokes every session for that user. It does not set a cookie; the user logs in afterwards.

`ResetPasswordDto`:

```json
{ "email": "alex@example.com", "code": "123456", "password": "a-new-long-enough-secret" }
```

Wrong, expired, or already-used `code` is `401 invalid_code` (do not distinguish those cases). Send cooldown, send cap, or too many guesses is `429 rate_limited`. Cooldown is 60 seconds per email+purpose; 5 sends per hour; 5 guesses then the challenge is spent and a new send is required; concurrent guesses share the same 5. A spent challenge keeps its cooldown and send count. A resend replaces the previous code. Operator seed/reset accounts skip this flow.

Signed in, the password and the email can change without the reset flow. Each keeps the caller's session and deletes every **other** session token for the account.

`ChangePasswordDto`:

```json
{ "currentPassword": "a-long-enough-secret", "newPassword": "a-new-long-enough-secret" }
```

`newPassword` follows the password rule above (`invalid_body`). A wrong `currentPassword` is `401 invalid_credentials`, not `unauthorized`: the session stays valid. Success is `204`, and the account's address gets a notice mail.

Email change is two steps, like register. `POST /auth/email/code` `{ "email": "<new address>", "password": "<current password>" }` checks the password, then mails a 6-digit code to the **new** address. The email does not change yet. Same address as now → `invalid_body`. Taken → `409 email_in_use`. `POST /auth/email/change` `{ "email": "<new address>", "code": "123456" }` switches the sign-in email and returns the updated `ProfileDto`. The old address gets a notice mail. The code is bound to the account that asked for it; another account sending it gets `invalid_code`. If someone registers the address between the two steps, the change is `409 email_in_use`. Code TTL, cooldown, send cap, and guess cap are the same as register.

Cookie flags: `HttpOnly`, `SameSite=Lax`, `Path=/`, `Secure` when the process is configured for HTTPS.

CORS is locked to the SPA origin(s) and allows credentials. Any request whose `Origin` is outside the allowlist (SPA origins plus the API's public origin) is answered `403` with an empty body, public routes included. Mutating cookie-backed requests must send an `Origin` (or `Referer`) in that allowlist.

Public: `POST /auth/login`, `POST /auth/register`, `POST /auth/register/code`, `POST /auth/password/forgot`, `POST /auth/password/reset`, `GET /avatars/:id`. Every other `/v1` resource requires a session. `GET /healthz` is outside `/v1` and stays public. Operator Swagger UI (`GET /swagger/`, `GET /openapi.json`) is also outside `/v1` and public on this loopback process.

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
	"email": "alex@example.com",
	"avatarUrl": null
}
```

`displayName` may be `""` when the user did not set a name. Names: the text pipeline (reject U+FFFD, NFC, reject Cc and bidi controls, strip zero-width characters, trim), 1–80 code points for project and activity-type names. Display name is the same pipeline at 0–80 code points. `email` is the login identifier. It is not writable on `PATCH /me`; change it with `/auth/email/code` then `/auth/email/change`. `avatarUrl` is JSON `null` when absent. `avatarUrl` stays the absolute URL `{PUBLIC_API_ORIGIN}/v1/avatars/{uuid}` (internal origin on local prod). The SPA rewrites it to a same-origin path before rendering. This is intentional. The stored value is the path only; the origin is prefixed at read time.

There is no `handle`. Chrome shows `displayName` if non-empty, otherwise the raw email (no `@` prefix).

`UpdateProfileDto` — all fields optional. Same present-vs-absent rule as `UpdateProjectDto`.

```json
{ "displayName": "Alex Dev" }
```

- `displayName`: the text pipeline above, at most 80 code points. Omit = leave unchanged. Input that is empty after the pipeline clears the name so the UI falls back to email: `""`, and also whitespace-only (including NBSP) or zero-width/FEFF-only input. That is `200` with `displayName: ""`, not `400` (shared text vectors, kind `displayName`). Bidi or Cc controls → `invalid_body`. `null` → `invalid_body`.
- Do not send `email` or `avatarUrl` on this body. Email changes through `/auth/email/*`. Avatar is only `PUT` / `DELETE /me/avatar`.

`PUT /me/avatar`:

- `Content-Type: multipart/form-data`
- Field name: `file` (one part)
- Detected type (magic bytes, not the client `Content-Type` or filename): `image/jpeg`, `image/png`, `image/webp`
- Max decoded size: 1 MiB
- Replace deletes the previous row (if any), inserts a new UUID, updates `avatarUrl`, returns `ProfileDto`
- Missing part, empty file, unknown type, or oversize → `invalid_body`

`DELETE /me/avatar` when already null is still `200` with `avatarUrl: null`.

`GET /avatars/:id` is public (no cookie). Success is the raw bytes with `Content-Type` from the stored row, `Cache-Control: public, max-age=31536000, immutable`, `X-Content-Type-Options: nosniff`, and `Content-Security-Policy: default-src 'none'; sandbox`. Unknown id → `404` `{ "error": { "code": "not_found", "message": "…" } }`.

### Preferences

Account-wide settings that follow the user across devices. [ADR-0017](./adr/0017-account-prefs.md). Theme and locale stay device-local.

| Method | Path | Auth | Body | Success | Errors |
| --- | --- | --- | --- | --- | --- |
| GET | `/me/prefs` | yes | — | `PrefsDto` | `unauthorized` |
| PATCH | `/me/prefs` | yes | `UpdatePrefsDto` | `PrefsDto` `200` | `unauthorized`, `invalid_json`, `invalid_body`, `not_found` |

`PrefsDto`:

```json
{ "dailyTargetMs": 28800000, "defaultProjectId": "proj-auth" }
```

Both fields are JSON `null` when unset. A user who never saved prefs gets both as `null`; the SPA applies its defaults (8 hours, first active project).

`UpdatePrefsDto` — all fields optional. Same present-vs-absent rule as `UpdateProjectDto`: omit leaves a field unchanged, `null` clears it.

```json
{ "dailyTargetMs": 21600000, "defaultProjectId": null }
```

- `dailyTargetMs`: integer from 60000 through 86400000 (one minute to one day). Anything else → `invalid_body`.
- `defaultProjectId`: a project this user owns. Archived is allowed. Unknown, malformed, or another user's id → `404 not_found`. Hard-deleting that project clears the pref.
- Any other field → `invalid_body`.

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
{ "name": "New tool", "color": "#3b82f6", "code": "TOOL", "progressPercent": 60 }
```

`code` may be `null` or omitted. Project code: ASCII only, `^[A-Z0-9-]{1,8}$` with at least one letter or digit. `---` and `ı` are 400. `A-1` is accepted. Empty or whitespace clears the code. `color` is a `#rrggbb` hex. `progressPercent` is optional 0–100; `null` or omit leaves it unset. Names: the text pipeline above, 1–80 code points.

`UpdateProjectDto` — all fields optional; `code: null` clears the chip; `progressPercent: null` clears the dashboard bar:

```json
{ "name": "Renamed", "code": null, "progressPercent": 80 }
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

`name` is a display label (the text pipeline above, 1–80 code points, stored as typed), unique per user case-insensitively. `"D\u200bUP"` normalizes to `"DUP"`, so the existing index returns `409 name_in_use`. The SPA shows this string; chips render it uppercase.

`color` is one of: `primary`, `secondary`, `tertiary`, `error`, `on-surface-variant`, `outline`, `primary-container`, `secondary-container`. The last two are stored ids; the SPA paints them as indigo and coral activity accents, not Material container fills.

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
| GET | `/sessions?status=active,stopped&from=…&to=…&limit=n&cursor=…` | — | `{ items: SessionDto[], nextCursor: string \| null }` newest-first | `invalid_query` |
| GET | `/sessions/active` | — | `SessionDto` | `session_not_active` |
| GET | `/sessions/:id` | — | `SessionDto` | `not_found` |
| POST | `/sessions` | `StartSessionDto` | `SessionDto` `201` | `session_already_active`, `not_found`, `project_archived`, `invalid_body` |
| POST | `/sessions/manual` | `CreateManualSessionDto` | `SessionDto` `201` | `not_found`, `invalid_body` |
| PATCH | `/sessions/:id` | `UpdateSessionDto` | `SessionDto` | `not_found`, `invalid_body` |
| DELETE | `/sessions/:id` | — | `204` | `not_found` |
| POST | `/sessions/:id/stop` | — | `SessionDto` | `not_found`, `invalid_transition` |

`SessionDto`:

```json
{
	"id": "sess-today-1",
	"projectId": "proj-alpha",
	"note": "Database schema migration script",
	"ticketId": null,
	"activityTypeId": "8f3e0c1a-2b4d-4e6f-8a90-b1c2d3e4f567",
	"status": "stopped",
	"startedAt": "2026-03-11T08:00:00.000Z",
	"endedAt": "2026-03-11T10:15:00.000Z"
}
```

`StartSessionDto`:

```json
{
	"projectId": "proj-auth",
	"note": "Refactoring Auth Service",
	"ticketId": null,
	"activityTypeId": null
}
```

`activityTypeId`: UUID of an activity type this user owns, or JSON `null`. Unknown id is `404 not_found`.  
`status`: `active` \| `stopped`

`GET /sessions/active` returns the active session. Idle → `404` `{ "error": { "code": "session_not_active", "message": "…" } }`.

`status` query is a comma-separated list of those enum values. Empty `status` query means no filter. `limit` is a positive integer, default **20**, max **100**. `cursor` is an opaque string from the previous page’s `nextCursor`; omit it on the first page. Anything else is `400 invalid_query`.

`from` and `to` are optional ISO-8601 instants with an offset, as in `SessionDto` (`2026-09-01T00:00:00.000Z`). They keep the sessions that **overlap** `[from, to)`: `startedAt < to`, and `endedAt > from` or the session is live. Either may be sent alone; with both, `to` must be after `from`. Order, `limit`, and `cursor` work as without them; send the same `from` / `to` with every page. A session that runs past an edge is returned whole — the client clips it. Use them to load one period (a timeline for a past range) instead of paging back from the newest session. Unparseable values or `to ≤ from` are `400 invalid_query`. [ADR-0014](./adr/0014-session-list-pagination.md) amendment 2026-10-05.

Session instants are compared at microsecond precision. `startedAt >= 2000-01-01T00:00:00Z`. `startedAt` and `endedAt` ≤ now+5min. Duration ≤ 7 days; for a live session that is `now − startedAt`, so a live `startedAt` older than 7 days is `400 invalid_body`. Stopping a session that has run longer than 7 days stores `endedAt = startedAt + 7 days`. A patch that omits both instants does not re-check bounds.

Notes ≤ 500 code points after trim; ticketId ≤ 64; one emoji counts as 1. Tabs, LF, CR allowed in notes. Other Cc and bidi controls rejected. Existing oversized notes still load; a patch that omits `note` still succeeds.

Session list body:

```json
{
	"items": [],
	"nextCursor": null
}
```

`nextCursor` is JSON `null` when this page is the last. Follow it as `cursor` to load the next page. Do not parse the cursor. Other lists stay `{ "items": T[] }`.

`UpdateSessionDto` — all fields optional. Same present-vs-absent rule as `UpdateProjectDto`. Do not send `status` or `id` (`invalid_body`).

```json
{
	"projectId": "proj-auth",
	"note": "Renamed task",
	"ticketId": null,
	"activityTypeId": null,
	"startedAt": "2026-03-11T08:00:00.000Z",
	"endedAt": "2026-03-11T10:15:00.000Z"
}
```

- `note`: trim; empty → `"Untitled session"`.
- `projectId`: must exist for this user. Archived is allowed.
- `activityTypeId` / `ticketId`: `null` clears.
- `endedAt`: required to stay set on stopped sessions; must stay `null` on live (use `/stop`).

`CreateManualSessionDto` — always inserts `status=stopped`. Allowed while a live session exists. Archived projects are allowed.

```json
{
	"projectId": "proj-auth",
	"note": "Forgot to start the timer",
	"ticketId": null,
	"activityTypeId": null,
	"startedAt": "2026-03-11T08:00:00.000Z",
	"endedAt": "2026-03-11T10:15:00.000Z"
}
```

`projectId`, `startedAt`, and `endedAt` are required. Same note / activity rules as start. `endedAt` must be after `startedAt`.

### Stats

Tracked time summed on the server, so charts over a long or past range do not page through `GET /sessions`. [ADR-0018](./adr/0018-day-totals.md).

| Method | Path | Body | Success | Typical errors |
| --- | --- | --- | --- | --- |
| GET | `/stats/days?from=YYYY-MM-DD&to=YYYY-MM-DD&timeZone=Area/City` | — | `{ items: DayTotalDto[] }` | `invalid_query` |

`DayTotalDto`:

```json
{
	"date": "2026-09-28",
	"projectId": "proj-auth",
	"activityTypeId": null,
	"durationMs": 5400000,
	"sessionCount": 2
}
```

- One row per `date` + `projectId` + `activityTypeId` with tracked time. Days with nothing are absent. Sorted by `date`, then `projectId`, then `activityTypeId` (`null` first).
- Only **stopped** sessions. The live session is not included; the client adds its elapsed time.
- A session belongs to the local date of its `startedAt` in `timeZone`, and its whole duration counts there, even if it runs past midnight.
- `durationMs` is the sum of `endedAt − startedAt` with both instants at millisecond precision (as in `SessionDto`).
- `from` and `to` are required, inclusive, and at most 400 days apart (`to` ≥ `from`). `timeZone` is a required IANA name such as `Europe/Warsaw` or `UTC`. Anything else is `400 invalid_query`.

---

## Domain vs DTO

The SPA’s UI types (`$lib/types/domain`) use `isArchived` and omit absent optionals. DTOs use `archived` and JSON `null`. Implement the DTO column. The SPA converts in `src/lib/api/mappers/`.

---

## Out of scope

Not in this contract. Do not invent them to “complete” the API without a contract amendment.

| Area | Client today |
| --- | --- |
| Theme / locale | Device-local |
| Percentages, labels, chart series | Computed on the client from `/stats/days` rows or loaded sessions |

---

## SPA attach

The SPA (`vynno`) calls this contract with `PUBLIC_API_BASE=/v1` and `credentials: 'include'`. It does not send `Authorization`. CORS must list the SPA origin in `SPA_ORIGIN`; cookies will not be stored if CORS is `*`. Cookie flags: [ADR-0008](./adr/0008-authentication.md).

Every read and write in the SPA goes through `HttpTimeTrackingRepository`. `PUBLIC_API_BASE` is `/v1` (same-origin BFF). Schema and mapper changes absorb wire-format drift; views and the session store are not rewritten for it.

Client IP through the BFF: the SvelteKit server deletes any inbound `X-Forwarded-For` and `X-Real-IP` and writes `getClientAddress()` to `X-Forwarded-For`. The production Node process sets `ADDRESS_HEADER=X-Forwarded-For` and `XFF_DEPTH=1`, so that address is the one Caddy appended. A browser-supplied `X-Forwarded-For` is never the rate-limit bucket key.

Client-only session behavior (the API does not enforce it): stopping a session younger than 1 second deletes it with `DELETE /sessions/:id` instead of keeping a 0s row. Logs show `<1s` for older sub-second rows.

A contract change is a paired change: this file (then `scripts/sync-contract`) + frontend `src/lib/api/schemas`.
