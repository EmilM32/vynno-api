# Plan — User-defined activity types

**Status:** Done  
**Last updated:** 2026-08-20  
**Tracking:** Open question #11, contract amendment, sibling SPA pairing  
**Depends on:** Phase 4 Done, [ADR-0012](../adr/0012-activity-types.md) Accepted

---

## Summary

Replace the closed `activityType` enum with a per-user activity type dictionary: display `name`, token `color`, UUID FK on sessions, full CRUD. Register starts with an empty list.

## Why now

The column was free `TEXT` and the domain was a closed slug list. Neither matches a user-managed dictionary with theme-token colors and locale-switched labels.

## Constraints

- Docs and SPA schemas ship with the API (breaking: `activityType` → `activityTypeId`).
- `internal/domain` imports neither Gin nor the database driver.
- Color tokens only; project hex is a different field.
- No generic dictionaries table. No seed defaults on register.

## Approach

1. Accept ADR-0012; amend contract, domain, PRD, OQ #11, roadmap, handoff.
2. goose `00004_activity_types.sql`: table + backfill existing session slugs + drop `activity_type` TEXT.
3. Domain normalize name (slug) and color (token). Store + service CRUD. Start session looks up the type.
4. HTTP `/v1/activity-types`. Register does not insert rows.
5. Playground `scripts/seed` still creates types so demo sessions have FKs. `scripts/reset` stays empty.
6. Frontend: schemas, repository, Timer picker, Settings CRUD, chip/chart token map, Insights join by id.

## Exit checklist

- [x] ADR-0012 Accepted; docs amended in both repos
- [x] `activity_types` table; `sessions.activity_type_id`; TEXT column gone
- [x] Existing slugs migrated; new users have `{ items: [] }`
- [x] CRUD + session-count; isolation; delete-in-use; duplicate name
- [x] `POST /sessions` accepts `activityTypeId` null or an owned UUID
- [x] Timer / Settings / chips / Insights use the dictionary
- [x] `go test ./...` green

## Related

- [../adr/0012-activity-types.md](../adr/0012-activity-types.md)
- [../api-contract.md](../api-contract.md)
- [../roadmap.md](../roadmap.md)
