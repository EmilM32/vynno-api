# ADR-0018: Day totals endpoint

**Status:** Accepted  
**Date:** 2026-09-30  
**Deciders:** Project owner

## Context

Insights and the Dashboard compute totals on the client from loaded sessions ([ADR-0014](./0014-session-list-pagination.md) §6). The SPA pages `GET /sessions` newest-first until it reaches the window it needs. Looking at a month a year ago downloads every session in between. A year heatmap would download a year of sessions on every Dashboard visit.

Every chart the SPA draws from a range can be built from one grouping: tracked time per **local date**, **project**, and **activity type**. The donut sums by project, the activity bars by activity type, the breakdown table by project and activity, and a heatmap by date.

The SPA already has a rule for which day a session belongs to: the local date of `startedAt`, with the whole duration counted there, even past midnight. The server must use the same rule or the two would disagree.

## Decision

1. **`GET /v1/stats/days?from=&to=&timeZone=`** returns `{ items: DayTotalDto[] }`, one row per (date, projectId, activityTypeId) that has tracked time. Rows are sorted by date, then projectId, then activityTypeId (`null` first). Days with nothing are absent.
2. **Civil dates and an IANA zone from the client.** `from` and `to` are inclusive `YYYY-MM-DD`. `timeZone` is required: the server has no idea which zone the user lives in, and a UTC default would silently shift evening sessions to the next day. `Local` and file paths are rejected. The binary embeds the time zone database.
3. **Same day rule as the SPA.** A session belongs to the local date of `startedAt` in `timeZone`, and its whole `endedAt − startedAt` counts on that date.
4. **Milliseconds as on the wire.** Instants are cut to milliseconds before subtracting, so a total matches what the SPA gets from the same `SessionDto`s.
5. **Stopped sessions only.** The live session changes every second. The SPA already has it and adds its elapsed time locally.
6. **At most 400 days per request.** That covers a 53-week heatmap and a 366-day custom Insights range. Longer spans are several requests. Every bad query is `400 invalid_query`.
7. **The server sums; the client shapes.** No percentages, labels, colors, or chart series on the wire. The SPA still owns display, rounding, and names.
8. **Computed per request from `sessions`.** No rollup table. The `(user_id, started_at)` index already serves the window scan.

## Consequences

### Positive

- A past range or a year heatmap is one small request instead of paging through history.
- One grouping serves every chart, so the endpoint does not grow per screen.
- No new table, no cache to invalidate after edits.

### Negative / tradeoffs

- The SPA needs to refetch after a session write, or show slightly stale totals until it does.
- Sessions that run past midnight stay on their start day. That is the existing rule; splitting them would change every client total too.
- Two request shapes to keep in step: sessions for lists, day totals for charts.

## Alternatives considered

| Option | Why not |
| --- | --- |
| `from` / `to` filter on `GET /sessions` | Still ships every session in the range for a heatmap, and the SPA's history list assumes one contiguous newest-first run. Kept as a later amendment for Logs. |
| One endpoint per chart (donut, bars, heatmap) | Three shapes for one grouping; each new chart is a contract change. |
| Aggregate in Postgres with `AT TIME ZONE` | Postgres and Go ship separate zone databases. Summing in Go keeps one rule, testable with the in-memory store. |
| Rollup table updated on every write | Invalidation on edit, delete, and manual entry for a table of a few thousand rows per user. |
| Default `timeZone` to UTC | Wrong dates for anyone not in UTC, with no error. |
| Include the live session | Stale the moment it is sent. |

## Related

- [../api-contract.md](../api-contract.md) Stats
- [0014-session-list-pagination.md](./0014-session-list-pagination.md)
- [../domain-model.md](../domain-model.md) §5.6
