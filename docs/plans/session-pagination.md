# Plan — Session list pagination

**Status:** Done  
**Last updated:** 2026-08-21  
**Tracking:** Backlog PAGE, roadmap Phase 5, open question #10  
**Depends on:** [ADR-0014](../adr/0014-session-list-pagination.md) Accepted

---

## Summary

Cursor-paginate `GET /v1/sessions` (newest-first keyset). The SPA loads 15 sessions on boot and fetches the next page as Logs scrolls. Projects and activity types stay unbounded lists. Insights aggregate endpoints are not part of this work.

## Why now

The full session list on boot does not scale. `limit` alone cannot ask for “the next page.” Offset pages drift when the timer starts.

## Constraints

- Amend the contract (both repos) before handlers.
- No new error codes. Bad `cursor` / `limit` / `status` are `invalid_query`.
- Do not add `nextCursor` to generic `{ items }` lists.
- `internal/domain` imports neither Gin nor the database driver.
- Opaque cursor; clients do not parse it.

## Approach

1. Accept ADR-0014; amend contract, PRD, domain, OQ #10, backlog, roadmap, handoff.
2. goose index `(user_id, started_at DESC, id DESC)`; sqlc keyset `ListSessions`; Memory double.
3. HTTP: default `limit` 20, max 100, `sessionListDTO` with `nextCursor`.
4. Frontend: schemas, first-page seed, store `loadMore` / `ensureThrough`, Logs sentinel, session-count for delete guards.

## Risks

| Risk | Failure mode | Mitigation |
| --- | --- | --- |
| Offset pages | Skip/duplicate after writes | Keyset on `(started_at, id)` |
| In-memory delete counts | UI allows delete, API 409 | `GET …/session-count` |
| Insights assume a full list | Wrong KPIs | Drain pages on that visit until period start |

## Out of scope

INS endpoints, server-side Logs search, prefs, paginating other lists, `total` on the session list.

## Exit checklist

- [x] ADR-0014 Accepted; docs updated in both repos
- [x] `GET /v1/sessions` returns `{ items, nextCursor }`; `limit` default 20, max 100
- [x] Keyset SQL + matching index; Memory store matches
- [x] OpenAPI lists `cursor` / `limit` / `nextCursor`
- [x] SPA first paint loads 15; Logs loads more on scroll; mutations reset to page 1
- [x] Dashboard/Insights drain only as far as the visible period
- [x] Project/activity-type delete uses session-count
- [x] `go test ./...` green

## Related

- [../adr/0014-session-list-pagination.md](../adr/0014-session-list-pagination.md)
- [../api-contract.md](../api-contract.md)
- [../roadmap.md](../roadmap.md)
