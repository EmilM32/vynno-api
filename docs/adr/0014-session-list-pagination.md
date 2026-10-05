# ADR-0014: Session list pagination

**Status:** Accepted  
**Date:** 2026-08-21  
**Deciders:** Project owner

## Context

`GET /v1/sessions` returned the full newest-first list. Optional `limit` was a cap, not a page: there was no way to ask for the next slice. The SPA hydrated every row on boot, so Logs, Dashboard, and Insights all paid for the entire history.

Unbounded `limit` dumps the table. Offset pages (`?offset=&limit=`) skip or duplicate rows when a timer starts or a log is deleted, and `OFFSET n` walks n rows.

## Decision

1. **Keyset / cursor pagination on `GET /sessions` only.** Sort is `started_at DESC, id DESC`. `id` breaks ties when two sessions share `startedAt`.
2. **`cursor` is an opaque string.** Clients must not parse it. The server encodes the last row of the previous page. Malformed cursors are `400 invalid_query`.
3. **`limit` is a positive integer, default 20, max 100.** Omitting `limit` no longer dumps the table. Values outside 1–100 are `invalid_query`.
4. **Response is `{ items, nextCursor }`.** `nextCursor` is JSON `null` on the last page. Do not add `hasMore`, `total`, or `offset`. Do not put `nextCursor` on `/projects` or `/activity-types`.
5. **No new error codes.** Bad `status`, `limit`, or `cursor` stay `invalid_query`.
6. **No aggregate endpoints in this change.** Dashboard and Insights still compute on the client from loaded sessions; the SPA drains further pages only as far as the visible period.

## Consequences

### Positive

- First paint and mutation refresh load a small page.
- Inserts at the top of the list do not shift later pages.
- The existing `(user_id, started_at)` access path extends cleanly with `id`.

### Negative / tradeoffs

- Clients cannot jump to page N. Logs is sequential scroll; that is enough.
- Omitting `limit` is no longer “everything.” Curl and Swagger walk `nextCursor`.
- Logs search only matches rows already loaded. Server-side search is a later amendment.
- Insights still has no dedicated aggregate resource (backlog INS).

## Alternatives considered

| Option | Why not |
| --- | --- |
| Offset (`offset` + `limit`) | Duplicates/skips after start/delete; large offsets scan. |
| Keep unbounded omit-`limit` | Boot would still load thousands if the SPA forgets `limit`. |
| Paginate every list | Projects and activity types stay small. |
| Return `total` | Extra count query; Logs does not show it. |
| Insights endpoints now | Separate backlog item (INS); not required for scroll. |

## Amendment (2026-09-30)

Backlog INS shipped as `GET /v1/stats/days` ([0018](./0018-day-totals.md)). Charts over past or long ranges read day totals instead of paging here. §6 stands for this endpoint: `GET /sessions` still has no aggregates and no date filter.

## Amendment (2026-10-05)

Backlog LOGS-RANGE. `GET /sessions` takes optional `from` / `to` instants and keeps the sessions that **overlap** `[from, to)` (`startedAt < to`, and `endedAt > from` or live). Keyset order, `limit`, and `cursor` are unchanged.

- Overlap, not "started in range": the SPA Insights timeline draws a session that crosses the range edge on both sides of it. A client that wants "started in range" filters the page it already has.
- Instants, not dates + `timeZone`: the client already holds the range edges in its zone. `/stats/days` needs a zone because the server buckets by local date; this filter does not bucket.
- No lower bound on `started_at` from the 7-day cap: a live session can outlive it, and a single user's history is small enough for the existing `(user_id, started_at)` index.
- §6 still stands: no aggregates here. Totals stay on `/stats/days` ([0018](./0018-day-totals.md)).

## Related

- [../api-contract.md](../api-contract.md)
- [0003-http-json-contract.md](./0003-http-json-contract.md)
- [0005-session-lifecycle.md](./0005-session-lifecycle.md)
