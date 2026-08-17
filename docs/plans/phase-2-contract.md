# Plan — Phase 2 contract v1

**Status:** In progress  
**Last updated:** 2026-08-17  
**Tracking:** Roadmap Phase 2  
**Depends on:** Phase 1 Done, [ADR-0001](../adr/0001-backend-stack.md), [ADR-0009](../adr/0009-persistence.md)

---

## Summary

Persist a profile, projects, and sessions in PostgreSQL and implement every resource in [../api-contract.md](../api-contract.md). Domain rules live in testable Go. The SPA can set `PUBLIC_API_BASE` at this origin `/v1` and complete Timer + project CRUD.

## Why now

Phase 1 is an empty process. The mock is still the system of record. This plan is the swap surface.

## Constraints

- Only documented paths, fields, and error codes.
- `internal/domain` imports neither Gin nor the database driver.
- In-memory store is a test double only.
- No auth. Open CORS (no credentials) so a local SPA origin can call us; lock origins in Phase 4.
- Internal `user_id` on rows; not on the wire ([ADR-0006](../adr/0006-single-user-tenancy.md)).
- Stopped sessions are immutable. No LOG-6 / LOG-7.

## Approach

1. goose migration: `users`, `profiles`, `projects`, `sessions`. Unique live session per user. Unique project `code` per user when set.
2. sqlc queries; `pgx` via `database/sql`.
3. Domain: normalize name/color/code/note; pause/resume/stop; last-active and hard-delete guards.
4. Service orchestrates domain + store. Memory store for handler tests; Postgres for real process + optional integration tests.
5. Gin `/v1` handlers and the contract error envelope.
6. Boot: migrate, seed one user + profile + one active project if the database is empty.
7. Apply migrations at process start.

## Risks

| Risk | Failure mode | Mitigation |
| --- | --- | --- |
| Extra JSON fields | `invalid_response` in the SPA | Match Valibot schemas: camelCase, JSON `null`, `tags` always an array |
| Pause math | Timer wrong after refresh | Table tests for resume and stop-from-paused |
| Two live sessions | Unique index + `session_already_active` on conflict | |
| Local `go test` needs Docker | Red tests on a laptop | Domain + HTTP tests use the memory store; Postgres tests skip without `DATABASE_URL` |

## Out of scope

Auth, pagination, prefs, insights endpoints, `PATCH /me`, production CORS lock, OpenAPI.

## Exit checklist

- [x] Every [../api-contract.md](../api-contract.md) resource implemented
- [x] Domain tests for [../domain-model.md](../domain-model.md) invariants
- [x] HTTP tests cover documented error codes
- [x] Seed profile + one active project on empty DB
- [ ] Restart keeps data (needs a running Postgres; Docker was not up at implement time)
- [x] `AGENTS.md` migrate commands
- [ ] Roadmap Phase 2 boxes; this plan **Done**

## Related

- [../roadmap.md](../roadmap.md)
- [../api-contract.md](../api-contract.md)
- [../frontend-handoff.md](../frontend-handoff.md)
