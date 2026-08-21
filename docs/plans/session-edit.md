# Plan — Session edit, delete, and manual entry

**Status:** Done  
**Last updated:** 2026-08-21  
**Tracking:** PRD SES-10 / SES-11, backlog LOG-6 / LOG-7  
**Depends on:** Phase 4 Done, [ADR-0005](../adr/0005-session-lifecycle.md) Amended

---

## Summary

Lift stopped-session immutability. Users can PATCH every writable field on any session, DELETE any session (including the live timer), and create a stopped log with explicit start/end (`POST /sessions/manual`) without running the timer.

Status still changes only through `/pause`, `/resume`, `/stop`. One live session remains. `note` stays the single task label (no title/description split, no Task table).

## Why now

Logs and Insights already read the session list. The SPA has no edit/delete/manual UI because the contract forbade it. The store could `UpdateSession` for verbs only; it could not delete a row, and the SQL update did not write `project_id`.

## Constraints

- Amend the contract (both repos) before handlers.
- No new error codes. Time-order and “do not PATCH status” are `invalid_body`.
- `internal/domain` imports neither Gin nor the database driver.
- Do not overload `POST /sessions` (that path stays live start).
- No goose migration.

## Approach

1. Amend ADR-0005 clause 8; update contract, domain, PRD, backlog, roadmap, handoff.
2. Domain: `ApplySessionPatch`, `ManualSession`. Elapsed inequalities; live `endedAt` stays null; stopped `endedAt` stays set.
3. Store: `DeleteSession`; add `project_id` to `UpdateSession`; `sqlc generate`. Memory double.
4. Service + HTTP: `PATCH /sessions/:id`, `DELETE /sessions/:id`, `POST /sessions/manual`. Present-vs-absent JSON like projects. OpenAPI via `route()`.
5. Frontend pairing: schemas, repository, Logs edit/delete + manual form, Timer live field PATCH.

## Risks

| Risk | Failure mode | Mitigation |
| --- | --- | --- |
| PATCH `status` | Second live session | Reject `status` / `pausedAt`; unique index |
| Shrink window, keep `pausedMs` | Negative duration | `invalid_body` if `pausedMs` does not fit |
| Overload `POST /sessions` | `session_already_active` blocks logs | Dedicated `/sessions/manual` |

## Out of scope

Pagination, prefs, insights endpoints, TMR-9 UI, PATCH of `status`, reopening a stopped row, Task table, overlap detection.

## Exit checklist

- [x] ADR-0005 amended; contract, domain, PRD, backlog, roadmap, handoff updated
- [x] `PATCH /v1/sessions/:id` updates writable fields; omit vs `null` as specified
- [x] `DELETE /v1/sessions/:id` → `204` for live and stopped
- [x] `POST /v1/sessions/manual` → `201` stopped log; works while a timer is running
- [x] `POST /v1/sessions` + pause/resume/stop unchanged
- [x] No new error codes; OpenAPI/Swagger lists the three routes
- [x] Store updates `project_id`; `DeleteSession` on Postgres and Memory
- [x] Frontend schemas, repository, Logs edit/delete, manual entry
- [x] `go test ./...` green

## Related

- [../adr/0005-session-lifecycle.md](../adr/0005-session-lifecycle.md)
- [../api-contract.md](../api-contract.md)
- [../roadmap.md](../roadmap.md)
