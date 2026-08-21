# ADR-0005: Session lifecycle

**Status:** Amended  
**Date:** 2026-08-14  
**Deciders:** Project owner  
**Inherited from:** Frontend domain model and mock repository (2026-08-12 / 2026-08-14)

## Context

The product is a single running timer. The SPA assumes at most one live session (`active` or `paused`), verb-based transitions, and pause accounting that keeps a live clock correct after refresh.

If the server auto-stops, allows two live sessions, or stores pause time differently, the Timer, Logs, and Insights will disagree with the mock the UI was built against.

## Decision

1. **At most one live session** (`active` or `paused`) per user. A second `POST /sessions` is `409 session_already_active`. **Do not auto-stop.**
2. **Idle** means no live session. `GET /sessions/active` returns that live session, or `404 session_not_active`.
3. **Start** creates a **new** row: `status=active`, `startedAt=now` (UTC ISO), `pausedMs=0`. Restart-from-recent is this, not a resume of a stopped log.
4. **Verbs only:** `POST /sessions/:id/pause|resume|stop`. No `PATCH` of `status`.
5. **Pause** only from `active` → set `pausedAt=now`. **Resume** only from `paused` → add `max(0, now - pausedAt)` to `pausedMs`, clear `pausedAt`. **Stop** from `active` or `paused`; if paused, apply resume accounting first; set `endedAt=now`.
6. Any other transition is `409 invalid_transition`. Unknown id is `404 not_found`.
7. **Empty / whitespace `note`** is stored as `"Untitled session"`.
8. **Stopped sessions are immutable** in v1. Edit, delete, and manual entry are backlog (LOG-6, LOG-7) and need a contract amendment.
9. **No Task entity.** `note`, optional `ticketId`, `activityTypeId`, and `tags` live on the session. Activity types are a user-owned dictionary ([0012-activity-types.md](./0012-activity-types.md)).

## Amendment (2026-08-21)

Clause 8 is lifted. LOG-6 and LOG-7 are in the contract.

1. **PATCH `/sessions/:id`** updates writable fields (`note`, `projectId`, `activityTypeId`, `ticketId`, `tags`, `startedAt`, `endedAt`, `pausedMs`, `targetDurationMs`). Omit = unchanged; JSON `null` clears nullable fields. **Do not accept `status` or `pausedAt`** — those stay verbs.
2. **DELETE `/sessions/:id`** hard-deletes any session, including the live one. Idle after deleting live. `204`.
3. **POST `/sessions/manual`** creates a `stopped` row with required `startedAt` and `endedAt`. Allowed while a live session exists. Archived projects are allowed. Does not auto-stop or replace the live timer.
4. **Elapsed inequalities** after any write: `endedAt > startedAt` when set; `pausedMs >= 0` and `pausedMs` does not exceed the interval; live `endedAt` stays null; stopped `endedAt` stays set; paused `pausedAt >= startedAt`.
5. Restart of a stopped log is still a new `POST /sessions`. PATCH cannot reopen a stopped row.
6. One live session, no auto-stop, empty `note` → `"Untitled session"` — unchanged.

## Consequences

### Positive

- Timer refresh can `GET /sessions/active` and reconstruct elapsed time.
- Logs and Insights stay a function of stopped sessions + timestamps.
- Matches frontend unit tests and e2e expectations.

### Negative / tradeoffs

- Users must stop before starting something else (product default; data integrity over convenience).
- Pause math is easy to get wrong around clock skew; tests must cover resume and stop-from-paused.
- Large histories are paged; see [0014-session-list-pagination.md](./0014-session-list-pagination.md).

## Alternatives considered

| Option | Why not |
| --- | --- |
| Auto-stop previous on start | Frontend explicitly rejected this for data integrity. |
| Generic `PATCH { status }` | Hides illegal transitions; contract uses verbs. |
| Store pause segments as a table from day one | Fine internally; the wire still exposes `pausedMs` + `pausedAt` only. |
| Separate Task table | Not needed for v1; recent tasks are derived from sessions. |

## Related

- [../domain-model.md](../domain-model.md)
- [../api-contract.md](../api-contract.md)
- [0003-http-json-contract.md](./0003-http-json-contract.md)
- [0004-project-lifecycle.md](./0004-project-lifecycle.md)
